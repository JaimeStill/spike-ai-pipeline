# goal · experiment.ai.spike-ai-pipeline

- **State:** building
- **Task:** client-streaming
- **Branch:** client-streaming

## Tasks

1. [x] host
2. [x] client-core
3. [ ] client-streaming
4. [ ] client-messages
5. [ ] client-responses
6. [ ] tools
7. [ ] documents
8. [ ] office-cli
9. [ ] office
10. [ ] audio
11. [ ] streams

## Task brief · client-streaming

```
Problem       v0.1.0 returns a response only once the model has finished, so a caller shows
              nothing for the tens of seconds gpt-oss-120b spends reasoning, can't stop a
              generation part way, and gets no help with an overloaded endpoint. Evidence 1
              needs chat, tools, and the reasoning round trip streamed as well as unstreamed,
              and evidence 6 needs a stream that processing can cancel. This task streams
              Generate as Open Responses' semantic events (spec 2026-04-24). It adds an SSE
              decoder written to the full WHATWG spec on the standard library alone, an
              accumulator that folds the events into the Response Generate would return,
              cancellation, and the opt-in retry policy. Tags client v0.2.0.

Behaviors
  1.  The SSE decoder follows the WHATWG event-stream parsing rules:
        - CR, LF, and CRLF line endings, and a leading BOM stripped
        - comment lines ignored; a field with no colon is a name with empty value
        - one leading space after the colon removed
        - data lines joined with "\n"; event names the type
        - an id holding NUL is ignored; retry is accepted as digits only
        - an event is dispatched at a blank line, never at end of stream without one
        - unknown fields are ignored
      It reads the body as the bytes arrive, holding only the event being built. It never
      reconnects, and it never sends Last-Event-ID.
  2.  The protocol gains Open Responses' streaming events as a closed Event sum type,
      under the spec's names. Each event carries its sequence_number and, where the
      spec gives them, output_index, content_index, and item_id:
        state      response.created, and the terminal response.completed,
                   response.incomplete, and response.failed, each with the Response
        items      response.output_item.added and response.output_item.done
        parts      response.content_part.added and response.content_part.done
        deltas     response.output_text.delta and .done, response.refusal.delta and
                   .done, response.reasoning.delta and .done (reasoning_text),
                   response.function_call_arguments.delta and .done
      Status gains the spec's in_progress. Summary events, annotations, and logprobs
      are outside the subset.
  3.  The Chat Completions format turns its chunks into events in the spec's order. A
      stream opens with response.created and closes with one terminal event, whose
      Response holds the finished output and the usage. Each item it produces gets its
      own events, in v0.1.0's output order: reasoning, then message, then function
      calls. For each item:
        - output_item.added, then content_part.added for each content part
        - that part's deltas and its .done, then content_part.done
        - output_item.done carrying the whole item
      A reasoning item's output_item.done carries the format's encrypted_content
      envelope, as Generate's does. Tool-call fragments are assembled by their index:
      the first fragment carries call_id and name, and later ones add argument text.
      finish_reason maps to status as in v0.1.0. The format treats data [DONE] as the
      end of the stream; a stream that ends after finish_reason without [DONE] is also
      complete.
  4.  The streamed Chat Completions request always sends stream true and
      stream_options.include_usage true. The unstreamed request sends stream false.
      Because stream is a typed field in both encodings, a Native stream or
      stream_options key is refused as a clash, before anything is sent.
  5.  (*Client).GenerateStream(ctx, req) returns an iter.Seq2 of events and errors, and
      it is lazy: nothing is validated, encoded, or sent until the sequence is ranged.
        - A refused request, a token source error, or a non-2xx reply (the same
          *APIError as Generate) is yielded as the sequence's only pair.
        - The request asks Accept text/event-stream.
        - Breaking out of the loop closes the body, and no goroutine outlives the loop.
        - Each range sends a new request; the doc says so.
  6.  An error during the stream ends the sequence with one error pair, and no event
      follows it:
        - a dropped connection, or a stream ending before its terminal event:
          io.ErrUnexpectedEOF, matchable with errors.Is
        - a cancelled or expired context: the context's error, wrapped, matchable
          with errors.Is
        - an error frame from the server (data {"error": {...}}): a typed stream
          error with its message, type, and code, matchable with errors.As
        - a chunk the format can't decode: a decode error
      Cancelling closes the connection, which stops the router's generation.
      llama.cpp's 30-second ping comments are ignored.
  7.  protocol.Accumulator folds events in the caller's own loop. Add(event) takes each
      event in turn and refuses one that doesn't follow the events before it, such as a
      delta for an item that wasn't added. Response() returns the Response folded so far.
      After a terminal event it equals the terminal event's Response, and for the same
      reply it equals what Generate decodes. After a broken stream it holds the
      partial output, with status in_progress.
  8.  Retry is a Config field, unset by default; unset, nothing is retried. The policy
      sets the most attempts, a base delay, and a most delay. New refuses a policy with
      fewer than one attempt, a negative delay, or a base delay above the most delay.
      With a policy:
        - Generate and Embed, and GenerateStream before its first byte, retry 408,
          429, 503, and a connection failure before any response
        - nothing else is retried: never a 500, another status, or a started stream
        - each attempt calls the token source again and resends the same encoded body
        - the wait is exponential backoff with full jitter, capped at the most delay
        - a Retry-After, in seconds or as an HTTP date, replaces the backoff
        - a Retry-After longer than the most delay, or a wait past the context's
          deadline, returns the last error at once
        - a context cancelled while waiting returns the context's error
        - the last attempt's error is returned as v0.1.0 returns it
      Transcribe is never retried, because its clip is a one-shot reader uploaded as it
      is read; its doc says so. Timeouts stay the caller's: the context, and the
      transport's ResponseHeaderTimeout, never http.Client.Timeout. The library sets no
      timeout or limit.
  9.  The base module still imports only the standard library (depguard), and the
      check passes.
  10. The integration tier passes against set A on gb10-2, every v0.1.0 test plus:
        streamed text, its accumulated Response completed with usage   gpt-oss-120b
        a streamed function_call (tool_choice required)                gpt-oss-120b
        a streamed reasoning and function_call round trip, the         gpt-oss-120b
          accumulated output sent back, ending completed
        a context cancelled mid-stream: the context's error, and a      gpt-oss-120b
          partial Response, status in_progress, holding the text so far
      Retry is proven by unit tests only: the router can't answer 408, 429, or 503
      on demand.

Go Elemental shape  (the standards the implementer can't load, made concrete)
  Tiers       Streaming stays in the standard tier: the spec's events under the spec's
              names, nothing of ours added. The native tier is unchanged, apart from the
              stream keys behaviour 4 refuses.
  Packages    The four layers and their import direction stay as they are. The SSE
              decoder is an internal package of the base module, used by the client and
              by no API package; exporting it later only adds to the API. Event and the
              Accumulator live in protocol, since folding the spec's events is the same
              for every format. The chunk-to-event translator lives in
              format/completions, in its decode file, with its chunk wire types beside
              the reply's.
  Contract    Format gains required methods, a stream encoder and a stateful stream
              decoder. Pre-1.0, semver allows the change in a minor version, and the
              CHANGELOG marks it as breaking for a format written outside the module. The
              decoder takes each SSE event's type and data, returns the protocol events it
              makes, and is told when the stream has ended. A new decoder serves each
              stream, so a Format value stays safe for concurrent use.
  Policy      Retry is the caller's policy, set in Config. STANDARDS.md's "ships no policy"
              line, the Config doc, and the README become: the library sets no timeout,
              retry, or limit unless the caller does; retry is opt-in.
  Iteration   iter.Seq2 from the standard library, ranged once per request. No channel and
              no goroutine feeds it: the decoder pulls from the body inside the range
              loop.
  Surface     (sketch; names may change in BUILD with the reason recorded)
                (*Client).GenerateStream(ctx, *protocol.Request)
                    iter.Seq2[protocol.Event, error]
                type protocol.Event interface{ ... }   // closed; spec-named event types
                type protocol.Accumulator struct{ ... }
                (*Accumulator).Add(protocol.Event) error
                (*Accumulator).Response() *protocol.Response
                protocol.StatusInProgress
                Config{ ...; Retry *RetryPolicy }
                RetryPolicy{MaxAttempts int; BaseDelay, MaxDelay time.Duration}
                *client.StreamError   // an error frame; errors.As
                format.Format += EncodeStreamRequest, NewStreamDecoder
  Docs        Every package comment's inventory gains the new identifiers, each with a
              one-clause role. The protocol doc names the streaming subset (behaviour 2)
              and what it leaves out. GenerateStream's doc states its laziness, its error
              pairs, and that each range sends a new request.
  Tests       Unit tests, hermetic:
                - SSE decoder: table tests over each rule in behaviour 1
                - completions: recorded router streams (text, tool call, reasoning,
                  length, error frame) translated to event sequences, and folded to the
                  Response that DecodeResponse gives for the same reply
                - client_test: GenerateStream and retry against an httptest server
                  (early break closes the body; dropped connection; cancel; 408, 429,
                  and 503 retried; 500 and a started stream not retried;
                  Retry-After in both forms)
              The backoff's jitter source is replaced only through export_test.go.
              Integration tier: the same root package, -tags integration.
  Checkpoint  BUILD stops after every slice, once it is committed with the check passing,
              with a short checkpoint brief: the source files the slice changed, the
              exported surface it added or changed, and its demo's result, for the
              architect to read the source. A redirect goes to the implementer before the
              next slice. Slice 2's checkpoint also follows a standards-review pass, since
              the Event, Accumulator, GenerateStream, and Format surface lands there.
  Release     As in v0.1.0: CHANGELOG [v0.2.0] with the Format change marked as
              breaking; the release preparation lands by PR; the tag is pushed only
              after main's CI passes.

Test seams    The client's exported API: GenerateStream ranged with an Accumulator, and the
              Config's retry policy, driven against an httptest server that writes SSE as
              written. completions' stream decoder is driven directly on recorded router
              streams, as its unstreamed decoder is. The integration tier drives the same
              API against SPIKE_ROUTER_URL.

Slices
  1. The SSE decoder: the internal package and its table tests over every parsing rule.
     Demo: the check passes, depguard included. (1, 9)
  2. Streamed text end to end:
       - Event (state, item, part, and the output_text and refusal events), in_progress
       - the Format additions, and stream as a typed field in both encodings
       - completions' translator for messages, finish_reason, and usage
       - lazy GenerateStream, and the Accumulator for messages
     Demo: integration streamed text on gpt-oss-120b, accumulated with usage.
     (2–5, 7, 10)
  3. Tool-call deltas: function_call events, argument fragments assembled by index,
     and their fold. Demo: integration streamed function_call on gpt-oss-120b.
     (2, 3, 7, 10)
  4. Reasoning deltas and the round trip: reasoning events, the envelope on the
     reasoning item's output_item.done, and the accumulated output sent back. Demo:
     integration streamed round trip on gpt-oss-120b ending completed. (2, 3, 7, 10)
  5. Cancellation and failure during the stream: context cancellation, a dropped
     connection, the error frame, ping comments, and the partial Response in
     in_progress. Demo: integration cancel mid-stream on gpt-oss-120b. (6, 7, 10)
  6. Retry: the Config policy and its checks in New; 408, 429, 503, and failure before
     any response; Retry-After; backoff with jitter; Transcribe exempt. Demo: the
     unit tests, against httptest. (8)
  7. Release v0.2.0: README, STANDARDS.md, and the package docs current; CHANGELOG
     dated; the tag pushed after main's CI passes. Demo: the GitHub release, and `go
     list -m github.com/JaimeStill/spike-ai-pipeline-client@v0.2.0` resolves. (9)

Out of scope  Anthropic Messages and its SSE events (client-messages); Open Responses as a
              wire format, decoding the router's /v1/responses stream (client-responses);
              reasoning summary events, output_text annotations, and logprobs; streamed
              vision and audio integration tests (input parts don't change the stream);
              reconnecting or resuming a stream (Last-Event-ID); retrying Transcribe or a
              started stream; streamed embeddings or transcription; an exported SSE
              package; Collect and other helpers over the Accumulator; a CI integration
              job.

Door          one-way for the v0.2.0 tag (the checksum database pins it; never re-cut).
              Everything else two-way: confined to spike-ai-pipeline-client, and v0.1.0's
              consumers move only when they take v0.2.0.

Release       v0.2.0 (spike-ai-pipeline-client)
```

