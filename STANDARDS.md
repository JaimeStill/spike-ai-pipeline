# Standards

The judgement calls the check can't enforce, one line each, for the standards-reviewer. What
`mise run check` enforces is never restated here.

The architecture pages that apply, in the architecture repository:

- `principles/minimal-footprint.md`
- `principles/composition-root.md`
- `principles/rolling-currency.md`
- `principles/service-tiers.md`
- `standards/go-elemental/principles/topology-and-naming.md`
- `standards/go-elemental/principles/tests-and-docs.md`

## Hosts and models

- No script, document, or committed file carries a host's tailnet address; hosts are named by
  their MagicDNS short name.
- Every command run on gb10-2 appears in the docs exactly as run, and every measurement names the
  llama.cpp build, gb10-2's stack, and the laptop's network path.
- A model of Chinese origin is chosen only when no alternative serves, and its entry records why
  (`standards-lab/context/ai-strategy.md`, "Constraint: model origin").
