package run

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/JaimeStill/spike-ai-pipeline-processing/dispatch"
	"github.com/JaimeStill/spike-ai-pipeline-processing/route"

	"github.com/JaimeStill/spike-ai-pipeline/internal/rundir"
	"github.com/JaimeStill/spike-ai-pipeline/internal/schema"
	"github.com/JaimeStill/spike-ai-pipeline/internal/stages"
)

// errorOutcome is the outcome the terminal and the summary give an input whose dispatch stopped
// with an error before its own event.
const errorOutcome = "error"

// execute runs inputs, which came from the path arguments args, through set's stages over the
// run directory out, workers at once on processing's pool, under a new run: it writes the run's
// manifest and its record, stores each output and quarantined file, and writes one line to w as
// each input's last event arrives, then the closing summary. Its working directory, under out,
// is removed once the dispatch is done.
//
// An error that stops one input's dispatch is that input's record line and terminal line, and
// the others go on. An error that stops the whole dispatch, the context ending, the index
// failing, or the run directory or the terminal refusing a write, is returned once the record is
// closed and the manifest names it.
func execute(ctx context.Context, w io.Writer, set stages.Set, out string, args []string, inputs []input, workers int) (err error) {
	dir, err := rundir.Open(out)
	if err != nil {
		return err
	}
	work, err := os.MkdirTemp(dir.Root(), ".work-")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(work)) }()
	d, err := dispatch.New(dispatch.Config{Stages: set.Stages, Index: dir.Index(), Dir: work})
	if err != nil {
		return err
	}

	m := schema.Manifest{Workers: workers, Tools: toolsOf(set), Stages: stagesOf(set)}
	for _, arg := range args {
		abs, err := filepath.Abs(arg)
		if err != nil {
			return err
		}
		m.Inputs = append(m.Inputs, abs)
	}
	run, err := dir.Begin(m, time.Now())
	if err != nil {
		return err
	}
	rec := &recorder{dir: dir, run: run, w: w, inputs: map[string]input{}, counts: map[string]int{}, tally: map[tallyKey]int{}}
	for _, in := range inputs {
		rec.inputs[in.path] = in
	}

	stopped := rec.dispatch(ctx, d, inputs, workers)
	ended := run.End(time.Now(), stopped)
	summary := rec.summary()
	return errors.Join(stopped, ended, summary)
}

// tallyKey is one cell of the closing summary: a route and an outcome.
type tallyKey struct {
	route   string
	outcome string
}

// recorder turns a run's events into its record, its stored outputs, and its terminal lines. It
// runs on the pool's loop alone, so it holds no lock.
type recorder struct {
	dir     *rundir.Dir
	run     *rundir.Run
	w       io.Writer
	inputs  map[string]input
	started sync.Map         // an input's path to when the pool took it, written by the pool's feeder
	counts  map[string]int   // outputs stored so far, by part SHA-256 and stage identity
	tally   map[tallyKey]int // inputs by route and outcome
}

// dispatch dispatches inputs on d's pool, recording each event, and returns the error that
// stopped the whole dispatch, if any.
func (r *recorder) dispatch(ctx context.Context, d *dispatch.Dispatcher, inputs []input, workers int) error {
	for pe, err := range d.Pool(ctx, r.timed(inputs), workers) {
		switch {
		case err != nil && pe.Input.Path == "":
			return err
		case err != nil:
			err = r.inputError(pe.Input.Path, err)
		case pe.Event.Output != nil:
			err = r.output(pe.Input.Path, pe.Event)
		default:
			err = r.part(pe.Input.Path, pe.Event)
		}
		// A record, store, or terminal that can't be written stops the run, and the loop's
		// return stops the pool.
		if err != nil {
			return err
		}
	}
	return nil
}

// timed yields inputs as dispatch inputs, storing when the pool asked for each, so an input's
// time runs from then to its last event; it counts any wait for a free worker.
func (r *recorder) timed(inputs []input) iter.Seq[dispatch.Input] {
	return func(yield func(dispatch.Input) bool) {
		for _, in := range inputs {
			r.started.Store(in.path, time.Now())
			if !yield(dispatch.Input{Path: in.path}) {
				return
			}
		}
	}
}

// output records ev, an output event of the input at path. An output prepared in this run is
// stored, and its Ref set to where, before the body returns, so processing's index records the
// Ref, and its inline Value cleared, so the index holds no second copy of what the store holds;
// a seen output's event carries the Ref the run that prepared it set, replayed by the index.
func (r *recorder) output(path string, ev dispatch.Event) error {
	o := ev.Output
	if ev.Outcome != dispatch.Seen {
		key := ev.Part.SHA256 + "\x00" + ev.Stage.String()
		r.counts[key]++
		ref, err := r.dir.Store(ev.Part, ev.Stage, r.counts[key], *o)
		if err != nil {
			return err
		}
		o.Ref, o.Value = ref, nil
	}
	part := schema.PartOf(ev.Part)
	return r.run.Write(schema.Event{
		Event: schema.OutputEvent, Input: r.inputs[path].absPath, Part: &part, Stage: stageOf(ev.Stage),
		Output: &schema.Output{
			Kind: string(o.Kind), Name: o.Name, MediaType: o.MediaType, Meta: o.Meta, Ref: o.Ref,
		},
		Outcome: string(ev.Outcome), Recorded: string(ev.Recorded),
	})
}

