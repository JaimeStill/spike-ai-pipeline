# The email task's inputs from the mail CLI

The `email` task (processing's planned `email` package) adapts spike-ai-pipeline-mail v0.1.0.
The mail repository's
[README](https://github.com/JaimeStill/spike-ai-pipeline-mail/blob/main/README.md) is the CLI's
contract, with every field; the task depends on these parts of it:

- **Schema "1" line kinds:** `store` first, then for a .pst `folder` lines depth-first, each
  before its messages; `message` lines; and `skipped` lines, whose `kind` is `folder`,
  `message`, `attachment`, `contents`, or `hierarchy`. A .msg's embedded message is nested under
  its attachment record's `message`.
- **Attachment naming:** an attachment's `file` is the lowercase hex SHA-256 of its content plus
  its short stored extension, so `--attachments <dir>` holds each distinct content once.
- **Exit codes:** 0 read (records may be skipped), 1 output not written, 2 usage, 3 refused, 4
  the CLI's own fault.
- **Pin:** processing pins the CLI through mise's `github:JaimeStill/spike-ai-pipeline-mail`
  backend once v0.1.0 is released.
