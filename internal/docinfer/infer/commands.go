package infer

import (
	"context"
	"net/http"

	"github.com/standards-lab/go-cli-sdk"
	"github.com/standards-lab/go-core/graph"

	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/config"
	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/models"
	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/source"
	"github.com/JaimeStill/spike-ai-pipeline/internal/rundir"
)

// Commands returns the run command, which sends its requests through the HTTP client hc names.
func Commands(hc *graph.Node[*http.Client]) []*cli.Command {
	return []*cli.Command{command(hc)}
}

// command returns the run command: docinfer run <docproc-dir> --config <dir> --out <dir>
// [--run <id>].
func command(hc *graph.Node[*http.Client]) *cli.Command {
	var out, configDir, id string
	cmd := &cli.Command{
		Name:     "run",
		Summary:  "Send a docproc run's outputs to the configured models",
		Synopsis: "<docproc-dir>",
		Args:     cli.ExactArgs(1),
		// A run id names one directory under runs/, so an id that would reach outside it, a
		// path or a dot entry, is refused before anything is read, as docproc report refuses it.
		Validate: func(*cli.Invocation) error {
			if out == "" {
				return cli.Usagef("--out names no directory")
			}
			if configDir == "" {
				return cli.Usagef("--config names no directory")
			}
			if id != "" && !rundir.ValidID(id) {
				return cli.Usagef("--run %q is no run id", id)
			}
			return nil
		},
		Run: func(ctx context.Context, inv *cli.Invocation) error {
			cfg, err := config.Load(configDir)
			if err != nil {
				return err
			}
			run, err := source.Open(inv.Args[0], id)
			if err != nil {
				return err
			}
			client := inv.Get(hc)
			report, err := models.Check(ctx, client, cfg)
			if werr := report.Write(inv.Stdout); werr != nil {
				return werr
			}
			if err != nil {
				return err
			}
			return execute(ctx, inv.Stdout, job{cfg: cfg, run: run, http: client, out: out})
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&configDir, "config", "", "the `dir`ectory docinfer's configuration is read from")
	flags.StringVar(&out, "out", "", "the `dir`ectory the run writes in, made when it doesn't exist")
	flags.StringVar(&id, "run", "", "the `id` of the docproc run to read, the latest run when empty")
	cmd.Require("config", "out")
	return cmd.Use(hc)
}
