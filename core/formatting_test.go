package core

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/konradmalik/efm-langserver/types"
)

func TestFormattingRequireRootMatcher(t *testing.T) {
	base, _ := os.Getwd()
	filepath := filepath.Join(base, "foo")
	uri := toURI(filepath)

	h := &LangHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		RootPath: base,
		configs: map[string][]types.Language{
			"vim": {
				{
					LintCommand:        `echo ` + filepath + `:2:No it is normal!`,
					LintIgnoreExitCode: true,
					LintAfterOpen:      true,
					LintStdin:          true,
					RequireMarker:      true,
					RootMarkers:        []string{".vimlintrc"},
				},
			},
		},
		files: map[types.DocumentURI]*fileRef{
			uri: {
				LanguageID: "vim",
				Text:       "scriptencoding utf-8\nabnormal!\n",
			},
		},
	}

	rng := types.Range{Start: types.Position{Line: -1, Character: -1}, End: types.Position{Line: -1, Character: -1}}
	d, err := h.RangeFormatRequest(uri, rng, types.FormattingOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(d) != 0 {
		t.Fatal("text edits should be zero as we have no root marker for the language but require one", d)
	}
}
