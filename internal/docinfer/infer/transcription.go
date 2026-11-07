package infer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/JaimeStill/spike-ai-pipeline-processing/audio"

	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/config"
	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/outdir"
	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/transcribe"
)

// transcriptionName is the transcription capability's name, in the manifest and the held
// results' paths.
const transcriptionName = "transcription"

// transcriptionSettings returns the settings a window's transcript is held under: those that
// change it, the language and the prompt, each quoted, so neither, whatever it holds, can make
// two settings one.
func transcriptionSettings(t *config.Transcription) string {
	return fmt.Sprintf("language=%q\nprompt=%q", t.Language, t.Prompt)
}

// window is one distinct audio window, by content: the ref it was first seen under, and its
// result.
type window struct {
	ref    string
	result *outdir.Transcription
}

// transcriptionTexts makes the transcript of each part's audio windows in occs, counting the
// windows in m and recording each transcript it joins there, and returns occs with each part's
// window occurrences replaced by one of its transcript, as a source of kind transcript naming the
// stored text, at the place of the part's first window in the record, and a warning for each
// window the endpoint refused, each output line of a refused window recorded in m's failures.
// With transcription unset, every window occurrence is dropped, counted, and skipped.
//
// Each distinct window, by its bytes' SHA-256, is sent once: a window an earlier run held a
// result for, under the model and the settings, is sent in no request; the rest are sent through
// the transcriber, each transcript stored whole in dir, and its result held, as it completes.
// The results are held apart from the chunk settings, so a changed bound sends no window again.
// A window the endpoint refuses is recorded, not fatal; a refusal it gives with a 4xx reply is
// held, a verdict on the window, while one it gives with any other is not, so the next run sends
// that window again.
//
// A part's windows, those of one part cut by one stage, an audio part's or a video's, are then
// joined in plan order, by their meta's index, whatever order the record lists them in, by
// processing's audio.Join at its default seam bound: a refused window joins as an empty
// transcript, as does a window the plan's count has but the record doesn't, one its stage failed
// to cut. The joined text is stored whole in dir, the same text the same file, and recorded in
// m's transcripts with how each seam was merged. Joining reads only held and stored texts, so it
// runs again, unchanged, on every run.
func transcriptionTexts(ctx context.Context, j job, dir *outdir.Dir, m *outdir.Manifest, occs []occurrence) ([]occurrence, []string, error) {
	for _, o := range occs {
		if o.audio != "" {
			m.Counts.Windows++
		}
	}
	t := j.cfg.Transcription
	if t == nil {
		var texts []occurrence
		for _, o := range occs {
			if o.audio == "" {
				texts = append(texts, o)
			}
		}
		return texts, nil, nil
	}
	settings := transcriptionSettings(t)
	tc := &outdir.TranscriptionCounts{}
	m.Counts.Transcription = tc
	tokens := &outdir.Usage{}
	m.Counts.Tokens.Transcription = tokens

	sums := map[string]string{}     // each window file's SHA-256, by its path, each read once
	windows := map[string]*window{} // by SHA-256
	var order, pending, pendingSums []string
	for _, o := range occs {
		if o.audio == "" {
			continue
		}
		if _, ok := sums[o.audio]; ok {
			continue
		}
		s, err := fileSum(o.audio)
		if err != nil {
			return nil, nil, fmt.Errorf("output %s of %s: %w", o.output.Ref, o.input, err)
		}
		sums[o.audio] = s
		if windows[s] != nil {
			continue
		}
		w := &window{ref: o.output.Ref}
		windows[s], order = w, append(order, s)
		var held outdir.Transcription
		ok, err := dir.Held(transcriptionName, outdir.Key(transcriptionName, t.Model, settings, s), &held)
		if err != nil {
			return nil, nil, err
		}
		if ok {
			w.result = &held
			tc.Held++
			continue
		}
		pending, pendingSums = append(pending, o.audio), append(pendingSums, s)
	}
	tc.Distinct = len(windows)

	if len(pending) > 0 {
		cl, err := j.cfg.Endpoints[t.Endpoint].Client(j.http)
		if err != nil {
			return nil, nil, err
		}
		tr := transcribe.Transcriber{Client: cl, Model: t.Model, Prompt: t.Prompt, Language: t.Language, Concurrency: *t.Concurrency}
		tc.Requests, err = tr.Run(ctx, pending, func(i int, r transcribe.Result) error {
			s := pendingSums[i]
			held := &outdir.Transcription{Model: t.Model, Settings: settings, Content: s}
			windows[s].result = held
			if r.Rejected != nil {
				held.Error = r.Rejected.Error()
				if r.Rejected.Status < 400 || r.Rejected.Status > 499 {
					return nil
				}
			} else {
				tokens.Input += r.Usage.InputTokens
				tokens.Output += r.Usage.OutputTokens
				tokens.Total += r.Usage.TotalTokens
				rel, sum, err := dir.WriteText(r.Text)
				if err != nil {
					return err
				}
				held.Text, held.TextSHA256 = rel, sum
				tc.Transcribed++
			}
			_, err := dir.Hold(transcriptionName, outdir.Key(transcriptionName, t.Model, settings, s), held)
			return err
		})
		if err != nil {
			return nil, nil, fmt.Errorf("transcription: %w", err)
		}
	}

	var warnings []string
	texts := map[string]string{} // by window SHA-256; a refused window has none
	for _, s := range order {
		w := windows[s]
		r := w.result
		if r.Error != "" {
			tc.Failed++
			warnings = append(warnings, fmt.Sprintf("warning: transcription: failed window %s: %s", w.ref, r.Error))
			continue
		}
		text, err := dir.ReadText(r.Text, r.TextSHA256)
		if err != nil {
			return nil, nil, fmt.Errorf("transcription: transcript of window %s: %w", w.ref, err)
		}
		texts[s] = text
	}

	// Each window output line whose window was refused is a failure, the input's.
	for _, o := range occs {
		if o.audio == "" {
			continue
		}
		if r := windows[sums[o.audio]].result; r.Error != "" {
			m.Failures = append(m.Failures, outdir.Failure{Capability: transcriptionName, Input: o.input, Ref: o.output.Ref, Error: r.Error})
		}
	}

	// Each part's windows, by plan index, the first line of an index kept; and the parts in the
	// record's order of their first window.
	plans := map[windowGroup]map[int]occurrence{}
	counts := map[windowGroup]int{}
	for _, o := range occs {
		if o.audio == "" {
			continue
		}
		if plans[o.group] == nil {
			plans[o.group] = map[int]occurrence{}
		}
		if _, ok := plans[o.group][o.window]; !ok {
			plans[o.group][o.window] = o
		}
		counts[o.group] = max(counts[o.group], o.count, o.window+1)
	}
	var out []occurrence
	joined := map[windowGroup]bool{}
	for _, o := range occs {
		if o.audio == "" {
			out = append(out, o)
			continue
		}
		if joined[o.group] {
			continue
		}
		joined[o.group] = true
		occ, err := joinWindows(dir, m, plans[o.group], counts[o.group], sums, texts)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, occ)
	}
	return out, warnings, nil
}

