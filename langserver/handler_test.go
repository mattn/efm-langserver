package langserver

import (
	"testing"
)

func TestNoLinter(t *testing.T) {
	h := &langHandler{
		configs: map[string][]Language{},
		files: map[string]*File{
			"file:///foo": &File{},
		},
	}

	_, err := h.lint("file:///foo")
	if err == nil {
		t.Error("Should be an error if no linters")
	}
}

func TestNoFileMatched(t *testing.T) {
	h := &langHandler{
		configs: map[string][]Language{},
		files: map[string]*File{
			"file:///foo": &File{},
		},
	}

	_, err := h.lint("file:///bar")
	if err == nil {
		t.Error("Should be an error if no linters")
	}
}
