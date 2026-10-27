//go:build integration

package probe_test

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode"
)

// The files that configure the router, read for every expectation below.
const (
	presetPath = "../host/llama-preset/models.ini"
	pinPath    = "../host/llama.env"
)

// Set A's roles: the model each capability probe sends to. The preset names the set and its
// values; it doesn't assign roles, so the probe does, and checks that each is a preset section.
const (
	chatModel      = "ggml-org/gpt-oss-120b-GGUF:MXFP4"
	visionModel    = "ggml-org/gemma-4-26B-A4B-it-GGUF:Q4_0"
	audioModel     = "ggml-org/gemma-4-E4B-it-GGUF:Q8_0"
	embeddingModel = "mrutkows/granite-embedding-english-r2-GGUF:BF16"
)

// embeddingDims is granite-embedding-english-r2's width, from its card
// (https://huggingface.co/ibm-granite/granite-embedding-english-r2); the preset records it only in
// a comment.
const embeddingDims = 768

// maxClip is the longest clip the probe sends Gemma 4 E4B, which loops on clips longer than
// about 30 seconds.
const maxClip = 30 * time.Second

// samplingTolerance absorbs the float32 the server holds sampling in: top_p 0.95 reads back as
// 0.949999988079071.
const samplingTolerance = 1e-6

// requestTimeout bounds one request. The models are already loaded, so this covers generation
// only, which on a reasoning model such as gpt-oss-120b takes tens of seconds.
const requestTimeout = 3 * time.Minute

