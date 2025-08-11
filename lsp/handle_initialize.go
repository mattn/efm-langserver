package lsp

import (
	"context"
	"encoding/json"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/konradmalik/efm-langserver/types"
)

func (h *LspHandler) HandleInitialize(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (result types.InitializeResult, err error) {
	if req.Params == nil {
		return types.InitializeResult{}, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params types.InitializeParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return types.InitializeResult{}, err
	}

	return h.langHandler.Initialize(params)
}
