// Package schema is the run directory's schema "1": the types docproc writes as JSON, through
// encoding/json, and a later reader, docinfer, decodes. Each type's doc comment is its part of
// the contract: a field's JSON name, what it holds, and when it is absent. A field the comments
// don't call optional is always present, its zero value included.
//
// A time is a [Time]: RFC 3339, in UTC, with nine digits of nanoseconds, such as
// "2026-11-07T15:04:05.123456789Z".
//
// An input's [File] is its file's metadata, read when its dispatch ends: its name, the path
// argument it came from and its path under that argument, its size, its mode, and its modified
// time, and its created time where the filesystem records one, read through statx on Linux.
// [FileOf] reads it through a [Stater], [Statx] in production, which a test replaces.
//
// Every part, an input or one found in it, is a [Part]: processing's dispatch.Part as [PartOf]
// carries it, its name, size, SHA-256, flags, parent, depth, and where it was found, and its
// [ContentType], the kind its signature names, its route and why, its stored MIME type, and, for
// an input alone, the MIME type net/http.DetectContentType names, which [Detect] reads. A part
// found in another has no file of its own on the caller's disk, so it carries no [File].
//
// A run directory holds, for each run, a [Manifest], runs/<run>/manifest.json, and a record,
// runs/<run>/record.jsonl, one [Event] per line: an "output" line for each [Output], a "part"
// line for each part's own event, and an "error" line for an input whose dispatch stopped. Every
// path to a file docproc writes, an output's Ref, a [Quarantine]'s file and flags, and the
// manifest's record, is relative to the run directory, with slashes, so the run directory can be
// moved whole. An input's path and its argument, which docproc only reads, are absolute and
// cleaned, the manifest's inputs among them; a [File]'s "path" is relative to its "arg"; and a
// [Tool]'s path is the host's.
//
// The package exports:
//
//   - [Version], the schema's version, "1"
//   - [Time], a time as the schema writes it
//   - [Part], [ContentType], and [Flag], and [PartOf], which carries a dispatch.Part
//   - [File], an input's file metadata; [FileOf], which reads it; [Stat], what a [Stater]
//     returns; and [Statx], the production Stater
//   - [Detect], the MIME type of a file's first 512 bytes
//   - [Event], a record line, of kind [OutputEvent], [PartEvent], or [ErrorEvent], with its
//     [Stage], [Output], [Quarantine], and [Span]
//   - [Manifest], a run's manifest, with its [Tool]s and [StageState]s
package schema
