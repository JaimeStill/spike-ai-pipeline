# goal · experiment.ai.spike-ai-pipeline

- **State:** building
- **Task:** onenote
- **Branch:** onenote

## Tasks

1. [x] host
2. [x] client-core
3. [x] host-scripts
4. [x] client-streaming
5. [x] client-messages
6. [x] client-responses
7. [x] tools
8. [x] documents
9. [x] office-cli
10. [x] office
11. [x] mail-cli
12. [x] email
13. [x] mail-mime
14. [x] email-mime
15. [x] onenote-cli
16. [ ] onenote
17. [ ] gb10-2-align
18. [ ] audio
19. [ ] streams
20. [ ] formats

## Task brief · experiment.ai.spike-ai-pipeline · onenote

```
Problem       Processing reads PDFs, office documents and mail, but not
              OneNote, and its format packages have grown into one
              another: signature checks live in three packages, office
              owns picture conversion email's callers need, document and
              office each declare magick, office and email each copy the
              CLI contract. Every route is left for the caller to run, so
              an attachment holding an attachment is recursion every
              caller rebuilds, with no inspection, no dedupe across
              inputs, and a media directory that holds a whole store's
              attachments until the read ends. An enterprise share drive
              (large .pst stores, repeated content) needs all of that
              without losing data. mail can't resume a store.

              This task gives mail a record cursor (v0.3.0, released as
              a checkpoint first). In processing it cleans up the layers
              against those findings: `route` (identification and
              routing for every format in the spike's inventory),
              `picture` (conversion to PNG, SVG included, with the
              corrections and assessment), an internal CLI contract, and
              a written adapter convention. Then it adds an inspection
              step, the `onenote` adapter over the CLI at v0.1.0, and a
              dispatcher that inspects and prepares each part once per
              content across inputs, runs and workers, recurses with no
              default caps, holds media only until each part is
              consumed, reads zip, and resumes. It never runs inference.
              A finding; processing v0.6.0. The root's evidence 17 links
              the finding.

              Currency: processing and mail current; the root's llama.cpp
              b12110 held (email-mime). No upgrade slice.

Behaviors
  mail (spike-ai-pipeline-mail)
  1.  `--after <id>` resumes a .pst after a message node id and an
      mbox after a 1-based ordinal. Folder lines are always written.
      A .msg or .eml refuses it with exit 2; an id not in the store is
      refused with exit 3. A resumed read equals the tail of a whole
      read, byte for byte; a .pst skips without opening messages.
  2.  README, CHANGELOG v0.3.0, the mail-cli finding's resume timing on
      the scale store, and the v0.3.0 release.

  Processing: lower layers
  3.  `route` (standard library only) identifies content by signature:
      PDF; PNG, JPEG, GIF, BMP; TIFF, EMF, WMF; SVG; HEIC, WebP, AVIF;
      ZIP and OLE2 by extension; ODF; RTF; OneStore and MSCF; WAV,
      FLAC, MP3, Ogg, M4A; MP4, MKV, WebM, AVI; gzip, zstd, 7z;
      executables (PE, ELF, Mach-O, shebang); text by extension or BOM;
      the .eml rule kept. File(path, name, mimeType) returns a route
      and reason: document, office, vision, vision-converted, nested,
      onenote, archive, audio, video, text, unrouted. email's v0.5.0
      routes and reasons are unchanged for every fixture.
  4.  `picture` converts TIFF, EMF, WMF and SVG to PNG named by the
      original's SHA-256 (SVG at 150 dpi flattened on white; each
      picture in its own directory, so external references don't
      resolve; no lossy re-encode), and holds the corrections, their
      catalog and Assess, and the magick and LibreOffice specs. office
      and document call it; their outputs are byte-identical to v0.5.0
      on every fixture. ~/docs' script-bearing.svg converts with no
      script run.
  5.  office, email and onenote share one internal CLI contract (exit
      codes 1 to 4 as shared sentinels, ExitError, the stderr warning
      reader, limitedBuffer, spool); errors.Is matches a refusal from
      any of them through any package's ErrRefused.
  6.  The README records the adapter convention (Spec and Minimum,
      Config, a constructor refusing a missing or old Tool, Request
      with a media directory, typed exit errors, warnings, SHA-256
      media, iter.Seq2 streaming, a route per part) and each
      package's deviation. email.Route, office.ConvertPicture and its
      request, document.Corrector and kin, and both MagickSpecs move
      without aliases, each listed in the CHANGELOG.

  Processing: inspection
  7.  Every part is inspected before its stage, on the standard
      library, each finding a flag with its check, class and evidence.
      Active content, flagged and still processed: a signature that
      disagrees with the extension; a polyglot; executable content or
      an executable or double extension; OOXML vbaProject.bin, ActiveX
      parts, oleObject embeddings and external relationships; PDF
      /JavaScript, /JS, /OpenAction, /AA, /Launch, /EmbeddedFile,
      /RichMedia, /XFA, /Encrypt (hex-escaped forms included); SVG
      <script>, on* attributes, external hrefs, foreignObject.
      Unsafe to read, quarantined (no stage runs, content kept, the
      part yielded "quarantined" with its evidence): a zip entry name
      with "..", an absolute path or a drive letter; a symlink entry;
      a zip entry past both 1,000:1 and 1 GiB expanded. A caller
      policy overrides any check.
  8.  ~/docs' hazard inputs (javascript.pdf, script-bearing.svg, the
      macro files, the archive set, malformed PDFs) each yield their
      expected flags; the clean fixtures yield none.

  Processing: onenote
  9.  onenote runs the CLI, pinned from its v0.1.0 release through
      mise, on the exec runner; refuses a missing Tool or one older
      than 0.1.0, and a bad Request with ErrRequest before anything
      runs. Records yields Notebook, SectionGroup, Section, Page,
      Skipped and Unknown values while the CLI runs, one line at a
      time. A schema other than "1" is ErrSchema; exits 1 to 4 are an
      ExitError; warnings come back without their prefix.
  10. All of schema "1"'s content is typed (containers, items, blocks,
      paragraphs, runs incl. math, tables, images, embedded files, ink,
      recognised words). A same-run round trip of every fixture equals
      the CLI's stdout. A page's markdown is the CLI's, links untouched.
  11. A page's Media iterates every image, embedded file and ink record
      wherever it sits (container, outline block, table cell,
      paragraph), with its content path when a media directory was
      given, routed by route.File; ink vision-converted; recordings
      audio.
  12. onenote v0.1.0's public fixtures and expectations are copied into
      processing under a SHA-256 manifest, with a fixtures:onenote
      refresh from the tag; the check verifies them offline. ~/docs
      gains a onenote area (the public sets and the three large
      inputs) in MANIFEST.json with source, pin, licence and SHA-256.

  Processing: dispatcher
  13. Dispatch takes an input, the caller's stages and an index, hashes
      the input before any stage runs, and yields events depth-first:
      each Part (path, name, size, SHA-256, route and reason, flags,
      parent, depth, where found) with its outcome (prepared, no
      processor yet, seen, cycle, quarantined, encrypted, failed, and
      limited only under a cap the caller set) and its stage's
      outputs. It recurses into email's attachments and embedded
      messages, onenote's media, office's media and zip entries, and
      never calls a model.
  14. Stages are an exported interface; adapters for document, office,
      picture, email, onenote and zip ship in the dispatcher package;
      a caller may register its own.
  15. No count, byte or depth cap applies by default; a caller may set
      any. A part whose SHA-256 equals an ancestor's is "cycle".
  16. The index is an exported, content-addressed interface: outputs
      keyed by SHA-256 plus stage identity (name, tool version,
      configuration hash), flags by SHA-256 alone; look up, record,
      claim and wait. Content the index holds is "seen", carrying the
      recorded outputs and flags by the caller's keys, and is neither
      inspected nor prepared again. A claimed content in flight makes
      a second dispatch wait for its outputs. Shipped: an in-memory
      index, and a file-backed one on the standard library (one record
      per key in a caller-named directory, atomic rename, exclusive
      claim files reclaimed after a stale timeout).
  17. Each part's content file is removed once the caller's loop moves
      past its event unless the caller kept it; disk stays near one
      record's media in flight, mail held back through its pipe.
  18. Zip through archive/zip one entry at a time; entries written by
      SHA-256, names never used as paths; encrypted entries
      "encrypted"; Unicode names kept. ~/docs' zip-in-zip,
      unicode-filenames, password-protected and high-ratio (at a test
      threshold) inputs read as expected.
  19. A pool dispatches an iterator of inputs over N workers sharing
      one index, merging events with each input's identity.
  20. email's Request gains After, passed to mail; the Threader saves
      and restores its keys. A resumed read's threads equal a whole
      read's.

  Proof and release
  21. The integration tier runs every fixture through Dispatch against
      the real stages (office, document, email, picture then vision on
      gb10-2, zip), each route checked against hand route truth.
  22. The finding records, from recorded runs: routes against hand
      truth; inspection flags over ~/docs' hazards; vision's reading of
      handwriting_recognition.one's ink against a hand transcription
      beside OneNote's stored recognition, 3 greedy runs; the onenote
      adapter's time and peak memory beside the CLI alone on
      LargeData.one, Large Desktop.one and Large OneDrive.one; the
      dispatcher over mail's 2.1 GB scale store (peak memory, peak disk
      with removal on and off, time beside email alone); a resume run
      (stopped midway, resumed by cursor, against a whole read); the
      concurrent run (1, 2, 4, 8 workers over plaso's store, the scale
      store, a byte copy of it and mail's fixtures, one file-backed
      index: throughput, dedupe hits, peak memory and disk). Known
      limits: OneNote for the web can't export a notebook, so there is
      no authored OneNote truth; the CLI's memory unbounded (#48);
      mail's .eml cap at 4 GiB and .msg nesting cap at 16; office's
      whole-document model; the Threader's memory per message; legacy
      OLE2 macros undetected; routes without a processor yet.
  23. README, package docs, CHANGELOG v0.6.0. Pending edits: the
      onenote task's repos become mail, processing and root; formats
      gains text, xlsb, WordPerfect, ODG, tar, gzip, zstd, 7z and OLE2
      macro detection, and loses zip; audio gains the audio route's
      processor; onenote-fixture dropped, with a backlog note for an
      authored notebook (desktop OneNote on Windows or a Mac); the
      onenote README's stale "later release" line; ai-strategy's
      processing and onenote bullets; roadmap notes for ClamAV as a
      potential future check, a tool-wide memory limit, mail's .msg
      nesting past 16, office per-sheet output, and Threader spilling.
      The root's evidence 17 links the finding.

Test seams    mail's built binary over its fixtures, whole against
              resumed; route.File and Identify over signature samples;
              picture on the real magick; the inspector over ~/docs'
              hazards; onenote's Reader over the real pinned binary and
              a stand-in for exits 1 to 4; Dispatch and the pool over
              the fixtures, the main seam, which the integration and
              findings tiers drive too.

Slices        1.  mail cursor (behavior 1).
              2.  mail v0.3.0 (behavior 2). Checkpoint: a mail checkpoint
                  brief; on the architect's acceptance, mail's branch is
                  published, merged and tagged v0.3.0, ahead of SHIP.
              3.  route (3, 6 part).
              4.  picture (4, 6 part). Checkpoint: the ink PNG.
              5.  Internal CLI contract and README convention (5, 6).
                  Standards-review pass over 3 to 5.
              6.  Inspection (7, 8). Checkpoint: the flags over ~/docs'
                  hazards.
              7.  onenote contract and stream, fixtures copied, ~/docs
                  area (9, 12).
              8.  onenote typed content and round trip (10).
              9.  Page media and routes (11).
              10. Dispatcher: Part, events, stage interface, adapters,
                  top-level hashing, index (memory and file, claims),
                  cycle, seen, release after consumption (13 to 17).
                  Standards-review pass.
              11. Zip stage (18, quarantine rules of 7).
              12. Pool and resume: pool, email After, Threader save and
                  restore, pinning mail v0.3.0 (19, 20).
              13. Integration tier on gb10-2 (21). Checkpoint: the
                  routes-to-stages log.
              14. Finding (22); the scale store and its copy generated
                  outside any checkout. Checkpoint: the tables.
              15. Release preparation (23, processing). Standards-review
                  pass.
              16. Root: evidence 17's link (23, root).
              Every slice: the check passes.

Out of scope  Any inference in processing. Processors for audio, video,
              text, ODF, RTF, HEIC/WebP/AVIF, xlsb, WordPerfect, ODG,
              tar, gzip, zstd, 7z. OLE2 macro detection, oletools,
              ClamAV. Running macros or scripts, ever. A cursor for
              onenote or office. Office per-sheet streaming; Threader
              spilling; a tool memory bound. Chunking content for
              embeddings (the capstone's chunker). The authored OneNote
              notebook and any onenote release. Microsoft Graph.
              Rewriting fixture bytes. gb10-2 changes.

Door          One-way: mail v0.3.0, published at slice 2's checkpoint by
              the architect's decision (round 5, answer 6), ahead of the
              session brief; processing v0.6.0, which publishes the
              moved APIs and the dispatcher's contract. Two-way
              otherwise.

Release       mail v0.3.0 (at slice 2's checkpoint); processing v0.6.0
```

