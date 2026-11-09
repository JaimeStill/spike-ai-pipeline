// Command docproc runs processing's stages over a set of inputs. Its main derives the signal
// context and makes one call into the composition root, internal/docproc/app.
package main

import (
	"os"

	"github.com/standards-lab/go-cli-sdk"
	"github.com/standards-lab/go-core/process"

	"github.com/JaimeStill/spike-ai-pipeline/internal/docproc/app"
)

func main() {
	os.Exit(run())
}

// run runs docproc and returns its exit code. It is separate from main so the deferred stop
// releases the signal registration before os.Exit, which skips deferred calls.
func run() int {
	ctx, stop := process.SignalContext()
	defer stop()
	return app.New(cli.Streams{Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr}).Run(ctx, os.Args[1:])
}
