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

func NewHandler(efms []string, stdin bool, cmd string, args ...string) jsonrpc2.Handler {
	if efms == nil || len(efms) == 0 {
		efms = []string{"%f:%l:%m", "%f:%l:%c:%m"}
	}
	var langHandler = &LangHandler{
		files: make(map[string]*File),
		cmd:   cmd,
		args:  args,
		efms:  efms,
		stdin: stdin,
	}
	return jsonrpc2.HandlerWithError(langHandler.handle)
}

type LangHandler struct {
	files map[string]*File
	conn  *jsonrpc2.Conn
	cmd   string
	args  []string
	efms  []string
	stdin bool
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

func (h *LangHandler) UpdateFile(uri string, text string) error {
	f := &File{
		Text: text,
	}
	h.files[uri] = f

	fname, err := fromURI(uri)
	if err != nil {
		return err
	}

	efms, err := errorformat.NewErrorformat(h.efms)
	if err != nil {
		return err
	}
	dmap := make(map[string][]Diagnostic)
	for k, _ := range h.files {
		fname, err := fromURI(k)
		if err != nil {
			continue
		}
		dmap[fname] = []Diagnostic{}
	}

	cmd := exec.Command(h.cmd, h.args...)
	if h.stdin {
		cmd.Stdin = strings.NewReader(text)
	}
	b, err := cmd.CombinedOutput()
	if err != nil {
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
				if m.L == 0 {
					m.L = 1
				}
				if _, ok := dmap[m.F]; !ok {
					dmap[m.F] = []Diagnostic{}
				}
				fmt.Fprintf(os.Stderr, "%+v\n", m)
				dmap[m.F] = append(dmap[m.F], Diagnostic{
					Range: Range{
						Start: Position{
							Line:      m.L - 1,
							Character: m.C - 1,
						},
						End: Position{
							Line:      m.L - 1,
							Character: m.C - 1,
						},
					},
					Message:  m.M,
					Severity: 1,
				})
			}
		}
	}
	for k, v := range dmap {
		var diagnosticsParams PublishDiagnosticsParams
		diagnosticsParams.URI = toURI(k).String()
		diagnosticsParams.Diagnostics = v
		h.conn.Notify(context.Background(), "textDocument/publishDiagnostics", &diagnosticsParams)
	}
	return nil
}

func (h *LangHandler) handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	switch req.Method {
	case "initialize":
		return h.handleInitialize(ctx, conn, req)
	case "textDocument/didOpen":
		return h.handleTextDocumentDidOpen(ctx, conn, req)
	case "textDocument/didChange":
		return h.handleTextDocumentDidChange(ctx, conn, req)
	case "textDocument/didSave":
		return h.handleTextDocumentDidSave(ctx, conn, req)
	}

	return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound, Message: fmt.Sprintf("method not supported: %s", req.Method)}
}
