package report

import (
	"context"
	"fmt"

	"github.com/standards-lab/go-cli-sdk"
)

// Commands returns the report command. It declares no nodes: it reads a run directory and
// writes one file in it.
func Commands() []*cli.Command {
	return []*cli.Command{command()}
}

// command returns the report command: docproc report <run-dir> [--run <id>].
func command() *cli.Command {
	var id string
	cmd := &cli.Command{
		Name:     "report",
		Summary:  "Render a run's HTML report again, from its manifest and record",
		Synopsis: "<run-dir>",
		Args:     cli.ExactArgs(1),
		Run: func(_ context.Context, inv *cli.Invocation) error {
			rel, err := Write(inv.Args[0], id)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(inv.Stdout, rel)
			return err
		},
	}
	cmd.Flags().StringVar(&id, "run", "", "the `id` of the run to report, the latest run when empty")
	return cmd
}
