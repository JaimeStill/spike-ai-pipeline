package report_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/JaimeStill/spike-ai-pipeline/internal/docproc/report"
	"github.com/JaimeStill/spike-ai-pipeline/internal/docproc/schema"
)

// update rewrites the report's golden page from the render instead of comparing against it.
var update = flag.Bool("update", false, "rewrite testdata/report.golden.html")

// goldenRecord is the record internal/docproc/app's test pins, its times written <time> and its
// inputs' directory <in>.
var goldenRecord = filepath.Join("..", "app", "testdata", "record.golden.jsonl")

// placeholder matches the golden record's normalized times.
var placeholder = regexp.MustCompile(`"<time>"`)

// fixture returns the committed manifest and the committed golden record, its inputs' directory
// written /in and each of its times, in order, a millisecond after the one before, so every span
// has a length.
func fixture(t *testing.T) (schema.Manifest, []schema.Event) {
	t.Helper()
	var m schema.Manifest
	b, err := os.ReadFile(filepath.Join("testdata", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	rec, err := os.ReadFile(goldenRecord)
	if err != nil {
		t.Fatal(err)
	}
	base, n := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC), 0
	rec = placeholder.ReplaceAllFunc(rec, func([]byte) []byte {
		n++
		return []byte(`"` + schema.Time(base.Add(time.Duration(n)*time.Millisecond)).String() + `"`)
	})
	rec = bytes.ReplaceAll(rec, []byte("<in>"), []byte("/in"))
	return m, decode(t, string(rec))
}

// decode decodes record lines.
func decode(t *testing.T, lines string) []schema.Event {
	t.Helper()
	var evs []schema.Event
	dec := json.NewDecoder(strings.NewReader(lines))
	for dec.More() {
		var ev schema.Event
		if err := dec.Decode(&ev); err != nil {
			t.Fatal(err)
		}
		evs = append(evs, ev)
	}
	return evs
}

// render renders m and evs, failing the test on an error.
func render(t *testing.T, m schema.Manifest, evs []schema.Event) string {
	t.Helper()
	var buf bytes.Buffer
	if err := report.Render(&buf, m, evs); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// The page rendered from the record internal/docproc/app pins: a quarantined input with its flags
// and kept file, a text input with its output linked, an archive with a part found in it nested
// below, and an input whose dispatch stopped with an error.
func TestRender_Golden(t *testing.T) {
	m, evs := fixture(t)

	got := render(t, m, evs)

	golden := filepath.Join("testdata", "report.golden.html")
	if *update {
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("report differs from %s; go test ./internal/docproc/report -update rewrites it, and git diff shows how", golden)
	}
}

// The same manifest and record render the same bytes, and so does a record whose inputs'
// lines interleave in another order, as another run's workers would write them.
func TestRender_Deterministic(t *testing.T) {
	m, evs := fixture(t)
	first := render(t, m, evs)

	if again := render(t, m, evs); again != first {
		t.Error("a second render differs from the first")
	}
	// Each input's own lines keep their order; the inputs' order is reversed.
	var byInput [][]schema.Event
	for _, ev := range evs {
		if len(byInput) == 0 || byInput[len(byInput)-1][0].Input != ev.Input {
			byInput = append(byInput, nil)
		}
		byInput[len(byInput)-1] = append(byInput[len(byInput)-1], ev)
	}
	var reordered []schema.Event
	for _, b := range slices.Backward(byInput) {
		reordered = append(reordered, b...)
	}
	if got := render(t, m, reordered); got != first {
		t.Error("the inputs' order in the record changes the page")
	}
}

// A name, a reason, or any other text from the record is escaped: a part named <script> stays
// text. An image output shows inline and a text output is linked, each by its Ref relative to
// the report.
func TestRender_EscapesAndOutputs(t *testing.T) {
	m, _ := fixture(t)
	evs := decode(t, `
{"event":"output","input":"/in/<b>.zip","part":{"name":"<script>alert(1)</script>","size":3,"sha256":"aa","content_type":{"route":"picture"},"parent":"bb","depth":1},"output":{"kind":"image","name":"page \"1\"","media_type":"image/png","ref":"outputs/aa/picture-1/1.png"}}
{"event":"output","input":"/in/<b>.zip","part":{"name":"<script>alert(1)</script>","size":3,"sha256":"aa","content_type":{"route":"picture"},"parent":"bb","depth":1},"output":{"kind":"markdown","media_type":"text/markdown","ref":"outputs/aa/picture-1/2.md"}}
{"event":"part","input":"/in/<b>.zip","part":{"name":"<script>alert(1)</script>","size":3,"sha256":"aa","content_type":{"route":"picture"},"parent":"bb","depth":1},"outcome":"prepared","reason":"<i>why</i>","outputs":2,"children":0}
{"event":"part","input":"/in/<b>.zip","part":{"name":"<b>.zip","size":9,"sha256":"bb","content_type":{"route":"archive"},"depth":0},"outcome":"prepared","outputs":0,"children":1}
`)

	got := render(t, m, evs)

	for _, raw := range []string{"<script>alert", "<i>why", "<b>.zip"} {
		if strings.Contains(got, raw) {
			t.Errorf("the page holds %q unescaped", raw)
		}
	}
	for _, want := range []string{
		"&lt;script&gt;alert(1)&lt;/script&gt;",
		"&lt;i&gt;why&lt;/i&gt;",
		`<img src="../../outputs/aa/picture-1/1.png" alt="page &#34;1&#34;" loading="lazy">`,
		`<a href="../../outputs/aa/picture-1/2.md">markdown</a>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the page has no %s", want)
		}
	}
	if strings.Contains(got, `href="../../outputs/aa/picture-1/1.png">page`) {
		t.Error("the image output is linked as text as well as shown inline")
	}
}

// The tree nests each part under its parent's own event by SHA-256 and depth, keeps the
// record's order among siblings, gives each part the outputs that came before its own event,
// and leaves a part whose parent never had its own event at the top.
func TestRender_Tree(t *testing.T) {
	m, _ := fixture(t)
	line := func(event, sha, parent string, depth int) string {
		out := ""
		if event == schema.OutputEvent {
			out = fmt.Sprintf(`,"output":{"kind":"text","ref":"outputs/%s/t/1.txt"}`, sha)
		}
		return fmt.Sprintf(`{"event":%q,"input":"/in/a","part":{"name":%q,"size":1,"sha256":%q,"content_type":{},"parent":%q,"depth":%d}%s}`+"\n",
			event, sha, sha, parent, depth, out)
	}
	evs := decode(t, line("output", "c1", "b", 2)+
		line("part", "c1", "b", 2)+
		line("part", "b", "a", 1)+
		line("output", "d", "a", 1)+
		line("part", "d", "a", 1)+
		line("part", "c2", "x", 2)+ // its parent never has its own event
		line("part", "a", "", 0))

	got := shape(render(t, m, evs))

	if want := "c2:1 a:1 b:2 c1:3 [c1] d:2 [d]"; got != want {
		t.Errorf("tree = %s, want %s", got, want)
	}
}

// tags matches what shape reads of a page: list tags, part names, and output links.
var tags = regexp.MustCompile(`<(/?)(ul|li)\b([^>]*)>|<span class="name">([^<]*)</span>|href="\.\./\.\./outputs/([^/]*)/`)

// shape returns the part trees of page in document order, each part as name:depth, its depth
// the number of tree lists around it, and each output link as [sha].
func shape(page string) string {
	var stack []bool // each open list or item, true for a tree list
	var out []string
	for _, m := range tags.FindAllStringSubmatch(page, -1) {
		switch {
		case m[2] != "" && m[1] == "":
			stack = append(stack, m[2] == "ul" && strings.Contains(m[3], `class="tree"`))
		case m[2] != "":
			stack = stack[:len(stack)-1]
		case m[5] != "":
			out = append(out, "["+m[5]+"]")
		default:
			depth := 0
			for _, tree := range stack {
				if tree {
					depth++
				}
			}
			out = append(out, fmt.Sprintf("%s:%d", m[4], depth))
		}
	}
	return strings.Join(out, " ")
}
