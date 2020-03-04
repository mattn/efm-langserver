package langserver

import (
	"context"

	"github.com/sourcegraph/jsonrpc2"
)

func (h *langHandler) handleWorkspaceDidChangeConfiguration(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	return nil, nil
}
