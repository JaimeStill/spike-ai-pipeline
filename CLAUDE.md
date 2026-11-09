# spike-ai-pipeline

A spike for the Standards Lab reference architecture: the root of spike-ai-pipeline, whose layers
live in spike-ai-pipeline-client, -processing, -office, -mail, and -onenote beside it in
`~/experiments/`.
`README.md` is the front page; the repository is managed with the marathon workflow, so start
from `context/README.md`.

- **Standards:** `STANDARDS.md`, with pointers into the architecture repository.
- **Check:** `mise run check` (hermetic). `mise run integration` runs the integration-tagged
  tests against whatever router `SPIKE_ROUTER_URL` names. Its default still names gb10-2's,
  which now serves spike-model-layout's preset, so results there may differ from set A.
- **Currency:** `mise run currency`.
- **Goals:** the spike is `experiment.ai.spike-ai-pipeline` in the coordinator's roadmap,
  `standards-lab/context/roadmap.toml`, and has answered (`README.md`, "The answer"). Its goal
  record was deleted at the docinfer sync and again at the answer's sync; the docinfer-era
  version is at
  https://github.com/JaimeStill/spike-ai-pipeline/blob/075b9a0bae37165340a534d162c7d1c5704d6320/context/goals/experiment.ai.spike-ai-pipeline.md.
  experiment.ai's intake, `marathon intake experiment.ai`, runs next and archives this
  repository.
- **Dependencies:** published versions only, never a `replace` directive; the layers by tag.
- **Host:** gb10-2 served set A for the spike until 2026-11-09, when it left for
  experiment.hosting's spike-model-layout, run in its own session
  (`standards-lab/context/ai-hosting.md`, "Hosts and access"). spike-model-layout's `hosts` task
  moved gb10-2's router preset mount onto spike-model-engines and removed this spike's clone from
  gb10-2, so `mise run host-update` is no longer run: it would pull the removed clone and
  re-point gb10-2's router at this spike's preset. The host scripts stay as the record of how the
  spike ran set A (`README.md`, "gb10-2").
- **References:** the repositories this spike reads are keys in the coordinator's
  `references.toml` and `references.local.toml`: tau-protocol, tau-format, tau-provider,
  tau-agent, herald, spike-harness-driver, spike-model-hosting, spike-model-engines, go-core,
  go-database, sqlate, go-web-sdk, and go-observability. The spike reads them and never writes to
  them.
