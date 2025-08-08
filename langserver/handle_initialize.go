package langserver

import (
	"context"
	"encoding/json"
	"path/filepath"

	"github.com/sourcegraph/jsonrpc2"
)

func (h *langHandler) handleInitialize(_ context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	h.conn = conn

	var params InitializeParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	// https://microsoft.github.io/language-server-protocol/specification#initialize
	// The rootUri of the workspace. Is null if no folder is open.
	if params.RootURI != "" {
		rootPath, err := fromURI(params.RootURI)
		if err != nil {
			return nil, err
		}
		h.rootPath = filepath.Clean(rootPath)
	}

	var hasFormatCommand bool
	var hasRangeFormatCommand bool

	if params.InitializationOptions != nil {
		hasFormatCommand = params.InitializationOptions.DocumentFormatting
		hasRangeFormatCommand = params.InitializationOptions.RangeFormatting
	}

	for _, config := range h.configs {
		for _, v := range config {
			if v.FormatCommand != "" {
				hasFormatCommand = true
				if v.FormatCanRange {
					hasRangeFormatCommand = true
				}
			}
		}
	}

	return InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync:           TDSKFull,
			DocumentFormattingProvider: hasFormatCommand,
			RangeFormattingProvider:    hasRangeFormatCommand,
		},
	}, nil
}
