// Package report is docproc's run report: one static HTML page, rendered with html/template
// from a run's manifest and record alone, written into the run's directory as
// runs/<run id>/report.html. docproc run writes it as each run ends, and docproc report
// <run-dir> [--run <id>] renders it again, the latest run's by default, and prints its path.
//
// The page's header names the run, its start and end, its workers and its input arguments, the
// error that stopped it if one did, the stage table with each wired stage's identity and tools,
// every tool with its version or why it wasn't built, and the missing and excluded routes with
// their reasons. A summary counts the inputs, and every part, by route and outcome, in the stage
// table's route order. Then each input, sorted by path, shows its file's metadata, its time, and
// its part tree: each part with its route, outcome, content, stage, reason, error, flags, and
// quarantined file, the outputs its stage yielded, and the parts found in it, nested below.
//
// The tree is built from the record's order: a part's outputs, and the parts found in it, come
// before its own event, so a part's own event takes the outputs waiting under its SHA-256, parent,
// and depth, and the parts waiting at one depth more whose parent is its SHA-256.
//
// An output whose media type is an image's shows inline; every other output, text and Markdown
// among them, is linked. Every link and source is relative to the report, ../../ and the record's
// path relative to the run directory, so the run directory can be moved or archived whole. The
// page has no script, font, or style from elsewhere, and follows the reader's light or dark
// scheme. Rendering reads no clock: the same manifest and record render the same bytes.
//
// The package exports:
//
//   - [Name], the report's file name
//   - [Render], which renders a manifest and record, and [Write], which renders a run directory's
//     run into its report
//   - [Commands], which returns the report command
package report
