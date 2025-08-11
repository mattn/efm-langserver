package lsp

import (
	"context"
	"fmt"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/konradmalik/efm-langserver/core"
)

type LspHandler struct {
	langHandler *core.LangHandler
}

func NewHandler(langHandler *core.LangHandler) *LspHandler {
	return &LspHandler{langHandler: langHandler}
}

func (h *LspHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
	switch req.Method {
	case "initialize":
		return h.HandleInitialize(ctx, conn, req)
	case "initialized":
		return
	case "shutdown":
		return h.HandleShutdown(ctx, conn, req)
	case "textDocument/didOpen":
		return h.HandleTextDocumentDidOpen(ctx, conn, req)
	case "textDocument/didChange":
		return h.HandleTextDocumentDidChange(ctx, conn, req)
	case "textDocument/didSave":
		return h.HandleTextDocumentDidSave(ctx, conn, req)
	case "textDocument/didClose":
		return h.HandleTextDocumentDidClose(ctx, conn, req)
	case "textDocument/formatting":
		return h.HandleTextDocumentFormatting(ctx, conn, req)
	case "textDocument/rangeFormatting":
		return h.HandleTextDocumentRangeFormatting(ctx, conn, req)
	case "workspace/didChangeConfiguration":
		return h.HandleWorkspaceDidChangeConfiguration(ctx, conn, req)
	}

	return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound, Message: fmt.Sprintf("method not supported: %s", req.Method)}
}
