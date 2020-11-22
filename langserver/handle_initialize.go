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
	var hasCompletionCommand bool = params.InitializationOptions.Completion
	var hasHoverCommand bool = params.InitializationOptions.Hover
	var hasCodeActionCommand bool = params.InitializationOptions.CodeAction
	var hasSymbolCommand bool = params.InitializationOptions.DocumentSymbol
	var hasFormatCommand bool = params.InitializationOptions.DocumentFormatting
	var hasDefinitionCommand bool

	if len(h.commands) > 0 {
		hasCodeActionCommand = true
	}
	if h.provideDefinition {
		if _, err = exec.LookPath("ctags"); err == nil {
			hasDefinitionCommand = true
		}
	}
	for _, config := range h.configs {
		for _, v := range config {
			if v.CompletionCommand != "" {
				hasCompletionCommand = true
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

	if hasCompletionCommand {
		completion = &CompletionProvider{
			TriggerCharacters: []string{"*"},
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
