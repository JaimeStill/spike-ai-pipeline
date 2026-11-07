package app_test

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/standards-lab/go-core/process"

	docinfertest "github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/apptest"
	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/chunk"
	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/outdir"
)

// embedConfig is a config.json of the embeddings capability alone on the endpoint "fake" at url,
// with settings, a JSON object's members such as `"batch": 2`, added to it.
func embedConfig(url, settings string) string {
	if settings != "" {
		settings = ", " + settings
	}
	return fmt.Sprintf(`{
  "endpoints": {"fake": {"base_url": %q, "format": "completions"}},
  "embeddings": {"endpoint": "fake", "model": %q%s}
}`, url, embeddingModel, settings)
}

// runs returns the ids of the runs docinfer wrote in out, in the order they began.
func runs(t *testing.T, out string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(out, "runs"))
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, e := range entries {
		ids = append(ids, e.Name())
	}
	slices.Sort(ids)
	return ids
}

// result reads the latest run docinfer wrote in out: its manifest and its chunk lines.
func result(t *testing.T, out string) (outdir.Manifest, []outdir.Line) {
	t.Helper()
	ids := runs(t, out)
	if len(ids) == 0 {
		t.Fatalf("%s holds no run", out)
	}
	d, err := outdir.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	m, err := d.ReadManifest(ids[len(ids)-1])
	if err != nil {
		t.Fatal(err)
	}
	lines, err := d.ReadChunks(m.Run)
	if err != nil {
		t.Fatal(err)
	}
	return m, lines
}

// sent returns every input the endpoint's embeddings requests carried, in order.
func sent(e *docinfertest.Endpoint) []string {
	var out []string
	for _, r := range e.Embeddings() {
		out = append(out, r.Input...)
	}
	return out
}

// byInput returns the lines of input, of the source kind.
func byInput(lines []outdir.Line, input, kind string) []outdir.Line {
	var out []outdir.Line
	for _, l := range lines {
		if l.Input == input && l.Source.Kind == kind {
			out = append(out, l)
		}
	}
	return out
}

