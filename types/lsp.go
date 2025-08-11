package types

// DocumentURI is
type DocumentURI string

// InitializeParams is
type InitializeParams struct {
	RootURI               DocumentURI        `json:"rootUri,omitempty"`
	InitializationOptions *InitializeOptions `json:"initializationOptions,omitempty"`
	Capabilities          ClientCapabilities `json:"capabilities"`
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
	Capabilities ServerCapabilities `json:"capabilities"`
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

// ServerCapabilities is
type ServerCapabilities struct {
	TextDocumentSync           TextDocumentSyncKind `json:"textDocumentSync,omitempty"`
	DocumentFormattingProvider bool                 `json:"documentFormattingProvider,omitempty"`
	RangeFormattingProvider    bool                 `json:"documentRangeFormattingProvider,omitempty"`
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

// DiagnosticSeverity is
type DiagnosticSeverity int

const (
	Error DiagnosticSeverity = iota + 1
	Warning
	Information
	Hint
)

// Diagnostic is
type Diagnostic struct {
	Range              Range                          `json:"range"`
	Severity           DiagnosticSeverity             `json:"severity,omitempty"`
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

// DidChangeConfigurationParams is
type DidChangeConfigurationParams struct {
	Settings Config `json:"settings"`
}

// LogMessageParams is
type LogMessageParams struct {
	Type    MessageType `json:"type"`
	Message string      `json:"message"`
}
