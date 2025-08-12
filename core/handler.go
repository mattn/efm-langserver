package core

import (
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

	"github.com/mattn/go-unicodeclass"

	"github.com/konradmalik/efm-langserver/types"
)

type LangHandler struct {
	formatMu       sync.Mutex
	lintMu         sync.Mutex
	loglevel       int
	logger         *log.Logger
	configs        map[string][]types.Language
	files          map[types.DocumentURI]*fileRef
	lintDebounce   time.Duration
	lintTimer      *time.Timer
	formatDebounce time.Duration
	formatTimer    *time.Timer
	RootPath       string
	rootMarkers    []string

	// lastPublishedURIs is mapping from LanguageID string to mapping of
	// whether diagnostics are published in a DocumentURI or not.
	lastPublishedURIs map[string]map[types.DocumentURI]struct{}
}

type fileRef struct {
	LanguageID string
	Text       string
	Version    int
}

func NewConfig() *types.Config {
	languages := make(map[string][]types.Language)
	rootMarkers := make([]string, 0)
	return &types.Config{
		Languages:   &languages,
		RootMarkers: &rootMarkers,
	}
}

func NewHandler(logger *log.Logger, config *types.Config) *LangHandler {
	handler := &LangHandler{
		loglevel:     config.LogLevel,
		logger:       logger,
		configs:      *config.Languages,
		files:        make(map[types.DocumentURI]*fileRef),
		lintDebounce: config.LintDebounce,
		lintTimer:    nil,

		formatDebounce: config.FormatDebounce,
		formatTimer:    nil,
		rootMarkers:    *config.RootMarkers,

		lastPublishedURIs: make(map[string]map[types.DocumentURI]struct{}),
	}
	return handler
}

func (h *LangHandler) Initialize(params types.InitializeParams) (types.InitializeResult, error) {
	if params.RootURI != "" {
		rootPath, err := fromURI(params.RootURI)
		if err != nil {
			return types.InitializeResult{}, err
		}
		h.RootPath = filepath.Clean(rootPath)
	}

	var hasFormatCommand bool
	var hasRangeFormatCommand bool

	if params.InitializationOptions != nil {
		hasFormatCommand = params.InitializationOptions.DocumentFormatting
		hasRangeFormatCommand = params.InitializationOptions.RangeFormatting
	}

	for _, config := range h.configs {
		for _, lang := range config {
			if lang.FormatCommand != "" {
				hasFormatCommand = true
				if lang.FormatCanRange {
					hasRangeFormatCommand = true
					break
				}
			}
		}
	}

	return types.InitializeResult{
		Capabilities: types.ServerCapabilities{
			TextDocumentSync:           types.TDSKFull,
			DocumentFormattingProvider: hasFormatCommand,
			RangeFormattingProvider:    hasRangeFormatCommand,
		},
	}, nil
}

func (h *LangHandler) UpdateConfiguration(config *types.Config) (any, error) {
	if config.Languages != nil {
		h.configs = *config.Languages
	}
	if config.RootMarkers != nil {
		h.rootMarkers = *config.RootMarkers
	}
	if config.LogLevel > 0 {
		h.loglevel = config.LogLevel
	}
	if config.LintDebounce > 0 {
		h.lintDebounce = config.LintDebounce
	}
	if config.FormatDebounce > 0 {
		h.formatDebounce = config.FormatDebounce
	}
	if config.LogLevel > 0 {
		h.loglevel = config.LogLevel
	}

	return nil, nil
}

func (h *LangHandler) Close() {
	if h.formatTimer != nil {
		h.formatTimer.Stop()
	}
	if h.lintTimer != nil {
		h.lintTimer.Stop()
	}
}

func (h *LangHandler) OnCloseFile(uri types.DocumentURI) error {
	delete(h.files, uri)
	return nil
}

func (h *LangHandler) OnSaveFile(notifier notifier, uri types.DocumentURI) error {
	h.ScheduleLinting(notifier, uri, types.EventTypeSave)
	return nil
}

func (h *LangHandler) OnOpenFile(notifier notifier, uri types.DocumentURI, languageID string, version int, text string) error {
	f := &fileRef{
		Text:       text,
		LanguageID: languageID,
		Version:    version,
	}
	h.files[uri] = f

	h.ScheduleLinting(notifier, uri, types.EventTypeOpen)
	return nil
}

func (h *LangHandler) OnUpdateFile(notifier notifier, uri types.DocumentURI, text string, version *int, eventType types.EventType) error {
	f, ok := h.files[uri]
	if !ok {
		return fmt.Errorf("document not found: %v", uri)
	}
	f.Text = text
	if version != nil {
		f.Version = *version
	}

	h.ScheduleLinting(notifier, uri, eventType)
	return nil
}

func (h *LangHandler) findRootPath(fname string, lang types.Language) string {
	if dir := matchRootPath(fname, lang.RootMarkers); dir != "" {
		return dir
	}
	if dir := matchRootPath(fname, h.rootMarkers); dir != "" {
		return dir
	}

	return h.RootPath
}

func (f *fileRef) wordAt(pos types.Position) string {
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

func fromURI(uri types.DocumentURI) (string, error) {
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

func toURI(path string) types.DocumentURI {
	if isWindowsDrivePath(path) {
		path = "/" + path
	}
	return types.DocumentURI((&url.URL{
		Scheme: "file",
		Path:   filepath.ToSlash(path),
	}).String())
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
