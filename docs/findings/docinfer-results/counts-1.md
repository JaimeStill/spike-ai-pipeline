### The run

Run `01a1b954-a1ac-7147-bcf2-0778352063ef`, 2026-11-08T04:04:53.803677473Z to 2026-11-08T10:05:27.443967234Z, over docproc run `01a1b82d-54aa-73a2-81db-67577e9cb5da` in `/home/jaime/docinfer-finding/docproc`.

| Capability | Endpoint | Model | Settings |
|---|---|---|---|
| vision | gb10-2 | `google/gemma-4-26B-A4B-it-qat-q4_0-gguf:Q4_0` | concurrency 4, max_tokens 2048, prompt "Transcribe the text in this picture. If it has no text, describe it in one sentence.", reasoning "none" |
| transcription | gb10-2 | `ggml-org/gemma-4-E4B-it-GGUF:Q8_0` | concurrency 4, language "", prompt "Transcribe the speech in this audio verbatim, in its own language. Write only the words spoken. If there is no speech, write nothing." |
| embeddings | gb10-2 | `mrutkows/granite-embedding-english-r2-GGUF:BF16` | batch 16, chunk_size 2000, concurrency 4 |

### Requests, failures, and tokens by capability

Failures are the distinct images refused, windows refused, or chunks rejected, this run or held.

| Capability | Requests | Failures | Input tokens | Output tokens | Total tokens |
|---|---|---|---|---|---|
| vision | 3470 | 0 | 2879058 | 2299731 | 5178789 |
| transcription | 1254 | 0 | 881847 | 97762 | 979609 |
| embeddings | 1448 | 0 | 12133426 | 0 | 12133426 |

### What each capability did

- docproc output lines: 11116; image lines 4701, audio window lines 1259.
- vision: distinct images 3470: held 0, described 3470, incomplete 272 (empty 0), refused 0; texts embedded 4701.
- transcription: distinct windows 1254: held 0, transcribed 1254, failed 0; transcripts 51, seams matched 1075, unmatched 133.
- embeddings: text and Markdown outputs 4259, metadata texts 592; distinct chunks 23165: held 0, embedded 23165, rejected 0; chunk lines 28937.

### The embedding space

| Endpoint | Model | Dimensions | Vectors | Unit length | Tolerance | Min norm | Max norm | Chunk size | Chunker |
|---|---|---|---|---|---|---|---|---|---|
| gb10-2 | `mrutkows/granite-embedding-english-r2-GGUF:BF16` | 768 | 23165 | true | 0.001 | 0.9999999251977663 | 1.0000000794956279 | 2000 | 1 |

### Texts by kind

One text per output line, docproc's text and Markdown outputs from its record; one vision text per image line with a text; one metadata text per input; one transcript per part and stage with windows.

| Kind | Texts |
|---|---|
| markdown | 737 |
| text | 3522 |
| metadata | 592 |
| vision | 4701 |
| transcript | 51 |

### Chunks by source kind

Lines are the chunk record's, one per chunk per occurrence; distinct chunks are by text, within the kind.

| Kind | Lines | Distinct chunks | Rejected lines |
|---|---|---|---|
| metadata | 592 | 592 | 0 |
| output | 18485 | 15446 | 0 |
| transcript | 265 | 261 | 0 |
| vision | 9595 | 6890 | 0 |
| total | 28937 | 23165 | 0 |

### Chunk lengths, in runes, over the distinct chunks of each kind

| Kind | Chunks | Min | Median | p95 | Max |
|---|---|---|---|---|---|
| metadata | 592 | 276 | 327 | 360 | 418 |
| output | 15446 | 1 | 1934 | 1992 | 2000 |
| transcript | 261 | 16 | 1999 | 2000 | 2000 |
| vision | 6890 | 1 | 1085 | 1967 | 2000 |
| all | 23165 | 1 | 1884 | 1991 | 2000 |

### Failures by capability and error

Each is an output line whose image or window the endpoint refused; the error is cut at 200 characters.

| Capability | Lines | Inputs | Error |
|---|---|---|---|

