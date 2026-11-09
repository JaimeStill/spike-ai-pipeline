# docproc's run directory, schema "1"

`docproc run --out <dir>` writes a run directory, and docinfer, the inference CLI, reads one. This
page is the contract between them: the directory's layout, each run's manifest, and its record, one
JSON object per line. It is written from the types in `internal/docproc/schema`, whose doc comments
are the source; where this page and they disagree, they win and this page is wrong.

A reader checks the manifest's `"schema"` is `"1"` before reading anything else.

## Layout

```
<run dir>/
  index/                                processing's file index, shared by every run
  outputs/<sha256>/<stage>/<n><ext>     each output, stored once
  quarantine/<sha256>/content           a quarantined part's file,
  quarantine/<sha256>/flags.json        beside its flags
  runs/<run id>/manifest.json           each run's manifest
  runs/<run id>/record.jsonl            its record, an event per line
  runs/<run id>/report.html             its HTML report
```

- `<run id>` is a version 7 UUID, so the runs sort by when each began; the greatest is the latest.
- `<sha256>` is the lowercase hex SHA-256 of the part's content.
- `<stage>` is the stage's name, lowercased with any character outside `a-z`, `0-9`, and `-` made
  `-`, then `-` and the first twelve hex digits of the SHA-256 of its whole identity (name,
  version, and configuration hash), such as `document-3f2a…`. A new tool version or option set
  stores its outputs apart.
- `<n>` counts the part's outputs under that stage from 1, in the stage's order.
- `<ext>` comes from the output's media type through a fixed table, the same on every host:
  `.txt` (text/plain), `.md` (text/markdown), `.json` (application/json), `.png`, `.jpg`, `.gif`,
  `.bmp`, and `.wav` (audio/wav, audio/x-wav, audio/wave). Any other media type takes the
  extension of the file the stage wrote, lowercased, or none.

`index/` is processing's `dispatch.FileIndex`, and its format is processing's, not this contract's.
`report.html` is a rendering of the manifest and record (`docproc report`), not data to read. A
`.work-*` directory under the root exists only while a run is dispatching and is removed when it
ends.

Every file docproc writes is written whole, to a temporary name in its directory, synced, and
renamed, so a reader sees a whole file or none. An output or quarantined file already in place is
never rewritten.

## Paths

- **Inputs are absolute.** An input's path, a file's `"arg"`, and the manifest's `"inputs"` are
  absolute and cleaned: docproc only reads them, and they name the caller's disk.
- **Files docproc writes are relative to the run directory**, with slashes: an output's `"ref"`, a
  quarantine's `"file"` and `"flags"`, and the manifest's `"record"`. The run directory can be
  moved or archived whole.
- **A file's `"path"` is relative to its `"arg"`**, with slashes, `"."` when the argument is the
  file itself, so joining the two always names the file.
- **A tool's `"from"` is the host's path**, or its container image.

## Times

Every time is RFC 3339 in UTC with nine digits of nanoseconds, such as
`"2026-11-07T15:04:05.123456789Z"`.

## Fields

A field this page doesn't call optional is always present, its zero value included. An optional
field is absent, not null or empty, when it has no value.

### The manifest

`runs/<run id>/manifest.json`, one JSON object, written as the run starts and again, whole, as it
ends.

| Field | Type | Meaning |
|---|---|---|
| `schema` | string | The schema's version, `"1"`. |
| `run` | string | The run's id, a version 7 UUID. |
| `start` | time | When the run started. |
| `end` | time | When it ended. Optional: absent while it runs, or when it never ended. |
| `workers` | integer | How many inputs were dispatched at once. |
| `inputs` | array of string | The path arguments, files and directories, each absolute, cleaned. |
| `record` | string | The run's record, relative to the run directory: `runs/<run id>/record.jsonl`. |
| `tools` | array of tool | Each tool the run tried to build, in the tools table's order. |
| `stages` | array of stage state | Each route's stage, in the stage table's order. |
| `error` | string | The error that stopped the whole dispatch. Optional: absent when none did. |

