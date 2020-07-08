package langserver

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"

	"github.com/mattn/go-unicodeclass"
	"github.com/reviewdog/errorformat"
	"github.com/sourcegraph/jsonrpc2"
)

// Config is
type Config struct {
	Version   int                   `yaml:"version"`
	LogFile   string                `yaml:"logfile"`
	LogLevel  int                   `yaml:"loglevel"`
	Commands  []Command             `yaml:"commands"`
	Languages map[string][]Language `yaml:"languages"`

	// Toggle support for "go to definition" requests.
	ProvideDefinition bool `yaml:"provide-definition"`

	Filename string      `yaml:"-"`
	Logger   *log.Logger `yaml:"-"`
}

// Config1 is
type Config1 struct {
	Version   int                 `yaml:"version"`
	Logger    *log.Logger         `yaml:"-"`
	Commands  []Command           `yaml:"commands"`
	Languages map[string]Language `yaml:"languages"`
}

// Language is
type Language struct {
	LintFormats        []string `yaml:"lint-formats"`
	LintStdin          bool     `yaml:"lint-stdin"`
	LintOffset         int      `yaml:"lint-offset"`
	LintCommand        string   `yaml:"lint-command"`
	LintIgnoreExitCode bool     `yaml:"lint-ignore-exit-code"`
	FormatCommand      string   `yaml:"format-command"`
	FormatStdin        bool     `yaml:"format-stdin"`
	SymbolCommand      string   `yaml:"symbol-command"`
	SymbolStdin        bool     `yaml:"symbol-stdin"`
	SymbolFormats      []string `yaml:"symbol-formats"`
	CompletionCommand  string   `yaml:"completion-command"`
	CompletionStdin    bool     `yaml:"completion-stdin"`
	HoverCommand       string   `yaml:"hover-command"`
	HoverStdin         bool     `yaml:"hover-stdin"`
	HoverType          string   `yaml:"hover-type"`
	Env                []string `yaml:"env"`
}

// NewHandler create JSON-RPC handler for this language server.
func NewHandler(config *Config) jsonrpc2.Handler {
	if config.Logger == nil {
		config.Logger = log.New(os.Stderr, "", log.LstdFlags)
	}
	var handler = &langHandler{
		loglevel:          config.LogLevel,
		logger:            config.Logger,
		commands:          config.Commands,
		configs:           config.Languages,
		provideDefinition: config.ProvideDefinition,
		files:             make(map[DocumentURI]*File),
		request:           make(chan DocumentURI),
		conn:              nil,
		filename:          config.Filename,
	}
	go handler.linter()
	return jsonrpc2.HandlerWithError(handler.handle)
}

type langHandler struct {
	loglevel          int
	logger            *log.Logger
	commands          []Command
	configs           map[string][]Language
	provideDefinition bool
	files             map[DocumentURI]*File
	request           chan DocumentURI
	conn              *jsonrpc2.Conn
	rootPath          string
	filename          string
	folders           []string
}

// File is
type File struct {
	LanguageID string
	Text       string
	Version    int
}

func (f *File) WordAt(pos Position) string {
	lines := strings.Split(f.Text, "\n")
	if pos.Line < 0 || pos.Line > len(lines) {
		return ""
	}
	chars := utf16.Encode([]rune(lines[pos.Line]))
	if pos.Character < 0 || pos.Character > len(chars) {
		return ""
	}
	prevPos := 0
	currPos := -1
	prevCls := unicodeclass.Invalid
	for i, char := range chars {
		currCls := unicodeclass.Is(rune(char))
		if currCls != prevCls {
			if i <= pos.Character {
				prevPos = i
			} else {
				if char == '_' {
					continue
				}
				currPos = i
				break
			}
		}
		prevCls = currCls
	}
	if currPos < 0 {
		return ""
	}
	return string(utf16.Decode(chars[prevPos:currPos]))
}

func isWindowsDrivePath(path string) bool {
	if len(path) < 4 {
		return false
	}
	return unicode.IsLetter(rune(path[0])) && path[1] == ':'
}

func isWindowsDriveURI(uri string) bool {
	if len(uri) < 4 {
		return false
	}
	return uri[0] == '/' && unicode.IsLetter(rune(uri[1])) && uri[2] == ':'
}

func fromURI(uri DocumentURI) (string, error) {
	u, err := url.ParseRequestURI(string(uri))
	if err != nil {
		return "", err
	}
	if u.Scheme != "file" {
		return "", fmt.Errorf("only file URIs are supported, got %v", u.Scheme)
	}
	if isWindowsDriveURI(u.Path) {
		u.Path = u.Path[1:]
	}
	return u.Path, nil
}

