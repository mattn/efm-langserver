package langserver

import (
	"context"
	"encoding/json"

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
	if config.LogLevel > 0 {
		h.loglevel = config.LogLevel
	}
	if config.LintDebounce > 0 {
		h.lintDebounce = config.LintDebounce
	}
	if config.FormatDebounce > 0 {
		h.formatDebounce = config.FormatDebounce
	}
	if config.LogLevel > 0 {
		h.loglevel = config.LogLevel
	}

	return nil, nil
}
