package config_test

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/config"
)

// example is the committed configuration of set A on gb10-2.
var example = filepath.Join("..", "..", "..", "config", "example")

// clean empties the overrides the example's load could read.
func clean(t *testing.T) {
	t.Helper()
	for _, name := range []string{"DOCINFER_ENV", "DOCINFER_EMBEDDINGS_MODEL", "DOCINFER_VISION_MODEL", "DOCINFER_TRANSCRIPTION_MODEL", "DOCINFER_ENDPOINTS_GB10_2_TOKEN"} {
		t.Setenv(name, "")
	}
}

// The committed example loads and validates: set A's three models on gb10-2's router, with its
// autoload=false query, at the defaults.
func TestLoad_Example(t *testing.T) {
	clean(t)
	cfg, err := config.Load(example)
	if err != nil {
		t.Fatal(err)
	}
	e := cfg.Endpoints["gb10-2"]
	if e == nil || e.BaseURL != "http://gb10-2:8080/v1" || e.Format != config.Completions || e.Values().Encode() != "autoload=false" {
		t.Fatalf("endpoint gb10-2 = %+v, want the router's API root, completions, autoload=false", e)
	}
	for _, c := range []struct {
		name, model string
		got         *config.Capability
	}{
		{"embeddings", "mrutkows/granite-embedding-english-r2-GGUF:BF16", &cfg.Embeddings.Capability},
		{"vision", "google/gemma-4-26B-A4B-it-qat-q4_0-gguf:Q4_0", &cfg.Vision.Capability},
		{"transcription", "ggml-org/gemma-4-E4B-it-GGUF:Q8_0", &cfg.Transcription.Capability},
	} {
		if c.got.Endpoint != "gb10-2" || c.got.Model != c.model || *c.got.Concurrency != config.DefaultConcurrency {
			t.Errorf("%s = %+v, want gb10-2's %s, concurrency %d", c.name, c.got, c.model, config.DefaultConcurrency)
		}
	}
	if *cfg.Embeddings.Batch != config.DefaultBatch || *cfg.Embeddings.ChunkSize != config.DefaultChunkSize {
		t.Errorf("embeddings batch, chunk_size = %d, %d, want the defaults", *cfg.Embeddings.Batch, *cfg.Embeddings.ChunkSize)
	}
	if cfg.Vision.Prompt != config.DefaultVisionPrompt || *cfg.Vision.MaxTokens != config.DefaultVisionMaxTokens {
		t.Errorf("vision prompt, max_tokens = %q, %d, want the defaults", cfg.Vision.Prompt, *cfg.Vision.MaxTokens)
	}
	if cfg.Transcription.Prompt != "" || cfg.Transcription.Language != "" {
		t.Errorf("transcription prompt, language = %q, %q, want none", cfg.Transcription.Prompt, cfg.Transcription.Language)
	}
	if got := cfg.Env.Endpoints["gb10-2"].Token; got != "DOCINFER_ENDPOINTS_GB10_2_TOKEN" {
		t.Errorf("gb10-2's token override = %q, want DOCINFER_ENDPOINTS_GB10_2_TOKEN", got)
	}
}

// The example's secrets file, copied to secrets.json, carries no token: it loads, and sends none.
func TestLoad_ExampleSecrets(t *testing.T) {
	clean(t)
	dir := t.TempDir()
	if err := os.CopyFS(dir, os.DirFS(example)); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(dir, "secrets.json.example"), filepath.Join(dir, "secrets.json")); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if tok := cfg.Endpoints["gb10-2"].Token; tok != "" {
		t.Errorf("token = %q, want none", tok)
	}
	t.Setenv("DOCINFER_ENDPOINTS_GB10_2_TOKEN", "from-env")
	if cfg, err = config.Load(dir); err != nil || cfg.Endpoints["gb10-2"].Token != "from-env" {
		t.Errorf("Load with the token override = %v, %v, want the override's token", cfg, err)
	}
}

// Every format an endpoint may name makes a client.
func TestEndpoint_Client(t *testing.T) {
	for _, f := range []string{config.Completions, config.Messages, config.Responses} {
		e := config.Endpoint{BaseURL: "http://localhost:8080/v1", Format: f, Token: "t", Query: map[string]string{"autoload": "false"}}
		if _, err := e.Client(http.DefaultClient); err != nil {
			t.Errorf("%s: Client() = %v", f, err)
		}
	}
}

// Two endpoint keys whose overrides would share names are refused.
func TestLoad_EndpointOverridesCollide(t *testing.T) {
	clean(t)
	dir := t.TempDir()
	body := `{"endpoints": {"a-b": {"base_url": "http://x/v1", "format": "completions"}, "a_b": {"base_url": "http://y/v1", "format": "completions"}},
	"embeddings": {"endpoint": "a-b", "model": "e"}}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Load(dir); err == nil {
		t.Error("Load() = nil, want the endpoints' shared overrides refused")
	}
}
