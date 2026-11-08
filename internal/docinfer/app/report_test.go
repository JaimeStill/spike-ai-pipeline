package app_test

import (
	"bytes"
	"context"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/standards-lab/go-core/process"

	docinfertest "github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/apptest"
)

// report runs docinfer report with args, and fails the test unless it exits with want. It
// returns the terminal's output, and fails the test when the command built any node: it reads
// docinfer's directory alone, and needs no endpoint.
func report(t *testing.T, want int, args ...string) (stdout, stderr string) {
	t.Helper()
	a, out, errs, built := newApp(t)
	if code := a.Run(context.Background(), append([]string{"report"}, args...)); code != want {
		t.Fatalf("code = %d, want %d; stdout = %q; stderr = %q", code, want, out, errs)
	}
	if log := built.Log(); len(log) != 0 {
		t.Errorf("nodes built = %q, want none", log)
	}
	return out.String(), errs.String()
}

// readReport returns the report of the run id in docinfer's directory out.
func readReport(t *testing.T, out, id string) string {
	t.Helper()
	return string(readFile(t, filepath.Join(out, "runs", id, "report.html")))
}

// links returns the page's href and src values, unescaped, as a browser reads them.
func links(page string) []string {
	var out []string
	for _, m := range regexp.MustCompile(`(?:href|src)="([^"]*)"`).FindAllStringSubmatch(page, -1) {
		out = append(out, html.UnescapeString(m[1]))
	}
	return out
}

// A run writes its report into its own directory as it ends, and its last line names it; docinfer
// report renders the same bytes again, the latest run's by default, and another run's by --run.
func TestReport_WrittenAndRerendered(t *testing.T) {
	clean(t)
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA, Transcribe: seamTexts(t, -1, 0)})
	cfg := configDir(t, map[string]string{"config.json": fullConfig(e.URL())})
	out := t.TempDir()

	stdout, _ := run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)
	run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)
	ids := runs(t, out)
	if len(ids) != 2 {
		t.Fatalf("runs = %q, want two", ids)
	}
	first, second := ids[0], ids[1]

	wantLine(t, stdout, `report runs/`+regexp.QuoteMeta(first)+`/report\.html`)
	written := map[string]string{first: readReport(t, out, first), second: readReport(t, out, second)}
	for id, page := range written {
		if !strings.Contains(page, id) {
			t.Errorf("run %s's report doesn't name its run", id)
		}
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

// The page shows the image inline beside its vision text, by a link relative to the report that
// resolves to the docproc run's image; the transcript beside its windows, each linked likewise;
// each source's chunk count; and the stored texts linked into docinfer's directory. The terminal
// prints a line per input, with its images, windows, chunks, failures, and time, and a closing
// summary of each capability's requests and failures.
func TestReport_Content(t *testing.T) {
	clean(t)
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA, Transcribe: seamTexts(t, -1, 0)})
	cfg := configDir(t, map[string]string{"config.json": fullConfig(e.URL())})
	out := t.TempDir()

	stdout, _ := run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)

	m, chunks := result(t, out)
	reportDir := filepath.Join(out, "runs", m.Run)
	page := readReport(t, out, m.Run)
	png := readFile(t, filepath.Join(docproc, filepath.FromSlash(pngRef)))

	var img string
	if s := regexp.MustCompile(`<img src="([^"]*)"`).FindStringSubmatch(page); s != nil {
		img = html.UnescapeString(s[1])
	}
	if !strings.HasSuffix(img, pngRef) || filepath.IsAbs(img) {
		t.Fatalf("image src = %q, want a relative link ending in %s", img, pngRef)
	}
	if b := readFile(t, filepath.Join(reportDir, filepath.FromSlash(img))); !bytes.Equal(b, png) {
		t.Errorf("image src %q resolves to a file other than the docproc run's image", img)
	}
	for _, l := range links(page) {
		if filepath.IsAbs(l) || strings.Contains(l, "://") {
			t.Errorf("link %q is not relative to the report", l)
			continue
		}
		if _, err := os.Stat(filepath.Join(reportDir, filepath.FromSlash(l))); err != nil {
			t.Errorf("link %q doesn't resolve from the report: %v", l, err)
		}
	}
	for _, ref := range windowRefs {
		if !strings.Contains(page, ref+`"`) {
			t.Errorf("the page doesn't link window %s", ref)
		}
	}
	for _, want := range []string{
		html.EscapeString(docinfertest.VisionText(png)),
		joinedText,
		"seams matched 2, unmatched 0",
		// report.pdf: page 1's text, its image's vision text, and its metadata, a chunk each.
		`<code>` + reportPDF + `</code> <span class="muted">sources 3, chunks 3</span>`,
		// talk.wav: its transcript and its metadata.
		`<code>` + talkWAV + `</code> <span class="muted">sources 2, chunks 2</span>`,
		`<span class="muted">chunks 1</span>`,
		"chunk lines 16",
		"None: no image refused, no window failed, and no chunk rejected.",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the page has no %s", want)
		}
	}
	for _, l := range chunks {
		if strings.Contains(page, l.Vector) {
			t.Errorf("the page holds the vector of chunk %q", l.Text)
		}
	}

	for _, line := range []string{
		`/srv/inputs/hello\.txt  images 0  windows 0  chunks 2  failures 0  \S+`,
		`/srv/inputs/copy/hello\.txt  images 0  windows 0  chunks 2  failures 0  \S+`,
		`/srv/inputs/report\.pdf  images 1  windows 0  chunks 3  failures 0  \S+`,
		`/srv/inputs/notes\.docx  images 0  windows 0  chunks 3  failures 0  \S+`,
		`/srv/inputs/talk\.wav  images 0  windows 3  chunks 2  failures 0  \S+`,
		`/srv/inputs/blob\.bin  images 0  windows 0  chunks 1  failures 0  \S+`,
		`/srv/inputs/broken\.pdf  images 0  windows 0  chunks 2  failures 0  \S+`,
		`/srv/inputs/locked\.bin  images 0  windows 0  chunks 1  failures 0  \S+`,
		`CAPABILITY\s+REQUESTS\s+FAILURES`,
		`vision\s+1\s+0`,
		`transcription\s+3\s+0`,
		`embeddings\s+1\s+0`,
	} {
		wantLine(t, stdout, line)
	}
}

