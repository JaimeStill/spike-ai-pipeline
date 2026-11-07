package app_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/standards-lab/go-cli-sdk"
	"github.com/standards-lab/go-core/lifecycle"
	"github.com/standards-lab/go-core/process"

	"github.com/JaimeStill/spike-ai-pipeline/internal/app"
	"github.com/JaimeStill/spike-ai-pipeline/internal/apptest"
)

// shutdownTimeout is the one variable docproc reads.
var shutdownTimeout = lifecycle.NewEnv("DOCPROC").ShutdownTimeout

// hello is the committed input a dispatch runs over: a text file, which the text stage, needing
// no tool, prepares.
var hello = filepath.Join("testdata", "inputs", "hello.txt")

// newApp returns the App over fresh buffers, and the recorder of the nodes its runs build.
func newApp(t *testing.T) (a *app.App, stdout, stderr *bytes.Buffer, built *apptest.Recorder) {
	t.Helper()
	stdout, stderr = new(bytes.Buffer), new(bytes.Buffer)
	a = app.New(cli.Streams{Stdin: strings.NewReader(""), Stdout: stdout, Stderr: stderr})
	return a, stdout, stderr, apptest.Builds(a)
}

// wantLine fails the test unless out has a line matching pattern, a regexp anchored to the line.
func wantLine(t *testing.T, out, pattern string) {
	t.Helper()
	if !regexp.MustCompile(`(?m)^` + pattern + `$`).MatchString(out) {
		t.Errorf("stdout has no line matching %q:\n%s", pattern, out)
	}
}

func TestRun_RootHelp(t *testing.T) {
	for _, args := range [][]string{nil, {"-h"}, {"--help"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			// An unreadable configuration proves the run reads none.
			t.Setenv(shutdownTimeout, "not-a-duration")
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
	t.Setenv(shutdownTimeout, "not-a-duration")
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
	t.Setenv(shutdownTimeout, "not-a-duration")

	a, stdout, stderr, _ := newApp(t)

	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Errorf("New() wrote %q, %q, want nothing", stdout, stderr)
	}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Replace after New() panicked: %v; want the graph unbuilt", r)
		}
	}()
	apptest.Halt(a)
}

