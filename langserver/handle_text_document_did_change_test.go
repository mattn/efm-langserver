package langserver

import (
	"context"
	"encoding/json"
	"log"
	"testing"
	"time"

	"github.com/sourcegraph/jsonrpc2"
)

func TestDidChangeMultipleContentChanges(t *testing.T) {
	uri := DocumentURI("file:///foo")
	h := &langHandler{
		logger:       log.New(log.Writer(), "", log.LstdFlags),
		lintDebounce: time.Millisecond,
		request:      make(chan lintRequest, 1),
		pendingLints: make(map[DocumentURI]eventType),
		files: map[DocumentURI]*File{
			uri: {
				LanguageID: "vim",
				Text:       "old",
			},
		},
	}

	params, err := json.Marshal(DidChangeTextDocumentParams{
		TextDocument: VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: TextDocumentIdentifier{URI: uri},
			Version:                2,
		},
		ContentChanges: []TextDocumentContentChangeEvent{
			{Text: "first"},
			{Text: "second"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	raw := json.RawMessage(params)

	req := &jsonrpc2.Request{Params: &raw}
	if _, err := h.handleTextDocumentDidChange(context.Background(), nil, req); err != nil {
		t.Fatal(err)
	}

	if h.files[uri].Text != "second" {
		t.Fatalf("text should be %q but got: %q", "second", h.files[uri].Text)
	}
	if h.files[uri].Version != 2 {
		t.Fatalf("version should be %v but got: %v", 2, h.files[uri].Version)
	}
}
