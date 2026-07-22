package project

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveUsesContextPriority(t *testing.T) {
	base := t.TempDir()
	dirs := map[string]string{}
	for _, name := range []string{"explicit", "worktree", "focused", "workspace", "fallback"} {
		dirs[name] = filepath.Join(base, name)
		if err := os.Mkdir(dirs[name], 0o755); err != nil {
			t.Fatal(err)
		}
	}

	got, err := Resolve(ResolveOptions{
		ExplicitRoot: dirs["explicit"], WorktreePath: dirs["worktree"],
		FocusedPaneCWD: dirs["focused"], WorkspaceCWD: dirs["workspace"], CWD: dirs["fallback"],
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got.Root != canonicalForTest(t, dirs["explicit"]) {
		t.Fatalf("Resolve() root = %q, want explicit", got.Root)
	}
}

func TestResolveNonGitUnicodeProject(t *testing.T) {
	root := filepath.Join(t.TempDir(), "projekt žluťoučký kůň")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Resolve(ResolveOptions{CWD: root})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	root = canonicalForTest(t, root)
	if got.Root != root || got.Name != "projekt žluťoučký kůň" || !strings.HasPrefix(got.ID, "p_") {
		t.Fatalf("Resolve() = %#v", got)
	}
	if got.Fingerprint != "path:"+root {
		t.Fatalf("Resolve() fingerprint = %q", got.Fingerprint)
	}
}

func TestResolveGitRemoteAndBranch(t *testing.T) {
	repo := newGitRepo(t, "repo with spaces")
	runGit(t, repo, "remote", "add", "origin", "https://user:secret@GitHub.COM/Org/Repo.git?token=secret")

	got, err := Resolve(ResolveOptions{CWD: filepath.Join(repo, "subdir")})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got.Root != repo || got.Fingerprint != "github.com/Org/Repo" || got.Branch == "" {
		t.Fatalf("Resolve() = %#v", got)
	}
	if strings.Contains(got.Fingerprint, "secret") || strings.Contains(got.Fingerprint, "user") {
		t.Fatalf("Resolve() leaked credentials: %#v", got)
	}
}

func TestResolveWorktreesShareIdentityWithoutRemote(t *testing.T) {
	repo := newGitRepo(t, "main")
	worktree := filepath.Join(t.TempDir(), "worktree")
	runGit(t, repo, "worktree", "add", "-b", "feature-test", worktree)

	mainProject, err := Resolve(ResolveOptions{CWD: repo})
	if err != nil {
		t.Fatal(err)
	}
	worktreeProject, err := Resolve(ResolveOptions{CWD: worktree})
	if err != nil {
		t.Fatal(err)
	}
	if mainProject.ID != worktreeProject.ID || !strings.HasPrefix(mainProject.Fingerprint, "git-common:") {
		t.Fatalf("worktree identities differ: %#v %#v", mainProject, worktreeProject)
	}
}

func TestResolveRejectsOverrideEscapingRepository(t *testing.T) {
	repo := newGitRepo(t, "repo")
	if err := os.WriteFile(filepath.Join(repo, ".herdr-memory.toml"), []byte("root = \"../outside\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(ResolveOptions{CWD: repo}); err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("Resolve() error = %v", err)
	}
}

func TestResolveRejectsOverrideSymlinkEscapingRepository(t *testing.T) {
	repo := newGitRepo(t, "repo")
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(repo, "linked")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".herdr-memory.toml"), []byte("root = \"linked\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(ResolveOptions{CWD: repo}); err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("Resolve() symlink escape error = %v", err)
	}
}

func newGitRepo(t *testing.T, name string) string {
	t.Helper()
	repo := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(filepath.Join(repo, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "README.md")
	runGit(t, repo, "commit", "-m", "initial")
	return canonicalForTest(t, repo)
}

func canonicalForTest(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}