## Decisions

- setup: four repositories: spike-ai-pipeline (root: record, host scripts, consumer check, later
  the capstone), -client (Go), -processing (Go), -office (.NET Open XML CLI).
- setup: Go repositories copy spike-model-hosting's setup (mise go 1.27.3, golangci-lint 2.15.0,
  hermetic check, currency, integration against `SPIKE_ROUTER_URL`, gb10-2's router by default).
- setup: -office pins .NET 10 in mise; its check runs format, build and xUnit v3 tests with
  warnings as errors, and a NativeAOT publish; releases attach linux-x64 and linux-arm64 binaries.
- setup: all four Apache-2.0. The layers tag plain vX.Y.Z with a CHANGELOG and get CI (check on
  pull requests and main, tag-triggered release) in their first task; integration stays local.
  The root has no CI until the capstone. Consumers take layers by tag, never a replace directive.
- setup: references are the coordinator's keys: tau-protocol, tau-format, tau-provider,
  tau-agent, herald, spike-harness-driver, spike-model-hosting, spike-model-engines, go-core,
  go-database, sqlate, go-web-sdk, go-observability. tau-orchestrate excluded: superseded by
  spike-harness-driver's workflow layer.
- setup: gb10-2 serves set A from llama.cpp's router container (server-cuda13-b11669), from
  spike-model-engines' llama.cpp scripts copied into the root with their source commit;
  `mise run host-update` runs at each session's start. gb10-2 serves models only; media tools
  run where processing runs. Host access follows ai-hosting.md, "Hosts and access".
