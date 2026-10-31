# goal · experiment.ai.spike-ai-pipeline

- **State:** brief ready
- **Task:** office-cli
- **Branch:** office-cli

## Tasks

1. [x] host
2. [x] client-core
3. [x] host-scripts
4. [x] client-streaming
5. [x] client-messages
6. [x] client-responses
7. [x] tools
8. [x] documents
9. [ ] office-cli
10. [ ] office
11. [ ] audio
12. [ ] streams

## Task brief · experiment.ai.spike-ai-pipeline · office-cli

```
Problem       The office layer is a scaffold that answers only --version, so the spike has no
              way to read docx, xlsx, or pptx. Evidence 12 asks for the Open XML CLI's
              structure fidelity, memory, cold start, and binary size against markitdown and
              xberg on fixed files, with chart data and embedded media in place of page
              images. The later `office` task's processing adapter needs a released binary
              with a versioned output contract. Processing has no document model to match:
              the documents task released page images only. A NativeAOT publish on Open XML
              SDK 3.5.1 builds with no warnings at 28.5 MB, and it read all three formats in
              about 15 ms and 42 MB peak. GitHub's hosted arm64 runners are free on this
              public repository. This task adds the extraction, its Markdown rendering,
              committed fixtures with exact ground truth, CI on both architectures, a release
              that attaches both binaries, and the baseline. It tags office v0.1.0.

Behaviors
  1.  The CLI is one NativeAOT binary named spike-ai-pipeline-office. It builds on the
      .NET 10 base library, the Open XML SDK, and System.CommandLine, and on nothing else;
      Markdig is a test-only dependency. The publish for linux-x64 and for linux-arm64
      gives no AOT or trim warning, and any suppression carries a comment saying why it is
      safe. --version prints the version from the assembly.
  2.  The CLI takes one input path, --format json|markdown (json when omitted), and
      --media <dir>. It writes the chosen rendering to stdout and nothing else there.
      Usage errors print usage to stderr and exit 2. Each refusal happens before any
      output is written, with a message on stderr and a nonzero exit: a missing file, a
      format other than docx, xlsx, or pptx, an encrypted package, or a malformed package.
      With --media, every embedded picture is written to that directory, as stored,
      once per distinct content, named by the SHA-256 of its bytes plus its stored
      extension. Every media block that uses it points at that one file. Without
      --media, nothing is written to disk and the names still appear in the model. A
      linked (external) picture gets a media block with its target and no file; office
      never fetches it.
  3.  The JSON model is one tree for all three formats, and the spike owns its schema. A
      document records its source format and holds ordered blocks: headings with their
      level, paragraphs, nested ordered and unordered lists, and tables of rows and
      cells. A chart block holds its type, title, and series (name, categories, values).
      A media block holds its name relative to the media directory, its content type,
      its alternative text, its source part path, and where it sits (section, sheet, or
      slide). A sheet block and a slide block hold their own blocks.
      The split follows processing's text-first flow: everything but media blocks is the
      document's text, complete without any inference, and media blocks are the only
      content a higher layer may send to vision. A chart with its data in the package
      comes out as data, never as an image. The
      model is serialized by the base library's source-generated JSON. The schema is
      office's output contract: it is versioned with office's tags, and every change to
      it is a CHANGELOG entry.
  4.  docx body: headings, paragraphs, lists, tables, charts, and media come out in
      document order. Tracked changes are read in their accepted state: insertions and a
      move's destination are kept; deletions and a move's source are dropped; a paragraph
      whose paragraph mark was deleted joins the next one. No deleted text appears in any
      output, and moved text appears once.
  5.  docx headers and footers are kept for each section, with their kind (default,
      first, or even). A first-page variant counts only when the section asks for a
      distinct first page, and an even-page variant only when the document turns on
      separate even and odd headers. A section without its own header or footer inherits
      the previous section's. Their content is the same blocks as the body.
  6.  docx footnotes and endnotes: each reference marks its place in the text, and each
      note holds its blocks. Word's separator notes are skipped.
  7.  docx comments: each comment holds its text, author, the anchored range (the
      commented text, located in the body), its replies in order, and its resolved
      state.
  8.  xlsx: sheets come out in workbook order, each as a table of cells. A cell holds a
      typed value (number, text, boolean, or error); a number with a date format becomes
      ISO 8601, and no other number format is applied. A formula cell holds its cached
      value and its formula text; with no cached value, the value is null and the formula
      is still given. A cell that is part of a shared formula, other than the anchor cell
      holding the text, carries the anchor cell's reference in place of formula text.
      Nothing is evaluated. Charts and media on a sheet go with that sheet.
  9.  pptx: slides come out in presentation order, each with its blocks (text, lists,
      tables, charts, media) and its notes.
  10. The Markdown rendering is the same model, in this form:
      - headings as #, lists as nested lists, tables as GFM tables
      - a chart as its title followed by a table of its series
      - media as image links relative to the media directory
      - each sheet and each slide under its own heading, a slide's notes after its
        content
      - each distinct header first and each distinct footer last, once each
      - footnotes and endnotes as GFM footnotes ([^n])
      - comments as footnote-style references ([^cN] at the end of the commented text),
        their definitions after the notes, giving the author, the resolved state, and
        the replies
  11. The fixtures are committed and never generated in the check, in two sets:
      - Generated: a committed Python generator (python-docx, openpyxl, python-pptx, all
        pinned) writes each file from committed sources, and each file is then re-saved
        by LibreOffice headless. The generator writes each file's element-level ground
        truth, covering every behavior above.
      - Real-world: about a dozen files from the permissive corpora (the Open XML SDK's
        test assets, Apache POI's test data, Docling's tests, and ClosedXML's tests),
        with hand-written element-level ground truth. They include a Word-authored docx
        with a chart, since python-docx writes none.
      A sources manifest lists, for each real-world file, its repository, commit, path,
      license, and SHA-256, and the license notices travel with the files; no GPL file
      is committed. The fixture README gives the regenerating commands, the LibreOffice
      version, and each file's SHA-256. The pool is these two sets, and nothing beyond
      them runs, locally or in CI.
  12. The baseline tier: `mise run baseline` runs on the laptop, never in CI, over the
      pool only. It runs office, markitdown, and xberg, the last two pinned in mise and
      covered by currency.
      - Structure fidelity: each tool's Markdown is parsed by Markdig and compared with
        the ground truth element by element: heading levels, list items, table cells,
        sheet names, slide order and notes, chart values, media references, headers and
        footers, notes, and comments. The results are one row per element kind and tool.
      - Memory and time: peak resident memory per file is taken from the finished child
        process's own rusage, or from a cgroup's peak memory. Time per file and cold
        start are measured over repeated runs, giving the first run, the median, and the
        spread.
      - Footprint and license: office's binary size per architecture, markitdown's
        installed footprint, xberg's binary size, and each tool's license.
      - Output: each run's result JSON is committed beside the findings page, one
        directory per run.
  13. A findings page in office holds the fidelity table, memory, time, cold start,
      footprint, and licenses. It explains the rows a baseline tool can't score, such as
      comments, which neither renders in Markdown, and headers and footers, which
      markitdown doesn't read. The root's evidence 12 links the page; that one line
      lands with the goal record's commit in the root.
  14. CI:
      - The check runs on every pull request into main and every push to it, on the
        ubuntu-24.04 and ubuntu-24.04-arm runners, through mise, the same check a
        developer runs.
      - office's marathon config makes the merge wait on the pull request's checks.
      - A ruleset on main requires a pull request and forbids deleting or force-pushing
        the branch.
      - The currency script also reports GitHub Action pins that trail their latest
        release.
  15. Release: a pushed tag vX.Y.Z creates the GitHub release from the tag's CHANGELOG
      section. Each runner builds its architecture's binary natively and attaches it as a
      tar.gz holding the binary and LICENSE, named by binary, tag, and runtime identifier.
      A SHA256SUMS file covers both archives, and each binary runs its --version before it
      is attached. Runs for one tag are serialized. The tag is pushed only after main's
      CI passes.
  16. Office's README, its capability map, and the CHANGELOG (Keep a Changelog, with a
      v0.1.0 entry) describe the CLI surface, the model, the Markdown form, the release
      assets, and these known limits:
      - formulas are not evaluated; number formats other than dates are not applied
      - tracked changes are read accepted only
      - no page images
      - legacy .doc, .xls, and .ppt are not read
      - content outside the behaviors above (for example equations, text boxes, SmartArt,
        and embedded OLE objects) is not extracted

Test seams    The converter's entry point (a document in, the model out) and the Markdown
              renderer over the model, exercised in process against the fixtures and their
              ground truth. The CLI's process contract (arguments, stdout, stderr, exit
              code, the media directory) runs on the published NativeAOT binary inside the
              check. The baseline tier runs the three tools as processes over the pool.

Slices        1. The model and the docx body: the CLI surface and refusals, the binary's
                 name, the one-tree model and its JSON, the docx body with accepted
                 tracked changes, the Markdown renderer for those blocks, and the
                 generator with its first docx fixtures and ground truth. Checkpoint:
                 source files the slice changed, lowest dependency first, each with what
                 to look for, after a standards-review pass.
              2. docx layers: headers and footers per section and kind, footnotes and
                 endnotes, comments with anchors, replies, and resolved state, and docx
                 charts and media through --media, with their Markdown forms and
                 fixtures. Checkpoint as slice 1, without the review pass.
              3. xlsx: sheets, typed cells, ISO dates, cached values and formula text,
                 shared formulas, and sheet charts and media, with fixtures. Checkpoint
                 as slice 2.
              4. pptx: slides in order, notes, tables, charts, and media, with fixtures.
                 Checkpoint as slice 2.
              5. The real-world set: about a dozen files, the sources manifest, the
                 notices, and hand-written ground truth, all passing in the check.
                 Checkpoint as slice 2, plus the manifest for the architect to confirm the
                 files.
              6. CI and release: the check on both runners, the merge waiting on checks,
                 the ruleset on main (applied by the session through `gh api`), Action-pin
                 currency, and the release workflow building both archives and SHA256SUMS.
                 Checkpoint as slice 2.
              7. The baseline: markitdown and xberg pinned in mise, `mise run baseline`,
                 the scorer, the measures, the committed results, the findings page, and
                 the root's evidence 12 link. Checkpoint as slice 2 if the slice changed
                 source, plus the fidelity table.
              8. Release preparation: CHANGELOG v0.1.0 with the known limits, the README,
                 the capability map, and the version. Checkpoint as slice 2.
              Every slice: check passes.

Out of scope  any inference, in the CLI or its tests: office prepares content and never calls
              a model; vision over extracted media, summaries, and embeddings belong to the
              higher layers. Converting media to vision-ready formats (EMF, WMF, TIFF are
              written as stored). Processing's adapter, legacy conversion through LibreOffice,
              and the integration tier that sends office's media through vision on gb10-2
              (the `office` task, by processing's precedent); page images or any render path; formula evaluation; number formats
              other than dates; tracked changes as stored; Docling's schema; corpora or
              sweeps beyond the committed pool; a CI job for the baseline tier; macro-enabled
              and template variants as distinct inputs; the root's consumer check (office
              ships a binary, not a Go module).

Door          one-way for the release: tag v0.1.0 and its assets make the binary's name, the
              asset naming, the CLI surface, and the JSON and Markdown contract public.
              Two-way otherwise: a new, unreleased CLI, plus repository settings (the
              ruleset and the merge wait), which can be changed back.

Release       spike-ai-pipeline-office v0.1.0
```

