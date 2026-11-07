package app_test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/standards-lab/go-core/process"

	"github.com/JaimeStill/spike-ai-pipeline/internal/apptest"
)

// report runs docproc report with args, and fails the test unless it exits with want. It
// returns the terminal's output.
func report(t *testing.T, want int, args ...string) (stdout, stderr string) {
	t.Helper()
	a, out, errs, built := newApp(t)
	if code := a.Run(context.Background(), append([]string{"report"}, args...)); code != want {
		t.Fatalf("code = %d, want %d; stderr = %q", code, want, errs)
	}
	if log := built.Log(); len(log) != 0 {
		t.Errorf("nodes built = %q, want none", log)
	}
	return out.String(), errs.String()
}

// readReport returns the report of the run id under the run directory out.
func readReport(t *testing.T, out, id string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(out, "runs", id, "report.html"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// A run writes its report into its own directory as it ends, and its summary names it; docproc
// report renders the same page again from the record, the latest run's by default, and another
// run's by --run.
func TestReport_WrittenAndRerendered(t *testing.T) {
	in, out := inputs(t, "hello.txt", "notes.zip")
	lookup := sevenZip(t)
	stdout := runOnce(t, lookup, process.ExitOK, in, "--out", out, "--workers", "1")
	runOnce(t, lookup, process.ExitOK, in, "--out", out, "--workers", "1")
	rs := runs(t, out)
	first, second := rs[0].manifest.Run, rs[1].manifest.Run

	wantLine(t, stdout, `report runs/`+regexp.QuoteMeta(first)+`/report\.html`)
	written := map[string]string{first: readReport(t, out, first), second: readReport(t, out, second)}
	if !strings.Contains(written[first], first) || !strings.Contains(written[first], "notes.txt") {
		t.Errorf("the first run's report doesn't name its run and nested part:\n%s", written[first])
	}
	if !strings.Contains(written[second], `data-outcome="seen"`) {
		t.Error("the re-run's report shows no seen part")
	}
	for _, id := range []string{first, second} {
		if err := os.Remove(filepath.Join(out, "runs", id, "report.html")); err != nil {
			t.Fatal(err)
		}
	}

	got, _ := report(t, process.ExitOK, out)
	if got != "runs/"+second+"/report.html\n" {
		t.Errorf("stdout = %q, want the latest run's report", got)
	}
	got, _ = report(t, process.ExitOK, out, "--run", first)
	if got != "runs/"+first+"/report.html\n" {
		t.Errorf("stdout = %q, want the first run's report", got)
	}
	for id, want := range written {
		if readReport(t, out, id) != want {
			t.Errorf("run %s's report rendered again differs from the one its run wrote", id)
		}
	}
}

// Each link in the report is relative to it, so the run directory moved elsewhere still
// resolves every one.
func TestReport_LinksRelative(t *testing.T) {
	in, out := inputs(t, "hello.txt", "evil.zip")
	runOnce(t, apptest.NoTools, process.ExitOK, in, "--out", out, "--workers", "1")
	id := runs(t, out)[0].manifest.Run
	moved := filepath.Join(t.TempDir(), "moved")
	if err := os.Rename(out, moved); err != nil {
		t.Fatal(err)
	}

	page := readReport(t, moved, id)

	links := regexp.MustCompile(`(?:href|src)="([^"]*)"`).FindAllStringSubmatch(page, -1)
	if len(links) != 3 {
		t.Errorf("%d links, want 3: hello.txt's text, and evil.zip's kept file and flags", len(links))
	}
	for _, l := range links {
		if !strings.HasPrefix(l[1], "../../") {
			t.Errorf("link %q is not relative to the report", l[1])
		}
		if _, err := os.Stat(filepath.Join(moved, "runs", id, filepath.FromSlash(l[1]))); err != nil {
			t.Errorf("link %q doesn't resolve once moved: %v", l[1], err)
		}
	}
}

// A run directory with no run, a run id it doesn't hold, and a missing argument each fail.
func TestReport_Errors(t *testing.T) {
	empty := t.TempDir()
	if err := os.Mkdir(filepath.Join(empty, "runs"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, stderr := report(t, process.ExitFailure, empty)
	if !strings.Contains(stderr, "holds no run") {
		t.Errorf("stderr = %q, want no run", stderr)
	}
	report(t, process.ExitFailure, empty, "--run", "nope")
	report(t, process.ExitUsage)
}
