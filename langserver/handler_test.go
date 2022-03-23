package langserver

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLintNoLinter(t *testing.T) {
	h := &langHandler{
		logger:  log.New(log.Writer(), "", log.LstdFlags),
		configs: map[string][]Language{},
		files: map[DocumentURI]*File{
			DocumentURI("file:///foo"): {},
		},
	}

	_, err := h.lint(context.Background(), "file:///foo")
	if err != nil {
		t.Fatal("Should not be an error if no linters")
	}
}

func TestLintNoFileMatched(t *testing.T) {
	h := &langHandler{
		logger:  log.New(log.Writer(), "", log.LstdFlags),
		configs: map[string][]Language{},
		files: map[DocumentURI]*File{
			DocumentURI("file:///foo"): {},
		},
	}

	_, err := h.lint(context.Background(), "file:///bar")
	if err == nil {
		t.Fatal("Should be an error if no linters")
	}
}

func TestLintFileMatched(t *testing.T) {
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

	uriToDiag, err := h.lint(context.Background(), uri)
	d := uriToDiag[uri]
	if err != nil {
		t.Fatal(err)
	}
	if len(d) != 1 {
		t.Fatal("diagnostics should be only one", d)
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

func TestLintFileMatchedForce(t *testing.T) {
	base, _ := os.Getwd()
	file := filepath.Join(base, "foo")
	uri := toURI(file)

	h := &langHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		rootPath: base,
		configs: map[string][]Language{
			wildcard: {
				{
					LintCommand:        `echo ` + file + `:2:No it is normal!`,
					LintIgnoreExitCode: true,
					LintStdin:          true,
				},
			},
		},
		files: map[DocumentURI]*File{
			uri: &File{
				LanguageID: "vim",
				Text:       "scriptencoding utf-8\nabnormal!\n",
			},
		},
	}

	uriToDiag, err := h.lint(context.Background(), uri)
	d := uriToDiag[uri]
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

// column 0 remains unchanged, regardles of the configured offset
// column 0 indicates a whole line (although for 0-based column linters we can not distinguish between word starting at 0 and the whole line)
func TestLintOffsetColumnsZero(t *testing.T) {
	base, _ := os.Getwd()
	file := filepath.Join(base, "foo")
	uri := toURI(file)

	h := &langHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		rootPath: base,
		configs: map[string][]Language{
			wildcard: {
				{
					LintCommand:        `echo ` + file + `:2:0:msg`,
					LintFormats:        []string{"%f:%l:%c:%m"},
					LintIgnoreExitCode: true,
					LintStdin:          true,
					LintOffsetColumns:  1,
				},
			},
		},
		files: map[DocumentURI]*File{
			uri: &File{
				LanguageID: "vim",
				Text:       "scriptencoding utf-8\nabnormal!\n",
			},
		},
	}

	uriToDiag, err := h.lint(context.Background(), uri)
	d := uriToDiag[uri]
	if err != nil {
		t.Fatal(err)
	}
	if len(d) != 1 {
		t.Fatal("diagnostics should be only one")
	}
	if d[0].Range.Start.Character != 0 {
		t.Fatalf("range.start.character should be %v but got: %v", 0, d[0].Range.Start.Character)
	}
}

// without column offset, 1-based columns are assumed, which means that we should get 0 for column 1
// as LSP assumes 0-based columns
func TestLintOffsetColumnsNoOffset(t *testing.T) {
	base, _ := os.Getwd()
	file := filepath.Join(base, "foo")
	uri := toURI(file)

	h := &langHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		rootPath: base,
		configs: map[string][]Language{
			wildcard: {
				{
					LintCommand:        `echo ` + file + `:2:1:msg`,
					LintFormats:        []string{"%f:%l:%c:%m"},
					LintIgnoreExitCode: true,
					LintStdin:          true,
				},
			},
		},
		files: map[DocumentURI]*File{
			uri: &File{
				LanguageID: "vim",
				Text:       "scriptencoding utf-8\nabnormal!\n",
			},
		},
	}

	uriToDiag, err := h.lint(context.Background(), uri)
	d := uriToDiag[uri]
	if err != nil {
		t.Fatal(err)
	}
	if len(d) != 1 {
		t.Fatal("diagnostics should be only one")
	}
	if d[0].Range.Start.Character != 0 {
		t.Fatalf("range.start.character should be %v but got: %v", 0, d[0].Range.Start.Character)
	}
}

// for column 1 with offset we should get column 1 back
// without the offset efm would subtract 1 as it expects 1 based columns
func TestLintOffsetColumnsNonZero(t *testing.T) {
	base, _ := os.Getwd()
	file := filepath.Join(base, "foo")
	uri := toURI(file)

	h := &langHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		rootPath: base,
		configs: map[string][]Language{
			wildcard: {
				{
					LintCommand:        `echo ` + file + `:2:1:msg`,
					LintFormats:        []string{"%f:%l:%c:%m"},
					LintIgnoreExitCode: true,
					LintStdin:          true,
					LintOffsetColumns:  1,
				},
			},
		},
		files: map[DocumentURI]*File{
			uri: &File{
				LanguageID: "vim",
				Text:       "scriptencoding utf-8\nabnormal!\n",
			},
		},
	}

	uriToDiag, err := h.lint(context.Background(), uri)
	d := uriToDiag[uri]
	if err != nil {
		t.Fatal(err)
	}
	if len(d) != 1 {
		t.Fatal("diagnostics should be only one")
	}
	if d[0].Range.Start.Character != 1 {
		t.Fatalf("range.start.character should be %v but got: %v", 1, d[0].Range.Start.Character)
	}
}

