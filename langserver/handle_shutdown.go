package langserver

import (
	"context"

	"github.com/sourcegraph/jsonrpc2"
)

func (h *langHandler) handleShutdown(_ context.Context, conn *jsonrpc2.Conn, _ *jsonrpc2.Request) (result any, err error) {
	h.shutdown()
	return nil, conn.Close()
}

func (h *langHandler) shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.isShutdown {
		return
	}
	h.isShutdown = true
	if h.lintTimer != nil {
		h.lintTimer.Stop()
		h.lintTimer = nil
	}
	close(h.request)
}
