// Package models is docinfer's check, as it starts, that each capability's model is served: for
// each endpoint a capability uses, it GETs the endpoint's /models listing, once, with the
// endpoint's query and token, and finds each model there. The client has no listing, so the
// check is net/http and encoding/json over the OpenAI listing, {"data": [{"id": ...}]}, reading
// the "status": {"value": ...} llama.cpp's router adds to each model.
//
// A model the listing doesn't name, and an endpoint the request can't reach, stop docinfer. An
// endpoint that gives no listing, a reply that isn't 2xx or holds no "data" list, is a warning,
// its models unchecked; so is a listed model whose status isn't "loaded", such as the router's
// "unloaded" for a model it would load only on request, which it doesn't under
// --no-models-autoload. A capability the configuration leaves unset is skipped. The table shows
// each capability's endpoint, model, and state: "loaded", "listed" for a model listed with no
// status, the status the endpoint reports, such as "unloaded", "unlisted", "unchecked",
// "unreachable", or "skipped".
//
// The package exports:
//
//   - [Check], which lists the endpoints and returns a [Report] of a [Row] for each capability,
//     and its warnings
//   - [Report.Write], which writes the table and the warnings
//   - the states: [Loaded], [Listed], [Unlisted], [Unchecked], [Unreachable], and [Skipped]
package models
