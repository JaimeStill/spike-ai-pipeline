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

slices 5/5 committed · standards ✓ · spec — · editor ✓

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
- answer: the answer line says "every image, audio window, and text prepared from a real share", not "every input of a real share", since docproc prepared 515 of ~/docs' 592 inputs.
- answer: the root names the capstone repository, `JaimeStill/data-intelligence`, without a link until it exists.
- answer: mise.toml's gb10-2 comments (`SPIKE_ROUTER_URL` and the `host-update` task) say gb10-2 left the spike, alongside the README and CLAUDE.md.

## Pending edits

- coordinator · context/ai-strategy.md: insert under "## Answers · experiment.ai":

```markdown
### Answer · experiment.ai.spike-ai-pipeline

**Question:** Can a Go model client and a message, document, and stream processing layer, built
under go-elemental, carry a transactional, evented pipeline (metadata capture, inference
extraction, and embeddings stored in PostgreSQL with pgvector for semantic search) that a browser
follows stage by stage in real time?

**Answer:** Yes for the two layers it kept: a standard-library Go client over three wire formats
and a processing layer of owned system-tool adapters carried every image, audio window, and text
prepared from a real share, ~/docs, through vision, transcription and embeddings with no request
failed; the transactional, evented pipeline in pgvector and the browser that follows it moved,
unproven, to capstone.data-intelligence.

The items keep the spike's own evidence numbers.

**1.** The client: chat with tools and a reasoning round trip, streamed and unstreamed, vision,
audio input, embeddings, and transcription each pass against set A; its retry policy is proven
only by tests. docinfer ran 3,470 images, 1,254 audio windows, and 28,937 chunk lines of ~/docs
through it in 21,634 s, 0 of 6,172 requests failed. Proven by the client's integration tier
(b11669) and a running demo, docinfer over ~/docs (b12098, 2026-11-08).

**2.** Anthropic Messages and Open Responses fit under Chat Completions' request and response
model with no change to callers; the router drops some hints on both (reasoning effort, structured
output's schema, `strict`, the `none` tool choice, `text`). Proven only by tests: the client's
integration tier (b11669, 2026-10-27 and 2026-10-28).

**3.** The client's base module is standard library only. Proven only by the check: a strict
depguard rule over every Go file.

**4.** A PDF's pages go through vision whole and as a page stream; Ghostscript wins the renderer
bake-off (0.126 s and 27 MiB a page), run as a separate program since it is AGPL, and the model
reads with reasoning off; page 7's NOFORN marking is read on no build. Proven only by tests:
processing's integration and bake-off tiers (b11669, 2026-10-29 and 2026-10-30).

**5.** Mixed: long audio cut at silences and transcribed in windows is 2.6 to 3.4 times faster
than one unsplit pass, but less accurate on many speakers (word error rate 0.298 against 0.258 on
a 4-speaker meeting); the gpt-oss seam pass is falsified, making every run worse. Proven only by
tests: processing's findings tier (b12098, 2026-11-05 and 2026-11-06).

**6.** Skype for Business conversation history in a PST passes through as conversation records,
each of 4 equal to the hand truth. Fixture only: the PST is generated, its transcript HTML
invented, and no real store read. Proven only by processing's findings tier (2026-11-07, no
model).

**9.** The dependency runs one way, processing to client: processing imports the client only in
test code and runs no inference; the root's docinfer imports both. Proven by the check, whose
depguard rule keeps the client from importing anything but the standard library, and by
docinfer's running demo (item 1); processing's rule allows the client anywhere, so the test-only
import is how processing v0.9.0 stands, not a rule. Where the layer lives is experiment.ai's
intake's decision.

**11.** ImageMagick's corrections don't help vision reliably: which help and which hurt flips with
the DPI and the build. Proven only by tests: processing's bake-off tier (b11669, 2026-10-29 and
2026-10-30).

**12.** The Open XML CLI keeps 1,768 of 1,768 elements where markitdown and xberg lose most, at a
17.2 ms median from a 28.80 MiB binary; LibreOffice's legacy round trip loses structure in .doc, a
little in .xls, nothing in .ppt, from a 1.02 GB container. Proven by office's bench (2026-10-31,
no model) and processing's integration tier (b11669, 2026-11-02).

**13.** Live media is processed as it arrives: three videos read at real time with no window,
frame, or request failed, each window transcribed 0.8 to 4.6 s after it is yielded, and a
cancellation stops the stream within 0.5 s. Proven only by tests: processing's findings tier's
paced runs (b12098, 2026-11-06).

**14.** One tool, ffprobe, runs through the exec and container runners under the same contract;
docproc wired all ten stages over ~/docs, 515 of 592 inputs prepared and 4 failed. Proven by
processing's integration tier and a running demo, docproc over ~/docs (2026-11-07, no model).

**16.** The mail CLI recovers every stored field of the generated fixtures' truth (147 of 180, 33
not stored, none wrong) and reads a 2.1 GB store in 13.27 s; the adapter places 11 of 11 messages
under the parent the hand truth gives and routes 51 of 51 attachments. Proven by mail's bench
(from 2026-11-03, no model) and processing's findings and integration tiers (b11669, 2026-11-03
and 2026-11-04).

**17.** The OneNote CLI recovers every expected field of the public fixtures (236 of 236 over the
sections) and reads a 161 MB section in 0.24 s; vision reads a page's ink at a character error
rate of 0.028 against 0.68 for OneNote's own recognition. No authored notebook exists, so no
authored hand truth. Proven by onenote's bench (2026-11-04, no model) and processing's findings
tier (b11669, 2026-11-05).

Moved to capstone.data-intelligence, not failed: 7 (the transactional pipeline into pgvector,
its events and UI), 8 (the client through a gateway), 10 (pgvector as a native-tier extension),
15 (the browser's consumption of the events), and the gates, spike-model-layout's and
spike-model-gateway's answers, which join the capstone's own (`data-intelligence.md`, "Evidence"
and "Gates").

Architect's input to the intake, not a spike finding: the architect expects the processing
library and its tools to become supplemental repositories, like sqlate and blobfs.

[The answer](https://github.com/JaimeStill/spike-ai-pipeline#the-answer) ·
[spike-ai-pipeline](https://github.com/JaimeStill/spike-ai-pipeline)
```
