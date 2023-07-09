package langserver

import (
	"context"
	"path/filepath"

	"github.com/sourcegraph/jsonrpc2"
)

func (h *langHandler) handleWorkspaceWorkspaceFolders(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	return h.workspaceFolders()
}

func (h *langHandler) workspaceFolders() (result any, err error) {
	workspaces := []WorkspaceFolder{}
	for _, workspace := range h.folders {
		workspaces = append(workspaces, WorkspaceFolder{
			URI:  toURI(workspace),
			Name: filepath.Base(workspace),
		})
	}
	return workspaces, nil
}
