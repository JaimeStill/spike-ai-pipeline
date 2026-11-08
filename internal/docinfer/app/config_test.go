package app_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/standards-lab/go-core/process"

	docinfertest "github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/apptest"
)

// Each configuration error stops the run with exit 1, naming what is wrong, before any request
// and before the table.
func TestRun_ConfigErrors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config string // config.json, %[1]q the endpoint's URL
		env    map[string]string
		want   []string
	}{
		{
			name:   "missing embeddings",
			config: `{"endpoints": {"fake": {"base_url": %[1]q, "format": "completions"}}, "vision": {"endpoint": "fake", "model": "v"}}`,
			want:   []string{"embeddings: no capability configured; it is required"},
		},
		{
			name:   "unknown endpoint",
			config: `{"endpoints": {"fake": {"base_url": %[1]q, "format": "completions"}}, "embeddings": {"endpoint": "fake", "model": "e"}, "vision": {"endpoint": "elsewhere", "model": "v"}}`,
			want:   []string{`vision: unknown endpoint "elsewhere"`},
		},
		{
			name:   "unknown format",
			config: `{"endpoints": {"fake": {"base_url": %[1]q, "format": "chat"}}, "embeddings": {"endpoint": "fake", "model": "e"}}`,
			want:   []string{`endpoints "fake": unknown format "chat"`},
		},
		{
			name:   "no model and no endpoint",
			config: `{"endpoints": {"fake": {"base_url": %[1]q, "format": "completions"}}, "embeddings": {"endpoint": "fake"}, "transcription": {"model": "t"}}`,
			want:   []string{"embeddings: no model", "transcription: no endpoint"},
		},
		{
			name:   "bad settings",
			config: `{"endpoints": {"fake": {"base_url": %[1]q, "format": "messages"}}, "embeddings": {"endpoint": "fake", "model": "e", "batch": 0, "chunk_size": -5}, "vision": {"endpoint": "fake", "model": "v", "max_tokens": 0, "concurrency": 0}}`,
			want:   []string{"embeddings: batch 0 is below 1", "embeddings: chunk_size -5 is below 1", "vision: max_tokens 0 is below 1", "vision: concurrency 0 is below 1"},
		},
		{
			name:   "unknown reasoning",
			config: `{"endpoints": {"fake": {"base_url": %[1]q, "format": "completions"}}, "embeddings": {"endpoint": "fake", "model": "e"}, "vision": {"endpoint": "fake", "model": "v", "reasoning": "maximal"}}`,
			want:   []string{`vision: unknown reasoning "maximal"; want one of ["none" "low" "medium" "high" "xhigh"]`},
		},
		{
			name:   "unknown reasoning override",
			config: `{"endpoints": {"fake": {"base_url": %[1]q, "format": "completions"}}, "embeddings": {"endpoint": "fake", "model": "e"}, "vision": {"endpoint": "fake", "model": "v"}}`,
			env:    map[string]string{"DOCINFER_VISION_REASONING": "off"},
			want:   []string{`vision: unknown reasoning "off"`},
		},
		{
			name:   "base url with a query",
			config: `{"endpoints": {"fake": {"base_url": %[1]q, "format": "responses"}}, "embeddings": {"endpoint": "fake", "model": "e"}}`,
			want:   []string{"has a query or fragment; set query instead"},
		},
		{
			name:   "unknown key",
			config: `{"endpoints": {"fake": {"base_url": %[1]q, "format": "completions", "timeout": "1m"}}, "embeddings": {"endpoint": "fake", "model": "e"}}`,
			want:   []string{`unknown field "timeout"`},
		},
		{
			name:   "override that doesn't parse",
			config: `{"endpoints": {"fake": {"base_url": %[1]q, "format": "completions"}}, "embeddings": {"endpoint": "fake", "model": "e"}}`,
			env:    map[string]string{"DOCINFER_EMBEDDINGS_BATCH": "many"},
			want:   []string{"DOCINFER_EMBEDDINGS_BATCH"},
		},
		{
			name:   "bad override",
			config: `{"endpoints": {"fake": {"base_url": %[1]q, "format": "completions"}}, "embeddings": {"endpoint": "fake", "model": "e"}}`,
			env:    map[string]string{"DOCINFER_ENDPOINTS_FAKE_FORMAT": "grpc"},
			want:   []string{`endpoints "fake": unknown format "grpc"`},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clean(t)
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			e := docinfertest.Serve(t, docinfertest.Fake{Models: setA})
			url := e.URL()
			if tc.name == "base url with a query" {
				url += "?key=1"
			}
			cfg := configDir(t, map[string]string{"config.json": fmt.Sprintf(tc.config, url)})

			stdout, stderr := run(t, process.ExitFailure, "run", docproc, "--config", cfg, "--out", t.TempDir())

			for _, want := range tc.want {
				if !strings.Contains(stderr, want) {
					t.Errorf("stderr = %q, want it to contain %q", stderr, want)
				}
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if reqs := e.Requests(); len(reqs) != 0 {
				t.Errorf("requests = %v, want none", reqs)
			}
		})
	}
}

