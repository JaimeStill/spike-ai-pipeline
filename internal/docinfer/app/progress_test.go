package app_test

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/standards-lab/go-core/process"

	docinfertest "github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/apptest"
)

// progressLine matches one request's progress line: its capability, its count over the phase's
// total to send, and the rest, its names, outcome, and time.
var progressLine = regexp.MustCompile(`(?m)^(vision|transcription|embeddings) (\d+)/(\d+)  (.*)$`)

// progressLines returns out's request progress lines by capability, each its count, its total,
// and the rest of its line.
func progressLines(out string) map[string][][3]string {
	got := map[string][][3]string{}
	for _, m := range progressLine.FindAllStringSubmatch(out, -1) {
		got[m[1]] = append(got[m[1]], [3]string{m[2], m[3], m[4]})
	}
	return got
}

// A run prints, as each capability's phase begins, how many distinct items it sends and holds,
// and, as each request completes, its line: the capability, a running count reaching the
// phase's total to send, what it carried, its outcome, and its time. A re-run, every result held,
// says each phase has nothing to send and prints no request line.
func TestRun_ProgressLines(t *testing.T) {
	clean(t)
	e := docinfertest.Serve(t, docinfertest.Fake{Models: setA})
	cfg := configDir(t, map[string]string{"config.json": fmt.Sprintf(`{
  "endpoints": {"fake": {"base_url": %q, "format": "completions"}},
  "embeddings": {"endpoint": "fake", "model": %q, "batch": 4, "concurrency": 2},
  "vision": {"endpoint": "fake", "model": %q},
  "transcription": {"endpoint": "fake", "model": %q, "concurrency": 2}
}`, e.URL(), embeddingModel, visionModel, audioModel)})
	out := t.TempDir()

	stdout, _ := run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)

	m, _ := result(t, out)
	wantLine(t, stdout, regexp.QuoteMeta("vision: distinct images 1: to send 1, held 0"))
	wantLine(t, stdout, regexp.QuoteMeta("transcription: distinct windows 3: to send 3, held 0"))
	wantLine(t, stdout, regexp.QuoteMeta(fmt.Sprintf("embeddings: distinct chunks %d: to send %d, held 0", m.Counts.Chunks, m.Counts.Chunks)))
	wantLine(t, stdout, regexp.QuoteMeta("vision 1/1  "+reportPDF+"  "+pngRef+"  ok  ")+`\S+`)
	for _, ref := range windowRefs {
		wantLine(t, stdout, `transcription [123]/3  `+regexp.QuoteMeta(talkWAV+"  "+ref+"  ok  ")+`\S+`)
	}
	lines := progressLines(stdout)
	wantRequests := map[string]int{"vision": 1, "transcription": 3, "embeddings": m.Counts.Requests}
	totals := map[string]int{"vision": 1, "transcription": 3, "embeddings": m.Counts.Chunks}
	for capability, n := range wantRequests {
		got := lines[capability]
		if len(got) != n {
			t.Errorf("%s progress lines = %d, want one a request, %d:\n%s", capability, len(got), n, stdout)
			continue
		}
		last := 0
		for _, l := range got {
			count, _ := strconv.Atoi(l[0])
			if count <= last || l[1] != strconv.Itoa(totals[capability]) {
				t.Errorf("%s progress line %q, want a rising count over %d", capability, l, totals[capability])
			}
			last = count
		}
		if last != totals[capability] {
			t.Errorf("%s progress count reaches %d, want %d", capability, last, totals[capability])
		}
	}
	if m.Counts.Requests < 2 {
		t.Fatalf("embeddings requests = %d, want several, so the test sees the count rise", m.Counts.Requests)
	}
	for _, l := range lines["embeddings"] {
		if !regexp.MustCompile(`^/srv/inputs/\S+  (inputs \d+  )?chunks [1-4]  ok  \S+$`).MatchString(l[2]) {
			t.Errorf("embeddings progress line ends %q, want an input, its inputs, its chunks, ok, and a time", l[2])
		}
	}
	// A request's line comes before the input lines whose work it ends: every input here waits
	// for a chunk, so none ends before the first request's line.
	first := regexp.MustCompile(`(?m)^embeddings \d+/`).FindStringIndex(stdout)
	input := regexp.MustCompile(`(?m)^/srv/inputs/\S+  images `).FindStringIndex(stdout)
	if first == nil || input == nil || input[0] < first[0] {
		t.Errorf("an input line precedes the first embeddings request line:\n%s", stdout)
	}

	again, _ := run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", out)

	if got := progressLines(again); len(got) != 0 {
		t.Errorf("re-run progress lines = %v, want none: every result is held", got)
	}
	wantLine(t, again, regexp.QuoteMeta("vision: distinct images 1: held 1, nothing to send"))
	wantLine(t, again, regexp.QuoteMeta("transcription: distinct windows 3: held 3, nothing to send"))
	wantLine(t, again, regexp.QuoteMeta(fmt.Sprintf("embeddings: distinct chunks %d: held %d, nothing to send", m.Counts.Chunks, m.Counts.Chunks)))
	wantLine(t, again, `/srv/inputs/report\.pdf  images 1  windows 0  chunks \d+  failures 0  \S+`)
}

// A refused image, a failed window, and a rejected chunk each print their outcome on their
// request's line.
func TestRun_ProgressOutcomes(t *testing.T) {
	clean(t)
	bad := func(in string) bool { return strings.Contains(in, "Revenue") }
	e := docinfertest.Serve(t, docinfertest.Fake{
		Models: setA, Reject: bad, Transcribe: seamTexts(t, 1, 400),
		Vision: func(docinfertest.VisionRequest) docinfertest.VisionReply {
			return docinfertest.VisionReply{Status: 400}
		},
	})
	cfg := configDir(t, map[string]string{"config.json": fullConfig(e.URL())})

	stdout, _ := run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", t.TempDir())

	wantLine(t, stdout, regexp.QuoteMeta("vision 1/1  "+reportPDF+"  "+pngRef+"  refused  ")+`\S+`)
	wantLine(t, stdout, `transcription [123]/3  `+regexp.QuoteMeta(talkWAV+"  "+windowRefs[1]+"  failed  ")+`\S+`)
	wantLine(t, stdout, `embeddings \d+/\d+  `+regexp.QuoteMeta(reportPDF+"  chunks 1  rejected  ")+`\S+`)
}