// A run embeds every text and Markdown output line, never a record, image, or audio directly, the
// image's vision text, and one metadata text per input with file metadata, the error line's input among them; writes a chunk
// line per chunk per occurrence, with its input, part, and output, and the vector as the endpoint
// returned it; sends each distinct chunk once; and records the run in its manifest.
func TestRun_Embeds(t *testing.T) {
	clean(t)
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA})
	cfg := configDir(t, map[string]string{"config.json": fullConfig(e.URL())})
	out := filepath.Join(t.TempDir(), "out")

	stdout, _ := run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)

	m, lines := result(t, out)
	ids := runs(t, out)
	if len(ids) != 1 || m.Run != ids[0] {
		t.Fatalf("runs = %q, manifest run %q, want one, the manifest's", ids, m.Run)
	}
	abs, err := filepath.Abs(docproc)
	if err != nil {
		t.Fatal(err)
	}
	if m.Schema != "1" || m.Start.IsZero() || m.End.IsZero() || m.Error != "" ||
		m.Docproc != (outdir.Docproc{Dir: abs, Run: latestRun}) || m.Chunks != "runs/"+m.Run+"/chunks.jsonl" {
		t.Errorf("manifest = %+v, want schema 1, its times, the docproc run %s, and its chunks", m, abs)
	}
	if len(m.Capabilities) != 2 || m.Capabilities[0].Name != "vision" {
		t.Fatalf("capabilities = %+v, want vision, then embeddings", m.Capabilities)
	}
	c := m.Capabilities[1]
	if c.Name != "embeddings" || c.Endpoint != "fake" || c.BaseURL != e.URL() || c.Model != embeddingModel ||
		fmt.Sprint(c.Settings) != "map[batch:16 chunk_size:2000 concurrency:4]" {
		t.Errorf("capability = %+v, want embeddings on fake with its defaults", c)
	}
	wantSkipped := []outdir.Skipped{{Name: "transcription", Reason: "configured; docinfer doesn't run it yet"}}
	if !slices.Equal(m.Skipped, wantSkipped) {
		t.Errorf("skipped = %+v, want %+v", m.Skipped, wantSkipped)
	}

	// Every line's vector is the endpoint's, not normalized, and its text its source's span.
	var minNorm, maxNorm = math.Inf(1), math.Inf(-1)
	for _, l := range lines {
		v := docinfertest.Vector(l.Text, docinfertest.DefaultDims)
		if l.Vector != outdir.EncodeVector(v) || l.Error != "" {
			t.Errorf("line %q: vector %q, error %q, want the endpoint's", l.Text, l.Vector, l.Error)
		}
		var sq float64
		for _, f := range v {
			sq += float64(f) * float64(f)
		}
		minNorm, maxNorm = min(minNorm, math.Sqrt(sq)), max(maxNorm, math.Sqrt(sq))
		if l.Source.Kind == outdir.SourceOutput {
			b, err := os.ReadFile(filepath.Join(docproc, filepath.FromSlash(l.Source.Ref)))
			if err != nil {
				t.Fatal(err)
			}
			if string(b[l.Span.Start:l.Span.End]) != l.Text || l.Output == nil || l.Source.Ref != l.Output.Ref {
				t.Errorf("line %+v: want its text its source file's span, and its output", l)
			}
		}
	}
	wantSpace := outdir.Space{
		Endpoint: "fake", Model: embeddingModel, Dimensions: docinfertest.DefaultDims, Vectors: m.Counts.Chunks,
		Unit: false, Tolerance: 1e-3, MinNorm: minNorm, MaxNorm: maxNorm, ChunkSize: 2000, Chunker: chunk.Version,
	}
	if m.Space == nil || *m.Space != wantSpace {
		t.Errorf("space = %+v, want %+v", m.Space, wantSpace)
	}

	// Text and Markdown outputs alone: five output lines, each one chunk at the default bound.
	var kinds []string
	for _, l := range lines {
		if l.Source.Kind == outdir.SourceOutput {
			kinds = append(kinds, l.Output.Kind)
		}
	}
	if want := []string{"text", "text", "text", "markdown", "text"}; !slices.Equal(kinds, want) {
		t.Errorf("output lines' kinds = %q, want %q", kinds, want)
	}

	// The same content under two inputs is two lines, its vector sent once.
	hello := "Hello from the docproc fixture."
	a, b := byInput(lines, "/srv/inputs/hello.txt", "output"), byInput(lines, "/srv/inputs/copy/hello.txt", "output")
	if len(a) != 1 || len(b) != 1 || a[0].Text != hello || b[0].Text != hello || a[0].Vector != b[0].Vector {
		t.Errorf("hello lines = %+v, %+v, want one each, the same vector", a, b)
	}
	if b[0].File == nil || b[0].File.Path != "copy/hello.txt" || b[0].Part.Outcome != "seen" {
		t.Errorf("copy line = %+v, want its own input's file and the seen part", b[0])
	}
	all := sent(e)
	for _, in := range all {
		if n := slices.Index(all, in); slices.Index(all[n+1:], in) >= 0 {
			t.Errorf("input %q sent twice, want each distinct chunk once", in)
		}
	}
	if !slices.Contains(all, hello) {
		t.Errorf("inputs sent = %q, want %q among them", all, hello)
	}

	// An output line carries its input's file metadata and content type, and its part's fields
	// and outcome, a failed part's among them.
	broken := byInput(lines, "/srv/inputs/broken.pdf", "output")
	if len(broken) != 1 || broken[0].Part == nil || *broken[0].Part != (outdir.Part{
		SHA256: "07e1e845732d27e3cd2f7b792c117ed9c3136200209bfd0c3720daec39173246", Name: "broken.pdf",
		Kind: "pdf", Route: "document", Depth: 0, Outcome: "failed",
	}) || broken[0].ContentType == nil || broken[0].ContentType.Detected != "application/pdf" ||
		broken[0].File == nil || broken[0].File.Name != "broken.pdf" || broken[0].Output.Meta["count"] != "2" {
		t.Errorf("broken.pdf lines = %+v, want the failed part's", broken)
	}

	// One metadata text per input with file metadata, the error line's included, listing no
	// part.
	var inputs []string
	for _, l := range lines {
		if l.Source.Kind == outdir.SourceMetadata {
			inputs = append(inputs, l.Input)
		}
	}
	wantInputs := []string{"/srv/inputs/hello.txt", "/srv/inputs/copy/hello.txt", "/srv/inputs/report.pdf",
		"/srv/inputs/notes.docx", "/srv/inputs/talk.wav", "/srv/inputs/blob.bin", "/srv/inputs/broken.pdf", "/srv/inputs/locked.bin"}
	if !slices.Equal(inputs, wantInputs) {
		t.Errorf("metadata lines' inputs = %q, want %q", inputs, wantInputs)
	}
	report := byInput(lines, "/srv/inputs/report.pdf", "metadata")
	wantReport := "file name: report.pdf\npath argument: /srv/inputs\npath: report.pdf\nsize: 48213 bytes\nmode: -rw-r--r--\n" +
		"modified: 2026-11-07T09:00:00.000000000Z\nkind: pdf\nroute: document\nroute reason: pdf signature\ndetected MIME type: application/pdf"
	if len(report) != 1 || report[0].Text != wantReport || report[0].Output != nil || report[0].Part.Outcome != "prepared" {
		t.Errorf("report.pdf metadata = %+v, want %q", report, wantReport)
	}
	locked := byInput(lines, "/srv/inputs/locked.bin", "metadata")
	if len(locked) != 1 || locked[0].Part != nil || locked[0].ContentType != nil || locked[0].File == nil ||
		!strings.HasPrefix(locked[0].Text, "file name: locked.bin\n") {
		t.Errorf("locked.bin metadata = %+v, want its file's alone", locked)
	}

	wantCounts := outdir.Counts{Outputs: 11, Texts: 5, Metadata: 8, Chunks: m.Counts.Chunks, Lines: 14,
		Embedded: m.Counts.Chunks, Requests: 1, Images: 1,
		Vision: &outdir.VisionCounts{Images: 1, Described: 1, Requests: 1, Texts: 1}}
	if !reflect.DeepEqual(m.Counts, wantCounts) || m.Counts.Chunks != len(all) {
		t.Errorf("counts = %+v, want %+v, its chunks the %d sent", m.Counts, wantCounts, len(all))
	}
	wantLine(t, stdout, `docinfer run `+m.Run+`: lines 14: .*chunks\.jsonl`)
}

