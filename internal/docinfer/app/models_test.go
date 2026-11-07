package app_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"

	"github.com/standards-lab/go-core/process"

	docinfertest "github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/apptest"
)

// listings returns the requests of reqs that are /models listings, leaving out the run's
// embeddings.
func listings(reqs []docinfertest.Request) []docinfertest.Request {
	var out []docinfertest.Request
	for _, r := range reqs {
		if r.Path == "/v1/models" {
			out = append(out, r)
		}
	}
	return out
}

// A model its endpoint's listing doesn't name stops the run with exit 1, once the table shows
// which, before the docproc run's outputs are counted.
func TestRun_ModelUnlisted(t *testing.T) {
	clean(t)
	e := docinfertest.Serve(t, docinfertest.Fake{Models: []docinfertest.Model{
		{ID: embeddingModel, Status: "loaded"}, {ID: audioModel, Status: "loaded"},
	}})
	cfg := configDir(t, map[string]string{"config.json": fullConfig(e.URL())})

	stdout, stderr := run(t, process.ExitFailure, "run", docproc, "--config", cfg, "--out", t.TempDir())

	wantLine(t, stdout, row("embeddings", "fake", embeddingModel, "loaded"))
	wantLine(t, stdout, row("vision", "fake", visionModel, "unlisted"))
	wantLine(t, stdout, row("transcription", "fake", audioModel, "loaded"))
	if want := `vision: endpoint "fake" doesn't list model "` + visionModel + `"`; !strings.Contains(stderr, want) {
		t.Errorf("stderr = %q, want it to contain %q", stderr, want)
	}
	if strings.Contains(stdout, "docproc run") {
		t.Errorf("stdout counts the outputs, want the run stopped first:\n%s", stdout)
	}
}

// An endpoint with no listing is a warning, its models unchecked, and the run goes on.
func TestRun_NoListing(t *testing.T) {
	clean(t)
	e := docinfertest.Serve(t, docinfertest.Fake{NoListing: true})
	cfg := configDir(t, map[string]string{"config.json": fullConfig(e.URL())})

	stdout, _ := run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", t.TempDir())

	for _, c := range []struct{ capability, model string }{{"embeddings", embeddingModel}, {"vision", visionModel}, {"transcription", audioModel}} {
		wantLine(t, stdout, row(c.capability, "fake", c.model, "unchecked"))
	}
	wantLine(t, stdout, `warning: endpoint "fake" gives no model listing \(GET http://[^ ]+/v1/models: 404 Not Found\); its models are unchecked`)
	if n := strings.Count(stdout, "warning:"); n != 1 {
		t.Errorf("stdout has %d warnings, want 1, the endpoint's alone:\n%s", n, stdout)
	}
	wantLine(t, stdout, `docproc run .*`)
}

// A listing that isn't the OpenAI shape is no listing, as a 404 is. The endpoint's other
// routes are a fake's, so the run's embeddings are answered.
func TestRun_ListingNotJSON(t *testing.T) {
	clean(t)
	fake, err := url.Parse(docinfertest.Serve(t, docinfertest.Fake{}).URL())
	if err != nil {
		t.Fatal(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(&url.URL{Scheme: fake.Scheme, Host: fake.Host})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			proxy.ServeHTTP(w, r)
			return
		}
		_, _ = w.Write([]byte("<html>models</html>"))
	}))
	t.Cleanup(srv.Close)
	cfg := configDir(t, map[string]string{"config.json": fullConfig(srv.URL + "/v1")})

	stdout, _ := run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", t.TempDir())

	wantLine(t, stdout, row("embeddings", "fake", embeddingModel, "unchecked"))
	wantLine(t, stdout, `warning: endpoint "fake" gives no model listing \(.*the reply holds no "data" list\); its models are unchecked`)
}

// A listed model whose status isn't loaded is a warning, its status the row's state; a model
// listed with no status, as an endpoint that isn't llama.cpp's router lists one, is listed.
func TestRun_ModelStatus(t *testing.T) {
	clean(t)
	e := docinfertest.Serve(t, docinfertest.Fake{Models: []docinfertest.Model{
		{ID: embeddingModel}, {ID: visionModel, Status: "loaded"}, {ID: audioModel, Status: "unloaded"},
	}})
	cfg := configDir(t, map[string]string{"config.json": fullConfig(e.URL())})

	stdout, _ := run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", t.TempDir())

	wantLine(t, stdout, row("embeddings", "fake", embeddingModel, "listed"))
	wantLine(t, stdout, row("vision", "fake", visionModel, "loaded"))
	wantLine(t, stdout, row("transcription", "fake", audioModel, "unloaded"))
	wantLine(t, stdout, `warning: transcription: endpoint "fake" lists model "`+audioModel+`" as unloaded`)
	if n := strings.Count(stdout, "warning:"); n != 1 {
		t.Errorf("stdout has %d warnings, want 1:\n%s", n, stdout)
	}
}

// Unset vision and transcription capabilities are skipped and listed; each endpoint a
// capability uses is listed once, and one no capability uses is never asked.
func TestRun_SkippedAndEndpointsListedOnce(t *testing.T) {
	clean(t)
	embed := docinfertest.Serve(t, docinfertest.Fake{Models: setA})
	unused := docinfertest.Serve(t, docinfertest.Fake{Models: setA})
	cfg := configDir(t, map[string]string{"config.json": fmt.Sprintf(`{
  "endpoints": {
    "embed": {"base_url": %q, "format": "completions"},
    "unused": {"base_url": %q, "format": "responses"}
  },
  "embeddings": {"endpoint": "embed", "model": %q}
}`, embed.URL(), unused.URL(), embeddingModel)})

	stdout, _ := run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", t.TempDir())

	wantLine(t, stdout, row("embeddings", "embed", embeddingModel, "loaded"))
	wantLine(t, stdout, row("vision", "-", "-", "skipped"))
	wantLine(t, stdout, row("transcription", "-", "-", "skipped"))
	want := []docinfertest.Request{{Method: "GET", Path: "/v1/models"}}
	if reqs := listings(embed.Requests()); len(reqs) != 1 || reqs[0] != want[0] {
		t.Errorf("embed requests = %v, want %v, with no query and no token", reqs, want)
	}
	if reqs := unused.Requests(); len(reqs) != 0 {
		t.Errorf("unused requests = %v, want none", reqs)
	}

	// Two capabilities on one endpoint list it once.
	cfg = configDir(t, map[string]string{"config.json": fullConfig(embed.URL())})
	run(t, process.ExitOK, "run", docproc, "--config", cfg, "--out", t.TempDir())
	if reqs := listings(embed.Requests()); len(reqs) != 2 {
		t.Errorf("embed requests = %v, want one more, for three capabilities", reqs)
	}
}

// An endpoint the listing's request can't reach stops the run, once the table shows it.
func TestRun_EndpointUnreachable(t *testing.T) {
	clean(t)
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL + "/v1"
	srv.Close()
	cfg := configDir(t, map[string]string{"config.json": fullConfig(url)})

	stdout, stderr := run(t, process.ExitFailure, "run", docproc, "--config", cfg, "--out", t.TempDir())

	wantLine(t, stdout, row("embeddings", "fake", embeddingModel, "unreachable"))
	wantLine(t, stdout, row("transcription", "fake", audioModel, "unreachable"))
	if !strings.Contains(stderr, `endpoint "fake": `) || strings.Count(stderr, `endpoint "fake"`) != 1 {
		t.Errorf("stderr = %q, want the endpoint's failure once", stderr)
	}
}
