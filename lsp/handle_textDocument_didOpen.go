package lsp

import (
	"context"
	"encoding/json"

	"github.com/konradmalik/efm-langserver/types"
	"github.com/sourcegraph/jsonrpc2"
)

func (h *LspHandler) HandleTextDocumentDidOpen(_ context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params types.DidOpenTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	notifier := NewNotifier(conn)
	if err := h.langHandler.OnOpenFile(notifier, params.TextDocument.URI, params.TextDocument.LanguageID, params.TextDocument.Version, params.TextDocument.Text); err != nil {
		return nil, err
	}

	return nil, nil
}