## Progress

slices 8/8 committed (051b02e, standards 7b9c235, a302ea8, 21940de, b717baf, ef4d7fa, 5b c039f37, 5c 470ed2d, 932c344, 6dc15b2, b8a64f8, 689f271, 5ba731c, 662ccd1, editor 1f209b5; ruleset 26280501 applied; checkpoints 1-4, 5b, 5c, 6, 7 accepted; slice 5 escalation answered; 5b, 5c added) · standards 29c7c2d · spec 2 rounds, gaps closed (662ccd1) · editor done · validated

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
- client-messages (checkpoint): withStatus and itemID stay duplicated in each adapter until a
  third format needs them.
- client-messages (checkpoint): the anthropic-version stays an unexported constant: a new version
  is new code and a new release.
- client-messages: the Messages stream decoder lives in its own stream.go. completions keeps its
  decoder in decode.go, and is not split to match.
- client-messages: a run of reasoning that no assistant message or function_call follows is
  dropped on encode, as completions drops it.
- client-messages: adjacent user messages and function_call_outputs join one user turn, results
  first; two plain user messages join too.
- client-messages: a tool with no parameters sends input_schema `{"type":"object"}`.
- client-messages: streamed tool arguments keep the stream's spacing, since the Accumulator
  requires an item's done arguments to equal its joined deltas.
- client-messages: the stream decoder refuses out-of-place events, such as a second text block
  opening in a message while the first is open, and ignores event and delta types it doesn't know.
- client-messages: streamed usage takes input tokens from message_start and output tokens from
  message_delta.
