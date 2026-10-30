#!/usr/bin/env bash
# Tests host-update's pure functions (scripts/host-update-lib.sh) against the fixtures in
# scripts/testdata/host-update/: the preflight's reading of apt, the CDI spec, and the host's
# facts; the router's next step; the preset, model-state, poll, network-path, and redaction
# readers; and the last line. It touches no host and reads no network. `mise run check` runs it;
# it prints each failure and exits 1 on any.
set -euo pipefail

dir=$(dirname "$(readlink -f -- "${BASH_SOURCE[0]}")")
# shellcheck source=scripts/host-update-lib.sh
source "$dir/host-update-lib.sh"
data=$dir/testdata/host-update

failures=0

# Compares what a command printed ($2) with what it should have ($3), naming the case ($1).
expect() {
	if [[ $2 != "$3" ]]; then
		printf 'host-update-test: %s\n  got:\n%s\n  want:\n%s\n' "$1" "$2" "$3"
		failures=$((failures + 1))
	fi
}

# Runs preflight on a facts fixture and gives its output and status as one string.
preflight_of() {
	local out status=0
	out=$(preflight gb10-2 <"$data/$1") || status=$?
	printf '%s\nstatus %s' "$out" "$status"
}

expect "apt_pending counts the simulation's Inst lines, not deferred phasing" \
	"$(apt_pending <"$data/apt-pending.txt")" \
	"linux-image-7.0.0-1021-nvidia
libnvidia-compute-580
nvidia-container-toolkit"
expect "apt_pending is empty when nothing would upgrade" \
	"$(apt_pending <"$data/apt-current.txt")" ""

expect "cdi_host_paths lists every hostPath" \
	"$(cdi_host_paths <"$data/cdi-current.yaml")" \
	"/run/nvidia-persistenced/socket
/usr/bin/nvidia-smi
/usr/lib/aarch64-linux-gnu/libcuda.so.580.178.04
/usr/lib/aarch64-linux-gnu/libnvidia-ml.so.580.178.04
/usr/lib/aarch64-linux-gnu/libnvidia-egl-wayland.so.1.1.23
/usr/lib/aarch64-linux-gnu/libnvidia-nvvm70.so.4
/lib/firmware/nvidia/580.178.04/gsp_ga10x.bin"
expect "cdi_host_paths unquotes a quoted hostPath" \
	"$(printf '%s\n' '- hostPath: "/usr/bin/nvidia-smi"' "  hostPath: '/dev/nvidia0'" | cdi_host_paths)" \
	"/usr/bin/nvidia-smi
/dev/nvidia0"
expect "cdi_driver_versions skips libraries versioned on their own" \
	"$(cdi_driver_versions <"$data/cdi-current.yaml")" "580.178.04"
expect "cdi_driver_versions reads a stale spec's driver" \
	"$(cdi_driver_versions <"$data/cdi-stale.yaml")" "580.95.05"

expect "preflight passes a ready host silently" \
	"$(preflight_of facts-ready.txt)" "
status 0"
expect "preflight stops on upgrades and a stale CDI spec, with the root commands" \
	"$(preflight_of facts-stale.txt)" \
	"gb10-2 needs root steps first, the architect's; the router is untouched:
  - the CDI spec names driver 580.95.05, but driver 580.178.04 runs
  - the CDI spec lists /usr/lib/aarch64-linux-gnu/libcuda.so.580.95.05, which is missing
  - 2 apt upgrade(s) pending: libnvidia-compute-580 nvidia-container-toolkit
Run on gb10-2, in order, then run mise run host-update again:
  sudo apt update && sudo apt upgrade
  sudo nvidia-ctk cdi generate --output=/var/run/cdi/nvidia.yaml
When the upgrade leaves /var/run/reboot-required, the next run asks for sudo reboot too.
status 1"
expect "preflight stops on a pending reboot" \
	"$(preflight_of facts-reboot.txt)" \
	"gb10-2 needs root steps first, the architect's; the router is untouched:
  - nvidia-smi fails (Failed to initialize NVML: Driver/library version mismatch), as when a driver upgrade waits on a reboot
Run on gb10-2, in order, then run mise run host-update again:
  sudo reboot
status 1"
expect "preflight asks for a CDI spec that is missing" \
	"$(preflight_of facts-cdi-absent.txt)" \
	"gb10-2 needs root steps first, the architect's; the router is untouched:
  - the CDI spec /var/run/cdi/nvidia.yaml is missing
Run on gb10-2, in order, then run mise run host-update again:
  sudo nvidia-ctk cdi generate --output=/var/run/cdi/nvidia.yaml
status 1"

expect "preset_ids names each section but [*]" \
	"$(preset_ids <"$data/preset.ini")" \
	"org/first-GGUF:Q8_0
org/second-GGUF:MXFP4"

