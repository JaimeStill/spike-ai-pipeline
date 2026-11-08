package report

import (
	"bytes"
	_ "embed"
	"html/template"
	"io"
	"maps"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/outdir"
	"github.com/JaimeStill/spike-ai-pipeline/internal/rundir"
	"github.com/JaimeStill/spike-ai-pipeline/internal/schema"
)

// name is the report's file name in its run's directory.
const name = "report.html"

// toRoot is the way from the report, at runs/<run id>/report.html, back to docinfer's directory,
// which every path docinfer records of a file it wrote is relative to.
const toRoot = "../../"

// textRunes is how many runes of a vision text or a transcript the page shows, before its
// ellipsis and the link to the whole stored text: enough for a page's or a short recording's
// text, and few enough that a run over thousands of images makes a page a browser opens.
const textRunes = 1000

//go:embed report.html.tmpl
var source string

// page is the report's template, parsed once.
var page = template.Must(template.New("report").Funcs(template.FuncMap{
	"time": func(t schema.Time) string { return t.String() },
}).Parse(source))

// Write renders the report of the run id in docinfer's directory at root, the latest run when id
// is empty, into runs/<run id>/report.html, replacing any report there, and returns that path
// relative to root. It reads the run's manifest, its chunk record when it has one, and the texts
// the page shows, all in root, and nothing of the docproc run, which the page links to.
func Write(root, id string) (string, error) {
	dir, err := outdir.OpenExisting(root)
	if err != nil {
		return "", err
	}
	if id == "" {
		if id, err = rundir.Latest(dir.Root()); err != nil {
			return "", err
		}
	}
	m, err := dir.ReadManifest(id)
	if err != nil {
		return "", err
	}
	var lines []outdir.Line
	if m.Chunks != "" {
		if lines, err = dir.ReadChunks(id); err != nil {
			return "", err
		}
	}
	reportDir := filepath.Join(dir.Root(), filepath.FromSlash(outdir.RunPath(id, "")))
	var buf bytes.Buffer
	if err := page.Execute(&buf, build(dir, m, lines, docprocLink(reportDir, m.Docproc.Dir))); err != nil {
		return "", err
	}
	return rundir.WriteRunFile(dir.Root(), id, name, func(w io.Writer) error {
		_, err := w.Write(buf.Bytes())
		return err
	})
}

