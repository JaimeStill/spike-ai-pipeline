package app_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
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
	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/config"
	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/outdir"
)

// pngRef is the committed run's one image output, report.pdf's page 1, a PNG.
const pngRef = "outputs/6466e450a16b77b865c5829d6b6c56d9f892956475642dbeb9ccc4340fe01b15/document-e88a7a4d9af5/2.png"

// reportPDF is the input pngRef belongs to.
const reportPDF = "/srv/inputs/report.pdf"

// visionTokens is the usage the fake endpoint's vision reply reports, in every format.
var visionTokens = outdir.Usage{Input: docinfertest.VisionInputTokens, Output: docinfertest.VisionOutputTokens, Total: docinfertest.VisionInputTokens + docinfertest.VisionOutputTokens}

// visionConfig is a config.json of the embeddings and vision capabilities on the endpoint "fake"
// at url, speaking format, with vision, a JSON object's members such as `"max_tokens": 64`, added
// to the vision capability, and embed to the embeddings capability.
func visionConfig(url, format, vision, embed string) string {
	if vision != "" {
		vision = ", " + vision
	}
	if embed != "" {
		embed = ", " + embed
	}
	return fmt.Sprintf(`{
  "endpoints": {"fake": {"base_url": %q, "format": %q}},
  "embeddings": {"endpoint": "fake", "model": %q%s},
  "vision": {"endpoint": "fake", "model": %q%s}
}`, url, format, embeddingModel, embed, visionModel, vision)
}

// readFile returns the bytes of the file at name, failing the test when it can't be read.
func readFile(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// hexSum returns the lowercase hex SHA-256 of b.
func hexSum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// byKind returns the lines whose source is of kind.
func byKind(lines []outdir.Line, kind string) []outdir.Line {
	var out []outdir.Line
	for _, l := range lines {
		if l.Source.Kind == kind {
			out = append(out, l)
		}
	}
	return out
}

// The image goes to the vision model in the endpoint's format, each of the three, as a data URL
// of its bytes and media type beside the default prompt, capped at the default 2048 tokens; its
// text is stored in docinfer's directory, cut and embedded like any text, its line carrying the
// image output's input, part, and output; and the manifest records the capability and its
// counts.
func TestRun_Vision(t *testing.T) {
	png := readFile(t, filepath.Join(docproc, filepath.FromSlash(pngRef)))
	for _, format := range []string{"completions", "messages", "responses"} {
		t.Run(format, func(t *testing.T) {
			clean(t)
			e := docinfertest.Serve(t, docinfertest.Fake{Models: setA})
			cfg := configDir(t, map[string]string{"config.json": visionConfig(e.URL(), format, "", "")})
			out := t.TempDir()

			stdout, _ := run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)

			vs := e.Visions()
			if len(vs) != 1 {
				t.Fatalf("vision requests = %d, want one", len(vs))
			}
			v := vs[0]
			if v.Format != format || v.Model != visionModel || v.Prompt != config.DefaultVisionPrompt ||
				v.MediaType != "image/png" || !bytes.Equal(v.Image, png) || v.MaxTokens != config.DefaultVisionMaxTokens {
				t.Errorf("vision request = {%s %s %q %s %d bytes, max %d}, want {%s %s %q image/png %d bytes, max %d}",
					v.Format, v.Model, v.Prompt, v.MediaType, len(v.Image), v.MaxTokens,
					format, visionModel, config.DefaultVisionPrompt, len(png), config.DefaultVisionMaxTokens)
			}

			m, lines := result(t, out)
			c := m.Capabilities[0]
			if len(m.Capabilities) != 2 || c.Name != "vision" || c.Endpoint != "fake" || c.BaseURL != e.URL() || c.Model != visionModel ||
				fmt.Sprint(c.Settings) != fmt.Sprintf("map[concurrency:4 max_tokens:2048 prompt:%s]", config.DefaultVisionPrompt) {
				t.Errorf("capabilities = %+v, want vision with its defaults, then embeddings", m.Capabilities)
			}
			if m.Counts.Images != 1 || m.Counts.Vision == nil ||
				*m.Counts.Vision != (outdir.VisionCounts{Images: 1, Described: 1, Requests: 1, Texts: 1}) {
				t.Errorf("counts = %+v, vision %+v, want one image described", m.Counts, m.Counts.Vision)
			}
			// Every format's reply gives the same usage; Messages gives no total, so it is the sum.
			if tk := m.Counts.Tokens.Vision; tk == nil || *tk != visionTokens {
				t.Errorf("vision tokens = %+v, want the reply's usage", tk)
			}

			text := docinfertest.VisionText(png)
			vl := byKind(lines, outdir.SourceVision)
			if len(vl) != 1 {
				t.Fatalf("vision lines = %+v, want one", vl)
			}
			l := vl[0]
			wantOutput := outdir.Output{Kind: "image", Name: "page 1", MediaType: "image/png", Ref: pngRef,
				Meta: map[string]string{"page": "1", "count": "1"}}
			if l.Text != text || l.Input != reportPDF || l.Part == nil || l.Part.Name != "report.pdf" || l.Part.Outcome != "prepared" ||
				l.Output == nil || !reflect.DeepEqual(*l.Output, wantOutput) ||
				l.File == nil || l.File.Name != "report.pdf" || l.ContentType == nil || l.Source.Incomplete != "" {
				t.Errorf("vision line = %+v, want the image's text with report.pdf's input, part, and output", l)
			}
			if l.Vector != outdir.EncodeVector(docinfertest.Vector(text, docinfertest.DefaultDims)) || l.Error != "" {
				t.Errorf("vision line vector %q, error %q, want the endpoint's", l.Vector, l.Error)
			}
			if !slices.Contains(sent(e), text) {
				t.Errorf("embedded %q, want the vision text among them", sent(e))
			}
			// The text is a file of docinfer's, named by its content, relative to --out.
			if l.Source.SHA256 != hexSum([]byte(text)) || l.Source.File != path.Join("texts", l.Source.SHA256[:2], l.Source.SHA256+".txt") {
				t.Errorf("vision source = %+v, want its text's file under texts/", l.Source)
			}
			if got := string(readFile(t, filepath.Join(out, filepath.FromSlash(l.Source.File)))); got != text {
				t.Errorf("stored text = %q, want %q", got, text)
			}
			wantLine(t, stdout, regexp.QuoteMeta("vision: images 1, distinct 1: held 0, described 1, incomplete 0, refused 0; requests 1"))
		})
	}
}

