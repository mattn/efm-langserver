package langserver

const wildcard = "="

// DocumentURI is
type DocumentURI string

// InitializeParams is
type InitializeParams struct {
	ProcessID             int                `json:"processId,omitempty"`
	RootURI               DocumentURI        `json:"rootUri,omitempty"`
	InitializationOptions InitializeOptions  `json:"initializationOptions,omitempty"`
	Capabilities          ClientCapabilities `json:"capabilities,omitempty"`
	Trace                 string             `json:"trace,omitempty"`
}

// InitializeOptions is
type InitializeOptions struct {
	DocumentFormatting bool `json:"documentFormatting"`
	Hover              bool `json:"hover"`
	DocumentSymbol     bool `json:"documentSymbol"`
	CodeAction         bool `json:"codeAction"`
	Completion         bool `json:"completion"`
}

// ClientCapabilities is
type ClientCapabilities struct {
}

// InitializeResult is
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities,omitempty"`
}

// MessageType is
type MessageType int

// LogError is
const (
	LogError   MessageType = 1
	LogWarning             = 2
	LogInfo                = 3
	LogLog                 = 4
)

// TextDocumentSyncKind is
type TextDocumentSyncKind int

// TDSKNone is
const (
	TDSKNone        TextDocumentSyncKind = 0
	TDSKFull                             = 1
	TDSKIncremental                      = 2
)

// CompletionProvider is
type CompletionProvider struct {
	ResolveProvider   bool     `json:"resolveProvider,omitempty"`
	TriggerCharacters []string `json:"triggerCharacters"`
}

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
	DocumentSymbolProvider     bool                         `json:"documentSymbolProvider,omitempty"`
	CompletionProvider         *CompletionProvider          `json:"completionProvider,omitempty"`
	DefinitionProvider         bool                         `json:"definitionProvider,omitempty"`
	DocumentFormattingProvider bool                         `json:"documentFormattingProvider,omitempty"`
	HoverProvider              bool                         `json:"hoverProvider,omitempty"`
	CodeActionProvider         bool                         `json:"codeActionProvider,omitempty"`
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

// CompletionParams is
type CompletionParams struct {
	TextDocumentPositionParams
	CompletionContext CompletionContext `json:"contentChanges"`
}

// CompletionContext is
type CompletionContext struct {
	TriggerKind      int     `json:"triggerKind"`
	TriggerCharacter *string `json:"triggerCharacter"`
}

// HoverParams is
type HoverParams struct {
	TextDocumentPositionParams
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
type FormattingOptions map[string]interface{}

// DocumentFormattingParams is
type DocumentFormattingParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Options      FormattingOptions      `json:"options"`
}

// TextEdit is
type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

// DocumentSymbolParams is
type DocumentSymbolParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// SymbolInformation is
type SymbolInformation struct {
	Name          string   `json:"name"`
	Kind          int64    `json:"kind"`
	Deprecated    bool     `json:"deprecated"`
	Location      Location `json:"location"`
	ContainerName *string  `json:"containerName"`
}

// CompletionItemKind is
type CompletionItemKind int

// TextCompletion is
const (
	TextCompletion          CompletionItemKind = 1
	MethodCompletion        CompletionItemKind = 2
	FunctionCompletion      CompletionItemKind = 3
	ConstructorCompletion   CompletionItemKind = 4
	FieldCompletion         CompletionItemKind = 5
	VariableCompletion      CompletionItemKind = 6
	ClassCompletion         CompletionItemKind = 7
	InterfaceCompletion     CompletionItemKind = 8
	ModuleCompletion        CompletionItemKind = 9
	PropertyCompletion      CompletionItemKind = 10
	UnitCompletion          CompletionItemKind = 11
	ValueCompletion         CompletionItemKind = 12
	EnumCompletion          CompletionItemKind = 13
	KeywordCompletion       CompletionItemKind = 14
	SnippetCompletion       CompletionItemKind = 15
	ColorCompletion         CompletionItemKind = 16
	FileCompletion          CompletionItemKind = 17
	ReferenceCompletion     CompletionItemKind = 18
	FolderCompletion        CompletionItemKind = 19
	EnumMemberCompletion    CompletionItemKind = 20
	ConstantCompletion      CompletionItemKind = 21
	StructCompletion        CompletionItemKind = 22
	EventCompletion         CompletionItemKind = 23
	OperatorCompletion      CompletionItemKind = 24
	TypeParameterCompletion CompletionItemKind = 25
)

