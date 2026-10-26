// Package probe holds the HTTP probe of gb10-2's llama.cpp router serving set A. The probe is a
// test behind the integration build tag, run by `mise run integration` against the router
// SPIKE_ROUTER_URL names; the hermetic check builds and lints it but never runs it. Its
// expectations come from the files that configure the router, not the probe: the set and each
// model's sampling and slots from the preset, engines/llama-preset/models.ini, and the build
// floor from the pin, engines/llama.env.
package probe
