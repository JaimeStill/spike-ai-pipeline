# Finding: set A on gb10-2, responses and transcription timestamps

Task `host`, behaviors 9 to 11 (`context/goals/experiment.ai.spike-ai-pipeline.md`, "Task brief ·
host"). Measured 2026-10-26, from the laptop, against gb10-2's llama.cpp router serving set A.

The commands here name the scripts' current directory, `host/`. They ran on gb10-2 from
`engines/`, as `engines/engines llama install`, before spike-model-engines renamed the directory
(host-rename, 7cd0ddb) and the host-scripts task re-copied the scripts from d285b06 under `host/`.
Only the paths were rewritten; no result changed.

## Results: Responses and transcription timestamps

| Question | Result |
|---|---|
| Does `/v1/responses` answer on gpt-oss-120b? | **Yes.** HTTP 200, `status` `completed`, output items `reasoning` then `message`, the message's `output_text` holding the reply. |
| Does transcription return segment or word timestamps? | **No.** `/v1/audio/transcriptions` accepts only `response_format` `json`, which carries `text` and `usage` and no timestamps. `verbose_json` is refused with HTTP 400; `timestamp_granularities[]` is ignored under `json`. |

The probe records both and asserts neither (`probe/router_test.go`, subtests `responses recorded`
and `timestamps recorded`). Its log lines:

```
=== RUN   TestRouter/responses_recorded
    router_test.go:365: recorded: /v1/responses on ggml-org/gpt-oss-120b-GGUF:MXFP4 answers: HTTP 200, status completed, output items [reasoning message], text "Hello, nice to meet you!"
=== RUN   TestRouter/timestamps_recorded
    router_test.go:394: recorded: transcription, verbose_json: HTTP 400, 0 segments, 0 words: {"error":{"code":400,"message":"Only 'json' response_format is supported for transcription","type":"invalid_request_error"}}
    router_test.go:394: recorded: transcription, verbose_json with segment and word granularities: HTTP 400, 0 segments, 0 words: {"error":{"code":400,"message":"Only 'json' response_format is supported for transcription","type":"invalid_request_error"}}
    router_test.go:394: recorded: transcription, json with segment and word granularities: HTTP 200, 0 segments, 0 words: {"type":"transcript.text.done","text":"The access code is 7429.","usage":{"type":"tokens","input_tokens":174,"output_tokens":11,"total_tokens":185,"input_tokens_details":{"cached_tokens":169}}}
    router_test.go:404: recorded: transcription on ggml-org/gemma-4-E4B-it-GGUF:Q8_0 returns no timestamps in any form tried
```

These lines come from a rerun of the committed probe, so their line numbers match the source. The
first run gave the same results; only gpt-oss-120b's sampled reply differed ("Hello!").

By hand, `srt`, `vtt`, and `text` are refused with the same 400 as `verbose_json`, and a request
with no `response_format` answers as `json` does (commands below).

For the client, Responses is a usable surface on the router for gpt-oss-120b, so the client's
plan round can revisit the setup decision that rejected it as "router support unverified".
Transcription returns text only, so the processing layer's seam alignment (evidence 5) aligns on
the overlapping words alone, without server timestamps.

## Build, host stack, and network path

- **Date:** 2026-10-26.
- **llama.cpp build:** `b11669-620d2225c` (`/props` `build_info`), image
  `ghcr.io/ggml-org/llama.cpp:server-cuda13-b11669`, the pin in `host/llama.env`; the router
  reports `version: 0.7.0-dev (build 11669, commit 620d2225c)`.
- **gb10-2's stack:** Ubuntu 24.04.5 LTS, DGX OS 7.6.0; kernel 7.0.0-1019-nvidia; driver
  580.178.04; CUDA 13.0; Docker 29.6.2; NVIDIA Container Toolkit 1.20.1.
- **The laptop's network path:** direct, under 1 ms (tailscale ping from the laptop, as
  host-update reports it).
