# goal · experiment.ai.spike-ai-pipeline

- **State:** building
- **Task:** answer
- **Branch:** answer

## Tasks

1. [x] host
2. [x] client-core
3. [x] host-scripts
4. [x] client-streaming
5. [x] client-messages
6. [x] client-responses
7. [x] tools
8. [x] documents
9. [x] office-cli
10. [x] office
11. [x] mail-cli
12. [x] email
13. [x] mail-mime
14. [x] email-mime
15. [x] onenote-cli
16. [x] onenote
17. [x] gb10-2-align
18. [x] audio
19. [x] streams
20. [x] formats
21. [x] docproc
22. [x] docinfer
23. [ ] answer

## Task brief · experiment.ai.spike-ai-pipeline · answer

```
Problem       The spike's work is done, but its README says "The answer: Not yet", so
              experiment.ai's intake has nothing to take in. The context page still lists
              evidence 7, 8, 10, 15 and the Gates as the spike's though they moved to
              capstone.data-intelligence; the capstone sentences and the gb10-2 section
              describe a plan and a host reservation that no longer hold; x/sys trails.
Behaviors
  1. The root requires golang.org/x/sys v0.50.0; currency reports only llama.cpp, held at
     b12098; the check passes.
  2. README "The answer" states the question, the one-line qualified-yes answer, then
     evidence 1-6, 9, 11-14, 16, 17 under the spike's own numbers. Each states its result,
     negatives and fixture-only results included, names where it is proven (test tier,
     check, or a running demo with build and date), and links its finding.
  3. Item 9 states the dependency direction (processing imports the client only in test
     code and runs no inference; docinfer imports both) and leaves the home to
     experiment.ai's intake.
  4. The answer ends with "The original evidence's other items moved:", listing 7, 8, 10,
     15 and the Gates with what each waits on in capstone.data-intelligence, linked to
     data-intelligence.md, as moved, not failed.
  5. The context page marks 7, 8, 10, 15 moved and its Gates as moved with the capstone;
     the README's, the context page's, and .claude/marathon.toml's capstone wording names
     capstone.data-intelligence and its repository.
  6. README "gb10-2" and CLAUDE.md "Host" state that gb10-2 left the spike on 2026-11-09
     for experiment.hosting's spike-model-layout, its clone removed and host-update no
     longer run; the scripts are kept as the record; the integration tier reads whatever
     router SPIKE_ROUTER_URL names.
  7. The goal record's pending edits carry ai-strategy.md's
     "### Answer · experiment.ai.spike-ai-pipeline" under "## Answers · experiment.ai", in
     the answer-section form, its answer line and numbers matching the README's, with the
     architect's expectation for the processing layer's home (supplemental repositories
     like sqlate and blobfs) marked as an input to the intake, not a spike finding.
  8. Every link the answer, moved list and pending edit add resolves.
Test seams    the root's check and currency (1); the answer and pending edit read against
              each finding they cite, every added link resolved (2-8)
Slices
  1. upgrade: x/sys v0.50.0, check passes (1)
  2. moved evidence and capstone wording (5)
  3. the answer in the README (2, 3, 4, 8)
  4. gb10-2 released: README section and CLAUDE.md line (6)
  5. the answer section in the pending edits; session brief with Sync section,
     next: none — run `marathon intake experiment.ai` (7)
Out of scope  layer repositories and their notes (the intake's); any gb10-2 step or model
              request; moving llama.cpp's pin; ai-strategy.md "Experiment" and
              "Assumptions" prose and ai-hosting.md's gb10-2 paragraph (the intake's and
              spike-model-layout's)
Door          two-way: prose and one module bump, no tag; the coordinator edit lands
              through the sync pull request
```

## Progress

slices 0/5 committed · standards — · spec — · editor —

## Decisions

The decisions of tasks 1-22 are in the record's last version:
https://github.com/JaimeStill/spike-ai-pipeline/blob/075b9a0bae37165340a534d162c7d1c5704d6320/context/goals/experiment.ai.spike-ai-pipeline.md

- answer: evidence 15 moved to capstone.data-intelligence with 7, 8 and 10; "11 to 17" in the task's roadmap summary was a slip, so the answer covers 1-6, 9, 11-14, 16 and 17.
- answer: the answer is a qualified yes, limited to the client and processing layers; the transactional pipeline, pgvector and the browser moved unproven to capstone.data-intelligence.
- answer: the answer keeps the spike's own evidence numbers, since the coordinator's notes cite them by number.
- answer: item 9 states only the dependency direction; the processing layer's home is experiment.ai's intake's. The architect expects the processing library and its tools to become supplemental repositories, like sqlate and blobfs.
- answer: the root's host prose says gb10-2 left the spike on 2026-11-09 for experiment.hosting's spike-model-layout.
- answer: no host-update this session; the task sends no model request, and gb10-2 is spike-model-layout's.
- answer: held llama.cpp at b12098: b12100 and later (latest b12211) can't resolve Google's QAT -hf tags (llama.cpp #31620, #31639).

## Pending edits

(slice 5 adds the answer section for standards-lab/context/ai-strategy.md)
