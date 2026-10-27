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
2. A second wire format, Anthropic Messages, fits under the same request and response model with
   no change to callers.
3. The client's base module is standard library only, and the check passes.
4. A PDF's pages go through vision as a whole blob and as a page stream, with the renderer
   bake-off's ranking, time and memory per page, license, and footprint.
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
    the fixed documents.
12. Office formats: the Open XML CLI's structure fidelity, memory, cold start, and binary size on
    fixed documents, spreadsheets, and decks, against markitdown and xberg; chart data and
    embedded media in place of page images; and .doc, .xls, and .ppt converted through
    LibreOffice in a container, with what the round trip loses and the container's footprint.
13. Live media: a live or real-time-paced audio and video source processed as it arrives, with
    per-window latency, as an event stream with cancellation.
14. One tool run through each runner, exec and container, under the same tool contract.
15. Capstone: the browser's consumption of the event sequence, NATS over WebSocket against a
    backend relay, compared on latency, ordering, auth, ports, and native-tier use.

## Gates

The host, client, and processing tasks run now. The capstone waits for three things: those tasks,
spike-model-gateway's answer, and experiment.html's intake. The spike returns to `planned` in
between.

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
