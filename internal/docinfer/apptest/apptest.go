package apptest

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"slices"
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
}

// Serve starts an Endpoint answering as fake says, closed when t ends.
func Serve(t testing.TB, fake Fake) *Endpoint {
	t.Helper()
	e := &Endpoint{fake: fake}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/models", e.models)
	mux.HandleFunc("POST /v1/embeddings", e.embed)
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
		"usage": map[string]int{"prompt_tokens": len(req.Input), "total_tokens": len(req.Input)},
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