A **tool**:

| Field | Type | Meaning |
|---|---|---|
| `program` | string | Its program, such as `"gs"`. |
| `minimum` | string | The minimum version processing declares for it. |
| `from` | string | Where it was found, a host path, or, for a tool run in a container, its image. Optional: absent when it wasn't found. |
| `version` | string | Its version. Optional: absent when it wasn't built. |
| `error` | string | Why it wasn't built, missing or too old. Optional. |

A tool only an excluded route needs isn't looked up, so it isn't listed.

A **stage state**:

| Field | Type | Meaning |
|---|---|---|
| `route` | string | The route, one of `document`, `office`, `vision-converted`, `vision`, `nested`, `onenote`, `archive`, `audio`, `video`, `text`. |
| `state` | string | `"wired"`, `"missing"`, or `"excluded"`. A missing or excluded route's parts come to `"no processor yet"`. |
| `stage` | stage | The wired stage's identity. Optional: absent unless wired. |
| `tools` | string | The wired stage's tools and their versions, such as `"gs 10.8.0, qpdf 12.4.2, pdfdetach 26.10.0"`. Optional: absent for a stage that runs none. |
| `reason` | string | Why a missing stage is missing. Optional. |

A **stage**, processing's `dispatch.Identity`; a part's outputs are prepared again whenever any of
it changes:

| Field | Type | Meaning |
|---|---|---|
| `name` | string | The stage's name, such as `"document"`. |
| `version` | string | Its tools' versions, as processing's `dispatch.Versions` joins them, and its adapter's. |
| `config` | string | A hash of the options that change what it prepares. |

### The record

