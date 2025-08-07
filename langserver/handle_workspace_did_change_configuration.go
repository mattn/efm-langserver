package langserver

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/sourcegraph/jsonrpc2"
)

func (h *langHandler) handleWorkspaceDidChangeConfiguration(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params DidChangeConfigurationParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	return h.didChangeConfiguration(&params.Settings)
}

func (h *langHandler) didChangeConfiguration(config *Config) (any, error) {
	if config.Languages != nil {
		h.configs = *config.Languages
	}
	if config.RootMarkers != nil {
		h.rootMarkers = *config.RootMarkers
	}
	if config.Commands != nil {
		h.commands = *config.Commands
	}
	if config.LogLevel > 0 {
		h.loglevel = config.LogLevel
	}
	if config.LintDebounce > 0 {
		h.lintDebounce = time.Duration(config.LintDebounce)
	}
	if config.FormatDebounce > 0 {
		h.formatDebounce = time.Duration(config.FormatDebounce)
	}

	if config.LogFile != "" {
		f, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o660)
		if err == nil {
			if h.logger != nil {
				if w, ok := h.logger.Writer().(*os.File); ok {
					w.Close()
				}
			}
			h.logger = log.New(f, "", log.LstdFlags)
		}
	}

	if config.LogLevel > 0 {
		h.loglevel = config.LogLevel
	}

	return nil, nil
}
