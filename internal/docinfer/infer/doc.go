// Package infer is docinfer's run command: docinfer run <docproc-dir> --config <dir> --out <dir>
// [--run <id>]. It reads its configuration from the directory --config names, internal/docinfer/
// config's, then the docproc run, internal/docinfer/source's, the latest in <docproc-dir> or the
// one --run names, refusing a schema other than "1", then checks each capability's model against
// its endpoint's listing, internal/docinfer/models', and prints the table of capabilities, before
// the inference runs. A configuration error, a schema refused, and a model the listing doesn't
// name each exit 1 before any request but the listings'.
//
// --run is one run's id: a path or a dot entry is a usage error, as docproc report has it.
// --config and --out are required.
//
// The inference is execute's. It begins a run in --out, internal/docinfer/outdir's directory,
// and embeds every text of the docproc run: each text and Markdown output line's stored file,
// whatever its part came to, never a record, an image, or an audio window directly, and one
// metadata text per input with file metadata, its file fields and content type as readable lines,
// an input whose dispatch stopped with an error among them. Each distinct text is cut by
// internal/docinfer/chunk to the configured bound, and each distinct chunk is embedded once, by
// internal/docinfer/embed, unless an earlier run holds its result, keyed by its content, the
// model, the bound, and the chunker's version; each result is held as it completes, so a run
// stopped part way resumes from what it stored. The run writes a chunk line per chunk per
// occurrence, the same content under two inputs two lines, one vector, and its manifest, with the
// embedding space. Vision and transcription, when configured, are skipped for now, the terminal
// and the manifest saying so; their texts will be occurrences cut and embedded like any other.
//
// The package exports:
//
//   - [Commands], which returns the run command over the HTTP client node it declares
package infer
