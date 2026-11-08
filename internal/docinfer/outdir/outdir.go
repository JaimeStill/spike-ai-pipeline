package outdir

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path"
	"path/filepath"

	"github.com/JaimeStill/spike-ai-pipeline/internal/rundir"
	"github.com/JaimeStill/spike-ai-pipeline/internal/schema"
)

// Version is the schema of docinfer's directory: its layout, manifest, chunk lines, and held
// results.
const Version = "1"

// The directory's subdirectories.
const (
	resultsDir = "results"
	runsDir    = "runs"
	textsDir   = "texts"
)

// Dir is docinfer's directory, the one --out names: its held results and its runs.
type Dir struct {
	root string
}

// Open opens docinfer's directory at root, making it when it doesn't exist.
func Open(root string) (*Dir, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, err
	}
	return &Dir{root: abs}, nil
}

// OpenExisting opens docinfer's directory at root, which must exist: a reader, such as the report
// command, opens it so, and makes nothing.
func OpenExisting(root string) (*Dir, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("%s is no directory", abs)
	}
	return &Dir{root: abs}, nil
}

// Root returns the directory's absolute path.
func (d *Dir) Root() string { return d.root }

// Key returns the key a result is held under: the lowercase hex SHA-256 of the capability, the
// model, the settings that change the result, such as a prompt or the chunk settings, and the
// SHA-256 of the content the result is of, each length-prefixed so no two tuples share a key.
func Key(capability, model, settings, content string) string {
	h := sha256.New()
	for _, s := range []string{capability, model, settings, content} {
		_, _ = fmt.Fprintf(h, "%d:%s\n", len(s), s) // a hash.Hash never fails a write
	}
	return hex.EncodeToString(h.Sum(nil))
}

// heldPath returns the held result key of capability's file, relative to the directory:
// results/<capability>/<first two hex digits>/<key>.json.
func heldPath(capability, key string) string {
	return path.Join(resultsDir, capability, key[:2], key+".json")
}

// Held reads the result key of capability into v, and reports whether one is held. A held file
// that can't be decoded is an error, not a miss.
func (d *Dir) Held(capability, key string, v any) (bool, error) {
	p := filepath.Join(d.root, filepath.FromSlash(heldPath(capability, key)))
	b, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return false, fmt.Errorf("%s: %w", p, err)
	}
	return true, nil
}

// Hold writes v as the result key of capability, whole, replacing any held before, and returns
// its path relative to the directory.
func (d *Dir) Hold(capability, key string, v any) (string, error) {
	rel := heldPath(capability, key)
	return rel, rundir.WriteFile(filepath.Join(d.root, filepath.FromSlash(rel)), func(w io.Writer) error {
		return encoder(w).Encode(v)
	})
}

// textPath returns the path of the text whose SHA-256 is sum, relative to the directory:
// texts/<first two hex digits>/<sum>.txt.
func textPath(sum string) string {
	return path.Join(textsDir, sum[:2], sum+".txt")
}

// WriteText writes text whole as a file of its own, named by its content, and returns its path
// relative to the directory and its SHA-256. A text written before is written again, the same.
func (d *Dir) WriteText(text string) (rel, sum string, err error) {
	h := sha256.Sum256([]byte(text))
	sum = hex.EncodeToString(h[:])
	rel = textPath(sum)
	return rel, sum, rundir.WriteFile(filepath.Join(d.root, filepath.FromSlash(rel)), func(w io.Writer) error {
		_, err := io.WriteString(w, text)
		return err
	})
}

// ReadText reads the text WriteText wrote at rel, a path relative to the directory, and checks it
// against sum, its SHA-256 when it was written: a text that is gone or changed is an error.
func (d *Dir) ReadText(rel, sum string) (string, error) {
	b, err := os.ReadFile(filepath.Join(d.root, filepath.FromSlash(rel)))
	if err != nil {
		return "", err
	}
	if h := sha256.Sum256(b); hex.EncodeToString(h[:]) != sum {
		return "", fmt.Errorf("%s: its SHA-256 isn't %s, the one it was written with", rel, sum)
	}
	return string(b), nil
}

// RunPath returns the file name of run id's directory, relative to the directory:
// runs/<id>/<name>.
func RunPath(id, name string) string { return path.Join(runsDir, id, name) }

// WriteManifest writes m as its run's manifest, runs/<run id>/manifest.json, whole, replacing the
// one written before.
func (d *Dir) WriteManifest(m Manifest) error {
	_, err := rundir.WriteRunFile(d.root, m.Run, "manifest.json", func(w io.Writer) error {
		enc := encoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(m)
	})
	return err
}