## Progress

slices 14/16 committed · mail released: PR #3 merged (eb73407), v0.3.0 tagged and published · processing: aa71cc8 route; bb315b4, 786a64d picture; b855570, 4836e73, 10241b2 cli contract and convention · standards pass 3–5 done (283298b, d4810c3) · f6fb8fe, aadb24c inspect; 809ce5c onenote contract; d6beab2 typed content; f8512da page media; d51cacc, 5764fd7 dispatch core; a7a9855, 8a0f8c6 adapters · standards pass 6–10 done (4f1f1f4, 2360399) · f4d8f65, b2a8ee5 zip · d17e321, 9f202e4, cf52e5c pool and resume · 70dd654, 7b3caf7 outputs stream (12b) · fd41110, 023ec96 disk and children bounds (12c); 98b5d33 no-stage content released · e89175c, 3ee9173 integration tier · 96797c6, 256e94c finding · editor — (notes: ~/docs README layout table needs onenote; context/README.md:53 content wording)

## Decisions

- setup: four repositories: spike-ai-pipeline (root: record, host scripts, consumer check, later
  the capstone), -client (Go), -processing (Go), -office (.NET Open XML CLI).
- setup: Go repositories copy spike-model-hosting's setup (mise go 1.27.3, golangci-lint 2.15.0,
  hermetic check, currency, integration against `SPIKE_ROUTER_URL`, gb10-2's router by default).
- setup: -office pins .NET 10 in mise; its check runs format, build and xUnit v3 tests with
  warnings as errors, and a NativeAOT publish; releases attach linux-x64 and linux-arm64 binaries.