// A second run over the same docproc run and configuration sends no request but the listing, and
// writes the same chunk lines under a run of its own.
func TestRun_RerunSendsNothing(t *testing.T) {
	clean(t)
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA})
	cfg := configDir(t, map[string]string{"config.json": embedConfig(e.URL(), `"chunk_size": 24`)})
	out := t.TempDir()

	run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)
	first, _ := result(t, out)
	firstLines, err := os.ReadFile(filepath.Join(out, first.Chunks))
	if err != nil {
		t.Fatal(err)
	}
	before := len(e.Embeddings())
	stdout, _ := run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)

	if n := len(e.Embeddings()) - before; n != 0 {
		t.Errorf("second run sent %d embeddings requests, want none", n)
	}
	second, _ := result(t, out)
	if second.Run == first.Run || len(runs(t, out)) != 2 {
		t.Errorf("runs = %q, want two", runs(t, out))
	}
	secondLines, err := os.ReadFile(filepath.Join(out, second.Chunks))
	if err != nil {
		t.Fatal(err)
	}
	if string(secondLines) != string(firstLines) {
		t.Errorf("second run's chunk lines differ from the first's")
	}
	if second.Counts.Held != first.Counts.Chunks || second.Counts.Embedded != 0 || second.Counts.Requests != 0 ||
		*second.Space != *first.Space {
		t.Errorf("second run = %+v, want every chunk held and the same space as %+v", second, first)
	}
	wantLine(t, stdout, `embeddings: texts 5, metadata 8, chunks \d+: held \d+, embedded 0, rejected 0; requests 0`)
}

