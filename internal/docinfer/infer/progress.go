package infer

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/outdir"
	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/source"
)

// progress writes a run's per-input terminal lines: one for each input of the docproc run's
// record, as its work ends. The run's capabilities run one after another over every input, vision,
// then transcription, then embeddings, so an input's work ends with its last chunk's embedding:
// an input none of whose chunks is sent, every one held, or with no chunk at all, ends as the
// embeddings begin, and every other as the request holding its last chunk completes.
//
// Each line names the input's path, its image and audio window output lines, its chunk lines,
// its failures, its output lines whose image or window was refused and its chunk lines whose
// chunk was rejected, and its time: how long after the run began its work ended, since an input's
// work is spread over the run's phases, each over every input, and has no start of its own.
//
// It writes the run's progress lines too: as each capability's phase begins, how many distinct
// images, windows, or chunks it has and how many of them it sends and how many earlier runs hold;
// and as each request completes, its line. A held result sends no request and has no line, so a
// re-run's terminal stays short.
type progress struct {
	w     io.Writer
	start time.Time
	order []string // the inputs, in the order of their first line in the record
	tally map[string]*inputTally
	ended []string // the inputs whose work ended with the request whose results are being taken
}

// inputTally is one input's line, as its work goes: what it counts, and the chunks, by SHA-256,
// still to be embedded, each with how many of the input's chunk lines it is.
type inputTally struct {
	images, windows, chunks, failures int
	waiting                           map[string]int
	ended                             bool
}

// newProgress returns the progress of a run that began at start over run, whose texts' first
// occurrences, before vision or transcription made any text, are occs, writing its lines to w.
func newProgress(w io.Writer, start time.Time, run *source.Run, occs []occurrence) *progress {
	p := &progress{w: w, start: start, tally: map[string]*inputTally{}}
	for _, ev := range run.Events {
		if p.tally[ev.Input] == nil {
			p.tally[ev.Input] = &inputTally{waiting: map[string]int{}}
			p.order = append(p.order, ev.Input)
		}
	}
	for _, o := range occs {
		t := p.of(o.input)
		switch {
		case o.image != "":
			t.images++
		case o.audio != "":
			t.windows++
		}
	}
	return p
}

// of returns input's tally, adding one for an input the record's lines didn't name.
func (p *progress) of(input string) *inputTally {
	t := p.tally[input]
	if t == nil {
		t = &inputTally{waiting: map[string]int{}}
		p.tally[input] = t
		p.order = append(p.order, input)
	}
	return t
}

// failed counts each of failures against its input.
func (p *progress) failed(failures []outdir.Failure) {
	for _, f := range failures {
		p.of(f.Input).failures++
	}
}

// chunk counts one of input's chunk lines, its chunk's SHA-256 s: one to be embedded when queued,
// or, when not, one whose result is held, a failure when rejected says the endpoint rejected it.
func (p *progress) chunk(input, s string, queued, rejected bool) {
	t := p.of(input)
	t.chunks++
	switch {
	case queued:
		t.waiting[s]++
	case rejected:
		t.failures++
	}
}

// waiters returns, for each chunk queued, the inputs waiting for it, each once, in the order of
// their first line.
func (p *progress) waiters() map[string][]string {
	w := map[string][]string{}
	for _, in := range p.order {
		for s := range p.tally[in].waiting {
			w[s] = append(w[s], in)
		}
	}
	return w
}

// embedded marks the chunk s embedded, or rejected, for each input waiting for it, inputs, and
// keeps each whose work that ends for flush, so the line of the request that carried s comes
// before the input lines it ends.
func (p *progress) embedded(inputs []string, s string, rejected bool) {
	for _, in := range inputs {
		t := p.tally[in]
		if rejected {
			t.failures += t.waiting[s]
		}
		delete(t.waiting, s)
		if len(t.waiting) == 0 {
			p.ended = append(p.ended, in)
		}
	}
}

// flush writes the line of each input embedded kept, in the order its work ended.
func (p *progress) flush() error {
	ended := p.ended
	p.ended = nil
	for _, in := range ended {
		if err := p.end(in); err != nil {
			return err
		}
	}
	return nil
}

// phase writes the line that begins capability's phase: its distinct items, what, n of them, the
// number it sends, send, and the number earlier runs hold, held. A phase with no item writes
// none.
func (p *progress) phase(capability, what string, n, send, held int) error {
	if n == 0 {
		return nil
	}
	sending := fmt.Sprintf("to send %d, held %d", send, held)
	if send == 0 {
		sending = fmt.Sprintf("held %d, nothing to send", held)
	}
	_, err := fmt.Fprintf(p.w, "%s: distinct %s %d: %s\n", capability, what, n, sending)
	return err
}

// request writes the line of one of capability's requests, as it completes: done of the phase's
// total items to send, now with an answer, the request's; what names the request's items, an
// input's path and, where it helps, an output's ref; and its outcome, and how long it took.
func (p *progress) request(capability string, done, total int, what []string, outcome string, took time.Duration) error {
	_, err := fmt.Fprintf(p.w, "%s %d/%d  %s  %s  %s\n",
		capability, done, total, strings.Join(what, "  "), outcome, took.Round(time.Millisecond))
	return err
}

// idle writes the line of each input, in the order of its first line, that waits for no chunk.
func (p *progress) idle() error {
	for _, in := range p.order {
		if len(p.tally[in].waiting) == 0 {
			if err := p.end(in); err != nil {
				return err
			}
		}
	}
	return nil
}

// end writes input's line, once.
func (p *progress) end(input string) error {
	t := p.tally[input]
	if t.ended {
		return nil
	}
	t.ended = true
	_, err := fmt.Fprintf(p.w, "%s  images %d  windows %d  chunks %d  failures %d  %s\n",
		input, t.images, t.windows, t.chunks, t.failures, time.Since(p.start).Round(time.Millisecond))
	return err
}