// TestRouter probes the router SPIKE_ROUTER_URL names, serving set A from the preset, with set
// A loaded by `mise run host-update`. Each check is a subtest, so a failure names the check.
// Every model-routed request carries autoload=false: the probe checks that set A's models are
// loaded, and never loads one itself. Two subtests record rather than assert, /v1/responses and
// transcription timestamps: each logs what the router answered, prefixed "recorded:", and
// passes either way.
func TestRouter(t *testing.T) {
	base := os.Getenv("SPIKE_ROUTER_URL")
	if base == "" {
		t.Skip("SPIKE_ROUTER_URL is unset: set it to the router to probe, such as http://gb10-2:8080")
	}
	p, err := loadPreset(presetPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{chatModel, visionModel, audioModel, embeddingModel} {
		if !slices.Contains(p.models, id) {
			t.Fatalf("%s has no section [%s]", presetPath, id)
		}
	}
	pin, err := loadPin(pinPath)
	if err != nil {
		t.Fatal(err)
	}
	r := &router{base: strings.TrimSuffix(base, "/"), client: &http.Client{Timeout: requestTimeout}}

	t.Run("health", func(t *testing.T) {
		var got struct {
			Status string `json:"status"`
		}
		r.get(t, "/health", &got)
		if got.Status != "ok" {
			t.Errorf("/health status = %q, want ok", got.Status)
		}
	})

	t.Run("models loaded", func(t *testing.T) {
		var got struct {
			Data []struct {
				ID     string `json:"id"`
				Status struct {
					Value string `json:"value"`
				} `json:"status"`
			} `json:"data"`
		}
		r.get(t, "/models", &got)
		status := make(map[string]string, len(got.Data))
		for _, m := range got.Data {
			status[m.ID] = m.Status.Value
		}
		for _, id := range p.models {
			if s, ok := status[id]; !ok {
				t.Errorf("/models doesn't list %s", id)
			} else if s != "loaded" {
				t.Errorf("/models lists %s as %q, want loaded", id, s)
			}
		}
	})

	t.Run("build", func(t *testing.T) {
		var got struct {
			BuildInfo string `json:"build_info"`
		}
		r.get(t, "/props", &got)
		t.Logf("router build_info: %s, pin %s", got.BuildInfo, pin)
		have, err := buildNumber(got.BuildInfo)
		if err != nil {
			t.Fatalf("/props build_info: %v", err)
		}
		floor, err := buildNumber(pin)
		if err != nil {
			t.Fatalf("%s LLAMA_BUILD: %v", pinPath, err)
		}
		if have < floor {
			t.Errorf("router build b%d is below the pin %s", have, pin)
		}
	})

	// Each model's defaults are its section's sampling keys, as /props reports them. A model
	// whose section sets none, such as the embedding model, is skipped: it doesn't sample.
	t.Run("sampling", func(t *testing.T) {
		for _, id := range p.models {
			t.Run(id, func(t *testing.T) {
				var got struct {
					Settings struct {
						Params map[string]json.RawMessage `json:"params"`
					} `json:"default_generation_settings"`
				}
				compared := 0
				for _, k := range samplingKeys {
					want, ok, err := p.float(id, k.preset)
					if err != nil {
						t.Fatal(err)
					}
					if !ok {
						continue
					}
					if compared == 0 {
						r.get(t, "/props?model="+url.QueryEscape(id), &got)
					}
					compared++
					var have float64
					raw, ok := got.Settings.Params[k.props]
					if !ok || json.Unmarshal(raw, &have) != nil {
						t.Errorf("%s /props has no numeric default %s: %s", id, k.props, raw)
					} else if math.Abs(have-want) > samplingTolerance {
						t.Errorf("%s default %s = %v, want the preset's %s = %v", id, k.props, have, k.preset, want)
					}
				}
				if compared == 0 {
					t.Skipf("the preset sets no sampling for %s", id)
				}
			})
		}
	})

	// Each model serves its own slots, its section's parallel: /props reports them as
	// total_slots.
	t.Run("slots", func(t *testing.T) {
		for _, id := range p.models {
			t.Run(id, func(t *testing.T) {
				want, ok, err := p.float(id, "parallel")
				if err != nil {
					t.Fatal(err)
				}
				if !ok {
					t.Fatalf("the preset sets no parallel for %s", id)
				}
				var got struct {
					TotalSlots int `json:"total_slots"`
				}
				r.get(t, "/props?model="+url.QueryEscape(id), &got)
				if float64(got.TotalSlots) != want {
					t.Errorf("%s total_slots = %d, want the preset's parallel = %v", id, got.TotalSlots, want)
				}
			})
		}
	})

	t.Run("chat", func(t *testing.T) {
		text := r.chat(t, chatModel, "Reply with one word: ready.", 2048)
		t.Logf("%s: %q", chatModel, text)
		// The prompt asks for one word; casing and punctuation are the model's to choose.
		if !strings.Contains(strings.ToLower(text), "ready") {
			t.Errorf("%s: the reply %q doesn't say ready", chatModel, text)
		}
	})

	t.Run("tool call", func(t *testing.T) {
		var got struct {
			Choices []struct {
				Message struct {
					ToolCalls []struct {
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
		}
		r.post(t, "/v1/chat/completions", map[string]any{
			"model":      chatModel,
			"max_tokens": 2048,
			"messages": []map[string]any{
				{"role": "user", "content": "What is the weather in Paris right now?"},
			},
			"tools": []map[string]any{{
				"type": "function",
				"function": map[string]any{
					"name":        "get_weather",
					"description": "Get the current weather for a city.",
					"parameters": map[string]any{
						"type":       "object",
						"properties": map[string]any{"city": map[string]any{"type": "string"}},
						"required":   []string{"city"},
					},
				},
			}},
			// The client constrains the shape it needs; the preset sets no tool policy.
			"tool_choice": "required",
		}, &got)
		if len(got.Choices) == 0 {
			t.Fatalf("%s: the reply has no choices", chatModel)
		}
		c := got.Choices[0]
		if len(c.Message.ToolCalls) == 0 {
			t.Fatalf("%s: the reply has no tool call (finish_reason %q)", chatModel, c.FinishReason)
		}
		call := c.Message.ToolCalls[0].Function
		t.Logf("%s: tool call %s %s", chatModel, call.Name, call.Arguments)
		if call.Name != "get_weather" {
			t.Errorf("the tool call names %q, want get_weather", call.Name)
		}
		var args struct {
			City string `json:"city"`
		}
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			t.Errorf("the tool call's arguments %q aren't JSON: %v", call.Arguments, err)
		} else if !strings.Contains(strings.ToLower(args.City), "paris") {
			t.Errorf("the tool call's city is %q, want Paris", args.City)
		}
	})

	t.Run("image", func(t *testing.T) {
		png := readFile(t, "testdata/shapes.png")
		text := r.chat(t, visionModel, []map[string]any{
			{"type": "text", "text": "Describe this image in one sentence."},
			{"type": "image_url", "image_url": map[string]any{
				"url": "data:image/png;base64," + base64.StdEncoding.EncodeToString(png),
			}},
		}, 512)
		t.Logf("%s: %q", visionModel, text)
		// shapes.png is a red circle, a blue square, and a green triangle. Each shape and each
		// colour must be named as a word of its own, singular or plural, in any phrasing, so
		// "centered" doesn't pass for red; which colour goes with which shape isn't checked.
		words := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) })
		for _, want := range []string{"circle", "square", "triangle", "red", "blue", "green"} {
			if !slices.Contains(words, want) && !slices.Contains(words, want+"s") {
				t.Errorf("%s: the description %q doesn't name %s", visionModel, text, want)
			}
		}
	})

	t.Run("audio", func(t *testing.T) {
		wav := readClip(t)
		text := r.chat(t, audioModel, []map[string]any{
			{"type": "text", "text": "Transcribe this audio clip."},
			{"type": "input_audio", "input_audio": map[string]any{
				"data":   base64.StdEncoding.EncodeToString(wav),
				"format": "wav",
			}},
		}, 512)
		t.Logf("%s: %q", audioModel, text)
		checkPhrase(t, audioModel, text)
	})

	t.Run("embeddings", func(t *testing.T) {
		var got struct {
			Data []struct {
				Embedding []float64 `json:"embedding"`
			} `json:"data"`
		}
		r.post(t, "/v1/embeddings", map[string]any{
			"model":           embeddingModel,
			"input":           "The access code is seven four two nine.",
			"encoding_format": "float",
		}, &got)
		if len(got.Data) != 1 {
			t.Fatalf("/v1/embeddings returned %d embeddings, want 1", len(got.Data))
		}
		if n := len(got.Data[0].Embedding); n != embeddingDims {
			t.Errorf("%s embedding has %d dimensions, want %d", embeddingModel, n, embeddingDims)
		}
	})

	t.Run("transcription", func(t *testing.T) {
		status, reply := r.transcribe(t, readClip(t), [][2]string{{"response_format", "json"}})
		if status != http.StatusOK {
			t.Fatalf("POST /v1/audio/transcriptions: %d: %s", status, reply)
		}
		var got struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(reply, &got); err != nil {
			t.Fatalf("POST /v1/audio/transcriptions: decoding the reply: %v: %s", err, reply)
		}
		t.Logf("%s: %q", audioModel, got.Text)
		checkPhrase(t, audioModel, got.Text)
	})

	// Recorded, never asserted: whether the router answers OpenAI's Responses API on the chat
	// model. The client's plan reads the log line.
	t.Run("responses recorded", func(t *testing.T) {
		b, err := json.Marshal(map[string]any{
			"model":             chatModel,
			"input":             "Say hello in one sentence.",
			"max_output_tokens": 2048,
		})
		if err != nil {
			t.Fatal(err)
		}
		status, reply := r.send(t, http.MethodPost, withNoAutoload("/v1/responses"), "application/json", b)
		var got struct {
			Status string `json:"status"`
			Output []struct {
				Type    string `json:"type"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"output"`
		}
		if status != http.StatusOK || json.Unmarshal(reply, &got) != nil || got.Status == "" {
			t.Logf("recorded: /v1/responses on %s doesn't answer: HTTP %d: %s", chatModel, status, reply)
			return
		}
		var types []string
		var text string
		for _, o := range got.Output {
			types = append(types, o.Type)
			for _, c := range o.Content {
				if c.Type == "output_text" {
					text += c.Text
				}
			}
		}
		t.Logf("recorded: /v1/responses on %s answers: HTTP %d, status %s, output items [%s], text %q",
			chatModel, status, got.Status, strings.Join(types, " "), text)
	})

	// Recorded, never asserted: whether transcription returns segment or word timestamps, in
	// each OpenAI-compatible request form that asks for them. The client's plan reads the log
	// lines.
	t.Run("timestamps recorded", func(t *testing.T) {
		wav := readClip(t)
		granularities := [][2]string{
			{"timestamp_granularities[]", "segment"},
			{"timestamp_granularities[]", "word"},
		}
		forms := []struct {
			name   string
			fields [][2]string
		}{
			{"verbose_json", [][2]string{{"response_format", "verbose_json"}}},
			{"verbose_json with segment and word granularities", append([][2]string{{"response_format", "verbose_json"}}, granularities...)},
			{"json with segment and word granularities", append([][2]string{{"response_format", "json"}}, granularities...)},
		}
		var found []string
		for _, f := range forms {
			status, reply := r.transcribe(t, wav, f.fields)
			var got struct {
				Segments []json.RawMessage `json:"segments"`
				Words    []json.RawMessage `json:"words"`
			}
			_ = json.Unmarshal(reply, &got)
			t.Logf("recorded: transcription, %s: HTTP %d, %d segments, %d words: %s",
				f.name, status, len(got.Segments), len(got.Words), reply)
			if status == http.StatusOK && len(got.Segments) > 0 {
				found = append(found, "segment ("+f.name+")")
			}
			if status == http.StatusOK && len(got.Words) > 0 {
				found = append(found, "word ("+f.name+")")
			}
		}
		if len(found) == 0 {
			t.Logf("recorded: transcription on %s returns no timestamps in any form tried", audioModel)
		} else {
			t.Logf("recorded: transcription on %s returns timestamps: %s", audioModel, strings.Join(found, ", "))
		}
	})
}

// samplingKeys pairs each preset sampling key with the name /props reports its default under.
var samplingKeys = []struct{ preset, props string }{
	{"temp", "temperature"},
	{"top-p", "top_p"},
	{"top-k", "top_k"},
	{"min-p", "min_p"},
}

// preset is the router's models preset: an INI file whose [*] section applies to every model
// and whose other sections are the models the router lists, in file order.
type preset struct {
	path     string
	global   map[string]string
	models   []string
	sections map[string]map[string]string
}

// loadPreset reads the preset at path. Lines starting with ; or # are comments.
func loadPreset(path string) (*preset, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	p := &preset{path: path, global: map[string]string{}, sections: map[string]map[string]string{}}
	var cur map[string]string
	s := bufio.NewScanner(f)
	for n := 1; s.Scan(); n++ {
		line := strings.TrimSpace(s.Text())
		switch {
		case line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#"):
		case strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]"):
			name := line[1 : len(line)-1]
			if name == "*" {
				cur = p.global
				continue
			}
			if _, ok := p.sections[name]; ok {
				return nil, fmt.Errorf("%s:%d: section [%s] repeats", path, n, name)
			}
			cur = map[string]string{}
			p.sections[name] = cur
			p.models = append(p.models, name)
		default:
			k, v, ok := strings.Cut(line, "=")
			if !ok || cur == nil {
				return nil, fmt.Errorf("%s:%d: %q is neither a section nor a key = value in one", path, n, line)
			}
			cur[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	if len(p.models) == 0 {
		return nil, fmt.Errorf("%s has no model sections", path)
	}
	return p, nil
}

// float returns key's value for model as a number: its section's, or else [*]'s. ok is false
// when neither sets it.
func (p *preset) float(model, key string) (v float64, ok bool, err error) {
	s, found := p.sections[model][key]
	if !found {
		s, found = p.global[key]
	}
	if !found {
		return 0, false, nil
	}
	v, err = strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false, fmt.Errorf("%s [%s] %s = %q is not a number", p.path, model, key, s)
	}
	return v, true, nil
}

// loadPin returns LLAMA_BUILD from the pin file at path, a shell file of NAME=value lines.
func loadPin(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	for line := range strings.Lines(string(b)) {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "LLAMA_BUILD="); ok {
			return v, nil
		}
	}
	return "", fmt.Errorf("%s sets no LLAMA_BUILD", path)
}

// router is the router under probe.
type router struct {
	base   string
	client *http.Client
}

// get decodes the JSON reply to GET path into out, failing t on a transport error or a status
// other than 200.
func (r *router) get(t *testing.T, path string, out any) {
	t.Helper()
	r.decode(t, http.MethodGet, path, "", nil, out)
}

// post sends body as JSON to path and decodes the reply into out, failing t as get does.
func (r *router) post(t *testing.T, path string, body, out any) {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	r.decode(t, http.MethodPost, path, "application/json", b, out)
}

// decode sends one request to path, with autoload=false, and decodes the JSON reply into out,
// failing t on a transport error, a status other than 200, or a reply that doesn't decode.
func (r *router) decode(t *testing.T, method, path, contentType string, body []byte, out any) {
	t.Helper()
	path = withNoAutoload(path)
	status, reply := r.send(t, method, path, contentType, body)
	if status != http.StatusOK {
		t.Fatalf("%s %s: %d: %s", method, path, status, reply)
	}
	if err := json.Unmarshal(reply, out); err != nil {
		t.Fatalf("%s %s: decoding the reply: %v: %s", method, path, err, reply)
	}
}

// send makes one request and returns its status and reply, failing t only on a transport
// error: the recorded subtests read a refusal rather than fail on it.
func (r *router) send(t *testing.T, method, path, contentType string, body []byte) (int, []byte) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, r.base+path, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := r.client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	reply, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("%s %s: reading the reply: %v", method, path, err)
	}
	return resp.StatusCode, reply
}

