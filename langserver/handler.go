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
	"sync"
	"time"
	"unicode"
	"unicode/utf16"

	"github.com/reviewdog/errorformat"
	"github.com/sourcegraph/jsonrpc2"

	"github.com/mattn/go-unicodeclass"
)

type eventType int

const (
	eventTypeChange eventType = iota
	eventTypeSave
	eventTypeOpen
)

type lintRequest struct {
	URI       DocumentURI
	EventType eventType
}

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
func NewHandler(logger *log.Logger, config *Config) jsonrpc2.Handler {
	handler := &langHandler{
		loglevel:     config.LogLevel,
		logger:       logger,
		configs:      *config.Languages,
		files:        make(map[DocumentURI]*File),
		request:      make(chan lintRequest),
		lintDebounce: time.Duration(config.LintDebounce),
		lintTimer:    nil,

		formatDebounce: time.Duration(config.FormatDebounce),
		formatTimer:    nil,
		conn:           nil,
		filename:       config.Filename,
		rootMarkers:    *config.RootMarkers,

		lastPublishedURIs: make(map[string]map[DocumentURI]struct{}),
	}
	go handler.linter()
	return jsonrpc2.HandlerWithError(handler.handle)
}

type langHandler struct {
	mu             sync.Mutex
	loglevel       int
	logger         *log.Logger
	configs        map[string][]Language
	files          map[DocumentURI]*File
	request        chan lintRequest
	lintDebounce   time.Duration
	lintTimer      *time.Timer
	formatDebounce time.Duration
	formatTimer    *time.Timer
	conn           *jsonrpc2.Conn
	rootPath       string
	filename       string
	rootMarkers    []string

	// lastPublishedURIs is mapping from LanguageID string to mapping of
	// whether diagnostics are published in a DocumentURI or not.
	lastPublishedURIs map[string]map[DocumentURI]struct{}
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

func (h *langHandler) lintRequest(uri DocumentURI, eventType eventType) {
	if h.lintTimer != nil {
		h.lintTimer.Reset(h.lintDebounce)
		return
	}
	h.lintTimer = time.AfterFunc(h.lintDebounce, func() {
		h.lintTimer = nil
		h.request <- lintRequest{URI: uri, EventType: eventType}
	})
}

func (h *langHandler) logMessage(typ MessageType, message string) {
	_ = h.conn.Notify(
		context.Background(),
		"window/logMessage",
		&LogMessageParams{
			Type:    typ,
			Message: message,
		})
}

func (h *langHandler) linter() {
	running := make(map[DocumentURI]context.CancelFunc)

	for {
		lintReq, ok := <-h.request
		if !ok {
			break
		}

		cancel, ok := running[lintReq.URI]
		if ok {
			cancel()
		}

		ctx, cancel := context.WithCancel(context.Background())
		running[lintReq.URI] = cancel

		go func() {
			uriToDiagnostics, err := h.lint(ctx, lintReq.URI, lintReq.EventType)
			if err != nil {
				h.logger.Println(err)
				return
			}

			for diagURI, diagnostics := range uriToDiagnostics {
				if diagURI == "file:" {
					diagURI = lintReq.URI
				}
				version := 0
				if _, ok := h.files[lintReq.URI]; ok {
					version = h.files[lintReq.URI].Version
				}
				_ = h.conn.Notify(
					ctx,
					"textDocument/publishDiagnostics",
					&PublishDiagnosticsParams{
						URI:         diagURI,
						Diagnostics: diagnostics,
						Version:     version,
					})
			}
		}()
	}
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

func (h *langHandler) lint(ctx context.Context, uri DocumentURI, eventType eventType) (map[DocumentURI][]Diagnostic, error) {
	f, ok := h.files[uri]
	if !ok {
		return nil, fmt.Errorf("document not found: %v", uri)
	}

	fname, err := fromURI(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid uri: %v: %v", err, uri)
	}
	fname = filepath.ToSlash(fname)

	var configs []Language
	if cfgs, ok := h.configs[f.LanguageID]; ok {
		for _, cfg := range cfgs {
			// if we require markers and find that they dont exist we do not add the configuration
			if dir := matchRootPath(fname, cfg.RootMarkers); dir == "" && cfg.RequireMarker {
				continue
			}
			switch eventType {
			case eventTypeOpen:
				// if LintAfterOpen is not true, ignore didOpen
				if !cfg.LintAfterOpen {
					continue
				}
			case eventTypeChange:
				// if LintOnSave is true, ignore didChange
				if cfg.LintOnSave {
					continue
				}
			default:
			}
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
		if h.loglevel >= 1 {
			h.logger.Printf("lint for LanguageID not supported: %v", f.LanguageID)
		}
		return map[DocumentURI][]Diagnostic{}, nil
	}

	uriToDiagnostics := map[DocumentURI][]Diagnostic{
		uri: {},
	}
	publishedURIs := make(map[DocumentURI]struct{})
	for i, config := range configs {
		// To publish empty diagnostics when errors are fixed
		if config.LintWorkspace {
			for lastPublishedURI := range h.lastPublishedURIs[f.LanguageID] {
				if _, ok := uriToDiagnostics[lastPublishedURI]; !ok {
					uriToDiagnostics[lastPublishedURI] = []Diagnostic{}
				}
			}
		}

		if config.LintCommand == "" {
			continue
		}

		command := config.LintCommand
		if !config.LintStdin && !config.LintWorkspace && !strings.Contains(command, "${INPUT}") {
			command = command + " ${INPUT}"
		}
		rootPath := h.findRootPath(fname, config)
		command = replaceCommandInputFilename(command, fname, rootPath)

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
			cmd = exec.CommandContext(ctx, "cmd", "/c", command)
		} else {
			cmd = exec.CommandContext(ctx, "sh", "-c", command)
		}
		cmd.Dir = rootPath
		cmd.Env = append(os.Environ(), config.Env...)
		if config.LintStdin {
			cmd.Stdin = strings.NewReader(f.Text)
		}
		b, err := cmd.CombinedOutput()
		if err != nil {
			if succeeded(err) {
				return nil, nil
			}
		}
		// Most of lint tools exit with non-zero value. But some commands
		// return with zero value. We can not handle the output is real result
		// or output of usage. So efm-langserver ignore that command exiting
		// with zero-value. So if you want to handle the command which exit
		// with zero value, please specify lint-ignore-exit-code.
		if err == nil && !config.LintIgnoreExitCode {
			h.logMessage(LogError, "command `"+command+"` exit with zero. probably you forgot to specify `lint-ignore-exit-code: true`.")
			continue
		}
		if h.loglevel >= 3 {
			h.logger.Println(command+":", string(b))
		}
		var source *string
		if config.LintSource != "" {
			source = &configs[i].LintSource
		}

		var prefix string
		if config.Prefix != "" {
			prefix = fmt.Sprintf("[%s] ", config.Prefix)
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
				if runtime.GOOS == "windows" && !strings.EqualFold(path, fname) {
					continue
				} else if path != fname {
					continue
				}
			} else {
				entry.Filename = filepath.ToSlash(entry.Filename)
			}
			word := ""

			// entry.Col is expected to be one based, if the linter returns zero based we
			// have the ability to add an offset here.
			// We only add the offset if the linter reports entry.Col > 0 because 0 means the whole line
			if config.LintOffsetColumns > 0 && entry.Col > 0 {
				entry.Col = entry.Col + config.LintOffsetColumns
			}

			if entry.Lnum == 0 {
				entry.Lnum = 1 // entry.Lnum == 0 indicates the top line, set to 1 because it is subtracted later
			}

			if entry.Col == 0 {
				entry.Col = 1 // entry.Col == 0 indicates the whole line without column, set to 1 because it is subtracted later
			} else {
				word = f.WordAt(Position{Line: entry.Lnum - 1 - config.LintOffset, Character: entry.Col - 1})
			}

			// we allow the config to provide a mapping between LSP types E,W,I,N and whatever categories the linter has
			if len(config.LintCategoryMap) > 0 {
				entry.Type = []rune(config.LintCategoryMap[string(entry.Type)])[0]
			}

			severity := 1
			if config.LintSeverity != 0 {
				severity = config.LintSeverity
			}

			switch entry.Type {
			case 'E', 'e':
				severity = 1
			case 'W', 'w':
				severity = 2
			case 'I', 'i':
				severity = 3
			case 'N', 'n':
				severity = 4
			}

			diagURI := uri
			if entry.Filename != "" {
				if filepath.IsAbs(entry.Filename) {
					diagURI = toURI(entry.Filename)
				} else {
					diagURI = toURI(filepath.Join(rootPath, entry.Filename))
				}
			}
			if runtime.GOOS == "windows" {
				if !strings.EqualFold(string(diagURI), string(uri)) && !config.LintWorkspace {
					continue
				}
			} else {
				if diagURI != uri && !config.LintWorkspace {
					continue
				}
			}

			if config.LintWorkspace {
				publishedURIs[diagURI] = struct{}{}
			}
			uriToDiagnostics[diagURI] = append(uriToDiagnostics[diagURI], Diagnostic{
				Range: Range{
					Start: Position{Line: entry.Lnum - 1 - config.LintOffset, Character: entry.Col - 1},
					End:   Position{Line: entry.Lnum - 1 - config.LintOffset, Character: entry.Col - 1 + len([]rune(word))},
				},
				Code:     itoaPtrIfNotZero(entry.Nr),
				Message:  prefix + entry.Text,
				Severity: severity,
				Source:   source,
			})
		}
	}

	// Update state here as no possibility of cancelation
	for _, config := range configs {
		if config.LintWorkspace {
			h.lastPublishedURIs[f.LanguageID] = publishedURIs
			break
		}
	}
	return uriToDiagnostics, nil
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
	h.lintRequest(uri, eventTypeSave)
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

func (h *langHandler) updateFile(uri DocumentURI, text string, version *int, eventType eventType) error {
	f, ok := h.files[uri]
	if !ok {
		return fmt.Errorf("document not found: %v", uri)
	}
	f.Text = text
	if version != nil {
		f.Version = *version
	}

	h.lintRequest(uri, eventType)
	return nil
}

func (h *langHandler) handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
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
