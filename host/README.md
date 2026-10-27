# host

This directory runs gb10-2's llama.cpp router: spike-model-engines' llama.cpp container scripts,
copied verbatim, and this repository's set A preset.

## The copied scripts and their source

Every file here except this README and `llama-preset/models.ini` is a byte-identical copy of the
file at the same path in spike-model-engines (https://github.com/JaimeStill/spike-model-engines)
at commit `d285b06`:

- `engines`
- `engines.sh`
- `engines-llama-install`
- `engines-llama-update`
- `engines-llama-uninstall`
- `llama.env`
- `llama.sh`
- `llama-release.sh`, which `llama.sh` sources for llama.cpp's versions

spike-model-engines also runs llama.cpp natively (`llama-native.sh`, the
`engines-llama-native-*` commands, `llama-router.service`, and `llama-router.sudoers`); those files
aren't copied, since gb10-2 serves from the container.

The copies are never edited here; a change comes as a new copy from a later commit, with the commit
and file list above updated. With spike-model-engines checked out beside this repository, this
prints nothing when the copies match their source:

```sh
for f in engines engines.sh engines-llama-install engines-llama-update engines-llama-uninstall llama.env llama.sh llama-release.sh; do
  git -C ../spike-model-engines show d285b06:host/$f | diff - host/$f
done
```

`llama-preset/models.ini` is this repository's own preset, serving set A. The copied `llama.sh`
mounts the `llama-preset/` directory beside itself, so the copy reads the preset unchanged.

## Commands

Run on gb10-2 from the clone: `host/engines llama install` starts the router at the pin
(`llama.env`), `host/engines llama update` moves it to the pin, and
`host/engines llama uninstall` removes it, keeping `/srv/models`. `mise run host-update` runs
install and update itself (`README.md`, "gb10-2").
