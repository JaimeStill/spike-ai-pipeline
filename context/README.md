# spike-ai-pipeline

The root of spike-ai-pipeline. The spike is the sub-goal `experiment.ai.spike-ai-pipeline` in the
coordinator's roadmap, `standards-lab/context/roadmap.toml`, where its tasks live; its goal record
is `context/goals/experiment.ai.spike-ai-pipeline.md` here. Its question is framed in
`standards-lab/context/ai-strategy.md`, "Experiment: spike-ai-pipeline", and the system-tools
principle its processing layer follows is `standards-lab/context/system-tools.md`.

## The question

Can a Go model client and a message, document, and stream processing layer, built under
go-elemental, carry a transactional, evented pipeline (metadata capture, inference extraction,
and embeddings stored in PostgreSQL with pgvector for semantic search) that a browser follows
stage by stage in real time?

## The evidence

1. Client: chat with tools and a reasoning round trip, streamed and unstreamed; vision, audio
   input, embeddings, and transcription; each proven against set A on gb10-2.
2. Two more wire formats, Anthropic Messages and Open Responses (the model's own baseline), fit
   under the same request and response model as Chat Completions, with no change to callers.
3. The client's base module is standard library only, and the check passes.
4. A PDF's pages go through vision as a whole blob and as a page stream, with the renderer
   bake-off's ranking, time and memory per page, license, and footprint. Processing's
   [documents finding](https://github.com/JaimeStill/spike-ai-pipeline-processing/blob/main/docs/findings/documents.md)
   records it.
5. Audio longer than the model's limit is cut at silences (ffmpeg silencedetect) into windows
   under 30 s that overlap by 1 to 2 s, transcribed concurrently within set A's audio slots, and
   joined by aligning the overlapping words at each seam; an optional gpt-oss pass smooths the
   seams. Seam accuracy and total time, against one unsplit pass, are recorded.
6. A chat stream passes through processing as events, with cancellation.
7. Capstone, after its gates: metadata capture, inference extraction, and embeddings in pgvector
   with a semantic search, each stage transactional. The event layer tracks every stage event in
   sequence, in real time, and the UI shows each stage as it runs.
8. After the gates: the client runs unchanged through spike-model-gateway's gateway.
9. Where the processing layer should live, and which way it depends on the client.
10. pgvector as a declared native-tier extension: its install, migration, image pin, and the
    packages allowed to use it.
11. Image correction: vision extraction accuracy with and without each ImageMagick correction, on
    the fixed documents. Processing's
    [documents finding](https://github.com/JaimeStill/spike-ai-pipeline-processing/blob/main/docs/findings/documents.md)
    records it.
12. Office formats: the Open XML CLI's structure fidelity, memory, cold start, and binary size on
    fixed documents, spreadsheets, and decks, against markitdown and xberg; chart data and
    embedded media in place of page images; and .doc, .xls, and .ppt converted through
    LibreOffice in a container, with what the round trip loses and the container's footprint.
    Office's
    [office-cli finding](https://github.com/JaimeStill/spike-ai-pipeline-office/blob/main/docs/findings/office-cli.md)
    records the baseline, and processing's
    [office finding](https://github.com/JaimeStill/spike-ai-pipeline-processing/blob/main/docs/findings/office.md)
    records the legacy conversion's loss and footprint and the vision runs over the pictures.
13. Live media: a live or real-time-paced audio and video source processed as it arrives, with
    per-window latency, as an event stream with cancellation.
14. One tool run through each runner, exec and container, under the same tool contract.
15. Capstone: the browser's consumption of the event sequence, NATS over WebSocket against a
    backend relay, compared on latency, ordering, auth, ports, and native-tier use.
16. Mail stores and messages: Outlook .pst stores, .msg messages, .eml messages, and mbox files
    read by the spike's mail CLI, .pst through Microsoft's outlook-pst, .msg through the CLI's own
    MS-OXMSG reader over cfb, and .eml and mbox through Stalwart's mail-parser, into
    folders, messages (headers, bodies, and threading fields), and attachments, each attachment
    routed to the stage its format takes; with the fields recovered per kind against hand truth
    from generated and public fixtures, the threads rebuilt, and each store's time. Mail's
    [mail-cli finding](https://github.com/JaimeStill/spike-ai-pipeline-mail/blob/main/docs/findings/mail-cli.md)
    records the fields recovered per kind and each store's time and peak memory, and processing's
    [email finding](https://github.com/JaimeStill/spike-ai-pipeline-processing/blob/main/docs/findings/email.md)
    records the threads rebuilt across folders, each attachment's route proven against its real
    stage, and the adapter's time and peak memory. The finding's addendum, for mail 0.2.0, scores
    the threads and routes of .eml messages and mbox files against hand truth, a by-value .eml
    routed as a nested message, and times the adapter on the scale mboxrd beside mail.
17. OneNote notebooks: .one sections, from desktop OneNote 2016 and later and from OneDrive,
    .onepkg exports, and notebook directories through their .onetoc2, read by the spike's OneNote
    CLI through onenote_parser into notebooks, sections, and pages at their latest revision (rich
    text, lists, tables, math, note tags, and a Markdown body), with images, embedded files, and
    ink (its recognized text and its strokes as SVG) routed to the stages their formats take;
    with the fields recovered against hand truth from public fixtures, and each input's time.
    OneNote's
    [onenote-cli finding](https://github.com/JaimeStill/spike-ai-pipeline-onenote/blob/main/docs/findings/onenote-cli.md)
    records every expected field recovered against the public fixtures' hand-written
    expectations: 236 of 236 over the committed sections, 138 of 138 over the committed
    notebooks, and 129 of 129 content fields. Every committed input reads in under 10 ms and
    4.5 MiB, and the large sections in 1.30 s at 19.5 MiB (11 MB), 0.25 s at 135 MiB (113 MB,
    desktop format), and 0.24 s at 135 MiB (161 MB, OneDrive format), so onenote_parser issue #48
    doesn't show at 161 MB. Processing's
    [onenote finding](https://github.com/JaimeStill/spike-ai-pipeline-processing/blob/main/docs/findings/onenote.md)
    records every OneNote medium routed to its real stage against hand route truth, inside 225
    parts from 119 fixture inputs dispatched with no mismatch. It also records page ink rendered
    to PNG and read by vision at a character error rate of 0.028, against 0.68 for OneNote's own
    stored recognition, and the adapter within 6% of the CLI's time on the large sections.

## Gates

The host, client, and processing tasks run now. The capstone waits for those tasks
(gb10-2-align, audio, streams, and formats remain) and for two answers: experiment.hosting's
spike-model-layout and spike-model-gateway. spike-model-engines and experiment.html's
spike-html-apps have answered, and neither experiment's intake is a gate. gb10-2-align itself
waits on spike-hosting-upgrades' verify-gb10. The spike returns to `planned` in between.

## Capability map

- **host**: gb10-2's llama.cpp router serving set A. The router runs from spike-model-engines'
  llama.cpp container scripts, copied from its commit d285b06 into `host/` beside the set A
  preset. `mise.host.toml` pins `hf`, which stages the models on gb10-2. `mise run host-update`
  brings gb10-2 current and loads set A at each session's start. It restarts a running router
  whose preset mount is dead, and re-creates one whose preset mount is sourced anywhere but
  `host/llama-preset/`, as after the scripts moved from `engines/`, with `host/engines llama
  update`. `mise run integration` probes every capability (`probe/`), and
  `docs/findings/host.md` records the probe runs.
- **consumers**: the check that builds on the layers at their released tags.
- **capstone**: the backend and web UI, once the gates clear.
