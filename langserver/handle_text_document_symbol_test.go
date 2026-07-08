package langserver

import (
	"log"
	"os"
	"path/filepath"
	"testing"
)

func TestSymbolFormats(t *testing.T) {
	base, _ := os.Getwd()
	file := filepath.Join(base, "foo")
	uri := toURI(file)

	h := &langHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		rootPath: base,
		configs: map[string][]Language{
			"vim": {
				{
					SymbolCommand: `echo ` + file + `:2:1:function!MyFunc`,
					SymbolStdin:   true,
					SymbolFormats: []string{"%f:%l:%c:%m"},
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

	symbols, err := h.symbol(uri)
	if err != nil {
		t.Fatal(err)
	}
	if len(symbols) != 1 {
		t.Fatalf("symbols should be only one but got: %v", symbols)
	}
	if symbols[0].Name != "MyFunc" {
		t.Fatalf("name should be %q but got: %q", "MyFunc", symbols[0].Name)
	}
	if symbols[0].Kind != int64(symbolKindMap["function"]) {
		t.Fatalf("kind should be %v but got: %v", symbolKindMap["function"], symbols[0].Kind)
	}
	if symbols[0].Location.Range.Start.Line != 1 {
		t.Fatalf("range.start.line should be %v but got: %v", 1, symbols[0].Location.Range.Start.Line)
	}
}

func TestSymbolNoDuplicateWhenMultipleFormatsMatch(t *testing.T) {
	base, _ := os.Getwd()
	file := filepath.Join(base, "foo")
	uri := toURI(file)

	h := &langHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		rootPath: base,
		configs: map[string][]Language{
			"vim": {
				{
					SymbolCommand: `echo ` + file + `:2:1:function!MyFunc`,
					SymbolStdin:   true,
					SymbolFormats: []string{"%f:%l:%c:%m", "%f:%l:%m"},
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

	symbols, err := h.symbol(uri)
	if err != nil {
		t.Fatal(err)
	}
	if len(symbols) != 1 {
		t.Fatalf("symbols should be only one but got: %v", symbols)
	}
	if symbols[0].Name != "MyFunc" {
		t.Fatalf("name should be %q but got: %q", "MyFunc", symbols[0].Name)
	}
	if symbols[0].Kind != int64(symbolKindMap["function"]) {
		t.Fatalf("kind should be %v but got: %v", symbolKindMap["function"], symbols[0].Kind)
	}
}
