package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

// WithLock runs fn while holding the flock at path, giving up with an error
// after timeout. Locks do not nest: flock is per file descriptor, so taking
// the same path twice in one call chain deadlocks until the timeout.
func WithLock(path string, timeout time.Duration, fn func() error) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create lock directory: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	fileLock := flock.New(path)
	locked, err := fileLock.TryLockContext(ctx, 10*time.Millisecond)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("lock timeout for %q: %w", path, ctx.Err())
		}
		return fmt.Errorf("lock %q: %w", path, err)
	}
	if !locked {
		return fmt.Errorf("lock timeout for %q", path)
	}
	defer func() { _ = fileLock.Unlock() }()
	return fn()
}
