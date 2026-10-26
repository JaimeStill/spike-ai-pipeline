# goal · experiment.ai.spike-ai-pipeline

- **State:** building
- **Task:** host
- **Branch:** host

## Tasks

1. [ ] host
2. [ ] client-core
3. [ ] client-streaming
4. [ ] client-messages
5. [ ] tools
6. [ ] documents
7. [ ] office-cli
8. [ ] office
9. [ ] audio
10. [ ] streams

## Task brief · host

```
Problem       gb10-2 is reserved for the spike but serves nothing: no
              router, no images, and three of set A's four models
              missing from its store. The client and processing tasks
              need set A serving on gb10-2 and kept current at each
              session's start. The client also needs to know two things
              before it is planned: whether the router serves
              /v1/responses, and whether transcription returns
              timestamps.

Behaviors     1. The root carries spike-model-engines' llama.cpp scripts
                 byte-identical to engines commit 698f75c, with that
                 commit recorded beside them; a diff against the source
                 at that commit is empty.
              2. The router's preset serves set A with spike-model-
                 hosting's startup values: gpt-oss-120b MXFP4 c 131072,
                 gemma-4-26B-A4B Q4_0 (ggml-org) c 32768 with images
                 capped at 512 tokens a batch, gemma-4-E4B Q8_0 c 32768,
                 embeddinggemma-2 Q8_0 c 8192 (embeddings only, mean
                 pooling, no mmproj); 4 slots each, unified KV, each
                 card's sampling. The preset loads nothing at startup.
              3. hf is pinned in a host environment of the root:
                 huggingface_hub 2.3.0, installed through uv 0.13.5,
                 with no root. The hermetic check never installs it.
              4. Root currency also reports the hf environment and the
                 llama.cpp pin (server-cuda13-b11669 against the build
                 the latest llama.cpp release names) when either
                 trails. Today it exits 0, and the check passes.
              5. gb10-2's model store holds the three missing set A
                 GGUFs and the mmproj files the preset needs, downloaded
                 with hf. No file is left incomplete, and gpt-oss-120b's
                 copy already in the store is reused, not downloaded
                 again.
              6. gb10-2 runs the router container at server-cuda13-
                 b11669 from the root's clone there, publishing port
                 8080 on its tailnet address only.
              7. `mise run host-update`, run on the laptop, drives
                 gb10-2 over Tailscale SSH. When gb10-2 has pending apt
                 upgrades or a stale CDI spec (one naming another driver
                 version, or a library path that is missing), it prints
                 the exact root commands for the architect, exits
                 nonzero, and leaves the router untouched.
              8. Otherwise host-update pulls the root's clone on gb10-2,
                 installs the router or moves it to the pin, reloads the
                 preset, loads each set A model through the router's
                 load API, and waits until all four are loaded, or fails
                 naming the one that didn't load. It prints the build,
                 gb10-2's stack, and the laptop's network path. A second
                 run changes nothing and says so.
              9. `mise run integration` passes against gb10-2. The probe
                 never loads a model itself, and checks: health; all
                 four models loaded; the build at or above the pin; each
                 model's sampling and slots match the preset; chat; a
                 tool call; image description; audio in chat;
                 768-dimension embeddings; transcription of the fixture
                 phrase through /v1/audio/transcriptions.
              10. The probe records whether /v1/responses answers on
                  gpt-oss-120b, and whether transcription returns
                  segment or word timestamps. It logs both and passes
                  either way.
              11. A findings page in the root gives the two results of
                  behavior 10 with the date, the llama.cpp build,
                  gb10-2's stack, the laptop's network path, and every
                  command exactly as run. It records memory readings
                  after set A loads, none gated; agrees with
                  spike-model-engines' gb10-1 measurement of responses
                  and transcription at the same build, or explains the
                  difference; names gb10-2 by short name and carries no
                  tailnet address.
              12. Every root step is the architect's, given as an exact
                  command; none is needed today. The session runs only
                  steps that need no sudo.

Test seams    The router's HTTP API on gb10-2, as the integration-tagged
              probe calls it from the laptop through SPIKE_ROUTER_URL.
              host-update's output and exit status are a second seam.

Slices        1. Serving set A (behaviors 1–6, 12): copy the scripts and
                 record their source, write the set A preset, pin hf and
                 extend currency, clone the root onto gb10-2, download
                 the three models with hf, and install the router. Demo:
                 /models lists set A's four ids, each loads by hand
                 through the load API, and currency and the check pass.
              2. host-update (7, 8, 12): the root-step and CDI preflight,
                 pull, update to the pin, load set A, and wait. Demo: a
                 run from the laptop leaves all four loaded and prints
                 the record; a second run reports nothing changed.
              3. The probe and the finding (9–11): the integration
                 probe across every capability, the two recorded
                 checks, and the findings page. Demo: `mise run
                 integration` passes against gb10-2, and the finding
                 reads as described.

Out of scope  Sizing set A for gb10-2 (spike-model-layout's); swapping
              to Google's QAT 26B-A4B; any client code; vLLM, and the
              vLLM model already in gb10-2's store; changes to
              spike-model-engines or spike-model-hosting; CI; any sudo
              the session runs itself; gb10-1 and the Framework.

Door          two-way: removing the container and images, deleting the
              models with hf, and removing the clone undo everything on
              gb10-2; nothing is released.
```

## Progress

slices 2/3 committed · standards — · spec — · editor —

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

## Pending edits

(none)