- client-messages: integration logs every outcome it doesn't assert as one `recorded: <format>
  <scenario>: <outcome>` line: effort and the named, allowed_tools, and none choices on both
  formats, and structured output on messages.

- client-responses: the Open Responses format is owned, in the base module as format/responses, on
  encoding/json/v2. Rejected: openai-go/v3 v3.89.0 (Azure and AWS SDKs, coder/websocket,
  gjson/sjson).
- client-responses: every Responses request is explicitly stateless: store false and include
  ["reasoning.encrypted_content"] are typed members, a Native store or include refused as a clash;
  previous_response_id never sent. Rejected: leaving both to server defaults and Native.
- client-responses: protocol.ReasoningParam gains Summary (auto, concise, detailed), checked by
  Validate; completions and messages drop it and list it. Rejected: leaving it out.
- client-responses: the stream decoder reads response.reasoning_text.delta/.done (OpenAI's names,
  sent by llama.cpp b11669) as the spec's response.reasoning events and emits only the spec's names;
  the package doc lists the aliases. Rejected: ignoring them and taking reasoning whole from
  output_item.done.
- client-responses: Validate refuses MaxOutputTokens 1–15 (the spec's minimum) for every format, a
  v0.4.0 behavior change. Rejected: refusing only in the Responses encoder.
- client-responses: a unit test pins the spike-ai-pipeline:input_audio wire form, and integration
  records the router's answer on gemma-4-E4B, passing either way. Rejected: unit test alone.
- client-responses: BUILD checkpoints only after slices that change source, each giving the changed
  files lowest dependency first with what to look for in each; a standards-review pass before slice
  2's. Rejected: a checkpoint after every slice regardless; one checkpoint.
- client-responses: reasoning goes back in the spec form by default (summary and the server's
  encrypted_content, never content), dropped at encode without server encrypted_content; a
  constructor option also resends the text as content for engines like llama.cpp. Envelope
  `spike-ai-pipeline:responses/v1:` holds the server's encrypted_content and the text; tests pin the
  literal. Integration runs Responses with the option on gb10-2 and asserts the round trip. Amends
  client-core's "a format never sends an input reasoning item's content" with this listed exception.
  Rejected: spec form only (no reasoning on llama.cpp); always resending content.
- client-responses (checkpoint): the Responses option is ResendReasoningText, a verb naming the
  act and what goes back. Rejected: ReasoningAsContent; IncludeReasoning, which reads as the
  spec's include member.
- client-responses (checkpoint): refuseStateful stays: previous_response_id and background are
  refused in Native, since stateful use is a new capability (typed fields, retrieve, poll,
  cancel, and delete operations), never a request field; the package doc says why.
- client-responses (checkpoint): a streamed item the router gives no status (its reasoning, in
  output_item.done and the terminal) is completed at its done, and keeps that status in the
  folded Response, which otherwise equals DecodeResponse of the terminal response; the router's
  unary reply gives the same item completed. Rejected: an empty status, which fails the shared
  streamed reasoning scenario.
- client-responses: the constructor is New(Options), an options struct as client.Config is.
  Rejected: functional options.
- client-responses: protocol exports MinMaxOutputTokens (16), and completions' package doc lists
  the hints it drops, as messages' does.
- client-responses: internal/native gains Has, the Native member lookup completions and responses
  share.
- client-responses: DecodeResponse refuses a unary reply whose status is in_progress, queued, or
  missing, since a stateless client can't fetch the rest (seen by the architect at slice 2's
  checkpoint). Amends behavior 7's "status taken as given".
- client-responses: a function_call with empty arguments goes out as "{}".
- client-responses: the envelope's content is an array of strings; both members are always
  present, and an envelope missing either is malformed, so dropped.
- client-responses: with ResendReasoningText, a reasoning item whose envelope holds server
  encrypted_content but no text goes out without content.
- client-responses: an allowed_tools choice sends every tool and lets the choice narrow them,
  where messages sends only the allowed tools.
- client-responses: two router quirks are documented in the package doc, not worked around: a
  reasoning item after an assistant turn replaces that turn's reasoning, and of consecutive
  reasoning items only the last reaches the model.
- client-responses: the stream decoder refuses a done event whose text or arguments differ from
  the joined deltas, a part opened out of order within its item, and an output_item.added that
  already carries content. It reads ErrorPayload.Param, and the response.failed after an error
  frame has zero usage.
- client-responses: the envelope prefixes stay off protocol's extension list: encrypted_content is
  opaque to the spec, so an envelope is no wire type.
- client-responses: the CHANGELOG marks the 1–15 refusal **Breaking:**, the only marker the
  CHANGELOG uses.
- client-responses: integration's per-format pending skips were scaffolding, removed once every
  format passed; slice 5 was a run only, with no commit.

- tools: the contract's unit is a Tool bound to one program at construction, built by a
  runner's constructor; Run takes arguments, a working directory, and streams. Rejected: a Runner
  shared by many programs, taking the program on every call.
- tools: one `tool` package holds the contract and both runners. Rejected: a sub-package per
  runner (no dependency weight; go-elemental splits by growth or weight, never topic alone).
- tools: the tool contract stays in spike-ai-pipeline-processing. Rejected: a repository of its
  own (the architect, at the brief's approval).
- tools: the container runner runs the docker CLI through the exec runner. Rejected:
  moby/moby/client v0.6.2 (pre-1.0, 15 direct requirements including OpenTelemetry); podman's
  bindings (about 182 requirements).
- tools: a container mounts the invocation's working directory at the same absolute path and
  works in it. Rejected: a fixed /work with arguments rewritten.
- tools: cancelling an exec run sends SIGTERM to the process group, then SIGKILL once the KillAfter
  period passes (Cmd.Cancel, WaitDelay, Setpgid); the package builds on Linux only. Rejected:
  SIGKILL to the process alone (CommandContext's default).
- tools: a version check is a minimum, dotted numbers captured by the tool's pattern and compared
  as numbers in-house. Rejected: an exact version; golang.org/x/mod/semver (needs a v prefix and
  strict semver, which 7.1.2-34 and n9.0.2 don't follow).
- tools: an image reference must carry a digest, its tag kept for readers. Rejected: tags alone.
- tools: a missing image is pulled at construction, and every run passes --pull never. Rejected:
  refusing and naming the pull command.
- tools: each container run is locked down (--rm, --network none, --user as the caller's uid:gid,
  --cap-drop ALL); an image's settings can relax the network and add capabilities. Rejected: the
  runtime's defaults.
- tools: evidence 14's tool is ffprobe, on the host (n9.0.2) and in
  linuxserver/ffmpeg:9.0.2-cli-ls87 by digest; a sh script covers interpreters in unit tests.
  Rejected: ImageMagick (dpokidov/imagemagick, single-maintainer, last updated 2026-01-30); a
  Python script.
- tools: a strict depguard rule holds processing to the standard library, itself, and
  spike-ai-pipeline-client, written as the client's. Rejected: discipline alone.
- tools: docker is the only container runtime. Rejected: any docker-compatible CLI (rootless
  podman breaks file ownership without --userns keep-id).
- tools: every container runs with --init. Rejected: the tool as PID 1, which ignores SIGTERM
  without a handler, so every cancel would end in SIGKILL.
- tools: currency reports image pins now, a newer tag of the pinned tag's shape counting as
  trailing, a rebuild of the same version (ls88 after ls87) included. Rejected: only a newer tool
  version; leaving the scan to the office task.
- tools: a failed run's error carries the program's name, its exit code, and the last 4 KiB of
  stderr when the caller passed no stderr writer; a cancelled run returns an error matching the
  context's. Rejected: the name and exit code alone.
- tools: container construction also runs the tool's version check inside the image. Rejected:
  trusting the digest alone.
- tools: the KillAfter period is 10 s when unset (docker stop's default). Rejected: a value every
  constructor requires.
- tools: the SIGTERM-to-SIGKILL period is `Spec.KillAfter`, "the KillAfter period", named after
  go-core's `reactor.CancelAfter`. Rejected: Grace, which go-web-sdk's `NewTransfer` and the
  timeouts principle use for a transfer deadline's allowance (architect, slice 2 checkpoint).
- tools: a cancelled run's process group gets SIGKILL while its leader is still unreaped, so the
  group's ID can't have been reused. Rejected: killing the group after the reap, a near-zero
  reuse window (architect, slice 2 checkpoint).
- tools: a cancelled container run goes around the exec runner's cancellation of the docker CLI:
  the runner runs docker stop --time=<the KillAfter period, rounded up to whole seconds> on the
  run's name under a fresh bounded context, retried each second while the container may not exist
  yet, then ends the CLI, waits for it, and always runs docker rm --force. The CLI's signal
  forwarding is never relied on, since it doesn't forward SIGKILL.
- tools: a container run whose context ends as the CLI exits nonzero on its own returns an error
  matching the context's, as an exec run does; a run that succeeded returns nil though its context
  ended afterward.
- tools: a cancelled run never waits on the caller's stdin: execRun owns an os.Pipe for a stdin
  that isn't an *os.File and stops waiting for its copy on cancel; after Run returns, the copy can
  make one more Read of the caller's reader. Rejected: requiring an *os.File stdin in the docs
  (architect, slice 3 checkpoint).
- tools: once the program has exited normally, Run waits for the stdin copy at most the run's
  KillAfter period: the spec's through the exec runner, and the docker CLI's 2 s through the
  container runner.
- tools: Tool is a struct holding an unexported run function that each runner supplies. Rejected:
  a Tool interface.
- tools: a Spec's VersionPattern matches the version command's stdout and stderr together, and
  its first capture group is the version.
- tools: ParseVersion accepts dotted decimal numbers only; a prefix (n9.0.2) or suffix (-34) is
  an error, so the pattern captures only the numbers.
- tools: Tool.Name is the program's base name; NewScript's Tool takes the script's base name, and
  its version is the interpreter's.
- tools: an empty Invocation.Dir is the process's working directory; the container runner makes
  the directory absolute for its mount and refuses one holding a colon, which would split
  docker's --volume.
- tools: the container runner passes every valued docker flag as --flag=value, so no caller
  value reads as a flag.
- tools: a container run's name is tool- and 26 random base32 digits, lowercased (130 bits).
- tools: the docker CLI's spec is fixed in the package: the server's version
  (`docker version --format={{.Server.Version}}`), minimum 20.10, the first with run's --pull.
  The caller chooses only the CLI's name or path.
- tools: exit codes 125 (docker's own failure) and 126 and 127 (an entrypoint docker can't run)
  are docker's, and come back as an *ExitError named for the spec's program.
- tools: currency's image-pin scan reads repo:tag@sha256: literals in the Go source, joining
  literals split by +, and compares each pin's tag with the image's Docker Hub tags of the same
  shape (each dotted number read as N), newest by sort -V, among the 100 most recently updated.
  A reference naming another registry is skipped.
- tools: currency uses only mise's declared tools, gh, and base utilities. Rejected: perl, which
  mise.toml doesn't declare.
- tools: a known limit, out of scope: if processing's own process is SIGKILLed, a running
  container runs on until its tool exits, and --rm removes it afterward.

- documents: whole blob is the page stream's per-page vision calls followed by one final
  inference over the joined page results, returning one response, with no page cap (round 3,
  amending round 1). Rejected: every page image in one vision request; the whole blob and page
  stream as the PDF's input side (bytes versus an io.Reader). Superseded by M: the whole blob is
  test code.
- documents: processing requires spike-ai-pipeline-client v0.4.0 by tag and calls vision
  through a one-method Generate interface that *client.Client satisfies. Rejected: stopping at
  page images and leaving vision to the root. Superseded by M: processing stops at prepared pages,
  and the client is a test-only requirement.
- documents: fixtures are generated from committed typst sources, about six clean letter pages
  and about four ImageMagick-degraded copies, ground truth from the source. Rejected:
  public-domain government documents scored against pdftotext's text layer. Amended after slice 2:
  nine harsher degraded pages and a scanned set.
- documents: extraction accuracy is character error rate on whitespace-collapsed text, written
  in test code, with word error rate beside it. Rejected: word error rate alone; field-level
  scores.
- documents: scored vision runs are greedy (temperature 0 through Native), 3 runs per cell,
  spread recorded; a difference inside the spread is a tie. Rejected: the card's temperature 1
  with more repeats.
- documents: only the bake-off's winner is released; the four other renderers stay as -tags
  bakeoff test code so the bake-off reruns. Rejected: all five adapters exported.
- documents: tool's Invocation gains an optional Usage pointer that the exec runner fills (peak
  RSS, user and system CPU) from the process's rusage; the container runner reports none.
  Rejected: measuring memory outside the contract; a Tool.RunUsage method. Amended at slice 3's
  checkpoint: no peak RSS.
- documents: every tool runs on the host through the exec runner. Rejected:
  penpotapp/imagemagick:7.1.2-32 for corrections (no gs, so no PDF); minidocks images (latest
  tag only); building images.
- documents: the corrections are the scan set, typed per-request options in a fixed order:
  -deskew 40%, -despeckle, -contrast-stretch 0.5%x0.5%, -unsharp 0x1, -colorspace Gray;
  evidence 11 scores each alone, all five, and none. Rejected: herald's brightness-contrast and
  saturation set.
- documents: the bake-off is its own tier, mise run bakeoff (-tags bakeoff), each run capped at
  10 minutes; integration stays a smoke test. Rejected: the bake-off inside mise run
  integration.
- documents: the release is processing v0.2.0, additive only.
- documents: BUILD checkpoints after each slice that changes source, with a standards-review
  pass before slice 2's, and the ranking table for the architect's confirmation before the
  winner is wired in. Rejected: source checkpoints alone.
- documents: one package, document, holds the page-renderer interface, the released renderer,
  the corrections, and vision. Rejected: a pdf package beside an imaging package (a split by
  topic alone). Amended by M: vision is test code.
- documents: the page-renderer interface is per page, Count and Render(ctx, pdf path, page,
  dpi, w), one process per page. Rejected: a per-document renderer yielding pages from one
  process.
- documents: vision is a Vision value built from a Generate, a Renderer, and an optional
  corrections stage, with Document and Pages methods. Rejected: package functions taking all
  three on every call. Superseded by M: Vision is the test harness; Preparer is released.
- documents: a failed page in Pages yields its error and the stream carries on; a failed spool
  or count ends it. Rejected: ending the stream at the first failed page.
- documents: Pages yields in page order, holding later pages finished early. Rejected:
  completion order.
- documents: Pages' concurrency is per request, zero meaning 1; the bake-off and integration
  set 4. Rejected: a default of 4 (set A's slots, which the package can't know).
- documents: no page cap in the whole blob; the server's request-body limit is what matters
  (the architect, round 2 Q8). Rejected: a built-in 16-page cap; a caller's MaxPages.
- documents: the default DPI is 150, per request; Gemma 4 26B-A4B's ~477 image tokens per
  letter page saturate from ~110 dpi, so 150 costs no context. Rejected: 110 dpi; a pixel
  budget.
- documents: the bake-off's transcription prompt asks for plain text in reading order,
  exported as a constant; the instructions stay a per-request value. Rejected: Markdown. Amended
  by M: the constant is unexported test code.
- documents: Document's final step is a caller's protocol.Request template (model,
  instructions, text format, output limit), required. Rejected: a prompt string sent to the
  vision model. Amended after slice 5 (a digest, not a rewrite) and by M (test code).
- documents: the final request carries one user message with one text part per page, each
  beginning "Page N of M:". Rejected: one joined text with delimiter lines.
- documents: Document returns every page's result (number, response, error) beside the final
  response. Rejected: the final response alone.
- documents: a page over the server's 100 MiB request-body limit (cpp-httplib 0.61.0 in
  llama.cpp b11669) is sent anyway, and its 413 is that page's *client.APIError; the package
  sets no size limit. Rejected: refusing before sending against a caller's MaxRequestBytes. After
  M, the test harness's (spec review).
- documents: a failed page fails Document before the final step, the result's pages still
  filled in. Rejected: a final step over the pages that succeeded.
- documents: a known limit, out of scope: Document's final step over plain-text transcripts may
  exceed the final model's context on long documents (about 30 to 45 pages at 32768), tuned if
  it bites. After M, the test harness's (spec review).
- documents (slice 2 checkpoint): spool's tests reach it through export_test.go for now; slice 5
  moves them onto Pages and Document and deletes export_test.go. Rejected: spooling only through
  an entry point in slice 2.
- documents (slice 2 checkpoint): Render refuses only a DPI of zero or less and a page below 1,
  running no tool; Pages and Document refuse a page past the last against their own count, and a
  direct Render past the last returns the renderer tool's own error. Amends behavior 3. Rejected:
  CheckRender counting on every Render (one extra process per page, inside the bake-off's time).
- documents: the one-method interface is document.Generator. Amended by M: generator, unexported
  test code.
- documents (fixtures, after slice 2): a scanned fixture set of ~6 pages from herald's
  _project/marked-documents/marked-documents.pdf (Lexmark CX725 scans, image only, already public
  in herald's repository) is copied into processing's testdata, pages picked for legible text,
  ground truth transcribed by the agent with redacted text excluded and confirmed by the
  architect at slice 3's checkpoint; it joins the ranking and evidence 11's matrix, reported
  apart from the synthetic set, whose faults are made harsher across a range of severity.
  Amends round 1 Q3. Rejected: reading the scans from herald by path; the generated set alone.
- documents (slice 3 checkpoint): tool.Usage drops PeakRSS and keeps UserCPU and SystemCPU: Go
  starts a program with a vfork-style clone, so Linux records the parent's RSS in the child's
  maxrss at exec. The bake-off measures peak memory through a re-exec launcher in test code.
  Amends behavior 1. Rejected: PeakRSS documented as at least the caller's RSS; a launcher in
  tool for every run.
- documents (slice 3 checkpoint): scanned page 5's truth adds the seal's "DEPARTMENT OF DEFENSE"
  and the handwritten "M.M.", leaving "UNITED STATES OF AMERICA" out; the other scanned truths
  stand. Source page 19 (faded SECRET//NOFORN markings at the foot, which herald found read
  inconsistently) joins the scanned set (architect).
- documents (slice 3 checkpoint): gs is the released renderer, by the agreed order: tied on
  accuracy, fastest (0.12 s/page), least memory (27 MiB), one tool; AGPL, run as a separate
  program. Rejected: vips (0.13 s/page, 90 MiB, GPL through pdfload's poppler).
- documents (slice 3 checkpoint): evidence 11's matrix also runs at 72 dpi, since every
  synthetic fault scores 0 at 150 dpi. Rejected: the 150-dpi matrix alone.
- documents: the bake-off runs once per renderer and set (15 runs, each capped at 10 minutes);
  one renderer's three sets overran 10 minutes in one run.
- documents (after slice 3): the scanned rerun found page 7's faded foot reads SECRET but never
  NOFORN, and 7 runs (8 by the committed JSON, which the findings cite) hit the 8192-token cap
  reasoning and came back empty. Transcription runs with
  reasoning off (Effort none; Gemma's template switch through Native if the router ignores it);
  error rate is scored over completed runs, and incomplete runs are a criterion of their own,
  ranked right after error rate; the whole bake-off reruns, and the winner is what it ranks
  first, with a stop for the architect only if it isn't gs. Rejected: a larger MaxOutputTokens.
- documents: reasoning goes off through the typed Effort none (reasoning_effort "none"), which
  llama.cpp b11669 honors for Gemma 4 (56 tokens and 4 s against 8192 and 152 s on scanned page 7);
  no Native template switch is needed. Every run completed; gs ranks first on error rate and time.
  Scanned page 7's faded NOFORN still never reads, and some seal text is still invented or
  repeated; that is a finding for the findings page.
- documents: whitespace-only output counts as incomplete.
- documents (slice 4 checkpoint): export_test.go keeps Ghostscript's render and renderArgs for
  the bake-off, a measurement probe of the released code path, with a comment saying why; slice 5
  removes only Spool. Rejected: the bake-off rebuilding gs's arguments itself.
- documents (architect, standing): any change to gb10-2's model configuration (the preset: context,
  slots, image tokens, sampling) is raised as an escalation, never made in BUILD. Neither a page
  request nor Document's final step should hit a context limit; one that does is escalated.
- documents: Pages and Document share one Request whose Final field Pages ignores. Rejected: a
  Document request embedding the page request.
- documents: a refused request is the stream's single error the first time Pages is ranged; both
  templates are validated as the requests they build, once, before any page runs.
- documents: a page whose response isn't completed fails Document as a failed page, its response
  kept; Pages leaves the status to the caller.
- documents: a final-step part is "Page N of M:", a newline, then the page's output text; a PDF
  with no pages is a Document error.
- documents (architect, after slice 5): Document digitizes, then digests. The page transcriptions
  are the digitized content, and Result gains the document's text, pages joined in order with
  their numbers; the final step derives a lightweight digest for search (summary, title, kind,
  dates, markings, keywords) as structured output under a JSON schema, and never replaces the
  transcriptions. Amends behavior 8. Rejected: a final step re-writing the whole document as one
  clean text (paraphrase and invention, output growing with the document).
- documents: Result.Text includes every page, a page without a response as its heading alone;
  the integration digest's schema stays test code, not released API; the final step's output
  cap is 4096 tokens (a request parameter).
- documents (architect, after J): Request gains PageRetries (zero: none): a page whose response is
  incomplete is re-sent up to that many times before it counts as failed, since scanned page 7
  loops on the seal's text to the 8192-token cap on some runs (1 of 4) with reasoning off.
  Integration sets 1 and caps a page at 2048 output tokens. Rejected: a repetition penalty
  (changes the greedy output every score rests on); leaving it to callers.
- documents: PageRetries re-sends a nil response too, builds a fresh request copy per attempt
  from the same PNG, and never retries Document's final request; Page.Attempts is 0 for a page
  that failed before sending. Three integration runs needed no retry; after M, page 7 stayed
  incomplete after its retry on 2 of 5 scanned runs (see the scanned-set record line).
- documents: evidence 11's matrix is its own task, `mise run bakeoff-corrections` (28 runs, each
  capped at 10 minutes, then the matrix); it puts TranscribeInstructions in the request's
  Instructions, as Vision does, so its none cell compares only with its own cells, not with the
  renderer bake-off's numbers.
- documents (architect, slice 6 checkpoint): processing's libraries provide pre-inference
  preparation; a layer above defines and orchestrates inference and embeddings, so inference
  accuracy and structure in this spike only prove the preparation usable. Confidence scoring, as
  herald had, is that layer's concern.
- documents (slice 6 checkpoint, M): processing releases a prepared-page stream, Images(ctx, pdf,
  req) iter.Seq2[PageImage, error] (spool, count, render, correct, page order, bounded
  concurrency, cancellation); Vision, Pages, Document, Generator, and TranscribeInstructions move
  to test code in the integration and bake-off tiers, which still prove evidence 4, and the client
  becomes a test-only requirement. Amends behaviors 2, 6-8, and 13-15. Rejected: releasing Vision
  beside the stream; keeping the release as it was. Refined by the (M) line below: the harness is
  untagged test code.
- documents (slice 6 checkpoint, L): instruction placement (in the image's message or in the
  request's Instructions) is measured through Vision on the clean, degraded, and scanned sets, 3
  runs each, without correction, and reported as a finding about the inference layer.
- documents (slice 6 checkpoint, N): processing gains the correction catalog (each Correction's
  name and a one-line description for prompts) and ParseCorrection; text layers per page (whether
  one exists, and its text, from gs txtwrite); Assess(ctx, png) (skew angle, contrast, noise,
  sharpness, grayscale, from magick); and regions (Render takes an optional crop, a band of the
  page in page fractions, as one gs run). The bake-off stages the flow the inference layer
  follows: text layer first, vision only for pages without one, a first pass returning
  transcription, judgement, and corrections with Assess's figures in the prompt, a second pass
  only for flagged pages, and a header/footer crop pass on the scanned set, compared with none
  and all-five on calls, time, and error rate. Amends herald's flow, which ran a full first pass
  on every page. Rejected: without regions; the catalog and combined pass alone.
- documents (M): the released API is Preparer (NewPreparer(Renderer, *Corrector)) and
  Images(ctx, pdf, ImagesRequest) iter.Seq2[PageImage, error]; the vision harness (vision,
  pages, document, generator, transcribeInstructions) lives unexported in untagged
  document/vision_test.go so its unit tests stay in the check; the client is test-only.
- documents: the scanned-set Document integration test records a page still incomplete after its
  retry (page 7's seal loop, 2 of 5 runs after M) instead of failing, since inference outcomes
  only prove the preparation usable; the clean set still asserts completion.
- documents (N): text and regions reach Images through optional interfaces (TextExtractor,
  RegionRenderer) that Preparer type-asserts, so Renderer isn't widened; SkipText implies Text.
- documents (N): Ghostscript.Text uses txtwrite with -sstdout=%stderr.
- documents (N): RenderRegion is one gs run of PostScript over pdf_main.ps's
  pdfshowpage_setpage/finish, pixel-identical to Render for the whole page.
- documents (N): Assess is one magick run on Corrector; grayscale is the fraction of pixels with
  HCL chroma over 0.2 below 0.001.
- documents (N, finding): txtwrite keeps the page's layout, so a two-column page's text layer reads
  across the columns (CER 0.67, every word present): a finding for the inference layer.
- documents (L, N runs): every staged run puts the instructions beside the image, which L
  measured better (scanned CER 0.086 against 0.117, deterministic).
- documents (N runs): the first-pass schema puts the transcription first and the corrections as an
  enum of the catalog's names; a first pass cut off or unparsable leaves the page incomplete, with
  no second pass.
- documents (N, results): text layers made no model calls on the clean set but read two-column
  and form pages out of order; the model missed moderate skew, fade, and blur and judged every
  scanned page poor for its redactions; second passes rarely helped (scanned: 1 helped, 2 hurt, 4
  same); all-five beat none on the scanned set (0.068 against 0.086); a 300-dpi full-width band
  costs as many image tokens as a page and still didn't read page 7's NOFORN ("SPERET NORDEN").
- documents (O): image tokens follow a region's area, not its shape (about 2,100 to 2,400
  pixels a token, floor 82); a small tile on a known spot reads what the page and its band miss
  (page 5's seal on all six reads, at about a fifth of a page's tokens), but page 7's faded NOFORN
  reads in no tile at 200 or 300 dpi ("NOORDEN"), so vision can't be relied on for it. A tile
  holding a redaction drew a refusal. The hypothesis that wide bands are squashed doesn't hold.
- documents: the bake-off ranks by pairwise wins, each pair decided by the first criterion
  outside the run-to-run spread, since a tie inside the spread isn't transitive; pdftocairo
  (mean CER 0.0364) ranks below vips (0.0414) by time. gs wins under a plain ordering too.
- documents (spec review): the 413 and the final-step context limit belong to the test harness
  after M, so the v0.2.0 CHANGELOG names neither; the findings page covers both.
- documents (spec review): the bake-off result JSON the findings cite is committed beside the page,
  in processing's docs/findings/documents-results/, one directory per run, so its numbers trace to
  files.
- documents (editor): documents-results has one directory per mise task's run: renderers
  (reasoning off), renderers-reasoning-on, corrections, placement, flow, and tiles. The earlier
  six-page scanned run isn't committed, since the findings don't cite it.

- vision-qat (architect, after documents; outside the roadmap's tasks): set A on gb10-2 is held to
  ai-hosting.md's quantization floor. Gemma 4 26B-A4B moves from ggml-org's post-training Q4_0
  (with a Q8_0 projector) to Google's QAT Q4_0 (google/gemma-4-26B-A4B-it-qat-q4_0-gguf, with its
  own projector), as ai-hosting.md's set A names it: the floor accepts a QAT Q4_0, and the memory
  is the same. Supersedes the host decision that deferred the swap to spike-model-benchmarks.
  gpt-oss-120b MXFP4 (native), gemma-4-E4B Q8_0 (dense under 8B, ASR), and granite BF16
  (embedding) already meet the floor. The bake-offs rerun on the QAT build.
- vision-qat: the preset serves google/gemma-4-26B-A4B-it-qat-q4_0-gguf:Q4_0 (its one model file and
  its projector, picked by the Q4_0 tag); image tokens per page are unchanged (223/416/477/477).
  gb10-2's model store keeps set A alone: ggml-org/embeddinggemma-2-GGUF,
  nvidia/Gemma-4-26B-A4B-NVFP4, and ggml-org/gemma-4-26B-A4B-it-GGUF are removed with their blobs
  (architect). llama.cpp caps the vision model's image tokens at 512 by n_ubatch 512 where the
  model allows 1120: a preset setting, raised only by escalation.
- vision-qat (architect): raise the vision model's image-token budget to the model's own cap
  (1120, read from the model) if the memory pool allows: llama.cpp caps it at n_ubatch, 512 by
  default, so the vision section takes ub = 1120 after the QAT run at 512 finishes, measured
  (memory before and after, set A fully loaded) and reverted if it doesn't fit; the bake-offs
  then rerun, giving post-training at 512, QAT at 512, and QAT at 1120 side by side.
- vision-qat (Q): gs stays the released renderer. The renderer ranking scores accuracy on the
  pages with a text or vector layer (clean, degraded) and reports scanned pages apart, since every
  renderer decodes the same embedded scan; on QAT at 512 image tokens the full ranking put vips
  first and gs fourth on scanned page 7 alone (its seal block repeated three times with gs, once
  with vips). Rejected: the ranking unchanged; re-deciding after the 1,120 run.
- vision-qat (R): ub rises to 1,120 now; corrections, placement, flow, and tiles run at 1,120
  only; the renderers alone carry post-training 512, QAT 512, and QAT 1,120. Rejected: all five
  at 512 first.
- vision-qat: the vision section sets ub = 1120 and leaves b at 2048 and mtmd-batch-max-tokens at
  512 (a first image always encodes alone in its batch); the cap warning is gone, and the vision
  instance grew about 800 MiB (17,346 to 18,143 MiB) with set A loaded and about 16 GiB available.
  Image tokens per letter page: 223 (72 dpi), 416 (100), 920 (150), 1104 from 200 dpi up.
- vision-qat (Q): ranked on clean and degraded, gs is first on both builds, time deciding every
  pair (accuracy ties); scanned results are logged as not ranked (processing 302c3c5).
- vision-qat (S): the default DPI stays 150, per request. Its first reason (150 and 200 dpi cost
  the same 477 tokens) no longer holds at the full budget (920 against 1,104); the new reason:
  150 dpi reads clean and degraded print perfectly, more tokens per whole page didn't help scans
  (scanned CER rose from 0.119 to 0.133 for gs going from 477 to 920 tokens, all of it page 7) and
  slowed requests (3.0 to 5.1 s), and 200 dpi is untested on whole pages. Tiles are the lever for
  small marks. Rejected: a 200-dpi default.
- vision-qat (T): ub = 1120 stays on gb10-2: no run looped at the full budget (QAT at 512 looped
  twice), it is the model's own budget, and it fits. Rejected: returning to 512.
- vision-qat: on the QAT build at 1,120 image tokens, gs ranks first in every column (time
  deciding every pair); page 7's NOFORN is never read, page 7's seal draws more invention, page
  5's seal reads only from a tile; corrections at 150 dpi now help (Deskew, Unsharp) where they
  hurt; second passes changed nothing.
- office-cli: one task, with the markitdown and xberg baseline beside the v0.1.0 release; slices
  go one format at a time. Rejected: the baseline as its own task.
- office-cli: the JSON model's schema is the spike's own, versioned with office's tags; processing
  has no model to match. Rejected: Docling's DoclingDocument (page provenance, layout boxes).
- office-cli: the chosen rendering (--format json|markdown) goes to stdout, media to --media's
  directory, named relative to it. Rejected: always writing everything into an output directory.
- office-cli: the binary is spike-ai-pipeline-office. Rejected: office (too generic);
  SpikeAiPipeline.Office.
- office-cli: System.CommandLine 2.0.14 parses arguments (MIT, no AOT warnings, +1.75 MB).
  Rejected: a hand-written parser.
- office-cli: fidelity is scored on each tool's Markdown, parsed by Markdig (test-only), against
  element-level ground truth; office's JSON is checked exactly in its own tests. Rejected: scoring
  native outputs.
- office-cli: linux-arm64 is built natively on ubuntu-24.04-arm. Rejected: cross-compiling on x64.
- office-cli: release assets are a tar.gz per runtime identifier (binary and LICENSE) plus
  SHA256SUMS. Rejected: bare binaries.
- office-cli: office's merge waits on the pull request's checks. Rejected: merging without
  waiting, as client and processing still do.
- office-cli: docx tracked changes are read accepted. Rejected: as stored with revisions marked.
- office-cli: docx headers and footers are in v0.1.0, per section with default, first, and even
  kinds. Rejected: out as a known limit.
- office-cli: docx footnotes and endnotes are in v0.1.0, separator notes skipped. Rejected: out.
- office-cli (architect): docx comments are in v0.1.0 with text, author, anchored range, replies,
  and resolved state, because comments can hold critical context. Rejected: out as a known limit.
- office-cli: xlsx cells carry cached value plus formula text, never evaluated; null when no cached
  value; shared-formula cells carry the anchor's reference. Rejected: value only; evaluating.
- office-cli: xlsx cells hold a typed value with dates as ISO 8601, no other number format.
  Rejected: displayed text through number formats.
- office-cli: one model tree for all three formats; sheets and slides are blocks. Rejected: a
  top-level type per format.
- office-cli (architect): the corpus is only the committed pool, the generated fixtures re-saved by
  LibreOffice plus about a dozen hand-scored real-world files from permissive corpora (30 by the
  lines below), under a sources manifest. Rejected: a larger must-not-crash set; real only;
  generated only.
- office-cli (architect): no robustness sweep; the baseline runs on the pool alone. Rejected: a
  sweep over POI's and LibreOffice's ~2,400 files.
- office-cli: CI's check runs on ubuntu-24.04 and ubuntu-24.04-arm. Rejected: x64 only.
- office-cli: comments render in Markdown as footnote-style [^cN] references, defined after the
  notes with author, resolved state, and replies. Rejected: comments in the JSON alone.
- office-cli (architect): office runs no inference, not even in tests; its Markdown is the
  document's text and media blocks are the only content for vision, following processing's
  text-first flow. The gb10-2 vision integration over office's media belongs to processing's
  `office` task (pending edit to the roadmap's tasks.office). Rejected: a vision tier in office.
- office-cli (architect): Outlook .pst joins the spike as its own task, `pst`, after `office` and
  before `audio`, through a coordinator pull request at this session's end; it is a container
  format, not OOXML, needing its own adapter. Rejected: leaving mail stores out of the spike.
- office-cli: media files are content-addressed (SHA-256 plus stored extension), written once
  per distinct picture so the higher layer runs vision once per picture and can cache by hash;
  each media block keeps its part path, content type, alt text, and location; linked pictures
  get a block with their target and no file. Rejected: ordinal names; the package's part paths.
- office-cli: the release job runs each architecture's binary's --version before attaching it;
  slice 5's checkpoint has the architect confirm the real-world manifest; the session applies
  main's ruleset through gh api; macro-enabled and template variants are out of scope
  (superseded by the slice 5 escalation: read as their base format).
- office-cli (architect, slice 1 checkpoint): a deleted paragraph mark right before a table keeps
  its carried text in place as its own paragraph before the table. Rejected: joining it to the
  paragraph after the table, out of document order.
- office-cli (architect, slice 1 checkpoint): the Markdown renderer gets independent tests over
  hand-built models with hand-written expectations, beside the snapshot goldens.
- office-cli (slice 1): refusals exit 3; format sniffed from content; lists grouped by abstract
  numbering definition (LibreOffice re-saves split one list across instances); empty paragraphs
  and headings dropped, empty list items kept; merged cells as colSpan/rowSpan on the anchor.
- office-cli (architect, slice 3 checkpoint): index xlsx merges by cell in slice 4, since the
  baseline times each file. Rejected: scanning every merge per cell.
- office-cli (slice 3): xlsx tables are sparse (rows carry `row`, cells their `ref`); hidden and very
  hidden sheets are both `hidden: true`; the 1900 leap day, negative serials, and years past 9999
  stay numbers.
- office-cli (architect, slice 4 checkpoint): add an xlsx merge case that exercises the merge
  index's expiry in slice 5.
- office-cli (slice 4): each slide renders as `# Slide N` with its own headings one level down and
  notes under `**Notes**`; shapes read in shape-tree order; media `slide` is the 0-based index.
