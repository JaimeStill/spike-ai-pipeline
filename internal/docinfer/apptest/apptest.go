package apptest

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/standards-lab/go-core/graph"
	"github.com/standards-lab/go-core/lifecycle"

	"github.com/JaimeStill/spike-ai-pipeline/internal/apptest"
	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/app"
)

// Builds returns a Recorder of the name of each node a's runs build, in the order each Build
// begins them. A run that builds nothing records nothing. Call it before a runs.
func Builds(a *app.App) *apptest.Recorder {
	r := &apptest.Recorder{}
	a.Graph().Observe(r.Record)
	return r
}

// ErrHalted is what a run of an App given to [Halt] fails with, labelled "lifecycle config".
var ErrHalted = errors.New("halted")

// Halt makes a's lifecycle configuration node fail with ErrHalted, so a run of a command that
// declares nodes fails its Build before anything starts. Halt is a Replace, so it panics once a
// has run a Build.
func Halt(a *app.App) {
	a.Graph().Replace(a.Nodes().LifecycleConfig, func(*graph.Scope) (lifecycle.Config, error) {
		return lifecycle.Config{}, ErrHalted
	})
}

// Model is one model a [Fake] lists, by its id, with the status llama.cpp's router reports for
// it, such as "loaded" or "unloaded"; an empty Status lists the model with no status, as an
// endpoint that isn't the router does.
type Model struct {
	ID     string
	Status string
}

// Fake is what a fake endpoint, [Serve]'s, answers: its /models listing, or, with NoListing, a
// 404 there, and its embeddings, each input's [Vector] of Dims dimensions.
type Fake struct {
	Models    []Model
	NoListing bool
	// Dims is the dimensions of each embedding; zero is [DefaultDims].
	Dims int
	// Reject, when set, rejects an embeddings request any of whose inputs it reports, with a
	// 400 and an OpenAI-style error body.
	Reject func(input string) bool
	// OnEmbed, when set, is called with each embeddings request's context and its number,
	// counting from 1, before the request is answered, so a test can cancel its run there, or
	// hold the request in flight.
	OnEmbed func(ctx context.Context, n int)
	// Vision, when set, answers each vision request; nil answers each with [VisionText] of its
	// image, complete.
	Vision func(VisionRequest) VisionReply
	// OnVision, when set, is called with each vision request's context and its number, counting
	// from 1, before the request is answered, as OnEmbed is.
	OnVision func(ctx context.Context, n int)
	// Transcribe, when set, answers each transcription request; nil answers each with
	// [TranscriptText] of its file.
	Transcribe func(TranscriptionRequest) TranscriptionReply
	// OnTranscribe, when set, is called with each transcription request's context and its
	// number, counting from 1, before the request is answered, as OnEmbed is.
	OnTranscribe func(ctx context.Context, n int)
}

// TranscriptionRequest is one transcription request an [Endpoint] received, a multipart form:
// its model, prompt, language, and response format fields, each "" when absent, and its file
// part's name and bytes.
type TranscriptionRequest struct {
	Model          string
	Prompt         string
	Language       string
	ResponseFormat string
	FileName       string
	File           []byte
}

// TranscriptionReply is how a [Fake] answers a transcription request: with Text, or, when Status
// is set, with that status and an OpenAI-style error body.
type TranscriptionReply struct {
	Text   string
	Status int
}

// TranscriptText is the transcript a [Fake] gives a file by default: the start of its SHA-256
// after one word, so a test can tell one window's transcript from another's, and no two
// windows' transcripts align at a seam.
func TranscriptText(file []byte) string {
	sum := sha256.Sum256(file)
	return fmt.Sprintf("window %x", sum[:6])
}

// VisionRequest is one vision request an [Endpoint] received, in any of the three formats: the
// format, by its route's name in docinfer's configuration, "completions", "messages", or
// "responses"; the model; the text part's prompt; the image's media type and bytes, decoded from
// its data URL or base64 source; the cap on the reply's output tokens; and the reasoning effort,
// as the format carries it: Chat Completions' reasoning_effort, Open Responses' reasoning.effort,
// or Messages' thinking, "none" for thinking disabled and output_config.effort for adaptive,
// empty when the request sends none.
type VisionRequest struct {
	Format    string
	Model     string
	Prompt    string
	MediaType string
	Image     []byte
	MaxTokens int
	Reasoning string
}

