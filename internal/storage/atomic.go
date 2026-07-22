package storage

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func AtomicWrite(path string, data []byte, perm fs.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create destination directory %q: %w", dir, err)
	}
	if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
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
