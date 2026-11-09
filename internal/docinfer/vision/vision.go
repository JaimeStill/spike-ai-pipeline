package vision

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/JaimeStill/spike-ai-pipeline-client"
	"github.com/JaimeStill/spike-ai-pipeline-client/protocol"
)

// Describer sends images to a vision model: each with Prompt, its reply capped at MaxTokens
// output tokens, reasoning included, the model asked to reason with Reasoning's effort, at most
// Concurrency requests in flight. An empty Reasoning leaves the reasoning to the endpoint.
type Describer struct {
	Client      *client.Client
	Model       string
	Prompt      string
	MaxTokens   int
	Reasoning   protocol.ReasoningEffort
	Concurrency int
}

// Image is one image to send: the file it is read from, and its media type, such as
// "image/png", which its data URL names.
type Image struct {
	Path      string
	MediaType string
}

// Result is one image's reply: its Text, and, when the model stopped before finishing,
// Incomplete, the reason the response gives, such as "max_output_tokens", and Usage, the tokens
// the response says the request took; or, when the endpoint refused the image, Rejected, the
// endpoint's reply, with no Usage. Took is how long the request took, from its sending to its
// reply, refused or not.
type Result struct {
	Text       string
	Incomplete string
	Usage      protocol.Usage
	Rejected   *client.APIError
	Took       time.Duration
}

// Run sends images, in their order, one a request, keeping at most Concurrency requests in
// flight, and calls done with each image's index and Result as its request completes, one call
// at a time. It returns how many requests it sent.
//
// Each image is read as its request is made, so no more than Concurrency images are held at
// once. An image the endpoint refuses is a Result whose Rejected is the reply, not an error. Any
// other failure, an image that can't be read, a transport error, a reply that can't be decoded,
// a cancelled ctx, or an error done returns, stops the run: Run sends no request after it, waits
// for those in flight, and returns it. Each Result done was called with before then is the
// caller's to keep, so a run that stops resumes from what it stored.
func (d Describer) Run(ctx context.Context, images []Image, done func(i int, r Result) error) (int, error) {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	var (
		mu       sync.Mutex
		requests int
		wg       sync.WaitGroup
	)
	sem := make(chan struct{}, max(d.Concurrency, 1))
send:
	for i, img := range images {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			break send
		}
		if ctx.Err() != nil {
			<-sem
			break
		}
		wg.Go(func() {
			defer func() { <-sem }()
			r, sent, err := d.describe(ctx, img)
			mu.Lock()
			defer mu.Unlock()
			if sent {
				requests++
			}
			if err == nil {
				err = done(i, r)
			}
			if err != nil {
				cancel(err)
			}
		})
	}
	wg.Wait()
	return requests, context.Cause(ctx)
}

// describe sends img in one request and returns its Result, and whether a request was sent.
func (d Describer) describe(ctx context.Context, img Image) (Result, bool, error) {
	data, err := os.ReadFile(img.Path)
	if err != nil {
		return Result{}, false, err
	}
	req := &protocol.Request{
		Model: d.Model,
		Input: []protocol.Item{protocol.Message{Role: protocol.RoleUser, Content: []protocol.ContentPart{
			protocol.InputImage{ImageURL: protocol.DataURL(img.MediaType, data)},
			protocol.InputText{Text: d.Prompt},
		}}},
		MaxOutputTokens: d.MaxTokens,
		Reasoning:       protocol.ReasoningParam{Effort: d.Reasoning},
	}
	sent := time.Now()
	resp, err := d.Client.Generate(ctx, req)
	took := time.Since(sent)
	var rejected *client.APIError
	switch {
	case errors.As(err, &rejected):
		return Result{Rejected: rejected, Took: took}, true, nil
	case err != nil:
		return Result{}, true, err
	}
	r := Result{Text: text(resp), Usage: resp.Usage, Took: took}
	if resp.Status == protocol.StatusIncomplete {
		r.Incomplete = "incomplete"
		if resp.IncompleteDetails != nil && resp.IncompleteDetails.Reason != "" {
			r.Incomplete = string(resp.IncompleteDetails.Reason)
		}
	} else if resp.Status != protocol.StatusCompleted {
		return Result{}, true, fmt.Errorf("response has status %q", resp.Status)
	}
	return r, true, nil
}

// text returns resp's text: the output text of its assistant messages, in order, never its
// reasoning, each message's parts joined as they are and the messages by a blank line.
func text(resp *protocol.Response) string {
	var msgs []string
	for _, it := range resp.Output {
		m, ok := it.(protocol.Message)
		if !ok || m.Role != protocol.RoleAssistant {
			continue
		}
		var b strings.Builder
		for _, p := range m.Content {
			if t, ok := p.(protocol.OutputText); ok {
				b.WriteString(t.Text)
			}
		}
		if b.Len() > 0 {
			msgs = append(msgs, b.String())
		}
	}
	return strings.Join(msgs, "\n\n")
}
