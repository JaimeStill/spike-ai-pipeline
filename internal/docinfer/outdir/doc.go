// Package outdir is docinfer's own directory, the one --out names, schema "1":
//
//	<out>/
//	  results/<capability>/<xx>/<key>.json   each held result, keyed by Key
//	  runs/<run id>/manifest.json            each run's manifest
//	  runs/<run id>/chunks.jsonl             its chunk record, a line per chunk per occurrence
//
// A held result is keyed by [Key]: the SHA-256 of the content it is of, the capability, the
// model, and the settings that change it, such as the chunk bound and the chunker's version for
// embeddings, so a run over the same content and configuration finds what an earlier one, or an
// interrupted one, stored, and a changed setting finds nothing. Every file is written whole,
// through internal/rundir's WriteFile, to a temporary name renamed into place, so a reader sees a
// whole file or none, and a run stopped part way loses only what it hadn't stored. Every path
// docinfer records of a file it wrote is relative to the directory, with slashes.
//
// A run's [Manifest] is written as the run starts and again, whole, as it ends: its id, a
// version 7 UUID, its times, the docproc run it read, each capability it ran, with its endpoint,
// model, and settings, the capabilities it skipped, the embedding [Space], its [Counts], and what
// stopped it, if anything. Its chunk record holds a [Line] per chunk per place the chunk's text
// occurs: the chunk's text and byte [Span] in its [Source], the input's file metadata and content
// type, the [Part] and [Output] it belongs to, and its vector, base64 of little-endian float32s
// as the endpoint returned it, never normalized, or why the endpoint rejected it.
//
// The package exports:
//
//   - [Dir], opened by [Open], with its [Dir.Root], [Dir.Held] and [Dir.Hold], which read and
//     write a held result, [Dir.WriteManifest] and [Dir.WriteChunks], which write a run's
//     files, and [Dir.ReadManifest] and [Dir.ReadChunks], which read them
//   - [Key], a held result's key, and [RunPath], a run's file's path in the directory
//   - [Manifest], with its [Docproc], [Capability], [Skipped], [Space], and [Counts]
//   - [Line], with its [Span], [Source], of kind [SourceOutput] or [SourceMetadata], [Part], and
//     [Output]
//   - [Embedding], a held embeddings result
//   - [EncodeVector] and [DecodeVector], a vector's base64 form, and [UnitTolerance], within
//     which a vector's norm counts as unit length
//   - [Version], the directory's schema
package outdir
