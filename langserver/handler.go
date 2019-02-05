package langserver

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/haya14busa/errorformat"
	"github.com/sourcegraph/jsonrpc2"
)

func NewHandler(efms []string, stdin bool, offset int, cmd string, args ...string) jsonrpc2.Handler {
	if efms == nil || len(efms) == 0 {
		efms = []string{"%f:%l:%m", "%f:%l:%c:%m"}
	}
	var langHandler = &LangHandler{
		efms:    efms,
		cmd:     cmd,
		stdin:   stdin,
		offset:  offset,
		args:    args,
		files:   make(map[string]*File),
		request: make(chan string),
	}
	go langHandler.linter()
	return jsonrpc2.HandlerWithError(langHandler.handle)
}

type LangHandler struct {
	efms    []string
	stdin   bool
	offset  int
	cmd     string
	args    []string
	conn    *jsonrpc2.Conn
	files   map[string]*File
	request chan string
}

type File struct {
	Text string
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

func (h *LangHandler) linter() {
	for {
		uri, ok := <-h.request
		if !ok {
			break
		}
		for k, v := range h.lint(uri) {
			var diagnosticsParams PublishDiagnosticsParams
			diagnosticsParams.URI = toURI(k).String()
			diagnosticsParams.Diagnostics = v
			h.conn.Notify(context.Background(), "textDocument/publishDiagnostics", &diagnosticsParams)
		}
	}
}

func (h *LangHandler) lint(uri string) map[string][]Diagnostic {
	f, ok := h.files[uri]
	if !ok {
		fmt.Fprintf(os.Stderr, "document not found")
		return nil
	}

	fname, err := fromURI(uri)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		return nil
	}

	efms, err := errorformat.NewErrorformat(h.efms)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		return nil
	}
	diagnostics := make(map[string][]Diagnostic)
	for k, _ := range h.files {
		fname, err := fromURI(k)
		if err != nil {
			continue
		}
		diagnostics[fname] = []Diagnostic{}
	}

	cmd := exec.Command(h.cmd, h.args...)
	if h.stdin {
		cmd.Stdin = strings.NewReader(f.Text)
	}
	b, err := cmd.CombinedOutput()
	if err == nil {
		fmt.Fprintf(os.Stderr, "succeeded: %q", f.Text)
		return diagnostics
	}
	for _, line := range strings.Split(string(b), "\n") {
		for _, ef := range efms.Efms {
			m := ef.Match(string(line))
			if m == nil {
				continue
			}
			if h.stdin && (m.F == "stdin" || m.F == "-") {
				m.F = fname
			} else {
				m.F = filepath.ToSlash(m.F)
			}
			if m.C == 0 {
				m.C = 1
			}
			if _, ok := diagnostics[m.F]; !ok {
				diagnostics[m.F] = []Diagnostic{}
			}
			diagnostics[m.F] = append(diagnostics[m.F], Diagnostic{
				Range: Range{
					Start: Position{
						Line:      m.L - 1 - h.offset,
						Character: m.C - 1,
					},
					End: Position{
						Line:      m.L - 1 - h.offset,
						Character: m.C - 1,
					},
				},
				Message:  m.M,
				Severity: 1,
			})
		}
	}
	return diagnostics
}

func (h *LangHandler) closeFile(uri string) error {
	delete(h.files, uri)
	return nil
}

func (h *LangHandler) saveFile(uri string) error {
	h.request <- uri
	return nil
}

func (h *LangHandler) updateFile(uri string, text string) error {
	f := &File{
		Text: text,
	}
	h.files[uri] = f

	h.request <- uri
	return nil
}

func (h *LangHandler) handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
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
	}

	return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound, Message: fmt.Sprintf("method not supported: %s", req.Method)}
}