// withNoAutoload adds autoload=false to path's query. The router reads it on every request it
// routes to a model, and refuses one whose model isn't loaded rather than load it; requests it
// answers itself ignore it.
func withNoAutoload(path string) string {
	if strings.Contains(path, "?") {
		return path + "&autoload=false"
	}
	return path + "?autoload=false"
}

// chat sends one user message with content, a string or a list of content parts, to model on
// the OpenAI chat completions endpoint, and returns the reply's text, failing t if it is empty.
func (r *router) chat(t *testing.T, model string, content any, maxTokens int) string {
	t.Helper()
	var got struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}
	r.post(t, "/v1/chat/completions", map[string]any{
		"model":      model,
		"max_tokens": maxTokens,
		"messages":   []map[string]any{{"role": "user", "content": content}},
	}, &got)
	if len(got.Choices) == 0 {
		t.Fatalf("%s: the reply has no choices", model)
	}
	c := got.Choices[0]
	if strings.TrimSpace(c.Message.Content) == "" {
		t.Fatalf("%s: the reply's text is empty (finish_reason %q)", model, c.FinishReason)
	}
	return c.Message.Content
}

// transcribe uploads wav to the audio model on /v1/audio/transcriptions as OpenAI's API takes
// it, a multipart form with model, file, and fields in order, and returns the status and reply.
func (r *router) transcribe(t *testing.T, wav []byte, fields [][2]string) (int, []byte) {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if err := w.WriteField("model", audioModel); err != nil {
		t.Fatal(err)
	}
	for _, f := range fields {
		if err := w.WriteField(f[0], f[1]); err != nil {
			t.Fatal(err)
		}
	}
	fw, err := w.CreateFormFile("file", "phrase.wav")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(wav); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return r.send(t, http.MethodPost, withNoAutoload("/v1/audio/transcriptions"), w.FormDataContentType(), body.Bytes())
}

