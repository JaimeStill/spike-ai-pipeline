# goal · experiment.ai.spike-ai-pipeline

- **State:** building
- **Task:** client-messages
- **Branch:** client-messages

## Tasks

1. [x] host
2. [x] client-core
3. [x] host-scripts
4. [x] client-streaming
5. [ ] client-messages
6. [ ] client-responses
7. [ ] tools
8. [ ] documents
9. [ ] office-cli
10. [ ] office
11. [ ] audio
12. [ ] streams

## Task brief · experiment.ai.spike-ai-pipeline · client-messages

```
Problem       Evidence 2 needs a second wire format under the same request and response
              model, with no change to callers. The client speaks only Chat Completions.
              llama.cpp b11669's router serves Anthropic Messages at /v1/messages: it rebuilds
              each request as Chat Completions, keeps a fixed list of keys, and drops the rest.
              It streams Anthropic's typed events, with thinking blocks whose signature is
              empty. This task adds the Messages format as an owned adapter on the standard
              library, proves it against set A on gb10-2 with the same test code as Chat
              Completions, and tags client v0.3.0.

Behaviors
  1.  A messages format, built by its constructor and set as Config.Format, posts to
      "messages" below the base URL and sends the header anthropic-version: 2026-10-21. The
      package documentation pins that version. The token goes as Authorization: Bearer, as for
      every format. The base module still imports only the standard library, and the strict
      import rule passes.
  2.  The request carries the model and max_tokens from MaxOutputTokens. A request whose
      MaxOutputTokens is zero is refused at encode, before the token source is called or
      anything is sent. Instructions become the top-level system field. A system or developer
      message stays in its place as a system-role message of text blocks.
  3.  A user message's parts become text and image blocks. An image's data URL becomes a
      base64 source with its media type, and any other URL a url source. An image's detail is
      dropped, and InputAudio is refused at encode. A function_call_output becomes a
      tool_result block in a user message: its output string, or its parts as text blocks.
      Adjacent tool results and user content join one user turn, results first.
  4.  An assistant run (its reasoning, its message, and the function_calls after it) becomes
      one assistant message: thinking blocks, then text blocks, then tool_use blocks. A
      tool_use block's id is the call_id, and its input is the arguments parsed as JSON, with
      empty arguments sent as {}. A refusal part sent back goes as a text block. Reasoning goes
      out through its encrypted_content alone, under the client-core rule for reasoning: this
      format's envelope becomes a thinking block (thinking text and signature) or a
      redacted_thinking block (its data). A reasoning item with no envelope, another format's,
      or a malformed one is dropped and never an error.
  5.  Tools go out as name, description, input_schema, and strict. Tool choice maps auto to
      auto, required to any, none to none, and a named function to {"type":"tool","name"}. An
      allowed_tools choice sends only the allowed tools, with its mode mapped the same way.
      Reasoning effort none sends thinking {"type":"disabled"}. Effort low to xhigh sends
      thinking {"type":"adaptive"} plus output_config.effort of the same name. A JSON-schema
      text format sends output_config.format {type json_schema, schema}, dropping the format's
      name and description. Plain text sends nothing, and verbosity is dropped. The package
      documentation lists every hint it drops.
  6.  A reply decodes into protocol output in block order:
      - A thinking block becomes a Reasoning item: its content the thinking text, and its
        encrypted_content the envelope `spike-ai-pipeline:messages/v1:` + base64 of
        {"thinking","signature"}. Tests pin that literal.
      - A redacted_thinking block becomes a Reasoning item with no content, its envelope
        holding the block's data.
      - Each run of consecutive text blocks becomes one assistant message, one output_text
        part per block.
      - Each tool_use block becomes a function_call: its id the call_id, its input as JSON
        text the arguments.
      Each item's id is "<message id>/<output index>". Only the last item takes the response's
      status, and the items before it are completed. stop_reason maps to the status:
        end_turn, tool_use, stop_sequence     completed
        max_tokens, model_context_window_exceeded   incomplete, max_output_tokens
        refusal                               incomplete, content_filter
        pause_turn, or any other              an error
      Usage: InputTokens is input_tokens plus cache_read_input_tokens and
      cache_creation_input_tokens; OutputTokens is output_tokens; TotalTokens is their sum.
  7.  The native tier merges into the top level of the body. A member that clashes with one
      the typed fields set is refused at encode, as is stream, which is always typed (false
      unstreamed, true streamed). The router keeps only the members it reads, such as
      temperature, top_p, top_k, and chat_template_kwargs. The package documentation says so.
  8.  A non-2xx reply is an *APIError whose message and type are decoded from either body
      shape: Anthropic's {"type":"error","error":{...}} or llama.cpp's
      {"error":{"code","message","type"}}.
  9.  A streamed request becomes the spec's semantic events, opened by response.created
      (from message_start) and closed by one terminal event (at message_stop, by the table in
      behaviour 6):
      - A thinking block becomes a reasoning item with its reasoning_text part. Its thinking
        deltas become reasoning deltas. Its signature delta is kept, and the item's
        output_item.done carries the envelope.
      - A text block becomes a part of the open assistant message, or opens one.
      - A tool_use block becomes a function_call item. Its input_json deltas become argument
        deltas.
      Items and parts may be open at the same time and each closes at its own block's stop,
      since the router closes every block only at the end. message_delta's stop_reason and
      usage join the terminal Response. Ping events and comments change nothing, and an event
      type the decoder doesn't know is ignored. The folded Response equals DecodeResponse's
      for the same reply.
  10. In a stream, an error frame in either shape, recognized from its data alone, ends the
      stream with ErrorEvent and then response.failed, holding the output made before it:
      Anthropic's {"type":"error",...}, or a body with a message whose type isn't a Messages
      event, as llama.cpp's is. A stream that ends before message_stop ends with an error
      matching io.ErrUnexpectedEOF. The Format and StreamDecoder contracts don't change.
  11. Every Generate and GenerateStream scenario runs on gb10-2 through both formats from one
      piece of test code, the format the only difference, and every scenario sets
      MaxOutputTokens. Covered: chat, vision, tools, the reasoning round trip on gpt-oss-120b,
      structured output, streamed text, a streamed function_call, and a cancel mid-stream.
      Recorded but not asserted, passing either way, because the router drops their keys:
      structured output's JSON validity, the named and none tool choices, and reasoning
      effort's effect. Audio input on Messages is proven refused by a unit test.
  12. The module README, the context capability map, the format package documentation, and
      the CHANGELOG describe the Messages format; the CHANGELOG has a v0.3.0 entry.

Test seams    the Format methods called directly in unit tests (EncodeRequest, DecodeResponse,
              EncodeStreamRequest, NewStreamDecoder, DecodeError) over recorded router and
              Anthropic bodies and event sequences; the integration tier driving
              client.Generate and GenerateStream black-box against gb10-2, once per format
Slices        1. Unary text: the constructor, route, version header, max_tokens refusal,
                 Instructions and in-place system messages, text and image parts, audio
                 refused, decoding text replies (stop-reason table, usage, item ids), and both
                 error bodies. The integration tier runs the chat and vision scenarios through
                 both formats on gb10-2. Check passes. Checkpoint.
              2. Tools and reasoning: tools, the tool-choice mapping, allowed_tools narrowing,
                 tool_use and tool_result, the envelope round trip (thinking and
                 redacted_thinking), effort, structured output, and the Native clash rules.
                 The tools, reasoning round-trip, and structured scenarios run through both
                 formats on gb10-2 (the structured, named-choice, and effort results recorded).
                 Check passes. Checkpoint, after a standards-review pass.
              3. Streaming: the stream encoder and the stateful decoder (open parts at the same
                 time, signature folded into the envelope, ping and unknown events, both
                 error-frame shapes, an early end). The streamed text, function_call,
                 reasoning, and cancel scenarios run through both formats on gb10-2. Check
                 passes. Checkpoint.
              4. Every scenario through both formats in one run of the integration tier, with
                 the recorded results written into the run's log. Check passes. Checkpoint.
              5. Release preparation: the CHANGELOG's v0.3.0 entry, the module README, the
                 context capability map, and the package docs (dropped hints, version pin,
                 envelope). Check passes. Checkpoint.
Out of scope  /v1/messages/count_tokens and any token-counting operation; Anthropic's server
              tools, citations, documents, and prompt caching; budget_tokens as a typed field
              (reachable through Native); auth by x-api-key; streamed vision and audio; the
              Open Responses format (client-responses)
Door          one-way for the release: tag v0.3.0 is pushed. The code is two-way: the change
              adds a format and changes no existing contract.
Release       spike-ai-pipeline-client v0.3.0
```

