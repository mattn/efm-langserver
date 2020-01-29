package langserver

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"
	"unicode/utf16"

	"github.com/mattn/go-unicodeclass"
	"github.com/reviewdog/errorformat"
	"github.com/sourcegraph/jsonrpc2"
)

type Config struct {
	LogWriter io.Writer            `yaml:"-"`
	Languages map[string]*Language `yaml:"languages"`
}

type Language struct {
	LintFormats        []string `yaml:"lint-formats"`
	LintStdin          bool     `yaml:"lint-stdin"`
	LintOffset         int      `yaml:"lint-offset"`
	LintCommand        string   `yaml:"lint-command"`
	LintIgnoreExitCode bool     `yaml:"lint-ignore-exit-code"`
	FormatCommand      string   `yaml:"format-command"`
	SymbolCommand      string   `yaml:"symbol-command"`
	CompletionCommand  string   `yaml:"completion-command"`
	HoverCommand       string   `yaml:"hover-command"`
	HoverStdin         bool     `yaml:"hover-stdin"`
	Env                []string `yaml:"env"`
}

func NewHandler(config *Config) jsonrpc2.Handler {
	if config.LogWriter == nil {
		config.LogWriter = os.Stderr
	}
	var handler = &langHandler{
		logger:  log.New(config.LogWriter, "", log.LstdFlags),
		configs: config.Languages,
		files:   make(map[string]*File),
		request: make(chan string),
		conn:    nil,
	}
	go handler.linter()
	return jsonrpc2.HandlerWithError(handler.handle)
}

type langHandler struct {
	logger   *log.Logger
	configs  map[string]*Language
	files    map[string]*File
	request  chan string
	conn     *jsonrpc2.Conn
	rootPath string
}

type File struct {
	LanguageId string
	Text       string
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

func fromURI(uri string) (string, error) {
	u, err := url.ParseRequestURI(uri)
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

func toURI(path string) *url.URL {
	if isWindowsDrivePath(path) {
		path = "/" + path
	}
	return &url.URL{
		Scheme: "file",
		Path:   filepath.ToSlash(path),
	}
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
			})
	}
}

func (h *langHandler) lint(uri string) ([]Diagnostic, error) {
	f, ok := h.files[uri]
	if !ok {
		return nil, fmt.Errorf("document not found: %v", uri)
	}

	config, ok := h.configs[f.LanguageId]
	if !ok || config.LintCommand == "" {
		config, ok = h.configs["_"]
		if !ok {
			return nil, fmt.Errorf("lint for languageId not supported: %v", f.LanguageId)
		}
	}
	if config.LintCommand == "" {
		return nil, fmt.Errorf("lint for languageId not supported: %v", f.LanguageId)
	}

	fname, err := fromURI(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid uri: %v: %v", err, uri)
	}
	fname = filepath.ToSlash(fname)
	if runtime.GOOS == "windows" {
		fname = strings.ToLower(fname)
	}
	var command string

	if config.LintStdin {
		command = config.LintCommand
	} else {
		if strings.Index(config.LintCommand, "${INPUT}") != -1 {
			command = strings.Replace(config.LintCommand, "${INPUT}", fname, -1)
		} else {
			command = config.LintCommand + " " + fname
		}
	}

	formats := config.LintFormats
	if len(formats) == 0 {
		formats = []string{"%f:%l:%m", "%f:%l:%c:%m"}
	}

	efms, err := errorformat.NewErrorformat(formats)
	if err != nil {
		return nil, fmt.Errorf("invalid error-format: %v", config.LintFormats)
	}

	diagnostics := []Diagnostic{}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	cmd.Env = append(os.Environ(), config.Env...)
	if config.LintStdin {
		cmd.Stdin = strings.NewReader(f.Text)
	}
	b, err := cmd.CombinedOutput()
	if err == nil && !config.LintIgnoreExitCode {
		return diagnostics, nil
	}
	for _, line := range strings.Split(string(b), "\n") {
		for _, ef := range efms.Efms {
			m := ef.Match(string(line))
			if m == nil {
				continue
			}
			if config.LintStdin && (m.F == "stdin" || m.F == "-" || m.F == "<text>") {
				m.F = fname
			} else {
				m.F = filepath.ToSlash(m.F)
			}
			if m.C == 0 {
				m.C = 1
			}
			path, err := filepath.Abs(m.F)
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
			severity := 1
			switch {
			case m.T == 'E' || m.T == 'e':
				severity = 1
			case m.T == 'W' || m.T == 'w':
				severity = 2
			case m.T == 'I' || m.T == 'i':
				severity = 3
			case m.T == 'H' || m.T == 'h':
				severity = 4
			}
			diagnostics = append(diagnostics, Diagnostic{
				Range: Range{
					Start: Position{Line: m.L - 1 - config.LintOffset, Character: m.C - 1},
					End:   Position{Line: m.L - 1 - config.LintOffset, Character: m.C - 1},
				},
				Message:  m.M,
				Severity: severity,
			})
		}
	}

	return diagnostics, nil
}

