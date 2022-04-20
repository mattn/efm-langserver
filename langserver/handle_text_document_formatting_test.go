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

	d, err := h.formatRequest(uri, FormattingOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(d) != 0 {
		t.Fatal("text edits should be zero as we have no root marker for the language but require one", d)
	}
}
