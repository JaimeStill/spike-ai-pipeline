# spike-ai-pipeline

A spike for the Standards Lab reference architecture. It asks whether a Go model client and a
message, document, and stream processing layer, built under go-elemental, can carry a pipeline
that a browser follows live: transactional and evented, capturing metadata, extracting data by
inference, and storing embeddings in PostgreSQL with pgvector for semantic search, each stage
emitting the events the browser follows.

Its answer is evidence for standards-lab's experiment.ai intake; the spike decides nothing
itself. It informs go-ai's model-client surface and its wire-format layer, where the processing
layer lives, `v1.ai.local-models`, and v1.ai's library and service tasks.

## Repositories

The layers live in repositories of their own, checked out beside this one in `~/experiments/`,
each released by tag:

- [spike-ai-pipeline-client](https://github.com/JaimeStill/spike-ai-pipeline-client): the model
  client.
- [spike-ai-pipeline-processing](https://github.com/JaimeStill/spike-ai-pipeline-processing): the
  processing layer and its system-tool adapters.
- [spike-ai-pipeline-office](https://github.com/JaimeStill/spike-ai-pipeline-office): the Open XML
  CLI for office documents, a system tool processing runs.

This repository holds the spike's record, gb10-2's host scripts, the consumer check that builds
on the layers by tag, and, once its gates clear, the capstone: a backend on the layers and a web
UI on experiment.html's stack.

## The question

See `context/README.md`, "The question", and "The evidence".

## The answer

Not yet.
