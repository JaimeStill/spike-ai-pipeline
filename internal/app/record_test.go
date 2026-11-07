package app_test

import (
	"archive/zip"
	"bufio"
	"bytes"
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
// both in a fresh temporary directory, each file mode 0644. It returns in and out, each
// absolute.
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
// created times removed, and the inputs' directory, absolute in every "input" and "arg" and named
// by an error, written <in>, so the golden is the same on every machine.
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

// The manifest names the schema, the run, its times, its workers, its inputs as absolute paths,
// and every tool and stage, wired, missing with its reason, or excluded.
func TestRun_Manifest(t *testing.T) {
	in, out := inputs(t, "hello.txt")

	runOnce(t, apptest.NoTools, process.ExitOK, in, "--out", out, "--workers", "1", "--exclude", "vision")

	m := runs(t, out)[0].manifest
	if m.Schema != "1" || m.Workers != 1 || !slices.Equal(m.Inputs, []string{in}) {
		t.Errorf("manifest schema %q, workers %d, inputs %q; want 1, 1, [%s]", m.Schema, m.Workers, m.Inputs, in)
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

// An input's path and its argument are recorded absolute and cleaned, however they were given,
// while its path under the argument stays relative to it, and every path to a file the run
// wrote stays relative to the run directory.
func TestRun_InputPathsAbsolute(t *testing.T) {
	in, out := inputs(t, "hello.txt")
	t.Chdir(filepath.Dir(in))

	runOnce(t, apptest.NoTools, process.ExitOK, "./in/", "--out", "out", "--workers", "1")

	r := runs(t, out)[0]
	if !slices.Equal(r.manifest.Inputs, []string{in}) {
		t.Errorf("manifest inputs %q, want [%s]", r.manifest.Inputs, in)
	}
	for _, ev := range r.events(t) {
		if ev.Input != filepath.Join(in, "hello.txt") {
			t.Errorf("input %q, want %s", ev.Input, filepath.Join(in, "hello.txt"))
		}
		if ev.File != nil && (ev.File.Arg != in || ev.File.Path != "hello.txt") {
			t.Errorf("file arg %q, path %q; want %s, hello.txt", ev.File.Arg, ev.File.Path, in)
		}
		if ev.Output != nil && filepath.IsAbs(ev.Output.Ref) {
			t.Errorf("output ref %q is absolute", ev.Output.Ref)
		}
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
		if ev.Outcome != "seen" || ev.Recorded != "prepared" {
			t.Errorf("re-run line %+v is %q, recorded %q; want seen, recorded prepared", ev, ev.Outcome, ev.Recorded)
		}
		if ev.Output != nil {
			if !refs[ev.Output.Ref] {
				t.Errorf("re-run's Ref %q is none the first run stored", ev.Output.Ref)
			}
			seen++
		}
	}
	// hello.txt's text is replayed by the index, and notes.txt's, under notes.zip's child, is
	// copied from the first run's record.
	if seen != 2 {
		t.Errorf("re-run replayed %d outputs, want 2", seen)
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
	if strings.Contains(stdout, "warnings") {
		t.Errorf("re-run warns:\n%s", stdout)
	}
	matches, err := filepath.Glob(filepath.Join(out, ".work-*"))
	if err != nil || len(matches) != 0 {
		t.Errorf("working directories left: %q, %v", matches, err)
	}
}

// nested writes into dir the ZIP nest.zip, holding the committed notes.zip as its one entry, so
// notes.zip's text is two parts deep in it.
func nested(t *testing.T, dir string) {
	t.Helper()
	notes, err := os.ReadFile(filepath.Join("testdata", "inputs", "notes.zip"))
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("notes.zip")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(notes); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "nest.zip"), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

// normalized returns ev without what a re-run changes: its outcome and recorded outcome, its
// input's file times, and its input's time; and, for an input's own line, whose seen event
// processing writes, its reason.
func normalized(t *testing.T, ev schema.Event) string {
	t.Helper()
	ev.Outcome, ev.Recorded, ev.Time = "", "", nil
	if ev.File != nil {
		f := *ev.File
		f.Modified, f.Created = schema.Time{}, schema.Time{}
		ev.File = &f
	}
	if ev.Part != nil && ev.Part.Depth == 0 {
		ev.Reason = ""
	}
	b, err := json.Marshal(ev)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// recordedOf returns the outcome a seen copy of ev records: its own, or, seen, the one it
// recorded.
func recordedOf(ev schema.Event) string {
	if ev.Outcome == "seen" {
		return ev.Recorded
	}
	if ev.Event == schema.OutputEvent {
		return "prepared"
	}
	return ev.Outcome
}

// wantOrdered fails the test unless each part line of evs comes after the lines of every part
// found in it, input by input, as the record's order has it.
func wantOrdered(t *testing.T, evs []schema.Event) {
	t.Helper()
	type key struct {
		input, parent string
		depth         int
	}
	done := map[key]int{} // part lines not yet taken by their parent's
	for _, ev := range evs {
		if ev.Event != schema.PartEvent {
			continue
		}
		k := key{ev.Input, ev.Part.SHA256, ev.Part.Depth + 1}
		if n := *ev.Children; done[k] < n {
			t.Errorf("part %s at depth %d of %s has %d children, %d lines before it", ev.Part.Name, ev.Part.Depth, ev.Input, n, done[k])
		} else {
			done[k] -= n
		}
		if ev.Part.Depth > 0 {
			done[key{ev.Input, ev.Part.Parent, ev.Part.Depth}]++
		}
	}
}

// A re-run's record is whole on its own: a seen part's subtree, which processing doesn't
// dispatch again, is copied from the first run's record, so the re-run's lines are the first
// run's, every one seen, recording the outcome the first run gave it, in the record's order;
// evil.zip, quarantined, is quarantined again.
// The first run holds notes.zip twice, inside nest.zip and as an input, so its own record
// copies the second notes.zip's subtree from the first, a depth shallower.
func TestRun_ReRunRecordWhole(t *testing.T) {
	in, out := inputs(t, "hello.txt", "notes.zip", "evil.zip")
	nested(t, in)
	lookup := sevenZip(t)

	first := runOnce(t, lookup, process.ExitOK, in, "--out", out, "--workers", "1")
	runOnce(t, lookup, process.ExitOK, in, "--out", out, "--workers", "2")

	rs := runs(t, out)
	if len(rs) != 2 {
		t.Fatalf("%d runs, want 2", len(rs))
	}
	if strings.Contains(first, "warnings") {
		t.Errorf("first run warns:\n%s", first)
	}
	one, two := rs[0].events(t), rs[1].events(t)
	recorded := map[string][]string{} // a first-run line, normalized, to what a copy records
	var want []string
	for _, ev := range one {
		n := normalized(t, ev)
		want = append(want, n)
		recorded[n] = append(recorded[n], recordedOf(ev))
	}
	var got []string
	for _, ev := range two {
		n := normalized(t, ev)
		got = append(got, n)
		// A quarantined input, which the index never records, is quarantined again.
		requarantined := ev.Outcome == "quarantined" && slices.Equal(recorded[n], []string{"quarantined"})
		if !requarantined && (ev.Outcome != "seen" || !slices.Contains(recorded[n], ev.Recorded)) {
			t.Errorf("re-run line %s is %q, recorded %q; want seen, recorded one of %q", n, ev.Outcome, ev.Recorded, recorded[n])
		}
		if ev.Warning != "" {
			t.Errorf("re-run line %s warns %q", n, ev.Warning)
		}
		if ev.Output != nil {
			if _, err := os.Stat(filepath.Join(out, filepath.FromSlash(ev.Output.Ref))); err != nil {
				t.Errorf("re-run's ref: %v", err)
			}
		}
	}
	slices.Sort(want)
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("re-run's lines =\n%s\nwant the first run's\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	wantOrdered(t, one)
	wantOrdered(t, two)

	// notes.txt is two deep in nest.zip and one deep in notes.zip, in both runs.
	depths := map[int]int{}
	for _, ev := range two {
		if ev.Event == schema.PartEvent && ev.Part.Name == "notes.txt" {
			depths[ev.Part.Depth]++
		}
	}
	if depths[1] != 1 || depths[2] != 1 {
		t.Errorf("re-run's notes.txt lines by depth = %v, want one at 1 and one at 2", depths)
	}
}

// A seen part whose children no record in the run directory holds, as when the index is kept
// and its runs are not, is recorded with a warning, its children's lines missing, and the run
// goes on: its terminal line and the summary count the warning, and the report shows it.
func TestRun_ReRunNoRecordWarns(t *testing.T) {
	in, out := inputs(t, "hello.txt", "notes.zip")
	lookup := sevenZip(t)
	runOnce(t, lookup, process.ExitOK, in, "--out", out, "--workers", "1")
	if err := os.RemoveAll(filepath.Join(out, "runs")); err != nil {
		t.Fatal(err)
	}

	stdout := runOnce(t, lookup, process.ExitOK, in, "--out", out, "--workers", "1")

	r := runs(t, out)[0]
	var warned []schema.Event
	for _, ev := range r.events(t) {
		if ev.Part != nil && ev.Part.Depth > 0 {
			t.Errorf("a line for %s, which no record holds", ev.Part.Name)
		}
		if ev.Warning != "" {
			warned = append(warned, ev)
		}
	}
	if len(warned) != 1 || warned[0].Part.Name != "notes.zip" || warned[0].Outcome != "seen" ||
		!strings.Contains(warned[0].Warning, "seen with 1 children") {
		t.Fatalf("warned lines = %+v, want notes.zip's, seen with 1 children", warned)
	}
	q := regexp.QuoteMeta
	wantLine(t, stdout, q(filepath.Join(in, "notes.zip"))+`  archive  seen  outputs 0  children 1  warnings 1  \S+`)
	wantLine(t, stdout, q(filepath.Join(in, "hello.txt"))+`  text  seen  outputs 1  children 0  \S+`)
	wantLine(t, stdout, `warnings 1: seen parts whose children no record .*`)
	page, err := os.ReadFile(filepath.Join(out, "runs", r.manifest.Run, "report.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), warned[0].Warning) || !strings.Contains(string(page), "<dt>Warnings</dt>") {
		t.Error("the report doesn't show the warning")
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
