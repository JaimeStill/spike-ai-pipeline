package run

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/JaimeStill/spike-ai-pipeline/internal/rundir"
	"github.com/JaimeStill/spike-ai-pipeline/internal/schema"
)

// Processing's index replays a seen part's own outputs but dispatches none of the parts found in
// it: its entry references them, and their outputs are in the store. So that every run's record
// is whole on its own, the recorder copies a seen part's subtree, the lines of the parts found in
// it and their outputs, from the earliest record in the run directory that holds it, this run's
// own among them, last, for content seen twice in one run.

// subtreeKey is what a seen part's subtree is found by: its content's SHA-256 and its stage's
// identity, the index's key, so content an earlier run prepared under another stage identity,
// whose subtree may differ, is never copied for it.
type subtreeKey struct {
	sha   string
	stage schema.Stage
}

// keyOf returns the key ev, a part line, is found by, and whether it has one: a part with no
// SHA-256 or no stage has no subtree.
func keyOf(ev schema.Event) (subtreeKey, bool) {
	if ev.Part == nil || ev.Part.SHA256 == "" || ev.Stage == nil {
		return subtreeKey{}, false
	}
	return subtreeKey{ev.Part.SHA256, *ev.Stage}, true
}

// node is one part of a record: its own line, its output lines, and the nodes of the parts found
// in it, in the record's order.
type node struct {
	line     schema.Event
	outputs  []schema.Event
	children []*node
}

// whole reports whether n's record holds a line for every part found in it, as its children
// count says: a seen part's subtree in a record written before docproc copied subtrees has
// none.
func (n *node) whole() bool {
	return len(n.children) == count(n.line.Children)
}

// tree builds the nodes of one input's record lines, which arrive in the record's order: a
// part's output lines and the lines of the parts found in it before its own, as internal/report's
// tree takes them.
type tree struct {
	outputs map[outputKey][]schema.Event // output lines waiting for their part's own line
	waiting []*node                      // nodes waiting for their parent's own line
}

// outputKey is which part an output line belongs to: its SHA-256, its parent's, and its depth.
type outputKey struct {
	sha, parent string
	depth       int
}

// add adds ev, a line of the input's record, and returns its node when it is a part line.
func (t *tree) add(ev schema.Event) *node {
	if ev.Part == nil {
		return nil
	}
	k := outputKey{ev.Part.SHA256, ev.Part.Parent, ev.Part.Depth}
	switch ev.Event {
	case schema.OutputEvent:
		if t.outputs == nil {
			t.outputs = map[outputKey][]schema.Event{}
		}
		t.outputs[k] = append(t.outputs[k], ev)
		return nil
	case schema.PartEvent:
		n := &node{line: ev, outputs: t.outputs[k]}
		delete(t.outputs, k)
		if ev.Part.SHA256 != "" {
			kept := t.waiting[:0]
			for _, c := range t.waiting {
				if c.line.Part.Parent == ev.Part.SHA256 && c.line.Part.Depth == ev.Part.Depth+1 {
					n.children = append(n.children, c)
				} else {
					kept = append(kept, c)
				}
			}
			clear(t.waiting[len(kept):])
			t.waiting = kept
		}
		t.waiting = append(t.waiting, n)
		return n
	}
	return nil
}

// subtrees are the subtrees the records in a run directory hold, by the key their part is found
// by, and the trees of the inputs whose lines are still arriving.
type subtrees struct {
	found  map[subtreeKey]*node
	inputs map[string]*tree
}

// add adds ev, a record line, to its input's tree, and, once ev is a part line whose node is
// whole and has children, the node to found, unless an earlier record's is already there. An
// input's own line, its last, ends its tree.
func (s *subtrees) add(ev schema.Event) {
	t := s.inputs[ev.Input]
	if t == nil {
		t = &tree{}
		s.inputs[ev.Input] = t
	}
	n := t.add(ev)
	if n == nil {
		if ev.Event == schema.ErrorEvent {
			delete(s.inputs, ev.Input)
		}
		return
	}
	if k, ok := keyOf(n.line); ok && len(n.children) > 0 && n.whole() {
		if _, ok := s.found[k]; !ok {
			s.found[k] = n
		}
	}
	if n.line.Part.Depth == 0 {
		delete(s.inputs, ev.Input)
	}
}