func toURI(path string) DocumentURI {
	if isWindowsDrivePath(path) {
		path = "/" + path
	}
	return DocumentURI((&url.URL{
		Scheme: "file",
		Path:   filepath.ToSlash(path),
	}).String())
}

func (h *langHandler) logMessage(typ MessageType, message string) {
	h.conn.Notify(
		context.Background(),
		"window/logMessage",
		&LogMessageParams{
			Type:    typ,
			Message: message,
		})
}

func (h *langHandler) linter() {
	for {
		uri, ok := <-h.request
		if !ok {
			break
		}
		diagnostics, err := h.lint(uri)
		if err != nil {
			h.logger.Println(err)
			continue
		}
		h.conn.Notify(
			context.Background(),
			"textDocument/publishDiagnostics",
			&PublishDiagnosticsParams{
				URI:         uri,
				Diagnostics: diagnostics,
				Version:     h.files[uri].Version,
			})
	}
}

func (h *langHandler) findRootPath(fname string) string {
	for _, folder := range h.folders {
		if len(fname) > len(folder) && strings.EqualFold(fname[:len(folder)], folder) {
			return folder
		}
	}

	return h.rootPath
}

func isFilename(s string) bool {
	for _, f := range []string{
		"stdin",
		"-",
		"<text>",
		"<stdin>",
	} {
		if s == f {
			return true
		}
	}
	return false

}

func (h *langHandler) lint(uri DocumentURI) ([]Diagnostic, error) {
	f, ok := h.files[uri]
	if !ok {
		return nil, fmt.Errorf("document not found: %v", uri)
	}

	fname, err := fromURI(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid uri: %v: %v", err, uri)
	}
	fname = filepath.ToSlash(fname)
	if runtime.GOOS == "windows" {
		fname = strings.ToLower(fname)
	}

	var configs []Language
	if cfgs, ok := h.configs[f.LanguageID]; ok {
		for _, cfg := range cfgs {
			if cfg.LintCommand != "" {
				configs = append(configs, cfg)
			}
		}
	}
	if cfgs, ok := h.configs[wildcard]; ok {
		for _, cfg := range cfgs {
			if cfg.LintCommand != "" {
				configs = append(configs, cfg)
			}
		}
	}

	if len(configs) == 0 {
		h.logger.Printf("lint for LanguageID not supported: %v", f.LanguageID)
		return []Diagnostic{}, nil
	}

	diagnostics := []Diagnostic{}
	for _, config := range configs {
		if config.LintCommand == "" {
			continue
		}

		command := config.LintCommand
		if !config.LintStdin && !strings.Contains(command, "${INPUT}") {
			command = command + " ${INPUT}"
		}
		command = replaceCommandInputFilename(command, fname)

		formats := config.LintFormats
		if len(formats) == 0 {
			formats = []string{"%f:%l:%m", "%f:%l:%c:%m"}
		}

		efms, err := errorformat.NewErrorformat(formats)
		if err != nil {
			return nil, fmt.Errorf("invalid error-format: %v", config.LintFormats)
		}

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/c", command)
		} else {
			cmd = exec.Command("sh", "-c", command)
		}
		cmd.Dir = h.findRootPath(fname)
		cmd.Env = append(os.Environ(), config.Env...)
		if config.LintStdin {
			cmd.Stdin = strings.NewReader(f.Text)
		}
		b, err := cmd.CombinedOutput()
		// Most of lint tools exit with non-zero value. But some commands
		// return with zero value. We can not handle the output is real result
		// or output of usage. So efm-langserver ignore that command exiting
		// with zero-value. So if you want to handle the command which exit
		// with zero value, please specify lint-ignore-exit-code.
		if err == nil && !config.LintIgnoreExitCode {
			h.logMessage(LogError, "command exit with zero. probably you forgot to specify `lint-ignore-exit-code: true`.")
			continue
		}
		if h.loglevel >= 1 {
			h.logger.Println(command+":", string(b))
		}
		scanner := efms.NewScanner(bytes.NewReader(b))
		for scanner.Scan() {
			entry := scanner.Entry()
			if !entry.Valid {
				continue
			}
			if config.LintStdin && isFilename(entry.Filename) {
				entry.Filename = fname
				path, err := filepath.Abs(entry.Filename)
				if err != nil {
					continue
				}
				path = filepath.ToSlash(path)
				if runtime.GOOS == "windows" {
					path = strings.ToLower(path)
				}
				if path != fname {
					continue
				}
			} else {
				entry.Filename = filepath.ToSlash(entry.Filename)
			}
			word := ""
			if entry.Col == 0 {
				entry.Col = 1
			} else {
				word = f.WordAt(Position{Line: entry.Lnum - 1 - config.LintOffset, Character: entry.Col - 1})
			}
			severity := 1
			switch {
			case entry.Type == 'E' || entry.Type == 'e':
				severity = 1
			case entry.Type == 'W' || entry.Type == 'w':
				severity = 2
			case entry.Type == 'I' || entry.Type == 'i':
				severity = 3
			case entry.Type == 'H' || entry.Type == 'h':
				severity = 4
			}
			diagnostics = append(diagnostics, Diagnostic{
				Range: Range{
					Start: Position{Line: entry.Lnum - 1 - config.LintOffset, Character: entry.Col - 1},
					End:   Position{Line: entry.Lnum - 1 - config.LintOffset, Character: entry.Col - 1 + len(word)},
				},
				Code:     itoaPtrIfNotZero(entry.Nr),
				Message:  entry.Text,
				Severity: severity,
			})
		}
	}

	return diagnostics, nil
}

