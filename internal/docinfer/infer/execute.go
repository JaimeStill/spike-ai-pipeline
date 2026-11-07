package infer

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/config"
	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/source"
)

// job is one docinfer run's inputs, once its configuration is valid, its docproc run read, and
// its models checked: what the inference reads.
type job struct {
	cfg  *config.Config
	run  *source.Run
	http *http.Client
	out  string // the directory the run writes in
}

// execute runs j's inference, writing its terminal lines to w. It is the seam the inference
// plugs into: for now it reports what it read, the docproc run's id and its outputs by kind, and
// sends nothing and writes nothing to j.out.
func execute(_ context.Context, w io.Writer, j job) error {
	_, err := fmt.Fprintf(w, "docproc run %s: outputs %d: %s\n", j.run.Manifest.Run, len(j.run.Outputs), j.run.Counts())
	return err
}