// WriteChunks writes lines as run id's chunk record, runs/<id>/chunks.jsonl, one JSON object a
// line, whole, and returns its path relative to the directory.
func (d *Dir) WriteChunks(id string, lines []Line) (string, error) {
	return rundir.WriteRunFile(d.root, id, "chunks.jsonl", func(w io.Writer) error {
		buf := bufio.NewWriter(w)
		enc := encoder(buf)
		for _, l := range lines {
			if err := enc.Encode(l); err != nil {
				return err
			}
		}
		return buf.Flush()
	})
}

// encoder returns a JSON encoder to w that leaves <, >, and & as they are.
func encoder(w io.Writer) *json.Encoder {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc
}

// ReadChunks reads run id's chunk record.
func (d *Dir) ReadChunks(id string) ([]Line, error) {
	f, err := os.Open(filepath.Join(d.root, filepath.FromSlash(RunPath(id, "chunks.jsonl"))))
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }() // read only: a close error loses nothing
	var lines []Line
	dec := json.NewDecoder(f)
	for {
		var l Line
		if err := dec.Decode(&l); errors.Is(err, io.EOF) {
			return lines, nil
		} else if err != nil {
			return nil, fmt.Errorf("chunks.jsonl: line %d: %w", len(lines)+1, err)
		}
		lines = append(lines, l)
	}
}

// ReadManifest reads run id's manifest.
func (d *Dir) ReadManifest(id string) (Manifest, error) {
	var m Manifest
	b, err := os.ReadFile(filepath.Join(d.root, filepath.FromSlash(RunPath(id, "manifest.json"))))
	if err != nil {
		return m, err
	}
	return m, json.Unmarshal(b, &m)
}

// EncodeVector returns v as standard base64 of its little-endian float32s, as the embeddings
// endpoint sends it.
func EncodeVector(v []float32) string {
	b := make([]byte, 4*len(v))
	for i, f := range v {
		binary.LittleEndian.PutUint32(b[4*i:], math.Float32bits(f))
	}
	return base64.StdEncoding.EncodeToString(b)
}

// DecodeVector decodes a vector [EncodeVector] encoded.
func DecodeVector(s string) ([]float32, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	if len(b)%4 != 0 {
		return nil, fmt.Errorf("vector of %d bytes isn't whole float32s", len(b))
	}
	v := make([]float32, len(b)/4)
	for i := range v {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[4*i:]))
	}
	return v, nil
}

// Manifest is one docinfer run's manifest, runs/<run id>/manifest.json, written as the run
// starts and again, whole, as it ends.
type Manifest struct {
	// Schema is [Version].
	Schema string `json:"schema"`
	// Run is the run's id, a version 7 UUID.
	Run   string      `json:"run"`
	Start schema.Time `json:"start"`
	// End is absent while the run runs, or when it never ended.
	End schema.Time `json:"end,omitzero"`
	// Docproc is the docproc run read.
	Docproc Docproc `json:"docproc"`
	// Capabilities are the capabilities the run ran, in the order it ran them.
	Capabilities []Capability `json:"capabilities"`
	// Skipped are the capabilities it didn't run, and why.
	Skipped []Skipped `json:"skipped,omitempty"`
	// Space is the embedding space of the run's vectors; absent until the run ends.
	Space *Space `json:"embedding_space,omitempty"`
	// Chunks is the run's chunk record, relative to the directory: runs/<run id>/chunks.jsonl.
	// Absent until it is written.
	Chunks string `json:"chunks,omitzero"`
	// Transcripts are the transcripts the run joined, one per part and stage with audio windows,
	// in the record's order. Absent when transcription was skipped, or until the run joins them.
	Transcripts []Transcript `json:"transcripts,omitempty"`
	// Failures are the image output lines whose image vision refused and the audio window
	// output lines whose window transcription refused, each with why, vision's first, each in
	// the record's order. A refused image or window has no text and so no chunk line; this is
	// where a reader, such as the report, finds it. Absent when nothing was refused.
	Failures []Failure `json:"failures,omitempty"`
	Counts   Counts    `json:"counts"`
	// Error is what stopped the run. Absent when nothing did.
	Error string `json:"error,omitzero"`
}

// Failure is one output line a capability refused: the capability, vision or transcription, the
// input, the output's stored file, relative to the docproc run directory, and the endpoint's
// reply. An image or window under two output lines is two failures.
type Failure struct {
	Capability string `json:"capability"`
	Input      string `json:"input"`
	Ref        string `json:"ref"`
	Error      string `json:"error"`
}

