package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveFileUnlinksARegularFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("gone"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := RemoveFile(path); err != nil {
		t.Fatalf("RemoveFile() = %v", err)
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("file still exists: %v", err)
	}
}

func TestRemoveFileRefusesADirectory(t *testing.T) {
	dir := t.TempDir()
	if err := RemoveFile(dir); err == nil {
		t.Fatal("RemoveFile accepted a directory")
	}
}

func TestRemoveFileRefusesASymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.md")
	if err := os.WriteFile(target, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.md")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := RemoveFile(link); err == nil {
		t.Fatal("RemoveFile accepted a symlink")
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("symlink target was touched: %v", err)
	}
}

func TestContainsPathAcceptsStoreChildrenAndRejectsEscapes(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "notes", "a.md")
	if !ContainsPath(root, inside) {
		t.Fatalf("ContainsPath rejected %q under %q", inside, root)
	}
	if ContainsPath(root, filepath.Join(root, "..", "outside.md")) {
		t.Fatal("ContainsPath accepted a parent escape")
	}
	if ContainsPath("", inside) || ContainsPath(root, "") {
		t.Fatal("ContainsPath accepted an empty side")
	}
}
