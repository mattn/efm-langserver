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

func TestFindTagPatternWithDollar(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	if err := os.WriteFile(src, []byte("start$\nstart extra\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tags := filepath.Join(dir, "tags")
	if err := os.WriteFile(tags, []byte("mytag\tsrc.txt\t/^start$$/;\"\tv\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	h := &langHandler{
		logger:   log.New(log.Writer(), "", log.LstdFlags),
		rootPath: dir,
	}
	locations, err := h.findTag(tags, "mytag")
	if err != nil {
		t.Fatal(err)
	}
	if len(locations) != 1 {
		t.Fatalf("locations should be only one but got: %v", locations)
	}
	if locations[0].Range.Start.Line != 0 {
		t.Fatalf("range.start.line should be %v but got: %v", 0, locations[0].Range.Start.Line)
	}
}
