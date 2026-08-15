package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RemoveFile unlinks a regular file. Symlinks and directories are refused so a
// confused caller cannot follow a link out of the store or wipe a folder.
func RemoveFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("remove %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%q is not a regular file", path)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove %q: %w", path, err)
	}
	return nil
}

// ContainsPath reports whether path is inside root after cleaning. It does not
// follow the target, so a symlink in the store still counts as inside.
func ContainsPath(root, path string) bool {
	if root == "" || path == "" {
		return false
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, pathAbs)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
