#!/usr/bin/env bash
# host-update's functions, in one place: sourced by scripts/host-update.sh on the laptop and by
# scripts/host-update-test.sh, and shipped whole to gb10-2 by host-update, which runs the host_*
# and clone_* functions there over Tailscale SSH. It defines functions only, so sourcing it runs
# nothing.
#
# The functions without a host_ or clone_ prefix are pure: each reads text on stdin (or its
# arguments) and prints text, touching nothing, so the test script checks them against the
# fixtures in scripts/testdata/host-update/.

# The NVIDIA Container Toolkit's CDI spec, the list of the driver's files it mounts into a GPU
# container, generated at boot (spike-model-engines, docs/engines/llama.cpp/setup.md, "Bring the
# host current").
HOST_CDI_SPEC=/var/run/cdi/nvidia.yaml

# ---- Pure functions ----

# The packages an `apt upgrade` would install or upgrade, one per line, read from the output of
# `apt-get --simulate --with-new-pkgs upgrade` (its "Inst" lines). That simulation is what `apt
# upgrade` does, so an upgrade phasing defers, which `apt list --upgradable` lists but `apt
# upgrade` holds back, is not counted as pending.
apt_pending() {
	sed -nE 's/^Inst ([^ ]+) .*/\1/p'
}

# Every hostPath a CDI spec names, one per line, unquoted.
cdi_host_paths() {
	sed -nE "s/^[[:space:]-]*hostPath:[[:space:]]*[\"']?([^\"']+)[\"']?[[:space:]]*$/\1/p"
}

# The driver versions a CDI spec's files are named for, one per line, sorted and distinct: the
# version a driver library ends in (libcuda.so.580.178.04) and the firmware directory's
# (/lib/firmware/nvidia/580.178.04/). A library versioned on its own, such as
# libnvidia-egl-wayland.so.1.1.23, is not a driver version and is skipped.
cdi_driver_versions() {
	cdi_host_paths |
		sed -nE -e 's#^/lib/firmware/nvidia/([0-9]{3,}\.[0-9]+(\.[0-9]+)?)/.*#\1#p' \
			-e 's#.*\.so\.([0-9]{3,}\.[0-9]+(\.[0-9]+)?)$#\1#p' |
		sort -u
}

