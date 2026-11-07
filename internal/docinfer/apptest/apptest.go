package apptest

import (
	"encoding/json"
	"errors"
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
// 404 there.
type Fake struct {
	Models    []Model
	NoListing bool
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
	server   *httptest.Server
	fake     Fake
	mu       sync.Mutex
	requests []Request
}

// Serve starts an Endpoint answering as fake says, closed when t ends.
func Serve(t testing.TB, fake Fake) *Endpoint {
	t.Helper()
	e := &Endpoint{fake: fake}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/models", e.models)
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
