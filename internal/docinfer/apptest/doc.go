// Package apptest holds the fixtures docinfer's tests share for running it over the composition
// [app.App] publishes, and for standing in for the model servers it sends to. A test builds its
// App with [app.New] and applies the fixtures to it before it runs; a constructor a fixture does
// not swap stays the production one.
//
// [Builds] records the name of every node a run's Build begins, and [Halt] fails the lifecycle
// configuration node. [Serve] starts a fake endpoint, an httptest server under /v1, answering as
// its [Fake] says: a /models listing of its [Model]s, or none, and embeddings, each input's
// [Vector], rejecting the requests its Reject picks. Its [Endpoint.URL] is the base URL a test's
// configuration names, [Endpoint.Requests] what it received, and [Endpoint.Embeddings] each
// embeddings request's model and inputs.
//
// The package exports:
//
//   - [Builds], which returns a Recorder of the nodes the App's runs build
//   - [Halt], which stops the App's runs before anything starts, and [ErrHalted], what such a
//     run fails with
//   - [Serve], which starts an [Endpoint] answering as a [Fake] of [Model]s says, with its
//     [Endpoint.URL], [Endpoint.Requests], each a [Request], and [Endpoint.Embeddings], each an
//     [Embedding]
//   - [Vector], the embedding a Fake gives an input, of [DefaultDims] dimensions by default
package apptest