// joinWindows joins the transcripts of plan's windows, by plan index, n in the plan, each a
// window's text by its SHA-256, from sums and texts, an index with no window or no text an empty
// transcript; stores the joined text in dir; records it in m, counting its seams; and returns its
// occurrence: the input, part, and first window of plan's, its text the joined one.
func joinWindows(dir *outdir.Dir, m *outdir.Manifest, plan map[int]occurrence, n int, sums, texts map[string]string) (occurrence, error) {
	tc := m.Counts.Transcription
	transcripts := make([]string, n)
	refs := make([]string, n)
	var failed []int
	first := -1
	for i := range n {
		o, ok := plan[i]
		if !ok {
			failed = append(failed, i)
			continue
		}
		if first < 0 {
			first = i
		}
		refs[i] = o.output.Ref
		text, ok := texts[sums[o.audio]]
		if !ok {
			failed = append(failed, i)
		}
		transcripts[i] = text
	}
	occ := plan[first]
	// Join's default seam bound is the reach of its default 2 s overlap, the overlap docproc's
	// stages plan with by default; the window meta gives no other bound to use.
	joint, err := audio.Join(transcripts, audio.JoinRequest{})
	if err != nil {
		return occ, fmt.Errorf("transcription: join the windows of %s: %w", occ.input, err)
	}
	text := joint.Text()
	rel, sum, err := dir.WriteText(text)
	if err != nil {
		return occ, err
	}
	rec := outdir.Transcript{
		Input: occ.input, Part: occ.part, Stage: occ.group.stage.Name, Windows: refs, Failed: failed,
		File: rel, SHA256: sum, Seams: []outdir.Seam{},
	}
	for _, s := range joint.Seams {
		rec.Seams = append(rec.Seams, outdir.Seam{Matched: s.Matched, Matches: s.Matches, Overlap: s.Overlap, Position: s.Position})
		if s.Matched {
			tc.SeamsMatched++
		} else {
			tc.SeamsUnmatched++
		}
	}
	m.Transcripts = append(m.Transcripts, rec)
	tc.Transcripts++
	occ.audio, occ.text = "", text
	occ.source = outdir.Source{Kind: outdir.SourceTranscript, File: rel}
	return occ, nil
}

// fileSum returns the lowercase hex SHA-256 of the file at path, read as a stream.
func fileSum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }() // read only: a close error loses nothing
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
