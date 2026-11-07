// Package transcribe sends audio windows to an audio model through the client's Transcribe: each
// window a request, uploaded as a file to the OpenAI-compatible transcription route whatever
// format the endpoint speaks, with the configured prompt and language when set, at most a
// configured number of requests in flight, bounded by a buffered channel. A window the endpoint
// refuses, a reply that isn't 2xx, is a result recording the reply, not an error; any other
// failure stops the run. Each result is handed to the caller as its request completes, so the
// caller can store it then, and a run that stops part way loses only what was in flight. The
// client never retries a transcription; neither does Run, so a refused window is the caller's
// to send again in a later run.
//
// The package exports:
//
//   - [Transcriber], whose [Transcriber.Run] sends windows, each a file path, to a [Result]
package transcribe