- office-cli (slice 5): the implementer ran the binary over all 649 downloaded corpus files while
  selecting, beyond the architect's pool rule; nothing from that run is committed. It found three
  bugs, fixed in ef4d7fa (invalid attribute values and missing parts refused as malformed, not
  crashing; Excel Online's spaced shared-string index read).
- office-cli (architect, slice 5 escalation): documents nested past 8 tables are refused (exit 3)
  with the depth and location named. Rejected: id-referenced side tables (additive later).
- office-cli (architect, slice 5 escalation): fix the text-losing docx bugs (row/cell-level
  content controls, smart tags, ruby, a comment range ending between paragraphs), each with a
  real regression file. Rejected: known limits.
- office-cli (architect, slice 5 escalation): lists carry their start value and continuation
  (`start`), and cells their column offset from gridBefore (`skipBefore`), additive to schema
  "1". Rejected: known limits.
- office-cli (architect, slice 5 escalation): chartEx charts read their cached data into the chart
  block, falling back to the mc:Fallback picture as media when none is cached; Strict charts are
  read through a namespace map in ChartReader. Rejected: picture-only; known limits.
- office-cli (architect, slice 5 escalation): the real-world pool grows to 27 files: ComplexNumberedLists,
  tika-3816, WordWithAttachments, XSSFSheet.copyRows, SimpleStrict, WithDrawing, 56812, endnotes,
  form_footnotes, 61470, chartex, word_tables (re-zipped), and one O14ISOStrict chart file (SDK
  re-save), with the manifest noting each exception.
- office-cli (architect, slice 5 escalation): macro-enabled files and templates are read as their
  base format, macros ignored and never run, documented in README and CHANGELOG. Rejected: refusing.
- office-cli (architect, slice 5 escalation): a single invalid reference is skipped with a warning
  on stderr and the rest converts (exit 0); packages broken at ZIP or part level stay refused.
- office-cli: the escalation's work runs as slices 5b (docx fixes, nesting refusal, lenient
  references, macro docs, their files) and 5c (Strict and chartEx charts, the remaining files)
  before CI.
