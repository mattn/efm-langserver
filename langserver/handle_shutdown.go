package langserver

import (
	"context"

	"github.com/sourcegraph/jsonrpc2"
)

func (h *LangHandler) handleShutdown(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	close(h.request)
	return nil, nil
}
