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
// The inference is execute's: for now it prints the docproc run's id and how many outputs it
// read, by kind, and writes nothing.
//
// The package exports:
//
//   - [Commands], which returns the run command over the HTTP client node it declares
package infer
