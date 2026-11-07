// Package infer is docinfer's run command: docinfer run <docproc-dir> --config <dir> --out <dir>
// [--run <id>]. It reads its configuration from the directory --config names, internal/docinfer/
// config's, then the docproc run, internal/docinfer/source's, the latest in <docproc-dir> or the
// one --run names, refusing a schema other than "1", then checks each capability's model against
// its endpoint's listing, internal/docinfer/models', and prints the table of capabilities, before
// the inference runs. A configuration error, a schema refused, and a model the listing doesn't
// name each exit 1 before any request but the listings'.
//
// --run is one run's id: a path or a dot entry is a usage error, as docproc report has it.
// --config and --out are required.
//
// The inference is execute's. It begins a run in --out, internal/docinfer/outdir's directory.
// When vision is configured, it sends each distinct image of the docproc run, every image output
// line's stored file, whatever its part came to, once, by internal/docinfer/vision, as a data URL
// beside the prompt, its reply capped at max_tokens, unless an earlier run holds its result, keyed
// by the image's content, the model, and the prompt and cap; each text is stored whole in --out
// and its result held as it completes. A reply cut at the cap is recorded incomplete and its text
// kept; an image the endpoint refuses is recorded, not fatal, and held when the refusal is a 4xx.
// Unconfigured, vision skips the images, counting them.
//
// When transcription is configured, it sends each distinct audio window of the docproc run, every
// audio output line's stored file, an audio part's or a video's, whatever its part came to, once,
// by internal/docinfer/transcribe, to the OpenAI-compatible transcription route with the prompt
// and language, each when set, unless an earlier run holds its result, keyed by the window's
// content, the model, and the prompt and language; each transcript is stored whole in --out and
// its result held as it completes. A window the endpoint refuses is recorded, not fatal, and held
// when the refusal is a 4xx. Each part's windows, by stage, are then joined in plan order, the
// index each window's meta gives, by processing's audio.Join: a refused window, or one the plan
// counts but the record lacks, joins as an empty transcript. The joined transcript is stored in
// --out and recorded in the manifest with each seam's merge, matched or not. Unconfigured,
// transcription skips the windows, counting them.
//
// It then embeds every text of the docproc run: each text and Markdown output line's stored file,
// whatever its part came to, never a record, an image, or an audio window directly, each image
// output line's vision text, each part's joined transcript, its lines carrying the part's first
// window as their output, and one metadata text per input with file metadata, its file fields
// and content type as readable lines, an input whose dispatch stopped with an error among them.
// Each distinct text is cut by internal/docinfer/chunk to the configured bound, and each distinct
// chunk is embedded once, by internal/docinfer/embed, unless an earlier run holds its result,
// keyed by its content, the model, the bound, and the chunker's version; each result is held as
// it completes, so a run stopped part way resumes from what it stored. The vision and
// transcription results are held apart from the chunk settings, so a changed bound cuts and
// embeds again and sends no image and no window.
// The run writes a chunk line per chunk per occurrence, the same content under two inputs two
// lines, one vector, and its manifest, with the embedding space and its failures, each image or
// window output line whose image or window the endpoint refused, which have no chunk line.
//
// As each input's work ends, the terminal prints its line: its path, its image and audio window
// output lines, its chunk lines, its failures, refused images and windows and rejected chunks,
// and how long after the run began its work ended. The capabilities run one after another over
// every input, so an input's work ends with its last chunk's embedding, at once when none of its
// chunks is sent. A closing summary then counts each capability's requests and failures, a
// skipped capability saying so, and the run writes its report, internal/docinfer/report's page,
// runs/<run id>/report.html, and prints its path.
//
// The package exports:
//
//   - [Commands], which returns the run command over the HTTP client node it declares
package infer
