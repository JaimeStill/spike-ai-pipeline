# docinfer's directory, schema "1"

`docinfer run <docproc-dir> --out <dir>` writes docinfer's directory, and the capstone reads one:
its chunk records and vectors are what the capstone loads. This page is the contract: the
directory's layout, each run's manifest, its chunk record, one JSON object per line, the held
results, the stored texts, and how a chunk joins back to the docproc run it was cut from. It is
written from the types in `internal/docinfer/outdir`, whose doc comments are the source; where
this page and they disagree, they win and this page is wrong.

A reader checks the manifest's `"schema"` is `"1"` before reading anything else.

## Layout

```
<dir>/
  results/<capability>/<xx>/<key>.json   each held result, by its key
  texts/<xx>/<sha256>.txt                each text docinfer stored: a vision text, a window's
                                         transcript, or a joined transcript
  runs/<run id>/manifest.json            each run's manifest
  runs/<run id>/chunks.jsonl             its chunk record, a chunk line per chunk per occurrence
  runs/<run id>/report.html              its HTML report
```

- `<run id>` is a version 7 UUID, so the runs sort by when each began; the greatest is the latest.
- `<capability>` is `embeddings`, `vision`, or `transcription`.
- `<key>` is a held result's key ([Held results](#held-results)), and `<sha256>` the lowercase hex
  SHA-256 of a stored text's bytes; `<xx>` is the first two hex digits of either.

A run over an existing directory reuses its `results/` and `texts/` under a new run id. Each run's
`runs/<run id>/` is its own: no run reads or changes another's. `report.html` is a rendering of the
manifest, the chunk record, and the stored texts (`docinfer report`), not data to read.

Every file docinfer writes is written whole, to a temporary name in its directory, synced, and
renamed, so a reader sees a whole file or none. A held result is written as its request
completes, so a run that stops part way keeps everything it finished.

## Paths

- **Files docinfer writes are relative to its directory**, with slashes: the manifest's
  `"chunks"`, a transcript's `"file"`, a source's `"file"`, and a held result's `"text"`. The
  directory can be moved or archived whole.
- **docproc's outputs are relative to the docproc run directory**, with slashes, as docproc's
  record has them: an output's `"ref"`, a source's `"ref"`, a failure's `"ref"`, and a
  transcript's `"windows"`. The manifest's `docproc.dir` names that directory.
- **The docproc run directory is absolute**, cleaned, as the run that read it resolved it. A
  reader that has moved it resolves `"ref"`s against where it now is.
- **An input's path is absolute**, as docproc's record gives it, and names the caller's disk.

## Times

Every time is RFC 3339 in UTC with nine digits of nanoseconds, such as
`"2026-11-07T15:04:05.123456789Z"`, as docproc's are.

## Fields

A field this page doesn't call optional is always present, its zero value included. An optional
field is absent, not null or empty, when it has no value.

Fields this page calls docproc's (a chunk line's `file` and `content_type`, a part's names, and an
output's) are copied from docproc's record, schema "1", and mean what
[docs/docproc-schema.md](docproc-schema.md) says they mean.

### The manifest

`runs/<run id>/manifest.json`, one JSON object, written as the run starts and again, whole, as it
ends.

| Field | Type | Meaning |
|---|---|---|
| `schema` | string | The schema's version, `"1"`. |
| `run` | string | The run's id, a version 7 UUID. |
| `start` | time | When the run started. |
| `end` | time | When it ended. Optional: absent while it runs, or when it never ended. |
| `docproc` | docproc | The docproc run it read. |
| `capabilities` | array of capability | Each capability the run ran, in the order it ran them: `vision`, `transcription`, then `embeddings`, each that is configured. Embeddings is always configured. |
| `skipped` | array of skipped | Each capability it didn't run, and why. Optional: absent when it ran all three. |
| `embedding_space` | space | The embedding space of the run's vectors. Optional: absent while the run runs, and in a run that stopped before its embeddings completed. |
| `chunks` | string | The run's chunk record, relative to the directory: `runs/<run id>/chunks.jsonl`. Optional: absent until it is written, and in a run that stopped. |
| `transcripts` | array of transcript | Each transcript the run joined, one per part and stage with audio windows, in the docproc record's order of each part's first window. Optional: absent when transcription was skipped or the docproc run has no window. |
| `failures` | array of failure | Each image output line whose image vision refused, then each audio window output line whose window transcription refused, each in the docproc record's order. Optional: absent when nothing was refused. |
| `counts` | counts | What the run read, cut, and sent. |
| `error` | string | The error that stopped the run. Optional: absent when nothing did. |