// The configured prompt and cap are what is sent, and a changed prompt sends the image again.
func TestRun_VisionSettings(t *testing.T) {
	clean(t)
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA})
	cfg := configDir(t, map[string]string{"config.json": visionConfig(e.URL(), "messages", `"prompt": "Describe it.", "max_tokens": 64`, "")})
	out := t.TempDir()

	run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)
	t.Setenv("DOCINFER_VISION_PROMPT", "Describe it again.")
	run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)

	vs := e.Visions()
	if len(vs) != 2 || vs[0].Prompt != "Describe it." || vs[0].MaxTokens != 64 || vs[1].Prompt != "Describe it again." {
		t.Errorf("vision requests = %d, want two, the first with \"Describe it.\" and 64, the second the new prompt", len(vs))
	}
	m, _ := result(t, out)
	if m.Capabilities[0].Settings["prompt"] != "Describe it again." || m.Capabilities[0].Settings["max_tokens"] != float64(64) {
		t.Errorf("vision settings = %v, want the new prompt and 64", m.Capabilities[0].Settings)
	}
}

// A reply cut at the output cap, as each format reports it, is recorded incomplete, its text kept
// and embedded, and the terminal warns of it.
func TestRun_VisionIncomplete(t *testing.T) {
	const cut = "Three shapes: a red square, a green"
	for _, format := range []string{"completions", "messages", "responses"} {
		t.Run(format, func(t *testing.T) {
			clean(t)
			e := docinfertest.Serve(t, docinfertest.Fake{Models: setA, Vision: func(docinfertest.VisionRequest) docinfertest.VisionReply {
				return docinfertest.VisionReply{Text: cut, Incomplete: true}
			}})
			cfg := configDir(t, map[string]string{"config.json": visionConfig(e.URL(), format, "", "")})
			out := t.TempDir()

			stdout, _ := run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)

			m, lines := result(t, out)
			vl := byKind(lines, outdir.SourceVision)
			if len(vl) != 1 || vl[0].Text != cut || vl[0].Source.Incomplete != "max_output_tokens" || vl[0].Vector == "" {
				t.Errorf("vision lines = %+v, want the cut text, incomplete at max_output_tokens, embedded", vl)
			}
			if vc := m.Counts.Vision; vc == nil || vc.Incomplete != 1 || vc.Described != 1 {
				t.Errorf("vision counts = %+v, want one incomplete", vc)
			}
			wantLine(t, stdout, regexp.QuoteMeta("warning: vision: incomplete text of image "+pngRef+": max_output_tokens"))
		})
	}
}

