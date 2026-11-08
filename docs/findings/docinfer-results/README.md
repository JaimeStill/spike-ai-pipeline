# Results behind the docinfer finding

[`../docinfer.md`](../docinfer.md) cites these files. The runs themselves live on the laptop in
`~/docinfer-finding/`: `docproc/` (the docproc run), `docinfer-full/` (docinfer's directory),
`logs/`, `sample/`, and the run scripts.

## The docproc run

docproc over all of `~/docs`, run `01a1b82d-54aa-73a2-81db-67577e9cb5da`, 2026-11-07 22:42:20 to
23:11:01 UTC (1,720.674 s), built at `f77107a` (docproc SHA-256
`cd2f1da3b3a9bb21de06df4f8a73591c4721c40888de7686e85a5771232692c5` as rebuilt at the run's HEAD;
docproc's code is unchanged from `f77107a` to `14067a3`). It was not run again: docinfer reads it.

| File | Bytes | SHA-256 |
|---|---|---|
| `record.jsonl` | 9,583,988 | `307c478627198f4d87bfac0029a1ae9473176d146a8e2d1b7e494c66d5a4569b` |
| `manifest.json` | 5,040 | `a236e028fa4747c134fc74eac78e60778ed39bfb97f1758785a0e19fd1ab7865` |
| `report.html` | 4,687,737 | `fba322da1ed3f2610625ba1ff26e839ae0aa297cba9ebeab55be5470832ef082` |
| the run directory | 1,881,300,551 | |

## The docinfer runs

| Run | Id | Built at | docinfer SHA-256 | Wall |
|---|---|---|---|---|
| docinfer ~/docs 1 | `01a1b954-a1ac-7147-bcf2-0778352063ef` | `01d4b73` | `e715bb9e2cc1cae9bebf9d138e08671a17e35879c6bc939c983b128d3e0afc5d` | 21,634.388 s |
| docinfer ~/docs 2, the re-run | `01a1ba9f-443d-77f3-a010-1460884c935d` | `14067a3` | `0fabb3764ee8921304ab0ce1a25924702507f415851ac533387d5e84f34f68ec` | 3.113 s |

Both from the root, `F=~/docinfer-finding`:

```sh
"$F/bin/docinfer" run "$F/docproc" --config config/example --out "$F/docinfer-full"
```

through `run-docinfer-full.sh`, which times the run, tees its stdout and stderr, and saves the
router's log over it with `ssh gb10-2 docker logs --since <start> llama`. Each run's stderr was
empty. Before them, `mise run host-update` reported nothing changed, set A loaded, with gb10-2's
clone on branch `docinfer` for the preset.

| File | What it records |
|---|---|
| `run-1.txt`, `run-2.txt` | Each run's terminal output, stdout, verbatim. |
| `manifest-1.json`, `manifest-2.json` | Each run's `runs/<run id>/manifest.json`, verbatim. |
| `counts-1.md`, `counts-2.md` | `counts.sh` over each manifest and its chunk record. |
| `counts.sh` | The jq that derives the counts. |
| `du.txt` | `du -sb` and `du -sh` of docinfer's directory and its parts after the re-run, with the file counts. |
| `router-1-requests.tsv` | Run 1's router log cut to one line per request with its timing: the model, slot, task, prompt tokens and milliseconds, output tokens and milliseconds; 4,730 requests (embeddings log no generation timing, so none of theirs). |
| `textlayer.py` | The comparison of vision texts with docproc's text layer, page by page. |
| `textlayer-sample.json` | Its summary over all 2,234 paired pages, by input, and the seeded 30-page sample, each page's counts, its paths, and the words vision has that the layer lacks. |
| `prompt-compare-partway.txt` | The partway check's 6 requests, three described pages under the run's prompt and a stricter one. |
| `first-run.txt`, `first-run-vision-qc.txt` | The stopped first run's terminal and the quality check of its 2,086 held vision results. |
| `vortex-p20-reasoning-on.json`, `-none.json` | VORTEX's page 20 sent alone, reasoning on and off; each text field cut at 400 characters, its length noted. |
| `sample.md` | The 31-input sample, its runs, and its fixes. |

What isn't committed, for its size:

| File | Bytes | SHA-256 |
|---|---|---|
| run 1's `chunks.jsonl` | 193,306,945 | `44021f79a6d92f729bcc48e825ee9fccd7d1bf9dcc84ff2bdcf7c916e6fccca6` |
| run 2's `chunks.jsonl`, the final shape | 197,036,414 | `fe9ee71b988cf1b1e3dfebb7d0d65bde157e7bc4bd6380bf9c9df7b50c2dbb1a` |
| run 1's `report.html` | 8,098,762 | `9f292b9f516e2b1c076a688040a07d4548d6489bee749cdb1eff11ba1b01d252` |
| run 2's `report.html` | 8,098,749 | `50b09d47422fc78289ed9cd1dc4b5fabcfc5290fd580049e648cc7a2fc8ff141` |
| run 1's router log | 16,314,048 | `a7fdf212012d04c2d0664aa290bd3b8fbe933f2d0257cc6d4a3912625f2b683b` |
| `results/`, `texts/` | 101,750,297 in 27,889 files; 7,899,819 in 4,610 | |

The re-run's router log is empty: the router logged nothing over its window.
