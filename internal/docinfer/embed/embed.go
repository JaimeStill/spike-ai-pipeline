package embed

import (
	"context"
	"errors"
	"sync"

	"github.com/JaimeStill/spike-ai-pipeline-client"
)

// Embedder sends texts to an embedding model: Batch texts a request, at most Concurrency
// requests in flight.
type Embedder struct {
	Client      *client.Client
	Model       string
	Batch       int
	Concurrency int
}

// Result is one text's embedding: its Vector, as the endpoint returned it, or, when the endpoint
// rejected the text, Rejected, the endpoint's reply.
type Result struct {
	Vector   []float32
	Rejected *client.APIError
}

// Run embeds texts, in their order, Batch a request, keeping at most Concurrency requests in
// flight, and calls done with each text's index and Result as its request completes, one call at
// a time. It returns how many requests it sent.
//
// A batch the endpoint rejects, a reply that isn't 2xx, is sent again a text a request, so one
// text the endpoint rejects loses no other; a text it rejects alone is a Result whose Rejected is
// the reply, not an error. Any other failure, a transport error, a reply that can't be decoded, a
// cancelled ctx, or an error done returns, stops the run: Run sends no request after it, waits
// for those in flight, and returns it. Each Result done was called with before then is the
// caller's to keep, so a run that stops resumes from what it stored.
func (e Embedder) Run(ctx context.Context, texts []string, done func(i int, r Result) error) (int, error) {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	r := &run{e: e, done: done}
	sem := make(chan struct{}, max(e.Concurrency, 1))
	var wg sync.WaitGroup
	batch := max(e.Batch, 1)
send:
	for first := 0; first < len(texts); first += batch {
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
			if err := r.batch(ctx, first, texts[first:min(first+batch, len(texts))]); err != nil {
				cancel(err)
			}
		})
	}
	wg.Wait()
	return r.requests, context.Cause(ctx)
}

// run is one Run's state: its requests, counted, and done, called under mu.
type run struct {
	e        Embedder
	done     func(int, Result) error
	mu       sync.Mutex
	requests int
}

// batch embeds texts, the run's from first on, in one request, or, when the endpoint rejects
// it, one request a text.
func (r *run) batch(ctx context.Context, first int, texts []string) error {
	r.mu.Lock()
	r.requests++
	r.mu.Unlock()
	reply, err := r.e.Client.Embed(ctx, &client.EmbeddingRequest{Model: r.e.Model, Input: texts})
	var rejected *client.APIError
	switch {
	case err == nil:
		r.mu.Lock()
		defer r.mu.Unlock()
		for i, v := range reply.Vectors {
			if err := r.done(first+i, Result{Vector: v}); err != nil {
				return err
			}
		}
		return nil
	case !errors.As(err, &rejected):
		return err
	case len(texts) == 1:
		r.mu.Lock()
		defer r.mu.Unlock()
		return r.done(first, Result{Rejected: rejected})
	}
	for i := range texts {
		if err := r.batch(ctx, first+i, texts[i:i+1]); err != nil {
			return err
		}
	}
	return nil
}
