package schema

import (
	"github.com/JaimeStill/spike-ai-pipeline-processing/dispatch"
)

// Part is one part of an input, the input itself among them, as processing's dispatch reports
// it:
//
//   - "name": the name it was given: an input's file name, or the name its parent's stage gave
//     it, such as an attachment's stored name
//   - "size": its size in bytes
//   - "sha256": the lowercase hex SHA-256 of its content; empty for a part that couldn't be
//     read
//   - "content_type": its [ContentType]
//   - "flags": its inspection's flags, each a [Flag]; optional, absent when it has none
//   - "parent": the SHA-256 of the part it was found in; optional, absent for an input
//   - "depth": 0 for an input, one more than its parent's for a part found in it
//   - "where": where its parent's stage found it, in that stage's words, such as
//     "zip entry notes/a.txt"; optional, absent for an input
type Part struct {
	Name        string      `json:"name"`
	Size        int64       `json:"size"`
	SHA256      string      `json:"sha256"`
	ContentType ContentType `json:"content_type"`
	Flags       []Flag      `json:"flags,omitempty"`
	Parent      string      `json:"parent,omitzero"`
	Depth       int         `json:"depth"`
	Where       string      `json:"where,omitzero"`
}

// ContentType is what a part's content is:
//
//   - "kind": the format its signature names, processing's route.Kind, such as "pdf" or "zip";
//     optional, absent when no signature processing knows matches
//   - "route": the route processing gave it, such as "document" or "text", which names the stage
//     that prepares it; optional, absent for a part that couldn't be read
//   - "route_reason": why processing gave it that route; optional
//   - "stored": the MIME type it was stored under, such as an attachment's declared type;
//     optional, absent when its container declared none, as for every input
//   - "detected": the MIME type net/http.DetectContentType names for its first 512 bytes;
//     optional, present for an input alone
type ContentType struct {
	Kind        string `json:"kind,omitzero"`
	Route       string `json:"route,omitzero"`
	RouteReason string `json:"route_reason,omitzero"`
	Stored      string `json:"stored,omitzero"`
	Detected    string `json:"detected,omitzero"`
}

// Flag is one finding of a part's inspection:
//
//   - "check": the check that raised it, such as "zip-traversal"
//   - "class": what it means for the part, such as "unsafe"
//   - "evidence": where it is, in a few words, such as `entry "../evil.txt"`
type Flag struct {
	Check    string `json:"check"`
	Class    string `json:"class"`
	Evidence string `json:"evidence"`
}

// PartOf returns p as the schema carries it, with no detected MIME type: a caller that has an
// input's file sets ContentType.Detected through [Detect].
func PartOf(p dispatch.Part) Part {
	var flags []Flag
	for _, f := range p.Flags {
		flags = append(flags, Flag{Check: string(f.Check), Class: string(f.Class), Evidence: f.Evidence})
	}
	return Part{
		Name:   p.Name,
		Size:   p.Size,
		SHA256: p.SHA256,
		ContentType: ContentType{
			Kind: string(p.Kind), Route: string(p.Route), RouteReason: p.Reason, Stored: p.MIME,
		},
		Flags:  flags,
		Parent: p.Parent,
		Depth:  p.Depth,
		Where:  p.Where,
	}
}
