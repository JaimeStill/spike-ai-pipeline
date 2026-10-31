# spike-ai-pipeline

A spike for the Standards Lab reference architecture. It asks whether a Go model client and a
message, document, and stream processing layer, built under go-elemental, can carry a pipeline
that a browser follows live: transactional and evented, capturing metadata, extracting data by
inference, and storing embeddings in PostgreSQL with pgvector for semantic search, each stage
emitting the events the browser follows.

Its answer is evidence for standards-lab's experiment.ai intake; the spike decides nothing
itself. It informs go-ai's model-client surface and its wire-format layer, where the processing
layer lives, `v1.ai.local-models`, and v1.ai's library and service tasks.

## Repositories

The layers live in repositories of their own, checked out beside this one in `~/experiments/`,
each released by tag:

- [spike-ai-pipeline-client](https://github.com/JaimeStill/spike-ai-pipeline-client): the model
  client.
- [spike-ai-pipeline-processing](https://github.com/JaimeStill/spike-ai-pipeline-processing): the
  processing layer and its system-tool adapters. It prepares content for a model before inference,
  such as a PDF's pages as corrected images, regions, and text layers; a layer above runs the
  inference.
- [spike-ai-pipeline-office](https://github.com/JaimeStill/spike-ai-pipeline-office): the Open XML
  CLI, a NativeAOT binary that turns docx, xlsx, and pptx files into a JSON document model and
  Markdown, with chart data and embedded media in place of page images. It is built for
  processing to run as a system tool; processing's adapter is planned. Its README gives the CLI
  contract.

This repository holds the spike's record, gb10-2's host scripts, the consumer check that builds
on the layers by tag, and, once its gates clear, the capstone: a backend on the layers and a web
UI on experiment.html's stack.

## gb10-2

gb10-2 serves set A from llama.cpp's router container, run from this repository's clone there,
`~/repos/spike-ai-pipeline`, by spike-model-engines' scripts copied into `host/` (`host/README.md`).
Each session starts with `mise run host-update`, on the laptop, which drives gb10-2 over Tailscale
SSH (`scripts/host-update.sh` says each step). It stops before touching the router when gb10-2 needs
a root step, an apt upgrade, a fresh CDI spec, or a reboot, and prints the exact commands for the
architect; otherwise it pulls the clone, moves the router to the pin, reloads a changed preset,
loads set A, and prints the build, gb10-2's stack, and the laptop's network path. A second run
changes nothing and says so. A stopped llama container fails the run, which names the command that
starts it. A running one whose preset mount is sourced anywhere but `host/llama-preset/`, such as
`engines/llama-preset/` from before the scripts moved to `host/`, is re-created on the same image
with the mount moved, by `host/engines llama update`. A running one that can't see the clone's
preset, because a delivery recreated `host/llama-preset/` under its bind mount, is restarted.
`HOST_UPDATE_HOST` names another host.

host-update fast-forwards the clone's branch from its upstream. A task branch not yet merged has
no upstream, so a session delivers it from the laptop by push before running host-update:

```sh
git push ssh://gb10-2/~/repos/spike-ai-pipeline <branch>:refs/heads/<branch>
```

host-update sets the clone's `receive.denyCurrentBranch` to `updateInstead`, so the push updates
the checked-out branch and its files. Once the task merges, switch the clone to `main`, which
tracks `origin/main`, and host-update pulls it from then on:
`ssh gb10-2 git -C repos/spike-ai-pipeline switch main`.

`mise run integration` probes the router from the laptop (`probe/`). It checks every set A
capability, and checks the build, sampling, and slots against the pin and the preset.
[The host finding](docs/findings/host.md) records its runs, including whether the router answers
`/v1/responses` and whether transcription returns timestamps, and what the router's
`/v1/messages` and `/v1/responses` drop, from the client's integration runs.

## The question

See `context/README.md`, "The question", and "The evidence".

## The answer

Not yet.
