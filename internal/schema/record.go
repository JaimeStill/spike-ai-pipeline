package schema

// The kinds of a record line, its "event" field.
const (
	// OutputEvent is a line for one output a part's stage prepared, or, for a part the index has
	// seen, one output it recorded.
	OutputEvent = "output"
	// PartEvent is a line for a part's own event: its outcome, once every output and every part
	// found in it has its line.
	PartEvent = "part"
	// ErrorEvent is a line for an input whose dispatch stopped with an error before its own
	// event, such as an input that can't be read.
	ErrorEvent = "error"
)

// Event is one line of a run's record, runs/<run>/record.jsonl, a JSON object per line in the
// order the events arrived: an input's lines keep their order, the output and part lines of each
// part found in it before its own, and different inputs' lines interleave. Its fields:
//
//   - "event": the line's kind, "output", "part", or "error"
//   - "input": the input the line belongs to, its path relative to the run directory
//   - "part": the [Part] the line is about; absent from an "error" line
//   - "file": the input's [File]; present on an input's "part" line, and on an "error" line
//     when the file could still be read
//   - "stage": the [Stage] of the part's route; absent when its route has no stage
//   - "output": the [Output], on an "output" line alone
//   - "outcome": on a "part" line, what became of the part: "prepared", "no processor yet",
//     "seen", "cycle", "quarantined", "encrypted", "failed", or "limited"; on an "output" line,
//     "seen" for an output the index recorded earlier, absent for one prepared in this run
//   - "recorded": for a seen part or output, the outcome the index recorded for it when it was
//     prepared; optional
//   - "reason": why the part came to its outcome; optional
//   - "error": the error a failed or encrypted part came to, or that stopped an input's
//     dispatch on an "error" line; optional
//   - "outputs": on a "part" line, how many outputs the part yielded, or, seen, recorded
//   - "children": on a "part" line, how many parts its stage found in it
//   - "quarantine": on a quarantined part's "part" line, the [Quarantine], where its file is
//     kept; optional
//   - "time": the input's [Span], from when the pool took it to its last event; on an input's
//     "part" line and on an "error" line
type Event struct {
	Event      string      `json:"event"`
	Input      string      `json:"input"`
	Part       *Part       `json:"part,omitempty"`
	File       *File       `json:"file,omitempty"`
	Stage      *Stage      `json:"stage,omitempty"`
	Output     *Output     `json:"output,omitempty"`
	Outcome    string      `json:"outcome,omitzero"`
	Recorded   string      `json:"recorded,omitzero"`
	Reason     string      `json:"reason,omitzero"`
	Error      string      `json:"error,omitzero"`
	Outputs    *int        `json:"outputs,omitempty"`
	Children   *int        `json:"children,omitempty"`
	Quarantine *Quarantine `json:"quarantine,omitempty"`
	Time       *Span       `json:"time,omitempty"`
}

// Stage is a stage's identity, processing's dispatch.Identity: a part's outputs are prepared
// again whenever any of it changes.
//
//   - "name": the stage's name, such as "document"
//   - "version": its tools' versions, as processing's dispatch.Versions joins them, and its
//     adapter's
//   - "config": a hash of the options that change what it prepares
type Stage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Config  string `json:"config"`
}

// Output is one output a stage prepared, its content stored in the run directory's output
// store:
//
//   - "kind": "text", "markdown", "image", "record", or "audio"
//   - "name": the name the stage gave it, such as "page 3"; optional
//   - "media_type": its media type, such as "image/png"; optional
//   - "meta": what else the stage says of it, string to string, such as a page's number;
//     optional
//   - "ref": its stored file, relative to the run directory, at
//     outputs/<part sha256>/<stage>/<n><ext>; a seen output's ref names the copy the run that
//     prepared it stored
type Output struct {
	Kind      string            `json:"kind"`
	Name      string            `json:"name,omitzero"`
	MediaType string            `json:"media_type,omitzero"`
	Meta      map[string]string `json:"meta,omitempty"`
	Ref       string            `json:"ref"`
}

// Quarantine is where a quarantined part's file is kept, in the run directory, beside its
// flags, under quarantine/<part sha256>/:
//
//   - "file": the part's content, as "content", with no extension
//   - "flags": its flags, a JSON array of [Flag], as "flags.json"
type Quarantine struct {
	File  string `json:"file"`
	Flags string `json:"flags"`
}

// Span is when an input's dispatch ran: "start", when the pool took it, a wait for a free worker
// included, and "end", when its last event arrived, each a [Time].
type Span struct {
	Start Time `json:"start"`
	End   Time `json:"end"`
}

// Manifest is a run's manifest, runs/<run>/manifest.json, one JSON object, written as the run
// starts and again as it ends:
//
//   - "schema": the schema's version, "1"
//   - "run": the run's id, a version 7 UUID, which sorts by when the run started
//   - "start": when the run started, a [Time]
//   - "end": when it ended, a [Time]; absent while it runs, or when it never ended
//   - "workers": how many inputs were dispatched at once
//   - "inputs": the path arguments, files and directories, each relative to the run directory
//   - "record": the run's record, relative to the run directory
//   - "tools": each [Tool] the run tried to build
//   - "stages": each route's [StageState], in the stage table's order
//   - "error": the error that stopped the whole dispatch; optional, absent when none did
type Manifest struct {
	Schema  string       `json:"schema"`
	Run     string       `json:"run"`
	Start   Time         `json:"start"`
	End     Time         `json:"end,omitzero"`
	Workers int          `json:"workers"`
	Inputs  []string     `json:"inputs"`
	Record  string       `json:"record"`
	Tools   []Tool       `json:"tools"`
	Stages  []StageState `json:"stages"`
	Error   string       `json:"error,omitzero"`
}

// Tool is one tool a run tried to build:
//
//   - "program": its program, such as "gs"
//   - "minimum": the minimum version processing declares for it
//   - "from": where it was found, a host path, or, for a tool run in a container, its image;
//     optional, absent when it wasn't found
//   - "version": its version; optional, absent when it wasn't built
//   - "error": why it wasn't built, missing or too old; optional
type Tool struct {
	Program string `json:"program"`
	Minimum string `json:"minimum"`
	From    string `json:"from,omitzero"`
	Version string `json:"version,omitzero"`
	Error   string `json:"error,omitzero"`
}

// StageState is what became of one route's stage:
//
//   - "route": the route, such as "document"
//   - "state": "wired", "missing", or "excluded"; a missing or excluded route's parts have no
//     processor yet
//   - "stage": the wired stage's [Stage]; optional, absent unless wired
//   - "tools": the wired stage's tools and their versions; optional, absent for a stage that
//     runs none
//   - "reason": why a missing stage is missing; optional
type StageState struct {
	Route  string `json:"route"`
	State  string `json:"state"`
	Stage  *Stage `json:"stage,omitempty"`
	Tools  string `json:"tools,omitzero"`
	Reason string `json:"reason,omitzero"`
}
