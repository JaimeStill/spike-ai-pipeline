//go:build integration

package app_test

import (
	"os"
	"testing"

	"github.com/standards-lab/go-core/process"

	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/outdir"
)

// granite is set A's embedding model and the dimensions of its space.
const (
	granite     = "mrutkows/granite-embedding-english-r2-GGUF:BF16"
	graniteDims = 768
)

// TestRun_AgainstConfig loads the configuration DOCINFER_CONFIG names, such as config/example,
// which serves set A on gb10-2, and runs docinfer over the committed docproc run end to end: the
// models check lists each endpoint it names, and every chunk is embedded, its vector of the
// space's dimensions, set A's 768 for its embedding model. It never loads a model.
func TestRun_AgainstConfig(t *testing.T) {
	dir := os.Getenv("DOCINFER_CONFIG")
	if dir == "" {
		t.Skip("DOCINFER_CONFIG is unset: set it to a docinfer configuration directory, such as ../../../config/example")
	}
	out := t.TempDir()
	stdout, _ := run(t, process.ExitOK, "run", docproc, "--config", dir, "--out", out)
	t.Logf("docinfer run:\n%s", stdout)
	wantLine(t, stdout, `embeddings\s+\S+\s+\S+\s+\S+`)

	m, lines := result(t, out)
	t.Logf("embedding space: %+v; counts: %+v", *m.Space, m.Counts)
	if m.Space.Dimensions < 1 || len(lines) != m.Counts.Lines || m.Counts.Lines == 0 {
		t.Fatalf("space %+v, %d lines, counts %+v; want vectors and every line", *m.Space, len(lines), m.Counts)
	}
	if m.Space.Model == granite && m.Space.Dimensions != graniteDims {
		t.Errorf("%s: dimensions %d, want %d", granite, m.Space.Dimensions, graniteDims)
	}
	for _, l := range lines {
		if l.Error != "" {
			t.Errorf("chunk %q rejected: %s", l.Text, l.Error)
			continue
		}
		v, err := outdir.DecodeVector(l.Vector)
		if err != nil || len(v) != m.Space.Dimensions {
			t.Errorf("chunk %q: vector of %d dimensions (%v), want %d", l.Text, len(v), err, m.Space.Dimensions)
		}
	}
}