A **docproc**:

| Field | Type | Meaning |
|---|---|---|
| `dir` | string | The docproc run directory, absolute. |
| `run` | string | The docproc run's id: its manifest and record are `<dir>/runs/<run>/`. |

A **capability**:

| Field | Type | Meaning |
|---|---|---|
| `name` | string | `"vision"`, `"transcription"`, or `"embeddings"`. |
| `endpoint` | string | The endpoint's name in the configuration. |
| `base_url` | string | The endpoint's API root. |
| `model` | string | The model the requests named. |
| `settings` | object | The capability's settings, by their configuration names: vision's `prompt` (string), `max_tokens`, and `concurrency`; transcription's `prompt` and `language` (strings, empty when unset) and `concurrency`; embeddings' `batch`, `chunk_size`, and `concurrency`. The counts are integers. |

A **skipped**:

| Field | Type | Meaning |
|---|---|---|
| `name` | string | `"vision"` or `"transcription"`. |
| `reason` | string | Why it was skipped: `"not configured"`. |

A **space**, the embedding space of the run's vectors, measured over every distinct chunk's vector,
held or embedded in this run:

| Field | Type | Meaning |
|---|---|---|
| `endpoint` | string | The endpoint's name that made them. |
| `model` | string | The model that made them. |
| `dimensions` | integer | Every vector's length; docinfer stops a run whose vectors disagree. 0 when there is none. |
| `vectors` | integer | How many distinct vectors were measured. |
| `unit_length` | boolean | Whether every vector's Euclidean norm is within `tolerance` of 1: \|norm − 1\| ≤ tolerance. `false` when there is no vector. |
| `tolerance` | number | The tolerance, `0.001`. |
| `min_norm` | number | The least norm; 0 when there is no vector. |
| `max_norm` | number | The greatest norm; 0 when there is no vector. |
| `chunk_size` | integer | The bound each chunk was cut to, in characters (runes). |
| `chunker` | string | The chunker's version, `"1"`, which names how texts are cut. |

A **transcript**, one part's audio windows, cut by one stage, joined in plan order by processing's
`audio.Join`:

| Field | Type | Meaning |
|---|---|---|
| `input` | string | The input's absolute path. |
| `part` | part | The part whose windows they are. Optional. |
| `stage` | string | The name of the stage that cut the windows, such as `"audio"` or `"video"`. Optional. |
| `windows` | array of string | Each window's stored file, relative to the docproc run directory, in plan order, by the index each window's meta gives; `""` for an index the plan counts but the record has no line for, one its stage failed to cut. |
| `failed` | array of integer | The plan indexes of the windows that joined as an empty transcript: refused, or with no line. Optional: absent when none did. |
| `file` | string | The joined text, relative to the directory: `texts/<xx>/<sha256>.txt`. |
| `sha256` | string | The joined text's lowercase hex SHA-256. |
| `seams` | array of seam | How each seam was merged, one fewer than the windows: `seams[i]` is window `i`'s with window `i+1`. Empty, not absent, for one window. |

A **seam**, as `audio.Join` reports it:

| Field | Type | Meaning |
|---|---|---|
| `matched` | boolean | Whether the two windows' words aligned on at least 2 words, so they were spliced; an unmatched seam's windows are concatenated whole. |
| `matches` | integer | How many words the alignment matched; 0 when unmatched. |
| `overlap` | integer | How many words it overlapped; 0 when unmatched. |
| `position` | integer | The index, in the transcript's words, of the first word the later window gave. |

A **failure**, one output line whose image or window the endpoint refused; it has no text and so
no chunk line. An image or window under two output lines is two failures.

| Field | Type | Meaning |
|---|---|---|
| `capability` | string | `"vision"` or `"transcription"`. |
| `input` | string | The input's absolute path. |
| `ref` | string | The output's stored file, relative to the docproc run directory. |
| `error` | string | The endpoint's reply. |

The **counts**:

| Field | Type | Meaning |
|---|---|---|
| `outputs` | integer | The docproc run's output lines. |
| `texts` | integer | The text and Markdown output lines embedded, one per line. |
| `metadata` | integer | The metadata texts, one per input with file metadata. |
| `chunks` | integer | The distinct chunks, by content. |
| `lines` | integer | The chunk record's lines, one per chunk per occurrence. |
| `held` | integer | The distinct chunks whose embedding an earlier run held. |
| `embedded` | integer | The distinct chunks this run embedded. |
| `rejected` | integer | The distinct chunks the endpoint rejected, in this run or held. |
| `requests` | integer | The embeddings requests this run sent. |
| `images` | integer | The docproc run's image output lines, whether vision ran or was skipped. |
| `vision` | vision counts | What vision did. Optional: absent when it was skipped. |
| `windows` | integer | The docproc run's audio window output lines, an audio part's or a video's, whether transcription ran or was skipped. |
| `transcription` | transcription counts | What transcription did. Optional: absent when it was skipped. |
| `tokens` | tokens | The tokens each capability's requests took in this run. |

The **vision counts**:

| Field | Type | Meaning |
|---|---|---|
| `images` | integer | The distinct images, by content; an image under two output lines is sent once. |
| `held` | integer | The distinct images whose result an earlier run held. |
| `described` | integer | The distinct images this run sent and the model answered. |
| `incomplete` | integer | The distinct images whose text the model stopped before finishing, such as at the output cap, in this run or held; the text is kept and embedded. |
| `refused` | integer | The distinct images the endpoint refused, in this run or held. |
| `requests` | integer | The vision requests this run sent. |
| `texts` | integer | The vision texts embedded, one per image output line whose image has a text; an empty text is one, and cuts to no chunk. |

The **transcription counts**:

| Field | Type | Meaning |
|---|---|---|
| `distinct` | integer | The distinct windows, by content; a window under two output lines is sent once. |
| `held` | integer | The distinct windows whose result an earlier run held. |
| `transcribed` | integer | The distinct windows this run sent and the model transcribed. |
| `failed` | integer | The distinct windows the endpoint refused, in this run or held; each joins its transcript as an empty one. |
| `requests` | integer | The transcription requests this run sent. |
| `transcripts` | integer | The joined transcripts embedded, one per part and stage with windows; an empty transcript is one, and cuts to no chunk. |
| `seams_matched` | integer | The seams, in every transcript, whose windows' words aligned. |
| `seams_unmatched` | integer | The seams whose windows were concatenated whole. |

The **tokens**, each capability's requests' usage, summed over the replies the endpoint answered
in this run: a held result took no request, and a refusal or a rejection carries no usage, so
neither adds any. A re-run that sends no request has every count 0.

| Field | Type | Meaning |
|---|---|---|
| `vision` | usage | Vision's. Optional: absent when it was skipped. |
| `transcription` | usage | Transcription's. Optional: absent when it was skipped. |
| `embeddings` | usage | The embeddings'. An embedding generates nothing, so its `output` is 0. |

A **usage**, in tokens, as the endpoint's replies report them:

