package langserver

import (
	"context"
	"encoding/json"

	"github.com/sourcegraph/jsonrpc2"
)

func (h *langHandler) handleInitialize(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	h.conn = conn

	var params InitializeParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	h.rootPath = params.RootPath

	return InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync:           TDSKFull,
			DocumentFormattingProvider: true,
			DocumentSymbolProvider:     true,
		},
	}, nil
}
