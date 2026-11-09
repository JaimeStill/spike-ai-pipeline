package app

import (
	"github.com/standards-lab/go-cli-sdk"
	"github.com/standards-lab/go-core/graph"

	"github.com/JaimeStill/spike-ai-pipeline/internal/docproc/report"
	"github.com/JaimeStill/spike-ai-pipeline/internal/docproc/run"
)

// defineDomain defines the domain services on g into n. docproc has none of its own yet: its run
// command builds its stages over the tool lookup, an infrastructure node.
func defineDomain(*graph.Graph, *Nodes) {}

// mountDomain mounts the run command at root, over the tool lookup it declares, and the report
// command, which declares no nodes.
func mountDomain(root *cli.Command, n *Nodes) {
	root.Add(run.Commands(n.ToolLookup)...)
	root.Add(report.Commands()...)
}