// An image the endpoint refuses with a 4xx is recorded, not fatal, and held: the run embeds the
// rest, the image has no text and no line, and a re-run sends no vision request. One refused with
// any other status is recorded but not held, so a re-run sends it again.
func TestRun_VisionRefused(t *testing.T) {
	for _, tc := range []struct {
		status int
		held   bool
	}{{400, true}, {503, false}} {
		t.Run(fmt.Sprint(tc.status), func(t *testing.T) {
			clean(t)
			e := docinfertest.Serve(t, docinfertest.Fake{Models: setA, Vision: func(docinfertest.VisionRequest) docinfertest.VisionReply {
				return docinfertest.VisionReply{Status: tc.status}
			}})
			cfg := configDir(t, map[string]string{"config.json": visionConfig(e.URL(), "completions", "", "")})
			out := t.TempDir()

			stdout, _ := run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)

			m, lines := result(t, out)
			if vl := byKind(lines, outdir.SourceVision); len(vl) != 0 {
				t.Errorf("vision lines = %+v, want none", vl)
			}
			if vc := m.Counts.Vision; vc == nil || *vc != (outdir.VisionCounts{Images: 1, Refused: 1, Requests: 1}) {
				t.Errorf("vision counts = %+v, want the one image refused", vc)
			}
			if tk := m.Counts.Tokens.Vision; tk == nil || *tk != (outdir.Usage{}) {
				t.Errorf("vision tokens = %+v, want none: a refusal carries no usage", tk)
			}
			if m.Counts.Texts != 5 || m.Counts.Lines != 13 || m.Error != "" {
				t.Errorf("counts = %+v, error %q, want every other text embedded", m.Counts, m.Error)
			}
			wantLine(t, stdout, regexp.QuoteMeta(fmt.Sprintf("warning: vision: refused image %s: %d ", pngRef, tc.status))+".*image refused.*")

			run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)
			again, _ := result(t, out)
			want := outdir.VisionCounts{Images: 1, Held: 1, Refused: 1}
			if !tc.held {
				want = outdir.VisionCounts{Images: 1, Refused: 1, Requests: 1}
			}
			if n := len(e.Visions()); again.Counts.Vision == nil || *again.Counts.Vision != want || (n == 1) != tc.held {
				t.Errorf("re-run: %d vision requests in all, counts %+v; want %+v", n, again.Counts.Vision, want)
			}
		})
	}
}

// A second run over the same docproc run and configuration sends no vision request and no
// embeddings request, and writes the same chunk lines.
func TestRun_VisionRerunSendsNothing(t *testing.T) {
	clean(t)
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA})
	cfg := configDir(t, map[string]string{"config.json": visionConfig(e.URL(), "responses", "", "")})
	out := t.TempDir()

	run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)
	first, _ := result(t, out)
	firstLines := readFile(t, filepath.Join(out, first.Chunks))
	visions, embeds := len(e.Visions()), len(e.Embeddings())
	stdout, _ := run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)

	if v, n := len(e.Visions())-visions, len(e.Embeddings())-embeds; v != 0 || n != 0 {
		t.Errorf("second run sent %d vision and %d embeddings requests, want none", v, n)
	}
	second, _ := result(t, out)
	if !bytes.Equal(readFile(t, filepath.Join(out, second.Chunks)), firstLines) {
		t.Errorf("second run's chunk lines differ from the first's")
	}
	if vc := second.Counts.Vision; vc == nil || *vc != (outdir.VisionCounts{Images: 1, Held: 1, Texts: 1}) {
		t.Errorf("second run's vision counts = %+v, want the image held", vc)
	}
	if tk := second.Counts.Tokens; !reflect.DeepEqual(tk, outdir.Tokens{Vision: &outdir.Usage{}}) {
		t.Errorf("second run's tokens = %+v, want none: it sent no request", tk)
	}
	wantLine(t, stdout, regexp.QuoteMeta("vision: images 1, distinct 1: held 1, described 0, incomplete 0, refused 0; requests 0"))
}

// A changed chunk bound cuts and embeds every text again, the vision text among them, and sends
// no vision request: vision's results are held apart from the chunk settings.
func TestRun_VisionChunkSizeChange(t *testing.T) {
	clean(t)
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA})
	cfg := configDir(t, map[string]string{"config.json": visionConfig(e.URL(), "completions", "", "")})
	out := t.TempDir()

	run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)
	visions, before := len(e.Visions()), len(sent(e))
	t.Setenv("DOCINFER_EMBEDDINGS_CHUNK_SIZE", "12")
	run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)

	if n := len(e.Visions()) - visions; n != 0 {
		t.Errorf("second run sent %d vision requests, want none", n)
	}
	m, lines := result(t, out)
	if n := len(sent(e)) - before; n != m.Counts.Chunks || m.Counts.Held != 0 {
		t.Errorf("second run embedded %d, held %d; want all %d chunks embedded", n, m.Counts.Held, m.Counts.Chunks)
	}
	vl := byKind(lines, outdir.SourceVision)
	for _, l := range vl {
		if len([]rune(l.Text)) > 12 || l.Vector == "" {
			t.Errorf("vision chunk %q: want at most 12 runes, embedded", l.Text)
		}
	}
	if len(vl) < 2 {
		t.Errorf("vision lines = %d, want the text cut into several", len(vl))
	}
	if vc := m.Counts.Vision; vc == nil || vc.Held != 1 || vc.Requests != 0 {
		t.Errorf("vision counts = %+v, want the image held", vc)
	}
}

