package transcribe

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/JaimeStill/spike-ai-pipeline-client"
)

// Transcriber sends audio windows to an audio model: each with Prompt and Language, none sent
// when empty, at most Concurrency requests in flight.
type Transcriber struct {
	Client      *client.Client
	Model       string
	Prompt      string
	Language    string
	Concurrency int
}

// Result is one window's reply: its Text and Usage, the tokens the reply says the request took,
// or, when the endpoint refused the window, Rejected, the endpoint's reply, with no Usage. Took
// is how long the request took, from its sending to its reply, refused or not.
type Result struct {
	Text     string
	Usage    client.TranscriptionUsage
	Rejected *client.APIError
	Took     time.Duration
}

// Run sends windows, each the path of an audio file, in their order, one a request, keeping at
// most Concurrency requests in flight, and calls done with each window's index and Result as its
// request completes, one call at a time. It returns how many requests it sent.
//
// Each window is opened as its request is made and streamed as it is uploaded, so no window is
// held whole. A window the endpoint refuses is a Result whose Rejected is the reply, not an
// error. Any other failure, a window that can't be read, a transport error, a reply that can't
// be decoded, a cancelled ctx, or an error done returns, stops the run: Run sends no request
// after it, waits for those in flight, and returns it. Each Result done was called with before
// then is the caller's to keep, so a run that stops resumes from what it stored.
func (t Transcriber) Run(ctx context.Context, windows []string, done func(i int, r Result) error) (int, error) {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	var (
		mu       sync.Mutex
		requests int
		wg       sync.WaitGroup
	)
	sem := make(chan struct{}, max(t.Concurrency, 1))
send:
	for i, w := range windows {
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
			r, sent, err := t.transcribe(ctx, w)
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

// transcribe sends the window at path in one request and returns its Result, and whether a
// request was sent.
func (t Transcriber) transcribe(ctx context.Context, path string) (Result, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return Result{}, false, err
	}
	defer func() { _ = f.Close() }() // read only: a close error loses nothing
	sent := time.Now()
	reply, err := t.Client.Transcribe(ctx, &client.TranscriptionRequest{
		Model: t.Model, File: f, FileName: filepath.Base(path), Prompt: t.Prompt, Language: t.Language,
	})
	took := time.Since(sent)
	var rejected *client.APIError
	switch {
	case errors.As(err, &rejected):
		return Result{Rejected: rejected, Took: took}, true, nil
	case err != nil:
		return Result{}, true, err
	}
	return Result{Text: reply.Text, Usage: reply.Usage, Took: took}, true, nil
}
