# spike-ai-pipeline

A spike for the Standards Lab reference architecture: the root of spike-ai-pipeline, whose layers
live in spike-ai-pipeline-client, -processing, and -office beside it in `~/experiments/`.
`README.md` is the front page; the repository is managed with the marathon workflow, so start
from `context/README.md`.

- **Standards:** `STANDARDS.md`, with pointers into the architecture repository.
- **Check:** `mise run check` (hermetic). `mise run integration` runs the integration-tagged
  tests against the router `SPIKE_ROUTER_URL` names, gb10-2's by default.
- **Currency:** `mise run currency`.
- **Goals:** the spike's tasks live in the coordinator's roadmap,
  `standards-lab/context/roadmap.toml`, under `experiment.ai.spike-ai-pipeline`. Its goal record
  is `context/goals/experiment.ai.spike-ai-pipeline.md`.
- **Dependencies:** published versions only, never a `replace` directive; the layers by tag.
- **Host:** gb10-2, reserved for this spike (`standards-lab/context/ai-hosting.md`, "Hosts and
  access"). Sessions run steps that need no root over Tailscale SSH; root stays the architect's,
  run from exact commands. gb10-2 only serves models.
- **References:** the repositories this spike reads are keys in the coordinator's
  `references.toml` and `references.local.toml`: tau-protocol, tau-format, tau-provider,
  tau-agent, herald, spike-harness-driver, spike-model-hosting, spike-model-engines, go-core,
  go-database, sqlate, go-web-sdk, and go-observability. The spike reads them and never writes to
  them.
