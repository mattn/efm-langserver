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

	"github.com/sourcegraph/jsonrpc2"
)

func (h *langHandler) handleTextDocumentCompletion(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params CompletionParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	return h.completion(params.TextDocument.URI, &params)
}

func (h *langHandler) completion(uri DocumentURI, params *CompletionParams) ([]CompletionItem, error) {
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
			if cfg.CompletionCommand != "" {
				configs = append(configs, cfg)
			}
		}
	}
	if cfgs, ok := h.configs[wildcard]; ok {
		for _, cfg := range cfgs {
			if cfg.CompletionCommand != "" {
				configs = append(configs, cfg)
			}
		}
	}

	if len(configs) == 0 {
		if h.loglevel >= 1 {
			h.logger.Printf("completion for LanguageID not supported: %v", f.LanguageID)
		}
		return nil, nil
	}

	for _, config := range configs {
		if config.CompletionCommand == "" {
			return nil, nil
		}

		command := config.CompletionCommand

		if strings.Contains(command, "${POSITION}") {
			command = strings.Replace(command, "${POSITION}", fmt.Sprintf("%d:%d", params.TextDocumentPositionParams.Position.Line, params.Position.Character), -1)
		}
		if !config.CompletionStdin && !strings.Contains(command, "${INPUT}") {
			command = command + " ${INPUT}"
		}
		command = replaceCommandInputFilename(command, fname, h.rootPath)

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/c", command)
		} else {
			cmd = exec.Command("sh", "-c", command)
		}
		cmd.Dir = h.findRootPath(fname, config)
		cmd.Env = append(os.Environ(), config.Env...)
		if config.CompletionStdin {
			cmd.Stdin = strings.NewReader(f.Text)
		}
		b, err := cmd.CombinedOutput()
		if err != nil {
			h.logger.Printf("completion command failed: %v", err)
			return nil, fmt.Errorf("completion command failed: %v: %v", err, string(b))
		}
		if h.loglevel >= 1 {
			h.logger.Println(command+":", string(b))
		}

		result := []CompletionItem{}
		scanner := bufio.NewScanner(bytes.NewReader(b))
		for scanner.Scan() {
			result = append(result, CompletionItem{
				Label:      scanner.Text(),
				InsertText: scanner.Text(),
			})
		}
		return result, nil
	}

	return nil, fmt.Errorf("completion for LanguageID not supported: %v", f.LanguageID)
}
