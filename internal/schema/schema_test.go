package schema_test

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/JaimeStill/spike-ai-pipeline-processing/dispatch"
	"github.com/JaimeStill/spike-ai-pipeline-processing/inspect"
	"github.com/JaimeStill/spike-ai-pipeline-processing/route"

	"github.com/JaimeStill/spike-ai-pipeline/internal/schema"
)

// marshal returns v as JSON, failing the test on an error.
func marshal(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestTime_RFC3339UTCNanoseconds(t *testing.T) {
	east := time.FixedZone("east", 5*3600)
	for _, tc := range []struct {
		name string
		in   time.Time
		want string
	}{
		{"whole second", time.Date(2026, 11, 7, 15, 4, 5, 0, time.UTC), `"2026-11-07T15:04:05.000000000Z"`},
		{"nanoseconds", time.Date(2026, 11, 7, 15, 4, 5, 120, time.UTC), `"2026-11-07T15:04:05.000000120Z"`},
		{"other zone", time.Date(2026, 11, 7, 20, 4, 5, 7, east), `"2026-11-07T15:04:05.000000007Z"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := marshal(t, schema.Time(tc.in))
			if got != tc.want {
				t.Errorf("Marshal = %s, want %s", got, tc.want)
			}
			var back schema.Time
			if err := json.Unmarshal([]byte(got), &back); err != nil {
				t.Fatal(err)
			}
			if !time.Time(back).Equal(tc.in) {
				t.Errorf("Unmarshal = %v, want %v", time.Time(back), tc.in)
			}
		})
	}
}

func TestTime_UnmarshalRefusesNonString(t *testing.T) {
	var tm schema.Time
	if err := json.Unmarshal([]byte(`12`), &tm); err == nil {
		t.Error("Unmarshal(12) = nil, want an error")
	}
	if err := json.Unmarshal([]byte(`"yesterday"`), &tm); err == nil {
		t.Error(`Unmarshal("yesterday") = nil, want an error`)
	}
}

// stater returns a Stater that reports st for every path.
func stater(st schema.Stat) schema.Stater {
	return func(string) (schema.Stat, error) { return st, nil }
}

var (
	modified = time.Date(2026, 11, 1, 9, 30, 0, 250, time.UTC)
	created  = time.Date(2026, 10, 31, 8, 0, 0, 5, time.UTC)
)

func TestFileOf_CreatedPresent(t *testing.T) {
	st := schema.Stat{Size: 42, Mode: 0o640, Modified: modified, Created: created}

	f, err := schema.FileOf(filepath.Join("docs", "sub", "a.pdf"), "docs", stater(st))

	if err != nil {
		t.Fatal(err)
	}
	want := `{"name":"a.pdf","arg":"docs","path":"sub/a.pdf","size":42,"mode":"-rw-r-----",` +
		`"modified":"2026-11-01T09:30:00.000000250Z","created":"2026-10-31T08:00:00.000000005Z"}`
	if got := marshal(t, f); got != want {
		t.Errorf("FileOf =\n %s\nwant\n %s", got, want)
	}
}

// A filesystem that records no birth time leaves "created" out.
func TestFileOf_CreatedAbsent(t *testing.T) {
	st := schema.Stat{Size: 3, Mode: 0o644, Modified: modified}

	f, err := schema.FileOf("a.txt", "a.txt", stater(st))

	if err != nil {
		t.Fatal(err)
	}
	want := `{"name":"a.txt","arg":"a.txt","path":".","size":3,"mode":"-rw-r--r--",` +
		`"modified":"2026-11-01T09:30:00.000000250Z"}`
	if got := marshal(t, f); got != want {
		t.Errorf("FileOf =\n %s\nwant\n %s", got, want)
	}
}

func TestFileOf_StatError(t *testing.T) {
	failed := errors.New("stat failed")

	_, err := schema.FileOf("a.txt", "a.txt", func(string) (schema.Stat, error) { return schema.Stat{}, failed })

	if !errors.Is(err, failed) {
		t.Errorf("FileOf error = %v, want %v", err, failed)
	}
}

// Statx reads the real file: its size, mode, and modified time; its created time where the
// filesystem records one, which this test can't decide, and never off Linux.
func TestStatx_RealFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(path, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}

	st, err := schema.Statx(path)

	if err != nil {
		t.Fatal(err)
	}
	if st.Size != 5 || st.Mode != 0o600 || !st.Modified.Equal(info.ModTime()) {
		t.Errorf("Statx = %+v, want size 5, mode 0600, modified %v", st, info.ModTime())
	}
	if runtime.GOOS != "linux" && !st.Created.IsZero() {
		t.Errorf("Statx created = %v off Linux, want zero", st.Created)
	}
	if !st.Created.IsZero() && st.Created.After(time.Now()) {
		t.Errorf("Statx created = %v, in the future", st.Created)
	}
}

// Statx follows no symbolic link: it reads the link itself.
func TestStatx_SymbolicLink(t *testing.T) {
	dir := t.TempDir()
	target, link := filepath.Join(dir, "target"), filepath.Join(dir, "link")
	if err := os.WriteFile(target, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	st, err := schema.Statx(link)

	if err != nil {
		t.Fatal(err)
	}
	if st.Mode&fs.ModeSymlink == 0 {
		t.Errorf("Statx mode = %v, want a symbolic link's", st.Mode)
	}
}

func TestStatx_Missing(t *testing.T) {
	_, err := schema.Statx(filepath.Join(t.TempDir(), "absent"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Statx error = %v, want fs.ErrNotExist", err)
	}
}

func TestDetect(t *testing.T) {
	dir := t.TempDir()
	for _, tc := range []struct {
		name    string
		content string
		want    string
	}{
		{"text", "hello, world\n", "text/plain; charset=utf-8"},
		{"pdf", "%PDF-1.7\n", "application/pdf"},
		{"empty", "", "text/plain; charset=utf-8"},
		{"binary past 512", "\x00\x01\x02" + strings.Repeat("a", 600), "application/octet-stream"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name)
			if err := os.WriteFile(path, []byte(tc.content), 0o644); err != nil {
				t.Fatal(err)
			}

			got, err := schema.Detect(path)

			if err != nil || got != tc.want {
				t.Errorf("Detect = %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}

func TestDetect_Missing(t *testing.T) {
	if _, err := schema.Detect(filepath.Join(t.TempDir(), "absent")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Detect error = %v, want fs.ErrNotExist", err)
	}
}

// A nested part carries processing Part's fields, and nothing of a file: no path, After, or
// times.
func TestPartOf_Nested(t *testing.T) {
	p := dispatch.Part{
		Path: "/work/1/abc", Name: "notes.txt", MIME: "text/plain", Size: 12, SHA256: "abc",
		Kind: route.Unknown, Route: route.Text, Reason: "utf-8 text",
		Flags:  []inspect.Flag{{Check: inspect.DoubleExtension, Class: inspect.Active, Evidence: "notes.txt.exe"}},
		Parent: "def", Depth: 2, Where: "zip entry notes.txt", After: 9,
	}

	got := marshal(t, schema.PartOf(p))

	want := `{"name":"notes.txt","size":12,"sha256":"abc",` +
		`"content_type":{"route":"text","route_reason":"utf-8 text","stored":"text/plain"},` +
		`"flags":[{"check":"double-extension","class":"active","evidence":"notes.txt.exe"}],` +
		`"parent":"def","depth":2,"where":"zip entry notes.txt"}`
	if got != want {
		t.Errorf("PartOf =\n %s\nwant\n %s", got, want)
	}
}

// An input has no parent, place, or stored type, and those fields are absent.
func TestPartOf_Input(t *testing.T) {
	p := dispatch.Part{Path: "in.pdf", Name: "in.pdf", Size: 9, SHA256: "123", Kind: route.PDF, Route: route.Document}

	got := marshal(t, schema.PartOf(p))

	want := `{"name":"in.pdf","size":9,"sha256":"123","content_type":{"kind":"pdf","route":"document"},"depth":0}`
	if got != want {
		t.Errorf("PartOf =\n %s\nwant\n %s", got, want)
	}
}
