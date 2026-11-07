package report

import (
	"bytes"
	"cmp"
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"maps"
	"mime"
	"slices"
	"strings"
	"time"

	"github.com/JaimeStill/spike-ai-pipeline/internal/rundir"
	"github.com/JaimeStill/spike-ai-pipeline/internal/schema"
)

// Name is the report's file name in its run's directory.
const Name = "report.html"

// toRunDir is the way from the report, at runs/<run id>/report.html, back to the run directory,
// which every path the record writes is relative to.
const toRunDir = "../../"

// noRoute is how the report names the route of a part that has none, as the terminal does.
const noRoute = "-"

// errorOutcome is the outcome the report gives an input whose dispatch stopped with an error
// before its own event, as the terminal does.
const errorOutcome = "error"

//go:embed report.html.tmpl
var source string

// page is the report's template, parsed once.
var page = template.Must(template.New("report").Funcs(template.FuncMap{
	"time":   func(t schema.Time) string { return t.String() },
	"took":   took,
	"bytes":  byteSize,
	"short":  short,
	"href":   func(p string) string { return toRunDir + p },
	"n":      deref,
	"links":  func(os []output) []output { return filter(os, false) },
	"images": func(os []output) []output { return filter(os, true) },
}).Parse(source))

// Write renders the report of the run id in the run directory at root, the latest run when id is
// empty, into runs/<run id>/report.html, replacing any report there, and returns that path
// relative to root.
func Write(root, id string) (string, error) {
	if id == "" {
		latest, err := rundir.Latest(root)
		if err != nil {
			return "", err
		}
		id = latest
	}
	m, evs, err := rundir.Load(root, id)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := Render(&buf, m, evs); err != nil {
		return "", err
	}
	return rundir.WriteRunFile(root, id, Name, func(w io.Writer) error {
		_, err := w.Write(buf.Bytes())
		return err
	})
}

// Render writes the report of the run m describes, whose record is evs, to w. It reads nothing
// but its arguments, the clock included, so the same manifest and record always render the same
// bytes.
func Render(w io.Writer, m schema.Manifest, evs []schema.Event) error {
	return page.Execute(w, build(m, evs))
}

// view is what the template renders.
type view struct {
	M       schema.Manifest
	Took    string              // how long the run ran; empty when it never ended
	Unwired []schema.StageState // the missing and excluded routes, in the stage table's order
	Summary []tally
	Total   tally
	Inputs  []*input
}

// tally is one row of the summary: how many inputs, and how many parts, the inputs among them,
// came to a route and an outcome.
type tally struct {
	Route   string
	Outcome string
	Inputs  int
	Parts   int
}

// input is one input as the report shows it: its record's lines, as a part tree.
type input struct {
	Path    string
	Route   string
	Outcome string
	File    *schema.File
	Time    *schema.Span
	Error   string  // the error that stopped its dispatch before its own event; optional
	Roots   []*node // its own part, and any part whose parent never had its own event
	Loose   []output
}

// node is one part's own event, the outputs its stage yielded, and the parts found in it.
type node struct {
	Ev       schema.Event
	Outputs  []output
	Children []*node
}

// output is one output line as the report shows it.
type output struct {
	schema.Output
	Seen  bool   // the index recorded it earlier, and its Ref names that copy
	Href  string // its stored file, relative to the report
	Image bool   // its media type is an image's, so it shows inline
}

// build returns the view of the run m describes, whose record is evs. Inputs sort by path, and
// each input's parts keep the record's order, which is the order its stages found them.
func build(m schema.Manifest, evs []schema.Event) view {
	v := view{M: m}
	if !m.End.IsZero() {
		v.Took = took(schema.Span{Start: m.Start, End: m.End})
	}
	order := map[string]int{} // a route's rank, in the stage table's order
	for i, s := range m.Stages {
		order[s.Route] = i
		if s.State != "wired" {
			v.Unwired = append(v.Unwired, s)
		}
	}

	byInput := map[string][]schema.Event{}
	for _, ev := range evs {
		byInput[ev.Input] = append(byInput[ev.Input], ev)
	}
	counts := map[[2]string]*tally{}
	count := func(route, outcome string, isInput bool) {
		k := [2]string{route, outcome}
		t := counts[k]
		if t == nil {
			t = &tally{Route: route, Outcome: outcome}
			counts[k] = t
		}
		t.Parts++
		v.Total.Parts++
		if isInput {
			t.Inputs++
			v.Total.Inputs++
		}
	}
	for _, path := range slices.Sorted(maps.Keys(byInput)) {
		in := tree(path, byInput[path])
		v.Inputs = append(v.Inputs, in)
		for _, ev := range byInput[path] {
			switch ev.Event {
			case schema.PartEvent:
				count(routeOf(ev.Part), ev.Outcome, ev.Part.Depth == 0)
			case schema.ErrorEvent:
				count(noRoute, errorOutcome, true)
			}
		}
	}

	rank := func(route string) int {
		if i, ok := order[route]; ok {
			return i
		}
		return len(order)
	}
	for _, t := range counts {
		v.Summary = append(v.Summary, *t)
	}
	slices.SortFunc(v.Summary, func(a, b tally) int {
		return cmp.Or(cmp.Compare(rank(a.Route), rank(b.Route)), strings.Compare(a.Route, b.Route),
			strings.Compare(a.Outcome, b.Outcome))
	})
	return v
}

