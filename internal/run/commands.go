package run

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

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
			return execute(ctx, inv.Stdout, set, out, inv.Args, inputs, workers)
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&out, "out", "", "the `dir`ectory the run writes in, made when it doesn't exist")
	cli.StringsVar(flags, &exclude, "exclude", nil, "leave out the stage of `route`, which then has no processor yet")
	flags.IntVar(&workers, "workers", runtime.NumCPU(), "how many inputs are dispatched at once, 1 to this machine's CPUs")
	cmd.Require("out")
	return cmd.Use(lookup)
}

// input is one input to dispatch: its path, and the path argument it came from, the path itself
// for a file argument, each as given and cleaned, and each absolute, as the record writes them.
type input struct {
	path    string
	arg     string
	absPath string
	absArg  string
}

// collect returns the inputs paths name, in order, each once: a regular file as given, and every
// regular file under a directory, in lexical order. No symbolic link is followed, a path given
// among them included, and a symbolic link, like any file that isn't regular, is no input. A
// path that can't be read is an error. A file reached twice is one input, from the argument that
// reached it first.
func collect(paths []string) ([]input, error) {
	var inputs []input
	seen := map[string]bool{}
	add := func(path, arg string) error {
		path = filepath.Clean(path)
		if seen[path] {
			return nil
		}
		seen[path] = true
		absPath, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		absArg, err := filepath.Abs(arg)
		if err != nil {
			return err
		}
		inputs = append(inputs, input{path: path, arg: filepath.Clean(arg), absPath: absPath, absArg: absArg})
		return nil
	}
	for _, arg := range paths {
		info, err := os.Lstat(arg)
		if err != nil {
			return nil, err
		}
		switch {
		case info.Mode().IsRegular():
			if err := add(arg, arg); err != nil {
				return nil, err
			}
		case info.IsDir():
			err := filepath.WalkDir(arg, func(p string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.Type().IsRegular() {
					return add(p, arg)
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
