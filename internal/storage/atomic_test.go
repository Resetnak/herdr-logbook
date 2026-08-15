package storage

import (
	"os"
	"path/filepath"
	"runtime"
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
	if string(data) != "new" {
		t.Fatalf("AtomicWrite() data = %q", data)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("AtomicWrite() mode = %o", info.Mode().Perm())
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

func TestReadForRewriteHandlesMissingAndRegularFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.md")
	data, err := ReadForRewrite(path)
	if err != nil || data != nil {
		t.Fatalf("ReadForRewrite() missing file = %q, %v", data, err)
	}

	if err := os.WriteFile(path, []byte("# Note\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	data, err = ReadForRewrite(path)
	if err != nil || string(data) != "# Note\n" {
		t.Fatalf("ReadForRewrite() regular file = %q, %v", data, err)
	}
}

func TestReadForRewriteRefusesNonRegularFiles(t *testing.T) {
	dir := t.TempDir()
	if _, err := ReadForRewrite(dir); err == nil {
		t.Fatal("ReadForRewrite() accepted a directory")
	}

	if runtime.GOOS == "windows" {
		return
	}
	target := filepath.Join(dir, "target.md")
	link := filepath.Join(dir, "link.md")
	if err := os.WriteFile(target, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadForRewrite(link); err == nil {
		t.Fatal("ReadForRewrite() followed a symlink")
	}
}
