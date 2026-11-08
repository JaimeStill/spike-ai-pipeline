package infer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
	"uuid"

	"github.com/JaimeStill/spike-ai-pipeline-client"

	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/chunk"
	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/config"
	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/embed"
	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/outdir"
	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/report"
	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/source"
	"github.com/JaimeStill/spike-ai-pipeline/internal/docproc/schema"
)

// job is one docinfer run's inputs, once its configuration is valid, its docproc run read, and
// its models checked: what the inference reads.
type job struct {
	cfg  *config.Config
	run  *source.Run
	http *http.Client
	out  string // the directory the run writes in
}

// embeddings is the capability's name, in the manifest and the held results' paths.
const embeddings = "embeddings"

// execute runs j's inference, writing its terminal lines to w. It begins a run in j.out under a
// new id, a version 7 UUID, writing its manifest, then makes a text of every image of the docproc
// run by vision (visionTexts), when it is configured, and a transcript of every part's audio
// windows by transcription (transcriptionTexts), when it is configured, embeds every text of the
// run, vision's and the transcripts among them (embedTexts), and writes the run's chunk record
// and its manifest again, with its end, its embedding space, its transcripts, its failures, and
// its counts, and then its report, internal/docinfer/report's page. Vision and transcription run
// one after the other, and both before any text is embedded, so the embeddings are sent in one
// pass, as the embedder takes them. Each input's terminal line, progress's, is written as its
// work ends; then the warnings, each capability's counts, the closing summary, the chunk record's
// path, and the report's, each relative to j.out, as docproc prints its record's. A run that stops writes its manifest with its end and the error that
// stopped it, no chunk record, and its report; the results it stored are held for the next run.
func execute(ctx context.Context, w io.Writer, j job) error {
	start := time.Now()
	if _, err := fmt.Fprintf(w, "docproc run %s: outputs %d: %s\n", j.run.Manifest.Run, len(j.run.Outputs), j.run.Counts()); err != nil {
		return err
	}
	dir, err := outdir.Open(j.out)
	if err != nil {
		return err
	}
	emb := j.cfg.Embeddings
	m := outdir.Manifest{
		Schema:  outdir.Version,
		Run:     uuid.NewV7().String(),
		Start:   schema.Time(start),
		Docproc: outdir.Docproc{Dir: j.run.Root, Run: j.run.Manifest.Run},
	}
	if v := j.cfg.Vision; v != nil {
		m.Capabilities = append(m.Capabilities, outdir.Capability{
			Name: visionName, Endpoint: v.Endpoint, BaseURL: j.cfg.Endpoints[v.Endpoint].BaseURL, Model: v.Model,
			Settings: map[string]any{"prompt": v.Prompt, "max_tokens": *v.MaxTokens, "reasoning": v.Reasoning, "concurrency": *v.Concurrency},
		})
	} else {
		m.Skipped = append(m.Skipped, outdir.Skipped{Name: visionName, Reason: "not configured"})
	}
	if t := j.cfg.Transcription; t != nil {
		m.Capabilities = append(m.Capabilities, outdir.Capability{
			Name: transcriptionName, Endpoint: t.Endpoint, BaseURL: j.cfg.Endpoints[t.Endpoint].BaseURL, Model: t.Model,
			Settings: map[string]any{"prompt": t.Prompt, "language": t.Language, "concurrency": *t.Concurrency},
		})
	} else {
		m.Skipped = append(m.Skipped, outdir.Skipped{Name: transcriptionName, Reason: "not configured"})
	}
	m.Capabilities = append(m.Capabilities, outdir.Capability{
		Name: embeddings, Endpoint: emb.Endpoint, BaseURL: j.cfg.Endpoints[emb.Endpoint].BaseURL, Model: emb.Model,
		Settings: map[string]any{"batch": *emb.Batch, "chunk_size": *emb.ChunkSize, "concurrency": *emb.Concurrency},
	})
	if err := dir.WriteManifest(m); err != nil {
		return err
	}
	m.Counts.Outputs = len(j.run.Outputs)
	var lines []outdir.Line
	var warnings, failed, rejected []string
	occs, err := occurrences(j.run)
	p := newProgress(w, start, j.run, occs)
	if err == nil {
		occs, warnings, err = visionTexts(ctx, j, dir, &m, occs)
	}
	if err == nil {
		occs, failed, err = transcriptionTexts(ctx, j, dir, &m, occs)
		warnings = append(warnings, failed...)
	}
	if err == nil {
		p.failed(m.Failures)
		lines, rejected, err = embedTexts(ctx, j, dir, &m, occs, p)
	}
	if err == nil {
		m.Chunks, err = dir.WriteChunks(m.Run, lines)
	}
	m.End = schema.Time(time.Now())
	if err != nil {
		m.Error = err.Error()
		if werr := dir.WriteManifest(m); werr != nil {
			return errors.Join(err, werr)
		}
		_, rerr := report.Write(dir.Root(), m.Run)
		return errors.Join(err, rerr)
	}
	if err := dir.WriteManifest(m); err != nil {
		return err
	}
	page, err := report.Write(dir.Root(), m.Run)
	if err != nil {
		return err
	}
	for _, r := range rejected {
		warnings = append(warnings, "warning: embeddings: rejected "+r)
	}
	for _, s := range warnings {
		if _, err := fmt.Fprintln(w, s); err != nil {
			return err
		}
	}
	c := m.Counts
	if vc := c.Vision; vc != nil {
		_, err = fmt.Fprintf(w, "vision: images %d, distinct %d: held %d, described %d, incomplete %d (empty %d), refused %d; requests %d\n",
			c.Images, vc.Images, vc.Held, vc.Described, vc.Incomplete, vc.Empty, vc.Refused, vc.Requests)
	} else {
		_, err = fmt.Fprintf(w, "vision: not configured; images %d skipped\n", c.Images)
	}
	if err != nil {
		return err
	}
	if tc := c.Transcription; tc != nil {
		_, err = fmt.Fprintf(w, "transcription: windows %d, distinct %d: held %d, transcribed %d, failed %d; requests %d; "+
			"transcripts %d: seams matched %d, unmatched %d\n",
			c.Windows, tc.Distinct, tc.Held, tc.Transcribed, tc.Failed, tc.Requests, tc.Transcripts, tc.SeamsMatched, tc.SeamsUnmatched)
	} else {
		_, err = fmt.Fprintf(w, "transcription: not configured; windows %d skipped\n", c.Windows)
	}
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "embeddings: texts %d, metadata %d, chunks %d: held %d, embedded %d, rejected %d; requests %d\n",
		c.Texts, c.Metadata, c.Chunks, c.Held, c.Embedded, c.Rejected, c.Requests)
	if err != nil {
		return err
	}
	if err := summary(w, m); err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "docinfer run %s: lines %d: %s\nreport %s\n", m.Run, c.Lines, m.Chunks, page)
	return err
}

