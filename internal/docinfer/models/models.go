package models

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/config"
)

// The states a capability's row shows, beside a listed model's own status, such as "unloaded",
// which the endpoint reports.
const (
	// Loaded is a model the listing names with the status "loaded".
	Loaded = "loaded"
	// Listed is a model the listing names with no status, as an endpoint that isn't llama.cpp's
	// router lists one.
	Listed = "listed"
	// Unlisted is a model the listing doesn't name: docinfer refuses it.
	Unlisted = "unlisted"
	// Unchecked is a model on an endpoint that gives no listing, such as one that answers
	// /models with 404: docinfer warns and goes on.
	Unchecked = "unchecked"
	// Unreachable is a model on an endpoint the listing's request couldn't reach: docinfer
	// refuses it, since no request to it would be answered either.
	Unreachable = "unreachable"
	// Skipped is a capability the configuration leaves unset.
	Skipped = "skipped"
)

// listTimeout bounds one listing request, its reply's body included.
const listTimeout = 30 * time.Second

// maxListing bounds the listing's body docinfer reads.
const maxListing = 8 << 20

// Row is one capability's line of the table: its endpoint, its model, and their state.
type Row struct {
	Capability string
	Endpoint   string
	Model      string
	State      string
}

// Report is what [Check] found: a row for each capability, in config.Config.Capabilities'
// order, and its warnings, in the order found.
type Report struct {
	Rows     []Row
	Warnings []string
}

// Check lists the models of each endpoint a capability of cfg uses, once each, through hc, by
// GET <base_url>/models with the endpoint's query and token, and returns a row for each
// capability. A capability cfg leaves unset is skipped. An endpoint whose listing can't be had,
// a reply that isn't 2xx or that holds no "data" list, is a warning, and its models are
// unchecked; a listed model whose status isn't "loaded", such as llama.cpp's router's
// "unloaded", is a warning too. The error joins each model a listing doesn't name and each
// endpoint the request couldn't reach: either stops docinfer. The Report is whole either way, so
// its table can be shown before the error.
func Check(ctx context.Context, hc *http.Client, cfg *config.Config) (Report, error) {
	var (
		r        Report
		errs     []error
		listings = map[string]listing{} // by endpoint name, each listed once
	)
	for _, n := range cfg.Capabilities() {
		if n.Capability == nil {
			r.Rows = append(r.Rows, Row{Capability: n.Name, Endpoint: "-", Model: "-", State: Skipped})
			continue
		}
		name, model := n.Capability.Endpoint, n.Capability.Model
		l, ok := listings[name]
		if !ok {
			l = list(ctx, hc, cfg.Endpoints[name])
			listings[name] = l
			if l.unlisted != "" {
				r.Warnings = append(r.Warnings, fmt.Sprintf("endpoint %q gives no model listing (%s); its models are unchecked", name, l.unlisted))
			}
		}
		row := Row{Capability: n.Name, Endpoint: name, Model: model}
		switch status, listed := l.models[model]; {
		case l.err != nil:
			row.State = Unreachable
			if !ok {
				errs = append(errs, fmt.Errorf("endpoint %q: %w", name, l.err))
			}
		case l.unlisted != "":
			row.State = Unchecked
		case !listed:
			row.State = Unlisted
			errs = append(errs, fmt.Errorf("%s: endpoint %q doesn't list model %q", n.Name, name, model))
		case status == "":
			row.State = Listed
		default:
			row.State = status
			if status != Loaded {
				r.Warnings = append(r.Warnings, fmt.Sprintf("%s: endpoint %q lists model %q as %s", n.Name, name, model, status))
			}
		}
		r.Rows = append(r.Rows, row)
	}
	return r, errors.Join(errs...)
}

// listing is one endpoint's models by id, each with its status, empty when the endpoint reports
// none; or, unlisted, why it gave no listing; or err, the request's failure.
type listing struct {
	models   map[string]string
	unlisted string
	err      error
}

// list returns e's listing, as [Check] describes.
func list(ctx context.Context, hc *http.Client, e *config.Endpoint) listing {
	ctx, cancel := context.WithTimeout(ctx, listTimeout)
	defer cancel()
	u, err := url.Parse(e.BaseURL)
	if err != nil {
		return listing{err: err}
	}
	u = u.JoinPath("models")
	u.RawQuery = e.Values().Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return listing{err: err}
	}
	req.Header.Set("Accept", "application/json")
	if e.Token != "" {
		req.Header.Set("Authorization", "Bearer "+e.Token)
	}
	resp, err := hc.Do(req)
	if err != nil {
		return listing{err: err}
	}
	defer func() { _ = resp.Body.Close() }() // read only: a close error loses nothing
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return listing{unlisted: "GET " + redact(u) + ": " + resp.Status}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxListing))
	if err != nil {
		return listing{err: err}
	}
	// The OpenAI listing, {"data": [{"id": ...}]}; llama.cpp's router adds each model's
	// "status": {"value": "loaded"}. A status of another shape is no status.
	var reply struct {
		Data *[]struct {
			ID     string          `json:"id"`
			Status json.RawMessage `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &reply); err != nil || reply.Data == nil {
		return listing{unlisted: "GET " + redact(u) + ": the reply holds no \"data\" list"}
	}
	l := listing{models: map[string]string{}}
	for _, m := range *reply.Data {
		var status struct {
			Value string `json:"value"`
		}
		_ = json.Unmarshal(m.Status, &status) // a status of another shape is none
		l.models[m.ID] = status.Value
	}
	return l
}

// redact returns u without its query, which a deployment may use for a key.
func redact(u *url.URL) string {
	c := *u
	c.RawQuery = ""
	return c.String()
}

// Write writes r's table, CAPABILITY ENDPOINT MODEL STATE, and then each warning on a line of
// its own, "warning: " and the warning.
func (r Report) Write(w io.Writer) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	// tabwriter buffers every cell until Flush, so a write to w fails only there; err keeps the
	// first failure all the same.
	var err error
	row := func(cells ...string) {
		if err == nil {
			_, err = io.WriteString(tw, strings.Join(cells, "\t")+"\n")
		}
	}
	row("CAPABILITY", "ENDPOINT", "MODEL", "STATE")
	for _, x := range r.Rows {
		row(x.Capability, x.Endpoint, x.Model, x.State)
	}
	if ferr := tw.Flush(); err == nil {
		err = ferr
	}
	if err != nil {
		return err
	}
	for _, warning := range r.Warnings {
		if _, err := fmt.Fprintf(w, "warning: %s\n", warning); err != nil {
			return err
		}
	}
	return nil
}