| Field | Type | Meaning |
|---|---|---|
| `input` | integer | The inputs': for vision, the image's and the prompt's; for transcription, the clip's and its instruction's; for embeddings, every chunk text's, the reply's `prompt_tokens`. |
| `output` | integer | The generated: the vision text's, reasoning included, or the transcript's. |
| `total` | integer | Input and output together, as the reply reports it; for vision through the Messages format, which reports none, their sum. |

An endpoint that reports no usage leaves its counts 0.

### The chunk record

`runs/<run id>/chunks.jsonl`, one chunk line per line, written whole as the run ends. A run that
stopped writes none. A chunk line is one chunk in one place its text occurs: the same text under
two inputs, or under two output lines of one input, is two sets of lines, each chunk's vector the
same in both.

The lines follow the docproc record's order, each source's chunks in their order within it:

- a text or Markdown output's chunks at its output line;
- a vision text's chunks at its image's output line;
- a joined transcript's chunks at the output line of the part's first window in the record;
- an input's metadata text's chunks at its own part line, or at its error line.

| Field | Type | Meaning |
|---|---|---|
| `text` | string | The chunk's text: the source text's bytes from `span.start` to `span.end`. |
| `index` | integer | The chunk's place among its source text's chunks, from 0. |
| `span` | span | The chunk's byte span in its source text. |
| `source` | source | The text the chunk was cut from. |
| `input` | string | The input's absolute path, as docproc's record gives it. |
| `file` | docproc file | The input's file metadata, docproc's. Optional: absent when docproc recorded none. |
| `content_type` | docproc content type | The input's content type, from its own part line, docproc's. Optional: absent for an input with no part line, one whose dispatch stopped with an error. |
| `part` | part | The part the source belongs to: for a metadata text, the input's own part. Optional: absent for an input with no part line. |
| `output` | output | The docproc output the source is, or was made from: a text or Markdown output itself; for a vision text, its image; for a transcript, the first window in plan order of those it was joined from, which names the part's stage directory. Optional: absent for a metadata text. |
| `vector` | string | The chunk's embedding ([Vectors](#vectors)). Optional: absent when the endpoint rejected the chunk. |
| `error` | string | Why the endpoint rejected the chunk. Optional: absent when it didn't. |

Every line has a `vector` or an `error`, never both.

A **span**, `[start, end)`:

| Field | Type | Meaning |
|---|---|---|
| `start` | integer | The byte offset of the chunk's first byte in its source text. |
| `end` | integer | The byte offset just past its last byte. |

A **source**:

| Field | Type | Meaning |
|---|---|---|
| `kind` | string | One of the kinds below. |
| `ref` | string | For kind `output`, the output's stored file, relative to the docproc run directory, the same as the line's `output.ref`. Optional: absent for every other kind. |
| `file` | string | For kinds `vision` and `transcript`, the text docinfer stored, relative to its directory: `texts/<xx>/<sha256>.txt`. Optional: absent for `output` and `metadata`. |
| `sha256` | string | The lowercase hex SHA-256 of the whole source text. |
| `incomplete` | string | For kind `vision`, why the model stopped before finishing the text, such as `"max_output_tokens"`, the text being what it gave. Optional: absent for a text it finished, and for every other kind. |

The source **kinds**:

| Kind | The source text |
|---|---|
| `output` | A docproc text or Markdown output's stored file, whatever its part came to: a failed part's outputs, prepared before it failed, and a seen part's are embedded like any other. docproc's record, image, and audio outputs are never embedded directly. |
| `metadata` | An input's metadata text, one per input whose own part line or error line carries file metadata, made by docinfer: one `label: value` line for each of the file's fields and its content type's, in docproc's schema order (`file name`, `path argument`, `path`, `size` in bytes, `mode`, `modified`, `created`, `kind`, `route`, `route reason`, `stored MIME type`, `detected MIME type`), a field with no value left out. It lists none of the input's parts. It is not stored as a file; the chunk lines hold it. |
| `vision` | The vision model's text of an image output, stored by docinfer. |
| `transcript` | The joined transcript of one part's audio windows, cut by one stage, stored by docinfer and recorded in the manifest's `transcripts`. |

