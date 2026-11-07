package run

import (
	"context"
	"io"

	"github.com/JaimeStill/spike-ai-pipeline/internal/stages"
)

// Brief is brief, for the terminal's error tests.
var Brief = brief

// ErrorRunes is errorRunes, for the terminal's error tests.
const ErrorRunes = errorRunes

// Execute runs the inputs paths name through set's stages over the run directory out, one worker
// at once, as the run command does once it has built set, writing the terminal's lines to w.
func Execute(ctx context.Context, w io.Writer, set stages.Set, out string, paths ...string) error {
	inputs, err := collect(paths)
	if err != nil {
		return err
	}
	return execute(ctx, w, set, out, paths, inputs, 1)
}