`runs/<run id>/record.jsonl`, one event per line, in the order the events arrived. An input's lines
keep their order: the output and part lines of each part found in it come before its own, and its
own part line is its last. Different inputs' lines interleave. A seen part's subtree is copied in
from an earlier record, in the same order ([Seen content and replayed
subtrees](#seen-content-and-replayed-subtrees)), so a record is whole on its own.

Every line has an `"event"`, its kind, and an `"input"`, the absolute, cleaned path of the input it
belongs to. The three kinds:

- **`"output"`**: one output a part's stage prepared, or, for a part the index has seen, one output
  it recorded.
- **`"part"`**: a part's own event, its outcome, once every output and every part found in it has
  its line. An input's own part line, at depth 0, is its last.
- **`"error"`**: an input whose dispatch stopped with an error before its own event, such as an
  input that can't be read. It has no part line.

| Field | Type | `output` | `part` | `error` | Meaning |
|---|---|---|---|---|---|
| `event` | string | yes | yes | yes | The line's kind. |
| `input` | string | yes | yes | yes | The input's absolute path. |
| `part` | part | yes | yes | no | The part the line is about. |
| `file` | file | no | input's only | when readable | The input's file metadata: on the input's own part line, and on an error line when the file could still be read. |
| `stage` | stage | optional | optional | no | The stage of the part's route; absent when its route has no stage. |
| `output` | output | yes | no | no | The output. |
| `outcome` | string | `"seen"` or absent | yes | no | On a part line, what became of the part (below). On an output line, `"seen"` for an output the index recorded earlier, or one copied from an earlier record, absent for one prepared in this run. |
| `recorded` | string | optional | optional | no | For a seen part or output, the outcome the index recorded for it when it was prepared; for a copied line, the outcome its record gave it. |
| `reason` | string | no | optional | no | Why the part came to its outcome. |
| `error` | string | no | optional | yes | On a part line, the error a failed or encrypted part came to; on an error line, the error that stopped the input's dispatch. |
| `outputs` | integer | no | yes | no | How many outputs the part yielded, or, seen, recorded. |
| `children` | integer | no | yes | no | How many parts its stage found in it. |
| `warning` | string | no | optional | no | On a seen part line whose children no record in the run directory holds, so none of their lines is copied: what is missing. |
| `quarantine` | quarantine | no | quarantined only | no | Where a quarantined part's file is kept. |
| `time` | span | no | input's only | yes | The input's time, on its own part line and on an error line. |

A part line's `"outcome"`:

| Outcome | Meaning |
|---|---|
| `prepared` | Its stage prepared it. |
| `no processor yet` | Its route has no wired stage (missing, excluded, or route `unrouted`), or its stage has no processor for its kind yet. |
| `seen` | The index already holds its content, from this run or an earlier one over the same run directory: its outputs are the index's, not prepared again, and the lines of the parts found in it are copied from the record that holds them. |
| `cycle` | Its SHA-256 is one of its ancestors': it is neither prepared nor opened. |
| `quarantined` | Its inspection's verdict quarantines it: no stage runs, its file is kept under `quarantine/`, and its flags hold the evidence. |
| `encrypted` | Its stage can't read it because it is encrypted. |
| `failed` | It couldn't be read, routed, inspected, or prepared; `error` says why. The run goes on. |
| `limited` | It is past a dispatch cap: neither inspected nor prepared, and its siblings go on. |

A **part**, an input or one found in it:

| Field | Type | Meaning |
|---|---|---|
| `name` | string | Its name: an input's file name, or the name its parent's stage gave it, such as an attachment's stored name. |
| `size` | integer | Its size in bytes. |
| `sha256` | string | The lowercase hex SHA-256 of its content; empty for a part that couldn't be read. |
| `content_type` | content type | What its content is. |
| `flags` | array of flag | Its inspection's flags. Optional: absent when it has none. |
| `parent` | string | The SHA-256 of the part it was found in. Optional: absent for an input. |
| `depth` | integer | 0 for an input, one more than its parent's for a part found in it. |
| `where` | string | Where its parent's stage found it, in that stage's words, such as `"zip entry notes/a.txt"`. Optional: absent for an input. |

A nested part has no file of its own on the caller's disk, so it carries no `file` and no
filesystem times. Its tree is rebuilt from `parent` and `depth`: a part's own line takes the parts
before it at one depth more whose `parent` is its `sha256`.

A **content type**:

| Field | Type | Meaning |
|---|---|---|
| `kind` | string | The format its signature names, processing's `route.Kind`, such as `"pdf"` or `"zip"`. Optional: absent when no signature processing knows matches. |
| `route` | string | The route processing gave it, which names the stage that prepares it. Optional: absent for a part that couldn't be read. |
| `route_reason` | string | Why processing gave it that route. Optional. |
| `stored` | string | The MIME type it was stored under, such as an attachment's declared type. Optional: absent when its container declared none, as for every input. |
| `detected` | string | The MIME type `net/http.DetectContentType` names for its first 512 bytes. Optional: present on an input's own part line alone. |

A **flag**, one finding of a part's inspection:

| Field | Type | Meaning |
|---|---|---|
| `check` | string | The check that raised it, such as `"zip-traversal"`. |
| `class` | string | What it means for the part, such as `"unsafe"`. |
| `evidence` | string | Where it is, in a few words, such as `entry "../evil.txt"`. |

A **file**, an input's metadata, read when its dispatch ends:

| Field | Type | Meaning |
|---|---|---|
| `name` | string | Its file name. |
| `arg` | string | The path argument it came from, a file or a directory, absolute, cleaned. |
| `path` | string | Its path relative to `arg`, with slashes; `"."` when the argument is the file itself. |
| `size` | integer | Its size in bytes. |
| `mode` | string | Its mode and permissions as `ls` writes them, such as `"-rw-r--r--"`. |
| `modified` | time | Its modified time. |
| `created` | time | Its created time, statx's birth time. Optional: absent where the filesystem records none, and on any system but Linux. |

An **output**, its content in the output store:

| Field | Type | Meaning |
|---|---|---|
| `kind` | string | `"text"`, `"markdown"`, `"image"`, `"record"`, or `"audio"`. |
| `name` | string | The name the stage gave it, such as `"page 3"`. Optional. |
| `media_type` | string | Its media type, such as `"image/png"`. Optional. |
| `meta` | object, string to string | What else the stage says of it, such as a page's number. Optional. |
| `ref` | string | Its stored file, relative to the run directory: `outputs/<part sha256>/<stage>/<n><ext>`. |

A **quarantine**, under `quarantine/<part sha256>/`:

| Field | Type | Meaning |
|---|---|---|
| `file` | string | The part's content, `quarantine/<sha256>/content`, with no extension. |
| `flags` | string | Its flags, a JSON array of flag, `quarantine/<sha256>/flags.json`. |

A **span**, an input's time:

| Field | Type | Meaning |
|---|---|---|
| `start` | time | When the pool took the input, any wait for a free worker included. |
| `end` | time | When its last event arrived. |

## Seen content and replayed subtrees

Content is prepared once per run directory, by its SHA-256 and its stage's identity. The file index
records each part's outcome and outputs, and each output's `ref` as docproc set it when it stored
the output. When the same content comes again, in the same run or a later one over the same run
directory, under the same stage identity, its part line's outcome is `"seen"`, with `"recorded"`
naming the outcome it was prepared to, and each of its output lines is `"seen"` too, its `ref`
replayed from the index: it names the copy the run that prepared it stored, which is still under
`outputs/`.

Processing doesn't dispatch the parts found in seen content again, so docproc copies them. Before
the seen part's own line, it writes its subtree, the part lines of every part found in it, at any
depth, and their output lines, copied from the earliest record in the run directory that holds
that subtree for the same SHA-256 and stage identity, this run's own record last, for content seen
twice in one run. Each copied line:

- carries this run's `"input"`, and its part's `"depth"` under the seen part's, which can differ
  from the record it came from;
- is `"seen"`, its `"recorded"` the outcome its record gave it: a part line's outcome, or, for a
  line that was seen there too, its `"recorded"`; an output prepared there records its part's
  outcome;
- keeps everything else its record gave it, its `"reason"`, `"error"`, counts, `"quarantine"`,
  and its output's `"ref"`, which names a file under `outputs/`.

A copied part whose record holds no lines for its own children, as a record written before docproc
copied subtrees has for a seen part, takes them from the earliest record that does. So every run's
record is whole on its own: a reader never needs an earlier run's record to follow a seen part or
its outputs, and its part tree has the same lines the run that prepared it recorded.

A seen part with children no record in the run directory holds, as when `index/` is kept and its
`runs/` are not, is recorded with what the index gave it and a `"warning"` saying its children's
lines are missing; none of them is copied, and the run goes on. The input's terminal line ends its
counts with `warnings <n>`, the closing summary counts them, and the report shows the warning on the
part and counts them at its head. A record that can't be read is skipped, the terminal saying so,
and copies nothing.

Processing's index records a part only when its stage ran and its whole subtree came to an outcome
its content decides. A part that failed, or that holds a `failed` or `limited` part at any depth, is
not recorded, so a later run prepares it again and retries what failed; its outputs are stored
again under the same `ref`s, and the copies already there are kept. A part whose route has no stage
is not recorded either, so a later run gives it `"no processor yet"` again, or prepares it once its
stage is wired. Everything else a later run over unchanged inputs sees: it prepares nothing, and
every seen output's `ref` names a file an earlier run stored.

## What's stable

Within schema `"1"`: the layout above, every field's name, type, and meaning, the optional fields'
absence rules, the path rules, the outcome values, and the output store's naming. `index/`'s format
and `report.html`'s markup are not part of the contract.
