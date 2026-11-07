// Package chunk is docinfer's chunker: it cuts one text into chunks of at most a bound of
// characters, runes, each a byte span of the text, for the embeddings capability to embed.
//
// A text is cut along its structure, coarsest first. A text within the bound is one chunk.
// Otherwise it is cut before each Markdown heading, an ATX heading line ("#" to "######" and a
// space) outside a fenced code block, and the sections are packed greedily, each run of whole
// sections that fits the bound one chunk; a section that doesn't fit alone is cut the same way at
// the next level: at blank lines into paragraphs, then into sentences, after ".", "!", or "?" and
// any closing quote or bracket before whitespace, or "。", "！", or "？", and at each line break,
// so a list's items part without punctuation; and last, a hard cut every bound runes.
//
// Each chunk is trimmed of the white space at its ends, and its span is that of the trimmed text,
// so a chunk's Text is always the text's [Start, End) bytes. No chunk overlaps another, and no
// cut falls inside a UTF-8 character: an invalid byte counts as one character, as range over a
// string has it. An empty or white-space-only text, or stretch of one, yields no chunk. The same
// text and bound always give the same chunks.
//
// [Version] names the algorithm: docinfer records it in its manifest and keys its held
// embeddings by it, so a change to how texts are cut must change it.
//
// The package exports:
//
//   - [Split], which cuts a text into [Chunk]s
//   - [Version], the chunker's version
package chunk
