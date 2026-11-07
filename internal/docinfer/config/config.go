package config

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"

	"github.com/JaimeStill/spike-ai-pipeline-client"
	"github.com/JaimeStill/spike-ai-pipeline-client/format"
	"github.com/JaimeStill/spike-ai-pipeline-client/format/completions"
	"github.com/JaimeStill/spike-ai-pipeline-client/format/messages"
	"github.com/JaimeStill/spike-ai-pipeline-client/format/responses"
	"github.com/standards-lab/go-core/config"
)

// EnvPrefix is the prefix of every environment variable docinfer reads: DOCINFER_ENV selects the
// overlays, and each override is DOCINFER_ and its setting's path, such as
// DOCINFER_EMBEDDINGS_MODEL.
const EnvPrefix = "DOCINFER"

// The defaults Finalize gives a setting a capability leaves unset.
const (
	// DefaultVisionPrompt is what the vision model is asked of each image.
	DefaultVisionPrompt = "Transcribe the text in this picture. If it has no text, describe it in one sentence."
	// DefaultVisionMaxTokens caps the vision model's reply to one image.
	DefaultVisionMaxTokens = 2048
	// DefaultBatch is how many chunks one embeddings request carries.
	DefaultBatch = 16
	// DefaultChunkSize bounds a chunk, in characters.
	DefaultChunkSize = 2000
	// DefaultConcurrency is how many requests a capability keeps in flight at once.
	DefaultConcurrency = 4
)

// The wire formats an endpoint speaks, each one of the client's format packages.
const (
	Completions = "completions"
	Messages    = "messages"
	Responses   = "responses"
)

// formats are the formats an endpoint may name, in the order an error lists them.
var formats = []string{Completions, Messages, Responses}

// Config is docinfer's configuration: its named endpoints, and the three capabilities, each an
// endpoint and a model. Embeddings is required; Vision and Transcription are optional, and a nil
// one is skipped. Env records the variable names Finalize composed; it is excluded from JSON.
type Config struct {
	Endpoints     map[string]*Endpoint `json:"endpoints"`
	Embeddings    *Embeddings          `json:"embeddings"`
	Vision        *Vision              `json:"vision"`
	Transcription *Transcription       `json:"transcription"`
	Env           Env                  `json:"-"`
}

// Endpoint is one model server's API root and how docinfer speaks to it:
//
//   - "base_url": its API root, such as "http://gb10-2:8080/v1", an absolute http or https URL
//     with no query or fragment
//   - "format": the wire format a generation is sent in, "completions", "messages", or
//     "responses"
//   - "token": the bearer token each request carries; empty sends none. It belongs in the
//     secrets file
//   - "query": parameters added to every request's URL, such as {"autoload": "false"}; optional
type Endpoint struct {
	BaseURL string            `json:"base_url"`
	Format  string            `json:"format"`
	Token   string            `json:"token"`
	Query   map[string]string `json:"query"`
}

// Capability names the endpoint a capability's requests go to, by its key in "endpoints", and
// the model they name there, and bounds the requests it keeps in flight.
type Capability struct {
	Endpoint    string `json:"endpoint"`
	Model       string `json:"model"`
	Concurrency *int   `json:"concurrency"`
}

// Embeddings is the embeddings capability: the model each chunk of text is embedded by, how many
// chunks one request carries ("batch"), and the most characters a chunk holds ("chunk_size").
type Embeddings struct {
	Capability
	Batch     *int `json:"batch"`
	ChunkSize *int `json:"chunk_size"`
}

// Vision is the vision capability: the model each image is sent to, what it is asked of each
// ("prompt"), and the cap on its reply ("max_tokens").
type Vision struct {
	Capability
	Prompt    string `json:"prompt"`
	MaxTokens *int   `json:"max_tokens"`
}

// Transcription is the transcription capability: the model each audio window is sent to, with
// an optional "prompt" and "language"; empty sends none.
type Transcription struct {
	Capability
	Prompt   string `json:"prompt"`
	Language string `json:"language"`
}

// Env names the variables Finalize read: each endpoint's, by its key, and each capability's.
type Env struct {
	Endpoints     map[string]EndpointEnv
	Embeddings    EmbeddingsEnv
	Vision        VisionEnv
	Transcription TranscriptionEnv
}

// EndpointEnv names an endpoint's overrides, DOCINFER_ENDPOINTS_<KEY>_BASE_URL and the rest.
type EndpointEnv struct{ BaseURL, Format, Token string }

// CapabilityEnv names the overrides every capability has.
type CapabilityEnv struct{ Endpoint, Model, Concurrency string }

