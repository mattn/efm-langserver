package core

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/konradmalik/efm-langserver/types"
)

func TestLintNoLinter(t *testing.T) {
	h := &LangHandler{
		logger:  log.New(log.Writer(), "", log.LstdFlags),
		configs: map[string][]types.Language{},
		files: map[types.DocumentURI]*fileRef{
			types.DocumentURI("file:///foo"): {},
		},
	}

	_, err := h.lintDocument(context.Background(), nil, "file:///foo", types.EventTypeChange)
	if err != nil {
		t.Fatal("Should not be an error if no linters")
	}
}

func TestLintNoFileMatched(t *testing.T) {
	h := &LangHandler{
		logger:  log.New(log.Writer(), "", log.LstdFlags),
		configs: map[string][]types.Language{},
		files: map[types.DocumentURI]*fileRef{
			types.DocumentURI("file:///foo"): {},
		},
	}

	_, err := h.lintDocument(context.Background(), nil, "file:///bar", types.EventTypeChange)
	if err == nil {
		t.Fatal("Should be an error if no linters")
	}
}

func TestLintFileMatched(t *testing.T) {
	base, _ := os.Getwd()
	file := filepath.Join(base, "foo")
	uri := toURI(file)

	h := &LangHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		RootPath: base,
		configs: map[string][]types.Language{
			"vim": {
				{
					LintCommand:        `echo ` + file + `:2:No it is normal!`,
					LintIgnoreExitCode: true,
					LintStdin:          true,
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

	uriToDiag, err := h.lintDocument(context.Background(), nil, uri, types.EventTypeChange)
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

	h := &LangHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		RootPath: base,
		configs: map[string][]types.Language{
			types.Wildcard: {
				{
					LintCommand:        `echo ` + file + `:2:No it is normal!`,
					LintIgnoreExitCode: true,
					LintStdin:          true,
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

	uriToDiag, err := h.lintDocument(context.Background(), nil, uri, types.EventTypeChange)
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

	h := &LangHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		RootPath: base,
		configs: map[string][]types.Language{
			types.Wildcard: {
				{
					LintCommand:        `echo ` + file + `:2:0:msg`,
					LintFormats:        []string{"%f:%l:%c:%m"},
					LintIgnoreExitCode: true,
					LintStdin:          true,
					LintOffsetColumns:  1,
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

	uriToDiag, err := h.lintDocument(context.Background(), nil, uri, types.EventTypeChange)
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

	h := &LangHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		RootPath: base,
		configs: map[string][]types.Language{
			types.Wildcard: {
				{
					LintCommand:        `echo ` + file + `:2:1:msg`,
					LintFormats:        []string{"%f:%l:%c:%m"},
					LintIgnoreExitCode: true,
					LintStdin:          true,
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

	uriToDiag, err := h.lintDocument(context.Background(), nil, uri, types.EventTypeChange)
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

	h := &LangHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		RootPath: base,
		configs: map[string][]types.Language{
			types.Wildcard: {
				{
					LintCommand:        `echo ` + file + `:2:1:msg`,
					LintFormats:        []string{"%f:%l:%c:%m"},
					LintIgnoreExitCode: true,
					LintStdin:          true,
					LintOffsetColumns:  1,
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

	uriToDiag, err := h.lintDocument(context.Background(), nil, uri, types.EventTypeChange)
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

	h := &LangHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		RootPath: base,
		configs: map[string][]types.Language{
			types.Wildcard: {
				{
					LintCommand:        `echo ` + file + `:2:1:R:No it is normal!`,
					LintIgnoreExitCode: true,
					LintStdin:          true,
					LintFormats:        formats,
					LintCategoryMap:    mapping,
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

	uriToDiag, err := h.lintDocument(context.Background(), nil, uri, types.EventTypeChange)
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

	h := &LangHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		RootPath: base,
		configs: map[string][]types.Language{
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
		files: map[types.DocumentURI]*fileRef{
			uri: {
				LanguageID: "vim",
				Text:       "scriptencoding utf-8\nabnormal!\n",
			},
		},
	}

	d, err := h.lintDocument(context.Background(), nil, uri, types.EventTypeChange)
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

	h := &LangHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		RootPath: base,
		configs: map[string][]types.Language{
			"vim": {
				{
					LintCommand:        `echo ` + file + `:2:1:First file! && echo ` + file2 + `:1:2:Second file!`,
					LintFormats:        []string{"%f:%l:%c:%m"},
					LintIgnoreExitCode: true,
					LintWorkspace:      true,
				},
			},
		},
		files: map[types.DocumentURI]*fileRef{
			uri: {
				LanguageID: "vim",
				Text:       "scriptencoding utf-8\nabnormal!\n",
			},
			uri2: {
				LanguageID: "vim",
				Text:       "scriptencoding utf-8\nabnormal!\n",
			},
		},
		lastPublishedURIs: make(map[string]map[types.DocumentURI]struct{}),
	}

	d, err := h.lintDocument(context.Background(), nil, uri, types.EventTypeChange)
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
	d, err = h.lintDocument(context.Background(), nil, uri, types.EventTypeChange)
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

	h := &LangHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		RootPath: base,
		configs: map[string][]types.Language{
			"vim": {
				{
					LintCommand:        `echo ` + file + `:2:1:First file! && echo ` + file2 + `:1:2:Second file! && echo ` + file2 + `:Empty l and c!`,
					LintFormats:        []string{"%f:%l:%c:%m", "%f:%m"},
					LintIgnoreExitCode: true,
					LintWorkspace:      true,
				},
			},
		},
		files: map[types.DocumentURI]*fileRef{
			uri: {
				LanguageID: "vim",
				Text:       "scriptencoding utf-8\nabnormal!\n",
			},
			uri2: {
				LanguageID: "vim",
				Text:       "scriptencoding utf-8\nabnormal!\n",
			},
		},
		lastPublishedURIs: make(map[string]map[types.DocumentURI]struct{}),
	}

	d, err := h.lintDocument(context.Background(), nil, uri, types.EventTypeChange)
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
	if d[uri2][1].Range.Start.Character != 0 {
		t.Fatalf("second range.start.character should be %v but got: %v", 0, d[uri2][1].Range.Start.Character)
	}
	if d[uri2][1].Range.Start.Line != 0 {
		t.Fatalf("second range.start.line should be %v but got: %v", 0, d[uri2][1].Range.Start.Line)
	}

	startedFlagPath := "already-started"
	defer func() {
		_ = os.Remove(startedFlagPath)
	}()
	// Emulate heavy job
	h.configs["vim"][0].LintCommand = `touch ` + startedFlagPath + ` && sleep 1000000 && echo ` + file + `:2:1:First file only!`
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		_, _ = h.lintDocument(ctx, nil, uri, types.EventTypeChange)
	}()
	for {
		if _, err := os.Stat(startedFlagPath); errors.Is(err, os.ErrNotExist) {
			time.Sleep(50 * time.Microsecond)
			continue
		}
		break
	}
	cancel()
	h.configs["vim"][0].LintCommand = `echo ` + file + `:2:1:First file only!`
	d, err = h.lintDocument(context.Background(), nil, uri, types.EventTypeChange)
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

	h := &LangHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		RootPath: base,
		configs: map[string][]types.Language{
			"vim": {
				{
					LintCommand:        "echo ",
					LintIgnoreExitCode: true,
					LintStdin:          true,
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

	uriToDiag, err := h.lintDocument(context.Background(), nil, uri, types.EventTypeChange)
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

func TestLintOnSave(t *testing.T) {
	base, _ := os.Getwd()
	file := filepath.Join(base, "foo")
	uri := toURI(file)

	h := &LangHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		RootPath: base,
		configs: map[string][]types.Language{
			"vim": {
				{
					LintCommand:        `echo ` + file + `:2:No it is normal!`,
					LintIgnoreExitCode: true,
					LintStdin:          true,
					LintOnSave:         true,
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

	uriToDiag, err := h.lintDocument(context.Background(), nil, uri, types.EventTypeChange)
	if err != nil {
		t.Fatal(err)
	}
	d := uriToDiag[uri]
	if len(d) != 0 {
		t.Fatal("diagnostics should be empty", d)
	}

	uriToDiag, err = h.lintDocument(context.Background(), nil, uri, types.EventTypeSave)
	if err != nil {
		t.Fatal(err)
	}
	d = uriToDiag[uri]
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
