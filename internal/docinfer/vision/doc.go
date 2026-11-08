// Package vision sends images to a vision model through the client's Generate: each image a
// request, as a data URL beside the prompt, its reply capped at a configured number of output
// tokens, at a configured reasoning effort, in the endpoint's wire format, at most a configured
// number of requests in flight, bounded by a buffered channel. A reply cut at the cap is a result
// marked incomplete, its text kept; an image the endpoint refuses, a reply that isn't 2xx, is a
// result recording the reply, not an error; any other failure stops the run. Each result is
// handed to the caller as its request completes, so the caller can store it then, and a run that
// stops part way loses only what was in flight.
//
// The package exports:
//
//   - [Describer], whose [Describer.Run] sends images, each to a [Result]
//   - [Image], an image to send: its file and media type
package vision
