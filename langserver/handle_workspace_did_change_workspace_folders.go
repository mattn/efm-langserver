package langserver

import (
	"context"
	"encoding/json"

	"github.com/sourcegraph/jsonrpc2"
)

func (h *langHandler) handleDidChangeWorkspaceWorkspaceFolders(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params DidChangeWorkspaceFoldersParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	return h.didChangeWorkspaceFolders(&params)
}

func (h *langHandler) didChangeWorkspaceFolders(params *DidChangeWorkspaceFoldersParams) (result interface{}, err error) {
	var folders []string
	for _, removed := range params.Event.Removed {
		for _, folder := range h.folders {
			if toURI(folder) != removed.URI {
				folders = append(folders, folder)
			}
		}
	}
	for _, added := range params.Event.Added {
		found := false
		for _, folder := range h.folders {
			if toURI(folder) == added.URI {
				found = true
				break
			}
		}
		if !found {
			if folder, err := fromURI(added.URI); err == nil {
				folders = append(folders, folder)
			}
		}
	}
	h.folders = folders
	return nil, nil
}
