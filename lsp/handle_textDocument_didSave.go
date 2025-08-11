package lsp

import (
	"context"
	"encoding/json"

	"github.com/konradmalik/efm-langserver/types"
	"github.com/sourcegraph/jsonrpc2"
)

func (h *LspHandler) HandleTextDocumentDidSave(_ context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params types.DidSaveTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	notifier := NewNotifier(conn)
	if params.Text != nil {
		err = h.langHandler.OnUpdateFile(notifier, params.TextDocument.URI, *params.Text, nil, types.EventTypeSave)
	} else {
		err = h.langHandler.OnSaveFile(notifier, params.TextDocument.URI)
	}
	if err != nil {
		return nil, err
	}

	return nil, nil
}
