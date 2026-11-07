// Package app is docinfer's composition root. [New] describes the program's dependencies as one
// go-core graph and builds the command tree on go-cli-sdk over it; [App.Run] dispatches the
// process arguments over that tree. Each architecture layer has one file:
//
//   - app.go: [App], [New], and [Nodes], which call the layer files' define and mount functions
//     in layer order, lowest first;
//   - infrastructure.go: defineInfrastructure, the lifecycle configuration, read from
//     DOCINFER_SHUTDOWN_TIMEOUT, and the HTTP client every request is sent through;
//   - domain.go: defineDomain and mountDomain, which mounts the run command and the report
//     command;
//   - version.go: the version command, which declares no nodes.
//
// A command declares the nodes it needs with Use; the dispatcher builds them only when the
// command runs, so help, a usage error, or version reads no configuration and sends no request.
// The run command reads docinfer's own configuration, internal/docinfer/config's, from the
// directory its --config names, as it runs.
//
// The App publishes its composition: [App.Graph], [App.Nodes], and [App.Root] return the graph,
// a handle on each of its nodes, and the command tree, for a caller that observes what a run
// builds or Replaces a node, such as the HTTP client, before the App runs.
package app