// With vision unset, the image is skipped and counted: no vision request, no vision line, and the
// manifest lists vision as skipped.
func TestRun_VisionUnset(t *testing.T) {
	clean(t)
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA})
	cfg := configDir(t, map[string]string{"config.json": embedConfig(e.URL(), "")})
	out := t.TempDir()

	stdout, _ := run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)

	if vs := e.Visions(); len(vs) != 0 {
		t.Errorf("vision requests = %d, want none", len(vs))
	}
	m, lines := result(t, out)
	if vl := byKind(lines, outdir.SourceVision); len(vl) != 0 {
		t.Errorf("vision lines = %+v, want none", vl)
	}
	want := []outdir.Skipped{{Name: "vision", Reason: "not configured"}, {Name: "transcription", Reason: "not configured"}}
	if !slices.Equal(m.Skipped, want) || len(m.Capabilities) != 1 || m.Counts.Images != 1 || m.Counts.Vision != nil || m.Counts.Tokens.Vision != nil {
		t.Errorf("manifest skipped %+v, capabilities %+v, counts %+v; want vision skipped, its image counted", m.Skipped, m.Capabilities, m.Counts)
	}
	wantLine(t, stdout, regexp.QuoteMeta("vision: not configured; images 1 skipped"))
}

// An empty vision text is recorded, held, and stored, and cut to no chunk.
func TestRun_VisionEmptyText(t *testing.T) {
	clean(t)
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA, Vision: func(docinfertest.VisionRequest) docinfertest.VisionReply {
		return docinfertest.VisionReply{Text: ""}
	}})
	cfg := configDir(t, map[string]string{"config.json": visionConfig(e.URL(), "completions", "", "")})
	out := t.TempDir()

	run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)
	run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)

	if n := len(e.Visions()); n != 1 {
		t.Errorf("vision requests = %d, want one: the empty text is held", n)
	}
	m, lines := result(t, out)
	if vl := byKind(lines, outdir.SourceVision); len(vl) != 0 {
		t.Errorf("vision lines = %+v, want none", vl)
	}
	if vc := m.Counts.Vision; vc == nil || *vc != (outdir.VisionCounts{Images: 1, Held: 1, Texts: 1}) {
		t.Errorf("vision counts = %+v, want the empty text held and counted", vc)
	}
	empty := hexSum(nil)
	if b := readFile(t, filepath.Join(out, "texts", empty[:2], empty+".txt")); len(b) != 0 {
		t.Errorf("stored text = %q, want empty", b)
	}
}