// tree returns the input at path as its record's lines evs, in their order, make it. A part's
// outputs and the parts found in it come before its own event, so each part's own event takes
// the outputs waiting for it, by its SHA-256, parent, and depth, and the nodes waiting under it,
// those at one depth more whose parent is its SHA-256. Whatever still waits once the lines run
// out, which only a dispatch an error stopped leaves, stays at the top of the tree.
func tree(path string, evs []schema.Event) *input {
	in := &input{Path: path, Route: noRoute}
	type key struct {
		sha, parent string
		depth       int
	}
	keyOf := func(p *schema.Part) key { return key{p.SHA256, p.Parent, p.Depth} }
	var keys []key // the waiting outputs' keys, in the order they arrived
	outputs := map[key][]output{}
	var waiting []*node
	for _, ev := range evs {
		switch ev.Event {
		case schema.OutputEvent:
			if ev.Output == nil || ev.Part == nil {
				continue
			}
			k := keyOf(ev.Part)
			if _, ok := outputs[k]; !ok {
				keys = append(keys, k)
			}
			outputs[k] = append(outputs[k], outputOf(*ev.Output, ev.Outcome))
		case schema.PartEvent:
			if ev.Part == nil {
				continue
			}
			n := &node{Ev: ev}
			k := keyOf(ev.Part)
			n.Outputs = outputs[k]
			delete(outputs, k)
			if ev.Part.SHA256 != "" {
				waiting = slices.DeleteFunc(waiting, func(c *node) bool {
					if c.Ev.Part.Parent == ev.Part.SHA256 && c.Ev.Part.Depth == ev.Part.Depth+1 {
						n.Children = append(n.Children, c)
						return true
					}
					return false
				})
			}
			waiting = append(waiting, n)
			if ev.Part.Depth == 0 {
				in.Route, in.Outcome, in.File, in.Time = routeOf(ev.Part), ev.Outcome, ev.File, ev.Time
			}
		case schema.ErrorEvent:
			in.Outcome, in.File, in.Time, in.Error = errorOutcome, ev.File, ev.Time, ev.Error
		}
	}
	in.Roots = waiting
	for _, k := range keys {
		in.Loose = append(in.Loose, outputs[k]...)
	}
	return in
}

// outputOf returns o, an output line's output whose outcome is outcome, as the report shows it.
func outputOf(o schema.Output, outcome string) output {
	mt, _, _ := mime.ParseMediaType(o.MediaType)
	return output{
		Output: o,
		Seen:   outcome == "seen",
		Href:   toRunDir + o.Ref,
		Image:  strings.HasPrefix(mt, "image/"),
	}
}

// routeOf returns p's route as the report names it, "-" for none.
func routeOf(p *schema.Part) string {
	if p == nil || p.ContentType.Route == "" {
		return noRoute
	}
	return p.ContentType.Route
}

// took returns how long span ran, to the millisecond.
func took(span schema.Span) string {
	return time.Time(span.End).Sub(time.Time(span.Start)).Round(time.Millisecond).String()
}

// byteSize returns n bytes in binary units, to one decimal place above a KiB.
func byteSize(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	f, unit := float64(n)/1024, "KiB"
	for _, u := range []string{"MiB", "GiB", "TiB"} {
		if f < 1024 {
			break
		}
		f, unit = f/1024, u
	}
	return fmt.Sprintf("%.1f %s", f, unit)
}

// short returns the first twelve characters of s, a SHA-256 or a hash, as the report shows it
// beside the whole in a title.
func short(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

// deref returns *p, or 0 for nil.
func deref(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// filter returns the outputs of os that show inline, when image is true, or that are linked,
// when it is false, in their order.
func filter(os []output, image bool) []output {
	var kept []output
	for _, o := range os {
		if o.Image == image {
			kept = append(kept, o)
		}
	}
	return kept
}
