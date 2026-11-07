package app_test

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/standards-lab/go-core/process"

	"github.com/JaimeStill/spike-ai-pipeline/internal/apptest"
	"github.com/JaimeStill/spike-ai-pipeline/internal/schema"
)

// update rewrites the record's golden file from the run instead of comparing against it.
var update = flag.Bool("update", false, "rewrite testdata/record.golden.jsonl")

// sevenZip writes a stand-in for 7-Zip's 7zz, which prints the version line 7zz prints with no
// arguments and nothing else, and returns a lookup that finds it and no other tool. The archive
// stage it wires reads a ZIP through archive/zip, so the stand-in is never asked to extract.
func sevenZip(t *testing.T) func(string) (string, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "7zz")
	script := "#!/bin/sh\necho '7-Zip (z) 26.06 (x64) : Copyright (c) 1999-2026 Igor Pavlov : 2026-06-01'\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return func(program string) (string, error) {
		if program == "7zz" {
			return path, nil
		}
		return apptest.NoTools(program)
	}
}

// inputs copies the committed inputs named into a directory "in" beside a run directory "out",
// both in a fresh temporary directory, each file mode 0644, so every path the record writes,
// relative to "out", is the same on every run. It returns in and out.
func inputs(t *testing.T, names ...string) (in, out string) {
	t.Helper()
	root := t.TempDir()
	in, out = filepath.Join(root, "in"), filepath.Join(root, "out")
	if err := os.Mkdir(in, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		b, err := os.ReadFile(filepath.Join("testdata", "inputs", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(in, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(filepath.Join(in, name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return in, out
}

// runOnce runs docproc run over args, finding tools through lookup, and fails the test unless it
// exits with want. It returns the terminal's output.
func runOnce(t *testing.T, lookup func(string) (string, error), want int, args ...string) string {
	t.Helper()
	t.Setenv(shutdownTimeout, "")
	a, stdout, stderr, _ := newApp(t)
	apptest.Tools(a, lookup)
	if code := a.Run(context.Background(), append([]string{"run"}, args...)); code != want {
		t.Fatalf("code = %d, want %d; stderr = %q", code, want, stderr)
	}
	return stdout.String()
}

// run is one run's manifest and the raw lines of its record.
type run struct {
	manifest schema.Manifest
	lines    []string
}

// runs returns every run under the run directory out, in the order they began, by their version
// 7 ids.
func runs(t *testing.T, out string) []run {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(out, "runs"))
	if err != nil {
		t.Fatal(err)
	}
	var rs []run
	for _, e := range entries {
		var r run
		b, err := os.ReadFile(filepath.Join(out, "runs", e.Name(), "manifest.json"))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(b, &r.manifest); err != nil {
			t.Fatal(err)
		}
		if r.manifest.Run != e.Name() || r.manifest.Record != "runs/"+e.Name()+"/record.jsonl" {
			t.Errorf("manifest run %q, record %q, under runs/%s", r.manifest.Run, r.manifest.Record, e.Name())
		}
		f, err := os.Open(filepath.Join(out, filepath.FromSlash(r.manifest.Record)))
		if err != nil {
			t.Fatal(err)
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			r.lines = append(r.lines, sc.Text())
		}
		_ = f.Close()
		if err := sc.Err(); err != nil {
			t.Fatal(err)
		}
		rs = append(rs, r)
	}
	return rs
}

// events decodes r's record lines.
func (r run) events(t *testing.T) []schema.Event {
	t.Helper()
	evs := make([]schema.Event, len(r.lines))
	for i, line := range r.lines {
		if err := json.Unmarshal([]byte(line), &evs[i]); err != nil {
			t.Fatalf("record line %d: %v: %s", i, err, line)
		}
	}
	return evs
}

// runSpecific matches the record's fields that differ from run to run: its times. A created
// time, present only where the filesystem records one, is matched with its comma, and removed.
var (
	runSpecific = regexp.MustCompile(`"(modified|start|end)":"[^"]*"`)
	created     = regexp.MustCompile(`,"created":"[^"]*"`)
)

// The record's shape, pinned: one run over a text input, an archive holding a text file, an
// archive whose entry climbs out of it, and an input that can't be read, which make one line of
// every kind: an output prepared, a nested part's own event, an input's own event with its file,
// a quarantined input with its kept file, and an input's error. Its times are normalized, its
// created times removed, and the inputs' directory, which an error names, written <in>.
func TestRun_RecordGolden(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads a file of mode 0000, so no input fails")
	}
	in, out := inputs(t, "evil.zip", "hello.txt", "notes.zip")
	unreadable := filepath.Join(in, "zz-unreadable.txt")
	if err := os.WriteFile(unreadable, []byte("unread\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(unreadable, 0); err != nil {
		t.Fatal(err)
	}

	runOnce(t, sevenZip(t), process.ExitOK, in, "--out", out, "--workers", "1")

	rs := runs(t, out)
	if len(rs) != 1 {
		t.Fatalf("%d runs, want 1", len(rs))
	}
	got := strings.Join(rs[0].lines, "\n") + "\n"
	got = runSpecific.ReplaceAllString(created.ReplaceAllString(got, ""), `"$1":"<time>"`)
	got = strings.ReplaceAll(got, in, "<in>")
	golden := filepath.Join("testdata", "record.golden.jsonl")
	if *update {
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("record =\n%s\nwant\n%s", got, want)
	}
}

// The manifest names the schema, the run, its times, its workers, its inputs relative to the run
// directory, and every tool and stage, wired, missing with its reason, or excluded.
func TestRun_Manifest(t *testing.T) {
	in, out := inputs(t, "hello.txt")

	runOnce(t, apptest.NoTools, process.ExitOK, in, "--out", out, "--workers", "1", "--exclude", "vision")

	m := runs(t, out)[0].manifest
	if m.Schema != "1" || m.Workers != 1 || !slices.Equal(m.Inputs, []string{"../in"}) {
		t.Errorf("manifest schema %q, workers %d, inputs %q; want 1, 1, [../in]", m.Schema, m.Workers, m.Inputs)
	}
	if m.Start.IsZero() || m.End.IsZero() || m.End.String() < m.Start.String() || m.Error != "" {
		t.Errorf("manifest start %v, end %v, error %q; want an end after the start, no error", m.Start, m.End, m.Error)
	}
	if len(m.Tools) == 0 || m.Tools[0].Program != "gs" || m.Tools[0].Error != "gs: not found" || m.Tools[0].Version != "" {
		t.Errorf("manifest tools = %+v, want gs first, not found", m.Tools)
	}
	byRoute := map[string]schema.StageState{}
	for _, s := range m.Stages {
		byRoute[s.Route] = s
	}
	if len(m.Stages) != 10 {
		t.Errorf("manifest has %d stages, want 10", len(m.Stages))
	}
	if s := byRoute["vision"]; s.State != "excluded" || s.Stage != nil {
		t.Errorf("vision = %+v, want excluded", s)
	}
	if s := byRoute["archive"]; s.State != "missing" || s.Reason != "7zz: not found" {
		t.Errorf("archive = %+v, want missing, 7zz: not found", s)
	}
	if s := byRoute["text"]; s.State != "wired" || s.Stage == nil || s.Stage.Name != "text" {
		t.Errorf("text = %+v, want wired, its stage named", s)
	}
}

// The terminal prints one line per input as its last event arrives, then the summary counting
// the inputs by route and outcome, and where the run's record is.
func TestRun_TerminalSummary(t *testing.T) {
	in, out := inputs(t, "evil.zip", "hello.txt", "notes.zip")

	got := runOnce(t, sevenZip(t), process.ExitOK, in, "--out", out, "--workers", "2")

	q := regexp.QuoteMeta
	wantLine(t, got, q(filepath.Join(in, "hello.txt"))+`  text  prepared  outputs 1  children 0  \S+`)
	wantLine(t, got, q(filepath.Join(in, "notes.zip"))+`  archive  prepared  outputs 0  children 1  \S+`)
	wantLine(t, got, q(filepath.Join(in, "evil.zip"))+`  archive  quarantined  outputs 0  children 0  \S+`)
	wantLine(t, got, `ROUTE\s+OUTCOME\s+INPUTS`)
	wantLine(t, got, `archive\s+prepared\s+1`)
	wantLine(t, got, `archive\s+quarantined\s+1`)
	wantLine(t, got, `text\s+prepared\s+1`)
	wantLine(t, got, `total\s+3`)
	id := runs(t, out)[0].manifest.Run
	wantLine(t, got, `run `+q(id)+`  runs/`+q(id)+`/record\.jsonl`)
	if i, j := strings.Index(got, "text  prepared  outputs"), strings.Index(got, "OUTCOME"); i > j {
		t.Errorf("the summary comes before an input's line:\n%s", got)
	}
}

// A quarantined part's file is kept in the run directory beside its flags, the record naming
// both relative to it.
func TestRun_QuarantineKept(t *testing.T) {
	in, out := inputs(t, "evil.zip")

	runOnce(t, apptest.NoTools, process.ExitOK, in, "--out", out)

	evs := runs(t, out)[0].events(t)
	if len(evs) != 1 || evs[0].Quarantine == nil {
		t.Fatalf("record = %+v, want one quarantined part", evs)
	}
	q := evs[0].Quarantine
	kept, err := os.ReadFile(filepath.Join(out, q.File))
	if err != nil {
		t.Fatal(err)
	}
	orig, err := os.ReadFile(filepath.Join(in, "evil.zip"))
	if err != nil {
		t.Fatal(err)
	}
	if string(kept) != string(orig) {
		t.Error("the kept file isn't the quarantined part's content")
	}
	var flags []schema.Flag
	b, err := os.ReadFile(filepath.Join(out, q.Flags))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &flags); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(flags, evs[0].Part.Flags) || len(flags) == 0 {
		t.Errorf("kept flags = %+v, want the record's %+v", flags, evs[0].Part.Flags)
	}
	if filepath.Dir(q.File) != filepath.Dir(q.Flags) {
		t.Errorf("file %s and flags %s aren't side by side", q.File, q.Flags)
	}
}

// A re-run over an existing run directory, its inputs unchanged, prepares nothing: it reuses the
// index and the store under a new run id with its own manifest and record, every input is seen,
// and every output's Ref names the copy the first run stored.
func TestRun_ReRunSeesEverything(t *testing.T) {
	in, out := inputs(t, "hello.txt", "notes.zip")
	lookup := sevenZip(t)

	runOnce(t, lookup, process.ExitOK, in, "--out", out, "--workers", "1")
	stdout := runOnce(t, lookup, process.ExitOK, in, "--out", out, "--workers", "1")

	rs := runs(t, out)
	if len(rs) != 2 || rs[0].manifest.Run == rs[1].manifest.Run {
		t.Fatalf("%d runs, want 2, each its own id", len(rs))
	}
	refs := map[string]bool{}
	for _, ev := range rs[0].events(t) {
		if ev.Output != nil {
			if ev.Outcome != "" {
				t.Errorf("first run's output %+v is %q, want prepared", ev.Output, ev.Outcome)
			}
			refs[ev.Output.Ref] = true
		}
	}
	if len(refs) != 2 {
		t.Fatalf("first run stored %d outputs, want 2: hello.txt's text and notes.txt's", len(refs))
	}
	seen := 0
	for _, ev := range rs[1].events(t) {
		if ev.Outcome != "seen" {
			t.Errorf("re-run line %+v is %q, want seen", ev, ev.Outcome)
		}
		if ev.Output != nil {
			if !refs[ev.Output.Ref] {
				t.Errorf("re-run's Ref %q is none the first run stored", ev.Output.Ref)
			}
			seen++
		}
		if ev.Part != nil && ev.Part.Depth > 0 {
			t.Errorf("re-run dispatched the nested part %+v again", ev.Part)
		}
	}
	// The input's own outputs are replayed; notes.zip's text, its child's, is recorded under
	// its child, which a seen part references and doesn't dispatch again.
	if seen != 1 {
		t.Errorf("re-run replayed %d outputs, want 1", seen)
	}
	for ref := range refs {
		b, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(ref)))
		if err != nil {
			t.Error(err)
		} else if !strings.Contains(string(b), "Hello from docproc.") && !strings.Contains(string(b), "Notes kept") {
			t.Errorf("stored %s = %q, want one of the texts", ref, b)
		}
	}
	wantLine(t, stdout, `archive\s+seen\s+1`)
	wantLine(t, stdout, `text\s+seen\s+1`)
	matches, err := filepath.Glob(filepath.Join(out, ".work-*"))
	if err != nil || len(matches) != 0 {
		t.Errorf("working directories left: %q, %v", matches, err)
	}
}

// An error that stops the whole dispatch, here its context ending once the stages are built,
// exits 1, the manifest naming it and its end.
func TestRun_StoppedDispatchExits1(t *testing.T) {
	t.Setenv(shutdownTimeout, "")
	in, out := inputs(t, "hello.txt")
	a, _, stderr, _ := newApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// The lookup runs as the stages are built, before the dispatch starts.
	apptest.Tools(a, func(program string) (string, error) {
		cancel()
		return apptest.NoTools(program)
	})

	code := a.Run(ctx, []string{"run", in, "--out", out})

	if code != process.ExitFailure {
		t.Fatalf("code = %d, want %d; stderr = %q", code, process.ExitFailure, stderr)
	}
	m := runs(t, out)[0].manifest
	if m.Error != context.Canceled.Error() || m.End.IsZero() {
		t.Errorf("manifest error %q, end %v; want %q and an end", m.Error, m.End, context.Canceled)
	}
}
