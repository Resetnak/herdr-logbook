package project

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type CommandRunner func(context.Context, string, ...string) (string, error)

type ResolveOptions struct {
	ExplicitRoot   string
	WorktreePath   string
	FocusedPaneCWD string
	WorkspaceCWD   string
	CWD            string
	Runner         CommandRunner
}

type Project struct {
	ID              string   `toml:"id" json:"id"`
	Name            string   `toml:"name" json:"name"`
	Root            string   `toml:"root" json:"root"`
	Roots           []string `toml:"roots" json:"roots"`
	Branch          string   `toml:"-" json:"branch,omitempty"`
	Fingerprint     string   `toml:"-" json:"remote_fingerprint,omitempty"`
	StorageOverride string   `toml:"-" json:"storage_override,omitempty"`
}

type overrideConfig struct {
	ProjectID   string `toml:"project_id"`
	DisplayName string `toml:"display_name"`
	Root        string `toml:"root"`
	Storage     string `toml:"storage"`
}

func Resolve(options ResolveOptions) (Project, error) {
	selected := firstNonEmpty(options.ExplicitRoot, options.WorktreePath, options.FocusedPaneCWD, options.WorkspaceCWD, options.CWD)
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

	project := Project{Root: root, Roots: []string{root}, Name: filepath.Base(root), StorageOverride: override.Storage}
	if override.DisplayName != "" {
		project.Name = override.DisplayName
	}

	identity := "path:" + root
	if override.ProjectID != "" {
		identity = "override:" + override.ProjectID
	} else if gitRoot != "" {
		if remote, err := gitValue(runner, gitRoot, "config", "--get", "remote.origin.url"); err == nil && remote != "" {
			identity, err = SanitizeRemote(remote)
			if err != nil {
				return Project{}, err
			}
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
	path := filepath.Join(gitRoot, ".herdr-memory.toml")
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
	if override.Storage != "" && override.Storage != "central" && override.Storage != "repo" {
		return overrideConfig{}, fmt.Errorf("project override storage must be central or repo")
	}
	return override, nil
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