// A changed chunk bound cuts the texts again and embeds them again.
func TestRun_ChunkSizeChangeReembeds(t *testing.T) {
	clean(t)
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA})
	cfg := configDir(t, map[string]string{"config.json": embedConfig(e.URL(), "")})
	out := t.TempDir()

	run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)
	first, _ := result(t, out)
	before := len(sent(e))
	t.Setenv("DOCINFER_EMBEDDINGS_CHUNK_SIZE", "20")
	run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)
	second, lines := result(t, out)

	if n := len(sent(e)) - before; n != second.Counts.Chunks || second.Counts.Held != 0 {
		t.Errorf("second run sent %d inputs, held %d, want every one of its %d chunks sent", n, second.Counts.Held, second.Counts.Chunks)
	}
	if second.Counts.Chunks <= first.Counts.Chunks || second.Space.ChunkSize != 20 {
		t.Errorf("second run: %d chunks, bound %d; want more than the first's %d, bound 20", second.Counts.Chunks, second.Space.ChunkSize, first.Counts.Chunks)
	}
	for _, l := range lines {
		if n := len([]rune(l.Text)); n > 20 {
			t.Errorf("chunk %q has %d runes, want at most 20", l.Text, n)
		}
	}
}

// Each request carries at most batch chunks, and at most concurrency requests are in flight.
func TestRun_BatchAndConcurrency(t *testing.T) {
	clean(t)
	var (
		mu             sync.Mutex
		inFlight, most int
	)
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA, OnEmbed: func(context.Context, int) {
		mu.Lock()
		inFlight++
		most = max(most, inFlight)
		mu.Unlock()
		time.Sleep(20 * time.Millisecond)
		mu.Lock()
		inFlight--
		mu.Unlock()
	}})
	cfg := configDir(t, map[string]string{"config.json": embedConfig(e.URL(), `"batch": 2, "concurrency": 2, "chunk_size": 16`)})
	out := t.TempDir()

	run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)

	m, _ := result(t, out)
	reqs := e.Embeddings()
	for _, r := range reqs {
		if len(r.Input) > 2 || r.Model != embeddingModel {
			t.Errorf("request %+v, want at most 2 inputs, for %s", r, embeddingModel)
		}
	}
	if want := (m.Counts.Chunks + 1) / 2; len(reqs) != want || m.Counts.Requests != want {
		t.Errorf("requests = %d, manifest %d, want %d for %d chunks", len(reqs), m.Counts.Requests, want, m.Counts.Chunks)
	}
	if most > 2 {
		t.Errorf("at most %d requests in flight, want 2", most)
	}
}