## Progress

- slice 1 (unary text): client e1f4e3d; check passes; integration passes on gb10-2 (chat,
  incomplete, unknown model, vision through both formats). Interim: thinking blocks dropped
  until slice 2; unsupported inputs, Native, and streaming refused as "not yet supported".
  Checkpoint passed.
- slice 2 (tools, reasoning, effort, structured output, native tier): client 13c9fb4; check and
  integration pass (named/allowed choices, structured output, effort recorded on messages).
  Standards review: 45d0615, fb4b277 (test-only fixes). Checkpoint pending.

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

- host-scripts: the root's host/ copies spike-model-engines d285b06 (llama-gb10-native's merge,
  its main's tip), not host-rename's 7cd0ddb, which the roadmap named. Supersedes the host
  decision "copied verbatim from spike-model-engines 698f75c".
- host-scripts: host-update gains a "moved" router state that runs `host/engines llama update`
  to move the preset mount. Rejected: running the update over SSH by hand once.
- host-scripts: docs/findings/host.md's paths are rewritten to host/ with a rename note,
  following spike-model-engines' fe8bbeb.
- host-scripts: only the container set plus llama-release.sh is copied. Rejected: the native
  method's files; gb10-2 serves from the container, which has proven as performant as native.
- host-scripts: a running container at the pin with no preset mount also counts as moved, since
  `host/engines llama update` re-creates it with the mount too.
