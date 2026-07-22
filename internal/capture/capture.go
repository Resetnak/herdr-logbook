package capture

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Resetnak/herdr-logbook/internal/storage"
)

type Request struct {
	InboxDir    string
	LockPath    string
	Entry       Entry
	MaxBytes    int64
	LockTimeout time.Duration
}

func Append(request Request) (string, error) {
	if request.MaxBytes <= 0 {
		return "", fmt.Errorf("capture byte limit must be positive")
	}
	if int64(len(request.Entry.Text)) > request.MaxBytes {
		return "", fmt.Errorf("capture size %d exceeds limit %d bytes", len(request.Entry.Text), request.MaxBytes)
	}
	if !utf8.ValidString(request.Entry.Text) || strings.ContainsRune(request.Entry.Text, '\x00') {
		return "", fmt.Errorf("capture text must be valid UTF-8 without NUL bytes")
	}
	formatted, err := Format(request.Entry)
	if err != nil {
		return "", err
	}
	if request.LockTimeout <= 0 {
		request.LockTimeout = 2 * time.Second
	}
	path := filepath.Join(request.InboxDir, request.Entry.Time.Format("2006-01")+".md")
	err = storage.WithLock(request.LockPath, request.LockTimeout, func() error {
		existing, err := os.ReadFile(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("read inbox %q: %w", path, err)
		}
		if errors.Is(err, os.ErrNotExist) {
			existing = []byte("# Inbox — " + request.Entry.Time.Format("2006-01") + "\n\n")
		} else if len(existing) > 0 && !strings.HasSuffix(string(existing), "\n\n") {
			if strings.HasSuffix(string(existing), "\n") {
				existing = append(existing, '\n')
			} else {
				existing = append(existing, '\n', '\n')
			}
		}
		return storage.AtomicWrite(path, append(existing, formatted...), 0o600)
	})
	if err != nil {
		return "", err
	}
	return path, nil
}