// A chunk the endpoint rejects is recorded on its lines, not fatal: its batch is sent again a
// chunk a request, so no other chunk is lost; a 4xx rejection is held, so a re-run sends nothing.
func TestRun_RejectedChunk(t *testing.T) {
	clean(t)
	bad := func(in string) bool { return strings.Contains(in, "Revenue") }
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA, Reject: bad})
	cfg := configDir(t, map[string]string{"config.json": embedConfig(e.URL(), "")})
	out := t.TempDir()

	stdout, _ := run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)

	m, lines := result(t, out)
	rejected := 0
	for _, l := range lines {
		switch {
		case bad(l.Text):
			rejected++
			if l.Vector != "" || !strings.Contains(l.Error, "400 Bad Request: input rejected") {
				t.Errorf("rejected line %q: vector %q, error %q, want no vector and the endpoint's error", l.Text, l.Vector, l.Error)
			}
		case l.Vector == "" || l.Error != "":
			t.Errorf("line %q: vector %q, error %q, want a vector", l.Text, l.Vector, l.Error)
		}
	}
	if rejected != 1 || m.Counts.Rejected != 1 || m.Counts.Embedded != m.Counts.Chunks-1 {
		t.Errorf("rejected lines %d, counts %+v, want one rejected and the rest embedded", rejected, m.Counts)
	}
	// One batch, rejected, then each chunk alone.
	if reqs := e.Embeddings(); len(reqs) != 1+m.Counts.Chunks || m.Counts.Requests != len(reqs) {
		t.Errorf("requests = %d, manifest %d, want the batch and then %d alone", len(reqs), m.Counts.Requests, m.Counts.Chunks)
	}
	wantLine(t, stdout, regexp.QuoteMeta("warning: embeddings: rejected chunk 0 of outputs/6466e450a16b77b865c5829d6b6c56d9f892956475642dbeb9ccc4340fe01b15/document-e88a7a4d9af5/1.txt: ")+".*input rejected.*")

	before := len(e.Embeddings())
	run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)
	if n := len(e.Embeddings()) - before; n != 0 {
		t.Errorf("re-run sent %d requests, want none: the rejection is held", n)
	}
	if again, _ := result(t, out); again.Counts.Rejected != 1 {
		t.Errorf("re-run counts = %+v, want the held rejection counted", again.Counts)
	}
}

// A run cancelled part way stores what completed; the next run sends only the rest. The
// cancelled run's manifest records its end and why it stopped, and it writes no chunk lines.
func TestRun_ResumesAfterCancel(t *testing.T) {
	clean(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA, OnEmbed: func(rctx context.Context, n int) {
		if n == 3 {
			cancel()
			select { // the run's cancel closes the request's connection
			case <-rctx.Done():
			case <-time.After(10 * time.Second):
			}
		}
	}})
	cfg := configDir(t, map[string]string{"config.json": embedConfig(e.URL(), `"batch": 1, "concurrency": 1`)})
	out := t.TempDir()
	a, _, stderr, _ := newApp(t)

	if code := a.Run(ctx, []string{"run", docproc, "--config", cfg, "--out", out}); code == process.ExitOK {
		t.Fatalf("cancelled run exited %d, want a failure", code)
	}
	if !strings.Contains(stderr.String(), "context canceled") {
		t.Errorf("stderr = %q, want the cancellation", stderr)
	}
	ids := runs(t, out)
	d, err := outdir.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	stopped, err := d.ReadManifest(ids[0])
	if err != nil {
		t.Fatal(err)
	}
	if stopped.End.IsZero() || !strings.Contains(stopped.Error, "context canceled") || stopped.Chunks != "" {
		t.Errorf("cancelled run's manifest = %+v, want its end, its error, and no chunks", stopped)
	}
	if _, err := os.Stat(filepath.Join(out, "runs", ids[0], "chunks.jsonl")); !os.IsNotExist(err) {
		t.Errorf("cancelled run's chunks.jsonl: stat = %v, want none", err)
	}
	first := sent(e)
	if len(first) != 3 {
		t.Fatalf("cancelled run sent %q, want three, one a request", first)
	}

	run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)

	m, lines := result(t, out)
	rest := sent(e)[3:]
	if slices.Contains(rest, first[0]) || slices.Contains(rest, first[1]) || !slices.Contains(rest, first[2]) {
		t.Errorf("re-run sent %q, want neither of %q, which completed, and %q, which didn't", rest, first[:2], first[2])
	}
	if m.Counts.Held != 2 || m.Counts.Embedded != m.Counts.Chunks-2 || len(rest) != m.Counts.Embedded {
		t.Errorf("re-run counts = %+v, sent %d, want two held and the rest embedded", m.Counts, len(rest))
	}
	for _, l := range lines {
		if l.Vector != outdir.EncodeVector(docinfertest.Vector(l.Text, docinfertest.DefaultDims)) {
			t.Errorf("line %q: vector %q, want the endpoint's", l.Text, l.Vector)
		}
	}
}