- host-scripts: the findings' dead-mount entry keeps `engines/`, where that event happened, with
  "(now `host/llama-preset/`)"; the findings' commands are rewritten to host/ per the note.
- host-scripts: clone_router_state sources the copied llama.sh for LLAMA_PRESET_DIR and
  llama_preset_mount, so host-update compares the same strings the update script does.

- client-messages: the Messages format is owned, in the base module as format/messages, on
  encoding/json/v2. Rejected: anthropic-sdk-go v2.4.0, which brings the AWS and Google SDKs,
  OpenTelemetry, and gjson, and retries twice by default through its own transport.
- client-messages: the stream decoder recognizes an error frame from its data alone: Anthropic's
  `{"type":"error",...}`, or a body with a message whose type isn't a Messages event, as llama.cpp
  b11669's `{"code","message","type":"server_error"}` is. Rejected: passing the SSE event name,
  which would change StreamDecoder.Decode.
- client-messages: a Messages request with MaxOutputTokens zero is refused at encode (Anthropic
  requires max_tokens). Rejected: sending none, which the router fills with a 4096 default.
- client-messages: effort none sends thinking disabled; low to xhigh sends adaptive thinking plus
  output_config.effort of the same name. The router ignores both, so integration records the
  effect. Rejected: llama.cpp's chat_template_kwargs.reasoning_effort, reachable through Native.
- client-messages: a JSON-schema text format sends output_config.format (schema only; name and
  description dropped). The router drops it, so integration records the result. Rejected:
  refusing structured output on Messages.
- client-messages: the format sends anthropic-version 2026-10-21, pinned in its package doc.
  Rejected: 2023-06-01.
- client-messages: system and developer messages go in place as system-role messages;
  Instructions are the top-level system field. Rejected: moving them into system.
- client-messages: hints Messages can't express (image detail, verbosity, strict on the router)
  are dropped and listed in the package doc; only InputAudio is refused, and a refusal part goes
  back as text. Rejected: refusing them.
- client-messages: an allowed_tools choice sends only the allowed tools, its mode mapped (auto to
  auto, required to any, none to none). Rejected: refusing it.
- client-messages: stop_reason refusal is incomplete with reason content_filter. Rejected:
  completed with a refusal part.
- client-messages: every Generate and GenerateStream scenario runs through both formats from the
  same test code; structured output, named tool choice, and effort are recorded, not asserted;
  audio is refused in a unit test. Rejected: a Messages-only subset.
- client-messages: BUILD checkpoints after every slice, with a standards-review pass before slice
  2's checkpoint. Rejected: one checkpoint.
- client-messages: the Messages envelope is `spike-ai-pipeline:messages/v1:` + base64 of
  {"thinking","signature"}, or {"data"} for redacted_thinking; tests pin the literal
  (client-core's encrypted_content rule).
- client-messages: Messages item ids are "<message id>/<output index>", as completions derives
  them.
- client-messages: the stream decoder allows items and parts open at the same time, each closing
  at its own block's stop, as llama.cpp b11669 closes every block at the end.
  model_context_window_exceeded maps to incomplete max_output_tokens; pause_turn is an error,
  since the client has no server tools.
- client-messages: /v1/messages/count_tokens is out of scope, with no backlog entry.

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
  v0.3.0:

  > - **`spike-ai-pipeline-client`** gives Go primitives over the engines' native APIs, built as
  >   owned, optimized infrastructure from the best of spike-harness-driver's `model` and tau's
  >   libraries, on the standard library alone. Its protocol is Open Responses (spec 2026-04-24)
  >   in its stateless subset; client extensions, audio input first, fill the spec's gaps and are
  >   removed when it adopts them. Four layers import one way: the `client` transport, with
  >   Generate, GenerateStream, Embed, and Transcribe; `protocol`; the `format` contract; and a
  >   `format/<api>` adapter per wire format, named by endpoint and chosen by typed constructor:
  >   `completions` and `messages` (Anthropic Messages) now, `responses` planned. GenerateStream
  >   yields Open Responses' streaming events, which a `protocol.Accumulator` folds into the
  >   Response Generate returns. Retry is opt-in.
- coordinator · roadmap, the `host-scripts` task: its summary and proof name d285b06
  (llama-gb10-native's merge) where they name host-rename's merge commit, and its summary
  says the container scripts plus llama-release.sh are copied.
- architecture · a retro candidate: "follow the spec wherever possible", the architect's standing
  rule for the client (client-streaming), may belong in STANDARDS.md or the architecture layer as
  a standard.
- coordinator · context/ai-hosting.md, "Set A's embedding model": gb10-2 serves
  granite-embedding-english-r2 BF16 from llama.cpp (spike-ai-pipeline, client-core).
- coordinator · context/ai-hosting.md, "Hosts and access" (optional): llama.cpp b11669's
  `/v1/messages` rebuilds a request as Chat Completions, dropping output_config, named and none
  tool choices, and a tool's strict; its stream error frame carries no `"type":"error"`.
