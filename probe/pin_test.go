package probe_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// pinPath is the file that pins the router's llama.cpp build.
const pinPath = "../host/llama.env"

// pinVar is the variable of host/llama.env that sets gb10's llama.cpp pin, the build gb10-2's
// router runs at least: llama_pin gb10 reads it (host/llama-release.sh).
const pinVar = "LLAMA_BUILD_GB10"

// loadPin returns gb10's pin, pinVar's value, from the pin file at path, a shell file of
// NAME=value lines.
func loadPin(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	for line := range strings.Lines(string(b)) {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), pinVar+"="); ok {
			return v, nil
		}
	}
	return "", fmt.Errorf("%s sets no %s", path, pinVar)
}

// TestLoadPin checks that loadPin reads gb10's pin and nothing else: not another class's pin, not
// a comment naming the variable, and not the single pin, LLAMA_BUILD, that llama.env set before
// the pins were per class. It also reads the repository's own pin file, which must set a b-build.
func TestLoadPin(t *testing.T) {
	cases := []struct {
		name, file, want string
	}{
		{"per class", "LLAMA_RELEASE=v0.8.0\nLLAMA_BUILD_STRIX_HALO=b11960\nLLAMA_BUILD_GB10=b12098\n", "b12098"},
		{"indented", "  LLAMA_BUILD_GB10=b12098\n", "b12098"},
		{"comment only", "# LLAMA_BUILD_GB10 is gb10's pin\nLLAMA_BUILD_STRIX_HALO=b11960\n", ""},
		{"single pin", "LLAMA_BUILD=b11669\n", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "llama.env")
			if err := os.WriteFile(path, []byte(c.file), 0o600); err != nil {
				t.Fatal(err)
			}
			got, err := loadPin(path)
			if c.want == "" {
				if err == nil {
					t.Fatalf("loadPin = %q, want an error", got)
				}
				return
			}
			if err != nil || got != c.want {
				t.Fatalf("loadPin = %q, %v, want %q", got, err, c.want)
			}
		})
	}

	t.Run("repository", func(t *testing.T) {
		got, err := loadPin(pinPath)
		if err != nil {
			t.Fatal(err)
		}
		if !regexp.MustCompile(`^b[0-9]+$`).MatchString(got) {
			t.Errorf("%s %s = %q, want a b-build such as b12098", pinPath, pinVar, got)
		}
	})
}
