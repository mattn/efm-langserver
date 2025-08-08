package langserver

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf16"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/mattn/go-unicodeclass"
)

type eventType int

const (
	eventTypeChange eventType = iota
	eventTypeSave
	eventTypeOpen
)

// Config is
type Config struct {
	Version        int
	LogLevel       int
	Languages      *map[string][]Language
	RootMarkers    *[]string
	LintDebounce   time.Duration
	FormatDebounce time.Duration

	Filename string
}

func NewConfig() *Config {
	languages := make(map[string][]Language)
	rootMarkers := make([]string, 0)
	return &Config{
		Languages:   &languages,
		RootMarkers: &rootMarkers,
	}
}

// Language is
type Language struct {
	Prefix             string
	LintFormats        []string
	LintStdin          bool
	LintOffset         int
	LintOffsetColumns  int
	LintCommand        string
	LintIgnoreExitCode bool
	LintCategoryMap    map[string]string
	LintSource         string
	LintSeverity       int
	LintWorkspace      bool
	LintAfterOpen      bool
	LintOnSave         bool
	FormatCommand      string
	FormatCanRange     bool
	FormatStdin        bool
	Env                []string
	RootMarkers        []string
	RequireMarker      bool
}

// NewHandler create JSON-RPC handler for this language server.
func NewHandler(logger *log.Logger, config *Config) *langHandler {
	handler := &langHandler{
		loglevel:     config.LogLevel,
		logger:       logger,
		configs:      *config.Languages,
		files:        make(map[DocumentURI]*File),
		lintDebounce: time.Duration(config.LintDebounce),
		lintTimer:    nil,

		formatDebounce: time.Duration(config.FormatDebounce),
		formatTimer:    nil,
		filename:       config.Filename,
		rootMarkers:    *config.RootMarkers,

		lastPublishedURIs: make(map[string]map[DocumentURI]struct{}),
	}
	return handler
}

type langHandler struct {
	mu             sync.Mutex
	loglevel       int
	logger         *log.Logger
	configs        map[string][]Language
	files          map[DocumentURI]*File
	lintDebounce   time.Duration
	lintTimer      *time.Timer
	formatDebounce time.Duration
	formatTimer    *time.Timer
	rootPath       string
	filename       string
	rootMarkers    []string

	// lastPublishedURIs is mapping from LanguageID string to mapping of
	// whether diagnostics are published in a DocumentURI or not.
	lastPublishedURIs map[string]map[DocumentURI]struct{}
}

func (h *langHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
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
	case "textDocument/rangeFormatting":
		return h.handleTextDocumentRangeFormatting(ctx, conn, req)
	case "workspace/didChangeConfiguration":
		return h.handleWorkspaceDidChangeConfiguration(ctx, conn, req)
	}

	return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound, Message: fmt.Sprintf("method not supported: %s", req.Method)}
}

// File is
type File struct {
	LanguageID string
	Text       string
	Version    int
}

// WordAt is
func (f *File) WordAt(pos Position) string {
	lines := strings.Split(f.Text, "\n")
	if pos.Line < 0 || pos.Line >= len(lines) {
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
	if currPos == -1 {
		currPos = len(chars)
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

func (h *langHandler) logMessage(conn *jsonrpc2.Conn, typ MessageType, message string) {
	_ = conn.Notify(
		context.Background(),
		"window/logMessage",
		&LogMessageParams{
			Type:    typ,
			Message: message,
		})
}

func matchRootPath(fname string, markers []string) string {
	dir := filepath.Dir(filepath.Clean(fname))
	var prev string
	for dir != prev {
		files, _ := os.ReadDir(dir)
		for _, file := range files {
			name := file.Name()
			isDir := file.IsDir()
			for _, marker := range markers {
				if strings.HasSuffix(marker, "/") {
					if !isDir {
						continue
					}
					marker = strings.TrimRight(marker, "/")
					if ok, _ := filepath.Match(marker, name); ok {
						return dir
					}
				} else {
					if isDir {
						continue
					}
					if ok, _ := filepath.Match(marker, name); ok {
						return dir
					}
				}
			}
		}
		prev = dir
		dir = filepath.Dir(dir)
	}

	return ""
}

func (h *langHandler) findRootPath(fname string, lang Language) string {
	if dir := matchRootPath(fname, lang.RootMarkers); dir != "" {
		return dir
	}
	if dir := matchRootPath(fname, h.rootMarkers); dir != "" {
		return dir
	}

	return h.rootPath
}

func isFilename(s string) bool {
	switch s {
	case "stdin", "-", "<text>", "<stdin>":
		return true
	default:
		return false
	}
}

func itoaPtrIfNotZero(n int) *string {
	if n == 0 {
		return nil
	}
	s := strconv.Itoa(n)
	return &s
}

func (h *langHandler) onCloseFile(uri DocumentURI) error {
	delete(h.files, uri)
	return nil
}

func (h *langHandler) onSaveFile(conn *jsonrpc2.Conn, uri DocumentURI) error {
	h.ScheduleLinting(conn, uri, eventTypeSave)
	return nil
}

func (h *langHandler) onOpenFile(conn *jsonrpc2.Conn, uri DocumentURI, languageID string, version int, text string) error {
	f := &File{
		Text:       text,
		LanguageID: languageID,
		Version:    version,
	}
	h.files[uri] = f

	h.ScheduleLinting(conn, uri, eventTypeOpen)
	return nil
}

func (h *langHandler) onUpdateFile(conn *jsonrpc2.Conn, uri DocumentURI, text string, version *int, eventType eventType) error {
	f, ok := h.files[uri]
	if !ok {
		return fmt.Errorf("document not found: %v", uri)
	}
	f.Text = text
	if version != nil {
		f.Version = *version
	}

	h.ScheduleLinting(conn, uri, eventType)
	return nil
}

func replaceCommandInputFilename(command, fname, rootPath string) string {
	ext := filepath.Ext(fname)
	ext = strings.TrimPrefix(ext, ".")

	command = strings.ReplaceAll(command, "${INPUT}", escapeBrackets(fname))
	command = strings.ReplaceAll(command, "${FILEEXT}", ext)
	command = strings.ReplaceAll(command, "${FILENAME}", escapeBrackets(filepath.FromSlash(fname)))
	command = strings.ReplaceAll(command, "${ROOT}", escapeBrackets(rootPath))

	return command
}

func escapeBrackets(path string) string {
	path = strings.ReplaceAll(path, "(", `\(`)
	path = strings.ReplaceAll(path, ")", `\)`)

	return path
}

func succeeded(err error) bool {
	exitErr, ok := err.(*exec.ExitError)
	// When the context is canceled, the process is killed,
	// and the exit code is -1
	return ok && exitErr.ExitCode() < 0
}
