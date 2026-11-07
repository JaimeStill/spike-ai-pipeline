// Package rundir is docproc's run directory, the --out of docproc run, laid out under schema
// "1":
//
//	index/                               processing's dispatch.FileIndex, shared by every run
//	outputs/<sha256>/<stage>/<n><ext>    each output, stored once by its part's SHA-256 and
//	                                     its stage's identity
//	quarantine/<sha256>/content          a quarantined part's file,
//	quarantine/<sha256>/flags.json       beside its flags
//	runs/<run id>/manifest.json          each run's schema.Manifest
//	runs/<run id>/record.jsonl           and its record, a schema.Event per line
//	runs/<run id>/report.html            and its report, internal/report's
//
// <stage> is the stage's name and the first twelve hex digits of the SHA-256 of its whole
// identity, name, version, and configuration hash, so a new tool version prepares content again
// and stores it apart. <n> counts the part's outputs from 1, in the stage's order, and <ext> is
// its media type's extension.
//
// A run over a directory an earlier run made reuses its index and its store: content the index
// has recorded is seen, and its outputs' Refs, which processing's index recorded as the caller
// set them, name the copies the earlier run stored. Each run has its own id, a version 7 UUID,
// and its own manifest and record. Every path to a file docproc writes, an output's Ref, a
// quarantined part's file and flags, and a run's record, is relative to the run directory, with
// slashes, so the directory can be moved whole; an input's path and its argument, which docproc
// only reads, are absolute, and a tool's path is the host's.
//
// Every file is written whole: to a temporary name in its directory, synced, and renamed. An
// output or a quarantined file already there is kept as it is, so content prepared again, as
// when an earlier run's part failed below it, is stored once.
//
// The package exports:
//
//   - [Dir], made by [Open], with its [Dir.Root], its [Dir.Index], [Dir.Store], which stores an
//     output and returns its Ref, and [Dir.Quarantine], which keeps a quarantined part's file
//   - [Run], begun by [Dir.Begin], with its [Run.ID], [Run.Record], [Run.Write], which appends a
//     record line, and [Run.End], which closes the record and writes the manifest's end
//   - [Runs], the runs in a run directory in the order they began, [Latest], the latest of
//     them, [ValidID], whether an id can name one run, [Load], which reads a run's manifest and
//     record, [ReadManifest] and [ReadRecord], which read each apart, so a reader can check the
//     schema first, and [WriteRunFile], which writes a file beside them, such as its report
package rundir