A **part**, the part the source belongs to, in docproc's names:

| Field | Type | Meaning |
|---|---|---|
| `sha256` | string | docproc's `part.sha256`. |
| `name` | string | docproc's `part.name`. |
| `kind` | string | docproc's `part.content_type.kind`. Optional. |
| `route` | string | docproc's `part.content_type.route`. Optional. |
| `where` | string | docproc's `part.where`. Optional: absent for an input. |
| `depth` | integer | docproc's `part.depth`. |
| `outcome` | string | The outcome the part's own part line gives it, such as `"prepared"`, `"seen"`, or `"failed"`. Optional: absent when the record holds no part line for it. |

An **output**, in docproc's names:

| Field | Type | Meaning |
|---|---|---|
| `kind` | string | docproc's `output.kind`: `"text"`, `"markdown"`, `"image"`, or `"audio"`. |
| `name` | string | docproc's `output.name`. Optional. |
| `media_type` | string | docproc's `output.media_type`. Optional. |
| `ref` | string | docproc's `output.ref`, relative to the docproc run directory. |
| `meta` | object, string to string | docproc's `output.meta`, such as an audio window's `"window"` index. Optional. |

### Chunks

Each text is cut on its own, so no chunk spans two texts. Each distinct text, by its SHA-256, is
cut once, and each distinct chunk, by its text's SHA-256, is embedded once. The chunker cuts a text
to at most `chunk_size` characters (runes) along its structure, coarsest first: a text within the
bound is one chunk; otherwise it is cut before each Markdown ATX heading outside a fenced code
block, then at blank lines, then at sentence ends and line breaks, then every `chunk_size` runes,
packing whole pieces while they fit. Each chunk is trimmed of white space at its ends, and its span
is that of the trimmed text. Chunks never overlap, and no cut falls inside a UTF-8 character. An
empty or white-space-only text yields no chunk line. The same text, bound, and `chunker` always
give the same chunks.

### Vectors

A chunk line's `vector` is standard base64 (with padding) of the vector's float32s, each 4 bytes
little-endian, in order: `dimensions` × 4 bytes. It is the vector the endpoint returned, never
normalized. Whether the space is unit length is the manifest's `embedding_space.unit_length`,
within `embedding_space.tolerance`: a reader that wants unit vectors, for a dot product to be a
cosine, normalizes them itself when it is `false`.

## Held results

Each result a capability's request returned is held, a file of its own, as the request completes:
a later run over the same content and configuration finds it and sends no request, and a run that
stopped part way resumes from what it held. A file docinfer holds and can't decode is an error,
not a miss.

A result's `<key>` is the lowercase hex SHA-256 of four strings, each written as its length in
bytes, `:`, the string, and a newline: the capability, the model, the settings, and the lowercase
hex SHA-256 of the content the result is of. The settings are those that change the result:

| Capability | Content | Settings |
|---|---|---|
| `embeddings` | the chunk's text | `chunk_size=<n> chunker=<version>` |
| `vision` | the image file's bytes | `max_tokens=<n>`, a newline, `prompt=<prompt>` |
| `transcription` | the window file's bytes | `language=<language>`, a newline, `prompt=<prompt>`, each value a Go-quoted string |

So a changed model, prompt, cap, or language sends that capability's requests again, and a changed
`chunk_size` cuts and embeds every text again while sending no image and no window: vision and
transcription are held apart from the chunk settings. Endpoint, format, batch, and concurrency are
not part of any key.

An **embeddings** result, `results/embeddings/<xx>/<key>.json`:

| Field | Type | Meaning |
|---|---|---|
| `model` | string | The model. |
| `settings` | string | The settings, as the key has them. |
| `content` | string | The chunk text's SHA-256. |
| `vector` | string | The vector, encoded as a chunk line's is. Optional: absent when the endpoint rejected the chunk. |
| `dimensions` | integer | The vector's length. Optional: absent with the vector. |
| `error` | string | Why the endpoint rejected the chunk. Optional. |

