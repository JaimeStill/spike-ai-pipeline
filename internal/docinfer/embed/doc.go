// Package embed sends texts to an embedding model through the client's Embed: batches of a
// configured size, a request each, at most a configured number in flight, bounded by a buffered
// channel. A batch the endpoint rejects is sent again a text a request, so a text it rejects is
// recorded as rejected and loses no other text its batch held; any other failure stops the run.
// Each result is handed to the caller as its request completes, so the caller can store it then,
// and a run that stops part way loses only what was in flight.
//
// The package exports:
//
//   - [Embedder], whose [Embedder.Run] embeds texts, each to a [Result]
package embed
