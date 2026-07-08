package langserver

import (
	"log"
	"os"
	"path/filepath"
	"testing"
)

func TestFormattingRequireRootMatcher(t *testing.T) {
	base, _ := os.Getwd()
	file := filepath.Join(base, "foo")
	uri := toURI(file)

	h := &langHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		rootPath: base,
		configs: map[string][]Language{
			"vim": {
				{
					LintCommand:        `echo ` + file + `:2:No it is normal!`,
					LintIgnoreExitCode: true,
					LintAfterOpen:      true,
					LintStdin:          true,
					RequireMarker:      true,
					RootMarkers:        []string{".vimlintrc"},
				},
			},
		},
		files: map[DocumentURI]*File{
			uri: {
				LanguageID: "vim",
				Text:       "scriptencoding utf-8\nabnormal!\n",
			},
		},
	}

	rng := Range{Position{-1, -1}, Position{-1, -1}}
	d, err := h.rangeFormatRequest(uri, rng, FormattingOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(d) != 0 {
		t.Fatal("text edits should be zero as we have no root marker for the language but require one", d)
	}
}

func TestFormattingCRLF(t *testing.T) {
	base, _ := os.Getwd()
	file := filepath.Join(base, "foo")
	uri := toURI(file)

	original := "a\r\nb\r\n"
	h := &langHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		rootPath: base,
		configs: map[string][]Language{
			"vim": {
				{
					FormatCommand: `echo a`,
					FormatStdin:   true,
				},
			},
		},
		files: map[DocumentURI]*File{
			uri: {
				LanguageID: "vim",
				Text:       original,
			},
		},
	}

	rng := Range{Position{-1, -1}, Position{-1, -1}}
	edits, err := h.rangeFormatting(uri, rng, FormattingOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got := applyEdits(t, original, edits); got != "a\r\n" {
		t.Fatalf("applying edits should produce %q but got: %q", "a\r\n", got)
	}
}
