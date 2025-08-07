package langserver

const wildcard = "="

// DocumentURI is
type DocumentURI string

// InitializeParams is
type InitializeParams struct {
	ProcessID             int                `json:"processId,omitempty"`
	RootURI               DocumentURI        `json:"rootUri,omitempty"`
	InitializationOptions *InitializeOptions `json:"initializationOptions,omitempty"`
	Capabilities          ClientCapabilities `json:"capabilities,omitempty"`
	Trace                 string             `json:"trace,omitempty"`
}

// InitializeOptions is
type InitializeOptions struct {
	DocumentFormatting bool `json:"documentFormatting"`
	RangeFormatting    bool `json:"documentRangeFormatting"`
}

// ClientCapabilities is
type ClientCapabilities struct{}

// InitializeResult is
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities,omitempty"`
}

// MessageType is
type MessageType int

// LogError is
const (
	_ MessageType = iota
	LogError
	LogWarning
	LogInfo
	LogLog
)

// TextDocumentSyncKind is
type TextDocumentSyncKind int

// TDSKNone is
const (
	TDSKNone TextDocumentSyncKind = iota
	TDSKFull
	TDSKIncremental
)

// WorkspaceFoldersServerCapabilities is
type WorkspaceFoldersServerCapabilities struct {
	Supported           bool `json:"supported"`
	ChangeNotifications bool `json:"changeNotifications"`
}

// ServerCapabilitiesWorkspace is
type ServerCapabilitiesWorkspace struct {
	WorkspaceFolders WorkspaceFoldersServerCapabilities `json:"workspaceFolders"`
}

// ServerCapabilities is
type ServerCapabilities struct {
	TextDocumentSync           TextDocumentSyncKind         `json:"textDocumentSync,omitempty"`
	DefinitionProvider         bool                         `json:"definitionProvider,omitempty"`
	DocumentFormattingProvider bool                         `json:"documentFormattingProvider,omitempty"`
	RangeFormattingProvider    bool                         `json:"documentRangeFormattingProvider,omitempty"`
	Workspace                  *ServerCapabilitiesWorkspace `json:"workspace,omitempty"`
}

// TextDocumentItem is
type TextDocumentItem struct {
	URI        DocumentURI `json:"uri"`
	LanguageID string      `json:"languageId"`
	Version    int         `json:"version"`
	Text       string      `json:"text"`
}

// VersionedTextDocumentIdentifier is
type VersionedTextDocumentIdentifier struct {
	TextDocumentIdentifier
	Version int `json:"version"`
}

// TextDocumentIdentifier is
type TextDocumentIdentifier struct {
	URI DocumentURI `json:"uri"`
}

// DidOpenTextDocumentParams is
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// DidCloseTextDocumentParams is
type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// TextDocumentContentChangeEvent is
type TextDocumentContentChangeEvent struct {
	Range       Range  `json:"range"`
	RangeLength int    `json:"rangeLength"`
	Text        string `json:"text"`
}

// DidChangeTextDocumentParams is
type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

// DidSaveTextDocumentParams is
type DidSaveTextDocumentParams struct {
	Text         *string                `json:"text"`
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// TextDocumentPositionParams is
type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// Location is
type Location struct {
	URI   DocumentURI `json:"uri"`
	Range Range       `json:"range"`
}

// Range is
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Position is
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// DiagnosticRelatedInformation is
type DiagnosticRelatedInformation struct {
	Location Location `json:"location"`
	Message  string   `json:"message"`
}

// Diagnostic is
type Diagnostic struct {
	Range              Range                          `json:"range"`
	Severity           int                            `json:"severity,omitempty"`
	Code               *string                        `json:"code,omitempty"`
	Source             *string                        `json:"source,omitempty"`
	Message            string                         `json:"message"`
	RelatedInformation []DiagnosticRelatedInformation `json:"relatedInformation,omitempty"`
}

// PublishDiagnosticsParams is
type PublishDiagnosticsParams struct {
	URI         DocumentURI  `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	Version     int          `json:"version"`
}

// FormattingOptions is
type FormattingOptions map[string]any

// DocumentFormattingParams is
type DocumentFormattingParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Options      FormattingOptions      `json:"options"`
}

// DocumentRangeFormattingParams is
type DocumentRangeFormattingParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
	Options      FormattingOptions      `json:"options"`
}

// TextEdit is
type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

// InsertTextFormat is
type InsertTextFormat int

// PlainTextTextFormat is
const (
	PlainTextTextFormat InsertTextFormat = 1
	SnippetTextFormat   InsertTextFormat = 2
)

// Command is
type Command struct {
	Title     string `json:"title" yaml:"title"`
	Command   string `json:"command" yaml:"command"`
	Arguments []any  `json:"arguments,omitempty" yaml:"arguments,omitempty"`
	OS        string `json:"-" yaml:"os,omitempty"`
}

// WorkspaceEdit is
type WorkspaceEdit struct {
	Changes         any `json:"changes"`         // { [uri: DocumentUri]: TextEdit[]; };
	DocumentChanges any `json:"documentChanges"` // (TextDocumentEdit[] | (TextDocumentEdit | CreateFile | RenameFile | DeleteFile)[]);
}

// MarkedString is
type MarkedString struct {
	Language string `json:"language"`
	Value    string `json:"value"`
}

// MarkupKind is
type MarkupKind string

// PlainText is
const (
	PlainText MarkupKind = "plaintext"
	Markdown  MarkupKind = "markdown"
)

// MarkupContent is
type MarkupContent struct {
	Kind  MarkupKind `json:"kind"`
	Value string     `json:"value"`
}

// WorkDoneProgressParams is
type WorkDoneProgressParams struct {
	WorkDoneToken any `json:"workDoneToken"`
}

// ExecuteCommandParams is
type ExecuteCommandParams struct {
	WorkDoneProgressParams

	Command   string `json:"command"`
	Arguments []any  `json:"arguments,omitempty"`
}

// PartialResultParams is
type PartialResultParams struct {
	PartialResultToken any `json:"partialResultToken"`
}

// DidChangeConfigurationParams is
type DidChangeConfigurationParams struct {
	Settings Config `json:"settings"`
}

// NotificationMessage is
type NotificationMessage struct {
	Method string `json:"message"`
	Params any    `json:"params"`
}

// DocumentDefinitionParams is
type DocumentDefinitionParams struct {
	TextDocumentPositionParams
	WorkDoneProgressParams
	PartialResultParams
}

// ShowMessageParams is
type ShowMessageParams struct {
	Type    MessageType `json:"type"`
	Message string      `json:"message"`
}

// LogMessageParams is
type LogMessageParams struct {
	Type    MessageType `json:"type"`
	Message string      `json:"message"`
}

// DidChangeWorkspaceFoldersParams is
type DidChangeWorkspaceFoldersParams struct {
	Event WorkspaceFoldersChangeEvent `json:"event"`
}

// WorkspaceFoldersChangeEvent is
type WorkspaceFoldersChangeEvent struct {
	Added   []WorkspaceFolder `json:"added,omitempty"`
	Removed []WorkspaceFolder `json:"removed,omitempty"`
}

// WorkspaceFolder is
type WorkspaceFolder struct {
	URI  DocumentURI `json:"uri"`
	Name string      `json:"name"`
}
