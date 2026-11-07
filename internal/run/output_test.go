package run_test

import (
	"bytes"
	"context"
	"errors"
	"iter"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"github.com/JaimeStill/spike-ai-pipeline-processing/dispatch"
	"github.com/JaimeStill/spike-ai-pipeline-processing/route"

	"github.com/JaimeStill/spike-ai-pipeline/internal/run"
	"github.com/JaimeStill/spike-ai-pipeline/internal/rundir"
	"github.com/JaimeStill/spike-ai-pipeline/internal/schema"
	"github.com/JaimeStill/spike-ai-pipeline/internal/stages"
)

// failing is a stage that fails every part it is given with err.
type failing struct{ err error }

func (failing) Identity() dispatch.Identity {
	return dispatch.Identity{Name: "failing", Version: "1", Config: "0"}
}

func (s failing) Prepare(context.Context, dispatch.Part, string) iter.Seq2[dispatch.Item, error] {
	return func(yield func(dispatch.Item, error) bool) { yield(dispatch.Item{}, s.err) }
}

// An input whose stage fails with an error of several lines, NUL and other control bytes among
// them, as Ghostscript's raw PNG output makes one, prints one terminal line, its error cut to
// its first line with those bytes replaced; the record keeps the error whole, and the report
// shows it escaped.
func TestExecute_ErrorIsOneTerminalLine(t *testing.T) {
	root := t.TempDir()
	path, out := filepath.Join(root, "hello.txt"), filepath.Join(root, "out")
	if err := os.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stageErr := errors.New("render: no PNG on stdout: \x89PNG\r\x00\x1b[31m\x7fend\n\x1a\nIHDR\x00\x00\x01\x00 tail")
	set := stages.Set{Stages: dispatch.Stages{route.Text: failing{stageErr}}}
	var w bytes.Buffer

	if err := run.Execute(context.Background(), &w, set, out, path); err != nil {
		t.Fatal(err)
	}

	// The input's line is the only one that names it, and the summary's blank line follows it.
	terminal := strings.Split(w.String(), "\n")
	at := -1
	for i, line := range terminal {
		if strings.Contains(line, path) {
			if at >= 0 {
				t.Fatalf("terminal names the input twice: %q", w.String())
			}
			at = i
		}
	}
	if at < 0 || terminal[at+1] != "" {
		t.Fatalf("terminal = %q, want one line for the input, then the summary", w.String())
	}
	line := terminal[at]
	for _, c := range line {
		if !unicode.IsPrint(c) {
			t.Errorf("terminal line %q holds %U", line, c)
		}
	}
	if want := "  text  failed: render: no PNG on stdout: \uFFFDPNG\uFFFD\uFFFD\uFFFD[31m\uFFFDend…  "; !strings.Contains(line, want) {
		t.Errorf("terminal line = %q, want it to hold %q", line, want)
	}

	id, err := rundir.Latest(out)
	if err != nil {
		t.Fatal(err)
	}
	_, evs, err := rundir.Load(out, id)
	if err != nil {
		t.Fatal(err)
	}
	var recorded string
	for _, ev := range evs {
		if ev.Event == schema.PartEvent {
			recorded = ev.Error
		}
	}
	if want := strings.ToValidUTF8(stageErr.Error(), "\uFFFD"); recorded != want {
		t.Errorf("record's error = %q, want %q", recorded, want)
	}
	page, err := os.ReadFile(filepath.Join(out, "runs", id, "report.html"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.ContainsRune(page, 0) {
		t.Error("the report holds a NUL byte")
	}
	if !bytes.Contains(page, []byte("IHDR")) {
		t.Error("the report doesn't show the error's later lines")
	}
}

// Brief trims an error, replaces its unprintable runes, keeps its first line, and cuts it at
// ErrorRunes runes, an ellipsis saying what it left out.
func TestBrief(t *testing.T) {
	long := strings.Repeat("é", run.ErrorRunes+1)
	for _, tc := range []struct{ in, want string }{
		{"plain", "plain"},
		{"  padded\n", "padded"},
		{"tab\there", "tab\uFFFDhere"},
		{"two\nlines", "two…"},
		{long, long[:2*run.ErrorRunes] + "…"},
		{long[:2*run.ErrorRunes], long[:2*run.ErrorRunes]},
	} {
		if got := run.Brief(errors.New(tc.in)); got != tc.want {
			t.Errorf("Brief(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
