package rundir

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"uuid"

	"github.com/JaimeStill/spike-ai-pipeline-processing/dispatch"

	"github.com/JaimeStill/spike-ai-pipeline/internal/schema"
)

// The run directory's subdirectories.
const (
	indexDir      = "index"
	outputsDir    = "outputs"
	quarantineDir = "quarantine"
	runsDir       = "runs"
)

// Dir is a run directory: its file index, its output store, its quarantine, and its runs.
type Dir struct {
	root  string
	index *dispatch.FileIndex
}

// Open opens the run directory at root, making it and its file index when they don't exist, or
// reusing the index an earlier run left there.
func Open(root string) (*Dir, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, err
	}
	idx, err := dispatch.NewFileIndex(filepath.Join(abs, indexDir), dispatch.FileOptions{})
	if err != nil {
		return nil, err
	}
	return &Dir{root: abs, index: idx}, nil
}

// Root returns the run directory's absolute path.
func (d *Dir) Root() string { return d.root }

// Index returns the run directory's file index, which every run over it shares.
func (d *Dir) Index() dispatch.Index { return d.index }

// Rel returns p relative to the run directory, with slashes, as the record writes every path.
func (d *Dir) Rel(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(d.root, abs)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}

// Store stores o, the nth output, counting from 1, that the stage id prepared for part, and
// returns its Ref: outputs/<part sha256>/<stage>/<n><ext>, relative to the run directory. The
// content is o's file, copied, or its inline Value. An output already stored there, as an
// earlier preparation of the same content under the same stage left it, is kept as it is: a
// file is only ever written whole, to a temporary name and renamed.
func (d *Dir) Store(part dispatch.Part, id dispatch.Identity, n int, o dispatch.Output) (string, error) {
	ref := path.Join(outputsDir, part.SHA256, stageDir(id), strconv.Itoa(n)+ext(o))
	dst := filepath.Join(d.root, filepath.FromSlash(ref))
	if exists(dst) {
		return ref, nil
	}
	if o.Path != "" {
		return ref, copyFile(dst, o.Path)
	}
	return ref, writeFile(dst, func(w io.Writer) error {
		_, err := w.Write(o.Value)
		return err
	})
}

// Quarantine keeps the file of part, which its inspection or its stage quarantined, beside its
// flags, under quarantine/<sha256>/, and returns where, relative to the run directory. A part
// already kept there is kept as it is: the flags are written last, so they mark a whole keep.
func (d *Dir) Quarantine(part dispatch.Part) (schema.Quarantine, error) {
	dir := path.Join(quarantineDir, part.SHA256)
	q := schema.Quarantine{File: path.Join(dir, "content"), Flags: path.Join(dir, "flags.json")}
	if exists(filepath.Join(d.root, filepath.FromSlash(q.Flags))) {
		return q, nil
	}
	if err := copyFile(filepath.Join(d.root, filepath.FromSlash(q.File)), part.Path); err != nil {
		return schema.Quarantine{}, err
	}
	flags := schema.PartOf(part).Flags
	if flags == nil {
		flags = []schema.Flag{}
	}
	err := writeFile(filepath.Join(d.root, filepath.FromSlash(q.Flags)), func(w io.Writer) error {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(flags)
	})
	if err != nil {
		return schema.Quarantine{}, err
	}
	return q, nil
}

// Run is one run over a run directory: its id, its manifest, and its record, under
// runs/<run id>/. It is not safe for concurrent use.
type Run struct {
	dir      string
	manifest schema.Manifest
	file     *os.File
	buf      *bufio.Writer
	enc      *json.Encoder
}

// Begin begins a run over d under a new id, a version 7 UUID, started at start: it makes the
// run's directory, writes its manifest from m, setting its schema, id, start, and record, and
// opens its record.
func (d *Dir) Begin(m schema.Manifest, start time.Time) (*Run, error) {
	id := uuid.NewV7().String()
	rel := path.Join(runsDir, id)
	dir := filepath.Join(d.root, filepath.FromSlash(rel))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	m.Schema, m.Run, m.Start, m.Record = schema.Version, id, schema.Time(start), path.Join(rel, "record.jsonl")
	r := &Run{dir: dir, manifest: m}
	if err := r.writeManifest(); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, "record.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return nil, err
	}
	r.file, r.buf = f, bufio.NewWriter(f)
	r.enc = json.NewEncoder(r.buf)
	r.enc.SetEscapeHTML(false)
	return r, nil
}

// ID returns the run's id.
func (r *Run) ID() string { return r.manifest.Run }

// Record returns the run's record, relative to the run directory.
func (r *Run) Record() string { return r.manifest.Record }

// Write appends ev to the run's record, one line.
func (r *Run) Write(ev schema.Event) error { return r.enc.Encode(ev) }

// End closes the run's record and writes its manifest again with its end, and, when the whole
// dispatch stopped, the error that stopped it.
func (r *Run) End(end time.Time, stopped error) error {
	err := errors.Join(r.buf.Flush(), r.file.Sync(), r.file.Close())
	r.manifest.End = schema.Time(end)
	if stopped != nil {
		r.manifest.Error = stopped.Error()
	}
	return errors.Join(err, r.writeManifest())
}

// writeManifest writes the run's manifest whole, to a temporary name and renamed over the one
// written before.
func (r *Run) writeManifest() error {
	return writeFile(filepath.Join(r.dir, "manifest.json"), func(w io.Writer) error {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		return enc.Encode(r.manifest)
	})
}

// stageDir returns the output store's directory name for the stage id: its name and the first
// twelve hex digits of the SHA-256 of its whole identity, so a new tool version or configuration
// stores its outputs apart.
func stageDir(id dispatch.Identity) string {
	sum := sha256.Sum256([]byte(id.String()))
	name := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			return r
		}
		return '-'
	}, strings.ToLower(id.Name))
	return name + "-" + hex.EncodeToString(sum[:])[:12]
}

// extensions are the stored file extensions of the media types processing's stages give their
// outputs. A fixed table, rather than the host's MIME database, keeps a Ref the same on every
// host.
var extensions = map[string]string{
	"text/plain":       ".txt",
	"text/markdown":    ".md",
	"application/json": ".json",
	"image/png":        ".png",
	"image/jpeg":       ".jpg",
	"image/gif":        ".gif",
	"image/bmp":        ".bmp",
	"audio/wav":        ".wav",
	"audio/x-wav":      ".wav",
	"audio/wave":       ".wav",
}

// ext returns the extension o's stored file takes: its media type's, else its file's own, else
// none.
func ext(o dispatch.Output) string {
	if mt, _, err := mime.ParseMediaType(o.MediaType); err == nil {
		if e, ok := extensions[mt]; ok {
			return e
		}
	}
	if o.Path != "" {
		return strings.ToLower(filepath.Ext(o.Path))
	}
	return ""
}

// exists reports whether p exists, as a file or anything else.
func exists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

// copyFile copies src to dst, as writeFile writes it.
func copyFile(dst, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }() // read only: a close error loses nothing
	return writeFile(dst, func(w io.Writer) error {
		_, err := io.Copy(w, in)
		return err
	})
}

// writeFile writes dst whole through write, to a temporary file in dst's directory, made when it
// doesn't exist, synced and renamed into place, replacing any dst, so a reader sees all of dst
// or none of it.
func writeFile(dst string, write func(io.Writer) error) (err error) {
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tmp-")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = os.Remove(tmp.Name()) // the write failed already; the leftover is the lesser loss
		}
	}()
	if err := write(tmp); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write %s: %w", dst, err)
	}
	if err := errors.Join(tmp.Sync(), tmp.Close()); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), dst)
}