func (h *langHandler) closeFile(uri string) error {
	delete(h.files, uri)
	return nil
}

func (h *langHandler) saveFile(uri string) error {
	h.request <- uri
	return nil
}

func (h *langHandler) openFile(uri string, languageId string) error {
	f := &File{
		Text:       "",
		LanguageId: languageId,
	}
	h.files[uri] = f
	return nil
}

func (h *langHandler) updateFile(uri string, text string) error {
	f, ok := h.files[uri]
	if !ok {
		return fmt.Errorf("document not found: %v", uri)
	}
	f.Text = text

	h.request <- uri
	return nil
}

func (h *langHandler) formatFile(uri string) ([]TextEdit, error) {
	f, ok := h.files[uri]
	if !ok {
		return nil, fmt.Errorf("document not found: %v", uri)
	}

	config, ok := h.configs[f.LanguageId]
	if !ok || config.FormatCommand == "" {
		config, ok = h.configs["_"]
		if !ok {
			return nil, fmt.Errorf("format for languageId not supported: %v", f.LanguageId)
		}
	}
	if config.FormatCommand == "" {
		return nil, fmt.Errorf("format for languageId not supported: %v", f.LanguageId)
	}

	fname, err := fromURI(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid uri: %v: %v", err, uri)
	}

	fname = filepath.ToSlash(fname)
	if runtime.GOOS == "windows" {
		fname = strings.ToLower(fname)
	}
	var command string

	if strings.Index(config.FormatCommand, "${INPUT}") != -1 {
		command = strings.Replace(config.FormatCommand, "${INPUT}", fname, -1)
	} else {
		command = config.FormatCommand + " " + fname
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	cmd.Env = append(os.Environ(), config.Env...)
	cmd.Stdin = strings.NewReader(f.Text)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.New(string(b))
	}
	h.logger.Println("format succeeded")
	text := strings.Replace(string(b), "\r", "", -1)
	flines := strings.Split(f.Text, "\n")

	return []TextEdit{
		{
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: len(flines), Character: len(flines[len(flines)-1])},
			},
			NewText: text,
		},
	}, nil
}