func TestLintCategoryMap(t *testing.T) {
	base, _ := os.Getwd()
	file := filepath.Join(base, "foo")
	uri := toURI(file)

	mapping := make(map[string]string)
	mapping["R"] = "I" // pylint refactoring to info

	formats := []string{"%f:%l:%c:%t:%m"}

	h := &langHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		rootPath: base,
		configs: map[string][]Language{
			wildcard: {
				{
					LintCommand:        `echo ` + file + `:2:1:R:No it is normal!`,
					LintIgnoreExitCode: true,
					LintStdin:          true,
					LintFormats:        formats,
					LintCategoryMap:    mapping,
				},
			},
		},
		files: map[DocumentURI]*File{
			uri: &File{
				LanguageID: "vim",
				Text:       "scriptencoding utf-8\nabnormal!\n",
			},
		},
	}

	uriToDiag, err := h.lint(context.Background(), uri)
	d := uriToDiag[uri]
	if err != nil {
		t.Fatal(err)
	}
	if len(d) != 1 {
		t.Fatal("diagnostics should be only one")
	}
	if d[0].Severity != 3 {
		t.Fatalf("Severity should be %v but is: %v", 3, d[0].Severity)
	}
}

// Test if lint is executed if required root markers for the language are missing
func TestLintRequireRootMarker(t *testing.T) {
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
			uri: &File{
				LanguageID: "vim",
				Text:       "scriptencoding utf-8\nabnormal!\n",
			},
		},
	}

	d, err := h.lint(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	if len(d) != 0 {
		t.Fatal("diagnostics should be zero as we have no root marker for the language but require one", d)
	}
}

