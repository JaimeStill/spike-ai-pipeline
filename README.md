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
  such as a PDF's pages as corrected images, regions, and text layers, an office document's
  model or Markdown with its pictures, a mail store's records, threads, and attachment routes, a
  OneNote page's Markdown and media, or a recording cut at its silences into windows a speech
  model takes whole. Its `media` package reads a live or recorded source's audio and video
  through one ffmpeg run and streams its windows and frames as they arrive. Its `dispatch` package
  walks an input and every part found in it, attachments, media, and ZIP entries, to those stages,
  a video among them, inspecting each part first and preparing each distinct content once. A layer
  above runs the inference.
- [spike-ai-pipeline-office](https://github.com/JaimeStill/spike-ai-pipeline-office): the Open XML
  CLI, a NativeAOT binary that turns docx, xlsx, and pptx files into a JSON document model and
  Markdown, with chart data and embedded media in place of page images. It is built for
  processing to run as a system tool, which processing's `office` package does. Its README gives
  the CLI contract.
- [spike-ai-pipeline-mail](https://github.com/JaimeStill/spike-ai-pipeline-mail): the mail CLI, a
  static Rust binary that reads Outlook .pst stores, through Microsoft's outlook-pst, .msg
  messages, through its own MS-OXMSG reader, and .eml messages and mbox files, through Stalwart's
  mail-parser, into JSON Lines of folders, messages with their threading fields, and attachments.
  Outlook .ost files are refused. Processing's `email` package runs it as a system tool, builds
  the threads, and names each attachment's stage.
- [spike-ai-pipeline-onenote](https://github.com/JaimeStill/spike-ai-pipeline-onenote): the
  OneNote CLI, a static Rust binary that reads OneNote .one sections, .onepkg exports, and
  notebook directories, through onenote_parser, into JSON Lines of sections and pages with their
  Markdown, images, embedded files, and ink. Processing's `onenote` package runs it as a system
  tool and routes each page's media.

This repository holds the spike's record, gb10-2's host scripts, the consumer check that builds
on the layers by tag, and, once its gates clear, the capstone: a backend on the layers and a web
UI on experiment.html's stack.

## docproc

docproc runs processing's whole suite over an input set: every file goes through processing's
dispatcher and its ten routes' stages, every part found inside one (an attachment, a ZIP entry, a
OneNote page's media) with it, and what each stage prepares is stored in a run directory. It runs
no inference; docinfer, planned, reads the run directory and does.

### Building

```sh
go build ./cmd/docproc   # or: go install ./cmd/docproc
```

Run it under mise (`mise exec -- docproc …`, or with mise activated in the shell): the root's
`mise.toml` pins the spike's own CLIs, office 0.1.0, mail 0.3.1, and onenote 0.1.0, and 7-Zip's
7zz 26.06, each from its GitHub release with its digest checked, and docproc finds every tool on
`PATH`.

### Host prerequisites

The rest come from the host. At start, docproc finds each tool, reads its version, and checks it
against processing's minimum:

| Tool | Minimum | Source | Stages that need it |
|---|---|---|---|
| `gs` (Ghostscript) | 10.0 | host | document |
| `qpdf` | 11.0 | host | document |
| `pdfdetach` (Poppler) | 22.02 | host | document |
| `magick` (ImageMagick) | 7.1 | host | office, vision-converted |
| `soffice` (LibreOffice) | 24.2 | processing's pinned image, run through `docker` | office, vision-converted |
| `spike-ai-pipeline-office` | 0.1.0 | mise pin | office |
| `spike-ai-pipeline-mail` | 0.2.0 | mise pin | nested (email) |
| `spike-ai-pipeline-onenote` | 0.1.0 | mise pin | onenote |
| `7zz` | 24.01 | mise pin | archive |
| `ffmpeg`, `ffprobe` | 9.0 | host | audio, video |

The vision and text stages need no tool. A tool that is missing or too old leaves out every stage
that needs it: docproc still runs, lists the stage as missing with the reason, and that route's
parts come to `no processor yet`.

### Running

```sh
docproc run <path>... --out <dir> [--workers N] [--exclude <route>]...
docproc report <run-dir> [--run <id>]
```

`docproc run` takes files, and directories walked for their regular files; no symbolic link is
followed, and a file reached twice is one input. `--out` (required) is the run directory, made when
it doesn't exist and reused when it does. `--workers` is how many inputs are dispatched at once,
this machine's CPU count by default, and from 1 to that count. `--exclude`, repeatable, leaves a
route's stage out (`document`, `office`, `vision-converted`, `vision`, `nested`, `onenote`,
`archive`, `audio`, `video`, `text`); its parts come to `no processor yet`.

`docproc report` renders a run's HTML report again from its manifest and record, the latest run's
by default, and prints its path. `docproc run` writes the same page as each run ends.

Exit codes: 0 when the run completes, even with failed parts or inputs; 1 when an error stops the
whole dispatch (the context ending, or the run directory or terminal refusing a write), once the
manifest names it, or when a path can't be read, or, for `report`, when the run directory has no
such run; 2 on a usage error, such as no path, no `--out`, `--workers` outside 1 to the CPU count,
or an unknown `--exclude` route, before any tool is looked up.

### Terminal output

docproc first prints two tables: each tool with its version, its minimum, and where it was found,
and each route with its stage, its state (`wired`, `missing`, or `excluded`), and its tools'
versions or its reason. Then, as each input's last event arrives, one line:

```
<path>  <route>  <outcome>[: <error>]  outputs <n>  children <n>  <time>
```

An input whose dispatch an error stopped prints `<path>  -  error: <error>  <time>`. A closing
summary counts the inputs by route and outcome, in the stage table's route order, then names the
run's id, its record, and its report.

### The run directory

```
index/                               processing's file index, shared by every run
outputs/<sha256>/<stage>/<n><ext>    each output, stored once by content and stage identity
quarantine/<sha256>/{content,flags.json}
runs/<run id>/manifest.json          each run's manifest
runs/<run id>/record.jsonl           its record, one event per line
runs/<run id>/report.html            its report
```

A run over an existing run directory reuses its index and output store under a new run id: content
an earlier run prepared is seen, not prepared again, and its outputs' refs name the copies already
stored. A seen part's subtree, which processing doesn't dispatch again, is copied into the new
record from the earliest record that holds it, each line marked seen, so every run's record is
whole on its own. Content that failed, or holds a part that failed or was limited, is prepared
again, and a part with no stage comes to `no processor yet` again. The directory is schema `"1"`, the contract docinfer reads: every field of the manifest and
the record, the path rules, and the store's naming are in
[docs/docproc-schema.md](docs/docproc-schema.md).

[The docproc finding](docs/findings/docproc.md) records docproc over ~/docs and its re-run.

## gb10-2

gb10-2 serves set A from llama.cpp's router container, run from this repository's clone there,
`~/repos/spike-ai-pipeline`, by spike-model-engines' scripts copied into `host/` (`host/README.md`).
Each session starts with `mise run host-update`, on the laptop, which drives gb10-2 over Tailscale
SSH (`scripts/host-update.sh` says each step). It stops before touching the router when gb10-2 needs
a root step, an apt upgrade, a fresh CDI spec, or a reboot, and prints the exact commands for the
architect; otherwise it pulls the clone, moves the router to gb10's pin, reloads a changed preset,
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
capability, and checks the build, sampling, slots, and prompt cache against the pin and the
preset.
[The host finding](docs/findings/host.md) records its runs, including whether the router answers
`/v1/responses` and whether transcription returns timestamps, and what the router's
`/v1/messages` and `/v1/responses` drop, from the client's integration runs.

## The question

See `context/README.md`, "The question", and "The evidence".

## The answer

Not yet.