// Docproc names the docproc run a docinfer run read: its run directory, absolute, and its id.
type Docproc struct {
	Dir string `json:"dir"`
	Run string `json:"run"`
}

// Capability is one capability a run ran: its endpoint, by name and base URL, its model, and
// its settings, by their configuration names.
type Capability struct {
	Name     string         `json:"name"`
	Endpoint string         `json:"endpoint"`
	BaseURL  string         `json:"base_url"`
	Model    string         `json:"model"`
	Settings map[string]any `json:"settings"`
}

// Skipped is a capability a run didn't run, and why.
type Skipped struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// UnitTolerance is how far a vector's Euclidean norm may be from 1 for [Space.Unit] to count it
// unit length: |norm - 1| <= UnitTolerance.
const UnitTolerance = 1e-3

// Space is the embedding space of a run's vectors: the endpoint and model that made them, their
// dimensions, whether every one is unit length within Tolerance, the least and greatest norm,
// and the chunk settings the texts were cut with. The vectors are stored as the endpoint
// returned them, never normalized.
type Space struct {
	Endpoint   string `json:"endpoint"`
	Model      string `json:"model"`
	Dimensions int    `json:"dimensions"`
	// Vectors counts the distinct vectors measured.
	Vectors   int     `json:"vectors"`
	Unit      bool    `json:"unit_length"`
	Tolerance float64 `json:"tolerance"`
	MinNorm   float64 `json:"min_norm"`
	MaxNorm   float64 `json:"max_norm"`
	// ChunkSize is the bound each chunk was cut to, in characters.
	ChunkSize int `json:"chunk_size"`
	// Chunker is the chunker's version, chunk.Version.
	Chunker string `json:"chunker"`
}

// Counts is what a run read, cut, and sent.
type Counts struct {
	// Outputs counts the docproc run's output lines.
	Outputs int `json:"outputs"`
	// Texts counts the text and Markdown output lines embedded, one per line.
	Texts int `json:"texts"`
	// Metadata counts the metadata texts, one per input with file metadata.
	Metadata int `json:"metadata"`
	// Chunks counts the distinct chunks, by content.
	Chunks int `json:"chunks"`
	// Lines counts the chunk record's lines, one per chunk per place it occurs.
	Lines int `json:"lines"`
	// Held counts the distinct chunks whose result an earlier run held.
	Held int `json:"held"`
	// Embedded counts the distinct chunks this run embedded.
	Embedded int `json:"embedded"`
	// Rejected counts the distinct chunks the endpoint rejected, this run or held.
	Rejected int `json:"rejected"`
	// Requests counts the embeddings requests this run sent.
	Requests int `json:"requests"`
	// Images counts the docproc run's image output lines, whether vision ran or was skipped.
	Images int `json:"images"`
	// Vision counts what vision did with the images. Absent when it was skipped.
	Vision *VisionCounts `json:"vision,omitempty"`
	// Windows counts the docproc run's audio window output lines, an audio part's or a video's,
	// whether transcription ran or was skipped.
	Windows int `json:"windows"`
	// Transcription counts what transcription did with the windows. Absent when it was skipped.
	Transcription *TranscriptionCounts `json:"transcription,omitempty"`
	// Tokens counts the tokens each capability's requests took, this run's only.
	Tokens Tokens `json:"tokens"`
}

// Tokens is the tokens each capability's requests took in one run, as the endpoint reported
// them in each reply it answered: a held result took none in this run, and a refusal or a
// rejection carries no usage, so neither adds any.
type Tokens struct {
	// Vision is vision's tokens. Absent when it was skipped.
	Vision *Usage `json:"vision,omitempty"`
	// Transcription is transcription's tokens. Absent when it was skipped.
	Transcription *Usage `json:"transcription,omitempty"`
	// Embeddings is the embeddings' tokens; an embedding has no output, so its Output is 0.
	Embeddings Usage `json:"embeddings"`
}

// Usage is the tokens a capability's requests took, summed over each reply: Input, the inputs'
// (prompt, image, or clip, and instructions), Output, the generated, and Total, as the endpoint
// reports it, or, where its format reports none, Input and Output together. An endpoint that
// reports no usage leaves each 0.
type Usage struct {
	Input  int `json:"input"`
	Output int `json:"output"`
	Total  int `json:"total"`
}

