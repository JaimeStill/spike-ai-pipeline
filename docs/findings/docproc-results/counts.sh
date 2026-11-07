#!/usr/bin/env bash
# counts.sh <record.jsonl>: the docproc finding's counts from one run's record, schema "1", as
# Markdown tables: inputs and parts by route and outcome, outputs by kind, the slowest inputs,
# the inspection flags, and every failed, encrypted, quarantined, or error line with its reason.
# Needs jq.
set -euo pipefail
rec=$1

# brief reduces an error to one printable line, as docproc's terminal prints it (internal/run's
# brief): its first line, each rune that isn't printable replaced by U+FFFD, cut at 400 runes,
# an ellipsis when any of it is left out. Processing's errors can run to several lines, and one
# carries Ghostscript's raw PNG output.
# shellcheck disable=SC2016 # the $s, $nl, and $l are jq's variables.
brief='def brief: (sub("^\\s+"; "") | sub("\\s+$"; "")) as $s | ($s | index("\n")) as $nl
  | (if $nl then $s[0:$nl] else $s end | gsub("[^\\p{L}\\p{M}\\p{N}\\p{P}\\p{S} ]"; "\ufffd")) as $l
  | if ($l | length) > 400 then $l[0:400] + "…" elif $nl then $l + "…" else $l end;'

# secs turns a schema "1" time, RFC 3339 with nine digits of nanoseconds, into seconds.
secs='def secs: (.[0:19] + "Z" | fromdateiso8601) + ("0." + .[20:29] | tonumber);'

table() { # table <title> <jq filter yielding {route, outcome}>
  echo "### $1"
  echo
  echo "| Route | Outcome | Count |"
  echo "|---|---|---|"
  jq -rs "[.[] | $2] | group_by([.route, .outcome]) | map([.[0].route, .[0].outcome, length])
    | sort_by(.[0], .[1]) | .[] | \"| \(.[0]) | \(.[1]) | \(.[2]) |\"" "$rec"
  jq -rs "[.[] | $2] | \"| total | | \(length) |\"" "$rec"
  echo
}

table "Inputs by route and outcome" \
  'select((.event == "part" and .part.depth == 0) or .event == "error")
   | {route: (.part.content_type.route // "-"), outcome: (.outcome // "error")}'
table "Nested parts (depth 1 and more) by route and outcome" \
  'select(.event == "part" and .part.depth > 0)
   | {route: (.part.content_type.route // "-"), outcome: .outcome}'

echo "### Parts by depth"
echo
echo "| Depth | Parts |"
echo "|---|---|"
jq -rs '[.[] | select(.event == "part") | .part.depth] | group_by(.) | .[]
  | "| \(.[0]) | \(length) |"' "$rec"
echo

echo "### Output lines by kind and whether seen"
echo
echo "| Kind | Outcome | Lines |"
echo "|---|---|---|"
jq -rs '[.[] | select(.event == "output") | {k: .output.kind, o: (.outcome // "prepared")}]
  | group_by([.k, .o]) | .[] | "| \(.[0].k) | \(.[0].o) | \(length) |"' "$rec"
jq -rs '[.[] | select(.event == "output")] | "| total | | \(length) |"' "$rec"
echo
echo "Distinct refs: $(jq -r 'select(.event == "output") | .output.ref' "$rec" | sort -u | wc -l)"
echo

echo "### The 15 slowest inputs"
echo
echo "| Time (s) | Route | Outcome | Outputs | Children | Input |"
echo "|---|---|---|---|---|---|"
jq -rs "$secs"' [.[] | select(.time) | {t: ((.time.end | secs) - (.time.start | secs)),
  r: (.part.content_type.route // "-"), o: (.outcome // "error"), n: (.outputs // 0),
  c: (.children // 0), i: .input}] | sort_by(-.t) | .[0:15] | .[]
  | "| \(.t * 1000 | round / 1000) | \(.r) | \(.o) | \(.n) | \(.c) | `\(.i)` |"' "$rec"
echo

echo "### Inspection flags by check and class, on part lines"
echo
echo "| Check | Class | Flags |"
echo "|---|---|---|"
jq -rs '[.[] | select(.event == "part") | .part.flags // [] | .[] | {c: .check, k: .class}]
  | group_by([.c, .k]) | .[] | "| \(.[0].c) | \(.[0].k) | \(length) |"' "$rec"
echo
echo "Flagged part lines by outcome: $(jq -rs '[.[] | select(.event == "part" and .part.flags)
  | .outcome] | group_by(.) | map("\(.[0]) \(length)") | join(", ")' "$rec")"
echo

echo "### Failed, encrypted, quarantined, and error lines"
echo
echo "Identical lines, the same part name, route, outcome, input, and reason, are counted once."
echo
echo "| Lines | Event | Depth | Route | Outcome | Part | Input | Reason or error |"
echo "|---|---|---|---|---|---|---|---|"
jq -rs "$brief"' [.[] | select(.event == "error" or (.event == "part"
    and (.outcome == "failed" or .outcome == "encrypted" or .outcome == "quarantined"
      or .recorded == "failed" or .recorded == "encrypted" or .recorded == "quarantined")))
  | {e: .event, d: (.part.depth // "-"), r: (.part.content_type.route // "-"),
     o: ((.outcome // "error") + (if .recorded then " (recorded \(.recorded))" else "" end)),
     n: (.part.name // "-"), i: .input,
     why: ([(.reason, .error | values | brief), (.quarantine.file // null)]
       | map(select(. != null)) | unique
       | join("; ") | gsub("\\|"; "\\|") | gsub("\n"; " "))}]
  | group_by([.e, .d, .r, .o, .n, .i, .why]) | .[]
  | "| \(length) | \(.[0].e) | \(.[0].d) | \(.[0].r) | \(.[0].o) | \(.[0].n) | `\(.[0].i)` | \(.[0].why) |"' "$rec"
