package langserver

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
)

func TestFindTagFile(t *testing.T) {
	cwd, _ := os.Getwd()
	cwd = filepath.Clean(filepath.Join(cwd, ".."))
	h := &langHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		rootPath: cwd,
	}
	base := h.findTagsFile(filepath.Join(cwd, "testdata/foo/bar"))
	if base == "" {
		t.Fatal("tags file must be found")
	}
	if base != filepath.Clean(filepath.Join(cwd, "testdata")) {
		t.Fatal("tags file must be location at testdata/tags")
	}
}

func TestFindTag(t *testing.T) {
	cwd, _ := os.Getwd()
	cwd = filepath.Clean(filepath.Join(cwd, ".."))
	h := &langHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		rootPath: cwd,
	}
	locations, err := h.findTag(filepath.Join(cwd, "testdata/tags"), "langHandler")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(locations)
}