- setup: the client is owned, optimized infrastructure from the best of spike-harness-driver's
  `model` and tau: one standard-library base module, a Format interface per wire format, no
  Provider registry; an endpoint is a Config (base URL, query, token source, caller's
  http.Client). Rejected: copy-and-extend of `model`; tau's global registries and map options.
- setup: client surface: chat with text, image, and audio parts, tools with tool_choice,
  structured output, reasoning returned and sent back; embeddings ([]float32); transcription. No
  text-to-speech: the router has no route for it.
- setup: streaming is an iter.Seq2 of typed deltas with an accumulator, over an in-house SSE
  decoder written to the full spec. Rejected: tau's goroutine-fed channel; a third-party SSE
  library, which the base module's standard-library-only rule forbids (Go's standard library
  has no SSE client).
- setup: retry is opt-in and off by default: 408, 429, 503, and pre-response connection failures
  only, honoring Retry-After, never a 500 or a started stream. Timeouts come from the context and
  ResponseHeaderTimeout, never http.Client.Timeout.
- setup: request options are typed fields plus a raw-JSON extension field, documented as native
  tier. Rejected: map[string]any; functional options.
- setup: the second wire format is Anthropic Messages. Rejected: Bedrock Converse (AWS, SigV4);
  OpenAI Responses (router support unverified).
