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

func (h *langHandler) completion(uri string, params *CompletionParams) ([]CompletionItem, error) {
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

	configs, ok := h.configs[f.LanguageId]
	if !ok {
		configs, ok = h.configs["_"]
		if !ok || len(configs) < 1 {
			return nil, fmt.Errorf("completion for languageId not supported: %v", f.LanguageId)
		}
	}
	found := 0
	for _, config := range configs {
		if config.CompletionCommand != "" {
			found++
		}
	}
	if found == 0 {
		return nil, fmt.Errorf("completion for languageId not supported: %v", f.LanguageId)
	}

	for _, config := range configs {
		if config.CompletionCommand == "" {
			return nil, nil
		}

		command := config.CompletionCommand

		if strings.Index(command, "${POSITION}") != -1 {
			command = strings.Replace(command, "${POSITION}", fmt.Sprintf("%d:%d", params.TextDocumentPositionParams.Position.Line, params.Position.Character), -1)
		}
		if strings.Index(command, "${INPUT}") != -1 {
			command = strings.Replace(command, "${INPUT}", fname, -1)
		} else {
			command = command + " " + "" + " " + fname
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
			h.logger.Printf("completion command failed: %v", err)
			return nil, fmt.Errorf("completion command failed: %v", err)
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

	return nil, fmt.Errorf("completion for languageId not supported: %v", f.LanguageId)
}
