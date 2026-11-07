# Results behind the docproc finding

[`../docproc.md`](../docproc.md) cites these files. They come from two runs of `docproc run` over
all of `~/docs`, both writing to the same run directory:

- **docproc ~/docs 1**, run `01a1b69e-a80b-7ece-871b-775878f2b217`: from 15:26:53 to 15:53:47 UTC
  on 2026-11-07, into an empty run directory.
- **docproc ~/docs 2**, run `01a1b6b7-7611-78a7-9713-78f04b7df7fc`: from 15:53:59 to 15:56:20 UTC
  on 2026-11-07, the re-run, over the run directory run 1 left.

Both used a binary built at the root's commit `5b1badf` (`5b1badf21dcfc8994999f63fd3097ac89518927d`)
on branch `docproc`, `docproc version` `v0.0.0-20261107152626-5b1badf21dcf`, SHA-256
`7846c2e3591bcb4de6212f5e32fc7035088283246e30082d401a4110d45f4950`. The slices after it change
documentation only.

The host is the laptop: Intel Core Ultra 9 185H, 22 logical CPUs, 30 GiB (33,058,963,456 bytes);
Linux 7.2.15 x86_64 (Omarchy); go 1.27.3; docker 29.9.2. `~/docs` is on btrfs; the run directory
was in the session's scratch directory on tmpfs (`/tmp`). The tools, as the manifests list them:

| Tool | Version | Minimum | Source |
|---|---|---|---|
| gs | 10.8.0 | 10.0 | host, `/usr/bin/gs` |
| qpdf | 12.4.2 | 11.0 | host, `/usr/bin/qpdf` |
| pdfdetach | 26.10.0 | 22.02 | host, `/usr/bin/pdfdetach` |
| magick | 7.1.2 | 7.1 | host, `/usr/bin/magick` |
| soffice | 26.8.2.2 | 24.2 | docker, `gotenberg/gotenberg:9.1.0-libreoffice@sha256:4ff340bb7b0753e7d037ceb75a08b09c3a9aebd0e6484df9516ec29ec31074ff`, image already present |
| spike-ai-pipeline-office | 0.1.0 | 0.1.0 | mise pin, release binary, SHA-256 `23c88e2724d291e2dd3d4ea37d0e8ec1f67b54c55397ae284b587ea89ddc8ee6` |
| spike-ai-pipeline-mail | 0.3.1 | 0.2.0 | mise pin, release binary, SHA-256 `a68a39309efe7af6edbacfc96d0530fe1944169b7424ad1d878d78641a46acef` |
| spike-ai-pipeline-onenote | 0.1.0 | 0.1.0 | mise pin, release binary, SHA-256 `5c1e8a25577fec4fbc982902c5b99c12bfe2e34f5d19a2264864c00fabc8c81b` |
| 7zz | 26.6 (26.06) | 24.01 | mise pin, release binary, SHA-256 `f94066e93f9a807547a3bff1021b3ec79d5ea4f57dfead018e9698182ad283b5` |
| ffmpeg | 9.0.2 | 9.0 | host, `/usr/bin/ffmpeg` |
| ffprobe | 9.0.2 | 9.0 | host, `/usr/bin/ffprobe` |

| File | What it records |
|---|---|
| `run-1.txt`, `run-2.txt` | Each run's terminal output, stdout, verbatim: the tools and stages tables, a line per input, and the summary. Each run's stderr was empty. They hold the raw error bytes the terminal printed before docproc reduced each error to one printable line: a failed input's error over several lines, and `compacted-syntax.pdf`'s Ghostscript PNG output, NUL bytes and all, which is why git treats them as binary. |
| `manifest-1.json`, `manifest-2.json` | Each run's `runs/<run id>/manifest.json`, verbatim. |
| `counts-1.md`, `counts-2.md` | `counts.sh` over each run's record: inputs and nested parts by route and outcome, parts by depth, output lines by kind, the 15 slowest inputs, the inspection flags, and every failed, encrypted, quarantined, or error line with its reason, reduced to one printable line as the terminal now prints it. They were regenerated from the same records, their SHA-256s below, when `counts.sh` took up that reduction; only the reason text changed, no count. |
| `counts.sh` | The jq that derives the counts from a record. |
| `du.txt` | `du -sb` and `du -sh` of the run directory and its `outputs/`, `index/`, and `runs/`, after each run. |

What isn't committed, for its size:

| File | Bytes | SHA-256 |
|---|---|---|
| run 1's `record.jsonl` | 9,546,275 | `cdd82c3ca1c47b017a1e426ff1430c9734025a36fe89c1335148a33c7a86c13f` |
| run 1's `report.html` | 4,663,884 | `d05bd88d03678248dd3d42ca6e8411569cad2d212a4d4b77bea1533c930a53bc` |
| run 2's `record.jsonl` | 6,057,525 | `7f38fe625f242868e1ab6ecfcd69855492d6af43ac72c06729ce25c7a74a5b20` |
| run 2's `report.html` | 3,565,408 | `1174cfd02bca169758b5eeba48dbf99cc4db4d9afc25e713f8b451d23fb74d16` |
| `outputs/`, after run 2 | 1,864,852,858, in 8,805 files | `c6b725e94486d0fcabc1ce335b466a3ac723f48a41752e5ac22f154c9cb0efc7`: the SHA-256 of `sha256sum` over every file, sorted by path, run from the run directory |

## Commands, as run

From this repository's root, `S` the session's scratch directory:

```sh
go build -o "$S/docproc" ./cmd/docproc

# run 1, then run 2, the same command over the same run directory
t0=$(date +%s.%N)
mise exec -- "$S/docproc" run ~/docs --out "$S/run" > "$S/run1.stdout" 2> "$S/run1.stderr"
t1=$(date +%s.%N)

du -sb "$S/run" "$S/run/outputs"   # and index/, runs/, each on its own
docs/findings/docproc-results/counts.sh "$S/run/runs/<run id>/record.jsonl" > counts-1.md
```

The wall times are `t1 - t0`, around `mise exec`, so they include mise's start. Workers were the
default, 22, `runtime.NumCPU()`.
