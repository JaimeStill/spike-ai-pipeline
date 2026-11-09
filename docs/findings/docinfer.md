# Finding: docinfer over ~/docs through set A on gb10-2, and its re-run

Task `docinfer`, behavior 12 ([the goal record](https://github.com/JaimeStill/spike-ai-pipeline/blob/075b9a0bae37165340a534d162c7d1c5704d6320/context/goals/experiment.ai.spike-ai-pipeline.md), "Task brief ·
experiment.ai.spike-ai-pipeline · docinfer"). Measured 2026-11-07 and 2026-11-08, from the laptop,
against gb10-2's llama.cpp router serving set A.

Every number here comes from recorded runs of `docinfer run <docproc dir> --config config/example
--out <dir>` over one docproc run of all of ~/docs: **docinfer ~/docs 1**, into an empty
directory, and **docinfer ~/docs 2**, the re-run, straight after. Before them, a first full run
with the model reasoning was stopped, and a 31-input sample was run three times to fix what it
showed ([the sample](docinfer-results/sample.md)). The logs, manifests, counts, and the
comparison with the text layer are committed in [`docinfer-results/`](docinfer-results/README.md).

## At a glance

| Question | Answer |
|---|---|
| What went in? | **docproc run `01a1b82d-54aa-73a2-81db-67577e9cb5da` over all of ~/docs**: 11,116 output lines, 4,701 image lines (3,470 distinct images), 1,259 audio window lines (1,254 distinct), 4,259 text and Markdown outputs, 592 inputs. |
| Did every request succeed? | **Yes. 0 failures**: 3,470 vision, 1,254 transcription, and 1,448 embeddings requests, none refused, none rejected. One attempt, no transport error. |
| How long? | **21,634 s (6 h 0 min 34 s) wall.** Vision 5.67 h, transcription 15.5 min, embeddings 5.1 min, one after another. The sample's estimate was 6.5 h. |
| How many tokens? | **Vision 2,879,058 in, 2,299,731 out; transcription 881,847 in, 97,762 out; embeddings 12,133,426.** |
| Did vision answer every image? | **Yes, with text.** 0 empty; 272 incomplete at the 2,048-token cap (7.8%), 262 of them synthetic pages of random letters (below). The stopped first run, reasoning on, had 454 empty of 2,086. |
| What came out? | **28,937 chunk lines, 23,165 distinct chunks**, each a 768-dimension unit vector; 51 transcripts, 1,075 seams matched and 133 unmatched. |
| How much disk? | **516,842,726 bytes for docinfer's directory**: 101,750,297 of held results in 27,889 files, 7,899,819 of texts in 4,610 files, 407,192,610 of the two runs' records. |
| Does a re-run send nothing? | **Yes: no inference request.** Vision, transcription, and embeddings sent 0 requests, everything held; only behavior 2's `GET /models` listing went out. **3.113 s wall**, the same 28,937 lines with the same texts, vectors, and sources; only each line's part is fuller. |
| What does vision add over a text layer? | **On born-digital pages, little: median word recall 0.91 (CFR) and 1.00 (DoDM) against docproc's text output.** Its gain is the pages whose layer lacks the text: outside the synthetic `text_heavy.pdf`, 49 pages with a thin layer (layer 1,636 words, vision 18,442); and 61 with none (8,021 words). Its loss: outside `text_heavy.pdf`, 24 of 1,289 pages with at least 100 layer words (1.9%) came back described instead of transcribed. |

## The stack

Each measurement here was made through the same stack:

| Part | What |
|---|---|
| Router | llama.cpp `ghcr.io/ggml-org/llama.cpp:server-cuda13-b12098`, version 0.8.0-dev (build 12098, commit 149f5641b), the preset in `host/llama-preset/models.ini` at this branch (vision: 4 slots, c 32768, ub 1120, `image-min-tokens` 560) |
| Host | gb10-2, NVIDIA GB10: Ubuntu 24.04.5 LTS, DGX OS 7.6.0, kernel 7.0.0-1025-nvidia, driver 580.178.04, CUDA 13.0, docker 29.6.2, NVIDIA Container Toolkit 1.20.1; set A loaded (gpt-oss-120b too, idle) |
| Network | the laptop to gb10-2 over Tailscale, direct, under 1 ms (`tailscale ping`) |
| docinfer | built at `01d4b73` for run 1 (SHA-256 `e715bb9e…`), at `14067a3` for the re-run, whose chunk line carries docproc's part whole (`b044ad8`): each line's part gains `size` and `content_type`, and `parent` and `flags` where docproc's part has them; no held result's key changes |
| Configuration | `config/example`: vision `reasoning` none, `max_tokens` 2048, concurrency 4; transcription with the example's prompt; embeddings batch 16, chunk size 2,000 runes |

## Results: run 1

**docinfer ~/docs 1**, run `01a1b954-a1ac-7147-bcf2-0778352063ef`, 04:04:53 to 10:05:29 UTC on
2026-11-08, exit 0, stderr empty, one attempt.

### Requests, failures, time, and tokens

| Capability | Requests | Failures | Input tokens | Output tokens | Time |
|---|---|---|---|---|---|
| vision | 3,470 | 0 | 2,879,058 | 2,299,731 | 20,395 s (5.67 h) |
| transcription | 1,254 | 0 | 881,847 | 97,762 | 930 s |
| embeddings | 1,448 | 0 | 12,133,426 | 0 | 306 s |

Time is each model's span in the router's log, its first request to its last. Vision's span
includes 6 requests sent beside the run during the partway check (below); the router counted
3,476 vision tasks. Every one of the router's 27,895 releases says `truncated = 0`.

- **Vision**: a median prompt of 952 tokens (a page is about 950, the 560 floor lifts small
  images); generation a median 32.59 tokens/s a request, 4 slots busy, 30.76 pooled. 4,701 image
  lines, 3,470 distinct: each image sent once.
- **Transcription**: a median prompt of 733 tokens a window; generation a median 30.62 tokens/s
  a request, 30.56 pooled. 51 transcripts, by part and stage; 1,075 seams matched, 133
  unmatched.

A request's rate is its output tokens over its output milliseconds, from
[`router-1-requests.tsv`](docinfer-results/router-1-requests.tsv) (the router's `eval time`
lines give the same medians); the median is over requests, vision's 3,476 including the 6 sent
beside the run, and the pooled rate is all output tokens over all output milliseconds.
- **Embeddings**: 23,165 distinct chunks in 1,448 requests of up to 16.

### Vision's texts

0 empty, 0 refused. The median text is 1,419 characters, the 95th percentile 4,290.

272 were cut at the 2,048-token cap:

- **262 are synthetic random-letter pages** inside `onenote/large/Large Desktop.one`: 261 of
  `text_heavy.pdf`, an 848-page PDF of random letter strings, about 3,700 characters a page, and
  1 of `mixed.pdf`, which holds the same kind of text. Nonsense words take several tokens each,
  so a full page doesn't fit 2,048. Their text is kept to the cap.
- **5 CFR pages and 5 others**: dense tables (a Gantt chart in
  `digitally-signed-3d-portfolio.pdf`, a microbiota tables page, a crop report) and two 1880s
  scans.

38 of the 272 are runaway loops, a line or phrase repeated until the cap: 32 on the synthetic pages,
3 CFR pages, the Gantt chart, and `old_scans-70` and `-71` (one pads with runs of spaces, one
repeats "meant to be hard work, privation, servitude,").

### Texts and chunks by kind

| Kind | Texts | Chunk lines | Distinct chunks | Median runes | p95 |
|---|---|---|---|---|---|
| output (text, Markdown) | 3,522 text, 737 Markdown | 18,485 | 15,446 | 1,934 | 1,992 |
| vision | 4,701 | 9,595 | 6,890 | 1,085 | 1,967 |
| transcript | 51 | 265 | 261 | 1,999 | 2,000 |
| metadata | 592 | 592 | 592 | 327 | 360 |
| **total** | | **28,937** | **23,165** | 1,884 | 1,991 |

A vision text is one per image line; the same image under two lines is two texts, one request.
No chunk was rejected.

### The vectors

| Model | Dimensions | Vectors | Min norm | Max norm |
|---|---|---|---|---|
| `mrutkows/granite-embedding-english-r2-GGUF:BF16` | 768 | 23,165 | 0.9999999252 | 1.0000000795 |

Every vector is unit length within the manifest's tolerance, 0.001.

### Disk

| Path | Bytes | du -sh | Files |
|---|---|---|---|
| docinfer's directory, after both runs | 516,842,726 | 610M | |
| `results/` | 101,750,297 | 200M | 27,889 |
| `texts/` | 7,899,819 | 22M | 4,610 |
| `runs/` | 407,192,610 | 389M | the two runs' chunk records, manifests, and reports |
| the docproc run directory, read only | 1,881,300,551 | 1.8G | |

`du -sh` counts blocks, so its small files round up.

## Results: the re-run

**docinfer ~/docs 2**, run `01a1ba9f-443d-77f3-a010-1460884c935d`, 10:06:02 to 10:06:06 UTC, the
same command and directory with docinfer rebuilt at `14067a3`, exit 0, stderr empty. **3.113 s
wall.**

**No inference request**: vision 0 (3,470 held), transcription 0 (1,254 held), embeddings 0
(23,165 held). The only request was behavior 2's `GET /models` listing, which the run checks
before it starts. The router's log over the re-run's window has no line at all.

It wrote 28,937 chunk lines. Without their part, they are run 1's line for line: `jq -cS
'del(.part)'` over each run's chunk record gives the same SHA-256,
`31a3585f5df8ff43879c2c9c91b30fb2b9c21c8214a4a9899b5e2bb17df9a767`, so the texts, vectors, and
sources are unchanged. The part is fuller: the spec review's change `b044ad8`, in the `14067a3`
build, carries docproc's part whole, so every line's part gains `size` and `content_type`, 9,333
lines gain `parent`, and 1,108 gain `flags`. Its chunk record is 197,036,414 bytes, SHA-256
`fe9ee71b988cf1b1e3dfebb7d0d65bde157e7bc4bd6380bf9c9df7b50c2dbb1a`.

## The first full run, with reasoning, stopped

The first full run, 2026-11-07 from 23:11 UTC over the same docproc run, sent each image without
a reasoning effort. It was stopped after 4 h. The session stopped the background shell it ran
in, then signalled the docinfer process; docinfer stopped vision and wrote its manifest at
03:24:22 UTC, its error `vision: terminated signal received`, 2,099 vision requests, 2,094
images answered ([`first-run-manifest.json`](docinfer-results/first-run-manifest.json)). The
`[exited with code 0]` that ends [`first-run.txt`](docinfer-results/first-run.txt) is the
background shell's report on its wrapper shell, not docinfer's exit status, which the run
script never recorded: its time log has a start line and no end. The quality check, of 2,086
held images taken just before the stop, found:

- **596 incomplete (29%) at the 2,048-token cap, 454 empty (22%)**, every empty one incomplete.
  The median text was 91 characters. ([`first-run-vision-qc.txt`](docinfer-results/first-run-vision-qc.txt))
- Gemma 4 26B-A4B reasons before it answers, and on a dense page its reasoning spent the cap.
  VORTEX's page 20 (`pdf/born-digital/vortex-150-mini-manual.pdf`), sent alone: reasoning on,
  2,048 completion tokens, finish `length`, no content and 7,575 characters of reasoning;
  `reasoning_effort` none, 333 tokens, finish `stop`, the page whole in 1,168 characters
  ([on](docinfer-results/vortex-p20-reasoning-on.json),
  [none](docinfer-results/vortex-p20-reasoning-none.json), cut at 400 characters a field).

So docinfer's vision now sends `reasoning`, `none` by default as processing's vision tier does,
inside the held results' key.

## The sample

31 inputs, one or two a route, run three times before the full run:
[`docinfer-results/sample.md`](docinfer-results/sample.md). It found and fixed two problems. A
small image of dense text got about 100 image tokens, and the model invented a paraphrase, then
looped; gb10-2's preset now floors an image at 560 tokens. gemma-4-E4B also answered windows of
music or noise with a remark that would be embedded; `config/example` now asks for only the words
spoken. A trial of 8 vision slots was 7% faster than 4 and was dropped.

## What vision adds over a text layer

For each document page with both a page image and a text output, paired by part and page, the
vision text's words (runs of letters or digits, lowercased, as multisets) were compared with the
text layer's: **recall**, the share of the layer's words the vision text has, and **precision**,
the share of the vision text's words the layer has. 2,234 pages paired
([`textlayer.py`](docinfer-results/textlayer.py),
[`textlayer-sample.json`](docinfer-results/textlayer-sample.json)).

| Input | Pages | Recall, median | p10 | Pooled recall | Pooled precision |
|---|---|---|---|---|---|
| `cfr-2024-title32-vol1.pdf` | 908 | 0.909 | 0.876 | 0.896 | 0.923 |
| `dodm-5200-01.pdf` | 111 | 1.000 | 0.976 | 0.978 | 0.932 |
| `stella-maberly-1896-ia-scan.pdf`, an OCR layer | 240 | 0.970 | 0.935 | 0.965 | 0.974 |
| `text_heavy.pdf`, synthetic | 836 | 0.565 | 0.410 | 0.537 | 0.559 |
| all | 2,234 | 0.897 | 0.502 | 0.774 | 0.796 |

A seeded sample of 30, 10 pages each of the CFR volume, DoDM 5200.01, and `text_heavy.pdf`, was
read in full: CFR median recall 0.92 (pooled 0.82, one page described), DoDM 1.00 (pooled 0.99),
`text_heavy.pdf` 0.73 (pooled 0.59).

- **Where a layer is whole, vision repeats it.** CFR's misses are mostly hyphenation and words the
  layer splits across its two columns, which vision joins. DoDM's are a table of contents
  vision padded with `&nbsp;` (364 on page 4). `text_heavy.pdf`'s low recall is random strings
  misread and pages cut at the cap; it measures nothing a real document has.
- **Where a layer is thin, vision is the text.** Leaving out `text_heavy.pdf`, 49 pages have a
  vision text at least twice the layer's words and at least 60 more: 43 CFR appendix pages,
  scanned forms and tables set into the born-digital volume, whose layer holds only the running
  head (11 to 35 words), 5 DoDM pages, and one page of `mixed.pdf`, the same synthetic
  random-letter text as `text_heavy.pdf`. Their layers hold 1,636 words; their vision texts
  18,442. `textlayer.py` derives these counts and writes its rule beside them, in the summary's
  `thin_layer`. CFR's page 77 is the
  SF-LLL form's instructions, 11 layer words and 1,343 transcribed.
- **Where there is no layer, vision is all there is.** 61 page images have no text output: scans
  such as `old_scans-*.pdf`, `degraded.pdf`, and 9 of the 1896 scan's pages, and 12 of
  `text_heavy.pdf`'s; 8,021 words of vision text, 7,865 without `text_heavy.pdf` (the summary's
  `no_layer`).
- **Figures.** A chart's axes and values, such as DoDM's page 56, come out as a description, words
  the layer lacks.
- **Where vision loses.** Leaving out `text_heavy.pdf`, 24 of the 1,289 pages with at least 100
  layer words (1.9%) came back described, "This image is a page from a legal document…", not
  transcribed: 18 CFR pages, 2 DoDM pages, two small portfolio tables, a page of ROC curves, and
  one page of the synthetic `mixed.pdf`. A page counts as described when its recall is under 0.5
  and its vision text opens by describing the image ("This image…", "The image…", "A … shows…");
  the rule and the list are the summary's `described`. Whether a page is described varies
  between sends, at temperature 1: sent again during the partway check
  ([`prompt-compare-partway.txt`](docinfer-results/prompt-compare-partway.txt)), CFR's page 21
  transcribed in full under the run's prompt (the second pair), and page 36, the third pair, was
  described again under the run's prompt and transcribed only under a stricter one ("Transcribe
  all the text in this picture, word for word, in reading order. Do not summarize or describe
  text…"). Two small
  embedded table images in the portfolio's `Landing Gear Project data.pdf` and `Technical
  Specification.pdf` are too low in resolution to read under either prompt.

For the capstone's decision to skip vision on a page with a text layer: on a page whose layer
holds its text, vision adds little beyond figure descriptions and may lose a page to a
description; the pages it rescues are those whose layer is thin or absent, which a skip rule
would have to tell apart from a page whose layer is whole.

## Transcripts

Every window transcribed, 0 failed. A search of the 51 transcripts for remarks about the audio
("the audio", "no speech", "discernible", and the like) found one match, and it is speech: "the
audio video of my television set", in an AMI meeting. Two short clips mix scripts:
`fleurs-ru_ru-1554` has "Wi-Fi", which the speaker says, and `fleurs-cmn_hans_cn-1605` has a stray
"quei" where the truth has 巨龟; the transcript is in traditional characters, the truth in
simplified.

## The partway check

At 40 minutes, 1,023 images held: 0 empty, 5 incomplete, 0 refused, no loop but the two old
scans. The text-layer comparison over the pages held then found the described pages above; 6
requests sent beside the run compared the prompts on three of them
([`prompt-compare-partway.txt`](docinfer-results/prompt-compare-partway.txt)). The run went on unchanged: the configuration was the
one approved, and changing the prompt would have sent every image again.

## Limits and caveats

- **One run**, from one laptop, against one host; times are wall clock, and the router's spans.
- **Messages ignores disabled thinking.** llama.cpp b12098's `/v1/messages` ignores
  `thinking: {"type":"disabled"}` (and `reasoning_effort`), so a vision endpoint speaking
  `messages` reasons anyway; only `chat_template_kwargs.enable_thinking` false stops it there.
  `config/example` speaks `completions`, which honors none, as `responses` does.
- **Sampling at temperature 1**, the model card's, makes vision's transcribe-or-describe choice and
  the Japanese clip's first word vary between runs.
- **The preset is outside the held results' key**: a change to gb10-2's serving, such as the image
  token floor, needs a fresh directory to take effect on held images.
- **No text is scored against truth**; the text layer is the reference above, and it is itself
  imperfect on scans.
