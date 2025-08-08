package langserver

import (
	"context"
	"encoding/json"

	"github.com/sourcegraph/jsonrpc2"
)

func (h *langHandler) handleTextDocumentDidSave(_ context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params DidSaveTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	if params.Text != nil {
		err = h.onUpdateFile(conn, params.TextDocument.URI, *params.Text, nil, eventTypeSave)
	} else {
		err = h.onSaveFile(conn, params.TextDocument.URI)
	}
	if err != nil {
		return nil, err
	}

	return nil, nil
}
