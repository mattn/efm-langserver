package langserver

import (
	"context"
	"path/filepath"

	"github.com/sourcegraph/jsonrpc2"
)

func (h *langHandler) handleWorkspaceWorkspaceFolders(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	return h.workspaceFolders()
}

func (h *langHandler) workspaceFolders() (result interface{}, err error) {
	workspaces := []WorkspaceFolder{}
	for _, workspace := range h.folders {
		workspaces = append(workspaces, WorkspaceFolder{
			URI:  toURI(workspace),
			Name: filepath.Base(workspace),
		})
	}
	return workspaces, nil
}