A **vision** result, `results/vision/<xx>/<key>.json`:

| Field | Type | Meaning |
|---|---|---|
| `model` | string | The model. |
| `settings` | string | The settings, as the key has them. |
| `content` | string | The image's SHA-256. |
| `text` | string | The stored text, relative to the directory. Optional: absent when the endpoint refused the image. An empty text is a file too. |
| `text_sha256` | string | The stored text's SHA-256. Optional: absent with `text`. |
| `incomplete` | string | Why the model stopped before finishing, such as `"max_output_tokens"`. Optional. |
| `error` | string | Why the endpoint refused the image. Optional. |

A **transcription** result, `results/transcription/<xx>/<key>.json`, has `model`, `settings`,
`content` (the window's SHA-256), `text`, `text_sha256`, and `error`, as a vision result's; a
silent window's empty transcript is a file too. The joined transcripts are not held: each run joins
them again from the held windows, unchanged.

A refusal or rejection the endpoint gives with a 4xx reply is held, a verdict on the content: a
later run sends that content again only when its key changes. One it gives with any other status
is recorded in the run (counted, a failure or a rejected chunk line) but not held, so the next run
sends it again. A request that fails any other way, a transport error, a reply that can't be
decoded, or a cancelled run, stops the run, and nothing of it is held.

## Texts

`texts/<xx>/<sha256>.txt` holds each text a model made, whole, named by its SHA-256: each image's
vision text, each window's transcript, and each joined transcript. The same text is the same file.
A reader checks a text against the SHA-256 that names it (`text_sha256`, a transcript's `sha256`,
or a source's `sha256`); docinfer stops a run whose held result names a text that is gone or
changed.

## Joining back to docproc's run

Every chunk line names where its text came from in docproc's run directory, the manifest's
`docproc.dir`, at run `docproc.run`: its record is `<dir>/runs/<run>/record.jsonl`
([docs/docproc-schema.md](docproc-schema.md)).

- **An output's stored file** is `output.ref` joined to `docproc.dir`. A seen output's `ref` names
  the copy an earlier docproc run stored, which is still under `outputs/`.
- **Its output line** in docproc's record is the output line whose `"input"` is the chunk line's
  `input` and whose `"output"."ref"` is `output.ref`; its `"part"` is the one whose `sha256`,
  `where`, and `depth` are the chunk line's `part`'s. A `ref` alone isn't enough: a seen content's
  `ref` occurs under every input that holds it.
- **Its part line** is the part line of the same input and part; `part.outcome` is that line's
  outcome. For a metadata text, the part is the input's own, at depth 0; an input with no part line
  has an error line in the record instead.
- **A vision text** joins through `output`, its image's output line.
- **A transcript** joins through `output`, its first window's output line, and through the
  manifest's `transcripts` entry whose `input`, `part.sha256`, and `file` are the line's `input`,
  `part.sha256`, and `source.file`: its `windows` name every window's output, in plan order.

A docinfer run reads one docproc run. Two docinfer runs over different docproc runs in the same
directory share its held results, by content.

## What a reader checks

1. The manifest's `schema` is `"1"`.
2. The run ended whole: `end` is present, `error` is absent, and `chunks` names the chunk record.
   A run with an `error` has no chunk record; its held results are kept for the next run.
3. Before using vectors: `embedding_space.dimensions`, and `embedding_space.unit_length`. Every
   vector of a run has the space's dimensions; a chunk line with an `error` has no vector.
4. For docproc's outputs: the docproc run directory `docproc.dir` names is where the reader can
   read it, or the reader resolves each `ref` against where it now is.

## What's stable

Within schema `"1"`: the layout above, every field's name, type, and meaning, the optional fields'
absence rules, the path rules, the source kinds, the vector encoding, the held results' keys and
fields, and the stored texts' naming. `report.html`'s markup is not part of the contract.
