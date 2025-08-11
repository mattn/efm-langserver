package lsp

import (
	"context"
	"encoding/json"

	"github.com/konradmalik/efm-langserver/types"
	"github.com/sourcegraph/jsonrpc2"
)

func (h *LspHandler) HandleTextDocumentDidClose(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params types.DidCloseTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	if err := h.langHandler.OnCloseFile(params.TextDocument.URI); err != nil {
		return nil, err
	}
	return nil, nil
}
