package storage

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

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
	if layout.Root != filepath.Join(repo, ".herdr", "logbook") || layout.Mode != "repo" {
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

func TestResolveValidatesModeStateDirAndProjectID(t *testing.T) {
	cfg := config.Default()
	root := t.TempDir()

	cases := map[string]struct{ stateDir, projectID, mode string }{
		"unknown mode":     {t.TempDir(), "p_abc", "sqlite"},
		"no state dir":     {"", "p_abc", "central"},
		"empty project id": {t.TempDir(), "", "central"},
		"slash in id":      {t.TempDir(), "p/abc", "central"},
	}
	for name, test := range cases {
		if _, err := Resolve(test.stateDir, root, test.projectID, cfg, test.mode); err == nil {
			t.Fatalf("%s: Resolve succeeded", name)
		}
	}
}

func TestResolveRepoStorageRejectsAbsoluteAndEmptyDirectories(t *testing.T) {
	root := t.TempDir()
	for name, dir := range map[string]string{
		"absolute": string(filepath.Separator) + filepath.Join("etc", "logbook"),
		"empty":    "",
	} {
		cfg := config.Default()
		cfg.Storage.RepoDirectory = dir
		if _, err := Resolve(t.TempDir(), root, "p_abc", cfg, "repo"); err == nil {
			t.Fatalf("%s repo directory was accepted", name)
		}
	}
}

func TestAtomicWriteFailsWhenTheDestinationDirectoryCannotExist(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	// A regular file where a directory is needed must surface as an error, not a panic.
	if err := AtomicWrite(filepath.Join(blocker, "nested", "now.md"), []byte("data"), 0o600); err == nil {
		t.Fatal("AtomicWrite succeeded through a regular file")
	}
}

func TestInitializeIsIdempotentAndRefusesAnUnreadableNow(t *testing.T) {
	layout, err := Resolve(t.TempDir(), t.TempDir(), "p_abc", config.Default(), "central")
	if err != nil {
		t.Fatal(err)
	}
	if err := Initialize(layout); err != nil {
		t.Fatal(err)
	}
	if err := Initialize(layout); err != nil {
		t.Fatalf("second Initialize: %v", err)
	}
	// A regular file where the store root belongs must be reported, not ignored.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	blocked := layout
	blocked.Root = filepath.Join(blocker, "store")
	blocked.Inbox = filepath.Join(blocked.Root, "inbox")
	blocked.Notes = filepath.Join(blocked.Root, "notes")
	blocked.Decisions = filepath.Join(blocked.Root, "decisions")
	if err := Initialize(blocked); err == nil {
		t.Fatal("Initialize succeeded with a regular file in place of the store root")
	}

	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		return // an unreadable directory is not enforced here
	}
	// A now.md that cannot even be stat'ed must abort rather than be overwritten:
	// the directory holding it is readable/writable but not traversable.
	sealed := filepath.Join(t.TempDir(), "sealed")
	if err := os.Mkdir(sealed, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(sealed, 0o700) })
	unreadable := layout
	unreadable.Now = filepath.Join(sealed, "now.md")
	if err := Initialize(unreadable); err == nil {
		t.Fatal("Initialize ignored an unreadable now.md")
	}
}

func TestWithLockRefusesAnUncreatableDirectoryAndTimesOut(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WithLock(filepath.Join(blocker, "locks", "store.lock"), time.Second, func() error {
		t.Fatal("callback ran without a lock")
		return nil
	}); err == nil || !strings.Contains(err.Error(), "create lock directory") {
		t.Fatalf("WithLock() directory error = %v", err)
	}

	// flock is per open file description, so a second holder must wait out the timeout.
	path := filepath.Join(t.TempDir(), "store.lock")
	held := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- WithLock(path, 5*time.Second, func() error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	err := WithLock(path, 50*time.Millisecond, func() error {
		t.Error("callback ran while the lock was held")
		return nil
	})
	close(release)
	if holderErr := <-done; holderErr != nil {
		t.Fatalf("lock holder failed: %v", holderErr)
	}
	if err == nil || !strings.Contains(err.Error(), "lock timeout") {
		t.Fatalf("WithLock() timeout error = %v", err)
	}
}

func TestAtomicWriteReportsAFailedTemporaryFile(t *testing.T) {
	dir := t.TempDir()
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("read-only directories are not enforced here")
	}
	if err := os.Chmod(dir, 0o500); err != nil { // readable and traversable, but not writable
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	if err := AtomicWrite(filepath.Join(dir, "now.md"), []byte("data"), 0o600); err == nil ||
		!strings.Contains(err.Error(), "create temporary file") {
		t.Fatalf("AtomicWrite() error = %v", err)
	}
}
