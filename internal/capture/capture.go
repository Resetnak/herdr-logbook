package capture

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	md "github.com/Resetnak/herdr-logbook/internal/markdown"
	"github.com/Resetnak/herdr-logbook/internal/storage"
)

type Request struct {
	InboxDir    string
	LockPath    string
	Entry       Entry
	MaxBytes    int64
	LockTimeout time.Duration
}

// Append validates the entry and appends it to the monthly inbox under the
// store lock.
func Append(request Request) (string, error) {
	formatted, path, err := prepare(&request)
	if err != nil {
		return "", err
	}
	if request.LockTimeout <= 0 {
		request.LockTimeout = 2 * time.Second
	}
	err = storage.WithLock(request.LockPath, request.LockTimeout, func() error {
		return appendToInbox(path, request.Entry.Time, formatted)
	})
	if err != nil {
		return "", err
	}
	return path, nil
}

// AppendLocked is Append for a caller that already holds the store lock —
// flock is per file descriptor, so taking it again would deadlock. Never call
// it without the lock held: the read-append-write below would race.
func AppendLocked(request Request) (string, error) {
	formatted, path, err := prepare(&request)
	if err != nil {
		return "", err
	}
	if err := appendToInbox(path, request.Entry.Time, formatted); err != nil {
		return "", err
	}
	return path, nil
}

func prepare(request *Request) (formatted []byte, path string, err error) {
	if request.MaxBytes <= 0 {
		return nil, "", fmt.Errorf("capture byte limit must be positive")
	}
	if int64(len(request.Entry.Text)) > request.MaxBytes {
		return nil, "", fmt.Errorf("capture size %d exceeds limit %d bytes", len(request.Entry.Text), request.MaxBytes)
	}
	if !utf8.ValidString(request.Entry.Text) || strings.ContainsRune(request.Entry.Text, '\x00') {
		return nil, "", fmt.Errorf("capture text must be valid UTF-8 without NUL bytes")
	}
	// CRLF is how a Windows pipe delivers ordinary text, so normalise it before
	// the control-character check rather than rejecting the whole capture.
	request.Entry.Text = strings.ReplaceAll(request.Entry.Text, "\r\n", "\n")
	if strings.ContainsFunc(request.Entry.Text, md.IsTerminalControl) {
		return nil, "", fmt.Errorf("capture text must not contain terminal control characters")
	}
	text, err := Format(request.Entry)
	if err != nil {
		return nil, "", err
	}
	return []byte(text), filepath.Join(request.InboxDir, request.Entry.Time.Format("2006-01")+".md"), nil
}

func appendToInbox(path string, entryTime time.Time, formatted []byte) error {
	existing, err := storage.ReadForRewrite(path)
	if err != nil {
		return fmt.Errorf("read inbox %q: %w", path, err)
	}
	// A zero-byte file is left behind by an interrupted write or a stray touch;
	// treat it like a missing one so the month heading is not lost.
	if len(existing) == 0 {
		existing = []byte("# Inbox — " + entryTime.Format("2006-01") + "\n\n")
	} else if !bytes.HasSuffix(existing, []byte("\n\n")) {
		if bytes.HasSuffix(existing, []byte("\n")) {
			existing = append(existing, '\n')
		} else {
			existing = append(existing, '\n', '\n')
		}
	}
	return storage.AtomicWrite(path, append(existing, formatted...), 0o600)
}