func (h *langHandler) symbol(uri string) ([]SymbolInformation, error) {
	f, ok := h.files[uri]
	if !ok {
		return nil, fmt.Errorf("document not found: %v", uri)
	}

	config, ok := h.configs[f.LanguageId]
	if !ok || config.SymbolCommand == "" {
		config, ok = h.configs["_"]
		if !ok {
			return nil, fmt.Errorf("symbol for languageId not supported: %v", f.LanguageId)
		}
	}
	if config.SymbolCommand == "" {
		config.SymbolCommand = "ctags -x --_xformat=%{input}:%n:1:%K!%N"
	}

	fname, err := fromURI(uri)
	if err != nil {
		h.logger.Println("invalid uri")
		return nil, fmt.Errorf("invalid uri: %v: %v", err, uri)
	}
	fname = filepath.ToSlash(fname)
	if runtime.GOOS == "windows" {
		fname = strings.ToLower(fname)
	}
	var command string

	if strings.Index(config.SymbolCommand, "${INPUT}") != -1 {
		command = strings.Replace(config.SymbolCommand, "${INPUT}", fname, -1)
	} else {
		command = config.SymbolCommand + " " + fname
	}

	efms, err := errorformat.NewErrorformat([]string{"%f:%l:%c:%m"})
	if err != nil {
		h.logger.Println("invalid error-format")
		return nil, fmt.Errorf("invalid error-format: %v", config.LintFormats)
	}

	symbols := []SymbolInformation{}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	cmd.Env = append(os.Environ(), config.Env...)

	b, err := cmd.CombinedOutput()
	if err != nil {
		h.logger.Printf("symbol command failed: %v", err)
		return nil, fmt.Errorf("symbol command failed: %v", err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		for _, ef := range efms.Efms {
			h.logger.Println(scanner.Text())
			m := ef.Match(string(scanner.Text()))
			if m == nil {
				h.logger.Println("ignore1")
				continue
			}
			m.F = filepath.ToSlash(m.F)
			if m.C == 0 {
				m.C = 1
			}
			path, err := filepath.Abs(m.F)
			if err != nil {
				h.logger.Println(err)
				continue
			}
			path = filepath.ToSlash(path)
			if runtime.GOOS == "windows" {
				path = strings.ToLower(path)
			}
			if path != fname {
				h.logger.Println(path, fname)
				continue
			}
			token := strings.SplitN(m.M, "!", 2)
			if len(token) != 2 {
				h.logger.Println("invalid token")
				continue
			}
			kind, ok := symbolKindMap[strings.ToLower(token[0])]
			if !ok {
				kind = symbolKindMap["file"]
			}
			symbols = append(symbols, SymbolInformation{
				Location: Location{
					URI: uri,
					Range: Range{
						Start: Position{Line: m.L - 1 - config.LintOffset, Character: m.C - 1},
						End:   Position{Line: m.L - 1 - config.LintOffset, Character: m.C - 1},
					},
				},
				Kind: int64(kind),
				Name: token[1],
			})
		}
	}

	return symbols, nil
}

func (h *langHandler) configFor(uri string) *Language {
	f, ok := h.files[uri]
	if !ok {
		return &Language{}
	}
	c, ok := h.configs[f.LanguageId]
	if !ok {
		return &Language{}
	}
	return c
}

func (h *langHandler) completion(uri string, params *CompletionParams) ([]CompletionItem, error) {
	f, ok := h.files[uri]
	if !ok {
		return nil, fmt.Errorf("document not found: %v", uri)
	}

	config, ok := h.configs[f.LanguageId]
	if !ok || config.CompletionCommand == "" {
		config, ok = h.configs["_"]
		if !ok {
			return nil, fmt.Errorf("completion for languageId not supported: %v", f.LanguageId)
		}
	}
	if config.CompletionCommand == "" {
		return nil, nil
	}

	fname, err := fromURI(uri)
	if err != nil {
		h.logger.Println("invalid uri")
		return nil, fmt.Errorf("invalid uri: %v: %v", err, uri)
	}
	fname = filepath.ToSlash(fname)
	if runtime.GOOS == "windows" {
		fname = strings.ToLower(fname)
	}
	command := config.CompletionCommand

	if strings.Index(command, "${POSITION}") != -1 {
		command = strings.Replace(command, "${POSITION}", fmt.Sprintf("%d:%d", params.TextDocumentPositionParams.Position.Line, params.Position.Character), -1)
	}
	if strings.Index(command, "${INPUT}") != -1 {
		command = strings.Replace(command, "${INPUT}", fname, -1)
	} else {
		command = command + " " + "" + " " + fname
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	cmd.Env = append(os.Environ(), config.Env...)

	b, err := cmd.CombinedOutput()

	if err != nil {
		h.logger.Printf("completion command failed: %v", err)
		return nil, fmt.Errorf("completion command failed: %v", err)
	}

	result := []CompletionItem{}
	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		result = append(result, CompletionItem{
			Label:      scanner.Text(),
			InsertText: scanner.Text(),
		})
	}
	return result, nil
}

func (h *langHandler) hover(uri string, params *HoverParams) (*Hover, error) {
	f, ok := h.files[uri]
	if !ok {
		return nil, fmt.Errorf("document not found: %v", uri)
	}

	config, ok := h.configs[f.LanguageId]
	if !ok || config.HoverCommand == "" {
		config, ok = h.configs["_"]
		if !ok {
			return nil, fmt.Errorf("hover for languageId not supported: %v", f.LanguageId)
		}
	}
	if config.HoverCommand == "" {
		return nil, fmt.Errorf("hover for languageId not supported: %v", f.LanguageId)
	}

	lines := strings.Split(f.Text, "\n")
	if params.Position.Line < 0 || params.Position.Line > len(lines) {
		return nil, fmt.Errorf("invalid position: %v", params.Position)
	}
	chars := utf16.Encode([]rune(lines[params.Position.Line]))
	if params.Position.Character < 0 || params.Position.Character > len(chars) {
		return nil, fmt.Errorf("invalid position: %v", params.Position)
	}
	prevPos := 0
	currPos := -1
	prevCls := unicodeclass.Invalid
	for i, char := range chars {
		currCls := unicodeclass.Is(rune(char))
		if currCls != prevCls {
			if i <= params.Position.Character {
				prevPos = i
			} else {
				currPos = i
				break
			}
		}
		prevCls = currCls
	}
	if currPos == -1 {
		currPos = len(chars)
	}
	word := string(utf16.Decode(chars[prevPos:currPos]))

	var command string

	if config.HoverStdin {
		command = config.HoverCommand
	} else {
		if strings.Index(config.HoverCommand, "${INPUT}") != -1 {
			command = strings.Replace(config.HoverCommand, "${INPUT}", word, -1)
		} else {
			command = config.HoverCommand + " " + word
		}
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	cmd.Env = append(os.Environ(), config.Env...)
	if config.HoverStdin {
		cmd.Stdin = strings.NewReader(word)
	}
	b, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	return &Hover{
		Contents: strings.TrimSpace(string(b)),
		Range: &Range{
			Start: Position{
				Line:      params.Position.Line,
				Character: prevPos,
			},
			End: Position{
				Line:      params.Position.Line,
				Character: currPos,
			},
		},
	}, nil
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
	case "textDocument/hover":
		return h.handleTextDocumentHover(ctx, conn, req)
	}

	return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound, Message: fmt.Sprintf("method not supported: %s", req.Method)}
}
