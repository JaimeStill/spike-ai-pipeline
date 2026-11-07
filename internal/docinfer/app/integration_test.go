//go:build integration

package app_test

import (
	"os"
	"testing"

	"github.com/standards-lab/go-core/process"
)

// TestRun_ModelsAgainstConfig loads the configuration DOCINFER_CONFIG names, such as
// config/example, which serves set A on gb10-2, and runs docinfer over the committed docproc run,
// so the models check lists each endpoint it names. It passes when every capability's model is
// listed; a warning, such as a model the router lists unloaded, is logged. It never loads a model.
func TestRun_ModelsAgainstConfig(t *testing.T) {
	dir := os.Getenv("DOCINFER_CONFIG")
	if dir == "" {
		t.Skip("DOCINFER_CONFIG is unset: set it to a docinfer configuration directory, such as ../../../config/example")
	}
	stdout, _ := run(t, process.ExitOK, "run", docproc, "--config", dir, "--out", t.TempDir())
	t.Logf("docinfer run:\n%s", stdout)
	wantLine(t, stdout, `embeddings\s+\S+\s+\S+\s+\S+`)
}
