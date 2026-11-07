# docinfer testdata

`docproc/` is a docproc run directory, schema "1" (`docs/docproc-schema.md`), written by hand
rather than by a docproc run, so it holds every case docinfer reads without a host's tools. Its
inputs, under `/srv/inputs`, are named, not kept: only the outputs are stored. It has no `index/`,
which docinfer doesn't read. Each part's SHA-256 is that of its name, and each stage directory's
digest is made up, but every `ref` names a file under `outputs/`.

- Run `019a5e2a-1b3c-7d4e-8f00-000000000001`, the first: `hello.txt`, prepared by the text stage,
  one text output.
- Run `019a5e2a-1b3c-7d4e-8f00-000000000002`, the latest, over `/srv/inputs`, ten outputs:
  - `hello.txt`, seen, its output line seen, naming the first run's copy: text;
  - `report.pdf`, prepared by the document stage: page 1's text and its image, a copy of
    `probe/testdata/shapes.png`;
  - `notes.docx`, prepared by the office stage: Markdown and a record;
  - `talk.wav`, prepared by the audio stage: its plan, a record, and three windows, 16 kHz mono
    WAVs cut from `probe/testdata/phrase.wav` at 0-2.5 s, 2-4.5 s, and 4 s to its end, each
    window's meta giving its span, cut, samples, and seams, as processing's audio stage does;
  - `blob.bin`, unrouted: no processor yet, no outputs;
  - `broken.pdf`, failed on its second page after its first page's text was prepared: one text
    output on a failed part.