// summary writes the closing summary to w: a table of each capability, in the order the run
// runs them, with the requests it sent and its failures, the distinct images or windows the
// endpoint refused, or the distinct chunks it rejected, this run or held; a capability skipped
// shows neither, and why.
func summary(w io.Writer, m outdir.Manifest) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	var err error
	row := func(cells ...string) {
		if err == nil {
			_, err = io.WriteString(tw, strings.Join(cells, "\t")+"\n")
		}
	}
	skipped := func(name string) {
		for _, s := range m.Skipped {
			if s.Name == name {
				row(name, "-", "-", "skipped: "+s.Reason)
			}
		}
	}
	c := m.Counts
	row()
	row("CAPABILITY", "REQUESTS", "FAILURES")
	if vc := c.Vision; vc != nil {
		row(visionName, strconv.Itoa(vc.Requests), strconv.Itoa(vc.Refused))
	} else {
		skipped(visionName)
	}
	if tc := c.Transcription; tc != nil {
		row(transcriptionName, strconv.Itoa(tc.Requests), strconv.Itoa(tc.Failed))
	} else {
		skipped(transcriptionName)
	}
	row(embeddings, strconv.Itoa(c.Requests), strconv.Itoa(c.Rejected))
	row()
	return errors.Join(err, tw.Flush())
}