func itoaPtrIfNotZero(n int) *string {
	if n == 0 {
		return nil
	}
	s := strconv.Itoa(n)
	return &s
}

func (h *langHandler) closeFile(uri DocumentURI) error {
	delete(h.files, uri)
	return nil
}

func (h *langHandler) saveFile(uri DocumentURI) error {
	h.request <- uri
	return nil
}

func (h *langHandler) openFile(uri DocumentURI, languageID string, version int) error {
	f := &File{
		Text:       "",
		LanguageID: languageID,
		Version:    version,
	}
	h.files[uri] = f
	return nil
}

func (h *langHandler) updateFile(uri DocumentURI, text string, version *int) error {
	f, ok := h.files[uri]
	if !ok {
		return fmt.Errorf("document not found: %v", uri)
	}
	f.Text = text
	if version != nil {
		f.Version = *version
	}

	h.request <- uri
	return nil
}

func (h *langHandler) configFor(uri DocumentURI) []Language {
	f, ok := h.files[uri]
	if !ok {
		return []Language{}
	}
	c, ok := h.configs[f.LanguageID]
	if !ok {
		return []Language{}
	}
	return c
}

func (h *langHandler) addFolder(folder string) {
	folder = filepath.Clean(folder)
	found := false
	for _, cur := range h.folders {
		if cur == folder {
			found = true
			break
		}
	}
	if !found {
		h.folders = append(h.folders, folder)
	}
}

func (h *langHandler) handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	switch req.Method {
	case "initialize":
		return h.handleInitialize(ctx, conn, req)
	case "initialized":
		return
	case "shutdown":
		return h.handleShutdown(ctx, conn, req)
	case "textDocument/didOpen":
		return h.handleTextDocumentDidOpen(ctx, conn, req)
	case "textDocument/didChange":
		return h.handleTextDocumentDidChange(ctx, conn, req)
	case "textDocument/didSave":
		return h.handleTextDocumentDidSave(ctx, conn, req)
	case "textDocument/didClose":
		return h.handleTextDocumentDidClose(ctx, conn, req)
	case "textDocument/formatting":
		return h.handleTextDocumentFormatting(ctx, conn, req)
	case "textDocument/documentSymbol":
		return h.handleTextDocumentSymbol(ctx, conn, req)
	case "textDocument/completion":
		return h.handleTextDocumentCompletion(ctx, conn, req)
	case "textDocument/definition":
		return h.handleTextDocumentDefinition(ctx, conn, req)
	case "textDocument/hover":
		return h.handleTextDocumentHover(ctx, conn, req)
	case "textDocument/codeAction":
		return h.handleTextDocumentCodeAction(ctx, conn, req)
	case "workspace/executeCommand":
		return h.handleWorkspaceExecuteCommand(ctx, conn, req)
	case "workspace/didChangeConfiguration":
		return h.handleWorkspaceDidChangeConfiguration(ctx, conn, req)
	case "workspace/workspaceFolders":
		return h.handleWorkspaceWorkspaceFolders(ctx, conn, req)
	}

	return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound, Message: fmt.Sprintf("method not supported: %s", req.Method)}
}

func replaceCommandInputFilename(command string, fname string) string {
	command = strings.Replace(command, "${INPUT}", fname, -1)
	command = strings.Replace(command, "${FILENAME}", filepath.FromSlash(fname), -1)
	return command
}
