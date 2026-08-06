package storage

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ReadForRewrite reads a file that is about to be rewritten in place, refusing
// to follow a symlink. A read-modify-write through one would copy the target's
// contents into a note and then replace the link with the result; index.Scan and
// app.LoadNotes already skip symlinks when reading, and the write paths agree.
// A missing file is not an error — the caller decides what an absent file means.
func ReadForRewrite(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("inspect %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%q is not a regular file", path)
	}
	return os.ReadFile(path)
}

// AtomicWrite replaces path via a same-directory temp file, fsync, and rename,
// so readers see either the old content or the new one, never a torn write.
func AtomicWrite(path string, data []byte, perm fs.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create destination directory %q: %w", dir, err)
	}
	// Lstat, not Stat: permissions are inherited from the file being replaced,
	// and a symlink must not donate its target's mode.
	if info, err := os.Lstat(path); err == nil && info.Mode().IsRegular() {
		perm = info.Mode().Perm()
	}

	temp, err := os.CreateTemp(dir, ".herdr-logbook-*")
	if err != nil {
		return fmt.Errorf("create temporary file for %q: %w", path, err)
	}
	tempPath := temp.Name()
	defer func() { _ = os.Remove(tempPath) }()

	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write temporary file for %q: %w", path, err)
	}
	if err := temp.Chmod(perm); err != nil {
		_ = temp.Close()
		return fmt.Errorf("set permissions for %q: %w", path, err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("flush temporary file for %q: %w", path, err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary file for %q: %w", path, err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace %q atomically: %w", path, err)
	}
	return nil
}