// VisionReply is how a [Fake] answers a vision request: with Text, cut at the output cap when
// Incomplete, as the format reports a reply stopped at its length; or, when Status is set, with
// that status and an OpenAI-style error body.
type VisionReply struct {
	Text       string
	Incomplete bool
	Status     int
}

// VisionText is the text a [Fake] gives an image by default: its size and the start of its
// SHA-256, so a test can tell one image's text from another's.
func VisionText(image []byte) string {
	sum := sha256.Sum256(image)
	return fmt.Sprintf("An image of %d bytes, SHA-256 %x.", len(image), sum[:6])
}

// The usage a [Fake] reports for each vision reply it answers and each transcription reply, in
// tokens, each count distinct so a test can tell which one it summed: a vision reply's input and
// output, in every format, its total their sum, given in the formats that give one; and a
// transcription reply's. An embeddings reply reports [EmbeddingTokens] of its inputs.
const (
	VisionInputTokens         = 11
	VisionOutputTokens        = 7
	TranscriptionInputTokens  = 5
	TranscriptionOutputTokens = 3
)

// EmbeddingTokens is the prompt tokens, and the total, a [Fake] reports for an embeddings
// request of inputs: their bytes, together.
func EmbeddingTokens(inputs []string) int {
	n := 0
	for _, in := range inputs {
		n += len(in)
	}
	return n
}

// DefaultDims is the dimensions of a [Fake]'s embeddings when its Dims is zero.
const DefaultDims = 4

// Vector is the embedding a [Fake] gives input, of dims dimensions: each the next byte of the
// SHA-256 chain of input, scaled into [0, 1). It is not unit length, so a test can tell a vector
// stored as returned from one normalized.
func Vector(input string, dims int) []float32 {
	v := make([]float32, dims)
	sum := sha256.Sum256([]byte(input))
	for i := range v {
		if i > 0 && i%len(sum) == 0 {
			sum = sha256.Sum256(sum[:])
		}
		v[i] = float32(sum[i%len(sum)]) / 256
	}
	return v
}

// Embedding is one embeddings request an [Endpoint] received: its model and inputs.
type Embedding struct {
	Model string
	Input []string
}

// Request is one request an [Endpoint] received: its method, path, raw query, and Authorization
// header.
type Request struct {
	Method        string
	Path          string
	Query         string
	Authorization string
}

// Endpoint is a fake model server, an httptest.Server under the API root /v1, answering as its
// Fake says and recording every request. It is safe for concurrent use.
type Endpoint struct {
	server     *httptest.Server
	fake       Fake
	mu         sync.Mutex
	requests   []Request
	embeddings []Embedding
	visions    []VisionRequest
	transcribe []TranscriptionRequest
}

// Serve starts an Endpoint answering as fake says, closed when t ends.
func Serve(t testing.TB, fake Fake) *Endpoint {
	t.Helper()
	e := &Endpoint{fake: fake}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/models", e.models)
	mux.HandleFunc("POST /v1/embeddings", e.embed)
	mux.HandleFunc("POST /v1/chat/completions", e.vision("completions"))
	mux.HandleFunc("POST /v1/messages", e.vision("messages"))
	mux.HandleFunc("POST /v1/responses", e.vision("responses"))
	mux.HandleFunc("POST /v1/audio/transcriptions", e.transcription)
	e.server = httptest.NewServer(e.record(mux))
	t.Cleanup(e.server.Close)
	return e
}

// URL returns e's API root, its base URL: the server's URL and /v1.
func (e *Endpoint) URL() string { return e.server.URL + "/v1" }

// Requests returns a copy of the requests e has received, in order.
func (e *Endpoint) Requests() []Request {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.requests)
}

// Embeddings returns a copy of the embeddings requests e has received, in the order each
// arrived.
func (e *Endpoint) Embeddings() []Embedding {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.embeddings)
}

