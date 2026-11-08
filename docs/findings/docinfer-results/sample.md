# docinfer over a sample of ~/docs

The first full run over ~/docs stopped after 4 h at 2,086 of about 3,500 distinct images: 29% of the
vision texts were incomplete at the 2048-token cap and 22% empty, because Gemma 4 26B-A4B reasoned
over dense pages until the cap ran out. Before running the full set again, a sample of 31 inputs was
run through docproc and docinfer, quality-checked, fixed, and run again. The runs live in
`~/docinfer-finding/sample/`: `inputs/` (copies, not links), `docproc/`, `docinfer/` (run 1),
`docinfer-2/` (run 2, the fixes), `docinfer-3/` (run 3, a slot trial), and `logs/` (each run's
terminal, time, and the router's log over it). `counts.sh` here renders a manifest's tables.

## The selection

1–2 inputs per docproc route, chosen from the docproc finding's per-input table and the first
run's record, 59,030,009 bytes in all. Paths are under `~/docs/`.

| Input | Bytes | Route | Why |
|---|---|---|---|
| `pdf/born-digital/vortex-150-mini-manual.pdf` | 493,024 | document | The dense-page manual whose page 20 came back empty with reasoning on; 17 page images |
| `pdf/multilang/eurlex-32011D0833/decision-2011-833-en.pdf` | 745,752 | document | A text-layer PDF of a few dense pages (4) |
| `pdf/multilang/multilang-rtl.pdf` | 145,215 | document | Arabic, Hebrew, Japanese, and Russian on the page |
| `pdf/scanned/clean.pdf` | 41,290 | document | A clean scan, 6 pages |
| `pdf/scanned/degraded.pdf` | 1,068,177 | document | The same pages degraded, 9 pages |
| `pdf/scanned/olmocr-bench/old_scans-17.pdf` | 261,570 | document | A 1914 typewritten letter, one page |
| `office/realworld/docx/word-with-attachments.docx` | 144,959 | office | A docx with 5 embedded parts, Cyrillic |
| `office/realworld/xlsx/simple-monthly-budget.xlsx` | 18,700 | office | A spreadsheet |
| `office/realworld/pptx/shapes.pptx` | 68,822 | office | A deck with a picture |
| `office/legacy/generated/media.libreoffice.doc` | 24,576 | office | A legacy .doc with 4 images |
| `office/legacy/real/basic_test_ppt_file.ppt` | 15,360 | office | A legacy .ppt |
| `office/odf/writer-odf13.odt` | 9,683 | office | No processor yet: metadata only |
| `email/generated/eml/03-html-inline-image.eml` | 1,164 | nested | An .eml with an inline image |
| `email/msg/tika/test-outlook2003.msg` | 83,968 | nested | A .msg with 11 icons |
| `email/generated/pst/mailbox.pst` | 271,360 | nested | A small .pst, CJK bodies |
| `onenote/onenote-rs/New Section 1.one` | 264,421 | onenote | A OneNote section |
| `onenote/onenote-rs/handwriting_recognition.one` | 180,020 | onenote | Handwriting |
| `archive/mixed-bundle.zip` | 43,464 | archive | An archive of mixed parts |
| `text/markdown/swarm-capstone.md` | 9,904 | text | Markdown |
| `text/encodings/koi8-r.txt`, `shift-jis.txt` | 235, 203 | text | Legacy encodings |
| `image/graphics/speech.png` | 64,819 | vision | A small, tall image of dense text |
| `image/photo/house-chair-bicycle.jpg` | 2,266,870 | vision | A photo with no text |
| `image/formats/tiff/scanned-3page-g4.tif` | 13,748 | vision-converted | A 3-page fax TIFF |
| `image/vector/pipeline-diagram.svg` | 1,076 | vision-converted | A diagram |
| `video/prelinger-duck-and-cover/duck-and-cover-theora.ogv` | 39,412,437 | audio | 9 min 15 s of narration, 23 windows |
| `audio/short/librispeech/librispeech-dev-clean-2277-149896-0000.flac` | 127,816 | audio | English speech |
| `audio/short/fleurs-hi/fleurs-hi_in-1601.wav` | 656,698 | audio | Hindi |
| `audio/short/fleurs-ja/fleurs-ja_jp-1607.wav` | 510,778 | audio | Japanese |
| `video/tears-of-steel/tos-clip-180s-480p.mp4` | 12,082,828 | video | 3 min: 35 frames, 7 windows, music and noise |
| `office/realworld/licenses/docling.LICENSE` | 1,072 | unrouted | No route: metadata only |

docproc ran over it in 6 s: 240 output lines, 111 image lines (109 distinct), 33 audio windows.

## The runs

| Run | Commit | Change | Wall | Vision | Incomplete | Empty |
|---|---|---|---|---|---|---|
| 1 | 81c78de | vision reasoning none | 268.7 s | 240 s | 1 | 0 |
| 2 | 2bf153a | image-min-tokens 560; transcription prompt | 300.9 s | 279 s | 0 | 0 |
| 3 | 0076d4b | trial: vision at 8 slots | 278.0 s | 260 s | 0 | 0 |

Vision time is the vision model's span in the router log. Run 2 is the configuration kept; 3 was
reverted (f4df451).

Run 2's counts: vision 109 requests, 84,697 input and 17,032 output tokens; transcription 33
requests, 21,942 and 2,190; embeddings 16 requests, 46,841 tokens. No failures. Transcription
took 20.8 s (0.63 s a window), embeddings 0.8 s. Chunks, distinct: metadata 31, output 93,
transcript 9, vision 114, 247 in all; vision median 174 runes, transcript 783, output 1,072. The
short chunks are whole small texts (a page whose text layer is only its number, "6"; an icon's
one-line description), not chunking debris.

## What the check found

- **Reasoning.** With `reasoning` none, no vision text came back empty, and VORTEX page 20,
  empty before, transcribed in full (1,165 characters, finish stop).
- **Too few image tokens.** speech.png, 210x746, got about 100 image tokens; the model wrote a
  paraphrase ("What's the virtue in so much vainglory?" for "What's he that wishes so?"), then
  repeated "That this day is called the feast of Crispian;" 149 times to the cap. Upscaled 2x
  (about 300 tokens) it read the speech word for word. gb10-2's preset now floors an image at
  560 tokens (`image-min-tokens`); in run 2 speech.png transcribes in full, and a video frame's
  book spines read "...DER NEDERLANDER IN DE TWEEDE WERELDOORLOG" where run 1 invented
  "MIDDELSCHLAND". The median vision prompt rose from 274 to 627 tokens.
- **Transcription remarks.** gemma-4-E4B answered two Tears of Steel windows of music and noise
  with "The audio consists only of instrumental music and does not contain discernible speech."
  and the like, embedded as transcript. `config/example` now sends a prompt asking for only the
  words spoken; in run 2 both windows are empty and the transcript reads clean.
- **Faithful pages.** VORTEX pages, the EUR-Lex pages, the clean and degraded scans, and the
  1914 letter transcribe faithfully, tables as Markdown; the RTL PDF's Arabic, Hebrew, Japanese,
  and Russian come through, under a one-line preface ("This image contains text in English,
  Arabic, and Hebrew..."). Diagram labels (VORTEX's stick-mode crosses) come out muddled.
- **Descriptions.** Photos, icons, and video frames get a one-sentence description; solid-colour
  images say so ("This is a solid red image."). The redacted fax page got a description in run 2
  where run 1 transcribed its header and footer: sampling at temperature 1 varies.
- **Transcripts.** duck-and-cover's 23 windows join at 22 matched seams into a faithful
  narration; English, Hindi (Devanagari) are clean; the Japanese clip's first word differs from
  the truth each run (興味を is heard as 蝶のように, 今日は...), as it did at temperature 0, so the
  card's sampling stays. Tears of Steel's 4 unmatched seams are around the empty and music
  windows.

## gb10-2

At 4 slots each vision slot generates about 20 tokens/s (56 alone), about 80 tokens/s together.
At 8 slots each fell to 10: the model is saturated at 4, so the preset keeps 4. The router's log
over run 2 shows no request truncated (389 releases, `truncated = 0`) and no deferral beyond the
slots docinfer's concurrency fills. Transcription
generates about 31 tokens/s a slot and spends no tokens reasoning.

## The full run, estimated

3,470 distinct images, 1,257 windows. The sample's rate, 2.56 s an image, gives 2.5 h, but the
full set's pages are denser: CFR's 915 pages run about 840 output tokens each (15 s alone, word
recall 0.91 against the text layer), text_heavy.pdf's 848 about 500. Estimating output tokens
by kind, about 1.65M at about 75 tokens/s together: vision about 6 h, transcription about 15 min,
embeddings a few minutes; **about 6.5 h** in all.