// EmbeddingsEnv names the embeddings capability's overrides, DOCINFER_EMBEDDINGS_*.
type EmbeddingsEnv struct {
	CapabilityEnv
	Batch, ChunkSize string
}

// VisionEnv names the vision capability's overrides, DOCINFER_VISION_*.
type VisionEnv struct {
	CapabilityEnv
	Prompt, MaxTokens string
}

// TranscriptionEnv names the transcription capability's overrides, DOCINFER_TRANSCRIPTION_*.
type TranscriptionEnv struct {
	CapabilityEnv
	Prompt, Language string
}

// Load reads the configuration in dir, the directory --config names, through go-core's layered
// load: config.json, its DOCINFER_ENV overlay, secrets.json, and its overlay, later files
// winning, then the DOCINFER_ overrides, then validates it. A dir that isn't a directory is an
// error, so a mistyped --config isn't read as an empty configuration.
func Load(dir string) (*Config, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("config: %s is no directory", dir)
	}
	return config.Load[Config](config.Options{Dir: dir, EnvPrefix: EnvPrefix})
}

// Merge overlays src's set fields onto c: an endpoint or a capability src sets is merged into c's
// field by field, so a secrets file can carry an endpoint's token alone.
func (c *Config) Merge(src *Config) {
	for name, e := range src.Endpoints {
		if e == nil {
			continue
		}
		if c.Endpoints == nil {
			c.Endpoints = map[string]*Endpoint{}
		}
		if c.Endpoints[name] == nil {
			c.Endpoints[name] = &Endpoint{}
		}
		c.Endpoints[name].merge(e)
	}
	if src.Embeddings != nil {
		c.Embeddings = cmpNew(c.Embeddings)
		c.Embeddings.merge(src.Embeddings)
	}
	if src.Vision != nil {
		c.Vision = cmpNew(c.Vision)
		c.Vision.merge(src.Vision)
	}
	if src.Transcription != nil {
		c.Transcription = cmpNew(c.Transcription)
		c.Transcription.merge(src.Transcription)
	}
}

// cmpNew returns p, or a new zero T when p is nil.
func cmpNew[T any](p *T) *T {
	if p == nil {
		return new(T)
	}
	return p
}

func (e *Endpoint) merge(src *Endpoint) {
	setString(&e.BaseURL, src.BaseURL)
	setString(&e.Format, src.Format)
	setString(&e.Token, src.Token)
	if len(src.Query) > 0 {
		if e.Query == nil {
			e.Query = map[string]string{}
		}
		maps.Copy(e.Query, src.Query)
	}
}

func (c *Capability) merge(src *Capability) {
	setString(&c.Endpoint, src.Endpoint)
	setString(&c.Model, src.Model)
	setInt(&c.Concurrency, src.Concurrency)
}

func (e *Embeddings) merge(src *Embeddings) {
	e.Capability.merge(&src.Capability)
	setInt(&e.Batch, src.Batch)
	setInt(&e.ChunkSize, src.ChunkSize)
}

func (v *Vision) merge(src *Vision) {
	v.Capability.merge(&src.Capability)
	setString(&v.Prompt, src.Prompt)
	setInt(&v.MaxTokens, src.MaxTokens)
}

func (t *Transcription) merge(src *Transcription) {
	t.Capability.merge(&src.Capability)
	setString(&t.Prompt, src.Prompt)
	setString(&t.Language, src.Language)
}

// setString sets *dst to src when src is set, not empty.
func setString(dst *string, src string) {
	if src != "" {
		*dst = src
	}
}

// setInt sets *dst to src when src is set, not nil.
func setInt(dst **int, src *int) {
	if src != nil {
		*dst = new(*src)
	}
}

// Finalize composes the override names from envPrefix, reads the overrides, applies the
// defaults, and validates, so every error comes before any request. An override of a capability
// the files leave unset sets it. An endpoint can be overridden only when the files name it, by
// its key. The errors: an override that doesn't parse, no embeddings capability, a capability
// with no endpoint or model or naming an endpoint "endpoints" doesn't hold, an endpoint whose
// base URL isn't an absolute http or https URL without a query or fragment or whose format isn't
// one of the three, two endpoint keys whose overrides share names, and a count below 1.
func (c *Config) Finalize(envPrefix string) error {
	if err := c.readEnv(envPrefix); err != nil {
		return err
	}
	c.defaults()
	return c.validate()
}