- office-cli (slice 5b): exit 4 "failed" for office's own unexpected errors; OLE objects'
  preview pictures come out as media; placeholder text of content controls and ruby readings are
  dropped; list levels clamp to Word's nine.
- office-cli (architect, slice 5b checkpoint): the truth format compares a table cell's full blocks
  (lists, headings), extended in slice 5c.
- office-cli (architect, slice 5c checkpoint): Of16-02.pptx stays in the pool (28 real files),
  though the session added it to 5c's brief beyond the 27 approved; a search for an Office-native
  chartEx file under a permissive license follows. (The checkpoint took both chartEx regression
  files as tool-written; the next line corrects it.)
- office-cli: correction to the 5c checkpoint: Of16-02.pptx is Office-native chartEx (typed
  chartEx, cx:chart under Requires="cx1", style/colors parts, embedded workbook, 1980 ZIP dates,
  no BOM); only POI's chartex.docx is tool-written. Office writes chartEx parts with no CRLF
  after the declaration and ` />`, and style/colors parts with no declaration; the tool
  fingerprint is a UTF-8 BOM with lowercase `utf-8`, a part typed as an ordinary chart, or
  c:chart in the drawing.
- office-cli (architect): add LilLeapo/rsWordParser's chartex-waterfall-2.docx and
  chartex-funnel-2.docx (MIT/Apache-2.0, Word-saved, Requires="cx1"/"cx2") as Office-native docx
  chartEx; the manifest records their self-reported provenance and core.xml's personal name. The
  pool is 30 real files. xlsx chartEx stays with the test-built package. Rejected: an
  architect-saved xlsx.
