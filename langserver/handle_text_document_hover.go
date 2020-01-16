package langserver

import (
	"context"
	"encoding/json"

	"github.com/sourcegraph/jsonrpc2"
)

func (h *langHandler) handleTextDocumentHover(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params HoverParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	return h.hover(params.TextDocument.URI, &params)
}
