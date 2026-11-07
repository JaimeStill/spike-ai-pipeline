package app_test

import (
	"bytes"
	"context"
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
	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/outdir"
)

// talkWAV is the committed run's audio input, whose part the audio stage cut into windowRefs.
const talkWAV = "/srv/inputs/talk.wav"

// windowRefs are talk.wav's three windows, in plan order.
var windowRefs = []string{
	"outputs/8a5c155464514137dcde8aacd5f9c8d29c564ca084fb85cf8a16af958421ee87/audio-4f9893728a26/2.wav",
	"outputs/8a5c155464514137dcde8aacd5f9c8d29c564ca084fb85cf8a16af958421ee87/audio-4f9893728a26/3.wav",
	"outputs/8a5c155464514137dcde8aacd5f9c8d29c564ca084fb85cf8a16af958421ee87/audio-4f9893728a26/4.wav",
}

// windowTexts are the transcripts seamTexts gives talk.wav's windows, in plan order: each shares
// its last two words with the next one's first two, as an overlap's words are, so Join splices
// each seam there, into joinedText.
var windowTexts = []string{
	"the quick brown fox jumps over",
	"jumps over the lazy dog and",
	"dog and then it ran away",
}

// joinedText is windowTexts joined, each seam matched on its two shared words.
const joinedText = "the quick brown fox jumps over the lazy dog and then it ran away"

// transcriptionConfig is a config.json of the embeddings and transcription capabilities on the
// endpoint "fake" at url, with transcription, a JSON object's members such as `"prompt": "x"`,
// added to the transcription capability.
func transcriptionConfig(url, transcription string) string {
	if transcription != "" {
		transcription = ", " + transcription
	}
	return fmt.Sprintf(`{
  "endpoints": {"fake": {"base_url": %q, "format": "completions"}},
  "embeddings": {"endpoint": "fake", "model": %q},
  "transcription": {"endpoint": "fake", "model": %q%s}
}`, url, embeddingModel, audioModel, transcription)
}

// windows returns the bytes of talk.wav's windows, in plan order.
func windows(t *testing.T) [][]byte {
	t.Helper()
	var out [][]byte
	for _, ref := range windowRefs {
		out = append(out, readFile(t, filepath.Join(docproc, filepath.FromSlash(ref))))
	}
	return out
}

// seamTexts returns a Fake's Transcribe answering each of talk.wav's windows with its
// windowTexts transcript, and refusing the window at refuse, when it is one, with status.
func seamTexts(t *testing.T, refuse, status int) func(docinfertest.TranscriptionRequest) docinfertest.TranscriptionReply {
	byContent := map[string]int{}
	for i, w := range windows(t) {
		byContent[hexSum(w)] = i
	}
	return func(r docinfertest.TranscriptionRequest) docinfertest.TranscriptionReply {
		i, ok := byContent[hexSum(r.File)]
		switch {
		case !ok:
			return docinfertest.TranscriptionReply{Text: docinfertest.TranscriptText(r.File)}
		case i == refuse:
			return docinfertest.TranscriptionReply{Status: status}
		}
		return docinfertest.TranscriptionReply{Text: windowTexts[i]}
	}
}

