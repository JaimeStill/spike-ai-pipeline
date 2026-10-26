# goal · experiment.ai.spike-ai-pipeline

- **State:** idle
- **Task:** none
- **Branch:** none

## Tasks

1. [x] host
2. [ ] client-core
3. [ ] client-streaming
4. [ ] client-messages
5. [ ] tools
6. [ ] documents
7. [ ] office-cli
8. [ ] office
9. [ ] audio
10. [ ] streams

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

## Pending edits

- coordinator · context/ai-hosting.md, "Hosts and access": the gb10-2 paragraph says the spike's
  sessions keep gb10-2 current; add that it serves set A from llama.cpp's router at
  server-cuda13-b11669, kept current by the root's `mise run host-update` at each session's
  start, and that a borrowing session restores the pipeline engine by leaving the llama container
  running, since host-update fails on a stopped one rather than starting it.
- coordinator · roadmap: the `host` task's summary says the router loads models on demand; it
  runs --no-models-autoload, and host-update loads set A through the load API.
