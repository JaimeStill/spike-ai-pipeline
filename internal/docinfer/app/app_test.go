package app_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/standards-lab/go-cli-sdk"
	"github.com/standards-lab/go-core/process"

	"github.com/JaimeStill/spike-ai-pipeline/internal/apptest"
	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/app"
	docinfertest "github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/apptest"
)

// docproc is the committed docproc run directory docinfer reads: testdata/README.md says what
// its two runs hold.
var docproc = filepath.Join("testdata", "docproc")

// The committed runs' ids, the first and the latest.
const (
	firstRun  = "019a5e2a-1b3c-7d4e-8f00-000000000001"
	latestRun = "019a5e2a-1b3c-7d4e-8f00-000000000002"
)

// Set A's models, as the example configuration names them.
const (
	embeddingModel = "mrutkows/granite-embedding-english-r2-GGUF:BF16"
	visionModel    = "google/gemma-4-26B-A4B-it-qat-q4_0-gguf:Q4_0"
	audioModel     = "ggml-org/gemma-4-E4B-it-GGUF:Q8_0"
)

// setA is a listing of set A's models, each loaded.
var setA = []docinfertest.Model{
	{ID: embeddingModel, Status: "loaded"},
	{ID: visionModel, Status: "loaded"},
	{ID: audioModel, Status: "loaded"},
}

// overrides are the variables docinfer reads that a test's environment could otherwise leak in;
// clean empties each.
var overrides = []string{
	"DOCINFER_ENV", "DOCINFER_SHUTDOWN_TIMEOUT",
	"DOCINFER_EMBEDDINGS_ENDPOINT", "DOCINFER_EMBEDDINGS_MODEL", "DOCINFER_EMBEDDINGS_CONCURRENCY",
	"DOCINFER_EMBEDDINGS_BATCH", "DOCINFER_EMBEDDINGS_CHUNK_SIZE",
	"DOCINFER_VISION_ENDPOINT", "DOCINFER_VISION_MODEL", "DOCINFER_VISION_CONCURRENCY",
	"DOCINFER_VISION_PROMPT", "DOCINFER_VISION_MAX_TOKENS", "DOCINFER_VISION_REASONING",
	"DOCINFER_TRANSCRIPTION_ENDPOINT", "DOCINFER_TRANSCRIPTION_MODEL", "DOCINFER_TRANSCRIPTION_CONCURRENCY",
	"DOCINFER_TRANSCRIPTION_PROMPT", "DOCINFER_TRANSCRIPTION_LANGUAGE",
	"DOCINFER_ENDPOINTS_FAKE_BASE_URL", "DOCINFER_ENDPOINTS_FAKE_FORMAT", "DOCINFER_ENDPOINTS_FAKE_TOKEN",
}

// clean empties every variable docinfer reads, so the test sees only what it sets.
func clean(t *testing.T) {
	t.Helper()
	for _, name := range overrides {
		t.Setenv(name, "")
	}
}

// newApp returns the App over fresh buffers, and the recorder of the nodes its runs build.
func newApp(t *testing.T) (a *app.App, stdout, stderr *bytes.Buffer, built *apptest.Recorder) {
	t.Helper()
	stdout, stderr = new(bytes.Buffer), new(bytes.Buffer)
	a = app.New(cli.Streams{Stdin: strings.NewReader(""), Stdout: stdout, Stderr: stderr})
	return a, stdout, stderr, docinfertest.Builds(a)
}

// run runs docinfer with args, and fails the test unless it exits with want. It returns the
// terminal's output.
func run(t *testing.T, want int, args ...string) (stdout, stderr string) {
	t.Helper()
	a, out, errs, _ := newApp(t)
	if code := a.Run(context.Background(), args); code != want {
		t.Fatalf("code = %d, want %d; stdout = %q; stderr = %q", code, want, out, errs)
	}
	return out.String(), errs.String()
}