// Visions returns a copy of the vision requests e has received, in the order each arrived.
func (e *Endpoint) Visions() []VisionRequest {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.visions)
}

// Transcriptions returns a copy of the transcription requests e has received, in the order each
// arrived.
func (e *Endpoint) Transcriptions() []TranscriptionRequest {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.transcribe)
}

// record records each request before next answers it.
func (e *Endpoint) record(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e.mu.Lock()
		e.requests = append(e.requests, Request{r.Method, r.URL.Path, r.URL.RawQuery, r.Header.Get("Authorization")})
		e.mu.Unlock()
		next.ServeHTTP(w, r)
	})
}

// models answers the listing, as llama.cpp's router does: {"data": [{"id", "status"}]}.
func (e *Endpoint) models(w http.ResponseWriter, _ *http.Request) {
	if e.fake.NoListing {
		http.NotFound(w, nil)
		return
	}
	type status struct {
		Value string `json:"value"`
	}
	type model struct {
		ID     string  `json:"id"`
		Object string  `json:"object"`
		Status *status `json:"status,omitempty"`
	}
	data := []model{}
	for _, m := range e.fake.Models {
		x := model{ID: m.ID, Object: "model"}
		if m.Status != "" {
			x.Status = &status{m.Status}
		}
		data = append(data, x)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"object": "list", "data": data}) // a test's client gone loses nothing
}