// TranscriptionCounts is what a run's transcription did: the windows it saw, by content, what
// became of each, and the transcripts it joined of them.
type TranscriptionCounts struct {
	// Distinct counts the distinct windows, by content; a window under two output lines is sent
	// once.
	Distinct int `json:"distinct"`
	// Held counts the distinct windows whose result an earlier run held.
	Held int `json:"held"`
	// Transcribed counts the distinct windows this run sent and the model transcribed.
	Transcribed int `json:"transcribed"`
	// Failed counts the distinct windows the endpoint refused, this run or held; each joins its
	// transcript as an empty one.
	Failed int `json:"failed"`
	// Requests counts the transcription requests this run sent.
	Requests int `json:"requests"`
	// Transcripts counts the joined transcripts embedded, one per part and stage with windows;
	// an empty transcript is one, cut to no chunk.
	Transcripts int `json:"transcripts"`
	// SeamsMatched counts the seams, in every transcript, whose windows' words aligned, so they
	// were spliced; SeamsUnmatched those whose windows were concatenated whole.
	SeamsMatched   int `json:"seams_matched"`
	SeamsUnmatched int `json:"seams_unmatched"`
}

// Transcript is one transcript a run joined: the part and stage whose windows it is of, the
// windows in plan order, the stored joined text, and how each seam between neighbouring windows
// was merged.
type Transcript struct {
	// Input is the input's absolute path, as docproc records it.
	Input string `json:"input"`
	Part  *Part  `json:"part,omitempty"`
	// Stage is the name of the stage that cut the windows, such as "audio" or "video".
	Stage string `json:"stage,omitzero"`
	// Windows are the windows' stored files, relative to the docproc run directory, in plan
	// order; a window the plan has but the record doesn't, one its stage failed to cut, is "".
	Windows []string `json:"windows"`
	// Failed are the plan indexes of the windows that joined as empty: refused, or absent.
	Failed []int `json:"failed,omitempty"`
	// File is the joined text's path, relative to docinfer's directory, and SHA256 its SHA-256.
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
	// Seams are the seams between the windows, one fewer than the windows: Seams[i] is window
	// i's with window i+1.
	Seams []Seam `json:"seams"`
}

// Seam is how one seam of a transcript was merged, as processing's audio.Join reports it.
type Seam struct {
	// Matched is whether the windows' words aligned on at least 2 words, so they were spliced;
	// an unmatched seam's windows are concatenated whole.
	Matched bool `json:"matched"`
	// Matches is how many words the alignment matched, and Overlap how many it overlapped; both
	// 0 for an unmatched seam.
	Matches int `json:"matches"`
	Overlap int `json:"overlap"`
	// Position is the index, in the transcript's words, of the first word the later window gave.
	Position int `json:"position"`
}

// VisionCounts is what a run's vision did: the images it saw, by content, and what became of
// each.
type VisionCounts struct {
	// Images counts the distinct images, by content; an image under two output lines is sent
	// once.
	Images int `json:"images"`
	// Held counts the distinct images whose result an earlier run held.
	Held int `json:"held"`
	// Described counts the distinct images this run sent and the model answered.
	Described int `json:"described"`
	// Incomplete counts the distinct images whose text the model stopped before finishing,
	// such as at the output cap, this run or held; the text is kept and embedded.
	Incomplete int `json:"incomplete"`
	// Empty counts the incomplete images whose text is empty: the model stopped before it gave
	// any, such as a reasoning model that spent the output cap reasoning.
	Empty int `json:"empty"`
	// Refused counts the distinct images the endpoint refused, this run or held.
	Refused int `json:"refused"`
	// Requests counts the vision requests this run sent.
	Requests int `json:"requests"`
	// Texts counts the vision texts embedded, one per image output line whose image has a text;
	// an empty text is one, cut to no chunk.
	Texts int `json:"texts"`
}

// Line is one line of a run's chunk record: one chunk, in one place it occurs. The same chunk
// under two inputs, or under two parts or output lines of one, is two lines, its vector the same.
type Line struct {
	// Text is the chunk's text, the source text's [Span.Start, Span.End) bytes.
	Text string `json:"text"`
	// Index counts the source text's chunks from 0.
	Index  int    `json:"index"`
	Span   Span   `json:"span"`
	Source Source `json:"source"`
	// Input is the input's absolute path, as docproc records it.
	Input string `json:"input"`
	// File is the input's file metadata, as docproc records it. Absent when it has none.
	File *schema.File `json:"file,omitempty"`
	// ContentType is the input's content type, as docproc records it. Absent for an input with
	// no part line, one whose dispatch stopped with an error.
	ContentType *schema.ContentType `json:"content_type,omitempty"`
	// Part is the part the source belongs to. Absent for an input with no part line.
	Part *Part `json:"part,omitempty"`
	// Output is the docproc output the source is, or, for a vision text, the image it was made
	// from, or, for a transcript, the first window, in plan order, of those it was joined from,
	// which names the part's stage directory. Absent for a metadata text.
	Output *Output `json:"output,omitempty"`
	// Vector is the chunk's embedding, standard base64 of little-endian float32s, as the
	// endpoint returned it. Absent when the endpoint rejected the chunk.
	Vector string `json:"vector,omitzero"`
	// Error is why the endpoint rejected the chunk. Absent when it didn't.
	Error string `json:"error,omitzero"`
}