// readSubtrees reads every record in the run directory at root once, in the order the runs
// began, and returns the subtrees they hold, ready for this run's own lines. A record that can't
// be read holds none, and is returned among skipped, by its run id, with why: a run directory
// with one broken record is still run over. A run directory with no runs yet holds none.
func readSubtrees(root string) (*subtrees, []error) {
	s := &subtrees{found: map[subtreeKey]*node{}, inputs: map[string]*tree{}}
	ids, err := rundir.Runs(root)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, []error{err}
	}
	var skipped []error
	for _, id := range ids {
		_, evs, err := rundir.Load(root, id)
		if err != nil {
			skipped = append(skipped, fmt.Errorf("run %s: %w", id, err))
			continue
		}
		for _, ev := range evs {
			s.add(ev)
		}
		// An input a stopped run never finished leaves its waiting lines; they hold no whole
		// subtree it didn't already add.
		clear(s.inputs)
	}
	return s, skipped
}

// replay returns the lines that copy the subtree of seen, a seen part's own line in this run,
// from the record that holds it, each at its depth under seen and of seen's input, ending with
// seen's own line; and how many seen parts, seen among them, had children no record holds, each
// of whose lines carries a warning instead.
func (s *subtrees) replay(seen schema.Event) ([]schema.Event, int) {
	var lines []schema.Event
	warnings := 0
	if count(seen.Children) > 0 {
		k, _ := keyOf(seen)
		if src := s.found[k]; src != nil {
			path := map[subtreeKey]bool{k: true}
			for _, c := range src.children {
				warnings += s.copy(&lines, c, seen.Input, seen.Part.Depth+1, path)
			}
		} else {
			seen.Warning = missing(seen)
			warnings++
		}
	}
	return append(lines, seen), warnings
}

// copy appends to lines the copy of n's subtree, n's outputs, the subtrees of the parts found in
// it, and n's own line, each of input, n at depth, every line seen, its recorded outcome the one
// its record gave it. A part whose record holds no line for the parts found in it, as one seen in
// a record written before docproc copied subtrees, takes them from the record that holds them,
// unless it is on path, the parts being copied above it. It returns how many parts it copied
// whose children no record holds.
func (s *subtrees) copy(lines *[]schema.Event, n *node, input string, depth int, path map[subtreeKey]bool) int {
	line := copied(n.line, input, depth, recordedOf(n.line))
	children, warnings := n.children, 0
	if !n.whole() {
		k, ok := keyOf(n.line)
		if src := s.found[k]; ok && src != nil && !path[k] {
			children = src.children
		} else {
			line.Warning = missing(line)
			warnings++
		}
	}
	for _, o := range n.outputs {
		recorded := recordedOf(n.line)
		if o.Outcome != "" {
			recorded = o.Recorded
		}
		*lines = append(*lines, copied(o, input, depth, recorded))
	}
	if k, ok := keyOf(n.line); ok && len(children) > 0 {
		path[k] = true
		defer delete(path, k)
	}
	for _, c := range children {
		warnings += s.copy(lines, c, input, depth+1, path)
	}
	*lines = append(*lines, line)
	return warnings
}

// copied returns ev, a line of another record or another input, as a line of input, its part at
// depth, seen, its recorded outcome recorded.
func copied(ev schema.Event, input string, depth int, recorded string) schema.Event {
	part := *ev.Part
	part.Depth = depth
	ev.Input, ev.Part, ev.Outcome, ev.Recorded, ev.Warning = input, &part, "seen", recorded, ""
	return ev
}

// recordedOf returns the outcome a copy of ev, a part line, records: the one its record gave
// it, or, when it was seen there too, the one it recorded.
func recordedOf(ev schema.Event) string {
	if ev.Outcome == "seen" {
		return ev.Recorded
	}
	return ev.Outcome
}

// missing returns the warning of ev, a seen part line whose children no record in the run
// directory holds.
func missing(ev schema.Event) string {
	return fmt.Sprintf("seen with %d children, but no record in the run directory holds their lines: they are not replayed", count(ev.Children))
}

// count returns *n, or 0 for nil.
func count(n *int) int {
	if n == nil {
		return 0
	}
	return *n
}
