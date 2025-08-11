package lsp

import (
	"context"

	"github.com/sourcegraph/jsonrpc2"
)

func (h *LspHandler) HandleShutdown(_ context.Context, conn *jsonrpc2.Conn, _ *jsonrpc2.Request) (result any, err error) {
	h.langHandler.Close()
	return nil, conn.Close()
}