// An image vision refuses, a window transcription refuses, and a chunk the endpoint rejects are
// each recorded: the manifest's failures hold the image and the window, the page lists all three
// with why, and each input's terminal line and the closing summary count them. A re-run, every
// failure held, sends nothing and ends every input at once, its lines the same.
func TestReport_Failures(t *testing.T) {
	clean(t)
	bad := func(in string) bool { return strings.Contains(in, "Revenue") }
	e := docinfertest.Serve(t, docinfertest.Fake{
		Models: setA, Reject: bad, Transcribe: seamTexts(t, 1, 400),
		Vision: func(docinfertest.VisionRequest) docinfertest.VisionReply {
			return docinfertest.VisionReply{Status: 400}
		},
	})
	cfg := configDir(t, map[string]string{"config.json": fullConfig(e.URL())})
	out := t.TempDir()

	stdout, _ := run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)

	m, _ := result(t, out)
	if len(m.Failures) != 2 ||
		m.Failures[0].Capability != "vision" || m.Failures[0].Input != reportPDF || m.Failures[0].Ref != pngRef ||
		!strings.Contains(m.Failures[0].Error, "400") ||
		m.Failures[1].Capability != "transcription" || m.Failures[1].Input != talkWAV || m.Failures[1].Ref != windowRefs[1] ||
		!strings.Contains(m.Failures[1].Error, "400") {
		t.Errorf("failures = %+v, want report.pdf's image, then talk.wav's window 1", m.Failures)
	}
	page := readReport(t, out, m.Run)
	for _, want := range []string{
		`<tr><td>refused image</td><td><code>` + reportPDF + `</code></td>`,
		`<tr><td>failed window</td><td><code>` + talkWAV + `</code></td>`,
		`<tr><td>rejected chunk</td><td><code>` + reportPDF + `</code></td><td>chunk 0 of <code>outputs/6466e450a16b77b865c5829d6b6c56d9f892956475642dbeb9ccc4340fe01b15/document-e88a7a4d9af5/1.txt</code></td>`,
		"image refused", "window refused", "input rejected",
		`<code>` + reportPDF + `</code> <span class="muted">sources 3, chunks 2</span> <span class="badge bad">failures 2</span>`,
		`<code>` + talkWAV + `</code> <span class="muted">sources 3, chunks 2</span> <span class="badge bad">failures 1</span>`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the page has no %s", want)
		}
	}
	// The refused image still shows inline, by its link into the docproc run.
	if !strings.Contains(page, `<img src="`) || !strings.Contains(page, pngRef+`" alt="refused image"`) {
		t.Error("the page doesn't show the refused image")
	}

	lines := []string{
		`/srv/inputs/report\.pdf  images 1  windows 0  chunks 2  failures 2  \S+`,
		`/srv/inputs/talk\.wav  images 0  windows 3  chunks 2  failures 1  \S+`,
		`/srv/inputs/hello\.txt  images 0  windows 0  chunks 2  failures 0  \S+`,
	}
	for _, line := range lines {
		wantLine(t, stdout, line)
	}
	wantLine(t, stdout, `vision\s+1\s+1`)
	wantLine(t, stdout, `transcription\s+3\s+1`)
	wantLine(t, stdout, `embeddings\s+\d+\s+1`)

	again, _ := run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)
	for _, line := range lines {
		wantLine(t, again, line)
	}
	wantLine(t, again, `vision\s+0\s+1`)
	wantLine(t, again, `transcription\s+0\s+1`)
	wantLine(t, again, `embeddings\s+0\s+1`)
}

