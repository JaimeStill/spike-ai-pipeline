# spike-ai-pipeline

A spike for the Standards Lab reference architecture: the root of spike-ai-pipeline, whose layers
live in spike-ai-pipeline-client, -processing, -office, -mail, and -onenote beside it in
`~/experiments/`.
`README.md` is the front page; the repository is managed with the marathon workflow, so start
from `context/README.md`.

- **Standards:** `STANDARDS.md`, with pointers into the architecture repository.
- **Check:** `mise run check` (hermetic). `mise run integration` runs the integration-tagged
  tests against the router `SPIKE_ROUTER_URL` names, gb10-2's by default.
- **Currency:** `mise run currency`.
- **Goals:** the spike is planned in the coordinator's roadmap,
  `standards-lab/context/roadmap.toml`, under `experiment.ai.spike-ai-pipeline`. Its goal record
  was deleted at the docinfer sync; its last version is at
  https://github.com/JaimeStill/spike-ai-pipeline/blob/075b9a0bae37165340a534d162c7d1c5704d6320/context/goals/experiment.ai.spike-ai-pipeline.md. A `marathon plan` session starts the next one.
- **Dependencies:** published versions only, never a `replace` directive; the layers by tag.
- **Host:** gb10-2, reserved for this spike (`standards-lab/context/ai-hosting.md`, "Hosts and
  access"). Sessions run steps that need no root over Tailscale SSH; root stays the architect's,
  run from exact commands. gb10-2 only serves models. Each session starts with
  `mise run host-update`, which brings gb10-2 current and loads set A (`README.md`, "gb10-2").
- **References:** the repositories this spike reads are keys in the coordinator's
  `references.toml` and `references.local.toml`: tau-protocol, tau-format, tau-provider,
  tau-agent, herald, spike-harness-driver, spike-model-hosting, spike-model-engines, go-core,
  go-database, sqlate, go-web-sdk, and go-observability. The spike reads them and never writes to
  them.
