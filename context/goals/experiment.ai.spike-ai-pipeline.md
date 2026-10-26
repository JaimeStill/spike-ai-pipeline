# goal · experiment.ai.spike-ai-pipeline

- **State:** building
- **Task:** client-core
- **Branch:** client-core

## Tasks

1. [x] host
2. [ ] client-core
3. [ ] client-streaming
4. [ ] client-messages
5. [ ] client-responses
6. [ ] tools
7. [ ] documents
8. [ ] office-cli
9. [ ] office
10. [ ] audio
11. [ ] streams

## Task brief · client-core

```
Problem       The client is an empty module, and processing, the root, and evidence 1 and 3
              all need a released model client before they can start. This task gives the
              client a stable request and response model built on Open Responses' item model
              (spec 2026-04-24, the stateless subset), Chat Completions as the first wire
              format folded into and out of it, embeddings and transcription beside it, CI,
              and v0.1.0, so the layers above can build on it by tag.

Behaviors
  1.  Building a client from a Config (base URL, query, token source, the caller's
      http.Client) refuses a missing http.Client, a missing or relative base URL, a base
      URL that carries a query or fragment, and a non-http(s) scheme. It makes no request.
      Later changes to the caller's query values don't reach requests.
  2.  Every request goes to the base URL plus its route, with exactly one slash between
      them, and carries the Config's query. The token source is called once per request:
      a non-empty token is sent as a bearer token, an empty one sends no Authorization
      header, and a source error fails the request before anything is sent.
  3.  A non-2xx reply comes back as a typed API error with the status, the endpoint's
      error message, and its type and code when it has them, matchable with errors.As.
      A transport failure or cancelled context comes back wrapped and matchable with
      errors.Is. The library sets no timeout, retry, or limit of its own.
  4.  The model follows Open Responses' item model, stateless subset, using the spec's
      names.
        Request:  model, instructions, input (a list of items), tools, tool_choice,
                  text.format, reasoning.effort, max_output_tokens, and a raw-JSON
                  extension field
        Response: output (a list of items), status, incomplete_details, and usage
                  (input_tokens, output_tokens, total_tokens,
                  output_tokens_details.reasoning_tokens)
      Items:
        message              role user, system, developer, or assistant. User content
                             is input_text, input_image, and the audio extension in
                             behavior 6; assistant content is output_text and refusal.
        function_call        call_id, name, arguments (a JSON string)
        function_call_output call_id, and output as a string or input_text parts
        reasoning            content (reasoning_text parts), summary (summary_text
                             parts), and encrypted_content, opaque and format-owned
      An item's id and status are carried when the format supplies them. Input
      reasoning items stay inside the spec: content is never sent on input. A format
      that must resend reasoning writes what it needs into encrypted_content, as a
      versioned envelope of its own, when it unfolds a reply, and reads it back when it
      folds; an envelope written by another format is dropped, never sent.
  5.  The chat format is a value of the Format interface, chosen by typed constructor at
      the composition root; there is no registry. A request with no model, no input, or
      an item the format can't express is refused before anything is sent.
  6.  Content parts:
        input_text   sent as text; a message that is a single input_text goes as a
                     plain string
        input_image  image_url as a data URL or remote URL, and detail; sent as image_url
        audio        the client extension spike-ai-pipeline:input_audio, holding the clip's
                     bytes and format (wav, mp3). Chat Completions sends it as input_audio.
  7.  Client extensions of the spec. Each one follows the spec's extension rule: its type
      is prefixed with the implementor slug, spike-ai-pipeline. All of them are listed in
      one documented place, each with the spec version it extends and the standard
      feature that would replace it. In this task the list has one entry: audio input,
      extending 2026-04-24, to be replaced when the spec adds an audio input part. The
      caller-facing Go type and constructor of an extension never carry the slug, so
      moving to a standard type later changes only the wire discriminator and the
      format mappings, not callers' code. The list lives in the root package's doc.go.
  8.  The Chat Completions format folds input into messages:
        instructions                  a leading system message
        message                       a message of its role
        a run of reasoning, assistant one assistant message: content from output_text,
          message and function_call   reasoning_content from the reasoning items'
          items                       Chat Completions envelopes, tool_calls from the
                                      calls, each call's id taken from its call_id
        function_call_output          a role tool message, tool_call_id from its call_id
      Unfolding the reply gives output in a fixed order: a reasoning item from
      reasoning_content (as content, and in the format's envelope), then a message item with output_text, then one function_call
      item per tool call.
      finish_reason maps to status:
        stop, tool_calls  completed
        length            incomplete, reason max_output_tokens
        content_filter    incomplete, reason content_filter
      reasoning.effort goes out as reasoning_effort.
  9.  Tools are function tools: name, description, parameters (a JSON Schema), and
      strict. tool_choice is auto, none, required, a named function {type function,
      name}, or allowed_tools {tools, mode}, each sent in its Chat Completions form.
  10. text.format json_schema (name, schema, strict) is sent as response_format
      json_schema. The output_text is the JSON the model produced.
  11. A request's raw-JSON extension field is merged into the top level of the wire body
      (native tier, documented as such). It is separate from the client extensions in
      behavior 7. A key that clashes with a field the typed model sets is refused before
      anything is sent.
  12. Embeddings sit outside the Format interface, on the OpenAI-compatible shape. One or
      more inputs come back as one []float32 per input, in input order, plus usage. The
      wire encoding is base64 little-endian float32, and an embedding whose byte length
      isn't a multiple of 4 is an error.
  13. Transcription sits outside the Format interface. It uploads a clip (any io.Reader,
      a file name, and a model) as multipart without holding the whole clip in memory,
      and returns the text and usage. A context cancelled during the upload stops it.
  14. The base module imports only the standard library, and the check fails on any
      other import.
  15. CI runs the check on every pull request into main and every push to it. A pushed
      v* tag creates the GitHub release from that version's CHANGELOG section. Currency
      reports a GitHub Action pinned below its latest release.
  16. The integration tier passes against set A on gb10-2:
        chat                                        gpt-oss-120b
        a function_call (tool_choice required)      gpt-oss-120b
        a reasoning and function_call round trip,   gpt-oss-120b
          ending completed
        text.format json_schema                     gpt-oss-120b
        input_image description                     gemma-4-26B-A4B
        audio input in chat                         gemma-4-E4B
        transcription                               gemma-4-E4B
        a 768-dimension embedding                   embeddinggemma-2
      It records what a named function and allowed_tools tool_choice return, and passes
      either way: llama.cpp b11669 treats both as auto.

Go Elemental shape  (the standards the implementer can't load, made concrete)
  Tiers       The client presents model inference as a service in two tiers (service-tiers).
              Standard tier: Open Responses' stateless subset, the external standard the
              library takes as its baseline (baseline-standards); nothing of ours hardens into
              it beyond the listed client extensions. Native tier: the request's raw-JSON field,
              named Native, documented as native tier. Each wire format is an adapter beneath
              one interface, Format, defined where it is consumed, in the root package.
  Packages    Four layers, imports one way: protocol ← format ← format/<api>; client
              imports protocol and format; no format imports client.
                client (module root)  transport: Config, Client, Respond, Embed,
                                      Transcribe, TokenSource, APIError
                protocol              Open Responses: Request, Response, items, content
                                      parts, Usage, Status, Validate, the client
                                      extensions and their list
                format                the Format contract converting protocol ⇄ wire,
                                      and what formats share
                format/completions    Chat Completions (/chat/completions, not the
                                      legacy /completions); later format/messages,
                                      format/responses
              The splits are consumption boundaries, not topics: formats depend on the
              protocol alone, callers build requests and items without the transport, and
              format/ grows to three implementations (topology-and-naming). All are packages
              of the base module, none a sub-module (no third-party weight).
              `internal/clienttest` holds the httptest helper shared by test packages.
              Conversion vocabulary: encode (protocol → wire), decode (wire → protocol);
              each format's files are encode.go, decode.go, wire.go (unexported wire
              types).
  Selection   A format is chosen by its typed constructor at the caller's composition root;
              no registry, no registration on import (dependencies, "selected by
              construction").
  Policy      The library ships none: no default model, token budget, timeout, retry, or
              size limit; the caller's http.Client and context carry them (baseline-standards).
  Validation  Config validates once, in New, before any effect; a request validates before
              the token source is called or anything is sent (validation-first). New returns
              an error; it never panics (a library, not a composition root).
  Types       A type is introduced only when it adds something: *http.Client, url.Values,
              jsontext.Value used as-is, no pass-through wrappers. Items and content parts are
              closed sum types (an interface with an unexported method), Go names from the
              spec's (Message, FunctionCall, FunctionCallOutput, Reasoning; InputText,
              InputImage, InputAudio, OutputText, Refusal, ReasoningText, SummaryText). A time
              the library returns is in time.UTC (utc-times).
  Surface     (sketch; names may change in BUILD with the reason recorded)
                client.New(Config) (*Client, error)
                Config{BaseURL string; Query url.Values; Token TokenSource;
                       HTTP *http.Client; Format format.Format}  // Format required
                type TokenSource func(context.Context) (string, error)
                (*Client).Respond(ctx, *protocol.Request) (*protocol.Response, error)
                (*Client).Embed(ctx, EmbeddingRequest) (*Embeddings, error)
                (*Client).Transcribe(ctx, TranscriptionRequest) (*Transcription, error)
                format.Format interface { Route; EncodeRequest; DecodeResponse; DecodeError }
                completions.New() *completions.Format
                *client.APIError   // errors.As
  Docs        One doc.go per API package holding only the package comment, which inventories
              every exported identifier with a one-clause role and lists the client
              extensions; every production symbol carries a doc comment (tests-and-docs).
  Tests       Unit tier: black-box (`client_test`, `protocol_test`, `format_test`,
              `completions_test`), hermetic, against
              an httptest server on a loopback port, asserting the wire request and decoded
              items; export_test.go only for a hook the API can't reach. The extension list is
              held by doc.go and review, not by a test that reads source (such a test is
              structure-sensitive). Integration tier: `-tags integration`, in a root-level
              `integration` package that drives the exported API black-box.
  Checkpoint  BUILD stops once, after slice 2 and a standards-review pass, with a short
              checkpoint brief of the exported surface for the architect's source read; a
              redirect goes to the implementer before slice 3.
  Release     CHANGELOG in Keep-a-Changelog form with a standing [Unreleased] and link
              references; release preparation lands by PR; the tag is pushed only after main's
              CI passes; the release job is serialized per tag (release-and-ci). The ruleset on
              main and Dependabot security updates (no dependabot.yml) are the architect's
              GitHub admin steps.

Test seams    The client's exported API: Config, the client constructor, a request of items
              through a Format value, embeddings, and transcription. Unit tests drive it
              against an httptest server, asserting the request on the wire and the decoded
              items. The integration tier drives the same API against SPIKE_ROUTER_URL.

Slices
  1. CI and the gate: CI pair, CHANGELOG, depguard stdlib-only, currency's action-pin scan.
     Demo: check passes; currency current with workflow pins scanned. (14, 15)
  2. Text chat end to end: Config, client, API error, item model with message items,
     Format interface, Chat Completions fold/unfold for messages and instructions, status
     and usage, raw-JSON extension merging. Demo: integration chat on gpt-oss-120b.
     (1–5, 8, 11, 16)
  3. Image and audio input, audio as the first client extension with the list and its
     test. Demo: integration on gemma-4-26B-A4B and gemma-4-E4B. (6, 7, 16)
  4. Tools: function tools, every tool_choice form, function_call and function_call_output
     items and their folding. Demo: integration function_call on gpt-oss-120b, named and
     allowed_tools recorded. (8, 9, 16)
  5. Reasoning: items unfolded from reasoning_content and folded back; reasoning.effort.
     Demo: integration round trip on gpt-oss-120b ending completed. (4, 8, 16)
  6. Structured output: text.format json_schema. Demo: strict schema on gpt-oss-120b
     decoded into a struct. (10, 16)
  7. Embeddings. Demo: 768-dimension embeddings on embeddinggemma-2, in order. (12, 16)
  8. Transcription. Demo: the phrase clip on gemma-4-E4B, "The access code is 7429."
     (13, 16)
  9. Release v0.1.0: README current, CHANGELOG dated, tag pushed after main's CI passes.
     Demo: the GitHub release, and `go list -m
     github.com/JaimeStill/spike-ai-pipeline-client@v0.1.0` resolves. (15)

Out of scope  Streaming, SSE, semantic events, accumulator, retry (client-streaming);
              Anthropic Messages (client-messages); Open Responses as a wire format
              (client-responses, v0.4.0); server-side state (store, previous_response_id,
              background, item_reference, compaction); input_file, input_video, image
              content in function_call_output; annotations and logprobs; text-to-speech;
              transcription timestamps; the ruleset on main (architect's GitHub admin step);
              a CI integration job.

Door          one-way for the v0.1.0 tag (checksum database pins it; never re-cut).
              Everything else two-way: confined to spike-ai-pipeline-client, nothing
              depends on it until the tag.

Release       v0.1.0 (spike-ai-pipeline-client)
```

