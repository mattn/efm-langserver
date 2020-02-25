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

	"github.com/sourcegraph/jsonrpc2"
)

func (h *langHandler) handleTextDocumentFormatting(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params DocumentFormattingParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	return h.formatting(params.TextDocument.URI)
}

func (h *langHandler) formatting(uri string) ([]TextEdit, error) {
	f, ok := h.files[uri]
	if !ok {
		return nil, fmt.Errorf("document not found: %v", uri)
	}

	fname, err := fromURI(uri)
	if err != nil {
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
		if config.FormatCommand != "" {
			found++
		}
	}
	if found == 0 {
		h.logger.Printf("format for LanguageID not supported: %v", f.LanguageID)
		return nil, nil
	}

	for _, config := range configs {
		if config.FormatCommand == "" {
			continue
		}

		command := config.FormatCommand
		if !config.FormatStdin && strings.Index(command, "${INPUT}") == -1 {
			command = command + " ${INPUT}"
		}
		command = strings.Replace(command, "${INPUT}", fname, -1)

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/c", command)
		} else {
			cmd = exec.Command("sh", "-c", command)
		}
		cmd.Env = append(os.Environ(), config.Env...)
		if config.FormatStdin {
			cmd.Stdin = strings.NewReader(f.Text)
		}
		b, err := cmd.CombinedOutput()
		if err != nil {
			continue
		}
		h.logger.Println("format succeeded")
		text := strings.Replace(string(b), "\r", "", -1)
		flines := strings.Split(f.Text, "\n")
		return []TextEdit{
			{
				Range: Range{
					Start: Position{Line: 0, Character: 0},
					End:   Position{Line: len(flines), Character: len(flines[len(flines)-1])},
				},
				NewText: text,
			},
		}, nil
	}

	return nil, fmt.Errorf("format for LanguageID not supported: %v", f.LanguageID)
}
