package langserver

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
)

func TestFindTags(t *testing.T) {
	cwd, _ := os.Getwd()
	cwd = filepath.Clean(filepath.Join(cwd, ".."))
	h := &langHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		rootPath: cwd,
	}
	base := h.findTags(filepath.Join(cwd, "testdata/foo/bar"))
	if base == "" {
		t.Fatal("tags file must be found")
	}
	if base != filepath.Clean(filepath.Join(cwd, "testdata")) {
		t.Fatal("tags file must be location at testdata/tags")
	}
}

func TestCtags(t *testing.T) {
	cwd, _ := os.Getwd()
	cwd = filepath.Clean(filepath.Join(cwd, ".."))
	h := &langHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		rootPath: cwd,
	}
	locations, err := h.ctags(filepath.Join(cwd, "testdata/tags"), "langHandler")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(locations)
}