- setup: all five Apache-2.0. The layers tag plain vX.Y.Z with a CHANGELOG and get CI (check on
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
- office-cli (architect, brief redirect): the markitdown and xberg comparison is closed: runs
  2026-10-31-1 and -2 and the findings page's three-tool tables stay as evidence 12's recorded
  answer, measured once at markitdown 0.1.9 and xberg 1.3.10. From now on `mise run baseline`
  measures office alone (fidelity against the truth, time, cold start, peak memory, size); the two
  tools leave mise.baseline.toml, the bench, and currency. Behavior 12 becomes office's own
  baseline. Rejected: removing the comparison with the tools.
- office-cli (redirect): run 2026-10-31-3 is office alone (fidelity 1.000 on every kind, median
  file 17.21 ms, peak 35.32 MiB, 11 s end to end); results.json moves to version 3. An uncommitted
  three-tool run found at runs/2026-10-31-3 before the redirect was moved, not deleted, to
  .state/baseline/stray-run-2026-10-31-3/.
- office-cli (architect, brief redirect 2): a baseline run keeps findings only with
  `--record [name]`, which writes docs/findings/office-cli/runs/<name>/ (default the date-stamped
  name); without it results stay in .state/baseline/. Two runs are kept, renamed: 2026-10-31-2 as
  `comparison` (the latest three-tool run) and 2026-10-31-4 as `office` (the latest office-only
  run, whose numbers the page now cites); -1, -3, and the stray 19:15 run are removed. Rejected:
  the stray 19:15 run as the comparison.
- office-cli (ship): office v0.1.0 released from b9bb1a2 after main's CI passed on ubuntu-24.04
  and ubuntu-24.04-arm; the release went draft → three assets (both tar.gz and SHA256SUMS,
  `sha256sum -c` OK, each holding the binary and LICENSE) → published; the linux-x64 binary
  prints 0.1.0. The linux-arm64 binary is 30,402,544 bytes, from pull request 1's CI run.
- office (round 1): processing gets office's binary from a mise pin,
  github:JaimeStill/spike-ai-pipeline-office 0.1.0 (checked against the release's digest), run
  through the exec runner. Rejected: a container image built from the release asset (it would
  need building and publishing, which documents rejected); Go code that downloads the asset
  and checks it.
- office (round 1, architect: "what about when it is emitted as markdown?"): office's JSON is
  decoded into typed Go values mirroring schema "1", with encoding/json/v2, and any other schema
  is refused. Markdown is plain text, since the structs type only the JSON. Its picture links
  name the media block's `name`, so the media list ties the two together. Rejected: raw JSON
  with only the media typed; a document model shared with PDF pages.
- office (round 1): the adapter offers both renderings, JSON and Markdown, one office run each.
  Rejected: JSON alone, with Markdown rebuilt in Go.
- office (round 1): the adapter is its own package, office, split from document for growth (the
  schema "1" types, legacy conversion, pst's attachments to come). Rejected: inside document.
- office (round 1): one conversion call handles legacy files when the adapter is built with an
  optional LibreOffice converter, as Preparer's Corrector is optional, and refuses them without
  one. Rejected: a separate converter the caller runs first.
- office (round 1, architect: "let's give these a try and if we run into any hurdles, we may have
  to explore building our own"): LibreOffice runs in gotenberg/gotenberg:8.39.0-libreoffice
  first, with linuxserver/libreoffice:25.8.7.3-r0-ls8 as the fallback. Building our own image is
  an escalation, not planned work. Rejected: building an image now.
- office (round 1, architect: "should we settle on a single standard for ALL images?", taken up
  in round 2): pictures vision can't decode are converted to PNG, EMF and WMF through
  LibreOffice and TIFF through magick. Rejected: leaving them out of vision as a known limit.
- office (round 1): the vision tier sends each distinct picture once to Gemma 4 26B-A4B on
  gb10-2, asserting completion and recording time and image tokens; charts are never sent.
  Rejected: scoring replies against hand-written descriptions; a final step over the Markdown.
- office (round 1): processing's OOXML fixtures are a handful of office v0.1.0's files, copied
  byte for byte with a sources file and license notices. Rejected: generating processing's
  own; reading office's repository at test time.
- office (round 1): legacy fixtures come from both sources: office's generated OOXML saved as
  legacy files by LibreOffice, with loss measured as the difference between models; and about
  three Office-written Apache POI files, one per format, against hand truth. Rejected: either
  source alone.
- office (round 1): the container's footprint for evidence 12 is the image's compressed and
  on-disk size, plus each conversion's wall time on the first run and on a repeat. Rejected:
  peak memory from the container's cgroup.
- office (round 2): one picture standard: pictures vision decodes (PNG, JPEG, GIF, BMP) are
  handed on as stored, only EMF, WMF and TIFF are converted to PNG, originals are kept, and no
  stage ever re-encodes a picture lossily. Measured: a JPEG decoded to PNG is pixel-identical
  (AE 0) and 4.9 to 8.6 times larger on photos, at the same token cost. Rejected: PNG for every
  picture with the original kept; PNG for every picture with the original dropped.
- office (round 2): a converted picture is named by its original's SHA-256 with .png. Rejected:
  the SHA-256 of the PNG's own bytes, which changes whenever the converter's version does.
- office (round 2): the adapter rewrites each Markdown link to a converted picture, swapping the
  exact <sha256>.<ext> name, and both results carry the media list (stored name, file handed
  onward, content type). Rejected: Markdown as office wrote it, with a map alone.
- office (round 2, after the architect asked how documents tested inference): the vision tests
  are office's own small integration test, with its own copy of the router client. Each picture
  is one request to Gemma 4 26B-A4B, asserting completion and recording time and image tokens,
  with no stand-in tier. Rejected: a shared test package under the integration tag; moving
  document's harness into one.
- office (round 2): the hermetic check runs a stand-in office for exits 1 to 4 and the warnings,
  and the real mise-pinned binary over the OOXML fixtures. Rejected: the stand-in alone with the
  real binary only in the integration tier, as document does for gs and magick; the real binary
  alone.
- office (round 2): processing ships the LibreOffice image as a default, pinned by digest, that a
  caller may replace, beside soffice's spec. Rejected: the spec alone, with each caller
  supplying the pin.
- office (brief): a legacy file is one with the OLE2 signature and a legacy extension in the
  request's file name (.doc and .dot to docx, .xls and .xlt to xlsx, .ppt, .pps and .pot to
  pptx). Any other OLE2 file, such as an encrypted OOXML package, goes to office to be refused.
  Rejected: reading the compound file's stream names in Go.
- office (escalation, slice 4): soffice's PNG export of an EMF or WMF writes the whole Draw page
  (794x1123 A4, the picture at 96 dpi in its middle). An EMF or WMF is rendered at a scale through
  draw_png_Export's PixelWidth/PixelHeight and trimmed to the picture with magick -trim +repage, PNG
  to PNG; it then needs both LibreOffice and magick. Measured: with-drawing.xlsx's three pictures
  trim to 256x179, 224x136, 180x193 at 96 dpi and 767x536, 672x405, 540x577 at 3x. Rejected: the
  page as written (~400 tokens for ~20 of picture); the 96 dpi trim (too coarse); a UNO macro
  through GraphicProvider (untested, changes how soffice runs).
- office (escalation, slice 4): the scale is a per-request value, 3 when unset, since vision's
  cost follows area (STANDARDS.md: a tool's options are per-request values). Rejected: a fixed
  scale.
- office: a conversion makes a fresh media directory and removes everything it made when it
  fails; content types come from a fixed extension table; an unknown inline kind is kept as an
  unknown block is; office's warnings and messages lose office's prefix and the spooled path.
- office: LibreOfficeMinimum is 24.2, Ubuntu 24.04 LTS's release; only 26.8.1.1, the default
  image's, is tested.
- office: soffice's stderr is never a Result warning, and a LibreOffice failure has no sentinel
  error; it carries soffice's message.
- office: soffice is given an explicit export filter, so a template (.dot, .xlt, .pot) exports
  as a document; a legacy file is spooled with its extension.
- office: the generated legacy set is made by generate.sh in the pinned image and isn't
  byte-reproducible, since LibreOffice stores the save time; MANIFEST records the committed sums.
- office: the metafile page is A4 through an en-GB profile, at most MaxMetafileScale 8; a blank
  metafile trims to 1x1 and passes; a multi-image TIFF converts frame 0; other kinds (SVG, ICO,
  WDP) are handed on as stored; converted PNGs are byte-reproducible (magick -define
  png:exclude-chunks=date,time).
- office: the vision tier dedupes by office's stored name and sends one request at a time, at
  temperature 0, output capped at 512 tokens.
- office: the TIFF fixture is built from POI's testtiff.tif, since no OOXML fixture of office's or
  POI's holds a TIFF office extracts.
- office (standards review): TestLibreOfficeImage is removed: it asserted a constant against
  itself.
- office (architect, brief redirect): pictures are listed and sent to vision as the document holds
  them, with no layout, since office's schema "1" has no position: a covered, off-slide, or
  cropped-away picture is still handed onward. Recorded as a known limit (README, CHANGELOG,
  findings), found by the architect in slide-objects' slide 3 (a red square under a blue one).
  Rejected: detecting overlap in processing (needs positions in office's model, a contract change).
- setup: five repositories, superseding four: -mail (Rust mail CLI) joins the layers; task pst
  splits into mail-cli (tags mail v0.1.0) and email (tags processing v0.4.0), before audio.
- setup: -mail pins Rust 1.99.1 in mise (minimal profile, clippy, rustfmt, x86_64 and aarch64
  musl targets); its check runs shellcheck, cargo fmt --check, clippy -D warnings, cargo test
  --locked, and a static musl release build; no cargo-deny. Currency: `cargo update --dry-run
  --verbose` plus mise tools and Action pins. Release workflow hand-written in office's shape.
- pst (round 1): readpst denied: last release 2021. Rejected: libpff/pffexport (alpha by its
  README, undocumented text layout, releases two to three years apart), go-pst (no release since
  2023), Aspose.Email and Independentsoft (paid).
- pst (round 2): .pst through Microsoft's outlook-pst 1.2.0 (MIT) in a CLI of the spike's own;
  .msg in scope, through msg_parser 0.3.6 (MIT) in the same CLI, one message model. Rejected:
  MsgReader (a second CLI and language, NativeAOT unproven), extract-msg (Python, GPL).
  (msg_parser superseded by mail-cli round 1: cfb and the CLI's own MS-OXMSG reader.)
- pst (round 2): processing's package is `email`; it names each attachment's route (PDF to
  document, OOXML and legacy office to office, pictures to vision, embedded mail nested, else
  unrouted) and the caller runs the stage, proven in the integration tier.
- pst (round 2): fixtures are public (libyal outlook.pst, java-libpst stores, Apache POI and
  MsgReader .msg files) plus .eml sources generated on Linux into .pst (PST Builder) and .msg
  (MsgKit), SHA-256s recorded. Rejected: Purview export (needs M365 E3/E5); no Windows.
- plan (mail layer): .ost is a known limit, refused with a reason: outlook-pst reads no .ost
  (client signature "SO"); an .ost is a server mailbox's cache, exportable as .pst.
- plan (mail layer): static musl binaries; processing runs the CLI from a mise github pin through
  the exec runner, as office.
- plan (mail layer): the CLI writes the folder tree, each message in its stored folder, and
  threading fields as stored, grouping nothing; processing's email builds threads across folders.
- plan (mail layer): evidence 16, mail stores, in the root's context/README.md.
- mail-cli (round 1): .msg is read through cfb by the CLI's own MS-OXMSG reader, sharing one MAPI-property mapping with .pst. Rejected: msg_parser 0.3.6 (no embedded messages, no In-Reply-To/References/conversation fields, drops non-UTF-8 ANSI); an upstream fix. Supersedes pst round 2's msg_parser line.
- mail-cli (round 1): v0.1.0 ships with .pst embedded messages as a known limit, listed with metadata and no content; method-5 attachments in a .pst are never opened (outlook-pst 1.2.0 fails or deadlocks; PR #65 unmerged). Rejected: holding the release.
- mail-cli (round 1): stdout is JSON Lines, schema "1": store line, folder lines depth-first before their messages, message lines, skipped lines. Rejected: one JSON document per input.
- mail-cli (round 1): an unreadable folder, message or attachment is skipped with a warning and a skipped line, panics caught per record, exit 0; exit 3 for an unreadable header or root. Rejected: failing the whole store.
- mail-cli (round 1, with round 2 Q1): HTML recovered from RTF (MS-OXRTFEX) is a derived body, the CLI's own code over compressed-rtf. Rejected: stored bodies alone.
- mail-cli (round 1): every item class is emitted with the common fields only. Rejected: mail classes only.
- mail-cli (round 1): .pst threading truth from Tika's testPST.pst and testPST_variousBodyTypes.pst (Apache-2.0); PST Builder covers the rest. Rejected: extending PST Builder upstream; dropping generated .pst.
- mail-cli (round 1): ANSI stores are read as an untested known limit. Rejected: refusing them.
- mail-cli (round 1): password-protected stores are read like any store. Rejected: refusing them.
- mail-cli (round 1): arguments parsed by hand on std. Rejected: lexopt; clap.
- mail-cli (round 1): evidence 16 measures the CLI alone. Rejected: a readpst/pffexport baseline.
- mail-cli (round 1): fixtures generated by PST Builder, MsgKit and a small C# program in a separate mise environment pinning .NET 10, never run by the check; outputs committed with SHA-256s. Rejected: a .NET SDK container.
- mail-cli (round 2): every message carries a derived Markdown body rendered by htmd (Apache-2.0) beside the stored bodies and the HTML from RTF. Rejected: no Markdown; Markdown in place of HTML (loses cid: links to inline pictures).
- mail-cli (round 2): code pages decoded by codepage-strings 1.0.2, a finished wrapper over maintained codepage, encoding_rs and oem_cp (it meets the maintained-dependency rule through them). Rejected: encoding_rs with codepage directly.
- mail-cli (round 2): evidence 16 reports time and peak memory on plaso's 59 MB store and a ~50k-message, ~2 GB store PST Builder generates from a seed, never committed. Rejected: plaso's store alone.
- mail-cli (round 2): release tags v0.1.0 in spike-ai-pipeline-mail; --version prints bare "0.1.0", checked against the tag, as office does.
- mail-cli: an HTML body's code page is PidTagInternetCodepage, else the charset its `<meta>` declares in its first 1024 bytes, else UTF-8 when the bytes are valid UTF-8, else the message's code page, then the store's; PidTagBodyHtml in PT_STRING8 is decoded the same way. A `<meta>` label resolves through encoding_rs's WHATWG labels and codepage's map, both now direct dependencies.
- mail-cli: US-ASCII (20127), which codepage-strings doesn't know and libyal's store names for its HTML, is decoded by the CLI itself; a byte past 0x7F becomes U+FFFD with a lossy-decoding warning.
- mail-cli: a .pst's store code page is PidTagMessageCodepage, else PidTagCodePageId, else Windows-1252; a message's PT_STRING8 uses its own PidTagMessageCodepage, else the store's.
- mail-cli: a .msg's PT_STRING8 is decoded by the message's code page, else its locale's ANSI code page (PidTagMessageLocaleId), else the internet code page unless it is a Unicode encoding, else the parent message's, else 1252, as POI's guess7BitEncoding orders it. Rejected: the internet code page second (slice 5), which decoded POI's 1252 strings under a UTF-8 internet code page to U+FFFD.
- mail-cli: HTML from RTF follows MS-OXRTFEX 2.2.3 and skips `\*\mhtmltag` (2.1.3.1.5), since the `\*\htmltag` beside it keeps the original tag with its `cid:` reference; per-font code pages are not tracked.
- mail-cli: the Markdown body (htmd 0.5.6) skips head, style and script; keeps `cid:` image sources; sets plain text as HTML paragraphs so htmd escapes it, then escapes what htmd leaves (|, ~, &, a leading "N)", a line-leading -=+#>*).
- mail-cli: outlook-pst =1.2.0, codepage-strings =1.0.2 and compressed-rtf =1.0.1 are pinned exactly, since the CLI works around faults of those releases (Cargo.toml names where); compressed-rtf's header sizes are checked before it slices by them.
- mail-cli: codepage-strings 1.0.2 is kept as round 2 decided, though the standards review flagged its last release (2022).
- mail-cli: sha2 0.11.1 (RustCrypto) hashes attachment content.
- mail-cli: schema "1"'s shapes: every line's kind is its `line` field; stored bodies sit at `body`'s top level and the CLI's own under `body.derived`; an embedded .msg message is the attachment record's `message`, with no `line`, file or size; attach methods are written by snake_case name (by_value … by_web_reference, none), an unknown one left out with a warning.
- mail-cli: an attachment's stored extension is PidTagAttachExtension, else the filename's, kept when 1 to 8 ASCII letters and digits, in its stored case; inline is ATT_MHTML_REF; size is the content's byte count, not PidTagAttachSize.
- mail-cli: times are written to the second; a folder's path is an array of display names from the IPM subtree's root; importance and sensitivity are words; the subject's U+0001 prefix marker is dropped; --attachments naming an existing non-directory is a usage error; arguments are read as OsString, so a non-UTF-8 path keeps the exit contract.
- mail-cli: unreadable folder tables give skipped lines of kind "contents" and "hierarchy", detected only when PidTagContentCount is above 0 or PidTagSubfolders is true, since outlook-pst 1.2.0 returns an unreadable table as none; PidTagAssociatedContentCount is not consulted (the CLI never reads the FAI table).
- mail-cli: after a caught reader panic the CLI reopens the store (and reopens an open message before its next attachment), since the panic poisons outlook-pst's file mutex; a .msg reopens its file the same way.
- mail-cli: .msg nesting is capped at 16 embedded messages; one deeper is listed without its message, with a warning.
- mail-cli: in a .msg, afStorage is written as its storage copied into a compound file of its own; an empty stream is read from its directory entry alone (cfb 0.16.0 can't read one at sector 0xFFFFFFFF, which OpenMcdf writes); named properties are read and warned about but unused.
- mail-cli: four Apache POI .msg fixtures (apache/poi 81ab5262) plus poi-truth.json (31 checks from POI's HSMF tests) cover behavior 11's POI part; Tika's .msg truth is Tika's own OutlookParserTest assertions, not values read from the CLI.
- mail-cli: an Exchange address is written as stored (PidTagEmailAddress), its SMTP address not at all; Tika's testMSG_chinese.msg To address scores a mismatch, recorded as a known limit.
- mail-cli: the generated fixtures are not byte-stable (PST Builder's random UIDs and search keys; MsgKit's random keys and conversion time), recorded in the manifest; their content is checked by the truth tests and goldens.
- mail-cli: the generator pins .NET SDK 10.0.402 (mise.fixtures.toml, global.json with rollForward disabled) and MimeKit 4.18.2 over the resolved 4.18.1, whose System.Security.Cryptography.Pkcs advisories fail the restore; currency scans the fixtures environment and the NuGet packages.
- mail-cli: the scale mode writes its own MIME from one seeded System.Random and pins the content SHA-256, not the .pst's bytes; its folders omit "Deleted Items", which PST Builder makes itself.
- mail-cli: peak memory is measured by office's rusage.c (wait4's ru_maxrss); a glibc build is timed beside the released musl binary, never released.
- mail-cli: the findings run v0.1.0 was recorded again at 32e5aa9, for the binary to be tagged, after the POI group was added; plaso's commit is pinned once, in scripts/findings.sh.
- mail-cli (standards review): Actions are pinned by exact release tag, as the sibling layers pin them, replacing slice 6's SHA pins. SHA pinning would be a workspace-wide change to rolling-currency.md, not made here.
- mail-cli: the release's build, version gate and pack live in scripts/release-pack.sh, so they run locally under shellcheck; the assets are spike-ai-pipeline-mail-<tag>-linux-x64.tar.gz and -linux-arm64.tar.gz (binary and LICENSE) and SHA256SUMS; actionlint is not added to the check.
- email (round 1, q1): llama.cpp held at b11669 on gb10-2, awaiting verify-gb10's dated report on gb10-1. The root's upgrade takes uv 0.13.7
  and huggingface_hub 2.4.0 only. Rejected: upgrading now, because the Gemma-4 FFN fix would change vision outputs mid-task.
- email (round 1, q2): threads follow RFC 5256 REFERENCES (the standardized JWZ) with base-subject gathering. A message whose headers name
  no parent links by its conversation index minus the last 5-byte block (MS-OXOMSG 2.2.1.3). Written on the standard library. Rejected:
  gatherstars-com/jwz v1.4.0 (unmaintained, no license, TUI dependencies, outside depguard); go-imap v2 (beta, server THREAD only); headers
  alone; the index and topic alone.
- email (round 1, q3): the Reader streams records as an iterator while the CLI runs, and threads come from the collected keys alone.
  Rejected: the whole store in memory.
- email (round 1, q4): a route is decided by content signature, with the extension deciding only within ZIP or OLE2. Rejected: the stored
  MIME type and extension.
- email (round 1, q5): TIFF, EMF and WMF route to vision through office's picture conversion, exported additively in v0.4.0. Rejected:
  unrouted.
- email (round 1, q6): a by-value .msg file routes as nested. A by-value .eml is unrouted, a known limit.
- email (round 2, 7′ b): mail v0.1.0's fixtures are copied into processing's testdata, pinned by a SHA-256 manifest, with a mise refresh
  task. Plus one .msg built once by mail's generator from processing's .eml sources (a docx, a legacy xls, a TIFF, an EMF). The check
  stays hermetic. Rejected: mail's fixtures alone; extending mail's fixtures in a mail v0.1.1.
- email (round 1, q8): the finding scores the fixtures (Tika's truth), counts plaso's store, and measures time and peak memory on plaso's
  store and the scale store.
- email (brief): the slice order is processing's upgrade, the root's upgrade, the build, then the root's evidence 16 link.
- email: the Stream holds the read's context, as sql.Rows does; Records takes none.
- email: placeholders follow RFC 5256's pruning: one survives only as a root holding several
  children, and a message with no references and no index parent is always a root.
- email: a route is a string type, `email.Route`: document, office, vision, vision-converted,
  nested, or unrouted. A by-value .msg is decided by its OLE2 signature and .msg extension, without
  walking its MS-OXMSG streams.
- email: office.ConvertPicture is a method on *Converter. It identifies the format by signature,
  copies the picture into Dir, since the container runner mounts only Dir, and fails before
  anything runs without its tool (ErrNoLibreOffice, ErrNoMagick).
- email: mail's goldens are copied with its fixtures (4.8 MB), pinned by MANIFEST; `fixtures:mail`
  refreshes them from the remote tag.
- email: the built .msg carries five attachments: the four office fixtures and, by value, mail's
  03-html-inline-image.msg as forwarded.msg, so the nested route's second read runs on real
  content (spec review round 1 gap).
- email: Reader.Read returns (*Stream, error) and refuses an invalid request with email.ErrRequest
  before mail runs.
- email: the vision tests' router client moved into internal/visiontest, built under the
  integration tag alone, and document, office, and email share it.
- email: tool's container leak check filters by the ffmpeg image, since office's and email's
  LibreOffice containers run in parallel.
- email: the findings tier has its own build tag, `findings`. A run keeps its results only when
  EMAIL_FINDINGS_RESULTS names a directory. Peak memory is ru_maxrss, read through a relaunched
  test binary.
- email: gb10-2's apt upgrades were pending during the integration and findings runs, since
  host-update needs the architect's root steps; the router, llama.cpp b11669 serving set A, was
  untouched.

- plan (2026-11-03): .eml and mbox are read by the mail CLI (mail-mime, mail v0.2.0) through Stalwart's mail-parser 1.0.1, then
  taken by processing (email-mime, processing v0.5.0); audio re-tags v0.6.0 and streams v0.7.0. Rejected: mailparse 0.18 (no mbox,
  no nested parse); mbox-reader (unmaintained); Go in processing (few encoded-word charsets, no mbox, duplicated naming and Markdown).
- plan (2026-11-03, q1): gb10-2-align runs after email-mime and before audio, so audio measures ASR on b12063. Its target is b12063,
  gb10-1's adopted pin, not currency's b11960.
- plan (2026-11-03, q2, q3): spike-scanned-documents runs on gb10-2 once this spike returns to `planned`, first among experiment.ai's
  planned spikes.
- plan (2026-11-03, q4): task formats (ODF/RTF, HEIC/WebP/AVIF, multi-page TIFF, encrypted PDFs, PDF forms and embedded files, zip
  recursion) after streams, processing v0.8.0, no new evidence item.
- plan (2026-11-03, q5): web content (HTML, MHTML, WARC, EPUB) waits as a roadmap comment until the capstone needs it.
- plan (2026-11-03, q6): ai-strategy.md's mail bullet applied in the plan pull request, extended for .eml and mbox.
- plan (2026-11-03, q7, q8): an mbox has no folder line; each message's id is its 1-based ordinal in the file. Rejected: an invented
  folder; the byte offset.
- plan (2026-11-03, q9): an .eml is accepted by an RFC 5322 header-shape check (first line a field, a From or Date field), else exit 3;
  email-mime's nested-.eml route uses the same rule. Rejected: anything mail-parser parses.
- plan (2026-11-03, q10): MIME parts labelled US-ASCII or ISO-8859-1 decode as WHATWG windows-1252; .pst and .msg code pages keep
  mail's U+FFFD-plus-warning rule.
- plan (2026-11-03, q11, q12): Thread-Index and Thread-Topic fill conversation_index and conversation_topic, with a generated fixture
  carrying both; Importance, X-Priority, and Sensitivity fill importance and sensitivity.
- plan (2026-11-03, q13): the schema stays "1", additive. Rejected: "2".
- plan (2026-11-03, q14): mail-mime's fixtures are mail's 8 generated .eml with truth.json, a generated mboxrd from them, and mime4j's
  mbox.rlug. Rejected: CPython's test_email data (real names, truth still to write).
- plan (2026-11-03, onenote, q1): OneNote gets its own Rust layer repository, spike-ai-pipeline-onenote, with mail's setup copied.
  Rejected: a subcommand of the mail CLI.
- plan (2026-11-03, onenote, q2): onenote_parser 2.1 (MPL-2.0; desktop OneNote 2016+ and OneDrive files since 2.0.0) is used
  unmodified. Rejected: an [MS-ONESTORE] reader of our own (several sessions, as [MS-PST] would have been); pyOneNote (embedded files
  only); Apache Tika's parser (Java, text only, desktop only).
- plan (2026-11-03, onenote, q3): inputs are .one sections, .onepkg exports, and notebook directories through their .onetoc2.
- plan (2026-11-03, onenote, q4): ink is written as its recognized text and its strokes as SVG, which processing rasterizes through
  magick for vision.
- plan (2026-11-03, onenote, q5): each page at its latest revision only; the recycle bin is skipped with a warning.
- plan (2026-11-03, onenote, q6): fixtures are onenote.rs's MIT samples, never its AGPL ones, plus a notebook the architect authors in
  OneNote with hand truth, since no free OneNote writer runs on Linux.
- plan (2026-11-03, onenote, q7): files only. Rejected: Microsoft Graph's OneNote API, the capstone's ingestion if anywhere.
- plan (2026-11-03, onenote, q8): two tasks, onenote-cli (onenote v0.1.0) and onenote (processing v0.6.0), under evidence 17.
- plan (2026-11-03, onenote, q9): the OneNote tasks run after email-mime and before gb10-2-align; audio re-tags v0.7.0, streams
  v0.8.0, formats v0.9.0.
- mail-mime (2026-11-04, r1 q1): marathon-architecture 0.3.0 is applied; an extension's "targets" version is a minimum, not
  an exact version.
- mail-mime (2026-11-04, r1 q2): mail-parser's windows-1252 decodes 0x81, 0x8D, 0x8F, 0x90 and 0x9D to U+FFFD, with a
  lossy-decoding warning; this narrows plan q10's "WHATWG windows-1252" by those five bytes. Rejected: exact WHATWG through
  encoding_rs, which needs a re-decoding path and an RFC 2047 decoder of mail's own.
- mail-mime (2026-11-04, r1 q3): the new .eml and the mboxrd form a new MIME fixture group with its own hand truth and SHA-256
  list; the mboxrd is written by a shell script under a mise task, rebuilt by the check and compared byte for byte; the
  generator's .eml set and the .pst and .msg are untouched. Rejected: the generator's set, whose rebuild changes every
  committed .pst and .msg. No standard tool writes mboxrd (Python's mailbox writes mboxo).
- mail-mime (2026-11-04, r1 q4): the generator's scale mode also writes its 50,000 seeded messages as an uncommitted mboxrd,
  timed beside the scale .pst in a findings section. Rejected: timing the committed fixtures alone.
- mail-mime (2026-11-04, brief): the planner-filled choices stand as the brief states them: store formats "eml" and "mbox";
  sender, times, transport headers, bodies and inline; embedded messages to depth 16; Importance over X-Priority; the
  Sensitivity map; Thread-Index base64 to hex; the header-shape check per mbox message; mbox.rlug with its splits, LICENSE and
  NOTICE; mail-parser by caret requirement 1.0.1.
- mail-mime (2026-11-04, build): an mbox's blank line before each separator belongs to the separator, as RFC 4155 and
  mboxrd give it, and is dropped from the message before it. mime4j's expected splits of mbox.rlug keep it, so the rlug
  test drops exactly one trailing "\n" from each split before comparing every field but id.
- mail-mime (2026-11-04, build): every separator line counts toward an mbox message's ordinal; an entry with nothing
  after its separator keeps its ordinal and writes no line, so ids are places in the file and can skip a number.
- mail-mime (2026-11-04, build): the MIME fixture group is tests/fixtures/mime/: eml/09-threading.eml and its hand truth
  truth.json; nine.mboxrd (separator "From MAILER-DAEMON Thu Jan  1 00:00:00 1970", LF separators, CRLF messages,
  `*.mboxrd -text`), which scripts/mboxrd.sh writes (`mise run mboxrd`) and a cargo test rebuilds; rlug/ with mbox.rlug,
  its five splits, LICENSE and NOTICE; sources.json, which pins mime4j at c9847207a5aa3ce2526bfee78edf3dbae9c22e35; and
  sha256.json, which lists every other file in the group.
- mail-mime (2026-11-04, build): a message/global part (RFC 6532) is an embedded message too, like the brief's
  message/rfc822: method embedded_message, nested under `message` to 16 levels.
- mail-mime (2026-11-04, build): a Sensitivity of "Normal" is unknown, left out with a warning, since the brief maps
  only Personal, Private and Company-Confidential; a MIME message never gets sensitivity "normal".
- mail-mime (2026-11-04, build): exit 3's reason for an input of no format mail reads is "neither a .pst, a .msg, an
  .eml nor an mbox", where v0.1.0's was "neither a .pst nor a .msg".
- mail-mime (2026-11-04, build): repeated MIME headers are read last, except Received, whose first gives delivery_time;
  Message-ID, In-Reply-To and References are unfolded and trimmed, their angle brackets kept; an HTML-only message has
  no body.text.
- mail-mime (2026-11-04, build): an mbox message of 4 GiB or more gets no size check of mail's own; mail-parser can't
  parse it, so it is skipped with a warning.
- mail-mime (2026-11-04, build): findings run v0.2.0 was made at b3f2f8b, before the version was raised, so its binary
  prints 0.1.0. Its scale.pst median, 14.82 s, is 12% above v0.1.0's 13.27 s on the same store, the spread between two
  days' runs on the laptop, so scale.mboxrd (6.86 s) is compared only with the same run's scale.pst; v0.1.0's .pst and
  .msg tables stand, and v0.2.0 repeated their scores exactly.
- mail-mime (2026-11-04, build): a header's encoded words, such as a Subject's or Thread-Topic's, decode lossily with no
  warning, since mail-parser reports no decoding problems there: the five undefined windows-1252 bytes become U+FFFD
  silently. A body or attachment part warns. It is a known limit in mail's README and CHANGELOG; an RFC 2047 decoder of
  mail's own stays out of scope (r1 q2).
- email-mime: held llama.cpp at b11669: gb10-2-align moves it to b12063; b12110 (v0.9.0) resizes
  Gemma 4 images without letterbox padding, which would shift vision outputs mid-spike. Rejected:
  moving it in this task.
- email-mime (round 1, q2): the nested-.eml route's stored-type condition is message/rfc822 or
  message/global, parsed as a media type, as mail nests both. Rejected: message/rfc822 alone.
- email-mime (round 1, q3): a by-value .eml whose first line is an mbox "From " separator fails the
  header-shape check and stays unrouted, a known limit. Rejected: also accepting mail's mbox
  separator.
- email-mime (round 1, q4): the MIME inputs are checked by a same-run round trip against mail's own
  stdout, plus hand-written record kinds, ids and skipped records. Rejected: goldens written by
  processing; a mail v0.2.1 that ships goldens.
- email-mime (round 1, q5): thread hand truth covers the nine-message mboxrd, mbox.rlug, and each
  .eml as one root; the index-parent path stays covered by its unit tests and Tika's .pst.
  Rejected: a MIME fixture of processing's own with an index-only parent.
- email-mime (round 1, q6): the finding's addendum also times the adapter on mail v0.2.0's scale
  mboxrd beside mail alone. Rejected: scoring the fixtures only.
- email-mime (round 1, q7): each route condition is proven by unit tests, plus 08-forward's .msg on
  the real binary; fixtures:built stays at mail v0.1.0 and the built .msg is not rebuilt.
  Rejected: rebuilding the built .msg with an .eml stored as application/octet-stream.
- email-mime (brief): the built .eml and mbox.rlug's five splits are read as MIME inputs.
- email-mime (2026-11-04, build): host-update stopped on gb10-2's root steps; the architect ran the apt upgrade
  (tailscale and nvidia userland among 28 packages) and regenerated the CDI spec, but the reboot it left pending waits
  until the architect can log in after it. The router was untouched and serves set A loaded (all four models), so the
  integration tier runs against it as it stands; the next session's host-update asks for the reboot.
- email-mime (2026-11-04, build): the hermetic check reads 17 MIME inputs beside mail's 22 .pst and .msg
  fixtures, 39 in all; the plan's 16 and 38 were an arithmetic slip. The findings tier reads 40, the
  built .msg among them.
- email-mime (2026-11-04, build): TestBuiltManifest no longer requires the built .msg's generator tag to
  equal the fixture copy's tag, only that it names a tag and a full commit, since the built .msg stays
  built by mail v0.1.0's generator while the copy moved to v0.2.0.
- email-mime (2026-11-04, build): a MIME read's skipped attachments are tested only through the stand-in
  mail, since mail skips a MIME attachment only when its reader panics; a real skipped message and its
  ordinal gap are proven through the real mail on a temporary mbox (TestRealMboxOrdinals).
- email-mime (2026-11-04, build): the nested .eml check runs after every signature, so a signature still
  decides first (a PDF named .eml routes as a document) and v0.4.0's routes and reasons are unchanged; a
  malformed media-type parameter is ignored, the type still read.
- email-mime (2026-11-04, build): the integration tier reads the MIME inputs after the stores and sends
  each picture once by content across the whole run; every MIME picture had already been sent from a
  .pst or .msg, so the MIME inputs sent 0.
- email-mime (2026-11-04, build): the findings run read mail's existing .state/scale set in place, not
  regenerated: its content SHA-256 and mboxrd SHA-256 match mail's v0.2.0 records, and no generator file
  changed between the set's build (b3f2f8b) and the tag. Results went to
  docs/findings/email-results/mail-0.2.0/ so the first run's files stay. The .pst times are compared
  only within one run, since the first run read the scale store from tmpfs and this one from the page
  cache.
- email-mime (2026-11-04, build): the standards review replaced headerShaped's byte loop with bufio's
  ReadSlice over a 64 KiB buffer (88f485e), with the same behavior.
- onenote-cli: currency reported nothing trailing in spike-ai-pipeline-onenote (Rust 1.99.1, jq 1.8.2,
  shellcheck 0.11.0 at latest); no upgrade slice.
- onenote-cli (round 1, q1): public fixtures are Interop-TestSuites' MIT OneNote files (fe87ed32) and
  onenote.rs's maintainer samples under MPL-2.0 (v2.1.0, 727389a), each pinned with SHA-256 lists and
  their LICENSE; LargeData.one and the two 113 MB LFS sections are fetched for timing, never committed.
  Rejected: Joplin's AGPL samples; the MIT files alone.
- onenote-cli (round 1, q5): BUILD starts on the free samples and pauses at the authored-truth slice.
- onenote-cli (round 1, q6): stdout is a notebook line, section_group and section lines depth-first in
  TOC order, a page line per page with its content nested, and skipped lines. Rejected: a line per
  section; one document per input.
- onenote-cli (round 1, q7 / round 2, q6): a paragraph's content is runs with formatting flags; the
  page line carries the Markdown body. Rejected: offset ranges (UTF-16); a Markdown string per
  paragraph; both.
- onenote-cli (round 1, q8): math as LaTeX through a port of one2html's MIT math lexer, parser and
  LaTeX renderer, its notice kept; Markdown $…$ and $$…$$. Rejected: a LaTeX writer of our own; the
  stored text with markers stripped.
- onenote-cli (round 1, q9): ink SVG, one per page and one per paragraph holding ink, written by the CLI.
  Rejected: one per stored ink object.
- onenote-cli (round 1, q10): Markdown orders containers by position, top to bottom then left to right;
  offsets stay in the model. Rejected: stored order.
- onenote-cli (round 1, q11): mail's CLI contract with office's `--media <dir>`, `--version`, exits 0–4,
  input sniffed from content. Rejected: `--attachments`.
- onenote-cli (round 1, q12): a OneDrive .zip is refused with exit 3. Rejected: reading zips.
- onenote-cli (round 1, q13 / round 2, q7): a password-protected section is skipped inside a notebook
  and refused as the input, with the parser's bare reason, a known limit; [MS-ONESTORE] 2.5.19 leaves
  the scheme undocumented and no open decryptor exists. Rejected: a --password option now; a later
  reverse-engineering task. A roadmap note records decryption for later.
- onenote-cli (round 1, q14): the recycle-bin warning for directories and .onepkg, through cab 0.6.0
  directly. Rejected: directories only.
- onenote-cli (round 1, q15): a THIRD-PARTY notice for onenote_parser (MPL-2.0) and one2html (MIT) in the
  README and each release archive. Rejected: the README alone.
- onenote-cli (round 1, q2 / round 2, q1): the architect authors in OneNote for the web on Linux; no
  Windows. Desktop format rests on the public files with structural checks, no hand truth. Rejected: a
  Windows 11 evaluation VM for a desktop variant and a real .onepkg.
- onenote-cli (round 2, q3): the approved authoring spec, for OneNote for the web; [try] marks a feature the
  current web help doesn't confirm, which the architect tries and reports. Use only generated pictures and files;
  never paste a whole text container from another section (onenote_parser #39).
  - Notebook "Spike Fixture", created in OneNote for the web, alone in a new OneDrive folder.
  - Section "Text":
    - page "Formatting": Heading 1 and 2, Normal paragraphs; bold, italic, underline, highlight, font colour, and
      [try] strikethrough, superscript, subscript; Quote and Code styles; a web link; a link to the page "Lists";
      accented, CJK and Arabic right-to-left text; an indented paragraph.
    - page "Lists": bullets three levels deep; numbering nested as 1/a/i; [try] a list restarted at 5.
    - page "Tables": a 3×3 table with a bold first row, a cell of two paragraphs, a cell with a bullet list, an
      empty cell, a shaded cell; a second table with all borders removed.
    - page "Tags": To Do, To Do checked, Important, Question, Remember for later, Definition, and a tag inside a
      table cell.
    - page "Layout": three text containers, two side by side and one below.
    - a subpage under "Lists", and [try] a subpage under that one; an untitled page.
    - page "Revisions": type "first", wait for it to sync, then change it to "second".
  - Section "Media":
    - page "Pictures": a generated PNG with alt text, a JPEG, and a PNG of printed text after "Copy Text from
      Picture" is run on it.
    - page "Files": [try] attach a .pdf, .docx, .txt and .csv; [try] insert the PDF as a printout.
  - [try] Section "Ink", if a Draw tab exists: handwritten "Hello world"; a pen-and-highlighter drawing in two
    colours; handwriting inside a text container; [try] a handwritten a²+b²=c² converted to typed math, and a
    typed fraction. Never Ink to Text.
  - [try] Section group "Group A" holding section "Nested", and inside it group "Inner" holding section "Deep".
  - An empty section.
  - One page deleted, so Deleted Notes holds it.
  - Download the OneDrive folder as a ZIP and deliver it as downloaded.
  - Left out, since the web can't make them: custom tags, a locked section, audio, merged or nested tables, screen
    clippings, a .onepkg.
- onenote-cli (round 2, q4): the session drafts truth.json from the spec; the architect reports [try]
  outcomes and departures. Rejected: the architect writing truth.json.
- onenote-cli (round 2, q5): .onepkg is covered by MSZIP packages the tests build through cab 0.6.0's
  writer; no OneNote-written package is a known limit. Rejected: leaving .onepkg out of v0.1.0.
- onenote-cli (rounds 2–3): authored under the architect's personal Microsoft account; the display
  name "Jaime Still" stays (public as GitHub's JaimeStill). Rejected: a fresh account; a temporary
  rename; patching the files.
- onenote-cli (round 4): the account CID in OneDrive-format files' resolutionId block is kept as
  downloaded; the pre-commit scan stops only on the account email and lists CID hits in the manifest.
  Rejected: patching the CID; a fresh account.
- onenote-cli (brief): onenote_parser by caret 2.1.0; an image with no extension named by its hash
  alone; image OCR text in the model, not in Markdown; a parser panic exits 3; times to the second;
  ids from FileIdentity and the page GUID.
- onenote-cli (2026-11-04, build): host-update stopped at gb10-2's pending reboot, which the architect
  can't run remotely; the router was untouched and still serves set A. This task uses no inference.
- onenote-cli (2026-11-04, build): Schnelle Notizen.one is left out of the MPL set: a third party attached it to
  onenote.rs issue #11 with no licence grant. Ink recognition rests on handwriting_recognition.one.
- onenote-cli (2026-11-04, build): onenote_parser prints debug dumps to stdout on some malformed sections, so the CLI
  writes its lines to a duplicate of stdout and points descriptor 1 at /dev/null (libc dup2, one unsafe call).
- onenote-cli (2026-11-04, build): an encrypted section is refused with the parser's own "Unexpected end of file".
- onenote-cli (2026-11-04, build): a page's title area is a `title` container, first in content, since the parser
  keeps it apart from contents(). A table cell holds outline items, so a list inside a cell survives, and a table
  carries its tags.
- onenote-cli (2026-11-04, build): an empty table cell is written `{}`, the one empty object the line rules allow,
  since a cell holds its position in the row.
- onenote-cli (2026-11-04, build): ink drawings and bounds are in ink units (0.01 mm), page ink at its stroke
  coordinates plus offsets as one2html places it; transparency read as 0 opaque per ISF and one2html; colour as
  COLORREF, none as black; ink_text is lines of {text, alternatives, language} with the raw LCID. Colour,
  transparency, pen tip and paragraph ink have unit tests only, since no fixture holds them. Checkpoint render of
  handwriting_recognition.one's page read correctly.
- onenote-cli (2026-11-04, build): a math run is {"math":"display"|"inline","latex"} with no text or styling; a
  failed parse keeps math with the LaTeX of its marker-stripped text and warns, exit 0. The port adds finl_unicode
  1.5.1 ((MIT OR Apache-2.0) AND Unicode-DFS-2016) for general categories; the third-party notice names it too.
  The only committed equation is a=b; Math.one is in onenote.rs's AGPL joplin/ set, a local sanity read only.
- onenote-cli (2026-11-04, build): skipped section lines come after the notebook's entries, in report order, not
  in their TOC place (behavior 5): onenote_parser drops a failing entry and keeps only its message. A section that
  fails inside a section group is lost silently, since the parser discards the nested notebook's report, and a TOC
  entry whose file is missing gets no warning, since the TOC's entries aren't public and a directory-only check
  would break package/directory equality. All three are known limits for the README.
- onenote-cli (2026-11-04, build): a .onetoc2 input is named for its directory, a package for its stem; skipped is
  {"line":"skipped","kind":"section","name","reason"}; no committed notebook lists a section group, so the test
  composes one from committed samples.
- onenote-cli: typed-path stays at 0.12.4, since onenote_parser 2.1.0 requires ^0.12.3 and takes its types in its
  API; 0.13.0 (2026-10-20, after the plan's currency run) waits for onenote_parser to move.
- onenote-cli (2026-11-04, build): Markdown places an ink container by its strokes' bounds (stored offsets are 0,0),
  writes page ink once, drops PageDateTime lines (creation_time holds the time), splits lists with `<!-- -->`, treats
  a CheckBox shape as To Do, escapes mail's htmd rules plus `$`, and keeps OCR text out.
- onenote-cli (2026-11-04, build): slice 9 (CI) runs before slice 8 while the authored ZIP is awaited, and the hand
  truth is drafted from the round 2 spec in the scratchpad, committed only with the fixture in slice 8.
- onenote-cli (2026-11-04, build): CI copies mail's workflow (checkout v7.0.1, mise-action v5.2.6, cache v6.1.0);
  the merge waits through `gh pr checks --watch` chained before `gh pr merge`, office's form; currency gains mail's
  Action-pin scan. No ruleset, as mail has none. Rejected: office's committed ruleset applied through gh api.
- onenote-cli (2026-11-04, build): the authored notebook's truth draft, authoring sheet, generated assets and the
  cut/fill tools sit in the session scratchpad (truth-draft/); the architect authors from the sheet. A resuming
  session without that scratchpad regenerates them from the round 2 q3 spec. Slice 11 runs before 10 and 8, since it
  needs no authored fixture.
- onenote-cli (escalation, 2026-11-04): OneNote for the web wouldn't create a notebook, so v0.1.0 ships without
  the authored fixture. Behaviors 15 (its authored part), 17 and 18 and slice 8 are dropped; the finding (behavior
  20) scores against the public fixtures' hand-written expectations; "no authored hand truth" is a known limit; a
  later task commits an authored notebook and its truth in a v0.1.x release. Rejected: OneDrive's New menu or a
  mobile app now; holding the task in handoff.
- onenote-cli (2026-11-04, build): the finding, from run v0.1.0, recovered every expected field: 236 of 236 in
  sections.json, 138 of 138 in notebooks.json, and 129 of 129 in tests/fixtures/content.json, which gives content
  fields as JSON pointers, as mail's truth does. Large OneDrive.one is 161 MB, not the brief's 113 MB, and peaks at
  135 MiB in 0.24 s, as its desktop twin does, so #48 doesn't show on it; #48 stays a known limit.
- onenote-cli (2026-11-04, build): content.json and the Rust content tests both hold the content values, since the
  tests also check whole arrays that a pointer list can't express.
- onenote-cli (2026-11-04, build): a desktop printout image with no stored extension takes its filename's extension,
  as behavior 12 says, so PNG bytes can be named .pdf. It stays a known limit, since processing routes images by
  signature. Rejected: sniffing image bytes for the extension in v0.1.0.
- onenote-cli (2026-11-04, build): `mise run check` runs scripts/notices.sh, which fails when
  THIRD-PARTY-NOTICES.md misses a crate either musl target links at runtime per Cargo.lock, or names one neither
  links. The final review had found thiserror's notice missing.
- onenote (round 1, q1→r2 q1): signature routing moves from email into a stdlib-only `route` package used by email, onenote, picture and office. Rejected: exporting from email; copying.
- onenote (r1 q2→r2 q2): picture conversion to PNG (TIFF, EMF, WMF, SVG) moves from office into a `picture` package; image conversion is not office's concern. Rejected: SVG added to office.
- onenote (r1 q3): ink and SVG rasterize at 150 dpi flattened on white. Rejected: 300 dpi.
- onenote (r1 q4, r3 q2): recordings get the named `audio` route with "no processor yet" until the audio task.
- onenote (r1 q5): all of schema "1"'s content is typed. Rejected: top-level fields with raw JSON.
- onenote (r1 q6): onenote v0.1.0's public fixtures are copied into processing under a SHA-256 manifest with a refresh task, and mirrored into ~/docs. Rejected: fetching at test time.
- onenote (r1 q7): no memory bound on the OneNote CLI; #48 a known limit, roadmap note for a tool-wide limit.
- onenote (r1 q8): ink scored by vision against a hand transcription beside OneNote's stored recognition, 3 greedy runs.
- onenote (r1 q10): the root's evidence 17 links the finding in this task.
- onenote (r2 q3): document's corrections and Assess move into `picture`.
- onenote (r2 q4): one internal CLI-contract package for office, email and onenote; exported errors unchanged.
- onenote (r2 q5): the lower-layer cleanup leads this task rather than a task of its own (architect: no context rebuild).
- onenote (r2 q6): ~/docs gains a onenote area holding the public sets and the three large inputs.
- onenote (r3 q1): processing gains a dispatcher that runs each part's preparation stage and recurses; no inference. Rejected: caller-run recipe.
- onenote (r3 q2): `route` names a route for every format in the inventory, "no processor yet" where none exists.
- onenote (r3 q3, q4): no processors pulled forward; text, xlsb, WordPerfect, ODG, tar, gzip, zstd, 7z added to formats.
- onenote (r3 q5, r4 q1, q2): a written adapter convention, plus an exported stage interface and a shared Part type in the dispatcher. Rejected: one interface every format package implements.
- onenote (r3 q6, r6 q1): onenote-fixture was folded in, then dropped: OneNote for the web offers no download or export of a notebook. Backlog note: an authored notebook needs desktop OneNote on Windows or a Mac. No onenote v0.1.1.
- onenote (r4 q6): zip recursion moves from formats into this task.
- onenote (r4 q7): every SVG converts to PNG through picture.
- onenote (r5 q1, q2): no count, byte or depth cap by default; caller-settable. Rejected: 10,000 parts, 1 GiB, depth 16 (data loss on enterprise stores).
- onenote (r5 q3, r6 q2, q5): unsafe means unsafe to read: zip entries escaping their directory, symlink entries, the bomb rule (past both 1,000:1 and 1 GiB) and cycles are quarantined, recorded and recoverable. Active content (macros, ActiveX, OLE embeddings, external relationships, PDF JavaScript and actions, SVG scripts, executables, mismatched or double extensions, polyglots) is flagged and still processed, since no stage executes it. Rejected: quarantining scripted content.
- onenote (r5 q4): each part's content file is removed once consumed unless the caller keeps it.
- onenote (r5 q5, r6 q6–q9): a caller-supplied content-addressed index, keyed by SHA-256 plus stage identity (flags by SHA-256 alone), checked for top-level inputs too; in-memory and file-backed indexes ship; PostgreSQL is the capstone's.
- onenote (r5 q6, r6 q10, q11): mail gains `--after <id>` in v0.3.0, released as a checkpoint before processing's changes; processing resumes by cursor plus restored Threader keys plus the index.
- onenote (r5 q7, r6 q14): the finding measures the dispatcher on mail's scale store and concurrently at 1, 2, 4, 8 workers.
- onenote (r6 q3): OLE2 macro detection deferred to formats. Rejected: oletools (an unmaintained Python dependency is a maintenance liability, unlike a stable Rust crate).
- onenote (r6 q4): ClamAV excluded; roadmap note as a potential future addition.
- onenote (r6 q12, q13): the index claims content in flight so concurrent dispatches never duplicate work; processing offers a pool over N workers.

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
  v0.3.0 (this replaces the bullet's v0.1.0 and v0.2.0 drafts, which were never applied):

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
  >   pages in order. Its `office` package runs the spike's Open XML CLI over docx, xlsx, and
  >   pptx, converts legacy .doc, .xls, and .ppt to OOXML through LibreOffice in a digest-pinned
  >   container, and converts EMF, WMF, and TIFF pictures to PNG; no stage re-encodes a picture
  >   lossily. ffmpeg and ffprobe take audio and video, live sources included.
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
- coordinator · roadmap, the `office` task's summary, rewritten to name office's released contract,
  the picture conversions, and the vision integration tier (office-cli, architect: office itself
  runs no inference):

  ```
  The adapter over the Open XML CLI at office v0.1.0, by its release assets (a tar.gz per
  linux-x64 and linux-arm64, and SHA256SUMS) and its CLI contract (README.md): the JSON model,
  schema "1", on stdout; embedded media through --media, named by SHA-256; exit codes 0
  converted, 1 media not written, 2 usage, 3 refused, 4 failed; warnings on stderr. Legacy .doc,
  .xls, and .ppt converted to OOXML by LibreOffice headless on the container runner, since office
  refuses them. EMF and WMF pictures converted to PNG through LibreOffice and TIFF through
  magick; no stage re-encodes a picture lossily. An integration tier measures the legacy round
  trip's loss and the container's footprint, and sends each distinct picture through vision on
  gb10-2 with its own test-only router client. Tags processing v0.3.0. Evidence 12.
  ```
- coordinator · roadmap, the `office` task (optional): add `proof = "The check passes, processing
  v0.3.0 is released, and docs/findings/office.md records the legacy round trip's loss and
  footprint and the vision runs over office's pictures."`
- coordinator · context/ai-strategy.md, the `spike-ai-pipeline-office` bullet: the sentence
  "Legacy .doc, .xls, and .ppt are converted to OOXML first by LibreOffice in a container" says
  office refuses them and processing's `office` package converts them to OOXML through
  LibreOffice in a container before running office.
- coordinator · context/system-tools.md, "First case" (optional): "LibreOffice in a container for
  legacy office formats" reads "LibreOffice in a container for legacy office formats and EMF and
  WMF pictures".
- coordinator · roadmap, the `office-cli` task (optional): add `proof = "The check passes on
  ubuntu-24.04 and ubuntu-24.04-arm, office v0.1.0 is released with both archives and SHA256SUMS,
  and docs/findings/office-cli.md records the baseline against markitdown and xberg."`
- coordinator · context/ai-strategy.md, the `spike-ai-pipeline-office` bullet (optional), after
  its first sentence: "Its JSON model is a versioned output contract (schema `1` at v0.1.0). Its
  Markdown is the document's text, complete without inference, and its media blocks are the only
  content a higher layer may send to vision; it runs no inference itself."
- roadmap note · spike-ai-pipeline-office (its next task): fixtures/README.md calls
  slide-objects' pictures "beside another", but generate.py places both of slide 3's at one
  default position, so the blue square covers the red; say so, or place them apart. office's
  README should state the limit processing now records: pictures are listed with no layout, so a
  covered, off-slide, or cropped-away picture is still a media block.
- coordinator · roadmap, the `mail-cli` task (optional): add `proof = "The check passes on
  ubuntu-24.04 and ubuntu-24.04-arm, mail v0.1.0 is released with both archives and SHA256SUMS,
  and docs/findings/mail-cli.md records the fields recovered per kind and each store's time and
  peak memory."`
- roadmap note · spike-ai-pipeline-mail (its next task): the static musl release binary takes 2.0
  to 2.3 times a glibc build's time on large stores (13.27 s against 6.51 s on the 2.1 GB scale
  store at v0.1.0, 14.82 s against 6.88 s at v0.2.0) and 1.7 times on the 2.7 GB scale mboxrd
  (6.86 s against 4.09 s). Its system time (3.2 s against 0.7 s on the store at v0.1.0, 2.6 to 2.8 s
  against 1.0 s on the mboxrd) points at musl's allocator (docs/findings/mail-cli.md, "The release
  binary against a glibc build" and ".eml and mbox"); measure a replacement allocator if
  processing's email runs find the time matters.
- coordinator · roadmap, the `email` task (optional): add `proof = "The check passes, processing
  v0.4.0 is released, and docs/findings/email.md records the threads and routes against hand
  truth, each route carried to its real stage, and the adapter's time and peak memory beside
  mail's."`
- coordinator · context/ai-strategy.md, the `spike-ai-pipeline-processing` bullet: the v0.3.0
  draft above gains, before its last sentence, "Its `email` package runs the spike's mail CLI over
  .pst stores and .msg messages, streams their records as typed values, rebuilds threads by RFC
  5256's REFERENCES algorithm with conversation-index parents, and names each attachment's stage
  by its content's signature, for the caller to run."
- coordinator · context/system-tools.md, "First case" (optional): "and the spike's own Open XML
  CLI, built as a system tool" reads "and the spike's own Open XML and mail CLIs, each built as a
  system tool".
- coordinator · roadmap, the `mail-mime` task (optional): add `proof = "The check passes on
  ubuntu-24.04 and ubuntu-24.04-arm, mail v0.2.0 is released with both archives and SHA256SUMS,
  and docs/findings/mail-cli.md records the .eml and mbox fields recovered against hand truth and
  the scale mboxrd's time and peak memory beside the scale store's."`
- coordinator · roadmap, the `mail-mime` task's summary (optional), to what the task built:
  "Nested message/rfc822 parts parsed" reads "Nested message/rfc822 and message/global parts
  parsed, to 16 levels"; "decoded as WHATWG windows-1252" reads "decoded as WHATWG windows-1252,
  except the five bytes it leaves undefined, which become U+FFFD"; and "a generated .eml carrying
  Thread-Index and Thread-Topic" reads "a hand-written .eml carrying Thread-Index, Thread-Topic,
  Sensitivity, and X-Priority".
- coordinator · context/ai-strategy.md, the `spike-ai-pipeline-mail` bullet: "Two limits are
  known: .ost, which outlook-pst doesn't read, is refused, and embedded messages in a .pst are
  listed without content, since outlook-pst 1.2.0 can't read them." reads "Its known limits:
  .ost, which outlook-pst doesn't read, is refused; embedded messages in a .pst are listed without
  content, since outlook-pst 1.2.0 can't read them; and every mbox is read as an mboxrd, so
  mboxcl's Content-Length is ignored and an mboxo body line that was ">From " loses its ">"."
- coordinator · roadmap, experiment.ai.spike-ai-pipeline: add a task after `formats` and before
  the capstone, for a CLI in spike-ai-pipeline that runs the full document-processing suite over
  an input set (such as `~/docs`) and routes each file to its processor. It comes once every
  processor task has shipped, so it has the whole suite to route to (the architect, 2026-11-04,
  during mail-mime). Its name, scope, and whether it scores against truth are settled by
  `marathon plan`.
- coordinator · roadmap, the `email-mime` task: `repos` gains `"spike-ai-pipeline"`, for evidence
  16's sentence. In its summary, "by extension .eml or type message/rfc822 plus mail's header-shape
  check, an exception to the signature rule" reads "by extension .eml or type message/rfc822 or
  message/global plus mail's header-shape check, when its content has no routed signature".
- coordinator · roadmap, the `email-mime` task (optional): add `proof = "The check passes,
  processing v0.5.0 is released, and docs/findings/email.md's addendum records the .eml and mbox
  threads and routes against hand truth and the adapter's time and peak memory on the scale mboxrd
  beside mail's."`
- coordinator · context/ai-strategy.md, the `spike-ai-pipeline-processing` bullet, to v0.5.0: in the
  pending v0.4.0 sentence, "runs the spike's mail CLI over .pst stores and .msg messages" reads "runs
  the spike's mail CLI over .pst stores, .msg messages, .eml messages, and mbox files", and "names each
  attachment's stage by its content's signature, for the caller to run" reads "names each
  attachment's stage by its content's signature, a by-value .eml by its name or stored type and its
  header's shape, for the caller to run".
- coordinator · roadmap, the `gb10-2-align` task (a note for its plan): currency now reports
  llama.cpp b12110 (v0.9.0, 2026-11-04), past the b12063 target. v0.9.0 resizes Gemma 4 images
  without letterbox padding and speeds Gemma 4 E2B/E4B prefill, so the target choice may want a
  fresh look.
- roadmap note · spike-ai-pipeline-mail (its next task): its .claude/marathon.toml still says "No CI runs yet"
  and merges without waiting on checks, though its CI runs; office and onenote chain `gh pr checks --watch`
  before the merge (onenote-cli).
- coordinator · roadmap, experiment.ai.spike-ai-pipeline: add a task `onenote-fixture` after `onenote-cli`, named
  "An authored OneNote notebook". The architect authors a notebook in OneNote (web, mobile, or desktop) and
  downloads it from OneDrive; it is committed with hand truth from onenote-cli's round 2 authoring spec (goal
  record, Decisions, "onenote-cli (round 2, q3)") after a scan for the account email; its package, built through
  cab, reads equal to its directory; the task tags onenote v0.1.x. OneNote for the web couldn't create a notebook
  during onenote-cli (2026-11-04).
- coordinator · roadmap, the `onenote-cli` task's summary: the sentence "Fixtures: onenote.rs's MIT samples, never
  its AGPL ones, and a notebook the architect authors in OneNote, in both variants, with hand truth." reads
  "Fixtures: Interop-TestSuites' OneNote files (MIT) and onenote.rs's maintainer samples (MPL-2.0), never its AGPL
  ones, with hand-written expectations; .onepkg through packages the tests build." The authored notebook moves to
  `onenote-fixture`.
- roadmap note · spike-ai-pipeline-onenote: OneNote section decryption waits for a specification or an open
  implementation, since [MS-ONESTORE] 2.5.19 leaves the scheme undocumented (onenote-cli, round 2 q7).
- coordinator · roadmap, the `onenote` task: onenote_parser issue #48 (a 375 MB OneDrive-format section needs over
  9 GB) bounds what the adapter can read. onenote's finding (spike-ai-pipeline-onenote's
  docs/findings/onenote-cli.md) measured the 161 MB Large OneDrive.one at 135 MiB, so #48 didn't show at that
  size, and advises the adapter to bound the CLI's memory all the same.
- coordinator · context/ai-strategy.md, the `spike-ai-pipeline-onenote` bullet, to v0.1.0: "Processing routes the
  pages' media to their stages (evidence 17)." reads "Processing's `onenote` package is to route the pages' media
  to their stages.", and the bullet ends with: "Its known limits: password-protected sections are refused, since
  [MS-ONESTORE] leaves their encryption undocumented; no OneNote-written .onepkg is tested; and onenote_parser
  issue #48 can make a large OneDrive section with a long edit history need gigabytes of memory. Its
  [onenote-cli finding](https://github.com/JaimeStill/spike-ai-pipeline-onenote/blob/main/docs/findings/onenote-cli.md)
  records every expected field recovered against the public fixtures' hand-written expectations, and its time
  and peak memory per input (evidence 17)."
- coordinator · roadmap, the `onenote-cli` task (optional): add `proof = "The check passes on ubuntu-24.04 and
  ubuntu-24.04-arm, onenote v0.1.0 is released with both archives and SHA256SUMS, and
  docs/findings/onenote-cli.md records the fields recovered per kind against the public fixtures' expectations
  and each input's time and peak memory."`
