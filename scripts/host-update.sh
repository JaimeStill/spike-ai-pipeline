#!/usr/bin/env bash
# host-update brings gb10-2 up to date and leaves set A loaded on its llama.cpp router. It runs on
# the laptop, at each session's start, through `mise run host-update`, and drives gb10-2 over
# Tailscale SSH; it needs no root there, and runs nothing that does.
#
#   1. Preflight: when gb10-2 has pending apt upgrades, a stale CDI spec (one naming another
#      driver version, or a file that is missing), or a pending reboot, it prints the exact root
#      commands for the architect and exits 1, the router untouched.
#   2. Pulls the root's clone on gb10-2 (~/repos/spike-ai-pipeline), fast-forwarding its branch
#      from its upstream. A branch with no upstream, a task branch not yet merged, is left as it
#      is: a session delivers it from the laptop by push (README.md, "gb10-2").
#   3. Installs the router at the pin (engines/llama.env) with the copied engines scripts when it
#      is absent, or moves it to the pin when it runs another image.
#   4. Has the router re-read the preset when it changed since the router last read it.
#   5. Loads each model the preset names through the router's load API, and waits until all are
#      loaded, or fails naming each that failed or didn't load in time.
#   6. Prints the llama.cpp build, gb10-2's stack, and the laptop's network path to it, with no
#      address, then what changed, or that nothing did, and a clone with no upstream at another
#      commit than the laptop's branch.
#
# The environment overrides its defaults: HOST_UPDATE_HOST (gb10-2), HOST_UPDATE_CLONE
# (repos/spike-ai-pipeline, from the home directory), and HOST_UPDATE_TIMEOUT (900 seconds, the
# wait for set A to load). HOST_UPDATE_FACTS names a file of preflight facts, in host_facts'
# format, to run the preflight on instead of the host's; it then stops after the preflight, with
# the host untouched.
set -euo pipefail

lib=$(dirname "$(readlink -f -- "${BASH_SOURCE[0]}")")/host-update-lib.sh
# shellcheck source=scripts/host-update-lib.sh
source "$lib"

host=${HOST_UPDATE_HOST:-gb10-2}
clone=${HOST_UPDATE_CLONE:-repos/spike-ai-pipeline}
timeout=${HOST_UPDATE_TIMEOUT:-900}
router=http://$host:8080

# Runs a host-update-lib function on the host with the arguments given: the library and the call
# travel as one bash -c script, base64-encoded so no shell between quotes them, with no stdin.
# Its output is redacted, so no tailnet address the host prints reaches the terminal.
on_host() {
	local script
	script=$(
		echo 'set -euo pipefail'
		cat "$lib"
		printf '%q ' "$@"
		echo
	)
	# shellcheck disable=SC2029 # the command is built here, to run there.
	ssh -o BatchMode=yes "$host" "bash -c \"\$(echo $(base64 -w 0 <<<"$script") | base64 -d)\"" \
		</dev/null 2> >(redact >&2) | redact
}

# Waits until the router answers its health check, or fails after $1 seconds.
wait_healthy() {
	local deadline=$((SECONDS + $1))
	until curl -fsS --max-time 5 "$router/health" >/dev/null 2>&1; do
		if ((SECONDS >= deadline)); then
			echo "host-update: the router on $host didn't answer $router/health in $1 s" >&2
			return 1
		fi
		sleep 2
	done
}

echo "host-update: $host"
changes=()
lag=

# 1. Preflight, before anything on the host changes.
if [[ -n ${HOST_UPDATE_FACTS:-} ]]; then
	facts=$(<"$HOST_UPDATE_FACTS")
else
	facts=$(on_host host_facts)
fi
if ! preflight "$host" <<<"$facts"; then
	exit 1
fi
if [[ -n ${HOST_UPDATE_FACTS:-} ]]; then
	echo "preflight: ready on the facts in $HOST_UPDATE_FACTS; stopping before the host"
	exit 0
fi
echo "preflight: no apt upgrade pending, no reboot pending, the CDI spec current"

