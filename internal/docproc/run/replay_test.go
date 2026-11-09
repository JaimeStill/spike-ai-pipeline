package run_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/JaimeStill/spike-ai-pipeline/internal/docproc/run"
	"github.com/JaimeStill/spike-ai-pipeline/internal/docproc/schema"
)

// stage is the one stage identity the replay's records name.
var stage = &schema.Stage{Name: "archive", Version: "1", Config: "c"}

// partLine returns a part line of input for the part sha, found in parent at depth, with its
// outcome, recorded outcome, and children count.
func partLine(input, sha, parent string, depth int, outcome, recorded string, children int) schema.Event {
	return schema.Event{
		Event: schema.PartEvent, Input: input, Stage: stage, Outcome: outcome, Recorded: recorded,
		Part:    &schema.Part{Name: sha, SHA256: sha, Parent: parent, Depth: depth},
		Outputs: new(int), Children: &children,
	}
}

// outputLine returns a prepared output line of input for the part sha, found in parent at depth.
func outputLine(input, sha, parent string, depth int) schema.Event {
	return schema.Event{
		Event: schema.OutputEvent, Input: input, Stage: stage,
		Part:   &schema.Part{Name: sha, SHA256: sha, Parent: parent, Depth: depth},
		Output: &schema.Output{Kind: "text", Ref: "outputs/" + sha + "/1.txt"},
	}
}

// briefLines returns each line as its kind, part, depth, input, outcome, and recorded outcome,
// and whether it warns.
func briefLines(evs []schema.Event) []string {
	var out []string
	for _, ev := range evs {
		out = append(out, fmt.Sprintf("%s %s %d %s %s/%s %t", ev.Event, ev.Part.SHA256, ev.Part.Depth, ev.Input,
			ev.Outcome, ev.Recorded, ev.Warning != ""))
	}
	return out
}

// Replay copies a seen part's subtree from the earliest record that holds it, at the seen
// part's depth and of its input, every line seen, recording the outcome its record gave it, in
// the record's order; a part a later record holds without its children, as a record written
// before subtrees were copied does, takes them from the record that holds them; and a seen part
// no record holds is copied with nothing, and warns.
func TestReplay(t *testing.T) {
	// The first run: z holds y, which holds t, whose text it prepared.
	first := []schema.Event{
		outputLine("/a", "t", "y", 2),
		partLine("/a", "t", "y", 2, "prepared", "", 0),
		partLine("/a", "y", "z", 1, "prepared", "", 1),
		partLine("/a", "z", "", 0, "prepared", "", 1),
	}
	// A later run, before subtrees were copied: w holds y, seen, and y's child has no line.
	second := []schema.Event{
		partLine("/b", "y", "w", 1, "seen", "prepared", 1),
		partLine("/b", "w", "", 0, "prepared", "", 1),
	}
	records := [][]schema.Event{first, second}

	for _, tc := range []struct {
		name     string
		seen     schema.Event
		want     []string
		warnings int
	}{{
		name: "from the first record, a depth deeper",
		seen: partLine("/c", "z", "q", 1, "seen", "prepared", 1),
		want: []string{
			"output t 3 /c seen/prepared false",
			"part t 3 /c seen/prepared false",
			"part y 2 /c seen/prepared false",
			"part z 1 /c seen/prepared false",
		},
	}, {
		name: "through a seen part with no children's lines",
		seen: partLine("/c", "w", "", 0, "seen", "prepared", 1),
		want: []string{
			"output t 2 /c seen/prepared false",
			"part t 2 /c seen/prepared false",
			"part y 1 /c seen/prepared false",
			"part w 0 /c seen/prepared false",
		},
	}, {
		name:     "held by no record",
		seen:     partLine("/c", "v", "", 0, "seen", "prepared", 2),
		want:     []string{"part v 0 /c seen/prepared true"},
		warnings: 1,
	}, {
		name: "with no children",
		seen: partLine("/c", "t", "", 0, "seen", "prepared", 0),
		want: []string{"part t 0 /c seen/prepared false"},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			got, warnings := run.Replay(records, tc.seen)
			if lines := briefLines(got); !slices.Equal(lines, tc.want) || warnings != tc.warnings {
				t.Errorf("Replay = %q, %d warnings; want %q, %d", lines, warnings, tc.want, tc.warnings)
			}
		})
	}
}
