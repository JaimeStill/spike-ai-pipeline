// Package run is docproc's run command: docproc run <path>... --out <dir>. It builds the stages
// over the tools the host holds, printing the tools and stages as tables, then dispatches every
// input through processing's pool over the run directory --out names, internal/rundir's, under a
// new run: it writes the run's manifest and its record, schema "1", stores each output once by
// its part's SHA-256 and its stage's identity, and keeps each quarantined part's file beside its
// flags.
//
// Each path is a file, or a directory walked for its regular files; no symbolic link is
// followed. As each input's last event arrives, the terminal prints one line: its path, route,
// outcome, outputs, children, and time, an error after the outcome cut to its first line, at
// most 400 runes, its unprintable runes replaced, and kept whole in the record. A closing summary counts the inputs by route and
// outcome, and names the run, its record, and its report, internal/report's HTML page, written
// into the run's directory as the run ends. A part that fails is counted, not fatal, and so is
// an input whose dispatch an error stopped, which its line and the summary call "error"; an
// error that stops the whole dispatch exits 1, once the manifest names it.
//
// An output prepared in the run is stored, and its Ref, processing's caller-set key, set to
// where, before the loop body returns, so the file index records the Ref: a later run over the
// same run directory sees the content and its output events replay the Ref, naming the copy
// already stored. Such an output's inline Value is cleared once stored, so the index holds no
// second copy of it.
//
// --exclude <route>, repeatable, leaves a route's stage out, and a route that isn't one of
// processing's ten is a usage error. --workers, runtime.NumCPU() by default, is how many inputs
// are dispatched at once; outside 1 to NumCPU it is a usage error.
//
// The package exports:
//
//   - [Commands], which returns the run command over the tool lookup node it declares
package run