// checkPhrase fails t unless text, model's transcript of phrase.wav, carries its code.
// phrase.wav says "The access code is seven four two nine." The code may come back as digits or
// words, spaced or not: compared with everything but letters and digits stripped, either form
// passes.
func checkPhrase(t *testing.T, model, text string) {
	t.Helper()
	if got := alphanumeric(text); !strings.Contains(got, "7429") && !strings.Contains(got, "sevenfourtwonine") {
		t.Errorf("%s: the transcript %q doesn't carry the code 7429", model, text)
	}
}

// readClip returns testdata/phrase.wav, failing t unless it is shorter than maxClip.
func readClip(t *testing.T) []byte {
	t.Helper()
	wav := readFile(t, "testdata/phrase.wav")
	d, err := wavDuration(wav)
	if err != nil {
		t.Fatalf("testdata/phrase.wav: %v", err)
	}
	if d >= maxClip {
		t.Fatalf("testdata/phrase.wav runs %v, want under %v", d, maxClip)
	}
	return wav
}

// wavDuration reads a RIFF WAVE file's length from its fmt chunk's byte rate and its data
// chunk's size.
func wavDuration(wav []byte) (time.Duration, error) {
	if len(wav) < 12 || string(wav[0:4]) != "RIFF" || string(wav[8:12]) != "WAVE" {
		return 0, fmt.Errorf("not a RIFF WAVE file")
	}
	var byteRate uint32
	for rest := wav[12:]; len(rest) >= 8; {
		id, size := string(rest[0:4]), binary.LittleEndian.Uint32(rest[4:8])
		rest = rest[8:]
		if uint64(size) > uint64(len(rest)) {
			return 0, fmt.Errorf("chunk %q runs past the file's end", id)
		}
		switch id {
		case "fmt ":
			if size < 12 {
				return 0, fmt.Errorf("fmt chunk is %d bytes, too short", size)
			}
			byteRate = binary.LittleEndian.Uint32(rest[8:12])
		case "data":
			if byteRate == 0 {
				return 0, fmt.Errorf("data chunk comes before a fmt chunk with a byte rate")
			}
			return time.Duration(float64(size) / float64(byteRate) * float64(time.Second)), nil
		}
		// Chunks are padded to an even length.
		rest = rest[min(len(rest), int(size)+int(size&1)):]
	}
	return 0, fmt.Errorf("no data chunk")
}

// buildNumber parses a llama.cpp build, such as b11669 or the build_info b11669-620d2225c, into
// its number.
func buildNumber(build string) (int, error) {
	digits, _, _ := strings.Cut(strings.TrimPrefix(build, "b"), "-")
	if !strings.HasPrefix(build, "b") || digits == "" {
		return 0, fmt.Errorf("%q is not a llama.cpp build such as b11669", build)
	}
	n, err := strconv.Atoi(digits)
	if err != nil {
		return 0, fmt.Errorf("%q is not a llama.cpp build such as b11669", build)
	}
	return n, nil
}

// alphanumeric lowercases s and keeps only its letters and digits, so "7 4 2 9." and
// "Seven, four, two, nine" compare as 7429 and sevenfourtwonine.
func alphanumeric(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, s)
}

// readFile returns the file at path, failing t if it can't be read.
func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