# 2. The clone.
read -r branch before after upstream <<<"$(on_host clone_pull "$clone")"
if [[ $upstream == - ]]; then
	echo "clone: $branch at $after, which has no upstream: a session delivers it by push"
	laptop=$(git -C "$MISE_PROJECT_ROOT" rev-parse --short --verify --quiet "refs/heads/$branch" || true)
	if [[ -n $laptop && $laptop != "$after" ]]; then
		push="git push ssh://$host/~/$clone $branch:refs/heads/$branch"
		echo "clone: the laptop's $branch is at $laptop; $push delivers it"
		lag="clone at $after, not the laptop's $branch at $laptop: deliver it with $push"
	fi
elif [[ $before != "$after" ]]; then
	echo "clone: $branch fast-forwarded from $upstream, $before to $after"
	changes+=("clone pulled")
else
	echo "clone: $branch at $after, current with $upstream"
fi

# 3. The router, at the pin.
declare -A router_state
while read -r key value; do
	router_state[$key]=$value
done <<<"$(on_host clone_router_state "$clone")"
pin=${router_state[pin]}
restarted=
if [[ ${router_state[image]} == - ]]; then
	on_host clone_engines "$clone" install
	changes+=("router installed at the pin")
	restarted=yes
elif [[ ${router_state[image]} != "$pin" ]]; then
	on_host clone_engines "$clone" update
	changes+=("router moved to the pin")
	restarted=yes
elif [[ ${router_state[status]} != running ]]; then
	echo "host-update: llama on $host is ${router_state[status]}; docker start llama starts it, on $host" >&2
	exit 1
else
	echo "router: at the pin, $pin"
fi
wait_healthy 120

# 4. The preset. A router started this run read the preset as it is now.
read -r digest recorded <<<"$(on_host clone_preset_digest "$clone")"
if [[ -n $restarted ]]; then
	on_host clone_preset_record "$clone" "$digest"
	echo "preset: read by the router as it started"
elif [[ $digest != "$recorded" ]]; then
	curl -fsS --max-time 30 "$router/models?reload=1" >/dev/null
	on_host clone_preset_record "$clone" "$digest"
	if [[ $recorded == - ]]; then
		echo "preset: no record of the preset the router last read; reloaded"
	else
		echo "preset: changed since the router last read it; reloaded"
	fi
	changes+=("preset reloaded")
else
	echo "preset: unchanged"
fi

# 5. Set A, loaded: every model the preset names.
mapfile -t ids < <(on_host clone_preset "$clone" | preset_ids)
if ((${#ids[@]} == 0)); then
	echo "host-update: the preset names no model" >&2
	exit 1
fi
declare -A requested=()
deadline=$((SECONDS + timeout))
while :; do
	declare -A state=()
	while read -r id value; do
		state[$id]=$value
	done < <(curl -fsS --max-time 30 "$router/models" | model_states)
	waiting=() failed=()
	for id in "${ids[@]}"; do
		case ${state[$id]:-absent} in
		loaded) ;;
		failed) failed+=("$id") ;;
		absent)
			echo "host-update: the router doesn't list $id, which the preset names" >&2
			exit 1
			;;
		loading) waiting+=("$id") ;;
		*)
			waiting+=("$id")
			if [[ -z ${requested[$id]:-} ]]; then
				curl -fsS --max-time 30 -X POST "$router/models/load" \
					-H 'Content-Type: application/json' \
					-d "$(jq -cn --arg id "$id" '{model: $id}')" >/dev/null
				echo "models: loading $id"
				requested[$id]=yes
			fi
			;;
		esac
	done
	if ((${#failed[@]})); then
		printf 'host-update: failed to load on %s: %s\n' "$host" "${failed[@]}" >&2
		exit 1
	fi
	((${#waiting[@]})) || break
	if ((SECONDS >= deadline)); then
		printf 'host-update: not loaded on %s after %s s: %s\n' "$host" "$timeout" "${waiting[@]}" >&2
		exit 1
	fi
	sleep 5
done
if ((${#requested[@]})); then
	changes+=("${#requested[@]} model(s) loaded")
fi
echo "models: all ${#ids[@]} of set A loaded: ${ids[*]}"

# 6. The record.
echo "record:"
printf '  %-8s %s\n' build "$(on_host host_build)"
while read -r item value; do
	printf '  %-8s %s\n' "$item" "$value"
done <<<"$(on_host host_stack)"
path=$(tailscale ping "$host" 2>&1 | tail -n 1 | ping_path || true)
printf '  %-8s %s\n' network "${path:-unknown: tailscale ping $host gave no pong} (tailscale ping from the laptop)"

summary "$lag" "${changes[@]}"