// Span is a chunk's byte span in its source text, [Start, End).
type Span struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// The kinds of a chunk's source text.
const (
	// SourceOutput is a docproc text or Markdown output, its stored file.
	SourceOutput = "output"
	// SourceMetadata is an input's metadata text, made from its file metadata and content type.
	SourceMetadata = "metadata"
	// SourceVision is a vision model's text of a docproc image output, stored by docinfer.
	SourceVision = "vision"
	// SourceTranscript is the joined transcript of a part's docproc audio windows, stored by
	// docinfer.
	SourceTranscript = "transcript"
)

// Source is the text a chunk was cut from: its kind, the stored file it is, and its SHA-256.
type Source struct {
	Kind string `json:"kind"`
	// Ref is a docproc output's stored file, relative to the docproc run directory. Absent for a
	// text docproc didn't store.
	Ref string `json:"ref,omitzero"`
	// File is a text docinfer stored, relative to its directory. Absent for one it didn't.
	File string `json:"file,omitzero"`
	// SHA256 is the lowercase hex SHA-256 of the whole source text.
	SHA256 string `json:"sha256"`
	// Incomplete is why the model stopped before finishing a text docinfer asked it for, such as
	// "max_output_tokens", the text being what it gave. Absent for a text it finished, or one
	// no model made.
	Incomplete string `json:"incomplete,omitzero"`
}

// Part is the part a chunk's source belongs to, in docproc's record's names, and the outcome its
// part line gives it.
type Part struct {
	SHA256  string `json:"sha256"`
	Name    string `json:"name"`
	Kind    string `json:"kind,omitzero"`
	Route   string `json:"route,omitzero"`
	Where   string `json:"where,omitzero"`
	Depth   int    `json:"depth"`
	Outcome string `json:"outcome,omitzero"`
}

// Output is a docproc output, in its record's names.
type Output struct {
	Kind      string            `json:"kind"`
	Name      string            `json:"name,omitzero"`
	MediaType string            `json:"media_type,omitzero"`
	Ref       string            `json:"ref"`
	Meta      map[string]string `json:"meta,omitempty"`
}

// Embedding is a held embeddings result: what it is keyed by, and the chunk's vector, or why the
// endpoint rejected it.
type Embedding struct {
	Model    string `json:"model"`
	Settings string `json:"settings"`
	// Content is the lowercase hex SHA-256 of the chunk's text.
	Content    string `json:"content"`
	Vector     string `json:"vector,omitzero"`
	Dimensions int    `json:"dimensions,omitzero"`
	Error      string `json:"error,omitzero"`
}

// Vision is a held vision result: what it is keyed by, and the image's text, a file docinfer
// stored, or why the endpoint refused the image.
type Vision struct {
	Model    string `json:"model"`
	Settings string `json:"settings"`
	// Content is the lowercase hex SHA-256 of the image's bytes.
	Content string `json:"content"`
	// Text is the stored text's path, relative to the directory, and TextSHA256 its SHA-256.
	// Both absent when the endpoint refused the image; an empty text is a file too.
	Text       string `json:"text,omitzero"`
	TextSHA256 string `json:"text_sha256,omitzero"`
	// Incomplete is why the model stopped before finishing the text, such as
	// "max_output_tokens". Absent when it finished.
	Incomplete string `json:"incomplete,omitzero"`
	// Error is why the endpoint refused the image. Absent when it didn't.
	Error string `json:"error,omitzero"`
}

// Transcription is a held transcription result: what it is keyed by, and the window's
// transcript, a file docinfer stored, or why the endpoint refused the window.
type Transcription struct {
	Model    string `json:"model"`
	Settings string `json:"settings"`
	// Content is the lowercase hex SHA-256 of the window's bytes.
	Content string `json:"content"`
	// Text is the stored transcript's path, relative to the directory, and TextSHA256 its
	// SHA-256. Both absent when the endpoint refused the window; an empty transcript, a silent
	// window's, is a file too.
	Text       string `json:"text,omitzero"`
	TextSHA256 string `json:"text_sha256,omitzero"`
	// Error is why the endpoint refused the window. Absent when it didn't.
	Error string `json:"error,omitzero"`
}