// readEnv composes the override names from prefix, recording them on c.Env, and applies every
// override set.
func (c *Config) readEnv(prefix string) error {
	c.Env = Env{
		Endpoints:     map[string]EndpointEnv{},
		Embeddings:    EmbeddingsEnv{capabilityEnv(prefix, "embeddings"), config.EnvName(prefix, "embeddings", "batch"), config.EnvName(prefix, "embeddings", "chunk_size")},
		Vision:        VisionEnv{capabilityEnv(prefix, "vision"), config.EnvName(prefix, "vision", "prompt"), config.EnvName(prefix, "vision", "max_tokens")},
		Transcription: TranscriptionEnv{capabilityEnv(prefix, "transcription"), config.EnvName(prefix, "transcription", "prompt"), config.EnvName(prefix, "transcription", "language")},
	}
	var errs []error
	owner := map[string]string{} // an override's name, to the endpoint key that composed it
	for _, name := range slices.Sorted(maps.Keys(c.Endpoints)) {
		env := EndpointEnv{
			BaseURL: config.EnvName(prefix, "endpoints", name, "base_url"),
			Format:  config.EnvName(prefix, "endpoints", name, "format"),
			Token:   config.EnvName(prefix, "endpoints", name, "token"),
		}
		c.Env.Endpoints[name] = env
		if env.BaseURL == "" {
			continue
		}
		if other, ok := owner[env.BaseURL]; ok {
			errs = append(errs, fmt.Errorf("endpoints %q and %q compose the same override names, such as %s", other, name, env.BaseURL))
			continue
		}
		owner[env.BaseURL] = name
		e := c.Endpoints[name]
		envString(&e.BaseURL, env.BaseURL)
		envString(&e.Format, env.Format)
		envString(&e.Token, env.Token)
	}

	var emb Embeddings
	errs = append(errs, emb.readEnv(c.Env.Embeddings.CapabilityEnv),
		envInt(&emb.Batch, c.Env.Embeddings.Batch), envInt(&emb.ChunkSize, c.Env.Embeddings.ChunkSize))
	if emb != (Embeddings{}) {
		c.Embeddings = cmpNew(c.Embeddings)
		c.Embeddings.merge(&emb)
	}
	var vis Vision
	errs = append(errs, vis.readEnv(c.Env.Vision.CapabilityEnv), envInt(&vis.MaxTokens, c.Env.Vision.MaxTokens))
	envString(&vis.Prompt, c.Env.Vision.Prompt)
	if vis != (Vision{}) {
		c.Vision = cmpNew(c.Vision)
		c.Vision.merge(&vis)
	}
	var tr Transcription
	errs = append(errs, tr.readEnv(c.Env.Transcription.CapabilityEnv))
	envString(&tr.Prompt, c.Env.Transcription.Prompt)
	envString(&tr.Language, c.Env.Transcription.Language)
	if tr != (Transcription{}) {
		c.Transcription = cmpNew(c.Transcription)
		c.Transcription.merge(&tr)
	}
	return errors.Join(errs...)
}

func capabilityEnv(prefix, name string) CapabilityEnv {
	return CapabilityEnv{
		Endpoint:    config.EnvName(prefix, name, "endpoint"),
		Model:       config.EnvName(prefix, name, "model"),
		Concurrency: config.EnvName(prefix, name, "concurrency"),
	}
}

func (c *Capability) readEnv(env CapabilityEnv) error {
	envString(&c.Endpoint, env.Endpoint)
	envString(&c.Model, env.Model)
	return envInt(&c.Concurrency, env.Concurrency)
}

// envString sets *dst to the variable name's value when it is set and not empty.
func envString(dst *string, name string) {
	if name != "" {
		setString(dst, os.Getenv(name))
	}
}

// envInt sets *dst to the variable name's value, an integer, when it is set and not empty.
func envInt(dst **int, name string) error {
	return config.SetFromEnv(dst, name, strconv.Atoi)
}

// defaults gives each set capability's unset settings their defaults.
func (c *Config) defaults() {
	if e := c.Embeddings; e != nil {
		defaultInt(&e.Concurrency, DefaultConcurrency)
		defaultInt(&e.Batch, DefaultBatch)
		defaultInt(&e.ChunkSize, DefaultChunkSize)
	}
	if v := c.Vision; v != nil {
		defaultInt(&v.Concurrency, DefaultConcurrency)
		defaultInt(&v.MaxTokens, DefaultVisionMaxTokens)
		if v.Prompt == "" {
			v.Prompt = DefaultVisionPrompt
		}
	}
	if t := c.Transcription; t != nil {
		defaultInt(&t.Concurrency, DefaultConcurrency)
	}
}

func defaultInt(p **int, v int) {
	if *p == nil {
		*p = &v
	}
}