// Test if lint can return diagnostics for multiple files
func TestLintMultipleFiles(t *testing.T) {
	base, _ := os.Getwd()
	file := filepath.Join(base, "foo")
	file2 := filepath.Join(base, "bar")
	uri := toURI(file)
	uri2 := toURI(file2)

	h := &langHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		rootPath: base,
		configs: map[string][]Language{
			"vim": {
				{
					LintCommand:        `echo ` + file + `:2:1:First file! && echo ` + file2 + `:1:2:Second file!`,
					LintFormats:        []string{"%f:%l:%c:%m"},
					LintIgnoreExitCode: true,
					LintWorkspace:      true,
				},
			},
		},
		files: map[DocumentURI]*File{
			uri: {
				LanguageID: "vim",
				Text:       "scriptencoding utf-8\nabnormal!\n",
			},
			uri2: {
				LanguageID: "vim",
				Text:       "scriptencoding utf-8\nabnormal!\n",
			},
		},
		lastPublishedURIs: make(map[string]map[DocumentURI]struct{}),
	}

	d, err := h.lint(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	if len(d) != 2 {
		t.Fatalf("diagnostics should be two, but got %#v", d)
	}
	if d[uri][0].Range.Start.Character != 0 {
		t.Fatalf("first range.start.character should be %v but got: %v", 0, d[uri][0].Range.Start.Character)
	}
	if d[uri][0].Range.Start.Line != 1 {
		t.Fatalf("first range.start.line should be %v but got: %v", 1, d[uri][0].Range.Start.Line)
	}
	if d[uri2][0].Range.Start.Character != 1 {
		t.Fatalf("second range.start.character should be %v but got: %v", 1, d[uri2][0].Range.Start.Character)
	}
	if d[uri2][0].Range.Start.Line != 0 {
		t.Fatalf("second range.start.line should be %v but got: %v", 0, d[uri2][0].Range.Start.Line)
	}

	h.configs["vim"][0].LintCommand = `echo ` + file + `:2:1:First file only!`
	d, err = h.lint(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	if len(d) != 2 {
		t.Fatalf("diagnostics should be two, but got %#v", d)
	}
	if d[uri][0].Range.Start.Character != 0 {
		t.Fatalf("first range.start.character should be %v but got: %v", 0, d[uri][0].Range.Start.Character)
	}
	if d[uri][0].Range.Start.Line != 1 {
		t.Fatalf("first range.start.line should be %v but got: %v", 1, d[uri][0].Range.Start.Line)
	}
	if len(d[uri2]) != 0 {
		t.Fatalf("second diagnostics should be empty but got: %v", d[uri2])
	}
}

// Test if lint can return diagnostics for multiple files even when cancelled
func TestLintMultipleFilesWithCancel(t *testing.T) {
	base, _ := os.Getwd()
	file := filepath.Join(base, "foo")
	file2 := filepath.Join(base, "bar")
	uri := toURI(file)
	uri2 := toURI(file2)

	h := &langHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		rootPath: base,
		configs: map[string][]Language{
			"vim": {
				{
					LintCommand:        `echo ` + file + `:2:1:First file! && echo ` + file2 + `:1:2:Second file!`,
					LintFormats:        []string{"%f:%l:%c:%m"},
					LintIgnoreExitCode: true,
					LintWorkspace:      true,
				},
			},
		},
		files: map[DocumentURI]*File{
			uri: {
				LanguageID: "vim",
				Text:       "scriptencoding utf-8\nabnormal!\n",
			},
			uri2: {
				LanguageID: "vim",
				Text:       "scriptencoding utf-8\nabnormal!\n",
			},
		},
		lastPublishedURIs: make(map[string]map[DocumentURI]struct{}),
	}

	d, err := h.lint(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	if len(d) != 2 {
		t.Fatalf("diagnostics should be two, but got %#v", d)
	}
	if d[uri][0].Range.Start.Character != 0 {
		t.Fatalf("first range.start.character should be %v but got: %v", 0, d[uri][0].Range.Start.Character)
	}
	if d[uri][0].Range.Start.Line != 1 {
		t.Fatalf("first range.start.line should be %v but got: %v", 1, d[uri][0].Range.Start.Line)
	}
	if d[uri2][0].Range.Start.Character != 1 {
		t.Fatalf("second range.start.character should be %v but got: %v", 1, d[uri2][0].Range.Start.Character)
	}
	if d[uri2][0].Range.Start.Line != 0 {
		t.Fatalf("second range.start.line should be %v but got: %v", 0, d[uri2][0].Range.Start.Line)
	}

	startedFlagPath := "already-started"
	defer os.Remove(startedFlagPath)
	// Emulate heavy job
	h.configs["vim"][0].LintCommand = `touch ` + startedFlagPath + ` && sleep 1000000 && echo ` + file + `:2:1:First file only!`
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		h.lint(ctx, uri)
	}()
	for true {
		if _, err := os.Stat(startedFlagPath); errors.Is(err, os.ErrNotExist) {
			time.Sleep(50 * time.Microsecond)
			continue
		}
		break
	}
	cancel()
	h.configs["vim"][0].LintCommand = `echo ` + file + `:2:1:First file only!`
	d, err = h.lint(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	if len(d) != 2 {
		t.Fatalf("diagnostics should be two, but got %#v", d)
	}
	if d[uri][0].Range.Start.Character != 0 {
		t.Fatalf("first range.start.character should be %v but got: %v", 0, d[uri][0].Range.Start.Character)
	}
	if d[uri][0].Range.Start.Line != 1 {
		t.Fatalf("first range.start.line should be %v but got: %v", 1, d[uri][0].Range.Start.Line)
	}
	if len(d[uri2]) != 0 {
		t.Fatalf("second diagnostics should be empty but got: %v", d[uri2])
	}
}

func TestLintNoDiagnostics(t *testing.T) {
	base, _ := os.Getwd()
	file := filepath.Join(base, "foo")
	uri := toURI(file)

	h := &langHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		rootPath: base,
		configs: map[string][]Language{
			"vim": {
				{
					LintCommand:        "echo ",
					LintIgnoreExitCode: true,
					LintStdin:          true,
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

	uriToDiag, err := h.lint(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	d, ok := uriToDiag[uri]
	if !ok {
		t.Fatal("didn't get any diagnostics")
	}
	if len(d) != 0 {
		t.Fatal("diagnostics should be an empty list", d)
	}
}

func TestHover(t *testing.T) {
	base, _ := os.Getwd()
	file := filepath.Join(base, "foo")
	uri := toURI(file)

	text := "test_test-test.test"
	for scenario, config := range map[string]struct {
		text       string
		position   Position
		hoverChars string
		expected   string
	}{
		"_ is default": {
			text,
			Position{
				Line:      0,
				Character: 0,
			},
			"_",
			"test_test",
		},
		"with -": {
			text,
			Position{
				Line:      0,
				Character: 0,
			},
			"_-",
			"test_test-test",
		},
		"with - and .": {
			text,
			Position{
				Line:      0,
				Character: 0,
			},
			"_-.",
			"test_test-test.test",
		},
		"inner position": {
			text,
			Position{
				Line:      0,
				Character: 4,
			},
			"_-.",
			"test_test-test.test",
		},
	} {
		h := &langHandler{
			logger:   log.New(log.Writer(), "", log.LstdFlags),
			rootPath: base,
			configs: map[string][]Language{
				"vim": {
					{
						HoverCommand: "echo ${INPUT}",
						HoverChars:   config.hoverChars,
					},
				},
			},
			files: map[DocumentURI]*File{
				uri: {
					LanguageID: "vim",
					Text:       config.text,
				},
			},
		}
		t.Run(scenario, func(t *testing.T) {
			hover, err := h.hover(uri, &HoverParams{
				TextDocumentPositionParams{
					TextDocument: TextDocumentIdentifier{uri},
					Position:     config.position,
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			content := hover.Contents.(MarkupContent).Value
			if content != config.expected {
				t.Fatal("invalid hover contents:", content+",", "exptected:", config.expected)
			}
		})
	}
}