// docprocLink returns the way from the report's directory, reportDir, to the docproc run
// directory, docproc, as a relative URL path ending in a slash, each element escaped: the page's
// images and windows are linked through it, so it holds no copy of them. Where no relative path
// reaches it, it is docproc's file URL.
func docprocLink(reportDir, docproc string) string {
	rel, err := filepath.Rel(reportDir, docproc)
	if err != nil {
		return (&url.URL{Scheme: "file", Path: filepath.ToSlash(docproc) + "/"}).String()
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/") + "/"
}

// view is what the template renders.
type view struct {
	M    outdir.Manifest
	Took string // how long the run ran; empty when it never ended
	// Docproc is the way from the report to the docproc run directory, as docprocLink gives it.
	Docproc  string
	Inputs   []*input
	Refused  []outdir.Failure // the image output lines vision refused
	Failed   []outdir.Failure // the audio window output lines transcription refused
	Rejected []rejected       // the chunk lines whose chunk the endpoint rejected
}

// input is one input as the page shows it: its sources, in the chunk record's order, then its
// transcripts that cut to no chunk, then its refused outputs, and its totals.
type input struct {
	Path     string
	Sources  []*src
	Chunks   int
	Failures int
}

// src is one text of an input as the page shows it: one occurrence of a source text, with the
// output it was made from, its text when docinfer stored it, and its chunk lines; or an output a
// capability refused, with why.
type src struct {
	Kind string // an outdir.Source kind, or "refused" for an output a capability refused
	// Output is the docproc output: the text output itself, the image the vision text is of, or
	// the refused image or window. Nil for metadata and transcripts.
	Output *outdir.Output
	Href   string // Output's stored file, relative to the report
	Image  bool   // Output is an image, shown inline
	// Text is the stored text's first textRunes runes, Cut whether there is more, and TextHref
	// the whole stored text, relative to the report; TextError is why it couldn't be read.
	Text       string
	Cut        bool
	TextHref   string
	TextError  string
	Incomplete string
	Transcript *outdir.Transcript
	Windows    []window // the transcript's windows, in plan order
	// Matched and Unmatched count the transcript's seams whose windows' words aligned, and
	// those concatenated whole.
	Matched, Unmatched int

	Chunks   int    // the occurrence's chunk lines
	Rejected int    // those whose chunk the endpoint rejected
	Error    string // why a capability refused the output
}

// window is one of a transcript's windows as the page links it.
type window struct {
	Index  int
	Href   string // relative to the report; empty for a window the record lacks
	Failed bool
}

// rejected is one chunk line whose chunk the endpoint rejected.
type rejected struct {
	Input  string
	Source string // the source's stored file or docproc ref, or "metadata"
	Index  int
	Error  string
}

// build returns the view of the run m describes, whose chunk record is lines, its texts read
// from dir and its docproc links under docproc. Inputs sort by path; each input's sources keep
// the chunk record's order, which is the docproc record's, then its transcripts with no chunk
// line, in the manifest's order, then its refused outputs, in the manifest's order. Nothing in
// it depends on when it is built.
func build(dir *outdir.Dir, m outdir.Manifest, lines []outdir.Line, docproc string) view {
	v := view{M: m, Docproc: docproc}
	if !m.End.IsZero() {
		v.Took = time.Time(m.End).Sub(time.Time(m.Start)).Round(time.Millisecond).String()
	}
	inputs := map[string]*input{}
	of := func(path string) *input {
		in := inputs[path]
		if in == nil {
			in = &input{Path: path}
			inputs[path] = in
		}
		return in
	}
	texts := map[string]*src{} // each stored text read once, by its file
	text := func(s *src, file, sum string) {
		s.TextHref = toRoot + file
		if t, ok := texts[file]; ok {
			s.Text, s.Cut, s.TextError = t.Text, t.Cut, t.TextError
			return
		}
		t, err := dir.ReadText(file, sum)
		if err != nil {
			s.TextError = err.Error()
		} else {
			s.Text, s.Cut = clip(t)
		}
		texts[file] = s
	}
	// Each manifest transcript, by its input and file, in the manifest's order, until a chunk
	// line's group claims it.
	type tkey struct{ input, file string }
	transcripts := map[tkey][]*outdir.Transcript{}
	for i := range m.Transcripts {
		t := &m.Transcripts[i]
		k := tkey{t.Input, t.File}
		transcripts[k] = append(transcripts[k], t)
	}

	// An occurrence's chunk lines are consecutive and count from 0, so each line of index 0
	// begins one.
	var cur *src
	for _, l := range lines {
		in := of(l.Input)
		if l.Index == 0 || cur == nil {
			cur = &src{Kind: l.Source.Kind, Incomplete: l.Source.Incomplete}
			switch l.Source.Kind {
			case outdir.SourceOutput:
				cur.Output = l.Output
				cur.Href = docproc + l.Source.Ref
			case outdir.SourceVision:
				cur.Output = l.Output
				cur.Image = true
				if l.Output != nil {
					cur.Href = docproc + l.Output.Ref
				}
				text(cur, l.Source.File, l.Source.SHA256)
			case outdir.SourceTranscript:
				k := tkey{l.Input, l.Source.File}
				if ts := transcripts[k]; len(ts) > 0 {
					transcript(cur, ts[0], docproc)
					transcripts[k] = ts[1:]
				}
				text(cur, l.Source.File, l.Source.SHA256)
			}
			in.Sources = append(in.Sources, cur)
		}
		cur.Chunks++
		in.Chunks++
		if l.Error != "" {
			cur.Rejected++
			in.Failures++
			src := l.Source.Ref
			switch {
			case l.Source.File != "":
				src = l.Source.File
			case l.Source.Kind == outdir.SourceMetadata:
				src = outdir.SourceMetadata
			}
			v.Rejected = append(v.Rejected, rejected{Input: l.Input, Source: src, Index: l.Index, Error: l.Error})
		}
	}
	for i := range m.Transcripts {
		t := &m.Transcripts[i]
		k := tkey{t.Input, t.File}
		if ts := transcripts[k]; len(ts) > 0 && ts[0] == t {
			transcripts[k] = ts[1:]
			s := &src{Kind: outdir.SourceTranscript}
			transcript(s, t, docproc)
			text(s, t.File, t.SHA256)
			of(t.Input).Sources = append(of(t.Input).Sources, s)
		}
	}
	for _, f := range m.Failures {
		in := of(f.Input)
		in.Failures++
		s := &src{Kind: "refused", Href: docproc + f.Ref, Image: f.Capability == "vision", Error: f.Error,
			Output: &outdir.Output{Ref: f.Ref}}
		in.Sources = append(in.Sources, s)
		if f.Capability == "vision" {
			v.Refused = append(v.Refused, f)
		} else {
			v.Failed = append(v.Failed, f)
		}
	}
	for _, p := range slices.Sorted(maps.Keys(inputs)) {
		v.Inputs = append(v.Inputs, inputs[p])
	}
	return v
}

// transcript gives s the transcript t, its windows linked under docproc, and its seams counted.
func transcript(s *src, t *outdir.Transcript, docproc string) {
	s.Transcript, s.Windows = t, windows(t, docproc)
	for _, seam := range t.Seams {
		if seam.Matched {
			s.Matched++
		} else {
			s.Unmatched++
		}
	}
}

// windows returns t's windows as the page links them, under docproc.
func windows(t *outdir.Transcript, docproc string) []window {
	ws := make([]window, len(t.Windows))
	for i, ref := range t.Windows {
		ws[i] = window{Index: i, Failed: slices.Contains(t.Failed, i)}
		if ref != "" {
			ws[i].Href = docproc + ref
		}
	}
	return ws
}

// clip returns s's first textRunes runes, and whether it had more.
func clip(s string) (string, bool) {
	if utf8.RuneCountInString(s) <= textRunes {
		return s, false
	}
	n := 0
	for i := range s {
		if n == textRunes {
			return s[:i], true
		}
		n++
	}
	return s, false
}