# Reads the facts host_facts prints and, when gb10-2 needs a root step before the router may be
# touched, prints why and the exact root commands for the architect, and returns 1. Prints nothing
# and returns 0 when the host is ready. $1 names the host in what it prints.
#
# The commands come from spike-model-engines' setup (docs/engines/llama.cpp/setup.md, "Bring the
# host current"): upgrade through apt, then regenerate the CDI spec, since an upgrade that moves a
# driver library leaves the spec naming a file that is gone; and reboot when the host waits on
# one, last, since a reboot also regenerates the spec.
preflight() {
	local host=$1 key value driver='' reboot='' driver_error='' cdi_absent=''
	local -a pending=() cdi_drivers=() cdi_missing=() why=() run=()
	while read -r key value; do
		case $key in
		driver) driver=$value ;;
		driver-error) driver_error=$value ;;
		reboot-required) reboot=yes ;;
		apt-pending) pending+=("$value") ;;
		cdi-absent) cdi_absent=yes ;;
		cdi-driver) cdi_drivers+=("$value") ;;
		cdi-missing) cdi_missing+=("$value") ;;
		esac
	done

	local cdi_stale='' v
	if [[ -n $cdi_absent ]]; then
		why+=("the CDI spec $HOST_CDI_SPEC is missing")
		cdi_stale=yes
	fi
	if [[ -n $driver ]]; then
		for v in "${cdi_drivers[@]}"; do
			if [[ $v != "$driver" ]]; then
				why+=("the CDI spec names driver $v, but driver $driver runs")
				cdi_stale=yes
			fi
		done
	fi
	for v in "${cdi_missing[@]}"; do
		why+=("the CDI spec lists $v, which is missing")
		cdi_stale=yes
	done
	if ((${#pending[@]})); then
		why+=("${#pending[@]} apt upgrade(s) pending: ${pending[*]}")
		run+=("sudo apt update && sudo apt upgrade")
	fi
	if [[ -n $cdi_stale || ${#pending[@]} -gt 0 ]]; then
		run+=("sudo nvidia-ctk cdi generate --output=$HOST_CDI_SPEC")
	fi
	if [[ -n $driver_error ]]; then
		why+=("nvidia-smi fails ($driver_error), as when a driver upgrade waits on a reboot")
		reboot=yes
	elif [[ -n $reboot ]]; then
		why+=("$host waits on a reboot (/var/run/reboot-required)")
	fi
	[[ -n $reboot ]] && run+=("sudo reboot")

	((${#run[@]})) || return 0
	echo "$host needs root steps first, the architect's; the router is untouched:"
	printf '  - %s\n' "${why[@]}"
	echo "Run on $host, in order, then run mise run host-update again:"
	printf '  %s\n' "${run[@]}"
	if ((${#pending[@]})) && [[ -z $reboot ]]; then
		echo "When the upgrade leaves /var/run/reboot-required, the next run asks for sudo reboot too."
	fi
	return 1
}

# The section names of a llama.cpp models preset, one per line, but [*], the defaults every
# section takes: the ids the router lists and clients send in "model".
preset_ids() {
	sed -nE 's/^[[:space:]]*\[([^]]+)\][[:space:]]*$/\1/p' | grep -vx '\*' || true
}

# Each model the router lists, from its GET /models JSON, as "<id> <state>": loaded, loading, or
# unloaded, as the router names them, or failed when its last load failed.
model_states() {
	jq -r '.data[] | "\(.id) \(if .status.failed then "failed" else .status.value end)"'
}

# The network path in a `tailscale ping` line, as "direct, <latency>" or "DERP(<region>),
# <latency>", with no address: the line names the peer's tailnet address and, when direct, its
# endpoint, and neither is printed or recorded. Prints nothing for a line that isn't a pong.
ping_path() {
	sed -nE -e 's/^pong from .* via (DERP\([^)]*\)) in ([^ ]+)$/\1, \2/p' \
		-e 't' -e 's/^pong from .* via .* in ([^ ]+)$/direct, \1/p'
}

# Its input with every IPv4 address and every bracketed or Tailscale (fd7a:) IPv6 address
# replaced by <redacted>, so no tailnet address reaches the terminal or the record.
redact() {
	sed -E -e 's/([0-9]{1,3}\.){3}[0-9]{1,3}/<redacted>/g' \
		-e 's/\[[0-9A-Fa-f:]+\]/<redacted>/g' \
		-e 's/fd7a:[0-9A-Fa-f:]+/<redacted>/g'
}

# ---- Functions run on the host ----

# The facts preflight reads, one per line as "<key> <value>": the running driver (or the error
# nvidia-smi gives), a pending reboot, each package an apt upgrade would install, and each driver
# version the CDI spec names and each file it lists that is missing. Reads only, as the invoking
# user: the apt simulation reads the package lists as of the host's last apt update, which
# apt-daily runs.
host_facts() {
	local out p
	if out=$(nvidia-smi --query-gpu=driver_version --format=csv,noheader 2>&1); then
		echo "driver $(head -n 1 <<<"$out")"
	else
		echo "driver-error $(tr '\n' ' ' <<<"$out")"
	fi
	[[ -e /var/run/reboot-required ]] && echo "reboot-required yes"
	out=$(apt-get --simulate --with-new-pkgs upgrade 2>/dev/null)
	apt_pending <<<"$out" | sed 's/^/apt-pending /'
	if [[ -r $HOST_CDI_SPEC ]]; then
		cdi_driver_versions <"$HOST_CDI_SPEC" | sed 's/^/cdi-driver /'
		cdi_host_paths <"$HOST_CDI_SPEC" | while read -r p; do
			[[ -e $p ]] || echo "cdi-missing $p"
		done
	else
		echo "cdi-absent yes"
	fi
}

# Fast-forwards the clone at $1 (relative to the home directory) on its current branch from its
# upstream, and prints "<branch> <commit before> <commit after> <upstream, or - for none>". A
# branch with no upstream, such as a task branch not yet on the remote, is left as it is: a
# session delivers it from the laptop by push, which receive.denyCurrentBranch=updateInstead lets
# update the checked-out branch and its files, so this sets that once.
clone_pull() {
	cd "$1" || return
	[[ $(git config --get receive.denyCurrentBranch || true) == updateInstead ]] ||
		git config receive.denyCurrentBranch updateInstead
	local branch before upstream
	branch=$(git symbolic-ref --short HEAD)
	before=$(git rev-parse --short HEAD)
	if upstream=$(git rev-parse --abbrev-ref --symbolic-full-name '@{upstream}' 2>/dev/null); then
		git pull --ff-only --quiet </dev/null
	else
		upstream=-
	fi
	echo "$branch $before $(git rev-parse --short HEAD) $upstream"
}

# The router's state, from the clone at $1, as three lines: "pin <image>" (engines/llama.env),
# "image <image the llama container runs, or ->", and "status <the container's status, or ->".
clone_router_state() {
	# shellcheck source=engines/llama.env
	source "$1/engines/llama.env"
	echo "pin $LLAMA_IMAGE"
	local image=- status=-
	if [[ -n $(docker container ls --all --quiet --filter 'name=^llama$') ]]; then
		image=$(docker container inspect --format '{{.Config.Image}}' llama)
		status=$(docker container inspect --format '{{.State.Status}}' llama)
	fi
	echo "image $image"
	echo "status $status"
}

# Runs the copied engines command `engines llama <verb>` from the clone at $1, with no stdin.
clone_engines() {
	"$1/engines/engines" llama "$2" </dev/null
}

# The preset's digest and the digest of the preset the router last read, as "<current>
# <recorded, or ->". The record is .state/preset.sha256 in the clone, untracked.
clone_preset_digest() {
	local current recorded=-
	current=$(sha256sum "$1/engines/llama-preset/models.ini" | cut -d' ' -f1)
	[[ -r $1/.state/preset.sha256 ]] && recorded=$(<"$1/.state/preset.sha256")
	echo "$current $recorded"
}

# Records the digest given as the preset the router last read.
clone_preset_record() {
	mkdir -p "$1/.state"
	echo "$2" >"$1/.state/preset.sha256"
}

# The preset the router serves, from the clone at $1.
clone_preset() {
	cat "$1/engines/llama-preset/models.ini"
}

# The llama.cpp build the router runs, as its llama-server prints it.
host_build() {
	docker exec llama /app/llama-server --version 2>&1 | grep -m 1 '^version:'
}

# The host's stack, one "<item> <value>" line each, by the commands spike-model-engines' setup
# records it with (docs/engines/llama.cpp/setup.md, "Record the host stack").
host_stack() {
	# shellcheck source=/dev/null
	. /etc/os-release
	echo "os $PRETTY_NAME, DGX OS $(sed -nE 's/^DGX_OTA_VERSION="?([^"]*)"?/\1/p' /etc/dgx-release)"
	echo "kernel $(uname -r)"
	echo "driver $(nvidia-smi --query-gpu=driver_version --format=csv,noheader | head -n 1)"
	echo "cuda $(nvidia-smi | grep -o 'CUDA Version: [0-9.]*' | cut -d' ' -f3)"
	echo "docker $(docker version --format '{{.Server.Version}}')"
	echo "ctk $(nvidia-ctk --version | sed -nE 's/.*version ([0-9.]+).*/\1/p')"
}