// editRecord rewrites the latest run's record in dir, a copy of the committed docproc run, line
// by line, as edit returns its lines.
func editRecord(t *testing.T, dir string, edit func(lines [][]byte) [][]byte) {
	t.Helper()
	record := filepath.Join(dir, "runs", latestRun, "record.jsonl")
	lines := bytes.Split(bytes.TrimSuffix(readFile(t, record), []byte("\n")), []byte("\n"))
	b := append(bytes.Join(edit(lines), []byte("\n")), '\n')
	if err := os.WriteFile(record, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// isWindow reports whether line is an audio window's output line.
func isWindow(line []byte) bool { return bytes.Contains(line, []byte(`"kind":"audio"`)) }

// Each of the part's windows goes to the audio model's transcription route as a multipart
// upload, the model's and no prompt or language; the transcripts are joined in plan order, each
// seam matched on its overlap's words, and the joined text is stored, recorded in the manifest
// with its seams, and cut and embedded like any text, its line carrying talk.wav's input and
// part and its first window as its output.
func TestRun_Transcription(t *testing.T) {
	clean(t)
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA, Transcribe: seamTexts(t, -1, 0)})
	cfg := configDir(t, map[string]string{"config.json": transcriptionConfig(e.URL(), "")})
	out := t.TempDir()

	stdout, _ := run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)

	trs := e.Transcriptions()
	if len(trs) != 3 {
		t.Fatalf("transcription requests = %d, want three, one per window", len(trs))
	}
	ws := windows(t)
	names := map[string]string{}
	for i, w := range ws {
		names[hexSum(w)] = path.Base(windowRefs[i])
	}
	for _, r := range trs {
		if r.Model != audioModel || r.Prompt != "" || r.Language != "" || r.ResponseFormat != "json" || names[hexSum(r.File)] != r.FileName {
			t.Errorf("transcription request = {%s %q %q %s %s, %d bytes}, want {%s \"\" \"\" json}, one window's bytes under its name",
				r.Model, r.Prompt, r.Language, r.ResponseFormat, r.FileName, len(r.File), audioModel)
		}
	}

	m, lines := result(t, out)
	c := m.Capabilities[0]
	if len(m.Capabilities) != 2 || c.Name != "transcription" || c.Endpoint != "fake" || c.BaseURL != e.URL() || c.Model != audioModel ||
		fmt.Sprint(c.Settings) != "map[concurrency:4 language: prompt:]" {
		t.Errorf("capabilities = %+v, want transcription with its defaults, then embeddings", m.Capabilities)
	}
	want := outdir.TranscriptionCounts{Distinct: 3, Transcribed: 3, Requests: 3, Transcripts: 1, SeamsMatched: 2}
	if m.Counts.Windows != 3 || m.Counts.Transcription == nil || *m.Counts.Transcription != want {
		t.Errorf("counts = %+v, transcription %+v, want %+v", m.Counts, m.Counts.Transcription, want)
	}

	sum := hexSum([]byte(joinedText))
	file := path.Join("texts", sum[:2], sum+".txt")
	if len(m.Transcripts) != 1 {
		t.Fatalf("transcripts = %+v, want one", m.Transcripts)
	}
	tr := m.Transcripts[0]
	wantSeams := []outdir.Seam{{Matched: true, Matches: 2, Overlap: 2, Position: 5}, {Matched: true, Matches: 2, Overlap: 2, Position: 9}}
	if tr.Input != talkWAV || tr.Part == nil || tr.Part.Name != "talk.wav" || tr.Stage != "audio" ||
		!slices.Equal(tr.Windows, windowRefs) || len(tr.Failed) != 0 || tr.File != file || tr.SHA256 != sum ||
		!slices.Equal(tr.Seams, wantSeams) {
		t.Errorf("transcript = %+v, want talk.wav's windows joined into %s, seams %+v", tr, file, wantSeams)
	}
	if got := string(readFile(t, filepath.Join(out, filepath.FromSlash(file)))); got != joinedText {
		t.Errorf("stored transcript = %q, want %q", got, joinedText)
	}

	tl := byKind(lines, outdir.SourceTranscript)
	if len(tl) != 1 {
		t.Fatalf("transcript lines = %+v, want one", tl)
	}
	l := tl[0]
	wantOutput := outdir.Output{Kind: "audio", Name: "window 0", MediaType: "audio/wav", Ref: windowRefs[0],
		Meta: map[string]string{"window": "0", "count": "3", "start": "0", "end": "2.5", "cut": "silence", "samples": "40000", "seam_after": "2-2.5"}}
	if l.Text != joinedText || l.Input != talkWAV || l.Part == nil || l.Part.Name != "talk.wav" || l.Part.Outcome != "prepared" ||
		l.Output == nil || !reflect.DeepEqual(*l.Output, wantOutput) || l.File == nil || l.File.Name != "talk.wav" || l.ContentType == nil ||
		l.Source != (outdir.Source{Kind: "transcript", File: file, SHA256: sum}) {
		t.Errorf("transcript line = %+v, want the joined text with talk.wav's input and part, and its first window", l)
	}
	if l.Vector != outdir.EncodeVector(docinfertest.Vector(joinedText, docinfertest.DefaultDims)) || l.Error != "" {
		t.Errorf("transcript line vector %q, error %q, want the endpoint's", l.Vector, l.Error)
	}
	wantLine(t, stdout, regexp.QuoteMeta("transcription: windows 3, distinct 3: held 0, transcribed 3, failed 0; requests 3; transcripts 1: seams matched 2, unmatched 0"))
	if strings.Contains(stdout, "warning") {
		t.Errorf("stdout warns, want no warning:\n%s", stdout)
	}
}

