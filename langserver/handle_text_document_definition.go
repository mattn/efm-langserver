package langserver

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/mattn/go-unicodeclass"
	"github.com/sourcegraph/jsonrpc2"
)

func (h *langHandler) handleTextDocumentDefinition(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params DocumentDefinitionParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	return h.definition(params.TextDocument.URI, &params)
}

func (h *langHandler) findTag(fname string, tag string) ([]Location, error) {
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	locations := []Location{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.HasPrefix(text, "!") {
			continue
		}
		token := strings.SplitN(text, "\t", 4)
		if len(token) < 4 {
			continue
		}
		if token[0] == tag {
			token[2] = strings.TrimRight(token[2], `;"`)
			fullpath := filepath.Clean(filepath.Join(h.rootPath, token[1]))
			b, err := ioutil.ReadFile(fullpath)
			if err != nil {
				continue
			}
			lines := strings.Split(string(b), "\n")
			if strings.HasPrefix(token[2], "/") {
				pattern := token[2]
				hasPrefix := strings.HasPrefix(pattern, "/^")
				if hasPrefix {
					pattern = strings.TrimLeft(pattern, "/^")
				}
				hasSuffix := strings.HasSuffix(pattern, "$/")
				if hasSuffix {
					pattern = strings.TrimRight(pattern, "$/")
				}
				for i, line := range lines {
					match := false
					if hasPrefix && hasSuffix && line == pattern {
						match = true
					} else if hasPrefix && strings.HasPrefix(line, pattern) {
						match = true
					} else if hasSuffix && strings.HasSuffix(line, pattern) {
						match = true
					}
					if match {
						locations = append(locations, Location{
							URI: toURI(fullpath),
							Range: Range{
								Start: Position{Line: i, Character: 0},
								End:   Position{Line: i, Character: 0},
							},
						})
					}
				}
			} else {
				i, err := strconv.Atoi(token[2])
				if err != nil {
					continue
				}
				locations = append(locations, Location{
					URI: toURI(fullpath),
					Range: Range{
						Start: Position{Line: i - 1, Character: 0},
						End:   Position{Line: i - 1, Character: 0},
					},
				})
			}
		}
	}
	return locations, nil
}

func (h *langHandler) findTagsFile(fname string) string {
	base := filepath.Clean(filepath.Dir(fname))
	for {
		_, err := os.Stat(filepath.Join(base, "tags"))
		if err == nil {
			break
		}
		if base == h.rootPath {
			base = ""
			break
		}
		tmp := filepath.Dir(base)
		if tmp == "" || tmp == base {
			base = ""
			break
		}
		base = tmp
	}
	return base
}

func (h *langHandler) definition(uri DocumentURI, params *DocumentDefinitionParams) ([]Location, error) {
	f, ok := h.files[uri]
	if !ok {
		return nil, fmt.Errorf("document not found: %v", uri)
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
	tag := string(utf16.Decode(chars[prevPos:currPos]))

	fname, err := fromURI(uri)
	if err != nil {
		return nil, nil
	}
	fname = filepath.ToSlash(fname)
	if runtime.GOOS == "windows" {
		fname = strings.ToLower(fname)
	}

	base := h.findTagsFile(fname)
	if base == "" {
		return nil, nil
	}
	return h.findTag(filepath.Join(base, "tags"), tag)
}