- office-cli (architect, slice 7 checkpoint): the findings page adds a real-world-only fidelity
  table beside the pooled one, since the generated set and office's Markdown were built around
  the same truth. Chart values count wherever they appear between the chart's title and the next
  block, not only in a table. markitdown runs as shipped, magika (a local ONNX classifier inside
  its own process) noted; office never loads either baseline tool.
- office-cli (slice 7): baseline tool pins live in mise.baseline.toml (MISE_ENV=baseline), never in
  CI or the binary; peak memory via a C fork/wait4 wrapper; office matched 1,768/1,768 elements
  over the pool and 1,144/1,144 over the 30 real-world files.
- office-cli (slice 8, spec review): schema "1" leaves out an optional property with no value,
  never writing null, so absent means null or none; only an array keeps a null in place (a chart
  point the cache lacks). The serializer already did this; README and CHANGELOG now say so.
- office-cli (spec review): the release is created as a draft, its assets uploaded, then
  published, so a failed upload leaves a draft. Rejected: publishing before the upload.
- office-cli (spec review): the linux-arm64 binary's size comes from the pull request's CI run
  (the check job's summary), since a NativeAOT publish for arm64 can't link on the x64 laptop.
- office-cli (slice 8): office's README no longer says processing runs office or LibreOffice
  converts legacy files before it; office refuses them, and conversion is the caller's (the
  `office` task's).

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
  against the router's `/v1/responses`; tags client v0.4.0; evidence 2). In the spike's summary,
  the client's wire formats read "(Chat Completions, Anthropic Messages, and Open Responses)"
  where they read "(Chat Completions, then Anthropic Messages)". The root's evidence 2 already
  names all three (`context/README.md`).
- coordinator · roadmap, the `client-core` task: its summary says the stable model is Open
  Responses' item model, stateless subset, and its repos gain `spike-ai-pipeline`, where set A's
  embedding model was swapped to granite-embedding-english-r2 in the preset and probe.
- coordinator · roadmap, the spike's summary: the sentence saying set A's embedding model "is to
  change" to granite-embedding-english-r2 says it changed.
- coordinator · context/ai-strategy.md, the `spike-ai-pipeline-client` bullet, rewritten to
  v0.4.0:

  > - **`spike-ai-pipeline-client`** gives Go primitives over the engines' native APIs, built as
  >   owned, optimized infrastructure from the best of spike-harness-driver's `model` and tau's
  >   libraries, on the standard library alone. Its protocol is Open Responses (spec 2026-04-24)
  >   in its stateless subset; client extensions, audio input first, fill the spec's gaps and are
  >   removed when it adopts them. Four layers import one way: the `client` transport, with
  >   Generate, GenerateStream, Embed, and Transcribe; `protocol`; the `format` contract; and a
  >   `format/<api>` adapter per wire format, named by endpoint and chosen by typed constructor:
  >   `completions` (Chat Completions), `messages` (Anthropic Messages), and `responses` (Open
  >   Responses). GenerateStream yields Open Responses' streaming events, which a
  >   `protocol.Accumulator` folds into the Response Generate returns. Retry is opt-in.
- coordinator · roadmap, the `host-scripts` task: its summary and proof name d285b06
  (llama-gb10-native's merge) where they name host-rename's merge commit, and its summary
  says the container scripts plus llama-release.sh are copied.
- architecture · a retro candidate: "follow the spec wherever possible", the architect's standing
  rule for the client (client-streaming), may belong in STANDARDS.md or the architecture layer as
  a standard.
- coordinator · context/ai-hosting.md, "Set A's embedding model": gb10-2 serves
  granite-embedding-english-r2 BF16 from llama.cpp (spike-ai-pipeline, client-core).
- coordinator · context/ai-hosting.md, "Hosts and access" (optional): llama.cpp b11669's
  `/v1/messages` rebuilds a request as Chat Completions. It drops output_config, a tool's strict,
  and the none tool choice. It maps named and allowed_tools choices to required, yet gpt-oss-120b
  made no call under either and repeated its answer to the 1024-token bound. Its stream error
  frame carries no `"type":"error"`. Its `/v1/responses` also rebuilds a request as Chat
  Completions. It drops text, reasoning's summary, store, include, and an image's detail, and
  treats named and allowed_tools choices as auto. It refuses previous_response_id, a content part
  other than text, image, or file, and a reasoning item without content. Every reply reports
  completed, even at the token bound. It streams reasoning under OpenAI's
  `response.reasoning_text` names, with no sequence numbers or indexes, and its stream error
  frame is `{"error":{...}}` with no type.
- coordinator · roadmap, the `tools` task: add `proof = "One tool, ffprobe, runs through both
  runners, exec on the host and container in a digest-pinned image, under the same tool contract
  with equal results, and processing v0.1.0 is released."`
- coordinator · context/ai-strategy.md, the `spike-ai-pipeline-processing` bullet, rewritten to
  v0.2.0 (this replaces the bullet's v0.1.0 draft, which was never applied):

  > - **`spike-ai-pipeline-processing`** prepares messages, documents, and streams for native
  >   model capabilities before inference; a layer above defines and orchestrates the inference
  >   and embeddings. It runs mature system tools through adapters it owns (`system-tools.md`),
  >   with herald's `internal/format` as its prior art. Its `tool` package, on the standard library
  >   alone, is the contract: a Tool bound to one program, found and version-checked at
  >   construction, runs an invocation under the caller's context through the exec runner
  >   (binaries, and scripts through their interpreter) or the container runner (the docker CLI, an
  >   image pinned by digest, each run locked down), and reports the run's CPU time. Its `document`
  >   package prepares a PDF's pages: Ghostscript, which won the renderer bake-off against
  >   pdftoppm, pdftocairo, vips, and ImageMagick over gs, renders pages and regions as PNG and
  >   reads text layers; ImageMagick corrects page images, lists its corrections as a catalog for a
  >   prompt, and assesses a page for the defects they fix; and a page stream yields the prepared
  >   pages in order. ffmpeg and ffprobe take audio and video, live sources included.
- coordinator · roadmap, the `documents` task: name "PDF pages prepared for vision" where it reads
  "PDF pages through vision", and its summary says what the task delivered:

  ```
  Pre-inference preparation of a PDF's pages: the renderer bake-off (pdftoppm, pdftocairo, gs,
  vips, and magick over gs) behind the page-renderer interface, with gs released; ImageMagick
  corrections in a fixed order, their catalog for a prompt, and a page assessment; text layers,
  regions, and an ordered, concurrent page stream; and tool's CPU usage report. Inference is test
  code: a vision harness carries the pages through vision as a whole blob and as a page stream,
  and the bake-off tier measures the corrections, instruction placement, a staged two-pass flow,
  and tiles. Tags processing v0.2.0. Evidence 4, 11.
  ```
- coordinator · context/system-tools.md, "First case" (optional): add after its first sentence
  "spike-ai-pipeline-processing's `tool` package is the contract: a binary, a script through its
  interpreter, and a digest-pinned image through the docker CLI each run as one Tool, which its
  constructor finds and version-checks (evidence 14, processing v0.1.0)."
- coordinator · context/ai-hosting.md, "Hosts and access": the llama container publishes 8080 on
  gb10-2's Tailscale address, which races Tailscale at boot: docker's start fails ("cannot assign
  requested address"), leaves the container on no network, and a later `docker start` runs it with
  no published port. gb10-2 now sets `net.ipv4.ip_nonlocal_bind = 1`
  (/etc/sysctl.d/90-nonlocal-bind.conf, the architect, 2026-10-30), so the bind succeeds before
  Tailscale is up. A container already stranded is fixed by `docker network connect bridge llama`.
  The same note goes to spike-model-engines, whose copied scripts publish the port this way.
- roadmap note · spike-ai-pipeline root: host-update could detect a running llama container with no
  published port (or no network) and print the fix, instead of waiting out its 120 s health check.
- coordinator · roadmap, spike-model-benchmarks (architect, documents): measure vision models on
  spike-ai-pipeline-processing's committed scanned fixtures (document/testdata/scanned.pdf, ground
  truth beside it): a larger image-token budget than gemma-4-26B-A4B's default, a dense vision
  model, and a lighter quantization than Q4_0. Gemma 4 26B-A4B Q4_0 never read scanned page 7's
  faded NOFORN, even on a 122-token 300-dpi tile, which points to the model, not resolution
  (docs/findings/documents.md).
- coordinator · roadmap, spike-model-layout (architect, documents): gemma-4-26B-A4B's image-token
  budget on gb10-2 (the model's default --image-max-tokens, ~477 tokens a page; image tokens follow
  an image's area at about 2,100 to 2,400 pixels a token) is an input to sizing set A's vision
  slot; spike-ai-pipeline-processing's docs/findings/documents.md has the measurements. gb10-2
  serves ggml-org's Q4_0 weights with mmproj-gemma-4-26B-A4B-it-Q8_0, not Google's QAT Q4_0.
- coordinator · context/ai-hosting.md, set A on gb10-2: it serves Google's QAT Q4_0 of Gemma 4
  26B-A4B (google/gemma-4-26B-A4B-it-qat-q4_0-gguf, its own projector) since 2026-10-30, as set A
  names it, in place of ggml-org's post-training Q4_0, which fails the floor; the vision section
  sets ub = 1120, since llama.cpp caps a non-causal image's tokens at n_ubatch (default 512)
  where the model allows 1120 (+800 MiB). The Framework's set A likely has the same 512 cap.
  gb10-2's store holds set A alone (embeddinggemma-2 and an NVFP4 Gemma were removed).
- coordinator · roadmap, the `office` task's summary, rewritten to name office's released contract
  and the vision integration tier (office-cli, architect: office itself runs no inference):

  ```
  The adapter over the Open XML CLI at office v0.1.0, by its release assets (a tar.gz per
  linux-x64 and linux-arm64, and SHA256SUMS) and its CLI contract (README.md): the JSON model,
  schema "1", on stdout; embedded media through --media, named by SHA-256; exit codes 0
  converted, 1 media not written, 2 usage, 3 refused, 4 failed; warnings on stderr. Legacy .doc,
  .xls, and .ppt converted to OOXML by LibreOffice headless on the container runner, since office
  refuses them. An integration tier sends office's extracted media through vision on gb10-2 with
  processing's test-only vision harness. Tags processing v0.3.0. Evidence 12.
  ```
- coordinator · roadmap, the `office-cli` task (optional): add `proof = "The check passes on
  ubuntu-24.04 and ubuntu-24.04-arm, office v0.1.0 is released with both archives and SHA256SUMS,
  and docs/findings/office-cli.md records the baseline against markitdown and xberg."`
- coordinator · context/ai-strategy.md, the `spike-ai-pipeline-office` bullet (optional), after
  its first sentence: "Its JSON model is a versioned output contract (schema `1` at v0.1.0). Its
  Markdown is the document's text, complete without inference, and its media blocks are the only
  content a higher layer may send to vision; it runs no inference itself."
- coordinator · roadmap (architect-authorized for office-cli's session end, not sync: a
  `plan-experiment.ai.spike-ai-pipeline` pull request after office-cli ships): insert task `pst`
  after `office` and before `audio`, then add it to Tasks as 11 and renumber audio and streams
  to 12 and 13; audio's summary moves its tag to processing v0.5.0 and streams' to v0.6.0. Entry
  (`[goals.experiment.ai.spike-ai-pipeline.tasks.pst]`):
  name "Outlook mail stores", repos ["spike-ai-pipeline-processing"], summary: Outlook .pst (and
  .ost) stores, proprietary binary, not OOXML ([MS-PST]), read through a mature system tool on the
  container runner (libpff or readpst, chosen in its plan) into folders, messages (headers, body,
  threads), and attachments; each attachment routed back through the document stages. Tags
  processing v0.4.0. Evidence: a new item for mail stores, written when `pst` is planned.
