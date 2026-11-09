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
on the layers by tag, and two CLIs on the layers: docproc, which prepares an input set, and
docinfer, which runs inference over what docproc prepared. The capstone, a backend on the layers
and a web UI on experiment.html's stack, is capstone.data-intelligence, in its own repository,
`JaimeStill/data-intelligence` (created when the goal is staged); docinfer's
directory is its input.

## docproc

docproc runs processing's whole suite over an input set: every file goes through processing's
dispatcher and its ten routes' stages, every part found inside one (an attachment, a ZIP entry, a
OneNote page's media) with it, and what each stage prepares is stored in a run directory. It runs
no inference; [docinfer](#docinfer) reads the run directory and does.

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

docproc exits with one of three codes:

- **0** when the run completes, even with failed parts or inputs.
- **1** when an error stops the whole dispatch (the context ending, or the run directory or
  terminal refusing a write), once the manifest names it; when a path can't be read; or, for
  `report`, when the run directory has no such run.
- **2** on a usage error, before any tool is looked up: no path, no `--out`, `--workers` outside 1
  to the CPU count, an unknown `--exclude` route, or a `report --run` that is a path rather than a
  run id.

### Terminal output

docproc first prints two tables: each tool with its version, its minimum, and where it was found,
and each route with its stage, its state (`wired`, `missing`, or `excluded`), and its tools'
versions or its reason. A record in the run directory that can't be read prints
`warning: a record replays nothing: <error>` and is skipped. Then, as each input's last event
arrives, docproc prints one line for it:

```
<path>  <route>  <outcome>[: <error>]  outputs <n>  children <n>[  warnings <n>]  <time>
```

`warnings <n>` counts the seen parts in the input whose children no record in the run directory
holds, so docproc copied none of their lines
([replayed subtrees](docs/docproc-schema.md#seen-content-and-replayed-subtrees)). An input whose
dispatch an error stopped prints `<path>  -  error: <error>  <time>`.

The terminal prints an error as one line: the error's first line, trimmed, with each character
that isn't printable replaced by U+FFFD, and cut at 400 characters (runes), ending in `…` when
anything is cut. The record's `error` field holds the whole error.

A closing summary lists the inputs by route and outcome, in the stage table's route order, and
their total. When any seen part had children no record holds, a `warnings <n>` line counts them.
The last lines name the run's id and record, and its report once written.

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
again, and a part with no stage comes to `no processor yet` again. The directory is schema `"1"`,
the contract docinfer reads: every field of the manifest and the record, the path rules, and the
store's naming are in [docs/docproc-schema.md](docs/docproc-schema.md).

[The docproc finding](docs/findings/docproc.md) records docproc over ~/docs and its re-run.

### Updating the golden files

Two tests compare docproc's output with a golden file, and `-update` makes each rewrite its file
instead; no other package takes the flag. The `internal/docproc/report` package's test renders the
`internal/docproc/app` package's record golden, so a regenerated record golden needs the report
golden regenerated after it:

```sh
go test ./internal/docproc/app -update      # internal/docproc/app/testdata/record.golden.jsonl
go test ./internal/docproc/report -update   # internal/docproc/report/testdata/report.golden.html
```

`git diff` then shows what changed.

## docinfer

docinfer runs inference over one docproc run: every image output goes to a vision model, every
audio part's or video's windows go to a speech model and are joined into one transcript, and every
text, docproc's text and Markdown outputs, each vision text, each transcript, and one metadata text
per input, is cut into chunks and embedded. It reads the docproc run directory and never writes
to it; what it makes goes into a directory of its own, whose chunk records and vectors are the
capstone's input. It loads no database and normalizes no vector.

### Building

```sh
go build ./cmd/docinfer   # or: go install ./cmd/docinfer
```

docinfer needs no system tool: only HTTP access to the endpoints its configuration names.

### Configuration

`--config` names a directory, read through go-core's layered JSON load: `config.json`, its overlay
`config.<env>.json`, `secrets.json`, and its overlay `secrets.<env>.json`, later files winning,
`DOCINFER_ENV` naming `<env>`; then the `DOCINFER_` overrides. Each file is optional, but a key the
configuration doesn't declare stops the load.

```json
{
  "endpoints": {
    "gb10-2": {"base_url": "http://gb10-2:8080/v1", "format": "completions",
               "query": {"autoload": "false"}}
  },
  "embeddings":    {"endpoint": "gb10-2", "model": "...", "batch": 16, "chunk_size": 2000, "concurrency": 4},
  "vision":        {"endpoint": "gb10-2", "model": "...", "prompt": "...", "max_tokens": 2048, "reasoning": "none", "concurrency": 4},
  "transcription": {"endpoint": "gb10-2", "model": "...", "prompt": "", "language": "", "concurrency": 4}
}
```

Each endpoint is a model server's API root under a name of the caller's choosing:

- `base_url`: the API root, an absolute http or https URL with no query or fragment.
- `format`: the wire format a vision request is sent in, `completions` (Chat Completions),
  `messages` (Anthropic Messages), or `responses` (Open Responses). Embeddings go to the OpenAI
  embeddings route, and transcription to the OpenAI-compatible `/audio/transcriptions` route,
  whatever the format.
- `token`: the bearer token each request carries, empty for none. It belongs in `secrets.json`,
  which `.gitignore` keeps out of `config/*/`, as it does each `secrets.<env>.json`.
- `query`: parameters added to every request's URL, optional, such as llama.cpp's router's
  `autoload=false`, which keeps the router from loading a model a request names.

There is no default endpoint. Each capability names an endpoint and a model there, and
`concurrency`, the requests it keeps in flight, 4 by default:

| Capability | Required | Settings and defaults |
|---|---|---|
| `embeddings` | yes | `batch`, the chunks one request carries, 16; `chunk_size`, the most characters (runes) in a chunk, 2,000 |
| `vision` | no | `prompt`, what is asked of each image, by default processing's dispatch tier's, "Transcribe the text in this picture. If it has no text, describe it in one sentence."; `max_tokens`, the reply's cap, reasoning included, 2048; `reasoning`, how much the model reasons before it answers, one of the client's efforts, `none`, `low`, `medium`, `high`, or `xhigh`, by default `none`, as processing's dispatch tier asks |
| `transcription` | no | `prompt` and `language`, empty by default, which sends neither |

`config/example` sets a transcription prompt: "Transcribe the speech in this audio verbatim, in
its own language. Write only the words spoken. If there is no speech, write nothing." llama.cpp
sends it in place of its own instruction. Without it, gemma-4-E4B on gb10-2 answers a window of
music or noise with a remark, such as "The audio consists only of instrumental music and does not
contain discernible speech.", which would be embedded as transcript; with it, such a window comes
back empty, and speech, English, Hindi, or Japanese, transcribes as before.

A capability the configuration leaves unset is skipped: its outputs are counted, not sent.

Each setting has an override, `DOCINFER_` and its path upper-cased, each run of characters
outside `A-Z` and `0-9` made one `_`: `DOCINFER_EMBEDDINGS_MODEL`, `DOCINFER_VISION_MAX_TOKENS`,
`DOCINFER_TRANSCRIPTION_LANGUAGE`, and, for the endpoint `gb10-2`,
`DOCINFER_ENDPOINTS_GB10_2_BASE_URL`, `_FORMAT`, and `_TOKEN`. An override of a capability the
files leave unset sets it; an endpoint is overridden only when the files name it, and its `query`
has no override. `DOCINFER_SHUTDOWN_TIMEOUT` is the lifecycle's shutdown budget.

Every configuration error stops docinfer before any request: no embeddings capability, a
capability with no endpoint or model or naming an endpoint the configuration doesn't hold, a format
that isn't one of the three, a base URL that isn't an absolute http or https URL without a query or
fragment, a count below 1, an unknown key, or an override that doesn't parse.

### The set A example

[config/example](config/example) serves set A on gb10-2: one endpoint, `gb10-2`, at
`http://gb10-2:8080/v1` in the `completions` format with `autoload=false`, granite-embedding
English R2 (BF16) for embeddings, Gemma 4 26B-A4B (Q4_0) for vision, and Gemma 4 E4B (Q8_0) for
transcription, each at its defaults. `secrets.json.example` shows the secrets file's shape, its
token empty. `mise run host-update` left set A loaded while gb10-2 served the spike
("gb10-2").

The configuration isn't tied to gb10-2. To point docinfer at another workstation's server, or at
one per capability, copy the directory and change the endpoints and models: any server that lists
its models at `GET <base_url>/models` and answers the capabilities docinfer uses will do. Drop
`query` for a server that isn't llama.cpp's router, set `format` to what it speaks, add an endpoint
per server, and name each capability's model as that server lists it. For a one-off change,
the overrides reach the same settings without a copy:

```sh
DOCINFER_ENDPOINTS_GB10_2_BASE_URL=http://workstation:8080/v1 \
DOCINFER_EMBEDDINGS_MODEL=<model> \
  docinfer run <docproc-dir> --config config/example --out <dir>
```

### The models check

As it starts, after reading its configuration and the docproc run, docinfer lists the models of
each endpoint a capability uses, once each, by `GET <base_url>/models` with the endpoint's query
and token, and finds each capability's model in the OpenAI listing, `{"data": [{"id": ...}]}`,
reading the `"status": {"value": ...}` llama.cpp's router adds. It prints a table of each
capability's endpoint, model, and state:

| State | Meaning | docinfer |
|---|---|---|
| `loaded` | Listed with the status `loaded`. | runs |
| `listed` | Listed with no status, as a server that isn't llama.cpp's router lists it. | runs |
| another status, such as `unloaded` | Listed with that status. | warns and runs |
| `unchecked` | The endpoint gives no listing: a reply that isn't 2xx, or holds no `data` list. | warns and runs |
| `unlisted` | The listing doesn't name the model. | exits 1 |
| `unreachable` | The listing's request couldn't reach the endpoint. | exits 1 |
| `skipped` | The configuration leaves the capability unset. | skips it |

### Running

```sh
docinfer run <docproc-dir> --config <dir> --out <dir> [--run <id>]
docinfer report <dir> [--run <id>]
docinfer version
```

`docinfer run` reads the docproc run directory's latest run, or the one `--run` names, and refuses
a manifest whose schema isn't `"1"` before reading its record. `--config` (required) is the
configuration's directory, and `--out` (required) docinfer's directory, made when it doesn't exist
and reused when it does.

It reads every output line, whatever its part came to, a failed part's outputs and a seen part's
among them, and sends each distinct content once. The capabilities run one after another over the
whole run:

1. **Vision**: each distinct image, by its bytes, goes as a data URL beside the prompt, its reply
   capped at `max_tokens`, at the `reasoning` effort, in the endpoint's format: Chat Completions
   sends it as `reasoning_effort`, Open Responses as `reasoning.effort`, and Messages as `thinking`,
   `disabled` for `none` and `adaptive` with `output_config.effort` for the rest. A reply cut at the
   cap is recorded incomplete, its text kept and embedded; one cut before it gave any text is
   counted empty too. Gemma 4 26B-A4B on gb10-2 leaves many dense pages empty when it reasons, since
   its reasoning spends the whole cap ([the finding](docs/findings/docinfer.md)); at `none` it
   answers at once. llama.cpp b12098 honors `none` in Chat Completions and Open Responses, but its
   Messages route ignores `thinking` `disabled` (and `reasoning_effort`), so the model reasons
   anyway: on gb10-2, use the `completions` or `responses` format for vision.
2. **Transcription**: each distinct audio window goes to the transcription route. Each part's
   windows, by stage, are then joined in plan order, by the index each window's meta gives, by
   processing's `audio.Join`; a refused window, or one the plan counts but the record lacks, joins
   as an empty transcript. The manifest records each seam, matched or not.
3. **Embeddings**: each text is cut on its own, at Markdown headings, then blank lines, then
   sentences and line breaks, then a hard cut, to at most `chunk_size` characters, with no overlap
   and no cut inside a UTF-8 character. Each distinct chunk is embedded once, `batch` a request; a
   batch the endpoint rejects is sent again a chunk a request.

An image or window the endpoint refuses, and a chunk it rejects, is recorded, not fatal: the run
goes on without it.

`docinfer report` renders a run's HTML report again from its manifest, chunk record, and stored
texts, the latest run's by default, and prints its path relative to the directory. `docinfer run`
writes the same page as each run ends. The page shows each input's texts: an image inline beside
its vision text, a transcript with its windows, each with its chunk count, and every failure. It
links to docproc's outputs by the relative path from the report to the docproc run directory, so
the two directories can be moved together.

docinfer exits with one of three codes:

- **0** when the run completes, even with refused images or windows or rejected chunks.
- **1** on a configuration error; a docproc run that can't be read, has no run, or has a schema
  other than `"1"`; a model the listing doesn't name or an endpoint it can't reach, after the
  table; or an error that stops the inference (a transport error, a reply that can't be decoded,
  a file that can't be read or written, or the context ending), recorded in the run's manifest. For
  `report`, when the directory or run doesn't exist.
- **2** on a usage error, before anything is read: no docproc directory or more than one, no or an
  empty `--config` or `--out`, or, for either command, a `--run` that is a path or a dot entry
  rather than a run id.

### Terminal output

docinfer first prints the models table, `CAPABILITY ENDPOINT MODEL STATE`, and its warnings, then
the docproc run it read, `docproc run <id>: outputs <n>: <counts by kind>`. As each capability's
phase begins, vision, then transcription, then embeddings, it prints how many distinct images,
windows, or chunks the phase has, and how many it sends and how many earlier runs hold:

```
<capability>: distinct <images|windows|chunks> <n>: to send <n>, held <n>
<capability>: distinct <images|windows|chunks> <n>: held <n>, nothing to send
```

A phase with nothing at all prints no such line, and an unconfigured one none either. Then, as
each request completes, one line for it:

```
vision <done>/<to send>  <path>  <output ref>  ok|incomplete|empty|refused  <time>
transcription <done>/<to send>  <path>  <output ref>  ok|failed  <time>
embeddings <done>/<to send>  <path>[  inputs <n>]  chunks <n>  ok|rejected  <time>
```

`<done>` counts the images, windows, or chunks answered so far, rising to the phase's `to send`;
an embeddings request counts each chunk it carried. Vision and transcription name the input and
output ref the image or window was first seen under; embeddings names the first input waiting for
the request's chunks and, when more than one waits, how many. `empty` is an incomplete reply with
no text. `<time>` is the request's own, from its sending to its reply. A held result sends no
request and prints no line, so a re-run prints only the phase lines.

As each input's work ends, it prints one line for it, after the line of the request that ended
it:

```
<path>  images <n>  windows <n>  chunks <n>  failures <n>  <time>
```

`images` and `windows` count the input's image and audio window output lines, `chunks` its chunk
lines, and `failures` its output lines whose image or window was refused and its chunk lines whose
chunk was rejected. The capabilities run one after another, so an input's work ends with its last
chunk's embedding, and `<time>` is how long after the run began that was: an input with nothing to
send ends as the embeddings begin.

Then a `warning:` line for each refused image, incomplete vision text, an empty one saying so,
refused window, and rejected chunk; a line of counts for each capability, `not configured` for
one skipped; a closing summary of each capability's requests and failures, `CAPABILITY REQUESTS FAILURES`, a skipped one
saying why; and, last, the run's id, its chunk lines, the chunk record's path, and the report's,
each relative to `--out`, as docproc prints its record's.

### docinfer's directory

```
results/<capability>/<xx>/<key>.json   each held result, keyed by content, model, and settings
texts/<xx>/<sha256>.txt                each text a model made, and each joined transcript
runs/<run id>/manifest.json            each run's manifest
runs/<run id>/chunks.jsonl             its chunk record, a line per chunk per occurrence
runs/<run id>/report.html              its report
```

A chunk line holds the chunk's text, its byte span in its source text and the source's kind
(`output`, `metadata`, `vision`, or `transcript`), the input's file metadata and content type, the
part and docproc output it belongs to, and its vector, base64 of little-endian float32s as the
endpoint returned it, never normalized. The same chunk under two inputs is two lines, one vector.
The manifest records the docproc run read, each capability's endpoint, model, and settings, the
embedding space (its dimensions and whether every vector is unit length within 0.001), each
transcript's seams, each refused output, and the counts, among them the tokens each capability's
requests took in the run, as the endpoint's replies report them.

Each result is held as its request completes, keyed by the SHA-256 of its content, the
capability, the model, and the settings that change it: the prompt, cap, and reasoning effort
for vision, the prompt and language for transcription, and the chunk bound and the chunker's
version for embeddings. A run over the same docproc run and configuration sends no request but
the listings and writes the same chunk lines under a new run id; a changed chunk bound cuts and
embeds again and sends no image and no window; a run that stopped part way, its manifest naming
the error, resumes from what it held. A refusal the endpoint gives with a 4xx reply is held as a
verdict; any other is sent again by the next run. The directory is schema `"1"`, the contract the capstone reads: every field,
the path rules, the held results' keys, and how a chunk joins back to docproc's run are in
[docs/docinfer-schema.md](docs/docinfer-schema.md).

[The docinfer finding](docs/findings/docinfer.md) records docinfer over ~/docs through set A on
gb10-2 and its re-run, the sample run before it, and what vision adds over a text layer.

### The integration tier

`mise run integration` runs docinfer end to end, against the committed docproc run
(`internal/docinfer/app/testdata/docproc`), through the configuration `DOCINFER_CONFIG` names, and
skips it when that is unset:

```sh
DOCINFER_CONFIG="$PWD/config/example" mise run integration
```

It wants the models check to pass, the fixture's image described, its three windows transcribed
and joined into a transcript that isn't empty, and every chunk embedded at the space's dimensions,
768 for set A's embedding model. It never loads a model.

## gb10-2

gb10-2 left the spike on 2026-11-09 for experiment.hosting's spike-model-layout, which runs in its
own session. spike-model-layout's `hosts` task moved gb10-2's router preset mount onto
spike-model-engines' clone there and removed this repository's clone from gb10-2
(spike-model-layout's `docs/findings/hosts.md`, section D). `mise run host-update` is therefore no
longer run: it would pull the removed clone and point gb10-2's router back at this spike's preset.
The host scripts stay as the record of how the spike ran set A, described below as they ran.

gb10-2 served set A from llama.cpp's router container, run from this repository's clone there,
`~/repos/spike-ai-pipeline`, by spike-model-engines' scripts copied into `host/` (`host/README.md`).
Each session started with `mise run host-update`, on the laptop, which drove gb10-2 over Tailscale
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

host-update fast-forwards the clone's branch from its upstream. A task branch not yet merged had
no upstream, so a session delivered it from the laptop by push before running host-update:

```sh
git push ssh://gb10-2/~/repos/spike-ai-pipeline <branch>:refs/heads/<branch>
```

host-update sets the clone's `receive.denyCurrentBranch` to `updateInstead`, so the push updated
the checked-out branch and its files. Once a task merged, the clone was switched to `main`, which
tracks `origin/main`, and host-update pulled it from then on:
`ssh gb10-2 git -C repos/spike-ai-pipeline switch main`.

`mise run integration` probes the router `SPIKE_ROUTER_URL` names from the laptop (`probe/`). It
checks every set A capability, and checks the build, sampling, slots, and prompt cache against the
pin and the preset. Its default still names gb10-2's router, `http://gb10-2:8080`, which now
serves spike-model-layout's preset rather than this spike's, so a run there may differ from set A
as the spike ran it.
[The host finding](docs/findings/host.md) records its runs, including whether the router answers
`/v1/responses` and whether transcription returns timestamps, and what the router's
`/v1/messages` and `/v1/responses` drop, from the client's integration runs.

## The question

See `context/README.md`, "The question", and "The evidence".

## The answer

**Question:** Can a Go model client and a message, document, and stream processing layer, built
under go-elemental, carry a transactional, evented pipeline (metadata capture, inference
extraction, and embeddings stored in PostgreSQL with pgvector for semantic search) that a browser
follows stage by stage in real time?

**Answer:** Yes for the two layers it kept: a standard-library Go client over three wire formats
and a processing layer of owned system-tool adapters carried every image, audio window, and text
prepared from a real share, ~/docs, through vision, transcription and embeddings with no request
failed; the transactional, evented pipeline in pgvector and the browser that follows it moved,
unproven, to capstone.data-intelligence.

The items keep the spike's own numbers (`context/README.md`, "The evidence"). P stands for
spike-ai-pipeline-processing, and "the findings tier" for a test tier that runs the measurements
and commits their results.

1. The client: chat with tools and a reasoning round trip, streamed and unstreamed, vision, audio
   input (asserted on Chat Completions), embeddings, and transcription each pass against set A
   on gb10-2; the retry policy is proven by unit tests only. docinfer then ran vision,
   transcription, and embeddings through the client over all of ~/docs: 3,470 images, 1,254
   audio windows, and 28,937 chunk lines in 21,634 s, 0 of 6,172 requests failed, and the re-run
   sent no inference request (3.1 s). 272 vision answers stopped at the 2,048-token cap, 262 of
   them synthetic pages of random letters, and a first run with the model reasoning was stopped
   after 454 empty answers of 2,086. Proven by the client's integration tier (`integration/`,
   b11669) and a running demo, docinfer over ~/docs (b12098, 2026-11-08):
   [docinfer finding](docs/findings/docinfer.md).
2. Anthropic Messages and Open Responses fit under the same request and response model as Chat
   Completions with no change to callers: every `Generate` and `GenerateStream` scenario runs
   through all three from one piece of test code
   ([`integration/router_test.go`](https://github.com/JaimeStill/spike-ai-pipeline-client/blob/main/integration/router_test.go)).
   The router behind them drops hints: its `/v1/messages` drops `output_config`, so reasoning
   effort and structured output's schema reach no model, a tool's `strict`, and the `none` tool
   choice; its `/v1/responses` drops `text` and
   reasoning's summary, reports every reply completed, and refuses audio; at b12098 Messages also
   ignores `thinking: disabled`. Proven by the client's integration tier against gb10-2 (b11669,
   2026-10-27 and 2026-10-28): [host finding](docs/findings/host.md).
3. The client's base module is standard library only. Proven by the check only: a strict
   depguard rule in the client's lint, over every Go file, tests included, allows nothing but
   the standard library and the module
   ([`.golangci.yml`](https://github.com/JaimeStill/spike-ai-pipeline-client/blob/main/.golangci.yml)).
4. A PDF's pages go through vision whole and as a page stream. Ghostscript wins the renderer
   bake-off: it ties the others on accuracy and leads on time (0.126 s a page) and memory
   (27 MiB); it is AGPL, so the package runs it as a separate program. The model reads with
   reasoning off: with it on, 8 of 105 scanned pages came back empty and it invented seal text.
   Page 7's NOFORN is read on no build at any image budget. P releases preparation only; the
   inference is test code. Proven by P's integration and bake-off tiers against gb10-2 (b11669,
   2026-10-29 and 2026-10-30):
   [documents finding](https://github.com/JaimeStill/spike-ai-pipeline-processing/blob/main/docs/findings/documents.md).
5. Mixed. Long audio cut at its silences and transcribed 4 windows at once is 2.6 to 3.4 times
   faster than one unsplit pass, with no window failed up to a 4.4 h audiobook, but less accurate
   on many speakers (word error rate 0.298 against 0.258 on a 4-speaker meeting, 0.069 against
   0.053 on an 8-speaker call) and the same or slightly better on read speech and an accented
   call; most of the seams' excess edits are at seams with no words to align. The gpt-oss seam
   pass is falsified: it made every run worse. Proven by P's findings tier against gb10-2
   (b12098, 2026-11-05 and 2026-11-06):
   [audio finding](https://github.com/JaimeStill/spike-ai-pipeline-processing/blob/main/docs/findings/audio.md).
6. Skype for Business conversation history in a PST passes through as conversation records: each
   of the fixture's 4 conversation items equals the hand truth, and a message beside them stays a
   message. Fixture only: the PST is generated, its transcript HTML is invented, since no open
   sample exists, the turns within a transcript aren't parsed, and no real store was read.
   Proven by P's findings tier only (2026-11-07, no model):
   [formats finding](https://github.com/JaimeStill/spike-ai-pipeline-processing/blob/main/docs/findings/formats.md).
9. The dependency runs one way, processing to client. P v0.9.0 requires client v0.4.0 but imports
   it only in its test-support packages (`internal/visiontest`, `internal/audiotest`) and tests;
   its released packages prepare content and run no inference. The root's docinfer imports the
   client (its `config`, `embed`, `infer`, `transcribe`, and `vision` packages) and P. Proven by
   the check, whose depguard rule keeps the client from importing anything but the standard
   library (P's rule allows the client anywhere, so the test-only import is how v0.9.0 stands,
   not a rule), and by docinfer's running demo (item 1). Where the layer lives is experiment.ai's
   intake's decision. ([P's `go.mod`](https://github.com/JaimeStill/spike-ai-pipeline-processing/blob/main/go.mod))
11. ImageMagick's corrections don't help vision reliably: at 150 dpi the synthetic faults leave
    nothing to fix, at 72 dpi Unsharp helps the hardest page while Deskew and Despeckle hurt it,
    and on the real scans which corrections help and hurt flips with the DPI and the build.
    Proven by P's bake-off tier against gb10-2 (b11669, 2026-10-29 and 2026-10-30):
    [documents finding, "Evidence 11"](https://github.com/JaimeStill/spike-ai-pipeline-processing/blob/main/docs/findings/documents.md#evidence-11-do-image-corrections-help).
12. The Open XML CLI keeps every element the pool's truth holds, 1,768 of 1,768, where markitdown
    and xberg lose most slide lists, chart data, headers, footers, and comments; its median file
    takes 17.2 ms (xberg 12.3 ms, markitdown 675 ms) at 35.4 MiB, from a 28.80 MiB binary. The
    legacy round trip through LibreOffice loses structure in .doc (headings become one-item
    lists), a little in .xls, nothing in .ppt; its container is 441 MB to pull and 1.02 GB on
    disk, at 0.75 to 1.04 s a conversion. Proven by office's bench (2026-10-31, no model) and P's
    integration tier against gb10-2 (b11669, 2026-11-02):
    [office-cli finding](https://github.com/JaimeStill/spike-ai-pipeline-office/blob/main/docs/findings/office-cli.md),
    [office finding](https://github.com/JaimeStill/spike-ai-pipeline-processing/blob/main/docs/findings/office.md).
13. Live media is processed as it arrives: three videos read at real time, with no window, frame,
    or request failed; a window is yielded once the stream passes its 30 s limit (at most 15 s
    after its last sample, on NASA), transcribed 0.8 to 4.6 s later, a frame described within
    4 s, and a cancellation stops the stream within 0.5 s. The live transcript matches the file
    path's under the same detector, and is 0.015 worse on AMI than the file path's default. Proven
    by P's findings tier's paced runs against gb10-2 (b12098, 2026-11-06):
    [streams finding](https://github.com/JaimeStill/spike-ai-pipeline-processing/blob/main/docs/findings/streams.md).
14. One tool runs through each runner under the same contract: ffprobe through the exec runner
    and through the container runner, in a digest-pinned image
    ([`tool/ffprobe_integration_test.go`](https://github.com/JaimeStill/spike-ai-pipeline-processing/blob/main/tool/ffprobe_integration_test.go)).
    docproc's run over ~/docs wired all ten stages, soffice in a digest-pinned container and the
    rest on the host: 592 inputs, 515 prepared, 4 failed (three malformed or form-broken PDFs
    and OneNote's encryption fixture). Proven by P's integration tier and a running demo, docproc
    over ~/docs (2026-11-07, no model): [docproc finding](docs/findings/docproc.md).
16. The mail CLI recovers every field the generated fixtures' truth expects that the generator
    stores (147 of 180, 33 not stored, none wrong), 47 of 47 of Tika's threading truth, and 21 of
    22 and 31 of 31 .msg assertions, missing only an Exchange recipient's SMTP address, by
    design; it reads a 2.1 GB store in 13.27 s at 65 to 67 MiB. The adapter rebuilds 11 of 11
    threads, routes 51 of 51 attachments as the hand truth says, and adds 1 to 3% to mail's
    time. Proven by the mail bench (from 2026-11-03, no model) and P's findings and integration
    tiers, vision against gb10-2 (b11669, 2026-11-03 and 2026-11-04):
    [mail-cli finding](https://github.com/JaimeStill/spike-ai-pipeline-mail/blob/main/docs/findings/mail-cli.md),
    [email finding](https://github.com/JaimeStill/spike-ai-pipeline-processing/blob/main/docs/findings/email.md).
17. The OneNote CLI recovers every expected field of the public fixtures (236 of 236 over the
    sections, 138 of 138 over the notebooks, 129 of 129 content fields) and reads a 161 MB section
    in 0.24 s at 135 MiB; there is no authored notebook, so no authored hand truth. Every OneNote
    medium routes to its real stage, 225 parts from 119 inputs with no mismatch; vision reads a
    page's ink at a character error rate of 0.028 against 0.68 for OneNote's own recognition.
    Proven by the onenote bench (2026-11-04, no model) and P's findings tier, vision against
    gb10-2 (b11669, 2026-11-05):
    [onenote-cli finding](https://github.com/JaimeStill/spike-ai-pipeline-onenote/blob/main/docs/findings/onenote-cli.md),
    [onenote finding](https://github.com/JaimeStill/spike-ai-pipeline-processing/blob/main/docs/findings/onenote.md).

The original evidence's other items moved:

- 7, the transactional pipeline into pgvector with its events and UI: the capstone's build on
  the client and processing layer experiment.ai's intake promotes, after its design tasks.
- 8, the client unchanged through a gateway: spike-model-gateway's answer.
- 10, pgvector as a declared native-tier extension: the capstone, and the pgvector v1.data task
  that lands the Postgres-extension strategy.
- 15, the browser's consumption of the event sequence: the capstone's events design task, on
  the web client stack experiment.html's intake and v1.client promote.
- The gates, spike-model-layout's and spike-model-gateway's answers: the capstone's gates, which
  add experiment.ai's, experiment.hosting's, and experiment.html's intakes, graph-ergonomics,
  and the pgvector v1.data task.

Each moved with the capstone, not failed:
[`standards-lab/context/data-intelligence.md`](https://github.com/standards-lab/org/blob/main/context/data-intelligence.md),
"Evidence" and "Gates".
