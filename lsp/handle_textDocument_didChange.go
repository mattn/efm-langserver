package lsp

import (
	"context"
	"encoding/json"

	"github.com/konradmalik/efm-langserver/types"
	"github.com/sourcegraph/jsonrpc2"
)

func (h *LspHandler) HandleTextDocumentDidChange(_ context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params types.DidChangeTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	notifier := NewNotifier(conn)
	for _, change := range params.ContentChanges {
		if err := h.langHandler.OnUpdateFile(notifier, params.TextDocument.URI, change.Text, &params.TextDocument.Version, types.EventTypeChange); err != nil {
			return nil, err
		}
	}
	return nil, nil
}
