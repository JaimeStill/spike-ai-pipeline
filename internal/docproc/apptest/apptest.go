package apptest

import (
	"errors"
	"slices"
	"sync"

	"github.com/standards-lab/go-core/graph"
	"github.com/standards-lab/go-core/lifecycle"

	"github.com/JaimeStill/spike-ai-pipeline/internal/docproc/app"
	"github.com/JaimeStill/spike-ai-pipeline/internal/docproc/stages"
)

// Recorder records events in order. It is safe for concurrent use. The zero value is ready to
// use.
type Recorder struct {
	mu     sync.Mutex
	events []string
}

// Record appends event.
func (r *Recorder) Record(event string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
}

// Log returns a copy of the events recorded so far.
func (r *Recorder) Log() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.events)
}

// Builds returns a Recorder of the name of each node a's runs build, in the order each Build
// begins them. A run that builds nothing records nothing. Call it before a runs.
func Builds(a *app.App) *Recorder {
	r := &Recorder{}
	a.Graph().Observe(r.Record)
	return r
}

// ErrHalted is what a run of an App given to [Halt] fails with, labelled "lifecycle config".
var ErrHalted = errors.New("halted")

// Halt makes a's lifecycle configuration node fail with ErrHalted, so a run of a command that
// declares nodes fails its Build before anything starts. A run that builds nothing is
// unaffected. Halt is a Replace, so it panics once a has run a Build.
func Halt(a *app.App) {
	a.Graph().Replace(a.Nodes().LifecycleConfig, func(*graph.Scope) (lifecycle.Config, error) {
		return lifecycle.Config{}, ErrHalted
	})
}

// Tools makes a's runs find every tool through lookup instead of the host's PATH, the docker CLI
// included, so a test decides which tools a run has. Tools is a Replace, so it panics once a has
// run a Build.
func Tools(a *app.App, lookup stages.Lookup) {
	a.Graph().Replace(a.Nodes().ToolLookup, func(*graph.Scope) (stages.Lookup, error) {
		return lookup, nil
	})
}

// ErrNoTool is what [NoTools] fails every lookup with.
var ErrNoTool = errors.New("no such tool")

// NoTools is the lookup of a host with no tool: it finds no program. Given to [Tools], it leaves
// every stage that needs a tool out, and runs nothing on the host.
func NoTools(string) (string, error) { return "", ErrNoTool }