// validate returns every error in c, joined.
func (c *Config) validate() error {
	var errs []error
	for _, name := range slices.Sorted(maps.Keys(c.Endpoints)) {
		if err := c.Endpoints[name].validate(); err != nil {
			errs = append(errs, fmt.Errorf("endpoints %q: %w", name, err))
		}
	}
	if c.Embeddings == nil {
		errs = append(errs, errors.New("embeddings: no capability configured; it is required"))
	} else {
		errs = append(errs, c.check("embeddings", c.Embeddings.Capability),
			positive("embeddings", "batch", *c.Embeddings.Batch),
			positive("embeddings", "chunk_size", *c.Embeddings.ChunkSize))
	}
	if c.Vision != nil {
		errs = append(errs, c.check("vision", c.Vision.Capability),
			positive("vision", "max_tokens", *c.Vision.MaxTokens))
	}
	if c.Transcription != nil {
		errs = append(errs, c.check("transcription", c.Transcription.Capability))
	}
	return errors.Join(errs...)
}

// check returns what is wrong with the capability name's endpoint, model, and concurrency.
func (c *Config) check(name string, capability Capability) error {
	var errs []error
	switch {
	case capability.Endpoint == "":
		errs = append(errs, fmt.Errorf("%s: no endpoint", name))
	case c.Endpoints[capability.Endpoint] == nil:
		errs = append(errs, fmt.Errorf("%s: unknown endpoint %q", name, capability.Endpoint))
	}
	if capability.Model == "" {
		errs = append(errs, fmt.Errorf("%s: no model", name))
	}
	errs = append(errs, positive(name, "concurrency", *capability.Concurrency))
	return errors.Join(errs...)
}

// positive returns an error unless the setting key of capability name is at least 1.
func positive(name, key string, v int) error {
	if v < 1 {
		return fmt.Errorf("%s: %s %d is below 1", name, key, v)
	}
	return nil
}

// validate returns what is wrong with e's base URL and format: the URL is checked as the
// client's New checks it, so no endpoint fails only once a request is built.
func (e *Endpoint) validate() error {
	var errs []error
	if !slices.Contains(formats, e.Format) {
		errs = append(errs, fmt.Errorf("unknown format %q; want one of %q", e.Format, formats))
	}
	if e.BaseURL == "" {
		errs = append(errs, errors.New("no base_url"))
	} else if u, err := url.Parse(e.BaseURL); err != nil {
		errs = append(errs, fmt.Errorf("base_url: %w", err))
	} else if !u.IsAbs() || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		errs = append(errs, fmt.Errorf("base_url %q isn't an absolute http or https URL", e.BaseURL))
	} else if u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		errs = append(errs, fmt.Errorf("base_url %q has a query or fragment; set query instead", e.BaseURL))
	}
	return errors.Join(errs...)
}

// Values returns e's query as url.Values, nil when it has none.
func (e *Endpoint) Values() url.Values {
	if len(e.Query) == 0 {
		return nil
	}
	v := url.Values{}
	for k, s := range e.Query {
		v.Set(k, s)
	}
	return v
}

// Client returns a client of e, speaking its format over hc, its token sent on each request.
func (e *Endpoint) Client(hc *http.Client) (*client.Client, error) {
	cfg := client.Config{BaseURL: e.BaseURL, Query: e.Values(), HTTP: hc, Format: e.format()}
	if e.Token != "" {
		token := e.Token
		cfg.Token = func(context.Context) (string, error) { return token, nil }
	}
	return client.New(cfg)
}

// format returns the client format e names; validate has refused any other.
func (e *Endpoint) format() format.Format {
	switch e.Format {
	case Messages:
		return messages.New()
	case Responses:
		return responses.New(responses.Options{})
	default:
		return completions.New()
	}
}

// Capabilities returns the three capabilities by name, embeddings, vision, then transcription,
// the order the models table lists them, each with its Capability, nil for one the configuration
// leaves unset. The run runs them in another order: vision, transcription, then embeddings.
func (c *Config) Capabilities() []Named {
	named := []Named{{Name: "embeddings"}, {Name: "vision"}, {Name: "transcription"}}
	if c.Embeddings != nil {
		named[0].Capability = &c.Embeddings.Capability
	}
	if c.Vision != nil {
		named[1].Capability = &c.Vision.Capability
	}
	if c.Transcription != nil {
		named[2].Capability = &c.Transcription.Capability
	}
	return named
}

// Named is one capability by its name, with its Capability, nil when unset.
type Named struct {
	Name       string
	Capability *Capability
}
