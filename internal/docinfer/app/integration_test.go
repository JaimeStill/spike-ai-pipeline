//go:build integration

package app_test

import (
	"os"
	"strings"
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
// models check lists each endpoint it names, the fixture's image is sent to the vision model and
// its text is not empty, the fixture's three audio windows are sent to the audio model and their
// joined transcript is not empty, and every chunk is embedded, its vector of the space's dimensions, set
// A's 768 for its embedding model. It never loads a model.
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
	vc := m.Counts.Vision
	if vc == nil || vc.Images != 1 || vc.Refused != 0 || vc.Described+vc.Held != 1 {
		t.Fatalf("vision counts = %+v, want the fixture's one image described", vc)
	}
	var vision []string
	for _, l := range lines {
		if l.Source.Kind == outdir.SourceVision {
			vision = append(vision, l.Text)
		}
	}
	t.Logf("vision text of %s, incomplete %d: %q", pngRef, vc.Incomplete, vision)
	if len(vision) == 0 || strings.TrimSpace(strings.Join(vision, "")) == "" {
		t.Errorf("vision lines = %q, want the image's text, not empty", vision)
	}
	tc := m.Counts.Transcription
	if tc == nil || m.Counts.Windows != 3 || tc.Distinct != 3 || tc.Failed != 0 || tc.Transcribed+tc.Held != 3 || tc.Transcripts != 1 {
		t.Fatalf("transcription counts = %+v, want the fixture's three windows transcribed and joined", tc)
	}
	var transcript []string
	for _, l := range lines {
		if l.Source.Kind == outdir.SourceTranscript {
			transcript = append(transcript, l.Text)
		}
	}
	t.Logf("transcript of %s, seams %+v: %q", talkWAV, m.Transcripts[0].Seams, transcript)
	if len(transcript) == 0 || strings.TrimSpace(strings.Join(transcript, "")) == "" {
		t.Errorf("transcript lines = %q, want the windows' transcript, not empty", transcript)
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