// A --config that isn't a directory fails, rather than reading as an empty configuration.
func TestRun_ConfigDirMissing(t *testing.T) {
	clean(t)

	_, stderr := run(t, process.ExitFailure, "run", docproc, "--config", filepath.Join(t.TempDir(), "absent"), "--out", t.TempDir())

	if !strings.Contains(stderr, "no such file or directory") {
		t.Errorf("stderr = %q, want the missing directory named", stderr)
	}
}

// The layers apply in order: the DOCINFER_ENV overlay over config.json, the secrets file over
// both, and an override over every file; an override of a capability the files leave unset
// sets it.
func TestRun_ConfigLayers(t *testing.T) {
	clean(t)
	e := docinfertest.Serve(t, docinfertest.Fake{Models: []docinfertest.Model{
		{ID: "overlay-embed", Status: "loaded"}, {ID: "env-vision", Status: "loaded"}, {ID: audioModel, Status: "loaded"},
	}})
	cfg := configDir(t, map[string]string{
		"config.json":       fullConfig(e.URL()),
		"config.test.json":  `{"embeddings": {"model": "overlay-embed"}, "vision": {"model": "overlay-vision"}}`,
		"secrets.json":      `{"endpoints": {"fake": {"token": "base-token"}}}`,
		"secrets.test.json": `{"endpoints": {"fake": {"token": "overlay-token"}}}`,
	})
	t.Setenv("DOCINFER_ENV", "test")
	t.Setenv("DOCINFER_VISION_MODEL", "env-vision")

	stdout, _ := run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", t.TempDir())

	wantLine(t, stdout, row("embeddings", "fake", "overlay-embed", "loaded"))
	wantLine(t, stdout, row("vision", "fake", "env-vision", "loaded"))
	wantLine(t, stdout, row("transcription", "fake", audioModel, "loaded"))
	if reqs := listings(e.Requests()); len(reqs) != 1 || reqs[0].Authorization != "Bearer overlay-token" {
		t.Errorf("listings = %v, want one, with the secrets overlay's token", reqs)
	}
	for _, r := range e.Requests() {
		if r.Authorization != "Bearer overlay-token" {
			t.Errorf("request %v, want the secrets overlay's token", r)
		}
	}

	// The overrides alone set the vision capability a configuration leaves unset.
	clean(t)
	cfg = configDir(t, map[string]string{"config.json": fmt.Sprintf(
		`{"endpoints": {"fake": {"base_url": %q, "format": "completions"}}, "embeddings": {"endpoint": "fake", "model": "overlay-embed"}}`, e.URL())})
	t.Setenv("DOCINFER_VISION_ENDPOINT", "fake")
	t.Setenv("DOCINFER_VISION_MODEL", "env-vision")

	stdout, _ = run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", t.TempDir())

	wantLine(t, stdout, row("vision", "fake", "env-vision", "loaded"))
	wantLine(t, stdout, row("transcription", "-", "-", "skipped"))
}
