package outdir_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/outdir"
	"github.com/JaimeStill/spike-ai-pipeline/internal/schema"
)

// marshal returns v as compact JSON, <, >, and & left as they are.
func marshal(t *testing.T, v any) string {
	t.Helper()
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// The chunk line's JSON shape, every field set, is pinned: a reader keys on these names.
func TestLine_JSONShape(t *testing.T) {
	modified := schema.Time(time.Date(2026, 11, 7, 9, 0, 0, 0, time.UTC))
	l := outdir.Line{
		Text: "Page one.", Index: 2, Span: outdir.Span{Start: 10, End: 19},
		Source: outdir.Source{Kind: "vision", Ref: "outputs/ab/doc-1/1.txt", File: "texts/ab/ab12.txt", SHA256: "s0", Incomplete: "max_output_tokens"},
		Input:  "/srv/in/a.pdf",
		File:   &schema.File{Name: "a.pdf", Arg: "/srv/in", Path: "a.pdf", Size: 3, Mode: "-rw-r--r--", Modified: modified, Created: modified},
		ContentType: &schema.ContentType{Kind: "pdf", Route: "document", RouteReason: "pdf signature", Stored: "application/pdf",
			Detected: "application/pdf"},
		Part: &outdir.Part{
			Name: "a.pdf", Size: 5, SHA256: "p0",
			ContentType: schema.ContentType{Kind: "pdf", Route: "document", RouteReason: "pdf signature", Stored: "application/pdf"},
			Flags:       []schema.Flag{{Check: "pdf-js", Class: "active", Evidence: "/JS 1"}},
			Parent:      "q0", Depth: 1, Where: "zip entry a.pdf",
			Kind: "pdf", Route: "document", Outcome: "prepared",
		},
		Output: &outdir.Output{Kind: "image", Name: "page 1", MediaType: "image/png", Ref: "outputs/ab/doc-1/1.png", Meta: map[string]string{"page": "1"}},
		Vector: "AACAPw==", Error: "400 Bad Request",
	}
	want := `{"text":"Page one.","index":2,"span":{"start":10,"end":19},` +
		`"source":{"kind":"vision","ref":"outputs/ab/doc-1/1.txt","file":"texts/ab/ab12.txt","sha256":"s0","incomplete":"max_output_tokens"},` +
		`"input":"/srv/in/a.pdf",` +
		`"file":{"name":"a.pdf","arg":"/srv/in","path":"a.pdf","size":3,"mode":"-rw-r--r--","modified":"2026-11-07T09:00:00.000000000Z","created":"2026-11-07T09:00:00.000000000Z"},` +
		`"content_type":{"kind":"pdf","route":"document","route_reason":"pdf signature","stored":"application/pdf","detected":"application/pdf"},` +
		`"part":{"name":"a.pdf","size":5,"sha256":"p0",` +
		`"content_type":{"kind":"pdf","route":"document","route_reason":"pdf signature","stored":"application/pdf"},` +
		`"flags":[{"check":"pdf-js","class":"active","evidence":"/JS 1"}],"parent":"q0","depth":1,"where":"zip entry a.pdf",` +
		`"kind":"pdf","route":"document","outcome":"prepared"},` +
		`"output":{"kind":"image","name":"page 1","media_type":"image/png","ref":"outputs/ab/doc-1/1.png","meta":{"page":"1"}},` +
		`"vector":"AACAPw==","error":"400 Bad Request"}` + "\n"
	if got := marshal(t, l); got != want {
		t.Errorf("Line JSON =\n%s want\n%s", got, want)
	}

	// The optional fields are absent, not null or empty, when unset.
	bare := outdir.Line{Text: "x", Span: outdir.Span{End: 1}, Source: outdir.Source{Kind: "metadata", SHA256: "s"}, Input: "/i", Vector: "v"}
	want = `{"text":"x","index":0,"span":{"start":0,"end":1},"source":{"kind":"metadata","sha256":"s"},"input":"/i","vector":"v"}` + "\n"
	if got := marshal(t, bare); got != want {
		t.Errorf("Line JSON =\n%s want\n%s", got, want)
	}
}

// A vector survives its base64 form exactly, as little-endian float32s.
func TestVector_RoundTrip(t *testing.T) {
	v := []float32{1, -0.5, 0.25, 0}
	s := outdir.EncodeVector(v)
	if s != "AACAPwAAAL8AAIA+AAAAAA==" {
		t.Errorf("EncodeVector = %q", s)
	}
	back, err := outdir.DecodeVector(s)
	if err != nil || !slices.Equal(back, v) {
		t.Errorf("DecodeVector = %v, %v, want %v", back, err, v)
	}
	if _, err := outdir.DecodeVector("AAA="); err == nil {
		t.Errorf("DecodeVector of two bytes succeeded, want an error")
	}
}

// Every part of a key changes it, and no two tuples share one by moving a boundary.
func TestKey(t *testing.T) {
	base := outdir.Key("embeddings", "m", "chunk_size=2000 chunker=1", "c")
	for _, k := range []string{
		outdir.Key("vision", "m", "chunk_size=2000 chunker=1", "c"),
		outdir.Key("embeddings", "n", "chunk_size=2000 chunker=1", "c"),
		outdir.Key("embeddings", "m", "chunk_size=1000 chunker=1", "c"),
		outdir.Key("embeddings", "m", "chunk_size=2000 chunker=1", "d"),
		outdir.Key("embeddings", "mc", "hunk_size=2000 chunker=1", "c"),
	} {
		if k == base {
			t.Errorf("key %s repeats the base's", k)
		}
	}
	if len(base) != 64 || base != outdir.Key("embeddings", "m", "chunk_size=2000 chunker=1", "c") {
		t.Errorf("Key = %q, want 64 hex digits, the same each call", base)
	}
}

// A held result reads back as written; one never written is a miss, not an error.
func TestDir_Held(t *testing.T) {
	d, err := outdir.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	key := outdir.Key("embeddings", "m", "s", "c")
	var got outdir.Embedding
	if ok, err := d.Held("embeddings", key, &got); ok || err != nil {
		t.Fatalf("Held before Hold = %v, %v, want a miss", ok, err)
	}
	want := outdir.Embedding{Model: "m", Settings: "s", Content: "c", Vector: "AACAPw==", Dimensions: 1}
	rel, err := d.Hold("embeddings", key, want)
	if err != nil {
		t.Fatal(err)
	}
	if rel != "results/embeddings/"+key[:2]+"/"+key+".json" {
		t.Errorf("Hold path = %q", rel)
	}
	if ok, err := d.Held("embeddings", key, &got); !ok || err != nil || got != want {
		t.Errorf("Held = %v, %v, %+v, want %+v", ok, err, got, want)
	}
}

// A text is written whole under its SHA-256 and read back; one changed since it was written is an
// error, not its text.
func TestDir_Text(t *testing.T) {
	d, err := outdir.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	rel, sum, err := d.WriteText("a red square")
	if err != nil {
		t.Fatal(err)
	}
	if want := "texts/" + sum[:2] + "/" + sum + ".txt"; rel != want || len(sum) != 64 {
		t.Errorf("WriteText = %q, %q; want %q", rel, sum, want)
	}
	if got, err := d.ReadText(rel, sum); err != nil || got != "a red square" {
		t.Errorf("ReadText = %q, %v; want the text", got, err)
	}
	if err := os.WriteFile(filepath.Join(d.Root(), filepath.FromSlash(rel)), []byte("a blue circle"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := d.ReadText(rel, sum); err == nil {
		t.Errorf("ReadText of a changed text: no error, want one")
	}
}