// embed answers an embeddings request as the OpenAI-compatible endpoint does, with
// encoding_format "base64", the only one the client asks for: {"data": [{"index", "embedding"}],
// "usage"}, each embedding base64 of little-endian float32s. A body it can't decode is a 400; so
// is one with an input Reject reports.
func (e *Endpoint) embed(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Model          string   `json:"model"`
		Input          []string `json:"input"`
		EncodingFormat string   `json:"encoding_format"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.EncodingFormat != "base64" {
		apiError(w, http.StatusBadRequest, "bad embeddings request")
		return
	}
	e.mu.Lock()
	e.embeddings = append(e.embeddings, Embedding{Model: req.Model, Input: req.Input})
	n := len(e.embeddings)
	e.mu.Unlock()
	if e.fake.OnEmbed != nil {
		e.fake.OnEmbed(r.Context(), n)
	}
	if e.fake.Reject != nil && slices.ContainsFunc(req.Input, e.fake.Reject) {
		apiError(w, http.StatusBadRequest, "input rejected")
		return
	}
	dims := e.fake.Dims
	if dims == 0 {
		dims = DefaultDims
	}
	type item struct {
		Object    string `json:"object"`
		Index     int    `json:"index"`
		Embedding string `json:"embedding"`
	}
	data := []item{}
	for i, in := range req.Input {
		v := Vector(in, dims)
		b := make([]byte, 4*len(v))
		for j, f := range v {
			binary.LittleEndian.PutUint32(b[4*j:], math.Float32bits(f))
		}
		data = append(data, item{Object: "embedding", Index: i, Embedding: base64.StdEncoding.EncodeToString(b)})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{ // a test's client gone loses nothing
		"object": "list", "model": req.Model, "data": data,
		"usage": map[string]int{"prompt_tokens": EmbeddingTokens(req.Input), "total_tokens": EmbeddingTokens(req.Input)},
	})
}

// apiError answers with status and an OpenAI-style error body carrying message.
func apiError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{ // a test's client gone loses nothing
		"error": map[string]any{"message": message, "type": "invalid_request_error", "code": status},
	})
}

// vision returns the handler of a generation request in format, the image and prompt of one
// user message: it decodes the request as the client's format sends it, records it, and answers
// as the Fake's Vision says, in the format's reply. A body it can't decode, or one with no image
// or prompt, is a 400.
func (e *Endpoint) vision(format string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := decodeVision(format, r)
		if err != nil {
			apiError(w, http.StatusBadRequest, "bad vision request: "+err.Error())
			return
		}
		e.mu.Lock()
		e.visions = append(e.visions, req)
		n := len(e.visions)
		e.mu.Unlock()
		if e.fake.OnVision != nil {
			e.fake.OnVision(r.Context(), n)
		}
		reply := VisionReply{Text: VisionText(req.Image)}
		if e.fake.Vision != nil {
			reply = e.fake.Vision(req)
		}
		if reply.Status != 0 {
			apiError(w, reply.Status, "image refused")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(visionBody(format, req.Model, reply)) // a test's client gone loses nothing
	}
}

// part is one content part of a vision request, in any of the three formats' names: a text
// part's text; a Chat Completions image_url's URL; an Open Responses input_image's image_url; and
// a Messages image's base64 source.
type part struct {
	Type     string          `json:"type"`
	Text     string          `json:"text"`
	ImageURL json.RawMessage `json:"image_url"`
	Source   struct {
		Type      string `json:"type"`
		MediaType string `json:"media_type"`
		Data      string `json:"data"`
	} `json:"source"`
}

// decodeVision decodes r's body as format's request of one user message of an image and a
// prompt.
func decodeVision(format string, r *http.Request) (VisionRequest, error) {
	var body struct {
		Model               string `json:"model"`
		MaxCompletionTokens int    `json:"max_completion_tokens"`
		MaxTokens           int    `json:"max_tokens"`
		MaxOutputTokens     int    `json:"max_output_tokens"`
		ReasoningEffort     string `json:"reasoning_effort"`
		Reasoning           struct {
			Effort string `json:"effort"`
		} `json:"reasoning"`
		Thinking struct {
			Type string `json:"type"`
		} `json:"thinking"`
		OutputConfig struct {
			Effort string `json:"effort"`
		} `json:"output_config"`
		Messages []struct {
			Role    string `json:"role"`
			Content []part `json:"content"`
		} `json:"messages"`
		Input []struct {
			Type    string `json:"type"`
			Role    string `json:"role"`
			Content []part `json:"content"`
		} `json:"input"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return VisionRequest{}, err
	}
	req := VisionRequest{Format: format, Model: body.Model}
	var parts []part
	switch format {
	case "completions":
		req.MaxTokens = body.MaxCompletionTokens
		req.Reasoning = body.ReasoningEffort
		if len(body.Messages) == 1 && body.Messages[0].Role == "user" {
			parts = body.Messages[0].Content
		}
	case "messages":
		req.MaxTokens = body.MaxTokens
		switch body.Thinking.Type {
		case "disabled":
			req.Reasoning = "none"
		case "adaptive":
			req.Reasoning = body.OutputConfig.Effort
		}
		if len(body.Messages) == 1 && body.Messages[0].Role == "user" {
			parts = body.Messages[0].Content
		}
	case "responses":
		req.MaxTokens = body.MaxOutputTokens
		req.Reasoning = body.Reasoning.Effort
		if len(body.Input) == 1 && body.Input[0].Type == "message" && body.Input[0].Role == "user" {
			parts = body.Input[0].Content
		}
	}
	for _, p := range parts {
		switch {
		case p.Type == "text" || p.Type == "input_text":
			req.Prompt = p.Text
		case p.Type == "image_url" || p.Type == "input_image":
			var url string
			if p.Type == "image_url" {
				var u struct {
					URL string `json:"url"`
				}
				if err := json.Unmarshal(p.ImageURL, &u); err != nil {
					return req, err
				}
				url = u.URL
			} else if err := json.Unmarshal(p.ImageURL, &url); err != nil {
				return req, err
			}
			mediaType, data, ok := strings.Cut(strings.TrimPrefix(url, "data:"), ";base64,")
			if !ok || !strings.HasPrefix(url, "data:") {
				return req, fmt.Errorf("image URL %.40q isn't a base64 data URL", url)
			}
			b, err := base64.StdEncoding.DecodeString(data)
			if err != nil {
				return req, err
			}
			req.MediaType, req.Image = mediaType, b
		case p.Type == "image" && p.Source.Type == "base64":
			b, err := base64.StdEncoding.DecodeString(p.Source.Data)
			if err != nil {
				return req, err
			}
			req.MediaType, req.Image = p.Source.MediaType, b
		}
	}
	if req.Image == nil || req.Prompt == "" {
		return req, errors.New("want one user message of an image and a prompt")
	}
	return req, nil
}

