package langserver

import (
	"strings"
	"testing"
)

// applyEdits applies line/character based TextEdits to a document the way an
// LSP client would, for verifying that the produced edits are valid and give
// the expected result.
func applyEdits(t *testing.T, text string, edits []TextEdit) string {
	t.Helper()
	lines := strings.SplitAfter(text, "\n")
	offset := func(p Position) int {
		if p.Line >= len(lines) {
			if p.Line > len(lines) || p.Character != 0 {
				t.Fatalf("position %v is out of bounds for %q", p, text)
			}
			return len(text)
		}
		o := 0
		for i := 0; i < p.Line; i++ {
			o += len(lines[i])
		}
		line := lines[p.Line]
		if p.Character > len(strings.TrimSuffix(line, "\n")) {
			t.Fatalf("position %v is out of bounds for %q", p, text)
		}
		return o + p.Character
	}
	// Apply in reverse so earlier offsets stay valid.
	for i := len(edits) - 1; i >= 0; i-- {
		start := offset(edits[i].Range.Start)
		end := offset(edits[i].Range.End)
		text = text[:start] + edits[i].NewText + text[end:]
	}
	return text
}

func TestComputeEditsNoTrailingNewline(t *testing.T) {
	before := "export const test = () => {}"
	after := "export const test = () => {};\n"

	edits := ComputeEdits("file:///foo", before, after)
	for _, e := range edits {
		if e.Range.Start.Line > 0 || e.Range.End.Line > 0 {
			t.Fatalf("edit references a line past the end of the document: %+v", e)
		}
	}
	if got := applyEdits(t, before, edits); got != after {
		t.Fatalf("applying edits should produce %q but got: %q", after, got)
	}
}

func TestComputeEditsNoTrailingNewlineMultiline(t *testing.T) {
	before := "aaa\nbbb"
	after := "aaa\nccc\nddd\n"

	edits := ComputeEdits("file:///foo", before, after)
	for _, e := range edits {
		if e.Range.Start.Line > 1 || e.Range.End.Line > 1 {
			t.Fatalf("edit references a line past the end of the document: %+v", e)
		}
	}
	if got := applyEdits(t, before, edits); got != after {
		t.Fatalf("applying edits should produce %q but got: %q", after, got)
	}
}

func TestComputeEditsTrailingNewline(t *testing.T) {
	before := "foo \nbar\n"
	after := "foo\nbar\n"

	edits := ComputeEdits("file:///foo", before, after)
	if got := applyEdits(t, before, edits); got != after {
		t.Fatalf("applying edits should produce %q but got: %q", after, got)
	}
}
