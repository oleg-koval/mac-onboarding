package archive

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddAndExtractFile(t *testing.T) {
	dir := t.TempDir()
	archPath := filepath.Join(dir, "test.tar.gz")

	// Write a source file.
	src := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(src, []byte("hello world"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := AddFile(archPath, src, "test/hello.txt"); err != nil {
		t.Fatalf("AddFile: %v", err)
	}

	tmp, err := ExtractFile(archPath, "test/hello.txt")
	if err != nil {
		t.Fatalf("ExtractFile: %v", err)
	}
	defer os.Remove(tmp)

	data, _ := os.ReadFile(tmp)
	if string(data) != "hello world" {
		t.Errorf("got %q, want %q", data, "hello world")
	}
}

func TestAddFileReplace(t *testing.T) {
	dir := t.TempDir()
	archPath := filepath.Join(dir, "test.tar.gz")

	src := filepath.Join(dir, "f.txt")
	os.WriteFile(src, []byte("v1"), 0600)
	AddFile(archPath, src, "f.txt")

	os.WriteFile(src, []byte("v2"), 0600)
	AddFile(archPath, src, "f.txt")

	tmp, _ := ExtractFile(archPath, "f.txt")
	defer os.Remove(tmp)
	data, _ := os.ReadFile(tmp)
	if string(data) != "v2" {
		t.Errorf("expected replacement: got %q", data)
	}
}

func TestExtractFileMissing(t *testing.T) {
	dir := t.TempDir()
	archPath := filepath.Join(dir, "test.tar.gz")

	src := filepath.Join(dir, "a.txt")
	os.WriteFile(src, []byte("a"), 0600)
	AddFile(archPath, src, "a.txt")

	_, err := ExtractFile(archPath, "nonexistent.txt")
	if err == nil {
		t.Error("expected error for missing entry")
	}
}

func TestAddAndExtractDir(t *testing.T) {
	dir := t.TempDir()
	archPath := filepath.Join(dir, "test.tar.gz")

	srcDir := filepath.Join(dir, "src")
	os.MkdirAll(filepath.Join(srcDir, "sub"), 0755)
	os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("a"), 0600)
	os.WriteFile(filepath.Join(srcDir, "sub", "b.txt"), []byte("b"), 0600)

	if err := AddDir(archPath, srcDir, "mymod"); err != nil {
		t.Fatalf("AddDir: %v", err)
	}

	dstDir := filepath.Join(dir, "dst")
	if err := ExtractDir(archPath, "mymod", dstDir); err != nil {
		t.Fatalf("ExtractDir: %v", err)
	}

	check := func(rel, want string) {
		data, err := os.ReadFile(filepath.Join(dstDir, rel))
		if err != nil {
			t.Errorf("read %s: %v", rel, err)
			return
		}
		if string(data) != want {
			t.Errorf("%s: got %q want %q", rel, data, want)
		}
	}
	check("a.txt", "a")
	check("sub/b.txt", "b")
}

func TestListEntries(t *testing.T) {
	dir := t.TempDir()
	archPath := filepath.Join(dir, "test.tar.gz")

	for _, name := range []string{"x.txt", "y.txt"} {
		src := filepath.Join(dir, name)
		os.WriteFile(src, []byte(name), 0600)
		AddFile(archPath, src, "mod/"+name)
	}

	entries, err := ListEntries(archPath)
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d: %v", len(entries), entries)
	}
}
