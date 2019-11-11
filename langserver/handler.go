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

	"github.com/reviewdog/errorformat"
	"github.com/sourcegraph/jsonrpc2"
)

type Config struct {
	LogWriter io.Writer            `yaml:"-"`
	Languages map[string]*Language `yaml:"languages"`
}

type Language struct {
	LintFormats   []string `yaml:"lint-formats"`
	LintStdin     bool     `yaml:"lint-stdin"`
	LintOffset    int      `yaml:"lint-offset"`
	LintCommand   string   `yaml:"lint-command"`
	FormatCommand string   `yaml:"format-command"`
	SymbolCommand string   `yaml:"symbol-command"`
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
		diagnostics := h.lint(uri)
		if diagnostics == nil {
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

func (h *langHandler) lint(uri string) []Diagnostic {
	f, ok := h.files[uri]
	if !ok {
		h.logger.Printf("document not found: %v", uri)
		return nil
	}

	config, ok := h.configs[f.LanguageId]
	if !ok || config.LintCommand == "" {
		h.logger.Printf("lint for languageId not supported: %v", f.LanguageId)
		return nil
	}

	fname, err := fromURI(uri)
	if err != nil {
		h.logger.Printf("invalid uri: %v: %v", err, uri)
		return nil
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
		h.logger.Printf("invalid error-format: %v", config.LintFormats)
		return nil
	}

	diagnostics := []Diagnostic{}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	if config.LintStdin {
		cmd.Stdin = strings.NewReader(f.Text)
	}
	b, err := cmd.CombinedOutput()
	if err == nil {
		h.logger.Println("lint succeeded")
		return diagnostics
	}
	for _, line := range strings.Split(string(b), "\n") {
		for _, ef := range efms.Efms {
			m := ef.Match(string(line))
			if m == nil {
				continue
			}
			if config.LintStdin && (m.F == "stdin" || m.F == "-") {
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
			diagnostics = append(diagnostics, Diagnostic{
				Range: Range{
					Start: Position{Line: m.L - 1 - config.LintOffset, Character: m.C - 1},
					End:   Position{Line: m.L - 1 - config.LintOffset, Character: m.C - 1},
				},
				Message:  m.M,
				Severity: 1,
			})
		}
	}

	return diagnostics
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
	cmd.Stdin = strings.NewReader(f.Text)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.New(string(b))
	}
	h.logger.Println("format succeeded")
	text := strings.Replace(string(b), "\r", "", -1)
	lines := strings.Split(text, "\n")

	return []TextEdit{
		{
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: len(lines), Character: len(lines[len(lines)-1])},
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
	if !ok {
		return nil, fmt.Errorf("symbol for languageId not supported: %v", f.LanguageId)
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

func (h *langHandler) handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	switch req.Method {
	case "initialize":
		return h.handleInitialize(ctx, conn, req)
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
	}

	return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound, Message: fmt.Sprintf("method not supported: %s", req.Method)}
}
