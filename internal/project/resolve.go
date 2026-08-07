package project

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	md "github.com/Resetnak/herdr-logbook/internal/markdown"
	"github.com/pelletier/go-toml/v2"
)

// CommandRunner executes git in a directory; tests inject a fake to resolve
// projects without a git binary.
type CommandRunner func(context.Context, string, ...string) (string, error)

// ResolveOptions carries the location candidates in priority order: explicit
// --project-root, Herdr worktree, focused pane cwd, workspace cwd, then CWD.
type ResolveOptions struct {
	ExplicitRoot   string
	WorktreePath   string
	FocusedPaneCWD string
	WorkspaceCWD   string
	CWD            string
	Runner         CommandRunner
}

// Project is a resolved identity: a stable ID derived from the credential-free
// remote fingerprint (or git common dir, or path), plus display metadata.
type Project struct {
	ID          string   `toml:"id" json:"id"`
	Name        string   `toml:"name" json:"name"`
	Root        string   `toml:"root" json:"root"`
	Roots       []string `toml:"roots" json:"roots"`
	Branch      string   `toml:"-" json:"branch,omitempty"`
	Fingerprint string   `toml:"-" json:"remote_fingerprint,omitempty"`
}

// overrideConfig is the repository-side .herdr-logbook.toml. It arrives with
// `git clone`, so every field here is attacker-controlled and must only be able
// to affect naming and layout *inside* the repository.
//
// There is deliberately no storage field: honouring one would let a repository
// redirect the user's private working memory into its own worktree, where the
// next `git add -A && git push` publishes it. Repo-local storage stays an
// explicit opt-in through the user's own config or --storage. A stale storage
// key in a repository is ignored rather than rejected, so existing checkouts
// keep resolving.
type overrideConfig struct {
	ProjectID   string `toml:"project_id"`
	DisplayName string `toml:"display_name"`
	Root        string `toml:"root"`
}

// Resolve picks the project the current invocation belongs to. Git worktrees
// of one repository share identity; a repository's .herdr-logbook.toml may
// adjust name and root but never where notes are stored.
func Resolve(options ResolveOptions) (Project, error) {
	selected := cmp.Or(options.ExplicitRoot, options.WorktreePath, options.FocusedPaneCWD, options.WorkspaceCWD, options.CWD)
	if selected == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return Project{}, fmt.Errorf("resolve current directory: %w", err)
		}
		selected = cwd
	}
	selected, err := canonicalDirectory(selected)
	if err != nil {
		return Project{}, err
	}

	runner := options.Runner
	if runner == nil {
		runner = defaultGitRunner
	}
	gitRoot := ""
	if root, err := gitValue(runner, selected, "rev-parse", "--show-toplevel"); err == nil {
		gitRoot, err = canonicalDirectory(root)
		if err != nil {
			return Project{}, err
		}
	} else {
		gitRoot = findGitRoot(selected)
	}

	root := selected
	if gitRoot != "" {
		root = gitRoot
	}
	override, err := loadOverride(root)
	if err != nil {
		return Project{}, err
	}
	if override.Root != "" {
		root, err = containedDirectory(root, override.Root)
		if err != nil {
			return Project{}, err
		}
	}

	project := Project{Root: root, Roots: []string{root}, Name: filepath.Base(root)}
	if name := displayName(override.DisplayName); name != "" {
		project.Name = name
	}

	identity := "path:" + root
	if override.ProjectID != "" {
		identity = "override:" + override.ProjectID
	} else if gitRoot != "" {
		// ponytail: a remote we cannot fingerprint (a local path like /srv/git/foo.git,
		// or any URL without a host) is not fatal — drop to the git-common identity
		// below rather than refusing to resolve the project at all.
		fingerprint := ""
		if remote, err := gitValue(runner, gitRoot, "config", "--get", "remote.origin.url"); err == nil && remote != "" {
			fingerprint, _ = SanitizeRemote(remote)
		}
		if fingerprint != "" {
			identity = fingerprint
			project.Fingerprint = identity
		} else if commonDir, err := gitValue(runner, gitRoot, "rev-parse", "--git-common-dir"); err == nil {
			if !filepath.IsAbs(commonDir) {
				commonDir = filepath.Join(gitRoot, commonDir)
			}
			commonDir, err = canonicalPath(commonDir)
			if err != nil {
				return Project{}, err
			}
			identity = "git-common:" + commonDir
			project.Fingerprint = identity
		}
	}
	if project.Fingerprint == "" {
		project.Fingerprint = identity
	}
	project.ID = StableID(identity)
	if gitRoot != "" {
		project.Branch, _ = gitValue(runner, gitRoot, "rev-parse", "--abbrev-ref", "HEAD")
		// A detached HEAD answers the literal "HEAD", which is not a branch.
		if project.Branch == "HEAD" {
			project.Branch = ""
		}
	}
	return project, nil
}

func defaultGitRunner(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	output, err := cmd.Output()
	return strings.TrimSpace(string(output)), err
}

func gitValue(runner CommandRunner, dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return runner(ctx, dir, args...)
}

func canonicalDirectory(path string) (string, error) {
	path, err := canonicalPath(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("inspect project directory %q: %w", path, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("project path %q is not a directory", path)
	}
	return path, nil
}

func canonicalPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("normalize path %q: %w", path, err)
	}
	if evaluated, err := filepath.EvalSymlinks(abs); err == nil {
		abs = evaluated
	}
	return filepath.Clean(abs), nil
}

func findGitRoot(start string) string {
	for current := start; ; current = filepath.Dir(current) {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
	}
}

func loadOverride(gitRoot string) (overrideConfig, error) {
	path := filepath.Join(gitRoot, ".herdr-logbook.toml")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return overrideConfig{}, nil
	}
	if err != nil {
		return overrideConfig{}, fmt.Errorf("read project override %q: %w", path, err)
	}
	var override overrideConfig
	if err := toml.Unmarshal(data, &override); err != nil {
		return overrideConfig{}, fmt.Errorf("decode project override %q: %w", path, err)
	}
	return override, nil
}

// maxDisplayNameRunes is what the Hub status bar can carry without pushing the
// branch and store hints off the line.
const maxDisplayNameRunes = 120

// displayName makes the repository-supplied project name safe to print. It is
// the one override field used verbatim — the status bar paints it on every
// frame and author.CreateDecision writes it into the body of every decision —
// so it gets the same treatment as captured text. An empty result means the
// caller keeps the directory name.
func displayName(raw string) string {
	name := strings.TrimSpace(md.StripTerminalControl(raw))
	if runes := []rune(name); len(runes) > maxDisplayNameRunes {
		name = strings.TrimSpace(string(runes[:maxDisplayNameRunes]))
	}
	return name
}

func containedDirectory(base, relative string) (string, error) {
	if filepath.IsAbs(relative) {
		return "", fmt.Errorf("project override root %q escapes repository boundary", relative)
	}
	candidate := filepath.Clean(filepath.Join(base, relative))
	rel, err := filepath.Rel(base, candidate)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("project override root %q escapes repository boundary", relative)
	}
	candidate, err = canonicalDirectory(candidate)
	if err != nil {
		return "", err
	}
	rel, err = filepath.Rel(base, candidate)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("project override root %q escapes repository boundary", relative)
	}
	return candidate, nil
}