// CompletionItemTag is
type CompletionItemTag int

// InsertTextFormat is
type InsertTextFormat int

// PlainTextTextFormat is
const (
	PlainTextTextFormat InsertTextFormat = 1
	SnippetTextFormat   InsertTextFormat = 2
)

// Command is
type Command struct {
	Title     string        `json:"title" yaml:"title"`
	Command   string        `json:"command" yaml:"command"`
	Arguments []interface{} `json:"arguments,omitempty" yaml:"arguments,omitempty"`
	OS        string        `json:"-" yaml:"os,omitempty"`
}

// WorkspaceEdit is
type WorkspaceEdit struct {
	Changes         interface{} `json:"changes"`         // { [uri: DocumentUri]: TextEdit[]; };
	DocumentChanges interface{} `json:"documentChanges"` // (TextDocumentEdit[] | (TextDocumentEdit | CreateFile | RenameFile | DeleteFile)[]);
}

// CodeAction is
type CodeAction struct {
	Title       string         `json:"title"`
	Diagnostics []Diagnostic   `json:"diagnostics"`
	IsPreferred bool           `json:"isPreferred"` // TODO
	Edit        *WorkspaceEdit `json:"edit"`
	Command     *Command       `json:"command"`
}

// CompletionItem is
type CompletionItem struct {
	Label               string              `json:"label"`
	Kind                CompletionItemKind  `json:"kind,omitempty"`
	Tags                []CompletionItemTag `json:"tags,omitempty"`
	Detail              string              `json:"detail,omitempty"`
	Documentation       string              `json:"documentation,omitempty"` // string | MarkupContent
	Deprecated          bool                `json:"deprecated,omitempty"`
	Preselect           bool                `json:"preselect,omitempty"`
	SortText            string              `json:"sortText,omitempty"`
	FilterText          string              `json:"filterText,omitempty"`
	InsertText          string              `json:"insertText,omitempty"`
	InsertTextFormat    InsertTextFormat    `json:"insertTextFormat,omitempty"`
	TextEdit            *TextEdit           `json:"textEdit,omitempty"`
	AdditionalTextEdits []TextEdit          `json:"additionalTextEdits,omitempty"`
	CommitCharacters    []string            `json:"commitCharacters,omitempty"`
	Command             *Command            `json:"command,omitempty"`
	Data                interface{}         `json:"data,omitempty"`
}

// Hover is
type Hover struct {
	Contents interface{} `json:"contents"`
	Range    *Range      `json:"range"`
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
	Markdown             = "markdown"
)

// MarkupContent is
type MarkupContent struct {
	Kind  MarkupKind `json:"kind"`
	Value string     `json:"value"`
}

// WorkDoneProgressParams is
type WorkDoneProgressParams struct {
	WorkDoneToken interface{} `json:"workDoneToken"`
}

// ExecuteCommandParams is
type ExecuteCommandParams struct {
	WorkDoneProgressParams

	Command   string        `json:"command"`
	Arguments []interface{} `json:"arguments,omitempty"`
}

// CodeActionKind is
type CodeActionKind string

// Empty is
const (
	Empty                 CodeActionKind = ""
	QuickFix                             = "quickfix"
	Refactor                             = "refactor"
	RefactorExtract                      = "refactor.extract"
	RefactorInline                       = "refactor.inline"
	RefactorRewrite                      = "refactor.rewrite"
	Source                               = "source"
	SourceOrganizeImports                = "source.organizeImports"
)

// CodeActionContext is
type CodeActionContext struct {
	Diagnostics []Diagnostic     `json:"diagnostics"`
	Only        []CodeActionKind `json:"only,omitempty"`
}

// PartialResultParams is
type PartialResultParams struct {
	PartialResultToken interface{} `json:"partialResultToken"`
}

// CodeActionParams is
type CodeActionParams struct {
	WorkDoneProgressParams
	PartialResultParams

	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
	Context      CodeActionContext      `json:"context"`
}

// DidChangeConfigurationParams is
type DidChangeConfigurationParams struct {
	Settings Config `json:"settings"`
}

// NotificationMessage is
type NotificationMessage struct {
	Method string      `json:"message"`
	Params interface{} `json:"params"`
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
