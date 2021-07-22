package langserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf16"

	"github.com/mattn/go-unicodeclass"
	"github.com/sourcegraph/jsonrpc2"
)

func (h *langHandler) handleTextDocumentHover(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params HoverParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	return h.hover(params.TextDocument.URI, &params)
}

func (h *langHandler) hover(uri DocumentURI, params *HoverParams) (*Hover, error) {
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

	lines := strings.Split(f.Text, "\n")
	if params.Position.Line < 0 || params.Position.Line > len(lines) {
		return nil, fmt.Errorf("invalid position: %v", params.Position)
	}
	chars := utf16.Encode([]rune(lines[params.Position.Line]))
	if params.Position.Character < 0 || params.Position.Character > len(chars) {
		return nil, fmt.Errorf("invalid position: %v", params.Position)
	}
	prevPos := 0
	currPos := -1
	prevCls := unicodeclass.Invalid
	for i, char := range chars {
		currCls := unicodeclass.Is(rune(char))
		if currCls != prevCls {
			if i <= params.Position.Character {
				prevPos = i
			} else {
				if char == '_' {
					continue
				}
				currPos = i
				break
			}
		}
		prevCls = currCls
	}
	if currPos == -1 {
		currPos = len(chars)
	}
	word := string(utf16.Decode(chars[prevPos:currPos]))

	var configs []Language
	if cfgs, ok := h.configs[f.LanguageID]; ok {
		for _, cfg := range cfgs {
			if cfg.HoverCommand != "" {
				configs = append(configs, cfg)
			}
		}
	}
	if cfgs, ok := h.configs[wildcard]; ok {
		for _, cfg := range cfgs {
			if cfg.HoverCommand != "" {
				configs = append(configs, cfg)
			}
		}
	}

	if len(configs) == 0 {
		if h.loglevel >= 1 {
			h.logger.Printf("hover for LanguageID not supported: %v", f.LanguageID)
		}
		return nil, nil
	}

	for _, config := range configs {
		if config.HoverCommand == "" {
			continue
		}

		command := config.HoverCommand
		if !config.HoverStdin && !strings.Contains(command, "${INPUT}") {
			command = command + " ${INPUT}"
		}
		command = strings.Replace(command, "${INPUT}", word, -1)

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/c", command)
		} else {
			cmd = exec.Command("sh", "-c", command)
		}
		cmd.Dir = h.findRootPath(fname, config)
		cmd.Env = append(os.Environ(), config.Env...)
		if config.HoverStdin {
			cmd.Stdin = strings.NewReader(word)
		}
		b, err := cmd.CombinedOutput()
		if err != nil {
			return nil, err
		}
		if h.loglevel >= 1 {
			h.logger.Println(command+":", string(b))
		}

		var content MarkupContent
		if config.HoverType == "markdown" {
			content.Kind = Markdown
		} else {
			content.Kind = PlainText
		}
		content.Value = strings.TrimSpace(string(b))

		return &Hover{
			Contents: content,
			Range: &Range{
				Start: Position{
					Line:      params.Position.Line,
					Character: prevPos,
				},
				End: Position{
					Line:      params.Position.Line,
					Character: currPos,
				},
			},
		}, nil
	}

	return nil, nil
}
