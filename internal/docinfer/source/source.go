package source

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/JaimeStill/spike-ai-pipeline/internal/docproc/rundir"
	"github.com/JaimeStill/spike-ai-pipeline/internal/docproc/schema"
)

// kinds are the output kinds of schema "1", in the order its contract lists them and [Run.Counts]
// writes them.
var kinds = []string{"text", "markdown", "image", "record", "audio"}

// Run is one docproc run, as docinfer reads it: the run directory it is in, its manifest, its
// record, and the record's output lines.
type Run struct {
	// Root is the docproc run directory, absolute: every Ref is relative to it.
	Root     string
	Manifest schema.Manifest
	// Events are the run's record, in the order docproc wrote it.
	Events []schema.Event
	// Outputs are the record's output lines, in its order: every one, whatever its part's
	// outcome, a failed part's and a seen part's among them, each with its input, part, and
	// stage.
	Outputs []schema.Event
}

// Open reads the run id of the docproc run directory at root, its latest run when id is empty.
// It checks the manifest's schema is "1" before reading the record, and refuses any other. An id
// that isn't one run's, a path or a dot entry, is an error; a command refuses it before Open.
func Open(root, id string) (*Run, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if id == "" {
		if id, err = rundir.Latest(abs); err != nil {
			return nil, err
		}
	} else if !rundir.ValidID(id) {
		return nil, fmt.Errorf("%q is no run id", id)
	}
	m, err := rundir.ReadManifest(abs, id)
	if err != nil {
		return nil, err
	}
	if m.Schema != schema.Version {
		return nil, fmt.Errorf("run %s has schema %q; docinfer reads schema %q", id, m.Schema, schema.Version)
	}
	evs, err := rundir.ReadRecord(abs, m)
	if err != nil {
		return nil, err
	}
	r := &Run{Root: abs, Manifest: m, Events: evs}
	for _, ev := range evs {
		if ev.Event == schema.OutputEvent && ev.Output != nil {
			r.Outputs = append(r.Outputs, ev)
		}
	}
	return r, nil
}

// Path returns the file o's Ref names, under the run directory.
func (r *Run) Path(o schema.Output) string {
	return filepath.Join(r.Root, filepath.FromSlash(o.Ref))
}

// Counts returns how many of r's outputs are of each kind, as "text 3, markdown 1, ...": the
// kinds of kinds, in its order, each that has an output, then any other kind, in sorted order.
func (r *Run) Counts() string {
	n := map[string]int{}
	for _, ev := range r.Outputs {
		n[ev.Output.Kind]++
	}
	var parts []string
	for _, k := range kinds {
		if n[k] > 0 {
			parts = append(parts, k+" "+strconv.Itoa(n[k]))
			delete(n, k)
		}
	}
	for _, k := range slices.Sorted(maps.Keys(n)) {
		parts = append(parts, k+" "+strconv.Itoa(n[k]))
	}
	return strings.Join(parts, ", ")
}