// visionBody returns reply as format's reply body, as llama.cpp's router gives it: a finished
// reply's stop reason stop, end_turn, or status completed; an incomplete one's length,
// max_tokens, or status incomplete with its reason.
func visionBody(format, model string, reply VisionReply) any {
	switch format {
	case "messages":
		stop := "end_turn"
		if reply.Incomplete {
			stop = "max_tokens"
		}
		return map[string]any{
			"id": "msg_fake", "type": "message", "role": "assistant", "model": model,
			"content":     []map[string]any{{"type": "text", "text": reply.Text}},
			"stop_reason": stop, "usage": map[string]int{"input_tokens": VisionInputTokens, "output_tokens": VisionOutputTokens},
		}
	case "responses":
		body := map[string]any{
			"id": "resp_fake", "object": "response", "model": model, "status": "completed",
			"output": []map[string]any{{
				"type": "message", "id": "msg_fake", "status": "completed", "role": "assistant",
				"content": []map[string]any{{"type": "output_text", "text": reply.Text, "annotations": []any{}}},
			}},
			"usage": map[string]int{
				"input_tokens": VisionInputTokens, "output_tokens": VisionOutputTokens,
				"total_tokens": VisionInputTokens + VisionOutputTokens,
			},
		}
		if reply.Incomplete {
			body["status"] = "incomplete"
			body["incomplete_details"] = map[string]string{"reason": "max_output_tokens"}
		}
		return body
	}
	finish := "stop"
	if reply.Incomplete {
		finish = "length"
	}
	return map[string]any{
		"id": "chatcmpl-fake", "object": "chat.completion", "model": model,
		"choices": []map[string]any{{
			"index": 0, "finish_reason": finish,
			"message": map[string]any{"role": "assistant", "content": reply.Text},
		}},
		"usage": map[string]int{
			"prompt_tokens": VisionInputTokens, "completion_tokens": VisionOutputTokens,
			"total_tokens": VisionInputTokens + VisionOutputTokens,
		},
	}
}

// transcription answers a transcription request as the OpenAI-compatible endpoint does under
// response_format json, the only one the client asks for: {"text", "usage"}. It decodes the
// multipart form, records it, and answers as the Fake's Transcribe says. A form it can't decode,
// or one with no model or no file, is a 400.
func (e *Endpoint) transcription(w http.ResponseWriter, r *http.Request) {
	req, err := decodeTranscription(r)
	if err != nil {
		apiError(w, http.StatusBadRequest, "bad transcription request: "+err.Error())
		return
	}
	e.mu.Lock()
	e.transcribe = append(e.transcribe, req)
	n := len(e.transcribe)
	e.mu.Unlock()
	if e.fake.OnTranscribe != nil {
		e.fake.OnTranscribe(r.Context(), n)
	}
	reply := TranscriptionReply{Text: TranscriptText(req.File)}
	if e.fake.Transcribe != nil {
		reply = e.fake.Transcribe(req)
	}
	if reply.Status != 0 {
		apiError(w, reply.Status, "window refused")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{ // a test's client gone loses nothing
		"text": reply.Text,
		"usage": map[string]any{
			"type": "tokens", "input_tokens": TranscriptionInputTokens, "output_tokens": TranscriptionOutputTokens,
			"total_tokens": TranscriptionInputTokens + TranscriptionOutputTokens,
		},
	})
}

// decodeTranscription decodes r's multipart form as the client sends it.
func decodeTranscription(r *http.Request) (TranscriptionRequest, error) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return TranscriptionRequest{}, err
	}
	req := TranscriptionRequest{
		Model: r.FormValue("model"), Prompt: r.FormValue("prompt"), Language: r.FormValue("language"),
		ResponseFormat: r.FormValue("response_format"),
	}
	f, h, err := r.FormFile("file")
	if err != nil {
		return req, err
	}
	defer func() { _ = f.Close() }() // read only: a close error loses nothing
	if req.File, err = io.ReadAll(f); err != nil {
		return req, err
	}
	req.FileName = h.Filename
	if req.Model == "" {
		return req, errors.New("want a model")
	}
	return req, nil
}
