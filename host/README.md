# host

This directory runs gb10-2's llama.cpp router: spike-model-engines' llama.cpp container scripts,
copied verbatim, and this repository's set A preset.

## The copied scripts and their source

Every file here except this README, `llama.env`, and `llama-preset/models.ini` is a
byte-identical copy of the file at the same path in spike-model-engines
(https://github.com/JaimeStill/spike-model-engines) at commit `dfc1d3d`, seven files:

- `engines`
- `engines.sh`
- `engines-llama-install`
- `engines-llama-update`
- `engines-llama-uninstall`
- `llama.sh`
- `llama-release.sh`, which `llama.sh` sources for llama.cpp's versions and each host class's pin

spike-model-engines also runs llama.cpp natively (`llama-native.sh`, the
`engines-llama-native-*` commands, `llama-router.service`, and `llama-router.sudoers`); those files
aren't copied, since gb10-2 serves from the container.

The copies are never edited here; a change comes as a new copy from a later commit, with the commit
and file list above updated. With spike-model-engines checked out beside this repository, this
prints nothing when the copies match their source:

```sh
for f in engines engines.sh engines-llama-install engines-llama-update engines-llama-uninstall llama.sh llama-release.sh; do
  git -C ../spike-model-engines show dfc1d3d:host/$f | diff - host/$f
done
```

Two files are this repository's own:

- `llama.env`, the pins, is dfc1d3d's with gb10's pin held at b12098 rather than upstream's
  b12063, every other variable as upstream sets it. gb10-2 needs b12098's fix for the idle-slot
  stall (llama.cpp #31391), and b12100 and later match an `-hf` quant tag only after `-`, `.` or
  `/` (#31620, #31639), so set A's vision model, `google/gemma-4-26B-A4B-it-qat-q4_0-gguf:Q4_0`,
  can't resolve on them; the file's comment says so. Once spike-model-engines' gb10 pin reaches
  the root's, a later copy can make `llama.env` a copy again.
- `llama-preset/models.ini`, the preset serving set A. The copied `llama.sh` mounts the
  `llama-preset/` directory beside itself, so the copy reads the preset unchanged.

## Commands

Run on gb10-2 from the clone: `host/engines llama install` starts the router at gb10's pin
(`llama.env`), `host/engines llama update` moves it to the pin, and
`host/engines llama uninstall` removes it, keeping `/srv/models`. `mise run host-update` runs
install and update itself (`README.md`, "gb10-2").
