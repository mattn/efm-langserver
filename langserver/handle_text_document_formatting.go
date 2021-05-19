package langserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

	return h.formatting(params.TextDocument.URI, params.Options)
}

func (h *langHandler) formatting(uri DocumentURI, options FormattingOptions) ([]TextEdit, error) {
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
		for _, cfg := range cfgs {
			if cfg.FormatCommand != "" {
				configs = append(configs, cfg)
			}
		}
	}
	if cfgs, ok := h.configs[wildcard]; ok {
		for _, cfg := range cfgs {
			if cfg.FormatCommand != "" {
				configs = append(configs, cfg)
			}
		}
	}

	if len(configs) == 0 {
		if h.loglevel >= 1 {
			h.logger.Printf("format for LanguageID not supported: %v", f.LanguageID)
		}
		return nil, nil
	}

	originalText := f.Text
	text := originalText
	formated := false

Configs:
	for _, config := range configs {
		if config.FormatCommand == "" {
			continue
		}

		command := config.FormatCommand
		if !config.FormatStdin && !strings.Contains(command, "${INPUT}") {
			command = command + " ${INPUT}"
		}
		command = replaceCommandInputFilename(command, fname, h.rootPath)

		for placeholder, value := range options {
			re, err := regexp.Compile(fmt.Sprintf(`\${([^:|^}]+):%s}`, placeholder))
			if err != nil {
				h.logger.Println(command+":", err)
				continue Configs
			}

			switch v := value.(type) {
			default:
				command = re.ReplaceAllString(command, fmt.Sprintf("$1 %v", v))
			case bool:
				if v {
					command = re.ReplaceAllString(command, "$1")
				}
			}
		}
		re := regexp.MustCompile(`\${[^}]*}`)
		command = re.ReplaceAllString(command, "")

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/c", command)
		} else {
			cmd = exec.Command("sh", "-c", command)
		}
		cmd.Dir = h.findRootPath(fname, config)
		cmd.Env = append(os.Environ(), config.Env...)
		if config.FormatStdin {
			cmd.Stdin = strings.NewReader(text)
		}
		var buf bytes.Buffer
		cmd.Stderr = &buf
		b, err := cmd.Output()
		if err != nil {
			h.logger.Println(command+":", buf.String())
			continue
		}

		formated = true

		if h.loglevel >= 1 {
			h.logger.Println(command+":", string(b))
		}
		text = strings.Replace(string(b), "\r", "", -1)
	}

	if formated {
		h.logger.Println("format succeeded")
		return ComputeEdits(uri, originalText, text), nil
	}

	return nil, fmt.Errorf("format for LanguageID not supported: %v", f.LanguageID)
}
