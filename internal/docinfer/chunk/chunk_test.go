package chunk_test

import (
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/JaimeStill/spike-ai-pipeline/internal/docinfer/chunk"
)

// texts returns each chunk's text.
func texts(cs []chunk.Chunk) []string {
	var out []string
	for _, c := range cs {
		out = append(out, c.Text)
	}
	return out
}

// valid fails the test unless every chunk of text is within bound runes, is the text's bytes at
// its span, is whole UTF-8 where text is, has no white space at its ends, and comes after the one
// before it without overlapping it.
func valid(t *testing.T, text string, bound int, cs []chunk.Chunk) {
	t.Helper()
	last := 0
	for i, c := range cs {
		if n := utf8.RuneCountInString(c.Text); n > bound || n == 0 {
			t.Errorf("chunk %d %q has %d runes, want 1 to %d", i, c.Text, n, bound)
		}
		if c.Start < last || c.End <= c.Start || c.End > len(text) {
			t.Fatalf("chunk %d span [%d, %d) after %d in %d bytes, want ordered and inside", i, c.Start, c.End, last, len(text))
		}
		if text[c.Start:c.End] != c.Text {
			t.Errorf("chunk %d text %q, want the span's %q", i, c.Text, text[c.Start:c.End])
		}
		if utf8.ValidString(text) && !utf8.ValidString(c.Text) {
			t.Errorf("chunk %d %q splits a UTF-8 character", i, c.Text)
		}
		if strings.TrimSpace(c.Text) != c.Text {
			t.Errorf("chunk %d %q has white space at an end", i, c.Text)
		}
		last = c.End
	}
}

func TestSplit_Empty(t *testing.T) {
	for _, text := range []string{"", " ", "\n\n\t  \n", "  "} {
		if cs := chunk.Split(text, 10); len(cs) != 0 {
			t.Errorf("Split(%q) = %v, want no chunk", text, cs)
		}
	}
}

// A text within the bound is one chunk, trimmed, its span past the white space.
func TestSplit_Fits(t *testing.T) {
	text := "\n  Hello, world.  \n"
	got := chunk.Split(text, 100)
	want := []chunk.Chunk{{Text: "Hello, world.", Start: 3, End: 16}}
	if !slices.Equal(got, want) {
		t.Errorf("Split = %v, want %v", got, want)
	}
}

// Sections are cut before each heading and packed whole while they fit; a heading inside a fenced
// code block is no cut.
func TestSplit_Headings(t *testing.T) {
	text := "# One\nalpha beta\n## Two\ngamma\n# Three\n```\n# not a heading\n```\nend"
	got := chunk.Split(text, 40)
	valid(t, text, 40, got)
	want := []string{"# One\nalpha beta\n## Two\ngamma", "# Three\n```\n# not a heading\n```\nend"}
	if !slices.Equal(texts(got), want) {
		t.Errorf("Split = %q, want %q", texts(got), want)
	}
}

// "#tag" and "####### seven" open no heading, so the section isn't cut there.
func TestSplit_NotHeadings(t *testing.T) {
	text := "# A\nxxxx\n#tag yyyy\n####### seven\nzzzz"
	got := chunk.Split(text, 30)
	valid(t, text, 30, got)
	// No heading cut and no blank line: the section is cut into lines, packed.
	want := []string{"# A\nxxxx\n#tag yyyy", "####### seven\nzzzz"}
	if !slices.Equal(texts(got), want) {
		t.Errorf("Split = %q, want %q", texts(got), want)
	}
}

// A section too long for the bound is cut at its blank lines, the paragraphs packed.
func TestSplit_Paragraphs(t *testing.T) {
	text := "First paragraph here.\n\nSecond one.\n\n\nThird paragraph, longer."
	got := chunk.Split(text, 40)
	valid(t, text, 40, got)
	want := []string{"First paragraph here.\n\nSecond one.", "Third paragraph, longer."}
	if !slices.Equal(texts(got), want) {
		t.Errorf("Split = %q, want %q", texts(got), want)
	}
}

// A paragraph too long for the bound is cut after its sentences, closing quotes kept with theirs,
// a full stop inside a word, such as a version's, no cut.
func TestSplit_Sentences(t *testing.T) {
	text := `He said "go." Then v1.2 shipped! Did it work? Yes。Done`
	got := chunk.Split(text, 20)
	valid(t, text, 20, got)
	want := []string{`He said "go."`, "Then v1.2 shipped!", "Did it work? Yes。", "Done"}
	if !slices.Equal(texts(got), want) {
		t.Errorf("Split = %q, want %q", texts(got), want)
	}
}

// A sentence too long for the bound is cut every bound runes.
func TestSplit_HardCut(t *testing.T) {
	text := "abcdefghijklmnopqrstuvwxyz"
	got := chunk.Split(text, 10)
	valid(t, text, 10, got)
	want := []string{"abcdefghij", "klmnopqrst", "uvwxyz"}
	if !slices.Equal(texts(got), want) {
		t.Errorf("Split = %q, want %q", texts(got), want)
	}
}

// The bound counts runes, not bytes, and a hard cut never falls inside a character.
func TestSplit_Multibyte(t *testing.T) {
	text := "日本語のテキストです😀😀😀🎉🎉"
	got := chunk.Split(text, 4)
	valid(t, text, 4, got)
	want := []string{"日本語の", "テキスト", "です😀😀", "😀🎉🎉"}
	if !slices.Equal(texts(got), want) {
		t.Errorf("Split = %q, want %q", texts(got), want)
	}
	// Five three-byte runes fit a bound of five, though they are fifteen bytes.
	if got := chunk.Split("ééééé", 5); len(got) != 1 {
		t.Errorf("Split(ééééé, 5) = %q, want one chunk", texts(got))
	}
}

// An invalid byte counts as one character, and a span still reproduces it.
func TestSplit_InvalidUTF8(t *testing.T) {
	text := "ab\xffcd\xfe"
	got := chunk.Split(text, 2)
	valid(t, text, 2, got)
	want := []string{"ab", "\xffc", "d\xfe"}
	if !slices.Equal(texts(got), want) {
		t.Errorf("Split = %q, want %q", texts(got), want)
	}
}

// Every chunk's span reproduces the text's slice exactly, the chunks cover the text's every
// character but white space, and the same text and bound give the same chunks.
func TestSplit_SpansAndDeterminism(t *testing.T) {
	text := strings.Repeat("# Heading\n\nSome text. More text, with ünïcödé! Another line\n- item one\n- item two\n\n", 20) +
		strings.Repeat("x", 333)
	for _, bound := range []int{1, 7, 20, 64, 500, 5000} {
		got := chunk.Split(text, bound)
		valid(t, text, bound, got)
		var joined strings.Builder
		for _, c := range got {
			joined.WriteString(c.Text)
		}
		if want := strings.Join(strings.Fields(text), ""); strings.Join(strings.Fields(joined.String()), "") != want {
			t.Errorf("bound %d: chunks drop non-space text", bound)
		}
		if again := chunk.Split(text, bound); !slices.Equal(got, again) {
			t.Errorf("bound %d: Split differs on a second call", bound)
		}
	}
}

// A bound below 1 is taken as 1.
func TestSplit_BoundBelowOne(t *testing.T) {
	got := chunk.Split("ab", 0)
	if want := []string{"a", "b"}; !slices.Equal(texts(got), want) {
		t.Errorf("Split(ab, 0) = %q, want %q", texts(got), want)
	}
}
