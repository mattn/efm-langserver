package langserver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/reviewdog/errorformat"
	"github.com/sourcegraph/jsonrpc2"
)

func (h *langHandler) handleTextDocumentSymbol(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params DocumentSymbolParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	return h.symbol(params.TextDocument.URI)
}

var symbolKindMap = map[string]int{
	"file":          1,
	"module":        2,
	"namespace":     3,
	"package":       4,
	"class":         5,
	"method":        6,
	"property":      7,
	"field":         8,
	"constructor":   9,
	"enum":          10,
	"interface":     11,
	"function":      12,
	"variable":      13,
	"constant":      14,
	"string":        15,
	"number":        16,
	"boolean":       17,
	"array":         18,
	"object":        19,
	"key":           20,
	"null":          21,
	"enummember":    22,
	"struct":        23,
	"event":         24,
	"operator":      25,
	"typeparameter": 26,
}

func (h *langHandler) symbol(uri string) ([]SymbolInformation, error) {
	f, ok := h.files[uri]
	if !ok {
		return nil, fmt.Errorf("document not found: %v", uri)
	}

	fname, err := fromURI(uri)
	if err != nil {
		h.logger.Println("invalid uri")
		return nil, fmt.Errorf("invalid uri: %v: %v", err, uri)
	}
	fname = filepath.ToSlash(fname)
	if runtime.GOOS == "windows" {
		fname = strings.ToLower(fname)
	}

	var configs []Language
	if cfgs, ok := h.configs[f.LanguageID]; ok {
		configs = append(configs, cfgs...)
	}
	if cfgs, ok := h.configs[wildcard]; ok {
		configs = append(configs, cfgs...)
	}

	found := 0
	for _, config := range configs {
		if config.SymbolCommand != "" {
			found++
		}
	}
	if found == 0 {
		h.logger.Printf("symbol for LanguageID not supported: %v", f.LanguageID)
		return nil, nil
	}

	symbols := []SymbolInformation{}
	for _, config := range configs {
		if config.SymbolCommand == "" {
			config.SymbolCommand = "ctags -x --_xformat=%{input}:%n:1:%K!%N"
		}

		var command string

		if strings.Index(config.SymbolCommand, "${INPUT}") != -1 {
			command = strings.Replace(config.SymbolCommand, "${INPUT}", fname, -1)
		} else {
			command = config.SymbolCommand + " " + fname
		}

		efms, err := errorformat.NewErrorformat([]string{"%f:%l:%c:%m"})
		if err != nil {
			h.logger.Println("invalid error-format")
			return nil, fmt.Errorf("invalid error-format: %v", config.LintFormats)
		}

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/c", command)
		} else {
			cmd = exec.Command("sh", "-c", command)
		}
		cmd.Env = append(os.Environ(), config.Env...)

		b, err := cmd.CombinedOutput()
		if err != nil {
			h.logger.Printf("symbol command failed: %v", err)
			return nil, fmt.Errorf("symbol command failed: %v", err)
		}

		scanner := bufio.NewScanner(bytes.NewReader(b))
		for scanner.Scan() {
			for _, ef := range efms.Efms {
				h.logger.Println(scanner.Text())
				m := ef.Match(string(scanner.Text()))
				if m == nil {
					h.logger.Println("ignore1")
					continue
				}
				m.F = filepath.ToSlash(m.F)
				if m.C == 0 {
					m.C = 1
				}
				path, err := filepath.Abs(m.F)
				if err != nil {
					h.logger.Println(err)
					continue
				}
				path = filepath.ToSlash(path)
				if runtime.GOOS == "windows" {
					path = strings.ToLower(path)
				}
				if path != fname {
					h.logger.Println(path, fname)
					continue
				}
				token := strings.SplitN(m.M, wildcard, 2)
				if len(token) != 2 {
					h.logger.Println("invalid token")
					continue
				}
				kind, ok := symbolKindMap[strings.ToLower(token[0])]
				if !ok {
					kind = symbolKindMap["file"]
				}
				symbols = append(symbols, SymbolInformation{
					Location: Location{
						URI: uri,
						Range: Range{
							Start: Position{Line: m.L - 1 - config.LintOffset, Character: m.C - 1},
							End:   Position{Line: m.L - 1 - config.LintOffset, Character: m.C - 1},
						},
					},
					Kind: int64(kind),
					Name: token[1],
				})
			}
		}
	}

	return symbols, nil
}
