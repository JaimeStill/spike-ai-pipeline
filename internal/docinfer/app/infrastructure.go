package app

import (
	"net/http"

	"github.com/standards-lab/go-core/graph"
	"github.com/standards-lab/go-core/lifecycle"

	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/config"
)

// defineInfrastructure defines the infrastructure nodes on g into n: the lifecycle
// configuration, and the HTTP client every request is sent through. It constructs nothing.
func defineInfrastructure(g *graph.Graph, n *Nodes) {
	n.LifecycleConfig = g.Define("lifecycle config", newLifecycleConfig)
	n.HTTPClient = g.Define("http client", newHTTPClient)
}

// newLifecycleConfig reads the lifecycle configuration, the shutdown budget, from its default and
// DOCINFER_SHUTDOWN_TIMEOUT, and validates it. The dispatcher adds this node to every Build, so a
// command that declares any node reads it.
func newLifecycleConfig(*graph.Scope) (lifecycle.Config, error) {
	var cfg lifecycle.Config
	err := cfg.Finalize(config.EnvPrefix)
	return cfg, err
}

// newHTTPClient returns the HTTP client every request is sent through, on a transport of its
// own. It sets no timeout: a request's is its context's, since a model's reply can take minutes,
// and http.Client.Timeout would cut it off.
func newHTTPClient(*graph.Scope) (*http.Client, error) {
	return &http.Client{Transport: http.DefaultTransport.(*http.Transport).Clone()}, nil
}