// verdict reports whether rejected, an endpoint's refusal, is a verdict on what was sent, to be
// held: a 4xx reply is, while any other, such as a 5xx, may not recur, so the next run sends what
// it refused again.
func verdict(rejected *client.APIError) bool {
	return rejected.Status >= 400 && rejected.Status <= 499
}

// sum returns the lowercase hex SHA-256 of s.
func sum(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// embedTexts cuts and embeds every text of occs, the occurrences of j's docproc run's texts,
// vision's and the transcripts among them, filling m's counts and embedding space, and returns
// the chunk lines, one per chunk per occurrence, in the record's order and each text's, and a
// description of each chunk the endpoint rejected. It writes each input's line to p as the
// input's work ends: an input with no chunk to send at once, and each other as its last chunk's
// result arrives.
//
// Each distinct text is cut once, and each distinct chunk is embedded once: a chunk an earlier
// run held a result for, by its content, the model, and the chunk settings, is sent in no
// request; the rest are sent through the embedder, each result held in dir as it completes. A
// chunk the endpoint rejects is recorded on its lines, not fatal; a rejection the endpoint gives
// with a 4xx reply is held too, a verdict on the chunk, while one it gives with any other is
// not, so the next run sends that chunk again.
func embedTexts(ctx context.Context, j job, dir *outdir.Dir, m *outdir.Manifest, occs []occurrence, p *progress) ([]outdir.Line, []string, error) {
	emb := j.cfg.Embeddings
	bound := *emb.ChunkSize
	settings := fmt.Sprintf("chunk_size=%d chunker=%s", bound, chunk.Version)

	cuts := map[string][]chunk.Chunk{}          // by source text's SHA-256
	results := map[string]*outdir.Embedding{}   // by chunk's SHA-256
	var pending, pendingSums, rejected []string // the chunks to send, first seen first
	for i := range occs {
		o := &occs[i]
		switch o.source.Kind {
		case outdir.SourceMetadata:
			m.Counts.Metadata++
		case outdir.SourceOutput:
			m.Counts.Texts++
		}
		o.source.SHA256 = sum(o.text)
		if _, ok := cuts[o.source.SHA256]; ok {
			continue
		}
		cs := chunk.Split(o.text, bound)
		cuts[o.source.SHA256] = cs
		for _, c := range cs {
			s := sum(c.Text)
			if _, ok := results[s]; ok {
				continue
			}
			var e outdir.Embedding
			held, err := dir.Held(embeddings, outdir.Key(embeddings, emb.Model, settings, s), &e)
			if err != nil {
				return nil, nil, err
			}
			if held {
				results[s] = &e
				m.Counts.Held++
				continue
			}
			results[s] = nil // queued
			pending, pendingSums = append(pending, c.Text), append(pendingSums, s)
		}
	}
	m.Counts.Chunks = len(results)

	// Each input's chunk lines, and the queued chunks it waits for; an input that waits for none
	// has ended its work.
	for _, o := range occs {
		for _, c := range cuts[o.source.SHA256] {
			s := sum(c.Text)
			e := results[s]
			p.chunk(o.input, s, e == nil, e != nil && e.Error != "")
		}
	}
	waiters := p.waiters()
	if err := p.idle(); err != nil {
		return nil, nil, err
	}

	if len(pending) > 0 {
		cl, err := j.cfg.Endpoints[emb.Endpoint].Client(j.http)
		if err != nil {
			return nil, nil, err
		}
		e := embed.Embedder{Client: cl, Model: emb.Model, Batch: *emb.Batch, Concurrency: *emb.Concurrency}
		var usage client.EmbeddingUsage
		m.Counts.Requests, usage, err = e.Run(ctx, pending, func(i int, r embed.Result) error {
			s := pendingSums[i]
			held := &outdir.Embedding{Model: emb.Model, Settings: settings, Content: s}
			results[s] = held
			hold := true
			if r.Rejected != nil {
				held.Error = r.Rejected.Error()
				hold = verdict(r.Rejected)
			} else {
				held.Vector, held.Dimensions = outdir.EncodeVector(r.Vector), len(r.Vector)
				m.Counts.Embedded++
			}
			if hold {
				if _, err := dir.Hold(embeddings, outdir.Key(embeddings, emb.Model, settings, s), held); err != nil {
					return err
				}
			}
			return p.embedded(waiters[s], s, r.Rejected != nil)
		})
		m.Counts.Tokens.Embeddings = outdir.Usage{Input: usage.PromptTokens, Total: usage.TotalTokens}
		if err != nil {
			return nil, nil, fmt.Errorf("embeddings: %w", err)
		}
	}

	space, err := measure(results)
	if err != nil {
		return nil, nil, err
	}
	space.Endpoint, space.Model, space.ChunkSize, space.Chunker = emb.Endpoint, emb.Model, bound, chunk.Version
	m.Space = &space

	var lines []outdir.Line
	reported := map[string]bool{}
	for _, o := range occs {
		for i, c := range cuts[o.source.SHA256] {
			s := sum(c.Text)
			e := results[s]
			lines = append(lines, outdir.Line{
				Text: c.Text, Index: i, Span: outdir.Span{Start: c.Start, End: c.End}, Source: o.source,
				Input: o.input, File: o.file, ContentType: o.contentType, Part: o.part, Output: o.output,
				Vector: e.Vector, Error: e.Error,
			})
			if e.Error != "" && !reported[s] {
				reported[s] = true
				m.Counts.Rejected++
				rejected = append(rejected, fmt.Sprintf("chunk %d of %s: %s", i, describe(o), e.Error))
			}
		}
	}
	m.Counts.Lines = len(lines)
	return lines, rejected, nil
}

// describe names o's source for the terminal: its output's ref, its input's metadata, the
// vision text of its image output's ref, or the transcript of its input's windows.
func describe(o occurrence) string {
	switch o.source.Kind {
	case outdir.SourceMetadata:
		return "the metadata of " + o.input
	case outdir.SourceVision:
		return "the vision text of " + o.output.Ref
	case outdir.SourceTranscript:
		return "the transcript of " + o.input
	}
	return o.source.Ref
}

// measure returns the embedding space of results' vectors: their dimensions, which must agree,
// how many there are, and their norms' least and greatest, and whether every norm is within
// outdir.UnitTolerance of 1.
func measure(results map[string]*outdir.Embedding) (outdir.Space, error) {
	sp := outdir.Space{Tolerance: outdir.UnitTolerance, Unit: true}
	for _, e := range results {
		if e.Vector == "" {
			continue
		}
		v, err := outdir.DecodeVector(e.Vector)
		if err != nil {
			return sp, fmt.Errorf("embeddings: held vector of chunk %s: %w", e.Content, err)
		}
		if sp.Vectors > 0 && len(v) != sp.Dimensions {
			return sp, fmt.Errorf("embeddings: vectors of %d and %d dimensions; the model's space has one", sp.Dimensions, len(v))
		}
		var sq float64
		for _, f := range v {
			sq += float64(f) * float64(f)
		}
		norm := math.Sqrt(sq)
		if sp.Vectors == 0 {
			sp.Dimensions, sp.MinNorm, sp.MaxNorm = len(v), norm, norm
		}
		sp.MinNorm, sp.MaxNorm = min(sp.MinNorm, norm), max(sp.MaxNorm, norm)
		sp.Unit = sp.Unit && math.Abs(norm-1) <= outdir.UnitTolerance
		sp.Vectors++
	}
	if sp.Vectors == 0 {
		sp.Unit = false
	}
	return sp, nil
}