## Progress

slices 2/9 committed (caf1fca, f6c9c5b) · checkpoint standards pass (d6dd49a, a86c831), awaiting the architect · standards — · spec — · editor —

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
  decoder written to the full spec. Rejected: tau's goroutine-fed channel.
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
- host: /v1/responses answers on gpt-oss-120b at b11669 (`docs/findings/host.md`); the setup
  rejection of Responses as "router support unverified" is open at the client's plan round.
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
  folds, and drops another format's envelope. Input reasoning items never carry content.
  Rejected: content on input reasoning items as a listed deviation from the spec.
- client-core: max_output_tokens is a typed request field, in the spec's stateless subset.
- client-core: the client follows service-tiers: Open Responses' stateless subset is the
  standard tier, the request's Native raw-JSON field the native tier, each wire format an
  adapter package beneath Format, named for its target API. Root package `client` holds the
  model, Config, Client, Format, errors, embeddings, and transcription; `chatcompletions` is a
  package, not a sub-module. Rejected: splitting by topic; sub-modules without third-party
  weight.
- client-core: Format is a required Config field, chosen once by typed constructor at the
  caller's composition root. Rejected: a Format argument on every Respond call.
- client-core: the integration tier is a root-level `integration` package, `-tags
  integration`, driving the exported API black-box. Rejected: tagged files beside each package.
- client-core: the extension list lives in the root doc.go, held by review. Rejected: a test that
  reads source (structure-sensitive).
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

## Pending edits

- coordinator · context/ai-hosting.md, "Hosts and access": the gb10-2 paragraph says the spike's
  sessions keep gb10-2 current; add that it serves set A from llama.cpp's router at
  server-cuda13-b11669, kept current by the root's `mise run host-update` at each session's
  start, and that a borrowing session restores the pipeline engine by leaving the llama container
  running, since host-update fails on a stopped one rather than starting it.
- coordinator · roadmap: the `host` task's summary says the router loads models on demand; it
  runs --no-models-autoload, and host-update loads set A through the load API.
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
  against the router's `/v1/responses`; tags client v0.4.0); extend evidence 2 in its README and
  the roadmap to a third format, Open Responses, the model's own baseline; and reword the
  `client-core` summary: the stable model is Open Responses' item model, stateless subset.
