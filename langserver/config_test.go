package langserver

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigHoverCharsDefault(t *testing.T) {
	dir := t.TempDir()
	yamlfile := filepath.Join(dir, "config.yaml")
	content := `version: 2
languages:
  vim:
    - hover-command: 'echo hover'
  lua:
    - hover-command: 'echo hover'
      hover-chars: '.'
`
	if err := os.WriteFile(yamlfile, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	config, err := LoadConfig(yamlfile)
	if err != nil {
		t.Fatal(err)
	}

	vim := (*config.Languages)["vim"]
	if len(vim) != 1 {
		t.Fatalf("vim should have one language config but got: %v", vim)
	}
	if vim[0].HoverChars != "_" {
		t.Fatalf("hover-chars should default to %q but got: %q", "_", vim[0].HoverChars)
	}

	lua := (*config.Languages)["lua"]
	if len(lua) != 1 {
		t.Fatalf("lua should have one language config but got: %v", lua)
	}
	if lua[0].HoverChars != "." {
		t.Fatalf("hover-chars should be %q but got: %q", ".", lua[0].HoverChars)
	}
}