- setup: processing runs mature system tools through adapters it owns
  (standards-lab/context/system-tools.md): an exec runner (binaries, and scripts through their
  interpreter) and a container runner (a pinned image per invocation), packages in the base
  module. Later, not now: an MCP runner (spike-go-harness), a sidecar-service runner (only if cold
  start dominates).
- setup: PDF renderer by bake-off: pdftoppm, pdftocairo, gs, vips, and magick over gs, ranked by
  vision extraction accuracy, then time and memory per page, streaming, license, image size, and
  number of tools. ImageMagick is the image tool and an image-correction stage, per request.
- setup: audio through ffmpeg and ffprobe: windows cut at silences under 30 s, transcribed
  concurrently, joined by aligning words at the seams.
- setup: office documents through the Open XML SDK as a NativeAOT CLI in -office: a JSON document
  model plus Markdown; no render path, chart data and embedded pictures instead; baseline
  markitdown and xberg. Rejected: Go OOXML libraries (excelize is xlsx-only; unioffice is
  commercial; the docx readers are small or AGPL); LibreOffice as the main office path (heavy).
- setup: legacy .doc, .xls, .ppt are converted to OOXML by LibreOffice headless in a pinned
  container, legacy files only. Rejected: b2xtranslator (unmaintained since 2018); NPOI (paid
  maintenance fee).
- setup: streams cover live or real-time-paced audio and video through ffmpeg, plus chat streams,
  as cancellable events.
- setup: the capstone's events go through go-messaging (outbox per stage transaction, relayed to
  JetStream). The browser's consumption, NATS over WebSocket or a backend relay, is measured and
  decided at the capstone's plan round.
- setup: pgvector on pgvector/pgvector:0.8.7-pg18, a declared native-tier extension; the
  workspace's extension strategy is written from this evidence by a v1.data task the intake
  writes.
- host: Gemma 4 26B-A4B is set A's ggml-org/gemma-4-26B-A4B-it-GGUF:Q4_0, the id
  spike-model-hosting measured and spike-harness-driver defaults to. Rejected: Google's QAT
  Q4_0, which meets experiment.hosting's quantization floor; a swap is a set change for
  spike-model-benchmarks.
- host: host-update loads set A through the router's load API. The engines llama.cpp scripts
  are copied verbatim from spike-model-engines 698f75c, keeping --no-models-autoload, and the
  preset loads nothing at startup. Rejected: autoloading on a model's first request, which
  would change the copied script.
- host: gb10-2's preset takes spike-model-hosting's set A startup values (c 131072, 32768,
  32768, 8192; 4 slots each; unified KV; card sampling). Rejected: the swept values; sizing
  gb10-2 is spike-model-layout's.
- host: `mise run host-update` runs on the laptop and drives gb10-2 over Tailscale SSH,
  pulling the root's clone there, which the preset is mounted from. Rejected: a task run on
  gb10-2 from its clone.
- host: host-update stops on pending apt upgrades or a stale CDI spec: it prints the exact
  root commands for the architect, exits nonzero, and leaves the router untouched. Rejected:
  reporting and carrying on.
- host: /v1/responses support and transcription timestamps are recorded, not asserted. The
  probe logs both and passes either way, and a findings page in the root carries them with
  the build, gb10-2's stack, and the network path. Rejected: failing the probe on either.
- host: the preset's [*] keeps the engines preset's GB10 keys (n-gpu-layers = all,
  dedup-cache-models) on top of hosting's set A values.
- host: an unmerged branch reaches gb10-2 by `git push ssh://gb10-2/...` into the clone, which
  host-update sets to receive.denyCurrentBranch=updateInstead; never through GitHub before
  acceptance.
- host: host-update counts `apt-get --simulate upgrade` Inst lines as pending upgrades. Rejected:
  `apt list --upgradable`, which lists upgrades phasing defers.
- host: a stopped llama container fails host-update instead of being started: a borrowing session
  may hold it.
- host: host-update reloads the preset when its sha256 differs from the one recorded in the
  clone's untracked `.state/preset.sha256`.
- host: the probe's tool call uses Chat Completions with tool_choice required.
- host: findings live under `docs/findings/`.
- host: /v1/responses answers on gpt-oss-120b at b11669 (`docs/findings/host.md`), which reopened
  the setup rejection of Responses as "router support unverified". client-core settled it:
  Anthropic Messages stays the second format, and Open Responses becomes the third,
  client-responses.
- host: transcription returns no timestamps at b11669: only response_format json, and
  timestamp_granularities is ignored; audio seams align on the overlapping words alone.
- host: gb10-2's spike-model-engines clone is removed; spike-ai-pipeline's clone is gb10-2's
  only engine source, and a borrow clones what it needs.

