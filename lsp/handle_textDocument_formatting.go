package lsp

import (
	"context"
	"encoding/json"

	"github.com/konradmalik/efm-langserver/types"
	"github.com/sourcegraph/jsonrpc2"
)

func (h *LspHandler) HandleTextDocumentFormatting(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params types.DocumentFormattingParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	return h.langHandler.Formatting(params.TextDocument.URI, nil, params.Options)
}

func (h *LspHandler) HandleTextDocumentRangeFormatting(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params types.DocumentRangeFormattingParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	return h.langHandler.Formatting(params.TextDocument.URI, &params.Range, params.Options)
}
