package langserver

import (
	"context"

	"github.com/sourcegraph/jsonrpc2"
)

func (h *langHandler) handleShutdown(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if h.lintTimer != nil {
		h.lintTimer.Stop()
	}

	close(h.request)
	return nil, conn.Close()
}