// images copies the committed docproc run and adds image output lines to report.pdf's, after
// its own: one of the committed image again, under its own ref; one of a copy of its bytes under
// another ref; and n images of their own, each the committed one's bytes and a byte of its own.
// It returns the copy, and the bytes of each image it added of its own.
func images(t *testing.T, n int) (string, [][]byte) {
	t.Helper()
	dir := copyDocproc(t)
	record := filepath.Join(dir, "runs", latestRun, "record.jsonl")
	b := readFile(t, record)
	var line map[string]any
	for l := range bytes.SplitSeq(b, []byte("\n")) {
		if bytes.Contains(l, []byte(`"kind":"image"`)) {
			if err := json.Unmarshal(l, &line); err != nil {
				t.Fatal(err)
			}
		}
	}
	png := readFile(t, filepath.Join(docproc, filepath.FromSlash(pngRef)))
	add := func(ref string, data []byte) {
		if data != nil {
			p := filepath.Join(dir, filepath.FromSlash(ref))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, data, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		line["output"].(map[string]any)["ref"] = ref
		l, err := json.Marshal(line)
		if err != nil {
			t.Fatal(err)
		}
		b = append(b, append(l, '\n')...)
	}
	add(pngRef, nil)
	add("outputs/extra/copy.png", png)
	var own [][]byte
	for i := range n {
		data := append(slices.Clone(png), byte(i))
		own = append(own, data)
		add(fmt.Sprintf("outputs/extra/%d.png", i), data)
	}
	if err := os.WriteFile(record, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir, own
}

// Each distinct image is sent once, whatever output lines name it, and each line has its vision
// text; at most concurrency vision requests are in flight.
func TestRun_VisionDistinctAndConcurrency(t *testing.T) {
	clean(t)
	var (
		mu             sync.Mutex
		inFlight, most int
	)
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA, OnVision: func(context.Context, int) {
		mu.Lock()
		inFlight++
		most = max(most, inFlight)
		mu.Unlock()
		time.Sleep(20 * time.Millisecond)
		mu.Lock()
		inFlight--
		mu.Unlock()
	}})
	dir, own := images(t, 4)
	cfg := configDir(t, map[string]string{"config.json": visionConfig(e.URL(), "completions", `"concurrency": 2`, "")})
	out := t.TempDir()

	run(t, process.ExitOK, "run", dir, "--config", cfg, "--out", out)

	vs := e.Visions()
	if len(vs) != 5 {
		t.Errorf("vision requests = %d, want 5, one per distinct image", len(vs))
	}
	seen := map[string]int{}
	for _, v := range vs {
		seen[hexSum(v.Image)]++
	}
	for s, n := range seen {
		if n != 1 {
			t.Errorf("image %s sent %d times, want once", s, n)
		}
	}
	m, lines := result(t, out)
	if m.Counts.Images != 7 || m.Counts.Vision == nil ||
		*m.Counts.Vision != (outdir.VisionCounts{Images: 5, Described: 5, Requests: 5, Texts: 7}) {
		t.Errorf("counts = %+v, vision %+v, want 7 image lines, 5 distinct", m.Counts, m.Counts.Vision)
	}
	vl := byKind(lines, outdir.SourceVision)
	var refs []string
	for _, l := range vl {
		refs = append(refs, l.Output.Ref)
	}
	wantRefs := []string{pngRef, pngRef, "outputs/extra/copy.png", "outputs/extra/0.png", "outputs/extra/1.png", "outputs/extra/2.png", "outputs/extra/3.png"}
	if !slices.Equal(refs, wantRefs) {
		t.Errorf("vision lines' refs = %q, want %q", refs, wantRefs)
	}
	if len(vl) == 7 && (vl[2].Text != vl[0].Text || vl[3].Text != docinfertest.VisionText(own[0])) {
		t.Errorf("vision texts = %q, %q, %q; want the copy's the committed image's, and each own image its own", vl[0].Text, vl[2].Text, vl[3].Text)
	}
	if most > 2 {
		t.Errorf("at most %d vision requests in flight, want 2", most)
	}
}

// A run cancelled during vision stops: its manifest records the error and it writes no chunk
// lines; the images described before it are held, so the next run sends only the rest.
func TestRun_VisionResumesAfterCancel(t *testing.T) {
	clean(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA, OnVision: func(rctx context.Context, n int) {
		if n == 3 {
			cancel()
			select { // the run's cancel closes the request's connection
			case <-rctx.Done():
			case <-time.After(10 * time.Second):
			}
		}
	}})
	dir, _ := images(t, 3)
	cfg := configDir(t, map[string]string{"config.json": visionConfig(e.URL(), "completions", `"concurrency": 1`, "")})
	out := t.TempDir()
	a, _, stderr, _ := newApp(t)

	if code := a.Run(ctx, []string{"run", dir, "--config", cfg, "--out", out}); code == process.ExitOK {
		t.Fatalf("cancelled run exited %d, want a failure", code)
	}
	if !strings.Contains(stderr.String(), "context canceled") {
		t.Errorf("stderr = %q, want the cancellation", stderr)
	}
	d, err := outdir.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	stopped, err := d.ReadManifest(runs(t, out)[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stopped.Error, "vision: ") || stopped.Chunks != "" || len(e.Embeddings()) != 0 {
		t.Errorf("cancelled run's manifest = %+v, %d embeddings requests; want vision's error, no chunks, nothing embedded", stopped, len(e.Embeddings()))
	}

	run(t, process.ExitOK, "run", dir, "--config", cfg, "--out", out)

	m, _ := result(t, out)
	vs := e.Visions()
	if vc := m.Counts.Vision; len(vs) != 3+2 || vc == nil || *vc != (outdir.VisionCounts{Images: 4, Held: 2, Described: 2, Requests: 2, Texts: 6}) {
		t.Errorf("%d vision requests in all, re-run counts %+v; want two held and the other two sent", len(vs), m.Counts.Vision)
	}
}
