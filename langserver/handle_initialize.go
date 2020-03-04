package langserver

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"

	"github.com/sourcegraph/jsonrpc2"
)

func (h *langHandler) handleInitialize(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	h.conn = conn

	var params InitializeParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	rootPath, err := fromURI(params.RootURI)
	if err != nil {
		return nil, err
	}
	h.rootPath = filepath.Clean(rootPath)
	h.addFolder(rootPath)

	var completion *CompletionProvider
	var hasHoverCommand bool
	var hasCodeActionCommand bool
	var hasSymbolCommand bool
	var hasFormatCommand bool
	var hasDefinitionCommand bool

	if len(h.commands) > 0 {
		hasCodeActionCommand = true
	}
	if _, err = exec.LookPath("ctags"); err == nil {
		hasDefinitionCommand = true
	}
	for _, config := range h.configs {
		for _, v := range config {
			if v.CompletionCommand != "" {
				completion = &CompletionProvider{
					TriggerCharacters: []string{"*"},
				}
			}
			if v.HoverCommand != "" {
				hasHoverCommand = true
			}
			if v.SymbolCommand != "" {
				hasSymbolCommand = true
			}
			if v.FormatCommand != "" {
				hasFormatCommand = true
			}
		}
	}

	return InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync:           TDSKFull,
			DocumentFormattingProvider: hasFormatCommand,
			DocumentSymbolProvider:     hasSymbolCommand,
			DefinitionProvider:         hasDefinitionCommand,
			CompletionProvider:         completion,
			HoverProvider:              hasHoverCommand,
			CodeActionProvider:         hasCodeActionCommand,
			Workspace: &ServerCapabilitiesWorkspace{
				WorkspaceFolders: WorkspaceFoldersServerCapabilities{
					Supported:           true,
					ChangeNotifications: true,
				},
			},
		},
	}, nil
}
