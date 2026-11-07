package app

import (
	"github.com/standards-lab/go-cli-sdk"
	"github.com/standards-lab/go-core/graph"

	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/infer"
)

// defineDomain defines the domain services on g into n. docinfer has none of its own yet: its
// run command reads its configuration itself, from the directory --config names, and sends its
// requests through the HTTP client, an infrastructure node.
func defineDomain(*graph.Graph, *Nodes) {}

// mountDomain mounts the run command at root, over the HTTP client it declares.
func mountDomain(root *cli.Command, n *Nodes) {
	root.Add(infer.Commands(n.HTTPClient)...)
}