// The configured prompt and language are what is sent, and a changed language sends every
// window again.
func TestRun_TranscriptionSettings(t *testing.T) {
	clean(t)
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA})
	cfg := configDir(t, map[string]string{"config.json": transcriptionConfig(e.URL(), `"prompt": "Transcribe the talk.", "language": "en"`)})
	out := t.TempDir()

	run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)
	t.Setenv("DOCINFER_TRANSCRIPTION_LANGUAGE", "de")
	run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)

	trs := e.Transcriptions()
	if len(trs) != 6 {
		t.Fatalf("transcription requests = %d, want six, the second run's three under the new language", len(trs))
	}
	for i, r := range trs {
		lang := "en"
		if i >= 3 {
			lang = "de"
		}
		if r.Prompt != "Transcribe the talk." || r.Language != lang {
			t.Errorf("request %d: prompt %q, language %q, want \"Transcribe the talk.\" and %s", i, r.Prompt, r.Language, lang)
		}
	}
	m, _ := result(t, out)
	if s := m.Capabilities[0].Settings; s["prompt"] != "Transcribe the talk." || s["language"] != "de" {
		t.Errorf("transcription settings = %v, want the prompt and de", s)
	}
}

// Windows are joined in plan order, by their meta's index, whatever order the record lists them
// in.
func TestRun_TranscriptionJoinsByIndex(t *testing.T) {
	clean(t)
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA, Transcribe: seamTexts(t, -1, 0)})
	dir := copyDocproc(t)
	editRecord(t, dir, func(lines [][]byte) [][]byte {
		var at []int
		for i, l := range lines {
			if isWindow(l) {
				at = append(at, i)
			}
		}
		// The windows' lines, 0, 1, 2, listed 2, 0, 1.
		w := [][]byte{lines[at[0]], lines[at[1]], lines[at[2]]}
		lines[at[0]], lines[at[1]], lines[at[2]] = w[2], w[0], w[1]
		return lines
	})
	cfg := configDir(t, map[string]string{"config.json": transcriptionConfig(e.URL(), "")})
	out := t.TempDir()

	run(t, process.ExitOK, "run", dir, "--config", cfg, "--out", out)

	m, lines := result(t, out)
	if len(m.Transcripts) != 1 || !slices.Equal(m.Transcripts[0].Windows, windowRefs) || m.Counts.Transcription.SeamsMatched != 2 {
		t.Errorf("transcripts = %+v, want the windows in plan order, each seam matched", m.Transcripts)
	}
	tl := byKind(lines, outdir.SourceTranscript)
	if len(tl) != 1 || tl[0].Text != joinedText || tl[0].Output.Ref != windowRefs[0] {
		t.Errorf("transcript lines = %+v, want %q, its output window 0", tl, joinedText)
	}
}

// A window the endpoint refuses joins as an empty transcript: its seams are unmatched and the
// windows either side concatenated; the run records it and warns of it, not fatal. A 4xx
// refusal is held, so a re-run sends no transcription request; one with any other status is
// not, so a re-run sends that window again.
func TestRun_TranscriptionFailedWindow(t *testing.T) {
	for _, tc := range []struct {
		status int
		held   bool
	}{{400, true}, {503, false}} {
		t.Run(fmt.Sprint(tc.status), func(t *testing.T) {
			clean(t)
			e := docinfertest.Serve(t, docinfertest.Fake{Models: setA, Transcribe: seamTexts(t, 1, tc.status)})
			cfg := configDir(t, map[string]string{"config.json": transcriptionConfig(e.URL(), "")})
			out := t.TempDir()

			stdout, _ := run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)

			m, lines := result(t, out)
			want := outdir.TranscriptionCounts{Distinct: 3, Transcribed: 2, Failed: 1, Requests: 3, Transcripts: 1, SeamsUnmatched: 2}
			if tc := m.Counts.Transcription; tc == nil || *tc != want {
				t.Errorf("transcription counts = %+v, want %+v", tc, want)
			}
			text := windowTexts[0] + " " + windowTexts[2]
			wantSeams := []outdir.Seam{{Position: 6}, {Position: 6}}
			if len(m.Transcripts) != 1 || !slices.Equal(m.Transcripts[0].Failed, []int{1}) || !slices.Equal(m.Transcripts[0].Seams, wantSeams) {
				t.Errorf("transcripts = %+v, want window 1 failed, both seams unmatched", m.Transcripts)
			}
			if tl := byKind(lines, outdir.SourceTranscript); len(tl) != 1 || tl[0].Text != text {
				t.Errorf("transcript lines = %+v, want %q", tl, text)
			}
			wantLine(t, stdout, regexp.QuoteMeta(fmt.Sprintf("warning: transcription: failed window %s: %d ", windowRefs[1], tc.status))+".*window refused.*")

			run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)
			again, _ := result(t, out)
			want = outdir.TranscriptionCounts{Distinct: 3, Held: 3, Failed: 1, Transcripts: 1, SeamsUnmatched: 2}
			if !tc.held {
				want = outdir.TranscriptionCounts{Distinct: 3, Held: 2, Failed: 1, Requests: 1, Transcripts: 1, SeamsUnmatched: 2}
			}
			if n := len(e.Transcriptions()); again.Counts.Transcription == nil || *again.Counts.Transcription != want || (n == 3) != tc.held {
				t.Errorf("re-run: %d transcription requests in all, counts %+v; want %+v", n, again.Counts.Transcription, want)
			}
		})
	}
}

