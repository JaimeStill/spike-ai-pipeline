#!/usr/bin/env bash
# currency reports every dependency and toolchain that trails its latest
# release, one line each, and exits non-zero when it reports any: the Go
# modules, the mise tools of mise.toml and of the host environment
# (mise.host.toml, hf's), and gb10's llama.cpp pin (host/llama.env). It reads
# the network and writes nothing. Run it through `mise run currency`.
#
# Each scan captures its command's output in a variable before reading it, so
# under set -e a scan command that fails fails currency instead of reporting
# nothing.
set -euo pipefail
cd "$MISE_PROJECT_ROOT"
export GOWORK=off

stale=0
report() {
	echo "$1"
	stale=1
}

go_minor=$(mise latest go | cut -d. -f1,2)

for mod in $GO_MODULES; do
	# Direct requirements with a newer version within their major.
	updates=$(cd "$mod" && go list -m -u -f \
		'{{if and (not .Main) (not .Indirect) .Update}}{{.Path}} {{.Version}} -> {{.Update.Version}}{{end}}' all)
	while read -r line; do
		[ -n "$line" ] && report "$mod/go.mod: $line"
	done <<<"$updates"

	# The go directive's minor against the current Go minor.
	directive=$(cd "$mod" && go mod edit -json | jq -r .Go | cut -d. -f1,2)
	[ "$directive" = "$go_minor" ] || report "$mod/go.mod: go $directive -> go $go_minor"
done

# mise tools pinned below their latest version: mise.toml's and the host environment's
# (mise.host.toml, hf's), read together with MISE_ENV=host, which layers the host file over
# mise.toml. mise lists a tool not installed here, such as hf on a laptop that never ran the host
# environment, with no bump, so its pin is compared with its latest instead.
outdated=$(MISE_ENV=host MISE_LOG_LEVEL=error mise outdated --bump --local --json |
	jq -r 'to_entries[] | (.value.bump // .value.latest) as $to | select($to != .value.requested) |
		"\(.value.source.path | split("/") | last): \(.key) \(.value.requested) -> \($to)"')
while read -r line; do
	[ -n "$line" ] && report "$line"
done <<<"$outdated"

# gb10's llama.cpp pin (host/llama.env, read through the copied llama_pin, for class gb10 by name,
# since gb10-2 is a GB10 whatever GPU runs currency) against the newest b-build that has a
# server-cuda13 image, server-cuda13-bNNNNN, among ggml's image tags on ghcr.io. The registry's tag
# list is read with an anonymous pull token, page by page. The pin is reported only when it is
# lower. A pin held on purpose is reported like any other, with llama_hold, why it is held, after
# it.
# shellcheck source=host/llama-release.sh
source host/llama-release.sh
pin=$(llama_pin gb10)
llama_hold="held at b12098 by gb10-2-align: b12100 and later can't resolve Google's QAT -hf tags"
llama_hold+=" (llama.cpp #31620, #31639)"
registry=https://ghcr.io
token=$(curl -fsSL "$registry/token?scope=repository:ggml-org/llama.cpp:pull" |
	jq -r '.token // empty') || token=
if [ -z "$token" ]; then
	echo "currency: ghcr.io gives no pull token for ggml-org/llama.cpp" >&2
	exit 2
fi
# Each page asks for the tags after the last one read (last=), until a page comes back empty.
last=
latest=0
while :; do
	page="/v2/ggml-org/llama.cpp/tags/list?n=1000${last:+&last=$last}"
	if ! tags=$(curl -fsSL -H "Authorization: Bearer $token" "$registry$page"); then
		echo "currency: ghcr.io's tag list for ggml-org/llama.cpp can't be read ($page)" >&2
		exit 2
	fi
	last=$(jq -r '.tags // [] | last // empty' <<<"$tags")
	[ -n "$last" ] || break
	builds=$(jq -r '.tags[] | capture("^server-cuda13-b(?<b>[0-9]+)$").b' <<<"$tags")
	for b in $builds; do
		if ((b > latest)); then latest=$b; fi
	done
done
if ((latest == 0)); then
	echo "currency: ghcr.io lists no server-cuda13 image for ggml-org/llama.cpp" >&2
	exit 2
fi
if ((${pin#b} < latest)); then
	report "host/llama.env: llama.cpp (gb10) $pin -> b$latest (server-cuda13), $llama_hold"
fi

exit "$stale"
