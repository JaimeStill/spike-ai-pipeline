package run

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"text/tabwriter"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/JaimeStill/spike-ai-pipeline-processing/route"

	"github.com/JaimeStill/spike-ai-pipeline/internal/docproc/schema"
	"github.com/JaimeStill/spike-ai-pipeline/internal/docproc/stages"
)

// errorOutcome is the outcome the terminal and the summary give an input whose dispatch stopped
// with an error before its own event.
const errorOutcome = "error"

// errorRunes is how many runes of an error the terminal prints, before its ellipsis: enough
// for the whole first line of every error the runs over ~/docs recorded, whose cause follows a
// path into the run's working directory, and no more than a few terminal rows.
const errorRunes = 400

// tallyKey is one cell of the closing summary: a route and an outcome.
type tallyKey struct {
	route   string
	outcome string
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

// brief returns err as the terminal prints it, one printable line: its first line, with each
// rune that isn't printable, a control byte, a byte that isn't UTF-8, or a format rune, replaced
// by U+FFFD, and cut at errorRunes runes. An ellipsis says more of it is in the record, whose
// error field holds it all: processing's errors can run to several lines, and one carries
// Ghostscript's raw PNG output. docs/findings/docproc-results/counts.sh reduces a record's
// errors the same way.
func brief(err error) string {
	first, _, more := strings.Cut(strings.TrimSpace(err.Error()), "\n")
	var b strings.Builder
	n := 0
	for _, c := range first {
		if n == errorRunes {
			more = true
			break
		}
		if !unicode.IsPrint(c) {
			c = utf8.RuneError
		}
		b.WriteRune(c)
		n++
	}
	if more {
		b.WriteString("…")
	}
	return b.String()
}

// summary writes the closing summary to the terminal: how many inputs came to each route and
// outcome, in the stage table's route order, then the total, how many seen parts had children no
// record holds, when any did, where the run's record is, and where its report is, page, when it
// was written.
func (r *recorder) summary(page string) error {
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
	if r.warned > 0 {
		row(fmt.Sprintf("warnings %d: seen parts whose children no record in the run directory holds, not replayed; their part lines' \"warning\" says so", r.warned))
	}
	row("run " + r.run.ID() + "  " + r.run.Record())
	if page != "" {
		row("report " + page)
	}
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