expect "model_states reads each state, failed over unloaded" \
	"$(model_states <"$data/models.json")" \
	"mrutkows/granite-embedding-english-r2-GGUF:BF16 loaded
google/gemma-4-26B-A4B-it-qat-q4_0-gguf:Q4_0 loading
ggml-org/gemma-4-E4B-it-GGUF:Q8_0 unloaded
ggml-org/gpt-oss-120b-GGUF:MXFP4 failed"

expect "router_action leaves a running router at the pin with a live preset mount alone" \
	"$(router_action <"$data/router-live.txt")" "current"
expect "router_action restarts a running router whose preset mount is dead" \
	"$(router_action <"$data/router-dead-mount.txt")" "restart"
expect "router_action moves a preset mount the clone's scripts left behind, never restarting it" \
	"$(router_action <"$data/router-moved.txt")" "move"
expect "router_action moves a preset mount sourced elsewhere even while it sees a preset" \
	"$(sed 's/^mount dead$/mount live/' "$data/router-moved.txt" | router_action)" "move"
expect "router_action moves a running router with no preset mount" \
	"$(sed 's/^source .*/source -/' "$data/router-live.txt" | router_action)" "move"
expect "router_action leaves a stopped router stopped" \
	"$(router_action <"$data/router-stopped.txt")" "stopped"
expect "router_action installs a router that is absent" \
	"$(router_action <"$data/router-absent.txt")" "install"
expect "router_action moves a router off another image before minding its mount" \
	"$(router_action <"$data/router-other-image.txt")" "update"

expect "poll_states gives each id its state, and absent when the listing lacks it" \
	"$(poll_states ok mrutkows/granite-embedding-english-r2-GGUF:BF16 ggml-org/gpt-oss-120b-GGUF:MXFP4 \
		org/missing-GGUF:Q8_0 <"$data/models.json")" \
	"mrutkows/granite-embedding-english-r2-GGUF:BF16 loaded
ggml-org/gpt-oss-120b-GGUF:MXFP4 failed
org/missing-GGUF:Q8_0 absent"
expect "poll_states keeps waiting on a poll that failed, never absent" \
	"$(poll_states failed ggml-org/gpt-oss-120b-GGUF:MXFP4 org/missing-GGUF:Q8_0 </dev/null)" \
	"ggml-org/gpt-oss-120b-GGUF:MXFP4 unanswered
org/missing-GGUF:Q8_0 unanswered"
expect "poll_states keeps waiting on a listing that isn't JSON" \
	"$(poll_states ok ggml-org/gpt-oss-120b-GGUF:MXFP4 <"$data/models-not-json.txt")" \
	"ggml-org/gpt-oss-120b-GGUF:MXFP4 unanswered"

expect "ping_path reads a direct path without its addresses" \
	"$(ping_path <<<'pong from gb10-2 (100.64.0.7) via 192.168.1.20:41641 in 2ms')" "direct, 2ms"
expect "ping_path reads a DERP relay" \
	"$(ping_path <<<'pong from gb10-2 (100.64.0.7) via DERP(nyc) in 31ms')" "DERP(nyc), 31ms"
expect "ping_path prints nothing for a line that isn't a pong" \
	"$(ping_path <<<'timeout waiting for ping reply')" ""
expect "ping_path keeps a latency under a millisecond" \
	"$(ping_path <<<'pong from gb10-2 (100.64.0.7) via 192.168.1.20:41641 in 870µs')" "direct, 870µs"
expect "ping_path reads tailscale's 0s as under a millisecond" \
	"$(ping_path <<<'pong from gb10-2 (100.64.0.7) via 192.168.1.20:41641 in 0s')" "direct, under 1ms"

expect "redact removes IPv4 and IPv6 addresses" \
	"$(redact <<<'on 100.64.0.7:8080 and [fd7a:115c:a1e0::1]:8080, fd7a:115c:a1e0::2; build 11669')" \
	"on <redacted>:8080 and <redacted>:8080, <redacted>; build 11669"

lag="clone at 62d32e2, not the laptop's host at bb39f4b: deliver it with git push ssh://gb10-2/~/repos/spike-ai-pipeline host:refs/heads/host"
expect "summary calls a current clone current" \
	"$(summary "")" \
	"host-update: nothing changed: clone current, router at the pin, preset unchanged, set A loaded"
expect "summary doesn't call a clone apart from the laptop current" \
	"$(summary "$lag")" \
	"host-update: nothing changed, but $lag; router at the pin, preset unchanged, set A loaded"
expect "summary lists the changes" \
	"$(summary "" "clone pulled" "preset reloaded")" \
	"host-update: changed: clone pulled, preset reloaded"
expect "summary lists the changes and the clone apart from the laptop" \
	"$(summary "$lag" "preset reloaded")" \
	"host-update: changed: preset reloaded; $lag"

if ((failures)); then
	echo "host-update-test: $failures failure(s)"
	exit 1
fi
echo "host-update-test: ok"
