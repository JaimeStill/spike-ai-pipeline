# engines

gb10-2's llama.cpp router, from spike-model-engines' llama.cpp scripts, copied verbatim.

## Source

Every file here except this README and `llama-preset/models.ini` is a byte-identical copy of the
file at the same path in spike-model-engines (https://github.com/JaimeStill/spike-model-engines)
at commit `698f75ca8c1898671ed6256e86c1bcbaf9271ac4`:

- `engines`
- `engines.sh`
- `engines-llama-install`
- `engines-llama-update`
- `engines-llama-uninstall`
- `llama.env`
- `llama.sh`

They are never edited here; a change comes as a new copy from a later commit, with this record
updated. With spike-model-engines checked out beside this repository, this prints nothing when the
copies match their source:

```sh
for f in engines engines.sh engines-llama-install engines-llama-update engines-llama-uninstall llama.env llama.sh; do
  git -C ../spike-model-engines show 698f75ca8c1898671ed6256e86c1bcbaf9271ac4:engines/$f | diff - engines/$f
done
```

`llama-preset/models.ini` is this repository's own: set A. The copied `llama.sh` mounts the
`llama-preset/` directory beside itself, so the copy reads it without a change.

## Commands

Run on gb10-2 from the clone: `engines/engines llama install` starts the router at the pin
(`llama.env`), `engines/engines llama update` moves it to the pin, and
`engines/engines llama uninstall` removes it, keeping `/srv/models`.