// configDir writes files, each a name and its content, into a new directory, and returns it.
func configDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// fullConfig is a config.json of set A's three capabilities on one endpoint, "fake", at url,
// with the router's query.
func fullConfig(url string) string {
	return fmt.Sprintf(`{
  "endpoints": {"fake": {"base_url": %q, "format": "completions", "query": {"autoload": "false"}}},
  "embeddings": {"endpoint": "fake", "model": %q},
  "vision": {"endpoint": "fake", "model": %q},
  "transcription": {"endpoint": "fake", "model": %q}
}`, url, embeddingModel, visionModel, audioModel)
}

// wantLine fails the test unless out has a line matching pattern, a regexp anchored to the line.
func wantLine(t *testing.T, out, pattern string) {
	t.Helper()
	if !regexp.MustCompile(`(?m)^` + pattern + `$`).MatchString(out) {
		t.Errorf("output has no line matching %q:\n%s", pattern, out)
	}
}

// row is the pattern of one capability's table row.
func row(capability, endpoint, model, state string) string {
	return regexp.QuoteMeta(capability) + `\s+` + regexp.QuoteMeta(endpoint) + `\s+` +
		regexp.QuoteMeta(model) + `\s+` + regexp.QuoteMeta(state)
}

func TestRun_RootHelp(t *testing.T) {
	for _, args := range [][]string{nil, {"-h"}, {"--help"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			// An unreadable configuration proves the run reads none.
			clean(t)
			t.Setenv("DOCINFER_SHUTDOWN_TIMEOUT", "not-a-duration")
			a, stdout, stderr, built := newApp(t)

			code := a.Run(context.Background(), args)

			if code != process.ExitUsage {
				t.Errorf("code = %d, want %d; stderr = %q", code, process.ExitUsage, stderr)
			}
			for _, cmd := range []string{"report", "run", "version"} {
				if !strings.Contains(stdout.String(), cmd) {
					t.Errorf("stdout = %q, want the root help listing %s", stdout, cmd)
				}
			}
			if log := built.Log(); len(log) != 0 {
				t.Errorf("nodes built = %q, want none", log)
			}
		})
	}
}

func TestRun_Version(t *testing.T) {
	clean(t)
	t.Setenv("DOCINFER_SHUTDOWN_TIMEOUT", "not-a-duration")
	a, stdout, stderr, built := newApp(t)

	code := a.Run(context.Background(), []string{"version"})

	if code != process.ExitOK {
		t.Errorf("code = %d, want %d; stderr = %q", code, process.ExitOK, stderr)
	}
	// A test binary carries no main module version.
	if stdout.String() != "(devel)\n" {
		t.Errorf("stdout = %q, want %q", stdout, "(devel)\n")
	}
	if log := built.Log(); len(log) != 0 {
		t.Errorf("nodes built = %q, want none", log)
	}
}

// New builds nothing: the graph freezes on its first Build and panics on a Replace after it, so
// Halt's Replace succeeding proves New never built.
func TestNew_IsCold(t *testing.T) {
	a, stdout, stderr, _ := newApp(t)

	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Errorf("New() wrote %q, %q, want nothing", stdout, stderr)
	}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Replace after New() panicked: %v; want the graph unbuilt", r)
		}
	}()
	docinfertest.Halt(a)
}

