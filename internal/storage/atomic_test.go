package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWriteReplacesContentAndPreservesPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := AtomicWrite(path, []byte("new"), 0o644); err != nil {
		t.Fatalf("AtomicWrite() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" || info.Mode().Perm() != 0o600 {
		t.Fatalf("AtomicWrite() data = %q mode = %o", data, info.Mode().Perm())
	}
}

func TestAtomicWriteCleansTemporaryFileAfterRenameFailure(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := AtomicWrite(target, []byte("data"), 0o600); err == nil {
		t.Fatal("AtomicWrite() error = nil")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "target" {
		t.Fatalf("AtomicWrite() left temporary files: %#v", entries)
	}
}
