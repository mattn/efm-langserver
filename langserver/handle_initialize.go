package langserver

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"

	"github.com/sourcegraph/jsonrpc2"
)

func (h *langHandler) handleInitialize(_ context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	h.conn = conn

	var params InitializeParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	// https://microsoft.github.io/language-server-protocol/specification#initialize
	// The rootUri of the workspace. Is null if no folder is open.
	if params.RootURI != "" {
		rootPath, err := fromURI(params.RootURI)
		if err != nil {
			return nil, err
		}
		h.rootPath = filepath.Clean(rootPath)
		h.addFolder(rootPath)
	}

	var completion *CompletionProvider
	var hasCompletionCommand bool
	var hasHoverCommand bool
	var hasCodeActionCommand bool
	var hasSymbolCommand bool
	var hasFormatCommand bool
	var hasRangeFormatCommand bool
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
				if v.FormatCanRange {
					hasRangeFormatCommand = true
				}
			}
		}
	}

	if params.InitializationOptions != nil {
		hasCompletionCommand = params.InitializationOptions.Completion
		hasHoverCommand = params.InitializationOptions.Hover
		hasCodeActionCommand = params.InitializationOptions.CodeAction
		hasSymbolCommand = params.InitializationOptions.DocumentSymbol
		hasFormatCommand = params.InitializationOptions.DocumentFormatting
		hasRangeFormatCommand = params.InitializationOptions.RangeFormatting
	}

	if hasCompletionCommand {
		chars := []string{"."}
		if len(h.triggerChars) > 0 {
			chars = h.triggerChars
		}
		completion = &CompletionProvider{
			TriggerCharacters: chars,
		}
	}

	return InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync:           TDSKFull,
			DocumentFormattingProvider: hasFormatCommand,
			RangeFormattingProvider:    hasRangeFormatCommand,
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
