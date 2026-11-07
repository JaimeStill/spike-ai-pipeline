package app

import (
	"os/exec"

	"github.com/standards-lab/go-core/graph"
	"github.com/standards-lab/go-core/lifecycle"

	"github.com/JaimeStill/spike-ai-pipeline/internal/stages"
)

// envPrefix is the prefix of every environment variable docproc reads its configuration from:
// go-core's config convention composes the rest, such as DOCPROC_SHUTDOWN_TIMEOUT.
const envPrefix = "DOCPROC"

// defineInfrastructure defines the infrastructure nodes on g into n: the lifecycle
// configuration, and the tool lookup the stages' tools are found through. It constructs nothing.
func defineInfrastructure(g *graph.Graph, n *Nodes) {
	n.LifecycleConfig = g.Define("lifecycle config", newLifecycleConfig)
	n.ToolLookup = g.Define("tool lookup", newToolLookup)
}

// newLifecycleConfig reads the lifecycle configuration, the shutdown budget, from its default and
// DOCPROC_SHUTDOWN_TIMEOUT, and validates it. The dispatcher adds this node to every Build, so a
// command that declares any node reads it.
func newLifecycleConfig(*graph.Scope) (lifecycle.Config, error) {
	var cfg lifecycle.Config
	err := cfg.Finalize(envPrefix)
	return cfg, err
}

// newToolLookup returns the host's PATH lookup, exec.LookPath, through which every tool a stage
// needs is found, the docker CLI included: under mise, PATH holds the root's pins, office, mail,
// onenote, and 7zz. A test replaces this node to decide what the host holds.
func newToolLookup(*graph.Scope) (stages.Lookup, error) {
	return exec.LookPath, nil
}