// Every usage error of run exits 2 before anything is built or read: no request is sent, and a
// --run that is a path rather than one run's id is refused even when it resolves to a real run,
// as ../runs/<id> does.
func TestRun_UsageErrors(t *testing.T) {
	clean(t)
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA})
	cfg := configDir(t, map[string]string{"config.json": fullConfig(e.URL())})
	out := filepath.Join(t.TempDir(), "out")
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"no dir", []string{"run", "--config", cfg, "--out", out}, "accepts 1 argument"},
		{"two dirs", []string{"run", docproc, docproc, "--config", cfg, "--out", out}, "accepts 1 argument"},
		{"no out", []string{"run", docproc, "--config", cfg}, "--out"},
		{"no config", []string{"run", docproc, "--out", out}, "--config"},
		{"empty out", []string{"run", docproc, "--config", cfg, "--out", ""}, "--out names no directory"},
		{"empty config", []string{"run", docproc, "--config", "", "--out", out}, "--config names no directory"},
		{"run is a path", []string{"run", docproc, "--config", cfg, "--out", out, "--run", "../runs/" + latestRun}, "is no run id"},
		{"run is dot", []string{"run", docproc, "--config", cfg, "--out", out, "--run", "."}, "is no run id"},
		{"run is dot-dot", []string{"run", docproc, "--config", cfg, "--out", out, "--run", ".."}, "is no run id"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, stdout, stderr, built := newApp(t)

			code := a.Run(context.Background(), tc.args)

			if code != process.ExitUsage {
				t.Errorf("code = %d, want %d; stderr = %q", code, process.ExitUsage, stderr)
			}
			if !strings.Contains(stderr.String(), tc.want) {
				t.Errorf("stderr = %q, want it to contain %q", stderr, tc.want)
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if log := built.Log(); len(log) != 0 {
				t.Errorf("nodes built = %q, want none", log)
			}
		})
	}
	if reqs := e.Requests(); len(reqs) != 0 {
		t.Errorf("requests = %v, want none", reqs)
	}
}

// A run whose Build fails exits 1 with the failure labelled with its node, before any request.
func TestRun_BuildFailureStopsRun(t *testing.T) {
	for _, tc := range []struct {
		name string
		halt bool
		want string
	}{
		{"invalid shutdown timeout", false, "docinfer run: lifecycle config: DOCINFER_SHUTDOWN_TIMEOUT"},
		{"halted", true, "docinfer run: lifecycle config: " + docinfertest.ErrHalted.Error()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clean(t)
			t.Setenv("DOCINFER_SHUTDOWN_TIMEOUT", "not-a-duration")
			e := docinfertest.Serve(t, docinfertest.Fake{Models: setA})
			cfg := configDir(t, map[string]string{"config.json": fullConfig(e.URL())})
			a, stdout, stderr, _ := newApp(t)
			if tc.halt {
				docinfertest.Halt(a)
			}

			code := a.Run(context.Background(), []string{"run", docproc, "--config", cfg, "--out", t.TempDir()})

			if code != process.ExitFailure {
				t.Errorf("code = %d, want %d; stderr = %q", code, process.ExitFailure, stderr)
			}
			if !strings.HasPrefix(stderr.String(), tc.want) {
				t.Errorf("stderr = %q, want it to start with %q", stderr, tc.want)
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if reqs := e.Requests(); len(reqs) != 0 {
				t.Errorf("requests = %v, want none", reqs)
			}
		})
	}
}

