package infer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/JaimeStill/spike-ai-pipeline-client/protocol"

	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/config"
	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/outdir"
	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/vision"
)

// visionName is the vision capability's name, in the manifest and the held results' paths.
const visionName = "vision"

// visionSettings returns the settings a vision result is held under: those that change it, the
// cap on the reply, the reasoning effort, and the prompt. The cap, a number, and the effort, one
// of the client's names, come first, so the prompt, whatever it holds, can't make two settings
// one.
func visionSettings(v *config.Vision) string {
	return fmt.Sprintf("max_tokens=%d\nreasoning=%s\nprompt=%s", *v.MaxTokens, v.Reasoning, v.Prompt)
}

// visionTexts makes the vision text of each image occurrence in occs, counting the images in m,
// and returns occs with each image occurrence that has a text given it, as a source of kind
// vision naming the stored text, and every other image occurrence dropped, recorded in m's
// failures, and a warning for each image the endpoint refused or the model left incomplete, one
// left incomplete with no text at all, such as a reasoning model that spent the cap reasoning,
// warned of and counted as empty. With
// vision unset, every image occurrence is dropped, counted, and skipped.
//
// Each distinct image, by its bytes' SHA-256, is sent once: an image an earlier run held a
// result for, under the model and the settings, is sent in no request; the rest are sent
// through the describer, each text stored whole in dir, and its result held, as it completes.
// The results are held apart from the chunk settings, so a changed bound sends no image again.
// An image the endpoint refuses is recorded, not fatal; a refusal it gives with a 4xx reply is
// held, a verdict on the image, while one it gives with any other is not, so the next run sends
// that image again. An image whose reply was cut at the output cap keeps its text, marked
// incomplete, and is embedded like any other.
//
// It writes the phase's progress lines to p: the images it sends and holds as it begins, and
// each request's line as it completes, naming the input and ref the image was first seen under.
func visionTexts(ctx context.Context, j job, dir *outdir.Dir, m *outdir.Manifest, occs []occurrence, p *progress) ([]occurrence, []string, error) {
	for _, o := range occs {
		if o.image != "" {
			m.Counts.Images++
		}
	}
	v := j.cfg.Vision
	if v == nil {
		var texts []occurrence
		for _, o := range occs {
			if o.image == "" {
				texts = append(texts, o)
			}
		}
		return texts, nil, nil
	}
	settings := visionSettings(v)
	vc := &outdir.VisionCounts{}
	m.Counts.Vision = vc
	tokens := &outdir.Usage{}
	m.Counts.Tokens.Vision = tokens

	// image is one distinct image: the input and ref it was first seen under, and its result.
	type image struct {
		input, ref string
		result     *outdir.Vision
	}
	sums := map[string]string{}   // each image file's SHA-256, by its path, each read once
	images := map[string]*image{} // by SHA-256
	var order, pendingSums []string
	var pending []vision.Image
	for _, o := range occs {
		if o.image == "" {
			continue
		}
		if _, ok := sums[o.image]; ok {
			continue
		}
		data, err := os.ReadFile(o.image)
		if err != nil {
			return nil, nil, fmt.Errorf("output %s of %s: %w", o.output.Ref, o.input, err)
		}
		h := sha256.Sum256(data)
		s := hex.EncodeToString(h[:])
		sums[o.image] = s
		if images[s] != nil {
			continue
		}
		img := &image{input: o.input, ref: o.output.Ref}
		images[s], order = img, append(order, s)
		var held outdir.Vision
		ok, err := dir.Held(visionName, outdir.Key(visionName, v.Model, settings, s), &held)
		if err != nil {
			return nil, nil, err
		}
		if ok {
			img.result = &held
			vc.Held++
			continue
		}
		pending = append(pending, vision.Image{Path: o.image, MediaType: mediaType(o, data)})
		pendingSums = append(pendingSums, s)
	}
	vc.Images = len(images)
	if err := p.phase(visionName, "images", vc.Images, len(pending), vc.Held); err != nil {
		return nil, nil, err
	}

	if len(pending) > 0 {
		cl, err := j.cfg.Endpoints[v.Endpoint].Client(j.http)
		if err != nil {
			return nil, nil, err
		}
		d := vision.Describer{Client: cl, Model: v.Model, Prompt: v.Prompt, MaxTokens: *v.MaxTokens,
			Reasoning: protocol.ReasoningEffort(v.Reasoning), Concurrency: *v.Concurrency}
		answered := 0
		vc.Requests, err = d.Run(ctx, pending, func(i int, r vision.Result) error {
			s := pendingSums[i]
			img := images[s]
			answered++
			if err := p.request(visionName, answered, len(pending), []string{img.input, img.ref}, visionOutcome(r), r.Took); err != nil {
				return err
			}
			held := &outdir.Vision{Model: v.Model, Settings: settings, Content: s}
			img.result = held
			if r.Rejected != nil {
				held.Error = r.Rejected.Error()
				if !verdict(r.Rejected) {
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
				held.Text, held.TextSHA256, held.Incomplete = rel, sum, r.Incomplete
				vc.Described++
			}
			_, err := dir.Hold(visionName, outdir.Key(visionName, v.Model, settings, s), held)
			return err
		})
		if err != nil {
			return nil, nil, fmt.Errorf("vision: %w", err)
		}
	}

	var warnings []string
	texts := map[string]string{} // by image SHA-256
	for _, s := range order {
		img := images[s]
		r := img.result
		if r.Error != "" {
			vc.Refused++
			warnings = append(warnings, fmt.Sprintf("warning: vision: refused image %s: %s", img.ref, r.Error))
			continue
		}
		text, err := dir.ReadText(r.Text, r.TextSHA256)
		if err != nil {
			return nil, nil, fmt.Errorf("vision: text of image %s: %w", img.ref, err)
		}
		if r.Incomplete != "" {
			vc.Incomplete++
			what := "incomplete text"
			if text == "" {
				vc.Empty++
				what = "empty incomplete text"
			}
			warnings = append(warnings, fmt.Sprintf("warning: vision: %s of image %s: %s", what, img.ref, r.Incomplete))
		}
		texts[s] = text
	}

	var out []occurrence
	for _, o := range occs {
		if o.image != "" {
			s := sums[o.image]
			text, ok := texts[s]
			if !ok {
				m.Failures = append(m.Failures, outdir.Failure{Capability: visionName, Input: o.input, Ref: o.output.Ref, Error: images[s].result.Error})
				continue
			}
			r := images[s].result
			o.text, o.source = text, outdir.Source{Kind: outdir.SourceVision, File: r.Text, Incomplete: r.Incomplete}
			vc.Texts++
		}
		out = append(out, o)
	}
	return out, warnings, nil
}

// visionOutcome returns r's outcome as its progress line gives it: refused, empty when the model
// stopped before finishing with no text, incomplete when it stopped with some, or ok.
func visionOutcome(r vision.Result) string {
	switch {
	case r.Rejected != nil:
		return "refused"
	case r.Incomplete != "" && r.Text == "":
		return "empty"
	case r.Incomplete != "":
		return "incomplete"
	}
	return "ok"
}

// mediaType returns the media type of o's image, whose bytes are data, for its data URL: the
// one docproc recorded, or, when it recorded none, the one its file's extension names, or, when
// that names none, the one its bytes are sniffed as.
func mediaType(o occurrence, data []byte) string {
	if o.mediaType != "" {
		return o.mediaType
	}
	if t := mime.TypeByExtension(filepath.Ext(o.image)); t != "" {
		if mt, _, err := mime.ParseMediaType(t); err == nil {
			return mt
		}
	}
	mt, _, _ := mime.ParseMediaType(http.DetectContentType(data)) // DetectContentType's are valid
	return mt
}