// A skipped capability's summary row says so, with no requests or failures.
func TestReport_SummarySkipped(t *testing.T) {
	clean(t)
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA})
	cfg := configDir(t, map[string]string{"config.json": embedConfig(e.URL(), "")})

	stdout, _ := run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", t.TempDir())

	wantLine(t, stdout, `vision\s+-\s+-\s+skipped: not configured`)
	wantLine(t, stdout, `transcription\s+-\s+-\s+skipped: not configured`)
	wantLine(t, stdout, `/srv/inputs/report\.pdf  images 1  windows 0  chunks 2  failures 0  \S+`)
}

// A directory with no run, a run id it doesn't hold, a directory that doesn't exist, and a
// missing argument each fail; none makes a directory.
func TestReport_Errors(t *testing.T) {
	clean(t)
	empty := t.TempDir()
	if err := os.Mkdir(filepath.Join(empty, "runs"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, stderr := report(t, process.ExitFailure, empty)
	if !strings.Contains(stderr, "holds no run") {
		t.Errorf("stderr = %q, want no run", stderr)
	}
	_, stderr = report(t, process.ExitFailure, empty, "--run", "nope")
	if !strings.Contains(stderr, "no such file or directory") {
		t.Errorf("stderr = %q, want the run's manifest missing", stderr)
	}
	missing := filepath.Join(t.TempDir(), "missing")
	report(t, process.ExitFailure, missing, "--run", "nope")
	if _, err := os.Stat(missing); err == nil {
		t.Error("report made the directory it was given")
	}
	report(t, process.ExitUsage)
}

// A --run that is a path rather than one run's id is a usage error, even one that resolves to a
// real run, as ../runs/<id> does.
func TestReport_RunIDOutsideRuns(t *testing.T) {
	clean(t)
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA})
	cfg := configDir(t, map[string]string{"config.json": embedConfig(e.URL(), "")})
	out := t.TempDir()
	run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)
	id := runs(t, out)[0]
	for _, bad := range []string{"../runs/" + id, "..", "."} {
		t.Run(bad, func(t *testing.T) {
			_, stderr := report(t, process.ExitUsage, out, "--run", bad)
			if !strings.Contains(stderr, "is no run id") {
				t.Errorf("stderr = %q, want the id refused", stderr)
			}
		})
	}
}
