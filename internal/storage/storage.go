package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Resetnak/herdr-logbook/internal/config"
)

const nowTemplate = `# Now

## Current task

Describe the task currently in progress.

## Next steps

- [ ] Add the next concrete action.

## Blockers

- None.

## Context

Record details that will help you resume later.
`

type Layout struct {
	Mode        string `json:"mode"`
	Root        string `json:"root"`
	Now         string `json:"now"`
	Inbox       string `json:"inbox"`
	Notes       string `json:"notes"`
	Decisions   string `json:"decisions"`
	ProjectFile string `json:"project_file,omitempty"`
	Lock        string `json:"lock"`
}

func Resolve(stateDir, projectRoot, projectID string, cfg config.Config, mode string) (Layout, error) {
	if mode == "" {
		mode = cfg.Storage.ProjectMode
	}
	if mode != "central" && mode != "repo" {
		return Layout{}, fmt.Errorf("storage mode must be central or repo")
	}
	if stateDir == "" {
		return Layout{}, fmt.Errorf("HERDR_PLUGIN_STATE_DIR is required")
	}
	if projectID == "" || strings.ContainsAny(projectID, `/\\`) {
		return Layout{}, fmt.Errorf("invalid project ID %q", projectID)
	}

	root := filepath.Join(stateDir, "store", "projects", projectID)
	if mode == "repo" {
		var err error
		root, err = containedPath(projectRoot, cfg.Storage.RepoDirectory)
		if err != nil {
			return Layout{}, err
		}
	}
	layout := Layout{
		Mode: mode, Root: root,
		Now: filepath.Join(root, "now.md"), Inbox: filepath.Join(root, "inbox"),
		Notes: filepath.Join(root, "notes"), Decisions: filepath.Join(root, "decisions"),
		Lock: filepath.Join(stateDir, "locks", projectID+".lock"),
	}
	if mode == "central" {
		layout.ProjectFile = filepath.Join(root, "project.toml")
	}
	return layout, nil
}

func Initialize(layout Layout) error {
	for _, dir := range []string{layout.Root, layout.Inbox, layout.Notes, layout.Decisions} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create memory directory %q: %w", dir, err)
		}
	}
	info, err := os.Stat(layout.Now)
	if err == nil && info.Size() > 0 {
		return nil
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect now.md: %w", err)
	}
	return AtomicWrite(layout.Now, []byte(nowTemplate), 0o600)
}

func containedPath(base, relative string) (string, error) {
	if base == "" || relative == "" || filepath.IsAbs(relative) || os.IsPathSeparator(relative[0]) {
		return "", fmt.Errorf("repository storage path %q escapes project root", relative)
	}
	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	if evaluated, evalErr := filepath.EvalSymlinks(baseAbs); evalErr == nil {
		baseAbs = evaluated
	}
	candidate := filepath.Clean(filepath.Join(baseAbs, relative))
	rel, err := filepath.Rel(baseAbs, candidate)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("repository storage path %q escapes project root", relative)
	}
	candidate, err = resolveExistingPrefix(candidate)
	if err != nil {
		return "", fmt.Errorf("resolve repository storage path: %w", err)
	}
	rel, err = filepath.Rel(baseAbs, candidate)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("repository storage path %q escapes project root", relative)
	}
	return candidate, nil
}

func resolveExistingPrefix(path string) (string, error) {
	current := path
	var missing []string
	for {
		if _, err := os.Lstat(current); err == nil {
			resolved, err := filepath.EvalSymlinks(current)
			if err != nil {
				return "", err
			}
			for i := len(missing) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, missing[i])
			}
			return filepath.Clean(resolved), nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("no existing path prefix for %q", path)
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
}
