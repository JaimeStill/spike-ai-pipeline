package infer

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/outdir"
	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/source"
	"github.com/JaimeStill/spike-ai-pipeline/internal/schema"
)

// textKinds are the output kinds whose stored file is a text docinfer embeds: never a record,
// and never an image or audio directly; their texts, a vision model's or a transcript, come from
// those capabilities.
var textKinds = map[string]bool{"text": true, "markdown": true}

// imageKind is the output kind whose stored file is an image, which vision makes a text of.
const imageKind = "image"

// occurrence is one place a text docinfer embeds occurs in the docproc run: the input, part, and
// output line it belongs to, and its source text. Each of the text's chunks is one chunk line per
// occurrence, so the same text under two inputs is two sets of lines. A capability that makes
// texts of its own, vision of an image or transcription of an audio window, adds an occurrence
// of its text, its Source naming the file it stored, and the text is cut and embedded as any
// other.
//
// An image output's occurrence is one awaiting its text: image is its stored file's path and
// mediaType its media type, as docproc records it, and its text and source are vision's to set.
type occurrence struct {
	input       string
	file        *schema.File
	contentType *schema.ContentType
	part        *outdir.Part
	output      *outdir.Output
	source      outdir.Source
	text        string
	image       string
	mediaType   string
}

// inputInfo is what an input's own lines say of it: its file metadata and content type, from its
// own part line or its error line, and its own part, with that line's outcome.
type inputInfo struct {
	file        *schema.File
	contentType *schema.ContentType
	part        *outdir.Part
}

// partKey identifies a part within its input's lines, so an output line finds its part line's
// outcome.
type partKey struct {
	input, sha256, parent, where, name string
	depth                              int
}

func keyOf(input string, p *schema.Part) partKey {
	return partKey{input: input, sha256: p.SHA256, parent: p.Parent, where: p.Where, name: p.Name, depth: p.Depth}
}

// partOf returns p in docinfer's names, with outcome.
func partOf(p *schema.Part, outcome string) *outdir.Part {
	return &outdir.Part{
		SHA256: p.SHA256, Name: p.Name, Kind: p.ContentType.Kind, Route: p.ContentType.Route,
		Where: p.Where, Depth: p.Depth, Outcome: outcome,
	}
}

// occurrences returns the occurrences of every text docinfer embeds in run, in its record's
// order: each text or Markdown output line, whatever its part's outcome, its file read from the
// run directory; each image output line, whatever its part's outcome, awaiting its vision text;
// and, at each input's own part line or error line, its metadata text, when the line carries
// file metadata. An output's stored text file that can't be read is an error.
func occurrences(run *source.Run) ([]occurrence, error) {
	// An input's own lines, and each part's outcome, come after its output lines; read them
	// first.
	inputs := map[string]*inputInfo{}
	outcomes := map[partKey]string{}
	for _, ev := range run.Events {
		switch {
		case ev.Event == schema.PartEvent && ev.Part != nil:
			outcomes[keyOf(ev.Input, ev.Part)] = ev.Outcome
			if ev.Part.Depth == 0 {
				ct := ev.Part.ContentType
				inputs[ev.Input] = &inputInfo{file: ev.File, contentType: &ct, part: partOf(ev.Part, ev.Outcome)}
			}
		case ev.Event == schema.ErrorEvent:
			if inputs[ev.Input] == nil {
				inputs[ev.Input] = &inputInfo{file: ev.File}
			}
		}
	}
	var occs []occurrence
	texts := map[string]string{} // by ref, each stored file read once
	for _, ev := range run.Events {
		info := inputs[ev.Input]
		if info == nil {
			info = &inputInfo{}
		}
		switch ev.Event {
		case schema.OutputEvent:
			if ev.Output == nil || (!textKinds[ev.Output.Kind] && ev.Output.Kind != imageKind) {
				continue
			}
			o := ev.Output
			var part *outdir.Part
			if ev.Part != nil {
				part = partOf(ev.Part, outcomes[keyOf(ev.Input, ev.Part)])
			}
			occ := occurrence{
				input: ev.Input, file: info.file, contentType: info.contentType, part: part,
				output: &outdir.Output{Kind: o.Kind, Name: o.Name, MediaType: o.MediaType, Ref: o.Ref, Meta: o.Meta},
			}
			if o.Kind == imageKind {
				occ.image, occ.mediaType = run.Path(*o), o.MediaType
				occs = append(occs, occ)
				continue
			}
			text, ok := texts[o.Ref]
			if !ok {
				b, err := os.ReadFile(run.Path(*o))
				if err != nil {
					return nil, fmt.Errorf("output %s of %s: %w", o.Ref, ev.Input, err)
				}
				text = string(b)
				texts[o.Ref] = text
			}
			occ.source, occ.text = outdir.Source{Kind: outdir.SourceOutput, Ref: o.Ref}, text
			occs = append(occs, occ)
		case schema.PartEvent, schema.ErrorEvent:
			if ev.File == nil || (ev.Event == schema.PartEvent && (ev.Part == nil || ev.Part.Depth != 0)) {
				continue
			}
			occs = append(occs, occurrence{
				input: ev.Input, file: info.file, contentType: info.contentType, part: info.part,
				source: outdir.Source{Kind: outdir.SourceMetadata},
				text:   metadataText(ev.File, info.contentType),
			})
		}
	}
	return occs, nil
}

// metadataText returns an input's metadata text: its file metadata and content type, as docproc
// records them, one readable line each, in the schema's order, an optional field that is absent
// left out. It lists none of the input's parts.
func metadataText(f *schema.File, ct *schema.ContentType) string {
	var b strings.Builder
	line := func(label, value string) {
		if value != "" {
			b.WriteString(label + ": " + value + "\n")
		}
	}
	line("file name", f.Name)
	line("path argument", f.Arg)
	line("path", f.Path)
	line("size", strconv.FormatInt(f.Size, 10)+" bytes")
	line("mode", f.Mode)
	if !f.Modified.IsZero() {
		line("modified", f.Modified.String())
	}
	if !f.Created.IsZero() {
		line("created", f.Created.String())
	}
	if ct != nil {
		line("kind", ct.Kind)
		line("route", ct.Route)
		line("route reason", ct.RouteReason)
		line("stored MIME type", ct.Stored)
		line("detected MIME type", ct.Detected)
	}
	return b.String()
}
