# Finding: set A on gb10-2, responses and transcription timestamps

Task `host`, behaviors 9 to 11 (`context/goals/experiment.ai.spike-ai-pipeline.md`, "Task brief ·
host"). Measured 2026-10-26, from the laptop, against gb10-2's llama.cpp router serving set A.

## The two results

| Question | Result |
|---|---|
| Does `/v1/responses` answer on gpt-oss-120b? | **Yes.** HTTP 200, `status` `completed`, output items `reasoning` then `message`, the message's `output_text` holding the reply. |
| Does transcription return segment or word timestamps? | **No.** `/v1/audio/transcriptions` accepts only `response_format` `json`, which carries `text` and `usage` and no timestamps. `verbose_json` is refused with HTTP 400; `timestamp_granularities[]` is ignored under `json`. |

The probe records both and asserts neither (`probe/router_test.go`, subtests `responses recorded`
and `timestamps recorded`). Its log lines from the run:

```
=== RUN   TestRouter/responses_recorded
    router_test.go:364: recorded: /v1/responses on ggml-org/gpt-oss-120b-GGUF:MXFP4 answers: HTTP 200, status completed, output items [reasoning message], text "Hello!"
=== RUN   TestRouter/timestamps_recorded
    router_test.go:393: recorded: transcription, verbose_json: HTTP 400, 0 segments, 0 words: {"error":{"code":400,"message":"Only 'json' response_format is supported for transcription","type":"invalid_request_error"}}
    router_test.go:393: recorded: transcription, verbose_json with segment and word granularities: HTTP 400, 0 segments, 0 words: {"error":{"code":400,"message":"Only 'json' response_format is supported for transcription","type":"invalid_request_error"}}
    router_test.go:393: recorded: transcription, json with segment and word granularities: HTTP 200, 0 segments, 0 words: {"type":"transcript.text.done","text":"The access code is 7429.","usage":{"type":"tokens","input_tokens":174,"output_tokens":11,"total_tokens":185,"input_tokens_details":{"cached_tokens":169}}}
    router_test.go:403: recorded: transcription on ggml-org/gemma-4-E4B-it-GGUF:Q8_0 returns no timestamps in any form tried
```

By hand, `srt`, `vtt`, and `text` are refused with the same 400 as `verbose_json`, and a request
with no `response_format` answers as `json` does (commands below).

What this means for the client: Responses is a usable surface on the router for gpt-oss-120b,
so the setup decision that rejected it as "router support unverified" can be revisited at the
client's plan round. Transcription is text only: the processing layer's seam alignment
(evidence 5) can't lean on server timestamps and has to align on the overlapping words alone.

## The record

- **Date:** 2026-10-26.
- **llama.cpp build:** `b11669-620d2225c` (`/props` `build_info`), image
  `ghcr.io/ggml-org/llama.cpp:server-cuda13-b11669`, the pin in `engines/llama.env`; the router
  reports `version: 0.7.0-dev (build 11669, commit 620d2225c)`.
- **gb10-2's stack:** Ubuntu 24.04.5 LTS, DGX OS 7.6.0; kernel 7.0.0-1019-nvidia; driver
  580.178.04; CUDA 13.0; Docker 29.6.2; NVIDIA Container Toolkit 1.20.1.
- **The laptop's network path:** direct, under 1 ms (tailscale ping from the laptop, as
  host-update reports it).
- **Model state:** all four of set A loaded by `mise run host-update` before the probe, the
  preset `engines/llama-preset/models.ini` unchanged: `ggml-org/gpt-oss-120b-GGUF:MXFP4`,
  `ggml-org/gemma-4-26B-A4B-it-GGUF:Q4_0`, `ggml-org/gemma-4-E4B-it-GGUF:Q8_0`,
  `ggml-org/embeddinggemma-2-GGUF:Q8_0`.

## The probe