// With every model loaded, run lists the endpoint once, with its query and the secrets file's
// token, shows each capability loaded, and reads the latest run's every output, the failed
// part's and the seen part's among them; then it embeds, each request with the query and token
// too, and so are the vision and transcription requests.
func TestRun_ReadsLatestRun(t *testing.T) {
	clean(t)
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA})
	cfg := configDir(t, map[string]string{
		"config.json":  fullConfig(e.URL()),
		"secrets.json": `{"endpoints": {"fake": {"token": "secret-token"}}}`,
	})
	out := filepath.Join(t.TempDir(), "out")
	a, stdout, stderr, built := newApp(t)

	code := a.Run(context.Background(), []string{"run", docproc, "--config", cfg, "--out", out})

	if code != process.ExitOK {
		t.Fatalf("code = %d, want %d; stderr = %q", code, process.ExitOK, stderr)
	}
	got := stdout.String()
	wantLine(t, got, `CAPABILITY\s+ENDPOINT\s+MODEL\s+STATE`)
	wantLine(t, got, row("embeddings", "fake", embeddingModel, "loaded"))
	wantLine(t, got, row("vision", "fake", visionModel, "loaded"))
	wantLine(t, got, row("transcription", "fake", audioModel, "loaded"))
	wantLine(t, got, regexp.QuoteMeta("docproc run "+latestRun+": outputs 12: text 5, markdown 1, image 1, record 2, audio 3"))
	wantLine(t, got, regexp.QuoteMeta("vision: images 1, distinct 1: held 0, described 1, incomplete 0 (empty 0), refused 0; requests 1"))
	wantLine(t, got, regexp.QuoteMeta("transcription: windows 3, distinct 3: held 0, transcribed 3, failed 0; requests 3; transcripts 1: seams matched 0, unmatched 2"))
	wantLine(t, got, `embeddings: texts 6, metadata 8, chunks \d+: held 0, embedded \d+, rejected 0; requests 1`)
	if strings.Contains(got, "warning") {
		t.Errorf("stdout warns, want no warning:\n%s", got)
	}
	reqs := e.Requests()
	want := docinfertest.Request{Method: "GET", Path: "/v1/models", Query: "autoload=false", Authorization: "Bearer secret-token"}
	if len(reqs) == 0 || reqs[0] != want {
		t.Fatalf("requests = %v, want the listing first, %v", reqs, want)
	}
	for _, r := range reqs[1:] {
		want := docinfertest.Request{Method: "POST", Path: "/v1/embeddings", Query: "autoload=false", Authorization: "Bearer secret-token"}
		if r.Path == "/v1/chat/completions" || r.Path == "/v1/audio/transcriptions" {
			want.Path = r.Path
		}
		if r != want {
			t.Errorf("request %v, want %v or the vision or transcription request's", r, want)
		}
	}
	if want := []string{"http client", "lifecycle config"}; !slices.Equal(slices.Sorted(slices.Values(built.Log())), want) {
		t.Errorf("nodes built = %q, want %q", built.Log(), want)
	}
}

// --run reads the run it names rather than the latest.
func TestRun_RunFlagNamesRun(t *testing.T) {
	clean(t)
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA})
	cfg := configDir(t, map[string]string{"config.json": fullConfig(e.URL())})

	stdout, _ := run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", t.TempDir(), "--run", firstRun)

	wantLine(t, stdout, regexp.QuoteMeta("docproc run "+firstRun+": outputs 1: text 1"))
}

// copyDocproc returns a copy of the committed docproc run directory, for a test that changes it.
func copyDocproc(t *testing.T) string {
	t.Helper()
	dst := filepath.Join(t.TempDir(), "docproc")
	if err := os.CopyFS(dst, os.DirFS(docproc)); err != nil {
		t.Fatal(err)
	}
	return dst
}

// A manifest whose schema isn't "1" is refused before its record is read, here a record that
// is gone, and before any request; so is a run the directory doesn't hold, and a directory with
// no run.
func TestRun_DocprocRunRefused(t *testing.T) {
	clean(t)
	changed := copyDocproc(t)
	manifest := filepath.Join(changed, "runs", latestRun, "manifest.json")
	b, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	b = bytes.Replace(b, []byte(`"schema": "1"`), []byte(`"schema": "2"`), 1)
	if err := os.WriteFile(manifest, b, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(changed, "runs", latestRun, "record.jsonl")); err != nil {
		t.Fatal(err)
	}
	empty := t.TempDir()
	if err := os.Mkdir(filepath.Join(empty, "runs"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"wrong schema", []string{changed}, `run ` + latestRun + ` has schema "2"; docinfer reads schema "1"`},
		{"no such run", []string{docproc, "--run", "nope"}, "no such file or directory"},
		{"no run", []string{empty}, "holds no run"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := docinfertest.Serve(t, docinfertest.Fake{Models: setA})
			cfg := configDir(t, map[string]string{"config.json": fullConfig(e.URL())})
			args := append([]string{"run", "--config", cfg, "--out", t.TempDir()}, tc.args...)

			stdout, stderr := run(t, process.ExitFailure, args...)

			if !strings.Contains(stderr, tc.want) {
				t.Errorf("stderr = %q, want it to contain %q", stderr, tc.want)
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
