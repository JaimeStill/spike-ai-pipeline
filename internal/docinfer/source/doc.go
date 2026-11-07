// Package source reads the docproc run docinfer infers over: one run of a docproc run directory,
// schema "1" (docs/docproc-schema.md), the latest by default, through internal/rundir and
// internal/schema. It checks the manifest's schema before reading anything else and refuses any
// but "1", then reads the record whole and keeps its every output line, whatever its part came
// to: a failed part's outputs, prepared before it failed, and a seen part's, whose Ref names the
// copy an earlier run stored, are read like any other.
//
// The package exports:
//
//   - [Run], one docproc run's manifest, record, and output lines, read by [Open]
//   - [Run.Path], an output's file, and [Run.Counts], its outputs counted by kind
//   - [Kinds], the output kinds of schema "1", in its order
package source
