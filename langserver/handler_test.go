package langserver

import (
	"log"
	"runtime"
	"strings"
	"testing"
)

func TestLintNoLinter(t *testing.T) {
	h := &langHandler{
		logger:  log.New(log.Writer(), "", log.LstdFlags),
		configs: map[string][]Language{},
		files: map[string]*File{
			"file:///foo": &File{},
		},
	}

	_, err := h.lint("file:///foo")
	if err != nil {
		t.Fatal("Should not be an error if no linters")
	}
}

func TestLintNoFileMatched(t *testing.T) {
	h := &langHandler{
		logger:  log.New(log.Writer(), "", log.LstdFlags),
		configs: map[string][]Language{},
		files: map[string]*File{
			"file:///foo": &File{},
		},
	}

	_, err := h.lint("file:///bar")
	if err == nil {
		t.Fatal("Should be an error if no linters")
	}
}

func TestLintFileMatched(t *testing.T) {
	base := "/base"
	file := "/base/foo"
	uri := "file:///base/foo"
	if runtime.GOOS == "windows" {
		base = "C:/base"
		file = "C:/base/foo"
		uri = "file:///C:/base/foo"
	}

	h := &langHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		rootPath: base,
		configs: map[string][]Language{
			"vim": {
				{
					LintCommand:        `echo ` + file + `:2:No it is normal!`,
					LintIgnoreExitCode: true,
					LintStdin:          true,
				},
			},
		},
		files: map[string]*File{
			uri: &File{
				LanguageID: "vim",
				Text:       "scriptencoding utf-8\nabnormal!\n",
			},
		},
	}

	d, err := h.lint(uri)
	if err != nil {
		t.Fatal(err)
	}
	if len(d) != 1 {
		t.Fatal("diagnostics should be only one")
	}
	if d[0].Range.Start.Line != 1 {
		t.Fatalf("range.start.line should be %v but got: %v", 1, d[0].Range.Start.Line)
	}
	if d[0].Range.Start.Character != 0 {
		t.Fatalf("range.start.character should be %v but got: %v", 0, d[0].Range.Start.Character)
	}
	if d[0].Severity != 1 {
		t.Fatalf("severity should be %v but got: %v", 0, d[0].Severity)
	}
	if strings.TrimSpace(d[0].Message) != "No it is normal!" {
		t.Fatalf("message should be %q but got: %q", "No it is normal!", strings.TrimSpace(d[0].Message))
	}
}