- client-core: the second wire format stays Anthropic Messages. Rejected: OpenAI Responses as the
  second format; the router turns it into chat completions, so it proves nothing on set A that
  Chat Completions doesn't.
- client-core: reasoning is an Open Responses reasoning item (reasoning_text content,
  summary_text summary, opaque format-owned encrypted_content), ordered in output beside message
  and function_call items. Rejected: one reasoning string field on the message.
- client-core: embeddings and transcription sit outside the Format interface, on the one
  OpenAI-compatible shape; Format covers chat only. Rejected: Format methods every format must
  implement.
- client-core: tool_choice takes every Open Responses form (auto, none, required, named function,
  allowed_tools); integration records the named forms and passes either way, since llama.cpp
  b11669 treats both as auto. Rejected: leaving the named forms out.
- client-core: wire types use encoding/json/v2, whose inline option merges the raw-JSON extension
  field. Rejected: encoding/json v1 with a hand-written merge.
- client-core: a strict depguard rule holds the base module to the standard library, so the
  check and CI prove evidence 3. Rejected: discipline alone.
- client-core: the stable model is Open Responses' item model, stateless subset, pinned to spec
  2026-04-24, using its names; each format folds items into its wire shape and back. Rejected: a
  model of the client's own; server-side state (store, previous_response_id, background).
