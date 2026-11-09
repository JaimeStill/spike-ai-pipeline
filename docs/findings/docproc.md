# Finding: docproc over ~/docs, and its re-run

Task `docproc`, behavior 9 ([the goal record](https://github.com/JaimeStill/spike-ai-pipeline/blob/075b9a0bae37165340a534d162c7d1c5704d6320/context/goals/experiment.ai.spike-ai-pipeline.md), "Task brief ·
experiment.ai.spike-ai-pipeline · docproc"). Measured 2026-11-07 on the laptop, with no inference
and no gb10-2.

Every number here comes from two recorded runs of `docproc run ~/docs --out <run dir>`, both into
the same run directory: **docproc ~/docs 1**, into an empty one, and **docproc ~/docs 2**, the
re-run, straight after. The binary was built at the root's commit `5b1badf`. Their terminal logs,
manifests, counts, and disk use are committed in
[`docproc-results/`](docproc-results/README.md), which also names the host, the tools with their
versions and digests, the commands, and the SHA-256 of the records and outputs not committed.

## At a glance

| Question | Answer |
|---|---|
| What went in? | **All of ~/docs: 592 files, 1,740,329,401 bytes**, every one an input; no symbolic link. |
| Which stages ran? | **All ten wired, none missing, none excluded.** Every tool found and at or above processing's minimum (table below). |
| How long? | **1614.67 s (26 min 55 s) wall for run 1, at 22 workers**, of which one input, a 4.4 h audiobook, took 1613.87 s. |
| What came of the inputs? | **515 prepared, 17 seen, 51 no processor yet, 5 encrypted, 4 failed, 0 error.** 331 more parts were found inside them. |
| How much disk? | **1,881,238,985 bytes (1.8G) for the run directory**, 1,864,852,858 of them (1.8G) in `outputs/`, 8,805 files, one per output prepared. |
| Quarantines? | **None.** 63 part lines carry inspection flags, every flag class `active`, and none was quarantined; no `quarantine/` directory was made. |
| Does a re-run prepare nothing? | **No, not quite.** 545 inputs were seen, 41 came to `no processor yet` again, and 6 were dispatched again: the 4 that failed, failing again, and 2 holding a part that failed. Their 1,831 outputs went to the refs run 1 had stored; `outputs/` didn't change. **142.21 s wall.** |

## Results: run 1

**docproc ~/docs 1**, run `01a1b69e-a80b-7ece-871b-775878f2b217`, 15:26:53 to 15:53:47 UTC, 22
workers (the default, `runtime.NumCPU()`), exit 0, stderr empty.

### Tools and stages

| Tool | Version | Minimum |
|---|---|---|
| gs | 10.8.0 | 10.0 |
| qpdf | 12.4.2 | 11.0 |
| pdfdetach | 26.10.0 | 22.02 |
| magick | 7.1.2 | 7.1 |
| soffice (in `gotenberg/gotenberg:9.1.0-libreoffice`, by digest) | 26.8.2.2 | 24.2 |
| spike-ai-pipeline-office | 0.1.0 | 0.1.0 |
| spike-ai-pipeline-mail | 0.3.1 | 0.2.0 |
| spike-ai-pipeline-onenote | 0.1.0 | 0.1.0 |
| 7zz | 26.6 | 24.01 |
| ffmpeg, ffprobe | 9.0.2 | 9.0 |

| Route | Stage | State | Tools |
|---|---|---|---|
| document | document | wired | gs 10.8.0, qpdf 12.4.2, pdfdetach 26.10.0 |
| office | office | wired | spike-ai-pipeline-office 0.1.0, soffice 26.8.2.2, magick 7.1.2 |
| vision-converted | picture | wired | magick 7.1.2, soffice 26.8.2.2 |
| vision | vision | wired | none |
| nested | email | wired | spike-ai-pipeline-mail 0.3.1 |
| onenote | onenote | wired | spike-ai-pipeline-onenote 0.1.0 |
| archive | archive | wired | 7zz 26.6 |
| audio | audio | wired | ffmpeg 9.0.2, ffprobe 9.0.2 |
| video | video | wired | ffmpeg 9.0.2, ffprobe 9.0.2 |
| text | text | wired | none |

No stage was missing.

### Inputs by route and outcome

| Route | prepared | seen | no processor yet | encrypted | failed | Inputs |
|---|---|---|---|---|---|---|
| document | 59 | 2 | | 3 | 3 | 67 |
| office | 104 | 2 | 10 | 1 | | 117 |
| vision-converted | 20 | | | | | 20 |
| vision | 9 | 1 | | | | 10 |
| nested | 71 | 1 | | | | 72 |
| onenote | 25 | | | | 1 | 26 |
| archive | 8 | | | 1 | | 9 |
| audio | 41 | | | | | 41 |
| video | 9 | | | | | 9 |
| text | 169 | 11 | | | | 180 |
| unrouted | | | 41 | | | 41 |
| **total** | **515** | **17** | **51** | **5** | **4** | **592** |

No input's dispatch stopped with an error. The 17 seen inputs are content another input in ~/docs
shares, seen within the run, such as `earnings22.LICENSE.md` after `earnings21.LICENSE.md`. The
51 with no processor yet are 41 files with no route (40 with no signature processing knows, 1 a
shebang script), 8 ODF documents, an Excel binary workbook (`simple.poi.xlsb`), and a WordPerfect
document, each of the last 10 refused by the office stage as having no processor yet for its
format.

### Nested parts by route and outcome

331 parts were found inside the inputs, 324 at depth 1 and 7 at depth 2: 923 parts in all.

| Route | prepared | seen | no processor yet | failed | Parts |
|---|---|---|---|---|---|
| document | 10 | 9 | | 1 | 20 |
| office | 4 | 5 | | 135 | 144 |
| vision-converted | 2 | 3 | | | 5 |
| vision | 49 | 66 | | | 115 |
| nested | 2 | 2 | | | 4 |
| archive | 1 | 2 | | | 3 |
| audio | 2 | 2 | | | 4 |
| text | 7 | 7 | | | 14 |
| unrouted | | | 22 | | 22 |
| **total** | **77** | **96** | **22** | **136** | **331** |

### Outputs

11,085 output lines: 8,805 prepared, each stored once, and 2,280 seen, each naming a copy already
stored. By kind, prepared and seen: image 3,531 and 1,162; text 2,411 and 1,098; audio 1,257 and
2; record 885 and 8; markdown 721 and 10. The record's distinct refs, 8,805, equal the files in
`outputs/`.

### Failed and encrypted

| Input | Route | Outcome | Reason |
|---|---|---|---|
| `pdf/large/cfr-2024-title32-vol1.pdf` | document | failed, 1,823 outputs | qpdf: form field `"USGPOSignature"`: value `"3033 0 R"`: neither a string nor a name |
| `pdf/edge/malformed/no-document-catalog.pdf` | document | failed | gs can't count its pages ("Couldn't initialise file"), qpdf can't find its trailer dictionary, and pdfdetach can't read its page catalog |
| `pdf/edge/malformed/compacted-syntax.pdf` | document | failed | 1 of 1 pages failed: gs rendered no PNG |
| `onenote/interop/SUT/Encryption.one` | onenote | failed | onenote refused it: "Unexpected end of file" |
| `pdf/edge/encrypted/encryption-openpassword.pdf`, `simple-open-password.pdf`, `unicode-password-corrigendum5.pdf` | document | encrypted | the PDF needs a password |
| `office/macro-template/protected-passtika.poi.xlsb` | office | encrypted | office refused an OLE compound file, an encrypted package or a legacy binary format |
| `archive/password-protected.zip` | archive | encrypted | 2 of 2 entries encrypted |

Paths are under `/home/jaime/docs/`. Two inputs came to `prepared` with a failed part inside them:
`onenote/large/LargeData.one`, whose 135 copies of `testing - large.docx`, one content, were each
refused by office as "not a docx, xlsx, or pptx package", and
`pdf/edge/signed/digitally-signed-3d-portfolio.pdf`, whose embedded `Design of Landing Gear.pdf`
failed on its form, as the CFR volume did: qpdf found a field's value an indirect reference,
neither a string nor a name. `counts-1.md` lists each of these lines with its reason reduced to
one line, as docproc's terminal now prints it. `run-1.txt`, the terminal log from before that
reduction, and the record hold each reason whole, such as `no-document-catalog.pdf`'s qpdf and
pdfdetach errors.

### Inspection flags

63 part lines carry flags, all class `active`: `signature-mismatch` 19, `pdf-embeddedfile` 14,
`pdf-js` 13, `pdf-openaction` 10, `pdf-aa` 9, `pdf-encrypt` 5, `pdf-javascript` 5, `vba-project` 5,
`external-relationship` 3, and one each of `executable`, `macro-sheet`, `ole-object`,
`ooxml-encrypted`, `svg-event-handler`, `svg-external-href`, `svg-script`, and `zip-encrypted`.
Under docproc's policy, processing's default, none was quarantined: their parts came to prepared 36,
seen 17, encrypted 5, no processor yet 3, and failed 2.

### Time

The run's wall time, 1614.67 s around `mise exec`, is one input's: `librivox-stella-maberly.m4b`, a
126,864,276-byte audiobook of 15,787 s (4 h 23 min), took 1613.87 s for 646 outputs. Every other
input had ended by 15:32:25, 5 min 32 s in; the audiobook ran alone for the remaining 21 min 22 s.
While it ran, each window's ffmpeg was seen reading the whole file from its start (`-i … -af
aresample=16000,atrim=start_sample=…:end_sample=…`), so each window decodes the book up to its own
end.

The 10 slowest inputs:

| Time (s) | Route | Outcome | Outputs | Children | Input |
|---|---|---|---|---|---|
| 1613.87 | audio | prepared | 646 | 0 | `audio/long/librivox-stella-maberly.m4b` |
| 325.77 | onenote | prepared | 4 | 13 | `onenote/large/Large OneDrive.one` |
| 325.76 | onenote | prepared | 4 | 13 | `onenote/large/Large Desktop.one` |
| 244.72 | audio | prepared | 157 | 0 | `audio/long/earnings22-4479944.mp3` |
| 231.90 | document | failed | 1823 | 0 | `pdf/large/cfr-2024-title32-vol1.pdf` |
| 193.21 | document | prepared | 491 | 0 | `pdf/scanned/real/stella-maberly-1896-ia-scan.pdf` |
| 80.83 | audio | prepared | 97 | 0 | `audio/long/earnings21-4330115.mp3` |
| 59.35 | document | prepared | 222 | 0 | `pdf/born-digital/dodm-5200-01.pdf` |
| 34.37 | audio | prepared | 108 | 0 | `video/ami-es2002d/ami-es2002d-mix-headset.wav` |
| 28.96 | document | prepared | 2 | 5 | `pdf/edge/signed/digitally-signed-3d-portfolio.pdf` |

The two large OneNote sections hold the same PDFs, by SHA-256: Desktop's 13 parts were prepared
(its `text_heavy.pdf` alone to 1,684 outputs), and OneDrive's 13 all came to `seen`, its input
ending 12 ms after Desktop's, as it waited on the parts Desktop was preparing.

### Disk

| Path | Bytes | du -sh |
|---|---|---|
| the run directory | 1,881,238,985 | 1.8G |
| `outputs/` | 1,864,852,858 | 1.8G |
| `index/` | 2,170,928 | 9.1M |
| `runs/` | 14,215,199 | 14M |

`runs/` holds run 1's manifest (5,040 bytes), record (9,546,275), and report (4,663,884). The run
directory is 1.08 times ~/docs' bytes.

## Results: the re-run

> **Note, 2026-11-07:** this re-run was recorded before docproc copied a seen part's subtree into
> its record ([schema page](../docproc-schema.md#seen-content-and-replayed-subtrees)). Its record
> has no lines for the parts inside a seen input, so its nested counts, 141 part lines, reflect
> only the 6 inputs dispatched again. A re-run now records every seen part's subtree; the
> [addendum](#addendum-the-re-run-with-subtree-replay) measures one.

**docproc ~/docs 2**, run `01a1b6b7-7611-78a7-9713-78f04b7df7fc`, 15:53:59 to 15:56:20 UTC, the
same command over the same run directory, 22 workers, exit 0, stderr empty. **142.21 s wall.**

| Route | seen | no processor yet | failed | prepared | Inputs |
|---|---|---|---|---|---|
| document | 63 | | 3 | 1 | 67 |
| office | 117 | | | | 117 |
| vision-converted | 20 | | | | 20 |
| vision | 10 | | | | 10 |
| nested | 72 | | | | 72 |
| onenote | 24 | | 1 | 1 | 26 |
| archive | 9 | | | | 9 |
| audio | 41 | | | | 41 |
| video | 9 | | | | 9 |
| text | 180 | | | | 180 |
| unrouted | | 41 | | | 41 |
| **total** | **545** | **41** | **4** | **2** | **592** |

Every input whose subtree came to an outcome its content decides was seen, the encrypted ones and
the 10 office inputs with no processor yet among them. The rest weren't, as processing's index has
it (`dispatch` package doc): a part that failed, or holds a failed or limited part, isn't recorded,
so a later dispatch prepares it again, and a part whose route has no stage records only its flags.

- The 41 unrouted inputs came to `no processor yet` again.
- The 4 failed inputs were dispatched again and failed again, with the same reasons.
- `LargeData.one` and `digitally-signed-3d-portfolio.pdf` were prepared again: their failed parts,
  the 135 `testing - large.docx` and `Design of Landing Gear.pdf`, failed again, and their other
  parts were seen.

6,959 output lines: 5,128 seen and 1,831 prepared, all 1,831 from those inputs (the CFR volume's
1,823, `LargeData.one`'s 4, and the portfolio's 2 and its embedded PDF's 2). Every ref the re-run
wrote, 6,949 distinct, is one run 1 wrote, and every one names a file in `outputs/`. `outputs/` was
the same after the re-run, 1,864,852,858 bytes in 8,805 files: the outputs prepared again were
stored under the refs already there, which the store keeps.

The re-run's time is the CFR volume's, re-rendered and failing on its form again: 141.47 s. Next
came the portfolio at 6.04 s and `LargeData.one` at 4.08 s; every seen input ended within 0.2 s.

| Path | Bytes | du -sh |
|---|---|---|
| the run directory | 1,890,866,958 | 1.8G |
| `outputs/` | 1,864,852,858 | 1.8G |
| `index/` | 2,170,928 | 9.1M |
| `runs/` | 23,843,172 | 23M |

The run directory grew by 9,627,973 bytes, the re-run's manifest, record (6,057,525), and report
(3,565,408).

## Addendum: the re-run with subtree replay

A third run, recorded 2026-11-07 after the finding, shows the replay; it doesn't change the numbers
above. **docproc ~/docs 3**, run `01a1b6d1-df4a-7a7d-b7bb-69e784458ac9`, ran the same command over
the same run directory after runs 1 and 2, from 16:22:49 to 16:25:08 UTC (138.80 s by its
manifest), with a binary built from `10d8e7c`, the commit that added subtree replay. It exited 0
with stderr empty, and its inputs' outcomes match run 2's row for row. Its files aren't committed.

| Record | Nested part lines | Output lines | Distinct refs | Warnings | Bytes |
|---|---|---|---|---|---|
| run 1 | 331 | 11,085 | 8,805 | | 9,546,275 |
| run 2, before replay | 141 | 6,959 | 6,949 | | 6,057,525 |
| run 3, with replay | 348 | 11,116 | 8,805 | 0 | 9,931,665 |

Run 3's refs are run 1's 8,805, every one a file in `outputs/`, including `mixed-bundle.zip`'s 14,
which run 2 lacked. Of its nested part lines, 212 are seen and 136 failed, the failures those of
the inputs dispatched again. Its 17 nested part lines beyond run 1's are subtrees run 1 itself left
out: run 1, also recorded before replay, gave no lines to the parts inside content it saw twice in
one run, such as the bundle inside `zip-in-zip.zip` and `mixed-bundle.tar.zst`, and run 3 copies
them.

Setting aside outcomes, times, and the reasons on the inputs' own lines, run 3's record holds every
line of run 1's but 12. Those 12 are the lines of inputs dispatched again whose error names run 1's
working directory or whose part the index now sees, and the encrypted inputs' own lines, to which
the index's seen event gives no error.

## What the runs show

- docproc runs processing's ten stages end to end over a 592-file, 1.7 GB set on a laptop, with
  every tool present, no input stopped by an error, no stage missing, and every failure confined to
  its part.
- The wall time is the slowest input's, not the set's: the pool's other 21 workers were idle for
  the last 21 min 22 s of run 1, behind one audiobook whose windows each decode it from the start.
- Behavior 6's "every input is Seen" holds for 545 of the 592 inputs, not all of them: the other 47
  are the 41 with no route, which the index never records, and the 6 that failed or hold a part
  that failed. A re-run retries every failure, and a failure inside a large input costs that
  input's whole work again: the CFR volume's 1,823 page outputs are rendered on every run, because
  its form fails after them.

## Limits

- **One run of each**, on one machine, the laptop, x86_64.
- **The run directory was on tmpfs**, and ~/docs on btrfs, its files likely in the page cache for
  the re-run.
- **Times are wall clock**, around `mise exec`; no CPU time or peak memory was recorded (the host
  has no `/usr/bin/time`).
- **No output is checked against truth**: processing's findings score the stages; this records what
  docproc made of the set.
