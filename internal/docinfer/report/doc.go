// Package report is docinfer's run report: one static HTML page, rendered with html/template
// from a run's manifest, its chunk record, and the texts docinfer stored, all in docinfer's
// directory, written into the run's directory as runs/<run id>/report.html. docinfer run writes
// it as each run ends, and docinfer report <dir> [--run <id>] renders it again, the latest run's
// by default, and prints its path; a --run that is a path or a dot entry is a usage error. It
// reads nothing of the docproc run and sends no request.
//
// The page's header names the run, its start and end, the docproc run it read, and the error
// that stopped it, if one did. Then it shows each capability the run ran, with its endpoint,
// model, and settings, and each it skipped, with why; the embedding space; the manifest's counts;
// and every failure: each image output line vision refused and each window output line
// transcription refused, from the manifest's failures, and each chunk line whose chunk the
// endpoint rejected, from the chunk record.
//
// Then each input, sorted by path, shows its sources: each occurrence of a text it embedded, in
// the chunk record's order, which is the docproc record's, beside its chunk count. A text or
// Markdown output is linked; a vision text shows beside its image, inline; a transcript shows
// with its windows, each linked; metadata shows its count alone. A vision text or transcript
// shows its first 1,000 runes, linked to the whole stored text, so a run over thousands of images
// makes a page of a size a browser opens; no chunk's text or vector is on the page. A transcript
// that cut to no chunk, and each output a capability refused, with why, follow. An image or
// window whose text is empty, and an image vision skipped, has no chunk line and is not shown,
// though the counts count it.
//
// Every link into docinfer's directory is relative to the report, ../../ and the path the
// manifest or chunk line records. Every link into the docproc run directory, an image, a window,
// or a text output, is the relative path from the report's directory to the directory the
// manifest names, computed as the page is rendered, then the output's ref: the page holds no copy
// of them, and docinfer's directory and the docproc run directory can be moved together. The
// page has no script, font, or style from elsewhere, and follows the reader's light or dark
// scheme. Rendering reads no clock: the same directory renders the same bytes.
//
// The package exports:
//
//   - [Name], the report's file name
//   - [Write], which renders a run of docinfer's directory into its report
//   - [Commands], which returns the report command
package report
