package run

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/JaimeStill/spike-ai-pipeline-processing/dispatch"
	"github.com/JaimeStill/spike-ai-pipeline-processing/route"
	"github.com/standards-lab/go-cli-sdk"
	"github.com/standards-lab/go-core/graph"

	"github.com/JaimeStill/spike-ai-pipeline/internal/stages"
)

// Commands returns the run command, which builds its stages over the tools lookup finds.
func Commands(lookup *graph.Node[stages.Lookup]) []*cli.Command {
	return []*cli.Command{command(lookup)}
}

// command returns the run command: docproc run <path>... --out <dir>.
func command(lookup *graph.Node[stages.Lookup]) *cli.Command {
	var (
		out      string
		exclude  []string
		workers  int
		excluded []route.Route
	)
	cmd := &cli.Command{
		Name:     "run",
		Summary:  "Run processing's stages over each input file, and each regular file under each directory",
		Synopsis: "<path>...",
		Args: func(args []string) error {
			if len(args) == 0 {
				return cli.Usagef("requires at least one path")
			}
			return nil
		},
		Validate: func(*cli.Invocation) error {
			if out == "" {
				return cli.Usagef("--out names no directory")
			}
			if n := runtime.NumCPU(); workers < 1 || workers > n {
				return cli.Usagef("--workers %d is outside 1 to %d, this machine's CPUs", workers, n)
			}
			routes, err := stages.ParseRoutes(exclude)
			if err != nil {
				return cli.Usagef("--exclude: %w", err)
			}
			excluded = routes
			return nil
		},
		Run: func(ctx context.Context, inv *cli.Invocation) error {
			set := stages.Build(ctx, inv.Get(lookup), excluded)
			if err := set.WriteTable(inv.Stdout); err != nil {
				return err
			}
			inputs, err := collect(inv.Args)
			if err != nil {
				return err
			}
			return dispatchAll(ctx, inv.Stdout, set.Stages, out, inputs, workers)
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&out, "out", "", "the `dir`ectory the run writes in, made when it doesn't exist")
	cli.StringsVar(flags, &exclude, "exclude", nil, "leave out the stage of `route`, which then has no processor yet")
	flags.IntVar(&workers, "workers", runtime.NumCPU(), "how many inputs are dispatched at once, 1 to this machine's CPUs")
	cmd.Require("out")
	return cmd.Use(lookup)
}

// collect returns the inputs paths name, in order, each once: a regular file as given, and every
// regular file under a directory, in lexical order. No symbolic link is followed, a path given
// among them included, and a symbolic link, like any file that isn't regular, is no input. A
// path that can't be read is an error.
func collect(paths []string) ([]dispatch.Input, error) {
	var inputs []dispatch.Input
	seen := map[string]bool{}
	add := func(path string) {
		path = filepath.Clean(path)
		if !seen[path] {
			seen[path] = true
			inputs = append(inputs, dispatch.Input{Path: path})
		}
	}
	for _, path := range paths {
		info, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		switch {
		case info.Mode().IsRegular():
			add(path)
		case info.IsDir():
			err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.Type().IsRegular() {
					add(p)
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		}
	}
	return inputs, nil
}

// dispatchAll dispatches inputs through stages, workers at once on processing's pool, with an
// in-memory index and a working directory under out, removed once the dispatch is done, and
// writes one line to w as each input's own event, its last, arrives. An error that stops one
// input's dispatch is that input's line, and the others go on; an error that stops the whole
// dispatch is returned.
//
// This is the run's minimal dispatch: it keeps no output and no record, which the run directory
// will.
func dispatchAll(ctx context.Context, w io.Writer, set dispatch.Stages, out string, inputs []dispatch.Input, workers int) (err error) {
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	work, err := os.MkdirTemp(out, ".work-")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(work)) }()
	d, err := dispatch.New(dispatch.Config{Stages: set, Index: dispatch.NewMemoryIndex(), Dir: work})
	if err != nil {
		return err
	}

	var started sync.Map // an input's path to when the pool took it
	for pe, err := range d.Pool(ctx, timed(inputs, &started), workers) {
		if err != nil && pe.Input.Path == "" {
			return err
		}
		ev := pe.Event
		if err == nil && (ev.Output != nil || ev.Part.Depth > 0) {
			continue
		}
		var took time.Duration
		if t, ok := started.Load(pe.Input.Path); ok {
			took = time.Since(t.(time.Time))
		}
		var line string
		if err != nil {
			line = fmt.Sprintf("%s  -  error: %v  %s\n", pe.Input.Path, err, took.Round(time.Millisecond))
		} else {
			outcome := string(ev.Outcome)
			if ev.Err != nil {
				outcome += ": " + ev.Err.Error()
			}
			line = fmt.Sprintf("%s  %s  %s  outputs %d  children %d  %s\n",
				pe.Input.Path, ev.Part.Route, outcome, ev.Outputs, ev.Children, took.Round(time.Millisecond))
		}
		// A terminal that can't be written to stops the run, and the loop's return stops the pool.
		if _, err := io.WriteString(w, line); err != nil {
			return err
		}
	}
	return nil
}

// timed yields inputs, storing in started when the pool asked for each, so an input's time runs
// from then to its last event; it counts any wait for a free worker.
func timed(inputs []dispatch.Input, started *sync.Map) iter.Seq[dispatch.Input] {
	return func(yield func(dispatch.Input) bool) {
		for _, in := range inputs {
			started.Store(in.Path, time.Now())
			if !yield(in) {
				return
			}
		}
	}
}
