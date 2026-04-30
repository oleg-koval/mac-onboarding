package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_example(t *testing.T) {
	// Load the example config — must parse without error.
	root := filepath.Join("..", "..")
	example := filepath.Join(root, "onboard.yaml.example")
	if _, err := os.Stat(example); err != nil {
		t.Skip("onboard.yaml.example not yet present")
	}
	if _, err := Load(example); err != nil {
		t.Fatalf("Load(example): %v", err)
	}
}

func TestLoad_missingFile(t *testing.T) {
	_, err := Load("/nonexistent/onboard.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoad_isSkipped(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "onboard.yaml")
	content := `
source:
  host: my-mac
modules:
  alfred:
    skip: true
  kitty:
    skip: false
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.IsSkipped("alfred") {
		t.Error("alfred should be skipped")
	}
	if cfg.IsSkipped("kitty") {
		t.Error("kitty should not be skipped")
	}
	if cfg.IsSkipped("brew") {
		t.Error("brew (not in config) should not be skipped")
	}
}

func TestExpand(t *testing.T) {
	home, _ := os.UserHomeDir()
	got := expand("~/foo/bar")
	want := filepath.Join(home, "foo/bar")
	if got != want {
		t.Errorf("expand: got %q want %q", got, want)
	}
}
