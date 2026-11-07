// Package run is docproc's run command: docproc run <path>... --out <dir>. It builds the stages
// over the tools the host holds, printing the tools and stages as tables, then dispatches every
// input through processing's pool and prints one line per input as its last event arrives.
//
// --exclude <route>, repeatable, leaves a route's stage out, and a route that isn't one of
// processing's ten is a usage error. --workers, runtime.NumCPU() by default, is how many inputs
// are dispatched at once; outside 1 to NumCPU it is a usage error.
//
// The package exports:
//
//   - [Commands], which returns the run command over the tool lookup node it declares
package run