// Every usage error of run exits 2 before anything is built: no tool is looked up, and no
// configuration is read.
func TestRun_UsageErrors(t *testing.T) {
	out := t.TempDir()
	cpus := runtime.NumCPU()
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"no paths", []string{"run", "--out", out}, "requires at least one path"},
		{"no out", []string{"run", hello}, "--out"},
		{"empty out", []string{"run", hello, "--out", ""}, "--out names no directory"},
		{"no workers", []string{"run", hello, "--out", out, "--workers", "0"}, "--workers 0 is outside 1 to " + strconv.Itoa(cpus)},
		{"too many workers", []string{"run", hello, "--out", out, "--workers", strconv.Itoa(cpus + 1)}, "--workers " + strconv.Itoa(cpus+1) + " is outside"},
		{"unknown route", []string{"run", hello, "--out", out, "--exclude", "text", "--exclude", "pictures"}, `--exclude: unknown route "pictures"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(shutdownTimeout, "not-a-duration")
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
}

// On a host with no tool, every stage that needs one is missing, each saying which, and the
// stages that need none are wired, but the one excluded; the committed text input is prepared
// by the text stage, and the run leaves its run directory in --out, its working directory gone.
func TestRun_NoTools(t *testing.T) {
	t.Setenv(shutdownTimeout, "")
	a, stdout, stderr, built := newApp(t)
	apptest.Tools(a, apptest.NoTools)
	out := filepath.Join(t.TempDir(), "out")

	code := a.Run(context.Background(), []string{"run", hello, "--out", out, "--exclude", "vision", "--workers", "1"})

	if code != process.ExitOK {
		t.Fatalf("code = %d, want %d; stderr = %q", code, process.ExitOK, stderr)
	}
	got := stdout.String()
	wantLine(t, got, `TOOL\s+VERSION\s+MINIMUM\s+FROM`)
	wantLine(t, got, `gs\s+-\s+10\.0\s+gs: not found`)
	wantLine(t, got, `soffice\s+-\s+24\.2\s+soffice: docker not found`)
	wantLine(t, got, `ROUTE\s+STAGE\s+STATE\s+TOOLS`)
	wantLine(t, got, `document\s+-\s+missing\s+gs: not found; qpdf: not found; pdfdetach: not found`)
	wantLine(t, got, `office\s+-\s+missing\s+spike-ai-pipeline-office: not found; soffice: docker not found; magick: not found`)
	wantLine(t, got, `vision-converted\s+-\s+missing\s+magick: not found; soffice: docker not found`)
	wantLine(t, got, `vision\s+-\s+excluded\s*`)
	wantLine(t, got, `nested\s+-\s+missing\s+spike-ai-pipeline-mail: not found`)
	wantLine(t, got, `onenote\s+-\s+missing\s+spike-ai-pipeline-onenote: not found`)
	wantLine(t, got, `archive\s+-\s+missing\s+7zz: not found`)
	wantLine(t, got, `audio\s+-\s+missing\s+ffmpeg: not found; ffprobe: not found`)
	wantLine(t, got, `video\s+-\s+missing\s+ffmpeg: not found; ffprobe: not found`)
	wantLine(t, got, `text\s+text\s+wired\s+none`)
	wantLine(t, got, regexp.QuoteMeta(hello)+`  text  prepared  outputs 1  children 0  \S+`)

	if want := []string{"lifecycle config", "tool lookup"}; !slices.Equal(slices.Sorted(slices.Values(built.Log())), want) {
		t.Errorf("nodes built = %q, want %q", built.Log(), want)
	}
	entries, err := os.ReadDir(out)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if want := []string{"index", "outputs", "runs"}; !slices.Equal(names, want) {
		t.Errorf("--out holds %q, want %q", names, want)
	}
}

// A tool older than processing's minimum leaves out the stages that need it, the table saying
// so; a tool only excluded stages need is never looked up.
func TestRun_ToolTooOld(t *testing.T) {
	t.Setenv(shutdownTimeout, "")
	qpdf := filepath.Join(t.TempDir(), "qpdf")
	if err := os.WriteFile(qpdf, []byte("#!/bin/sh\necho 'qpdf version 9.1.0'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	var looked apptest.Recorder
	a, stdout, stderr, _ := newApp(t)
	apptest.Tools(a, func(program string) (string, error) {
		looked.Record(program)
		if program == "qpdf" {
			return qpdf, nil
		}
		return apptest.NoTools(program)
	})
	excluded := []string{"office", "vision-converted", "nested", "onenote", "archive", "audio", "video"}
	args := []string{"run", hello, "--out", t.TempDir()}
	for _, r := range excluded {
		args = append(args, "--exclude", r)
	}

	code := a.Run(context.Background(), args)

	if code != process.ExitOK {
		t.Fatalf("code = %d, want %d; stderr = %q", code, process.ExitOK, stderr)
	}
	got := stdout.String()
	wantLine(t, got, `qpdf\s+-\s+11\.0\s+.*9\.1\.0.*older than the minimum.*`)
	wantLine(t, got, `document\s+-\s+missing\s+gs: not found; .*older than the minimum.*; pdfdetach: not found`)
	for _, r := range excluded {
		wantLine(t, got, regexp.QuoteMeta(r)+`\s+-\s+excluded\s*`)
	}
	if want := []string{"gs", "qpdf", "pdfdetach"}; !slices.Equal(looked.Log(), want) {
		t.Errorf("tools looked up = %q, want %q", looked.Log(), want)
	}
}

// A path that doesn't exist fails the run, after the stage table, with exit 1.
func TestRun_MissingPath(t *testing.T) {
	t.Setenv(shutdownTimeout, "")
	a, _, stderr, _ := newApp(t)
	apptest.Tools(a, apptest.NoTools)

	code := a.Run(context.Background(), []string{"run", filepath.Join(t.TempDir(), "absent"), "--out", t.TempDir()})

	if code != process.ExitFailure {
		t.Errorf("code = %d, want %d; stderr = %q", code, process.ExitFailure, stderr)
	}
	if !strings.Contains(stderr.String(), "no such file or directory") {
		t.Errorf("stderr = %q, want it to name the missing path", stderr)
	}
}

// A directory is walked for its regular files, at any depth; a symbolic link in it is not
// followed, and a file given twice is one input.
func TestRun_WalksDirectories(t *testing.T) {
	t.Setenv(shutdownTimeout, "")
	dir := t.TempDir()
	a1, sub := filepath.Join(dir, "a.txt"), filepath.Join(dir, "sub", "c.txt")
	if err := os.MkdirAll(filepath.Dir(sub), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{a1, sub} {
		if err := os.WriteFile(p, []byte(p+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(a1, filepath.Join(dir, "b.txt")); err != nil {
		t.Fatal(err)
	}
	a, stdout, stderr, _ := newApp(t)
	apptest.Tools(a, apptest.NoTools)

	code := a.Run(context.Background(), []string{"run", dir, a1, "--out", t.TempDir()})

	if code != process.ExitOK {
		t.Fatalf("code = %d, want %d; stderr = %q", code, process.ExitOK, stderr)
	}
	got := stdout.String()
	for _, p := range []string{a1, sub} {
		if n := strings.Count(got, p+"  text  prepared"); n != 1 {
			t.Errorf("%s has %d lines, want 1:\n%s", p, n, got)
		}
	}
	if strings.Contains(got, "b.txt") {
		t.Errorf("the symbolic link b.txt was dispatched:\n%s", got)
	}
}
