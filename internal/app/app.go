package app

import (
	"context"

	"github.com/standards-lab/go-cli-sdk"
	"github.com/standards-lab/go-core/graph"
	"github.com/standards-lab/go-core/lifecycle"

	"github.com/JaimeStill/spike-ai-pipeline/internal/stages"
)

// App is docproc: its dependency graph and the [Nodes] that describe it, its command tree, and
// the streams it reads from and reports to.
type App struct {
	graph   *graph.Graph
	nodes   Nodes
	root    *cli.Command
	streams cli.Streams
}

// New describes the graph, layer by layer, lowest first, builds the command tree, and returns
// the App. It is cold: it constructs nothing, reads no configuration, looks up no tool, and
// writes nothing until [App.Run].
func New(streams cli.Streams) *App {
	a := &App{
		graph: graph.New(),
		root: &cli.Command{
			Name:    "docproc",
			Summary: "docproc runs processing's stages over a set of inputs.",
		},
		streams: streams,
	}
	defineInfrastructure(a.graph, &a.nodes)
	defineDomain(a.graph, &a.nodes)
	a.root.Add(versionCommand())
	mountDomain(a.root, &a.nodes)
	return a
}

// Run dispatches args, the program arguments without the program name, and returns the process
// exit code. The dispatcher builds the nodes the selected command's path declares with Use from
// the App's graph, with the lifecycle configuration node, and shuts what it built down before Run
// returns. An App runs one command at a time, since a graph.Graph is not safe for concurrent use.
func (a *App) Run(ctx context.Context, args []string) int {
	return cli.Run(ctx, a.root, args, a.streams, cli.WithGraph(a.graph, a.nodes.LifecycleConfig))
}

// Nodes is the App's graph, one handle per node. Each layer file's define function fills its own
// part of one Nodes value. Each field's node is named as the dispatcher labels its errors.
type Nodes struct {
	LifecycleConfig *graph.Node[lifecycle.Config] // "lifecycle config"
	ToolLookup      *graph.Node[stages.Lookup]    // "tool lookup"
}

// Graph returns the graph a's commands are built from, as [New] described it. The program itself
// never calls it; it is published so a caller can, before the App runs, observe what a run builds
// with graph.Graph.Observe, or Replace a node's constructor with a substitute, such as a tool
// lookup that finds what a test decides.
func (a *App) Graph() *graph.Graph { return a.graph }

// Nodes returns a handle on each of a's graph nodes, for a caller's Replace or for a command it
// adds over them.
func (a *App) Nodes() Nodes { return a.nodes }

// Root returns a's root command. A command a caller Adds to it before the App runs is part of the
// App's tree from then on.
func (a *App) Root() *cli.Command { return a.root }
