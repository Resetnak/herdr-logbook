package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Resetnak/herdr-logbook/internal/config"
)

func TestResolveUsesCentralStorageByDefault(t *testing.T) {
	stateDir := t.TempDir()
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.Mkdir(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	layout, err := Resolve(stateDir, repo, "p_abc", config.Default(), "")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	want := filepath.Join(stateDir, "store", "projects", "p_abc")
	if layout.Mode != "central" || layout.Root != want || layout.Now != filepath.Join(want, "now.md") {
		t.Fatalf("Resolve() = %#v", layout)
	}
	if strings.HasPrefix(layout.Root, repo+string(filepath.Separator)) {
		t.Fatalf("central storage points into repository: %#v", layout)
	}
}

func TestResolveRepoStorageRequiresExplicitModeAndStaysInsideRoot(t *testing.T) {
	repo := t.TempDir()
	repo, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	layout, err := Resolve(t.TempDir(), repo, "p_abc", cfg, "repo")
	if err != nil {
		t.Fatal(err)
	}
	if layout.Root != filepath.Join(repo, ".herdr", "memory") || layout.Mode != "repo" {
		t.Fatalf("Resolve() = %#v", layout)
	}
	cfg.Storage.RepoDirectory = "../outside"
	if _, err := Resolve(t.TempDir(), repo, "p_abc", cfg, "repo"); err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("Resolve() traversal error = %v", err)
	}
}

func TestResolveRepoStorageRejectsSymlinkEscape(t *testing.T) {
	repo := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(repo, ".herdr")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := Resolve(t.TempDir(), repo, "p_abc", config.Default(), "repo"); err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("Resolve() symlink escape error = %v", err)
	}
}

func TestInitializeCreatesScaffoldAndPreservesNonEmptyNow(t *testing.T) {
	layout, err := Resolve(t.TempDir(), t.TempDir(), "p_abc", config.Default(), "central")
	if err != nil {
		t.Fatal(err)
	}
	if err := Initialize(layout); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	data, err := os.ReadFile(layout.Now)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "# Now") || !strings.Contains(string(data), "## Next steps") {
		t.Fatalf("Initialize() now.md = %q", data)
	}
	custom := []byte("# My current work\n")
	if err := os.WriteFile(layout.Now, custom, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Initialize(layout); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(layout.Now)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(custom) {
		t.Fatalf("Initialize() overwrote now.md: %q", data)
	}
}