`mise run integration` passed against gb10-2 (`SPIKE_ROUTER_URL` at its default, gb10-2's
router). Every check passed: health; all four models loaded; the build at the pin; each model's
sampling (temperature, top_p, top_k, min_p) and slots (4) as its preset section sets them, with
EmbeddingGemma 2's sampling skipped since its section sets none; chat ("ready"); a tool call
(`get_weather` with `{"city": "Paris"}`); image description by 26B-A4B ("The image features a
red circle, a blue square, and a green triangle arranged on a white background."); audio in chat
by E4B and transcription through `/v1/audio/transcriptions` by E4B, both "The access code is
7429."; and a 768-dimension embedding from EmbeddingGemma 2. The whole probe ran in about 7 s.

## Memory after set A loads

Read at 13:45 local, after the probe, with all four models loaded. Recorded, not gated. The GB10's
memory is unified, so `nvidia-smi` reports no device memory total ("Not Supported") and the
per-process figures are the CUDA allocations within the host's 124544 MiB.

`free -m`:

```
               total        used        free      shared  buff/cache   available
Mem:          124544      104887        1012        5855       25617       19656
Swap:          16383         291       16092
```

`nvidia-smi`, per llama-server process, each mapped to its model by `ps`:

| Process | Model | GPU memory |
|---|---|---|
| router | (the router itself) | 170 MiB |
| child | gpt-oss-120b MXFP4, c 131072 | 65008 MiB |
| child | gemma-4-26B-A4B Q4_0, c 32768, with mmproj | 16972 MiB |
| child | gemma-4-E4B Q8_0, c 32768, with mmproj | 6703 MiB |
| child | embeddinggemma-2 Q8_0, c 8192 | 1525 MiB |
| | **the router and set A** | **90378 MiB** |

The desktop session's own processes (Xorg, gnome-shell, kitty) held 290 MiB more. `docker stats`
reports the container at 9.103 GiB of 121.6 GiB, which counts its host-side pages, not the CUDA
allocations. About 19.2 GiB stays available with set A loaded.

## Against spike-model-engines on gb10-1

spike-model-engines hasn't measured either surface yet. Its brief's behavior 15 probes responses
on gpt-oss-120b and transcription on E4B on llama.cpp at the same build, b11669 (goal record
`context/goals/experiment.hosting.spike-model-engines.md` in spike-model-engines, task
`bench-gb10-container`), but at its commit `9bfa234` the task records slices 1/4 committed, with
the probes written and not run ("cells not run"), and its finding's API probe table
(`docs/findings/bench-gb10-container.md`, "API probes") is blank on both rows.

What engines has written is the expected shape, in its probe instructions
(`docs/engines/llama.cpp/api.md`, "Responses" and "Transcription", from commit `0c41696`): a
working Responses surface is `completed` with a `reasoning` item and a `message` item, and the
router "accepts only `response_format` `json`" for transcription. Both match what gb10-2 answers
here. When engines runs its probes, the comparison is against measurement, not its expectation;
its transcription clip (whisper.cpp's `jfk.wav`) differs from this probe's `phrase.wav`, which
changes the text, not whether timestamps come back.

## Commands, as run

From the laptop, at this repository's root, as run except in where files lived: each log the
session wrote to its scratch directory is written beside the command here instead, and the
hand-run requests read `probe/testdata/phrase.wav`, the byte-identical copy
(`probe/testdata/README.md`) of the spike-model-hosting fixture they were run from.

Bring gb10-2 current and load set A:

```sh
mise run host-update > host-update.log 2>&1; echo exit $?
```

Its output (exit 0):

```
host-update: gb10-2
preflight: no apt upgrade pending, no reboot pending, the CDI spec current
clone: host at 62d32e2, which has no upstream: a session delivers it by push
clone: the laptop's host is at bb39f4b; git push ssh://gb10-2/~/repos/spike-ai-pipeline host:refs/heads/host delivers it
router: at the pin, ghcr.io/ggml-org/llama.cpp:server-cuda13-b11669
preset: unchanged
models: all 4 of set A loaded: ggml-org/gpt-oss-120b-GGUF:MXFP4 ggml-org/gemma-4-26B-A4B-it-GGUF:Q4_0 ggml-org/gemma-4-E4B-it-GGUF:Q8_0 ggml-org/embeddinggemma-2-GGUF:Q8_0
record:
  build    version: 0.7.0-dev (build 11669, commit 620d2225c)
  os       Ubuntu 24.04.5 LTS, DGX OS 7.6.0
  kernel   7.0.0-1019-nvidia
  driver   580.178.04
  cuda     13.0
  docker   29.6.2
  ctk      1.20.1
  network  direct, under 1ms (tailscale ping from the laptop)
host-update: nothing changed: clone current, router at the pin, preset unchanged, set A loaded
```

The clone on gb10-2 was at 62d32e2, one commit behind the laptop, a commit that changes only the
goal record (`git diff --stat 62d32e2 bb39f4b`); the preset it mounts is the one the probe reads.

The surfaces by hand, before the probe was written:

```sh
R=http://gb10-2:8080; curl -s "$R/health"; echo; curl -s "$R/models?autoload=false" | jq -c '.data[] | {id, s: .status.value}'; curl -s "$R/props?model=ggml-org/embeddinggemma-2-GGUF:Q8_0&autoload=false" | jq -c '{build_info, total_slots, p: .default_generation_settings.params | {temperature, top_p, top_k, min_p}}'; curl -s "$R/props?autoload=false" | jq -c '{build_info}'
echo ---responses; time curl -s "$R/v1/responses?autoload=false" -H "Content-Type: application/json" -d '{"model": "ggml-org/gpt-oss-120b-GGUF:MXFP4", "input": "Say hello in one sentence.", "max_output_tokens": 1024}' | jq -c '{status, output: [.output[] | {type, content}]}'
echo ---transcr-json; curl -s "$R/v1/audio/transcriptions?autoload=false" -F model=ggml-org/gemma-4-E4B-it-GGUF:Q8_0 -F response_format=json -F file=@probe/testdata/phrase.wav; echo
echo ---verbose; curl -s "$R/v1/audio/transcriptions?autoload=false" -F model=ggml-org/gemma-4-E4B-it-GGUF:Q8_0 -F response_format=verbose_json -F 'timestamp_granularities[]=word' -F 'timestamp_granularities[]=segment' -F file=@probe/testdata/phrase.wav -w '\n%{http_code}\n'
```

```sh
R=http://gb10-2:8080; for rf in verbose_json srt vtt text; do echo "--- $rf"; curl -s "$R/v1/audio/transcriptions?autoload=false" -F model=ggml-org/gemma-4-E4B-it-GGUF:Q8_0 -F response_format=$rf -F file=@probe/testdata/phrase.wav -w ' %{http_code}\n'; done; echo "--- json+granularities"; curl -s "$R/v1/audio/transcriptions?autoload=false" -F model=ggml-org/gemma-4-E4B-it-GGUF:Q8_0 -F response_format=json -F 'timestamp_granularities[]=word' -F 'timestamp_granularities[]=segment' -F file=@probe/testdata/phrase.wav -w ' %{http_code}\n'; echo "--- no format"; curl -s "$R/v1/audio/transcriptions?autoload=false" -F model=ggml-org/gemma-4-E4B-it-GGUF:Q8_0 -F file=@probe/testdata/phrase.wav -w ' %{http_code}\n'; curl -s "$R/props?model=ggml-org/gemma-4-E4B-it-GGUF:Q8_0&autoload=false" | jq -c '{total_slots, p: .default_generation_settings.params | {temperature, top_p, top_k, min_p}}'
```

The probe, through the task and then verbose for its log lines:

```sh
mise run integration -- 2>&1 | tail -5; GOWORK=off SPIKE_ROUTER_URL=http://gb10-2:8080 go test -race -tags integration -count=1 -v ./probe/... 2>&1 | tee integration-v.log
```

Memory, after the probe:

```sh
date -Is; ssh gb10-2 free -m; ssh gb10-2 nvidia-smi; ssh gb10-2 nvidia-smi --query-compute-apps=pid,process_name,used_memory --format=csv; ssh gb10-2 docker stats --no-stream --format '"{{.Name}} {{.MemUsage}}"'
```

```sh
ssh gb10-2 ps -o pid=,args= -p 2881274,2881500,2881570,2881602,2883000 | sed -E 's/^ *([0-9]+) .*(--alias|-hf|--hf-repo|--model|-m) ([^ ]+).*/\1 \2 \3/' | cut -c1-200
```

The hermetic check:

```sh
mise run check 2>&1 | tail -6
```