- **Model state:** all four of set A loaded by `mise run host-update` before the probe, the
  preset `host/llama-preset/models.ini` unchanged: `ggml-org/gpt-oss-120b-GGUF:MXFP4`,
  `ggml-org/gemma-4-26B-A4B-it-GGUF:Q4_0`, `ggml-org/gemma-4-E4B-it-GGUF:Q8_0`,
  `ggml-org/embeddinggemma-2-GGUF:Q8_0`.

## The integration probe's checks

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

## Comparison with spike-model-engines on gb10-1

spike-model-engines hasn't measured either surface yet. Its task `bench-gb10-container` probes
responses on gpt-oss-120b and transcription on E4B on llama.cpp at the same build, b11669
(behavior 15 of its brief, in `context/goals/experiment.hosting.spike-model-engines.md`). At its
commit `9bfa234` the task records slices 1/4 committed, with the probes written and not run
("cells not run"), and its finding's API probe table (`docs/findings/bench-gb10-container.md`,
"API probes") is blank on both rows.

spike-model-engines' probe instructions give the expected shape (`docs/engines/llama.cpp/api.md`,
"Responses" and "Transcription", from commit `0c41696`): a working Responses surface is
`completed` with a `reasoning` item and a `message` item, and the router "accepts only
`response_format` `json`" for transcription. Both match what gb10-2 answers here. Once engines runs
its probes, compare against its measurement instead of its expectation. Its transcription clip
(whisper.cpp's `jfk.wav`) differs from this probe's `phrase.wav`, which changes the text, not
whether timestamps come back.

## Commands, as run

Each command runs from the laptop at this repository's root unless it says gb10-2. Two things
differ from the run, both in where files live: each log the session wrote to its scratch
directory is written beside the command here, and the hand-run requests read
`probe/testdata/phrase.wav`, a byte-identical copy (`probe/testdata/README.md`) of the
spike-model-hosting fixture they read.

Set gb10-2 up once, before the first host-update. On gb10-2, clone the root:

```sh
cd ~/repos && git clone https://github.com/JaimeStill/spike-ai-pipeline.git
```

From the laptop, deliver the task branch, which has no upstream, then check it out on gb10-2.
host-update has since set the clone's `receive.denyCurrentBranch` to `updateInstead`, so a later
push updates the checked-out branch and its files.

```sh
git push ssh://gb10-2/~/repos/spike-ai-pipeline host:refs/heads/host
```

```sh
cd ~/repos/spike-ai-pipeline && git checkout host
```

On gb10-2, in the clone, the download tool (`hf version` printed 2.3.0):

```sh
MISE_ENV=host mise install uv pipx:huggingface_hub; MISE_ENV=host mise exec -- hf version
```

Still on gb10-2 in the clone, the three downloads, run in parallel under nohup, then the router
image, the router, and a check for a partial download (none):

```sh
export MISE_ENV=host HF_HUB_CACHE=/srv/models HF_XET_HIGH_PERFORMANCE=1
mise exec -- hf download ggml-org/gemma-4-26B-A4B-it-GGUF --include gemma-4-26B-A4B-it-Q4_0.gguf --include mmproj-gemma-4-26B-A4B-it-Q8_0.gguf
mise exec -- hf download ggml-org/gemma-4-E4B-it-GGUF --include gemma-4-E4B-it-Q8_0.gguf --include mmproj-gemma-4-E4B-it-Q8_0.gguf
mise exec -- hf download ggml-org/embeddinggemma-2-GGUF --include embeddinggemma-2-Q8_0.gguf
docker pull ghcr.io/ggml-org/llama.cpp:server-cuda13-b11669
host/engines llama install
find /srv/models -name '*.incomplete'
```

| Download | Size | Time |
|---|---|---|
| gemma-4-26B-A4B-it Q4_0 and its mmproj | 15 GB | 2213 s |
| gemma-4-E4B-it Q8_0 and its mmproj | 8.1 GB | 1267 s |
| embeddinggemma-2 Q8_0 | 296 MB | 721 s |
| the router image | 7 GB | 2234 s |

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

The rerun for the log lines quoted above, against the probe as committed:

```sh
GOWORK=off SPIKE_ROUTER_URL=http://gb10-2:8080 go test -race -tags integration -count=1 -v ./probe/ > probe-v.log 2>&1; echo exit $?
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

## 2026-10-27: set A's embedding model is granite-embedding-english-r2

Task `client-core`. Set A's embedding model changed from EmbeddingGemma 2
(`ggml-org/embeddinggemma-2-GGUF:Q8_0`) to IBM's granite-embedding-english-r2
(`mrutkows/granite-embedding-english-r2-GGUF:BF16`), because vLLM can't serve EmbeddingGemma 2 on
the GB10 and set A has to serve on both engines (`context/goals/experiment.ai.spike-ai-pipeline.md`,
Decisions, "client-core: set A's embedding model…"). The sections above record the first run, with
EmbeddingGemma 2, and stay as measured. The preset's section and its reasons are in
`host/llama-preset/models.ini`: c 32768 over 4 slots, each its trained 8192; b = ub = 8192;
embeddings on; CLS pooling, as its GGUF declares.

- **llama.cpp build:** the router reports `version: 0.7.0-dev (build 11669, commit 620d2225c)`,
  as host-update's record prints it.
- **gb10-2's stack:** Ubuntu 24.04.5 LTS, DGX OS 7.6.0; kernel 7.0.0-1019-nvidia; driver
  580.178.04; CUDA 13.0; Docker 29.6.2; NVIDIA Container Toolkit 1.20.1.
- **The laptop's network path:** direct, under 1 ms (tailscale ping from the laptop, as
  host-update reports it).
- **Staging:** `granite-embedding-english-r2-bf16.gguf`, 299,876,704 bytes, downloaded into
  `/srv/models` by hf 2.3.0 in 9 s. EmbeddingGemma 2's file stays in the store, read by nothing,
  so the router still lists its cache entry, `unloaded`.
- **A dead preset mount, and the fix:** delivering the branch replaced the clone's
  `engines/llama-preset/` directory (now `host/llama-preset/`), which left the running container's
  bind mount on the old, empty one. The reload then failed: `/models?reload=1` returned HTTP 500,
  "preset file does not exist: /etc/llama-preset/models.ini". host-update now checks the mount and
  restarts a running router that can't see the preset the clone has, which binds the directory
  afresh; it still never starts a stopped router. Its run printed `router: restarted, since it
  couldn't see /etc/llama-preset/models.ini, which the clone has (a dead preset mount)` and
  requested the four loads. One `/models` poll during the loads was refused ("Could not connect to
  server"), which the run reported as the router not listing gpt-oss-120b, and it exited 1. A second
  run, with no change between, found all four loaded (exit 0, "nothing changed").
- **A refused poll, and the fix:** host-update's load loop now reads a `/models` request that
  fails (refused, non-2xx, or not the JSON expected) as a router not ready yet and keeps polling
  until its deadline; only a listing that lacks a preset model fails the run at once. Its run with
  the fix, at `54d55cb`, exited 0: "nothing changed", all four of set A loaded.
- **The load:** granite loaded with 4 slots of n_ctx 8192 (`/props`), its llama-server holding
  863 MiB of GPU memory (`nvidia-smi`), against EmbeddingGemma 2's 1525 MiB.
- **The probe:** `mise run integration` passed, every check as before, with granite's embedding at
  768 dimensions. By hand, the probe's input ("The access code is seven four two nine.", 11 tokens)
  embeds to a vector of norm 1.0000000085: llama.cpp normalizes the embedding.

The commands, from the laptop at this repository's root unless they say gb10-2. On gb10-2, in the
clone:

```sh
export MISE_ENV=host HF_HUB_CACHE=/srv/models HF_XET_HIGH_PERFORMANCE=1
mise exec -- hf download mrutkows/granite-embedding-english-r2-GGUF --include granite-embedding-english-r2-bf16.gguf
```

Delivering the branch to the clone, which was on `main`, then host-update and the probe:

```sh
git push ssh://gb10-2/~/repos/spike-ai-pipeline client-core:refs/heads/client-core
ssh gb10-2 git -C repos/spike-ai-pipeline switch client-core
mise run host-update
curl -s "http://gb10-2:8080/models?autoload=false" | jq -c '.data[] | {id, s: .status.value}'
mise run integration
```

The norm, the slots, and the memory:

```sh
R=http://gb10-2:8080; M=mrutkows/granite-embedding-english-r2-GGUF:BF16
curl -s "$R/v1/embeddings" -H 'Content-Type: application/json' -d "{\"model\":\"$M\",\"input\":\"The access code is seven four two nine.\",\"encoding_format\":\"float\"}" | jq -c '{n: (.data[0].embedding|length), norm: ([.data[0].embedding[]|.*.]|add|sqrt), usage}'
curl -s "$R/props?model=$M&autoload=false" | jq -c '{build_info, total_slots, n_ctx: .default_generation_settings.n_ctx}'
ssh gb10-2 nvidia-smi --query-compute-apps=pid,used_memory --format=csv,noheader
```

## 2026-10-27: the router's `/v1/messages`

Task `client-messages`. The client's Messages format (spike-ai-pipeline-client `format/messages`)
ran every Generate and GenerateStream scenario of the client's integration tier against gb10-2's
router, beside Chat Completions, and the run passed in 88.5 s. The router, at the same build,
`version: 0.7.0-dev (build 11669, commit 620d2225c)`, answers `/v1/messages` as follows:

- **The request:** the router rebuilds the body as a Chat Completions request. It drops
  `output_config`, so structured output came back as prose and reasoning effort had no consistent
  effect. It also drops a tool's `strict` and the `none` tool choice.
- **Named tool choices:** the router maps a named tool choice and an `allowed_tools` choice to
  required, yet gpt-oss-120b made no call under either. It repeated its answer, "4", until the
  1024-token bound, and each reply came back `incomplete` after about 21 s. Through Chat
  Completions the same requests completed with "4" in under 1 s.
- **Reasoning:** a reply's thinking blocks carry an empty signature. Sending them back works,
  since the router reads the thinking text and ignores the signature.
- **The stream:** every `content_block_stop` arrives at the end of the reply, so blocks are open
  at the same time. A stream error frame, read from the source at 620d2225c
  (`tools/server/server-context.cpp:5037-5046`) and not seen in a run, is `event: error` with
  data `{"code","message","type":"server_error"}`, with no `"type":"error"`.

The run's recorded lines for the dropped hints and choices:

```
reasoning_test.go:87: recorded: messages TestReasoningEffort: effort none: status completed, 75 bytes of reasoning, 39 output tokens
reasoning_test.go:87: recorded: messages TestReasoningEffort: effort low: status completed, 350 bytes of reasoning, 139 output tokens
reasoning_test.go:87: recorded: messages TestReasoningEffort: effort high: status completed, 214 bytes of reasoning, 77 output tokens
structured_test.go:48: recorded: messages TestStructuredOutput: output_text decodes as the schema's object: did not hold; output_text "The capital of France is **Paris**.", error jsontext: invalid character 'T' at start of value
tools_test.go:155: recorded: messages TestNamedToolChoices/named_function_get_weather: named function get_weather: status incomplete, output [output_text "4We need to output the number alone, no extra text.4444The answer is 4.44…
tools_test.go:155: recorded: messages TestNamedToolChoices/allowed_tools_get_weather,_required: allowed_tools get_weather, required: status incomplete, output [output_text "44No further.That is final.4Oops repeated. …
tools_test.go:155: recorded: messages TestNamedToolChoices/none: none: status completed, output [output_text "4"]
```

The run is the client's integration tier run verbose, from the laptop in
spike-ai-pipeline-client at `f7c7f01`:
`GOWORK=off SPIKE_ROUTER_URL=http://gb10-2:8080 mise exec -- go test -race -tags integration -count=1 -v ./...`.

## 2026-10-28: the router's `/v1/responses`

Task `client-responses`. The client's Open Responses format (spike-ai-pipeline-client
`format/responses`, with `ResendReasoningText`) ran every Generate and GenerateStream scenario of
the client's integration tier against gb10-2's router, beside Chat Completions and Messages, and
the run passed in 106.3 s. The router, at the same build, `version: 0.7.0-dev (build 11669, commit
620d2225c)`, answers `/v1/responses` as follows:

- **The request:** the router rebuilds the body as a Chat Completions request. It reads
  instructions, `max_output_tokens`, reasoning's effort, the tools, and the input items. It drops
  `text`, so structured output came back as prose; it also drops reasoning's summary, an image's
  detail, `store`, and `include`, and treats a named or `allowed_tools` tool choice as auto. Under
  both, gpt-oss-120b answered "4" without a call and completed, unlike through `/v1/messages`.
- **Refusals:** it refuses `previous_response_id`, a content part other than `input_text`,
  `input_image`, or `input_file` (the client's audio extension among them), and a reasoning item
  without content, which is the spec's input form.
- **Reasoning:** a reply's reasoning items carry an empty `encrypted_content`. The router reads a
  reasoning item's content back, so the reasoning round trip works only with the text resent as
  content. A reasoning item that follows an assistant turn replaces that turn's reasoning, and of
  consecutive reasoning items only the last reaches the model.
- **Status:** every reply reports `completed`, even one cut off at `max_output_tokens`.
- **The stream:** reasoning streams under OpenAI's names, `response.reasoning_text.delta` and
  `.done`. No event carries a `sequence_number`, `output_index`, or `content_index`; reasoning has
  no part events or done event, a function call no arguments done event, and the streamed
  reasoning item no status. Every item stays open until the end of the reply. A stream error
  frame, read from the source at 620d2225c (`tools/server/server-context.cpp:5037-5046`) and not
  seen in a run, is an unnamed event with data `{"error":{"code","message","type"}}`, with no
  `"type":"error"`.

The run's recorded lines for Open Responses:

```
chat_test.go:53: recorded: responses TestChatIncompleteAtMaxOutputTokens: status "completed", incomplete_details <nil>, output_tokens 16 at the bound 16
media_test.go:78: recorded: responses TestAudioInResponses: refused: 400 Bad Request: 'type' must be one of 'input_text', 'input_image', or 'input_file' (invalid_request_error, 400)
reasoning_test.go:87: recorded: responses TestReasoningEffort: effort none: status completed, 78 bytes of reasoning, 42 output tokens
reasoning_test.go:87: recorded: responses TestReasoningEffort: effort low: status completed, 71 bytes of reasoning, 40 output tokens
reasoning_test.go:87: recorded: responses TestReasoningEffort: effort high: status completed, 363 bytes of reasoning, 131 output tokens
structured_test.go:48: recorded: responses TestStructuredOutput: output_text decodes as the schema's object: did not hold; output_text "The capital of France is **Paris**.", error jsontext: invalid character 'T' at start of value
tools_test.go:155: recorded: responses TestNamedToolChoices/named_function_get_weather: named function get_weather: status completed, output [output_text "4"]
tools_test.go:155: recorded: responses TestNamedToolChoices/allowed_tools_get_weather,_required: allowed_tools get_weather, required: status completed, output [output_text "4"]
tools_test.go:155: recorded: responses TestNamedToolChoices/none: none: status completed, output [output_text "4"]
```

The run is the client's integration tier run verbose, from the laptop in
spike-ai-pipeline-client at `6032cf7`:
`GOWORK=off SPIKE_ROUTER_URL=http://gb10-2:8080 mise exec -- go test -race -tags integration -count=1 -v ./integration/`.
