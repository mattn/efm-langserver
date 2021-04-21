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

func (h *langHandler) symbol(uri DocumentURI) ([]SymbolInformation, error) {
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
		for _, cfg := range cfgs {
			if cfg.SymbolCommand != "" {
				configs = append(configs, cfg)
			}
		}
	}
	if cfgs, ok := h.configs[wildcard]; ok {
		for _, cfg := range cfgs {
			if cfg.SymbolCommand != "" {
				configs = append(configs, cfg)
			}
		}
	}

	if len(configs) == 0 {
		configs = []Language{
			{
				SymbolCommand: "ctags -x --_xformat=%{input}:%n:1:%K!%N",
				SymbolFormats: []string{"%f:%l:%c:%m"},
			},
		}
	}

	symbols := []SymbolInformation{}
	for _, config := range configs {
		command := config.SymbolCommand
		if !config.SymbolStdin && !strings.Contains(command, "${INPUT}") {
			command = command + " ${INPUT}"
		}
		command = replaceCommandInputFilename(command, fname, h.rootPath)

		formats := config.LintFormats
		if len(formats) == 0 {
			formats = []string{"%f:%l:%m", "%f:%l:%c:%m"}
		}

		efms, err := errorformat.NewErrorformat(formats)
		if err != nil {
			h.logger.Println("invalid error-format")
			return nil, fmt.Errorf("invalid error-format: %v", config.SymbolFormats)
		}

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/c", command)
		} else {
			cmd = exec.Command("sh", "-c", command)
		}
		cmd.Dir = h.findRootPath(fname, config)
		cmd.Env = append(os.Environ(), config.Env...)
		if config.SymbolStdin {
			cmd.Stdin = strings.NewReader(f.Text)
		}
		b, err := cmd.CombinedOutput()
		if err != nil {
			continue
		}
		if h.loglevel >= 1 {
			h.logger.Println(command+":", string(b))
		}

		scanner := bufio.NewScanner(bytes.NewReader(b))
		for scanner.Scan() {
			for _, ef := range efms.Efms {
				m := ef.Match(string(scanner.Text()))
				if m == nil {
					continue
				}
				if config.SymbolStdin && (m.F == "stdin" || m.F == "-" || m.F == "<text>") {
					m.F = fname
				} else {
					m.F = filepath.ToSlash(m.F)
				}
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
				token := strings.SplitN(m.M, "!", 2)
				kind := symbolKindMap["key"]
				if len(token) == 2 {
					if tmp, ok := symbolKindMap[strings.ToLower(token[0])]; ok {
						kind = tmp
					}
				} else {
					token = []string{"", m.M}
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
