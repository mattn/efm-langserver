package langserver

import (
	"context"
	"encoding/json"

	"github.com/sourcegraph/jsonrpc2"
)

func (h *langHandler) handleTextDocumentDidOpen(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params DidOpenTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	if err := h.openFile(params.TextDocument.URI, params.TextDocument.LanguageID, params.TextDocument.Version); err != nil {
		return nil, err
	}
	if err := h.updateFile(params.TextDocument.URI, params.TextDocument.Text, &params.TextDocument.Version, eventTypeOpen); err != nil {
		return nil, err
	}
	return nil, nil
}
