# probe testdata

Media the router probe sends to a model. Both are copied unchanged from spike-harness-driver
(https://github.com/JaimeStill/spike-harness-driver), `clutch/examples/media/` at commit
`9981a3e37466e6200c13b3dfcf0d2508cc164fbc`, whose README says how each was made. They are the
files spike-model-hosting's probe sends (`hosting/probe/testdata/`), byte for byte.

- `shapes.png`: a red circle, a blue square, and a green triangle on white, 480 pixels square.
  The probe asks Gemma 4 26B-A4B to describe it.
- `phrase.wav`: one speaker saying "The access code is seven four two nine.", 16 kHz mono, a few
  seconds long. The probe sends it to Gemma 4 E4B in chat and through
  `/v1/audio/transcriptions`. It stays short because Gemma 4 E4B loops on clips longer than
  about 30 seconds.

With spike-harness-driver checked out beside this repository, this prints nothing when the copies
match their source:

```sh
for f in shapes.png phrase.wav; do
  git -C ../spike-harness-driver show 9981a3e37466e6200c13b3dfcf0d2508cc164fbc:clutch/examples/media/$f | cmp - probe/testdata/$f
done
```
