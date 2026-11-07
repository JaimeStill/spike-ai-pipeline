// Package config is docinfer's configuration, read through go-core's layered JSON load from the
// directory --config names: config.json, its overlay config.<env>.json, secrets.json, and its
// overlay secrets.<env>.json, later files winning, DOCINFER_ENV naming <env>, then the DOCINFER_
// environment overrides. A key the configuration doesn't declare stops the load.
//
//	{
//	  "endpoints": {
//	    "gb10-2": {"base_url": "http://gb10-2:8080/v1", "format": "completions",
//	               "query": {"autoload": "false"}}
//	  },
//	  "embeddings":    {"endpoint": "gb10-2", "model": "...", "batch": 16, "chunk_size": 2000},
//	  "vision":        {"endpoint": "gb10-2", "model": "...", "max_tokens": 2048},
//	  "transcription": {"endpoint": "gb10-2", "model": "..."}
//	}
//
// Each endpoint is a model server's API root under a name of the caller's choosing, the wire
// format generations are sent to it in, "completions", "messages", or "responses", its bearer
// token, kept in secrets.json, and a query added to every request, such as llama.cpp's router's
// autoload=false, which the client's Query carries. There is no default endpoint.
//
// Each capability names an endpoint and a model there, and how many requests it keeps in flight,
// "concurrency", 4 by default. Embeddings is required: "batch", the chunks one request carries,
// 16 by default, and "chunk_size", the most characters in a chunk, 2,000 by default. Vision is
// optional: "prompt", what is asked of each image, by default "Transcribe the text in this
// picture. If it has no text, describe it in one sentence.", and "max_tokens", its reply's cap,
// 2048 by default. Transcription is optional, with an optional "prompt" and "language", empty by
// default, which send none. A capability the configuration leaves unset is skipped.
//
// Each setting has an override, DOCINFER_ and its path upper-cased: DOCINFER_EMBEDDINGS_MODEL,
// DOCINFER_VISION_MAX_TOKENS, DOCINFER_ENDPOINTS_GB10_2_TOKEN for the endpoint "gb10-2", and so
// on. An override of a capability the files leave unset sets it; an endpoint is overridden only
// when the files name it.
//
// Every error comes from [Load], before any request: no embeddings capability, a capability
// naming an endpoint the configuration doesn't hold, a format that isn't one of the three, a base
// URL that isn't an absolute http or https URL without a query or fragment, a count below 1, or
// an override that doesn't parse.
//
// The package exports:
//
//   - [Config], with its [Endpoint]s and its [Embeddings], [Vision], and [Transcription]
//     capabilities, each a [Capability] and its own settings; [Env] and its parts name the
//     overrides Finalize read
//   - [Load], which reads and validates a directory's configuration, through Config's
//     [Config.Merge] and [Config.Finalize]
//   - [Config.Capabilities], the capabilities by name, as [Named]
//   - [Endpoint.Values], an endpoint's query, and [Endpoint.Client], a client of it
//   - [EnvPrefix], the defaults, such as [DefaultVisionPrompt], and the format names,
//     [Completions], [Messages], and [Responses]
package config