// A second run over the same docproc run and configuration sends no transcription request and
// no embeddings request, and writes the same chunk lines and transcripts.
func TestRun_TranscriptionRerunSendsNothing(t *testing.T) {
	clean(t)
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA, Transcribe: seamTexts(t, -1, 0)})
	cfg := configDir(t, map[string]string{"config.json": transcriptionConfig(e.URL(), "")})
	out := t.TempDir()

	run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)
	first, _ := result(t, out)
	firstLines := readFile(t, filepath.Join(out, first.Chunks))
	transcriptions, embeds := len(e.Transcriptions()), len(e.Embeddings())
	stdout, _ := run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)

	if n, k := len(e.Transcriptions())-transcriptions, len(e.Embeddings())-embeds; n != 0 || k != 0 {
		t.Errorf("second run sent %d transcription and %d embeddings requests, want none", n, k)
	}
	second, _ := result(t, out)
	if !bytes.Equal(readFile(t, filepath.Join(out, second.Chunks)), firstLines) || !reflect.DeepEqual(second.Transcripts, first.Transcripts) {
		t.Errorf("second run's chunk lines or transcripts differ from the first's")
	}
	want := outdir.TranscriptionCounts{Distinct: 3, Held: 3, Transcripts: 1, SeamsMatched: 2}
	if tc := second.Counts.Transcription; tc == nil || *tc != want {
		t.Errorf("second run's transcription counts = %+v, want %+v", tc, want)
	}
	wantLine(t, stdout, regexp.QuoteMeta("transcription: windows 3, distinct 3: held 3, transcribed 0, failed 0; requests 0; transcripts 1: seams matched 2, unmatched 0"))
}

// A changed chunk bound cuts and embeds the transcript again and sends no transcription request:
// the windows' results are held apart from the chunk settings.
func TestRun_TranscriptionChunkSizeChange(t *testing.T) {
	clean(t)
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA, Transcribe: seamTexts(t, -1, 0)})
	cfg := configDir(t, map[string]string{"config.json": transcriptionConfig(e.URL(), "")})
	out := t.TempDir()

	run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)
	transcriptions := len(e.Transcriptions())
	t.Setenv("DOCINFER_EMBEDDINGS_CHUNK_SIZE", "16")
	run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)

	if n := len(e.Transcriptions()) - transcriptions; n != 0 {
		t.Errorf("second run sent %d transcription requests, want none", n)
	}
	m, lines := result(t, out)
	tl := byKind(lines, outdir.SourceTranscript)
	if len(tl) < 2 {
		t.Errorf("transcript lines = %d, want the transcript cut into several", len(tl))
	}
	for _, l := range tl {
		if len([]rune(l.Text)) > 16 || l.Vector == "" || joinedText[l.Span.Start:l.Span.End] != l.Text {
			t.Errorf("transcript chunk %q: want at most 16 runes of the transcript, embedded", l.Text)
		}
	}
	if tc := m.Counts.Transcription; tc == nil || tc.Held != 3 || tc.Requests != 0 {
		t.Errorf("transcription counts = %+v, want every window held", tc)
	}
}

// With transcription unset, the windows are skipped and counted: no transcription request, no
// transcript line, and the manifest lists transcription as skipped.
func TestRun_TranscriptionUnset(t *testing.T) {
	clean(t)
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA})
	cfg := configDir(t, map[string]string{"config.json": embedConfig(e.URL(), "")})
	out := t.TempDir()

	stdout, _ := run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)

	if trs := e.Transcriptions(); len(trs) != 0 {
		t.Errorf("transcription requests = %d, want none", len(trs))
	}
	m, lines := result(t, out)
	if tl := byKind(lines, outdir.SourceTranscript); len(tl) != 0 {
		t.Errorf("transcript lines = %+v, want none", tl)
	}
	if !slices.Contains(m.Skipped, outdir.Skipped{Name: "transcription", Reason: "not configured"}) ||
		m.Counts.Windows != 3 || m.Counts.Transcription != nil || m.Transcripts != nil {
		t.Errorf("manifest skipped %+v, counts %+v, transcripts %+v; want transcription skipped, its windows counted", m.Skipped, m.Counts, m.Transcripts)
	}
	wantLine(t, stdout, regexp.QuoteMeta("transcription: not configured; windows 3 skipped"))
}

