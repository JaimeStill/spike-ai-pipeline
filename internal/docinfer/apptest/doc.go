// Package apptest holds the fixtures docinfer's tests share for running it over the composition
// [app.App] publishes, and for standing in for the model servers it sends to. A test builds its
// App with [app.New] and applies the fixtures to it before it runs; a constructor a fixture does
// not swap stays the production one.
//
// [Builds] records the name of every node a run's Build begins, and [Halt] fails the lifecycle
// configuration node. [Serve] starts a fake endpoint, an httptest server under /v1, answering as
// its [Fake] says: a /models listing of its [Model]s, or none; embeddings, each input's
// [Vector], rejecting the requests its Reject picks; and vision, an image and a prompt in a
// generation request of any of the three formats, Chat Completions, Messages, and Open Responses,
// each answered in its format with [VisionText] of its image, or as its Vision says; and
// transcription, a multipart upload to /v1/audio/transcriptions, answered with [TranscriptText]
// of its file, or as its Transcribe says. Its
// [Endpoint.URL] is the base URL a test's configuration names, [Endpoint.Requests] what it
// received, [Endpoint.Embeddings] each embeddings request's model and inputs, and
// [Endpoint.Visions] each vision request's image, prompt, and cap, and [Endpoint.Transcriptions]
// each transcription request's file, model, prompt, and language.
//
// The package exports:
//
//   - [Builds], which returns a Recorder of the nodes the App's runs build
//   - [Halt], which stops the App's runs before anything starts, and [ErrHalted], what such a
//     run fails with
//   - [Serve], which starts an [Endpoint] answering as a [Fake] of [Model]s says, with its
//     [Endpoint.URL], [Endpoint.Requests], each a [Request], [Endpoint.Embeddings], each an
//     [Embedding], [Endpoint.Visions], each a [VisionRequest], answered by a [VisionReply], and
//     [Endpoint.Transcriptions], each a [TranscriptionRequest], answered by a
//     [TranscriptionReply]
//   - [Vector], the embedding a Fake gives an input, of [DefaultDims] dimensions by default, and
//     [VisionText], the text it gives an image by default, and [TranscriptText], the transcript
//     it gives a window by default
package apptest
