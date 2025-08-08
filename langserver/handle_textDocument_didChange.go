package langserver

import (
	"context"
	"encoding/json"

	"github.com/sourcegraph/jsonrpc2"
)

func (h *langHandler) handleTextDocumentDidChange(_ context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params DidChangeTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	for _, change := range params.ContentChanges {
		if err := h.onUpdateFile(conn, params.TextDocument.URI, change.Text, &params.TextDocument.Version, eventTypeChange); err != nil {
			return nil, err
		}
	}
	return nil, nil
}