- client-core: Open Responses as a wire format is its own task, client-responses, after
  client-messages (client v0.4.0, proven against the router's /v1/responses). Rejected: folding
  it into client-core or client-messages.
- client-core: the client extends the spec only where it lacks a feature: slug-prefixed types
  (spike-ai-pipeline:), listed in one place with the spec version and the standard feature that
  would replace each; caller Go types carry no slug, so adopting the standard changes only the
  wire mapping and removes the extension. First: spike-ai-pipeline:input_audio. Rejected:
  unprefixed custom types; slugs exposed to callers.
- client-core: reasoning goes back on input through encrypted_content: each format writes a
  versioned envelope of its own there when it unfolds a reply (Chat Completions: the
  reasoning_content text; later Anthropic: thinking text and signature), reads it back when it
  folds, and drops another format's envelope. A format never sends an input reasoning item's
  content: it ignores it, as the spec's input reasoning item has none. Rejected: content on
  input reasoning items as a listed deviation from the spec.
- client-core: max_output_tokens is a typed request field, in the spec's stateless subset.
- client-core: the client follows service-tiers: Open Responses' stateless subset is the
  standard tier, the request's Native raw-JSON field the native tier, and each wire format an
  adapter package beneath Format, named for its target API, not a sub-module. Rejected: splitting
  by topic; sub-modules without third-party weight. The package layout first planned here, one
  root package `client` holding the model, Config, Client, Format, errors, embeddings, and
  transcription, with a `chatcompletions` package, is superseded by the four layers
  (checkpoint, below).
- client-core: Format is a required Config field, chosen once by typed constructor at the
  caller's composition root. Rejected: a Format argument on every Generate call.
- client-core: the integration tier is a root-level `integration` package, `-tags
  integration`, driving the exported API black-box. Rejected: tagged files beside each package.
- client-core: the extension list lives in one doc.go, held by review: the root's at first,
  `protocol/doc.go` since the four layers. Rejected: a test that reads source
  (structure-sensitive).
- client-core: BUILD checkpoints once, after slice 2, for the architect's read of the Go source.
  Rejected: running every slice to the session brief.
- client-core (checkpoint): items and content parts stay value types; a *Message is refused at
  run time, documented and tested. Rejected: pointer receivers.
- client-core (checkpoint): Format.DecodeError returns message, type, and code as three strings;
  the Client owns Status and Body. Rejected: a result struct or an *APIError.
- client-core (checkpoint): four layers, imports one way: client (transport), protocol (Open
  Responses), format (the contract and shared parts), format/<api> (each implementation). The
  splits are consumption boundaries. Supersedes the single root package. Rejected: one root
  package mixing the spec, the transport, and the contract.
- client-core (checkpoint): the Open Responses package is `protocol`, as tau's is, leaving
  `responses` to the Open Responses wire format. Rejected: responses; openresponses.
- client-core (checkpoint): format packages are named for their endpoints: completions,
  messages, responses. OpenAI's API is covered by completions and responses, with OpenAI-only
  items reached natively; no `openai` package. Rejected: chatcompletions (long); chat
  (fits Messages equally).
- client-core (checkpoint): conversion is encode (protocol → wire) and decode (wire →
  protocol), each format in encode.go, decode.go, and wire.go. Rejected: fold and unfold.
- client-core (checkpoint): the Client's protocol method is Generate, a capability verb beside
  Embed and Transcribe, covering every protocol request (text, vision, audio input, tools,
  reasoning, structured output); its streamed counterpart in client-streaming is
  GenerateStream. Vision has no method of its own: an image is an InputImage part. Rejected:
  Respond (the model responds, not the client); Send (names transport, not capability); Chat
  (undersells vision and structured extraction).
- client-core (checkpoint): the protocol's unit stays Item, the spec's term; Input and Output
  name its direction, and one value moves from Output to the next Input. Item's doc comment says
  so. Rejected: a rename (Turn, Event, Entry, Element, Unit).
- client-core: max_output_tokens goes out as max_completion_tokens (llama.cpp b11669 aliases
  it; OpenAI deprecated max_tokens).
- client-core: Reasoning has no Status (the spec's reasoning item has none); Effort takes the
  spec's enum (none, low, medium, high, xhigh); the request carries Effort only, since Chat
  Completions has no form for the spec's reasoning summary. Reasoning with no assistant message
  or call after it is dropped: llama.cpp refuses an assistant message of reasoning alone. Its
  first clause is superseded by client-streaming.
- client-core: the Chat Completions envelope is `spike-ai-pipeline:completions/v1:` + base64 of
  {"reasoning_content"}; a layout change takes a new prefix, and tests pin the literal.
- client-core: Text carries the spec's PlainTextFormat and Verbosity beside JSONSchemaFormat
  (both in the spec's TextParam, both expressible in Chat Completions); json_object, absent from
  the spec's request formats, is reached through Native.
- client-core: EmbeddingRequest has Native, with the same validation and clash check as the
  protocol's; empty input strings are sent, as llama.cpp embeds them.
- client-core: the OpenAI-style error body decodes in format.DecodeOpenAIError, shared by Chat
  Completions and Embed/Transcribe; one Native validation and clash encoding live in
  internal/native, which protocol, the formats, and client share.
- client-core: TranscriptionRequest types Language and Prompt (llama.cpp honors both; Prompt
  replaces its instruction); the file part goes as application/octet-stream, llama.cpp reading
  the format from the bytes.
- client-core (v0.1.0 surface): Embed and Transcribe take *EmbeddingRequest and
  *TranscriptionRequest, refusing nil, as Generate takes *protocol.Request. Rejected: values.
- client-core (v0.1.0 surface): usage types stay one per endpoint, in its own names
  (protocol.Usage, EmbeddingUsage, TranscriptionUsage). Rejected: protocol.Usage for
  transcription.
- client-core (v0.1.0 surface): TranscriptionRequest has Native url.Values, sent as extra form
  fields, a clash with a typed field refused before sending. Rejected: no native tier.
- client-core (v0.1.0 surface): completions' unit tests call EncodeRequest and DecodeResponse
  directly; client_test drives a format through the transport. Rejected: entering through
  client.Generate.
- client-core (v0.1.0 surface): Native clashes are caught at encode, before anything is sent;
  Validate's doc says it doesn't check them. Rejected: a protocol-level field list.
- client-core: set A's embedding model is granite-embedding-english-r2
  (`mrutkows/granite-embedding-english-r2-GGUF:BF16`), replacing EmbeddingGemma 2, which vLLM
  can't serve on GB10 (coordinator ai-hosting.md, "Set A's embedding model"); the swap lands in
  client-core, so v0.1.0's evidence is on the set A that stays. Supersedes EmbeddingGemma 2 as
  the host task staged and loaded it. Rejected: a root task of its own after client-core.
- client-core: granite's gb10-2 section is c 32768 with 4 slots (each slot its trained 8192),
  b = ub = 8192, embeddings = true, no pooling key (its GGUF declares CLS). Rejected: c 8192
  (2048 per slot).
- client-core: host-update restarts a running llama container whose preset mount is dead (the
  container sees no /etc/llama-preset/models.ini because a delivery recreated
  engines/llama-preset/), then reloads; it still never starts a stopped container. The copied
  engines scripts stay untouched. Rejected: recording the gap for a later host task; mounting
  the clone's engines/ (an edit to a copied script).
- client-core: host-update's load loop treats a /models poll that fails (refused, non-2xx, not
  the JSON expected) as not ready yet and keeps polling to its deadline, and retries a failed
  load request; only a listing that lacks a preset model, a failed load, or the deadline fails
  the run. Without it the first run after a restart exited 1 on a refused poll. Rejected:
  leaving the restart path to fail its first run.
- client-streaming: streamed events are Open Responses' semantic streaming events (spec
  2026-04-24) under the spec's names, a closed protocol.Event type; Chat Completions turns its
  chunks into them, and client-responses will decode them almost as-is. Rejected: the client's
  own short deltas (text, reasoning, tool-call fragment, done).
- client-streaming: Chat Completions emits the spec's full per-item sequence (output_item and
  content_part added/done around each part's deltas), opened by response.created and closed by
  one terminal event. Rejected: emitting deltas alone.
- client-streaming: streaming is required of every Format: a stream encoder and a stateful
  stream decoder, a change v0.2.0 marks as breaking for a format written outside the module.
  Rejected: an optional streaming interface found by type assertion.
- client-streaming: GenerateStream(ctx, req) returns iter.Seq2[protocol.Event, error] and is
  lazy: it sends when ranged, a refused request, token error, or non-2xx reply is its only
  pair, breaking out closes the body, and each range sends a new request. Rejected: an eager
  (iter.Seq2, error) that sends at the call.
- client-streaming: protocol.Accumulator is a value the caller feeds in its own loop, Add(event)
  then Response(). Rejected: a Collect function that consumes the whole sequence.
- client-streaming: a stream that breaks mid-response leaves the Accumulator's partial Response,
  status in_progress (the spec's, added to Status); an error frame is a typed stream error with
  message, type, and code. Rejected: no Response after a broken stream. Its error-frame clause is
  superseded by the error event and response.failed (below).
- client-streaming: the streamed Chat Completions request always sends
  stream_options.include_usage, so the accumulated Response carries usage; llama.cpp sends none
  without it. Rejected: leaving it to Native.
- client-streaming: retry is a Config field, unset meaning off, applied by the Client to each
  attempt, calling the token source again and resending the same encoded body. Rejected: an
  http.RoundTripper the caller wraps around its transport (resends a stale token, depends on
  GetBody).
- client-streaming: the retry policy is a fixed struct (most attempts, base delay, most delay)
  with exponential backoff and full jitter, checked in New. Rejected: a caller function deciding
  each delay (invites retrying a 500).
- client-streaming: a Retry-After longer than the policy's most delay, or a wait past the
  context's deadline, returns the last error at once. Rejected: waiting the full Retry-After.
- client-streaming: Transcribe is never retried; its clip is a one-shot reader uploaded as it
  is read. Generate, GenerateStream before its first byte, and Embed retry. Rejected: retrying
  when the reader is an io.Seeker.
- client-streaming: the integration tier streams text, a function_call, and the reasoning and
  function_call round trip on gpt-oss-120b, and cancels a stream midway; retry is proven by unit
  tests only, as the router can't answer 408, 429, or 503 on demand. Rejected: streamed vision
  and audio on the Gemma models.
- client-streaming: the SSE decoder is an internal package of the base module. Rejected:
  exporting it for processing and the root (no second consumer; the capstone serves SSE rather
  than reads it).
- client-streaming: stream is a typed field in both Chat Completions encodings (false
  unstreamed, true streamed), so a Native stream or stream_options key is refused as a clash
  before anything is sent; v0.1.0 sent a Native "stream": true and then couldn't decode the
  reply. Rejected: leaving stream to Native.
- client-streaming: BUILD checkpoints after every slice for the architect's read of the source
  it changed (the architect's call at approval, as last session), with a standards-review pass
  before slice 2's. Rejected: one checkpoint after slice 2; running every slice to the session
  brief.
- client-streaming: the client follows the spec wherever it can: the architect's standing rule
  for it, set at slice 1's UTF-8 ruling.
- client-streaming: the SSE decoder decodes each line as UTF-8 with replacement, as the WHATWG
  standard does through the Encoding Standard's decoder: one U+FFFD per maximal subpart. It is
  written by hand, since utf8.DecodeRune gives one U+FFFD per invalid byte and bytes.ToValidUTF8
  one per run of them. Rejected: keeping invalid bytes as received.
- client-streaming: the client has no framing interface. SSE is its only framing, internal to the
  base module, and the Format's stream-decoder contract is framing-neutral. Rejected: a framing
  or stream-reader layer as tau has (protocol/streaming, and the provider's SSE and eventstream
  readers), until a format that isn't SSE arrives.
- client-streaming: StreamDecoder.Decode takes the event's data alone, since every planned format
  carries the event's type in the data (Open Responses: the event field MUST match the body's
  type; Anthropic Messages' bodies carry type; Chat Completions names none). Format stays one
  interface. Rejected: passing the SSE event name; splitting Format into unary and streaming
  interfaces; one encoder with a stream bool.
- client-streaming: the spec requires item ids, which Chat Completions doesn't give, so
  DecodeResponse and the stream decoder derive each as "<reply id>/<output index>". A reply with
  no id gives its items none, and a chunk with another reply's id is refused. A v0.1.0 behaviour
  change: v0.1.0's item ids were empty. Rejected: empty ids; an id of the output index alone,
  which repeats in every reply.
- client-streaming: item status follows the spec: only the last output item takes the response's
  status, and the items before it are completed, in DecodeResponse as in the stream. A v0.1.0
  behaviour change: v0.1.0 gave every message and function_call item the response's status.
- client-streaming: protocol.Reasoning has a Status, following the spec's normative "Required item
  fields" rule (every item MUST include id, type, and status) over its ReasoningBody reference
  schema, which omits it. DecodeResponse sets it, a v0.1.0 behaviour change. Supersedes
  client-core's "Reasoning has no Status".
- client-streaming: a server error frame ends the stream as the spec has it: the error event
  (protocol.ErrorEvent, whose ErrorPayload holds type, code, message, and param), then the
  terminal response.failed. Its Response is StatusFailed, with Response.Error (*ResponseError:
  code and message), the output made before the frame, and the open items in_progress; no error
  pair is yielded. A broken stream (a transport failure, an ended context, an undecodable chunk,
  or no terminal event) ends with one error pair, and the Accumulator holds an in_progress
  Response. Amends behaviours 6 and 7. Rejected: one *StreamError error pair, as slice 5 first
  built it.
- client-streaming: completions assembles tool-call fragments by their index in any order, which
  is what Chat Completions' index is for, and Open Responses has no rule that one item finishes
  before the next opens. The calls close at the finish, in index order, and the Accumulator
  places each item at its output_index. Amends behaviour 3. Rejected: calls one after another,
  each closing when the next begins (earlier dispatch, but stricter than the wire format).
- client-streaming: a request that fails before it is sent (the token source, building the
  request) is never retried. attempt splits into request, which calls the token source and builds
  the http.Request, and do, which sends it; send returns request's error before classifying it. A
  fix from the spec review.
- client-streaming: retry counts as a connection failure before any response a *net.OpError
  (refused dial, reset), io.EOF or io.ErrUnexpectedEOF before the reply's header, and a DNS
  failure the resolver calls temporary; never a context error, a token source error,
  ResponseHeaderTimeout, a refused TLS certificate, or an error of the caller's RoundTripper.
  Known gap: net/http's unexported error for a server closing an idle connection isn't matched,
  so it isn't retried.
- client-streaming: the integration tier is v0.1.0's, integration/ (package integration_test,
  -tags integration), where the brief's Tests line says "the same root package".

## Pending edits

- coordinator · context/ai-hosting.md, "Hosts and access": the gb10-2 paragraph says the spike's
  sessions keep gb10-2 current; add that it serves set A from llama.cpp's router at
  server-cuda13-b11669, kept current by the root's `mise run host-update` at each session's
  start, and that a borrowing session restores the pipeline engine by leaving the llama container
  running, since host-update fails on a stopped one rather than starting it.
- coordinator · roadmap, the `host` task's summary: the router runs --no-models-autoload, and
  host-update loads set A through the load API, where the summary says it loads models on demand;
  and the staged GGUFs name granite-embedding-english-r2 BF16 where they name embeddinggemma-2
  Q8_0.
- coordinator · context/ai-strategy.md, the `spike-ai-pipeline` bullet (capstone): pipeline
  storage for AI-assisted processing and semantic search. Source blobs stay in blob storage; the
  database holds what the stages derive from them, each stage's output a versioned artifact
  (source blob hash → extraction → chunk → embedding) keyed by its input and configuration hash
  (model, revision, prompt, parameters, chunker version), so a stage reruns only when its input or
  configuration changes. Embeddings live in registered embedding spaces (model, revision, dims,
  metric, normalization, prompt template, chunker), vectors partitioned by space with one partial
  ANN index each. A model change adds a space and re-embeds the stored chunks (blue/green, switched
  after a recall evaluation), a chunker change re-chunks, and only an extractor change reruns
  inference. Translation between spaces is limited to truncation and quantization within one
  model; a learned mapping only bridges a migration. Settled at the capstone's plan round.
- coordinator · roadmap: add task `client-responses` to experiment.ai.spike-ai-pipeline after
  `client-messages` ("The Open Responses format": the Open Responses wire format, near identity
  over the client's Open Responses-based model, with its semantic streaming events, proven
  against the router's `/v1/responses`; tags client v0.4.0), and extend evidence 2 in its README
  and the roadmap to a third format, Open Responses, the model's own baseline.
- coordinator · roadmap, the `client-core` task: its summary says the stable model is Open
  Responses' item model, stateless subset, and its repos gain `spike-ai-pipeline`, where set A's
  embedding model was swapped to granite-embedding-english-r2 in the preset and probe.
- coordinator · roadmap, the spike's summary: the sentence saying set A's embedding model "is to
  change" to granite-embedding-english-r2 says it changed.
- coordinator · context/ai-strategy.md, the `spike-ai-pipeline-client` bullet, rewritten to
  v0.2.0:

  > - **`spike-ai-pipeline-client`** gives Go primitives over the engines' native APIs, built as
  >   owned, optimized infrastructure from the best of spike-harness-driver's `model` and tau's
  >   libraries, on the standard library alone. Its protocol is Open Responses (spec 2026-04-24)
  >   in its stateless subset; client extensions, audio input first, fill the spec's gaps and are
  >   removed when it adopts them. Four layers import one way: the `client` transport, with
  >   Generate, GenerateStream, Embed, and Transcribe; `protocol`; the `format` contract; and a
  >   `format/<api>` adapter per wire format, named by endpoint and chosen by typed constructor:
  >   `completions` now, `messages` (Anthropic Messages) and `responses` planned. GenerateStream
  >   yields Open Responses' streaming events, which a `protocol.Accumulator` folds into the
  >   Response Generate returns. Retry is opt-in.
- spike-ai-pipeline · this record's Tasks: the roadmap's `host-scripts` task (after
  `client-core`, waiting on spike-model-engines' `host-rename`) is missing; it joins Tasks at the
  next plan or start, so the record and the roadmap agree.
- architecture · a retro candidate: "follow the spec wherever possible", the architect's standing
  rule for the client (client-streaming), may belong in STANDARDS.md or the architecture layer as
  a standard.
- coordinator · context/ai-hosting.md, "Set A's embedding model": gb10-2 serves
  granite-embedding-english-r2 BF16 from llama.cpp (spike-ai-pipeline, client-core).