// part records ev, a part's own event, of the input at path. A quarantined part's file is kept
// in the run directory. The input's own event, its last, carries its file's metadata and its
// time, and is its terminal line.
func (r *recorder) part(path string, ev dispatch.Event) error {
	delete(r.counts, ev.Part.SHA256+"\x00"+ev.Stage.String())
	part := schema.PartOf(ev.Part)
	rec := schema.Event{
		Event: schema.PartEvent, Input: r.inputs[path].absPath, Part: &part, Stage: stageOf(ev.Stage),
		Outcome: string(ev.Outcome), Recorded: string(ev.Recorded), Reason: ev.Reason,
		Outputs: &ev.Outputs, Children: &ev.Children,
	}
	if ev.Err != nil {
		rec.Error = ev.Err.Error()
	}
	if ev.Outcome == dispatch.Quarantined && ev.Part.Path != "" {
		q, err := r.dir.Quarantine(ev.Part)
		if err != nil {
			return err
		}
		rec.Quarantine = &q
	}
	if ev.Part.Depth > 0 {
		return r.run.Write(rec)
	}

	// The input's own event: its last.
	rec.File = r.file(path)
	if mt, err := schema.Detect(path); err == nil {
		part.ContentType.Detected = mt
	}
	span := r.span(path)
	rec.Time = &span
	if err := r.run.Write(rec); err != nil {
		return err
	}
	outcome := string(ev.Outcome)
	if ev.Err != nil {
		outcome += ": " + ev.Err.Error()
	}
	r.tally[tallyKey{routeName(ev.Part.Route), string(ev.Outcome)}]++
	return r.line(fmt.Sprintf("%s  %s  %s  outputs %d  children %d  %s\n",
		path, ev.Part.Route, outcome, ev.Outputs, ev.Children, r.took(span)))
}

// inputError records stop, the error that stopped the dispatch of the input at path before its
// own event, as its record line and its terminal line.
func (r *recorder) inputError(path string, stop error) error {
	span := r.span(path)
	rec := schema.Event{Event: schema.ErrorEvent, Input: r.inputs[path].absPath, File: r.file(path), Error: stop.Error(), Time: &span}
	if err := r.run.Write(rec); err != nil {
		return err
	}
	r.tally[tallyKey{routeName(""), errorOutcome}]++
	return r.line(fmt.Sprintf("%s  -  %s: %v  %s\n", path, errorOutcome, stop, r.took(span)))
}

// file returns the metadata of the input at path, its argument absolute, or nil when it can't be
// read.
func (r *recorder) file(path string) *schema.File {
	in := r.inputs[path]
	f, err := schema.FileOf(path, in.arg, schema.Statx)
	if err != nil {
		return nil
	}
	f.Arg = in.absArg
	return &f
}

// span returns the input at path's time, from when the pool took it to now.
func (r *recorder) span(path string) schema.Span {
	end := time.Now()
	start := end
	if t, ok := r.started.Load(path); ok {
		start = t.(time.Time)
	}
	return schema.Span{Start: schema.Time(start), End: schema.Time(end)}
}

// took returns span's length, to the millisecond, as the terminal writes it.
func (r *recorder) took(span schema.Span) time.Duration {
	return time.Time(span.End).Sub(time.Time(span.Start)).Round(time.Millisecond)
}

// line writes s, one terminal line.
func (r *recorder) line(s string) error {
	_, err := io.WriteString(r.w, s)
	return err
}

// summary writes the closing summary to the terminal: how many inputs came to each route and
// outcome, in the stage table's route order, then the total and where the run's record is.
func (r *recorder) summary() error {
	keys := slices.SortedFunc(func(yield func(tallyKey) bool) {
		for k := range r.tally {
			if !yield(k) {
				return
			}
		}
	}, func(a, b tallyKey) int {
		if c := routeRank(a.route) - routeRank(b.route); c != 0 {
			return c
		}
		return strings.Compare(a.outcome, b.outcome)
	})
	tw := tabwriter.NewWriter(r.w, 0, 0, 2, ' ', 0)
	var err error
	row := func(cells ...string) {
		if err == nil {
			_, err = io.WriteString(tw, strings.Join(cells, "\t")+"\n")
		}
	}
	row()
	row("ROUTE", "OUTCOME", "INPUTS")
	total := 0
	for _, k := range keys {
		row(k.route, k.outcome, fmt.Sprint(r.tally[k]))
		total += r.tally[k]
	}
	row("total", "", fmt.Sprint(total))
	row()
	row("run " + r.run.ID() + "  " + r.run.Record())
	return errors.Join(err, tw.Flush())
}

// routeName returns rt as the terminal names it, "-" for none.
func routeName(rt route.Route) string {
	if rt == "" {
		return "-"
	}
	return string(rt)
}

// routeRank returns where name, a route as routeName gives it, sorts in the summary: in the
// stage table's order, and any other route, "-" among them, after.
func routeRank(name string) int {
	if i := slices.Index(stages.Routes(), route.Route(name)); i >= 0 {
		return i
	}
	return len(stages.Routes())
}

// stageOf returns id as the record carries it, nil for no stage.
func stageOf(id dispatch.Identity) *schema.Stage {
	if id == (dispatch.Identity{}) {
		return nil
	}
	return &schema.Stage{Name: id.Name, Version: id.Version, Config: id.Config}
}

// toolsOf returns the tools set tried, as the manifest lists them.
func toolsOf(set stages.Set) []schema.Tool {
	tools := []schema.Tool{}
	for _, t := range set.Tools {
		st := schema.Tool{Program: t.Program, Minimum: t.Minimum, From: t.From, Version: t.Version}
		if t.Err != nil {
			st.Error = t.Err.Error()
		}
		tools = append(tools, st)
	}
	return tools
}

// stagesOf returns set's stage rows, as the manifest lists them.
func stagesOf(set stages.Set) []schema.StageState {
	rows := []schema.StageState{}
	for _, row := range set.Rows {
		rows = append(rows, schema.StageState{
			Route: string(row.Route), State: string(row.State), Stage: stageOf(row.Stage),
			Tools: row.Tools, Reason: row.Reason,
		})
	}
	return rows
}
