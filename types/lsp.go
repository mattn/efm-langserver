package types

type DocumentURI string

type InitializeParams struct {
	RootURI               DocumentURI        `json:"rootUri,omitempty"`
	InitializationOptions *InitializeOptions `json:"initializationOptions,omitempty"`
	Capabilities          ClientCapabilities `json:"capabilities"`
}

type InitializeOptions struct {
	DocumentFormatting bool `json:"documentFormatting"`
	RangeFormatting    bool `json:"documentRangeFormatting"`
}

type ClientCapabilities struct{}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
}

type MessageType int

const (
	_ MessageType = iota
	LogError
	LogWarning
	LogInfo
	LogLog
)

type TextDocumentSyncKind int

const (
	TDSKNone TextDocumentSyncKind = iota
	TDSKFull
	TDSKIncremental
)

type ServerCapabilities struct {
	TextDocumentSync           TextDocumentSyncKind `json:"textDocumentSync,omitempty"`
	DocumentFormattingProvider bool                 `json:"documentFormattingProvider,omitempty"`
	RangeFormattingProvider    bool                 `json:"documentRangeFormattingProvider,omitempty"`
}

type TextDocumentItem struct {
	URI        DocumentURI `json:"uri"`
	LanguageID string      `json:"languageId"`
	Version    int         `json:"version"`
	Text       string      `json:"text"`
}

type VersionedTextDocumentIdentifier struct {
	TextDocumentIdentifier
	Version int `json:"version"`
}

type TextDocumentIdentifier struct {
	URI DocumentURI `json:"uri"`
}

type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type TextDocumentContentChangeEvent struct {
	Range       Range  `json:"range"`
	RangeLength int    `json:"rangeLength"`
	Text        string `json:"text"`
}

type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

type DidSaveTextDocumentParams struct {
	Text         *string                `json:"text"`
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type Location struct {
	URI   DocumentURI `json:"uri"`
	Range Range       `json:"range"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type DiagnosticRelatedInformation struct {
	Location Location `json:"location"`
	Message  string   `json:"message"`
}

type DiagnosticSeverity int

const (
	Error DiagnosticSeverity = iota + 1
	Warning
	Information
	Hint
)

type Diagnostic struct {
	Range              Range                          `json:"range"`
	Severity           DiagnosticSeverity             `json:"severity,omitempty"`
	Code               *string                        `json:"code,omitempty"`
	Source             *string                        `json:"source,omitempty"`
	Message            string                         `json:"message"`
	RelatedInformation []DiagnosticRelatedInformation `json:"relatedInformation,omitempty"`
}

type PublishDiagnosticsParams struct {
	URI         DocumentURI  `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	Version     int          `json:"version"`
}

type FormattingOptions map[string]any

type DocumentFormattingParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Options      FormattingOptions      `json:"options"`
}

type DocumentRangeFormattingParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
	Options      FormattingOptions      `json:"options"`
}

type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

type DidChangeConfigurationParams struct {
	Settings Config `json:"settings"`
}

type LogMessageParams struct {
	Type    MessageType `json:"type"`
	Message string      `json:"message"`
}
