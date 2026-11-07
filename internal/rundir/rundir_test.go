package rundir_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/JaimeStill/spike-ai-pipeline-processing/dispatch"

	"github.com/JaimeStill/spike-ai-pipeline/internal/rundir"
	"github.com/JaimeStill/spike-ai-pipeline/internal/schema"
)

var (
	part  = dispatch.Part{SHA256: "abc123", Name: "a.pdf"}
	stage = dispatch.Identity{Name: "document", Version: "gs 10.5", Config: "ff"}
)

// open opens a run directory in a fresh temporary directory.
func open(t *testing.T) *rundir.Dir {
	t.Helper()
	d, err := rundir.Open(filepath.Join(t.TempDir(), "out"))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// readRef returns the stored content ref names in d.
func readRef(t *testing.T, d *rundir.Dir, ref string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(d.Root(), filepath.FromSlash(ref)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// refPattern is a Ref's shape: the part's SHA-256, the stage's name and identity hash, the
// output's number, and its media type's extension.
var refPattern = regexp.MustCompile(`^outputs/abc123/document-[0-9a-f]{12}/(\d+)(\.\w+)?$`)

func TestStore_InlineAndFile(t *testing.T) {
	d := open(t)
	src := filepath.Join(t.TempDir(), "page.png")
	if err := os.WriteFile(src, []byte("png bytes"), 0o600); err != nil {
		t.Fatal(err)
	}

	inline, err := d.Store(part, stage, 1, dispatch.Output{Kind: dispatch.RecordOutput, MediaType: "application/json", Value: []byte(`{"a":1}`)})
	if err != nil {
		t.Fatal(err)
	}
	file, err := d.Store(part, stage, 2, dispatch.Output{Kind: dispatch.ImageOutput, MediaType: "image/png", Path: src})
	if err != nil {
		t.Fatal(err)
	}
	other, err := d.Store(part, stage, 3, dispatch.Output{Kind: dispatch.TextOutput, MediaType: "text/x-unknown", Path: "/nowhere/notes.LOG"})

	for ref, want := range map[string]string{inline: "1.json", file: "2.png"} {
		if !refPattern.MatchString(ref) || filepath.Base(ref) != want {
			t.Errorf("Ref %q, want outputs/abc123/document-<hash>/%s", ref, want)
		}
	}
	if got := readRef(t, d, inline); got != `{"a":1}` {
		t.Errorf("inline stored %q", got)
	}
	if got := readRef(t, d, file); got != "png bytes" {
		t.Errorf("file stored %q", got)
	}
	// An unknown media type takes its file's extension; a file that can't be read is an error.
	if err == nil || filepath.Base(other) != "3.log" {
		t.Errorf("Store of a missing file = %q, %v; want 3.log and an error", other, err)
	}
	info, statErr := os.Stat(filepath.Join(d.Root(), filepath.FromSlash(file)))
	if statErr != nil || info.Mode().Perm() != 0o644 {
		t.Errorf("stored file mode = %v, %v; want 0644", info, statErr)
	}
}

// An output stored once is kept: storing the same part's output under the same stage again
// writes nothing, and another stage identity stores apart.
func TestStore_Once(t *testing.T) {
	d := open(t)
	first, err := d.Store(part, stage, 1, dispatch.Output{MediaType: "text/plain", Value: []byte("first")})
	if err != nil {
		t.Fatal(err)
	}

	again, err := d.Store(part, stage, 1, dispatch.Output{MediaType: "text/plain", Value: []byte("second")})
	if err != nil {
		t.Fatal(err)
	}
	newer := stage
	newer.Version = "gs 10.6"
	apart, err := d.Store(part, newer, 1, dispatch.Output{MediaType: "text/plain", Value: []byte("newer")})
	if err != nil {
		t.Fatal(err)
	}

	if again != first || readRef(t, d, first) != "first" {
		t.Errorf("second Store = %q holding %q, want %q kept as first stored", again, readRef(t, d, again), first)
	}
	if apart == first || readRef(t, d, apart) != "newer" {
		t.Errorf("a new stage version stored at %q, want apart from %q", apart, first)
	}
}

func TestQuarantine_BesideFlags(t *testing.T) {
	d := open(t)
	src := filepath.Join(t.TempDir(), "evil.zip")
	if err := os.WriteFile(src, []byte("zip bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := dispatch.Part{Path: src, SHA256: "def456", Flags: nil}

	q, err := d.Quarantine(p)

	if err != nil {
		t.Fatal(err)
	}
	if q.File != "quarantine/def456/content" || q.Flags != "quarantine/def456/flags.json" {
		t.Errorf("Quarantine = %+v", q)
	}
	if got := readRef(t, d, q.File); got != "zip bytes" {
		t.Errorf("kept %q", got)
	}
	if got := readRef(t, d, q.Flags); got != "[]\n" {
		t.Errorf("flags %q, want an empty array", got)
	}
}

func TestRel(t *testing.T) {
	d := open(t)
	for in, want := range map[string]string{
		filepath.Join(d.Root(), "runs", "x"):             "runs/x",
		filepath.Join(filepath.Dir(d.Root()), "in", "a"): "../in/a",
		d.Root(): ".",
	} {
		if got, err := d.Rel(in); err != nil || got != want {
			t.Errorf("Rel(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
}

// A run writes its manifest as it begins, without an end, and again as it ends; its record
// holds a line per Write.
func TestRun_ManifestAndRecord(t *testing.T) {
	d := open(t)
	start := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	r, err := d.Begin(schema.Manifest{Workers: 3, Inputs: []string{"../in"}}, start)
	if err != nil {
		t.Fatal(err)
	}
	manifest := func() schema.Manifest {
		var m schema.Manifest
		b, err := os.ReadFile(filepath.Join(d.Root(), "runs", r.ID(), "manifest.json"))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatal(err)
		}
		return m
	}
	if m := manifest(); m.Schema != "1" || m.Run != r.ID() || !m.End.IsZero() || m.Record != "runs/"+r.ID()+"/record.jsonl" {
		t.Errorf("begun manifest = %+v", m)
	}

	if err := r.Write(schema.Event{Event: schema.ErrorEvent, Input: "../in/a", Error: "<boom>"}); err != nil {
		t.Fatal(err)
	}
	if err := r.End(start.Add(time.Second), nil); err != nil {
		t.Fatal(err)
	}

	if m := manifest(); time.Time(m.End) != start.Add(time.Second) || m.Workers != 3 || m.Error != "" {
		t.Errorf("ended manifest = %+v", m)
	}
	if got := readRef(t, d, r.Record()); got != `{"event":"error","input":"../in/a","error":"<boom>"}`+"\n" {
		t.Errorf("record = %q", got)
	}
	r2, err := d.Begin(schema.Manifest{}, start)
	if err != nil {
		t.Fatal(err)
	}
	if r2.ID() <= r.ID() {
		t.Errorf("second run id %s doesn't sort after %s", r2.ID(), r.ID())
	}
	if err := r2.End(start, nil); err != nil {
		t.Fatal(err)
	}
}