// Each distinct window is sent once, whatever parts' output lines name it, and each part's
// windows are joined into a transcript of its own; at most concurrency transcription requests
// are in flight.
func TestRun_TranscriptionDistinctAndConcurrency(t *testing.T) {
	clean(t)
	var (
		mu             sync.Mutex
		inFlight, most int
	)
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA, Transcribe: seamTexts(t, -1, 0), OnTranscribe: func(context.Context, int) {
		mu.Lock()
		inFlight++
		most = max(most, inFlight)
		mu.Unlock()
		time.Sleep(20 * time.Millisecond)
		mu.Lock()
		inFlight--
		mu.Unlock()
	}})
	// talk.wav's windows again, under a second input of the same content.
	const copyWAV = "/srv/inputs/copy/talk.wav"
	dir := copyDocproc(t)
	editRecord(t, dir, func(lines [][]byte) [][]byte {
		for _, l := range slices.Clone(lines) {
			if isWindow(l) {
				lines = append(lines, bytes.Replace(l, []byte(`"input":"`+talkWAV+`"`), []byte(`"input":"`+copyWAV+`"`), 1))
			}
		}
		return lines
	})
	cfg := configDir(t, map[string]string{"config.json": transcriptionConfig(e.URL(), `"concurrency": 2`)})
	out := t.TempDir()

	run(t, process.ExitOK, "run", dir, "--config", cfg, "--out", out)

	trs := e.Transcriptions()
	seen := map[string]int{}
	for _, r := range trs {
		seen[hexSum(r.File)]++
	}
	if len(trs) != 3 || len(seen) != 3 {
		t.Errorf("transcription requests = %d of %d windows, want three, each window once", len(trs), len(seen))
	}
	m, lines := result(t, out)
	want := outdir.TranscriptionCounts{Distinct: 3, Transcribed: 3, Requests: 3, Transcripts: 2, SeamsMatched: 4}
	if tc := m.Counts.Transcription; m.Counts.Windows != 6 || tc == nil || *tc != want {
		t.Errorf("counts = %+v, transcription %+v, want 6 window lines and %+v", m.Counts, tc, want)
	}
	tl := byKind(lines, outdir.SourceTranscript)
	if len(tl) != 2 || tl[0].Input != talkWAV || tl[1].Input != copyWAV || tl[0].Text != joinedText || tl[1].Text != joinedText ||
		tl[0].Vector != tl[1].Vector {
		t.Errorf("transcript lines = %+v, want one per input, the same text and vector", tl)
	}
	if most > 2 {
		t.Errorf("at most %d transcription requests in flight, want 2", most)
	}
}

// A run cancelled during transcription stops: its manifest records the error and it writes no
// chunk lines; the windows transcribed before it are held, so the next run sends only the rest.
func TestRun_TranscriptionResumesAfterCancel(t *testing.T) {
	clean(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA, Transcribe: seamTexts(t, -1, 0), OnTranscribe: func(rctx context.Context, n int) {
		if n == 2 {
			cancel()
			select { // the run's cancel closes the request's connection
			case <-rctx.Done():
			case <-time.After(10 * time.Second):
			}
		}
	}})
	cfg := configDir(t, map[string]string{"config.json": transcriptionConfig(e.URL(), `"concurrency": 1`)})
	out := t.TempDir()
	a, _, stderr, _ := newApp(t)

	if code := a.Run(ctx, []string{"run", docproc, "--config", cfg, "--out", out}); code == process.ExitOK {
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
	if !strings.Contains(stopped.Error, "transcription: ") || stopped.Chunks != "" || len(e.Embeddings()) != 0 {
		t.Errorf("cancelled run's manifest = %+v, %d embeddings requests; want transcription's error, no chunks, nothing embedded", stopped, len(e.Embeddings()))
	}

	run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)

	m, lines := result(t, out)
	want := outdir.TranscriptionCounts{Distinct: 3, Held: 1, Transcribed: 2, Requests: 2, Transcripts: 1, SeamsMatched: 2}
	if n := len(e.Transcriptions()); n != 2+2 || m.Counts.Transcription == nil || *m.Counts.Transcription != want {
		t.Errorf("%d transcription requests in all, re-run counts %+v; want one held and the other two sent", n, m.Counts.Transcription)
	}
	if tl := byKind(lines, outdir.SourceTranscript); len(tl) != 1 || tl[0].Text != joinedText {
		t.Errorf("transcript lines = %+v, want %q", tl, joinedText)
	}
}
