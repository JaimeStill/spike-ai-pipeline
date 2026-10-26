#!/usr/bin/env bash
# currency reports every dependency and toolchain that trails its latest
# release, one line each, and exits non-zero when it reports any: the Go
# modules, the mise tools of mise.toml and of the host environment
# (mise.host.toml, hf's), and the llama.cpp pin (engines/llama.env). It reads
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

# The llama.cpp pin (engines/llama.env, copied from spike-model-engines) against the b-build the
# notes of llama.cpp's latest semver release name ("Nightly build:"), as spike-model-engines'
# currency reads it. A pin that trails is moved by a new copy of llama.env, not an edit here.
# shellcheck source=engines/llama.env
source engines/llama.env
release=$(curl -fsSL https://api.github.com/repos/ggml-org/llama.cpp/releases/latest)
latest=$(jq -r '.body | capture("Nightly build:[^b]*(?<b>b[0-9]+)").b' <<<"$release")
tag=$(jq -r .tag_name <<<"$release")
if [ -z "$latest" ]; then
	echo "currency: llama.cpp $tag's notes name no b-build (\"Nightly build:\")" >&2
	exit 2
fi
[ "$latest" = "$LLAMA_BUILD" ] || report "engines/llama.env: llama.cpp $LLAMA_BUILD -> $latest ($tag)"

exit "$stale"
