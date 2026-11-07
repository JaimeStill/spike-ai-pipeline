package schema

import (
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// File is an input's file metadata, as the filesystem holds it:
//
//   - "name": its file name
//   - "arg": the path argument it came from, a file or a directory, relative to the run
//     directory
//   - "path": its path relative to that argument, with slashes; "." when the argument is the
//     file itself, so joining "arg" and "path" always names the file
//   - "size": its size in bytes
//   - "mode": its mode and permissions, as ls writes them, such as "-rw-r--r--"
//   - "modified": its modified time, a [Time]
//   - "created": its created time, statx's birth time, a [Time]; optional, absent where the
//     filesystem records none, and on any system but Linux
type File struct {
	Name     string `json:"name"`
	Arg      string `json:"arg"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Mode     string `json:"mode"`
	Modified Time   `json:"modified"`
	Created  Time   `json:"created,omitzero"`
}

// Stat is what a [Stater] reads of a file: its size, mode, and modified time, and its created
// time, zero when the filesystem records none.
type Stat struct {
	Size     int64
	Mode     fs.FileMode
	Modified time.Time
	Created  time.Time
}

// Stater reads a file's [Stat], never following a symbolic link. [Statx] is the production
// one; a test gives [FileOf] its own.
type Stater func(path string) (Stat, error)

// FileOf returns the metadata of the file at path, which came from the path argument arg, read
// through stat. It doesn't touch "arg": the caller makes it relative to the run directory.
func FileOf(path, arg string, stat Stater) (File, error) {
	st, err := stat(path)
	if err != nil {
		return File{}, err
	}
	rel, err := filepath.Rel(arg, path)
	if err != nil {
		return File{}, err
	}
	return File{
		Name:     filepath.Base(path),
		Arg:      arg,
		Path:     filepath.ToSlash(rel),
		Size:     st.Size,
		Mode:     st.Mode.String(),
		Modified: Time(st.Modified),
		Created:  Time(st.Created),
	}, nil
}

// detectLen is how much of a file net/http.DetectContentType reads.
const detectLen = 512

// Detect returns the MIME type net/http.DetectContentType names for the first 512 bytes of the
// file at path, such as "text/plain; charset=utf-8", or "application/octet-stream" when no rule
// matches.
func Detect(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }() // read only: a close error loses nothing
	head := make([]byte, detectLen)
	n, err := io.ReadFull(f, head)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", err
	}
	return http.DetectContentType(head[:n]), nil
}
