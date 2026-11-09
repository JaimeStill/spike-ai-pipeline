#!/usr/bin/env bash
# counts.sh <manifest.json> [chunks.jsonl]: the docinfer finding's counts from one run, schema
# "1", as Markdown tables: each capability's requests, failures, and tokens; what the run read;
# the embedding space; the chunk lines and distinct chunks by source kind; the texts by kind; the
# chunks' lengths; the transcripts' seams; and the failures by capability and error. The chunk
# tables need the chunk record, which defaults to the one the manifest names beside it; the texts
# by kind read the docproc record the manifest names, when it is where the manifest says. Needs jq.
set -euo pipefail
man=$1
chunks=${2:-$(dirname "$man")/chunks.jsonl}

echo "### The run"
echo
jq -r '"Run `\(.run)`, \(.start) to \(.end // "(no end)"), over docproc run `\(.docproc.run)` in `\(.docproc.dir)`." +
  (if .error then " Error: \(.error)" else "" end)' "$man"
echo
echo "| Capability | Endpoint | Model | Settings |"
echo "|---|---|---|---|"
jq -r '.capabilities[] | "| \(.name) | \(.endpoint) | `\(.model)` | \(.settings | to_entries
  | map("\(.key) \(.value | tojson)") | join(", ") | gsub("\\|"; "\\|")) |"' "$man"
jq -r '.skipped // [] | .[] | "| \(.name) | - | - | skipped: \(.reason) |"' "$man"
echo

echo "### Requests, failures, and tokens by capability"
echo
echo "Failures are the distinct images refused, windows refused, or chunks rejected, this run or held."
echo
echo "| Capability | Requests | Failures | Input tokens | Output tokens | Total tokens |"
echo "|---|---|---|---|---|---|"
jq -r '.counts as $c | [
    (if $c.vision then ["vision", $c.vision.requests, $c.vision.refused, $c.tokens.vision] else empty end),
    (if $c.transcription then ["transcription", $c.transcription.requests, $c.transcription.failed, $c.tokens.transcription] else empty end),
    ["embeddings", $c.requests, $c.rejected, $c.tokens.embeddings]
  ] | .[] | "| \(.[0]) | \(.[1]) | \(.[2]) | \(.[3].input // "-") | \(.[3].output // "-") | \(.[3].total // "-") |"' "$man"
echo

echo "### What each capability did"
echo
jq -r '.counts | "- docproc output lines: \(.outputs); image lines \(.images), audio window lines \(.windows).",
  (if .vision then "- vision: distinct images \(.vision.images): held \(.vision.held), described \(.vision.described), incomplete \(.vision.incomplete) (empty \(.vision.empty // 0)), refused \(.vision.refused); texts embedded \(.vision.texts)." else "- vision: skipped." end),
  (if .transcription then "- transcription: distinct windows \(.transcription.distinct): held \(.transcription.held), transcribed \(.transcription.transcribed), failed \(.transcription.failed); transcripts \(.transcription.transcripts), seams matched \(.transcription.seams_matched), unmatched \(.transcription.seams_unmatched)." else "- transcription: skipped." end),
  "- embeddings: text and Markdown outputs \(.texts), metadata texts \(.metadata); distinct chunks \(.chunks): held \(.held), embedded \(.embedded), rejected \(.rejected); chunk lines \(.lines)."' "$man"
echo

echo "### The embedding space"
echo
echo "| Endpoint | Model | Dimensions | Vectors | Unit length | Tolerance | Min norm | Max norm | Chunk size | Chunker |"
echo "|---|---|---|---|---|---|---|---|---|---|"
jq -r '.embedding_space // empty | "| \(.endpoint) | `\(.model)` | \(.dimensions) | \(.vectors) | \(.unit_length) | \(.tolerance) | \(.min_norm) | \(.max_norm) | \(.chunk_size) | \(.chunker) |"' "$man"
echo

rec=$(jq -r '.docproc.dir + "/runs/" + .docproc.run + "/record.jsonl"' "$man")
if [ -f "$rec" ]; then
  echo "### Texts by kind"
  echo
  echo "One text per output line, docproc's text and Markdown outputs from its record; one vision text per image line with a text; one metadata text per input; one transcript per part and stage with windows."
  echo
  echo "| Kind | Texts |"
  echo "|---|---|"
  jq -r 'select(.event == "output" and (.output.kind == "text" or .output.kind == "markdown")) | .output.kind' "$rec" \
    | sort | uniq -c | awk '{print "| " $2 " | " $1 " |"}'
  jq -r '.counts | "| metadata | \(.metadata) |", "| vision | \(.vision.texts // 0) |", "| transcript | \(.transcription.transcripts // 0) |"' "$man"
  echo
fi

if [ ! -f "$chunks" ]; then
  exit 0
fi
# The chunk lines without their vectors, which the tables below don't read.
slim=$(mktemp)
trap 'rm -f "$slim"' EXIT
jq -c 'del(.vector) | {t: .text, k: .source.kind, ok: (.error == null)}' "$chunks" > "$slim"

echo "### Chunks by source kind"
echo
echo "Lines are the chunk record's, one per chunk per occurrence; distinct chunks are by text, within the kind."
echo
echo "| Kind | Lines | Distinct chunks | Rejected lines |"
echo "|---|---|---|---|"
jq -rs 'group_by(.k) | .[] | "| \(.[0].k) | \(length) | \(map(.t) | unique | length) | \(map(select(.ok | not)) | length) |"' "$slim"
jq -rs '"| total | \(length) | \(map(.t) | unique | length) | \(map(select(.ok | not)) | length) |"' "$slim"
echo

echo "### Chunk lengths, in runes, over the distinct chunks of each kind"
echo
echo "| Kind | Chunks | Min | Median | p95 | Max |"
echo "|---|---|---|---|---|---|"
# shellcheck disable=SC2016 # $n is jq's.
jq -rs 'def dist: map(.t) | unique | map(length) | sort | length as $n
    | [$n, .[0], .[(($n - 1) / 2 | floor)], .[(($n - 1) * 0.95 | floor)], .[$n - 1]];
  (group_by(.k) | .[] | [.[0].k] + dist), (["all"] + dist)
  | "| \(.[0]) | \(.[1]) | \(.[2]) | \(.[3]) | \(.[4]) | \(.[5]) |"' "$slim"
echo

echo "### Failures by capability and error"
echo
echo "Each is an output line whose image or window the endpoint refused; the error is cut at 200 characters."
echo
echo "| Capability | Lines | Inputs | Error |"
echo "|---|---|---|---|"
jq -r '.failures // [] | group_by([.capability, .error]) | .[]
  | "| \(.[0].capability) | \(length) | \(map(.input) | unique | length) | \(.[0].error[0:200] | gsub("\n"; " ") | gsub("\\|"; "\\|")) |"' "$man"
echo
