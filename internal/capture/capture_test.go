package capture

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAppendWritesMonthlyInbox(t *testing.T) {
	dir := t.TempDir()
	when := time.Date(2026, 7, 22, 14, 32, 0, 0, time.UTC)
	path, err := Append(Request{
		InboxDir: dir, LockPath: filepath.Join(t.TempDir(), "capture.lock"),
		Entry: Entry{Time: when, Text: "First note"}, MaxBytes: 1024, LockTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if path != filepath.Join(dir, "2026-07.md") {
		t.Fatalf("Append() path = %q", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "# Inbox — 2026-07\n\n") || !strings.Contains(string(data), "First note") {
		t.Fatalf("Append() data = %q", data)
	}
}

func TestAppendRejectsOversizedContentWithoutCreatingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := Append(Request{
		InboxDir: dir, LockPath: filepath.Join(t.TempDir(), "capture.lock"),
		Entry: Entry{Time: time.Now(), Text: "12345"}, MaxBytes: 4, LockTimeout: time.Second,
	})
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("Append() size error = %v", err)
	}
	entries, readErr := os.ReadDir(dir)
	if readErr != nil || len(entries) != 0 {
		t.Fatalf("Append() created files: %#v, %v", entries, readErr)
	}
}

func TestAppendConcurrentCapturesArePreserved(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(t.TempDir(), "capture.lock")
	when := time.Date(2026, 7, 22, 14, 32, 0, 0, time.UTC)
	var wg sync.WaitGroup
	for i := range 12 {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := Append(Request{
				InboxDir: dir, LockPath: lockPath,
				Entry:    Entry{Time: when, Text: fmt.Sprintf("note-%02d", i)},
				MaxBytes: 1024, LockTimeout: 2 * time.Second,
			})
			if err != nil {
				t.Errorf("Append(%d) error = %v", i, err)
			}
		}()
	}
	wg.Wait()
	data, err := os.ReadFile(filepath.Join(dir, "2026-07.md"))
	if err != nil {
		t.Fatal(err)
	}
	for i := range 12 {
		if count := strings.Count(string(data), fmt.Sprintf("note-%02d", i)); count != 1 {
			t.Fatalf("note-%02d count = %d\n%s", i, count, data)
		}
	}
}

func TestAppendValidatesTheRequestBeforeTouchingDisk(t *testing.T) {
	dir := t.TempDir()
	lock := filepath.Join(t.TempDir(), "capture.lock")
	when := time.Date(2026, 7, 22, 14, 32, 0, 0, time.UTC)

	cases := map[string]Request{
		"no byte limit": {InboxDir: dir, LockPath: lock, Entry: Entry{Time: when, Text: "note"}},
		"invalid utf-8": {InboxDir: dir, LockPath: lock, Entry: Entry{Time: when, Text: "note \xff"}, MaxBytes: 1024},
		"nul byte":      {InboxDir: dir, LockPath: lock, Entry: Entry{Time: when, Text: "note\x00"}, MaxBytes: 1024},
		"blank text":    {InboxDir: dir, LockPath: lock, Entry: Entry{Time: when, Text: "   "}, MaxBytes: 1024},
	}
	for name, request := range cases {
		if _, err := Append(request); err == nil {
			t.Fatalf("%s: Append succeeded", name)
		}
	}
	if entries, err := os.ReadDir(dir); err != nil || len(entries) != 0 {
		t.Fatalf("a rejected capture created files: %#v, %v", entries, err)
	}
}

// Notes are rendered straight into a terminal — as list titles without Glamour,
// and through Glamour in the preview. Glamour passes OSC sequences through
// untouched, so an escape that reaches storage reaches the terminal: OSC 52
// rewrites the reader's system clipboard, OSC 0 retitles their window. Reject
// the control characters at the door, where the UTF-8 and NUL checks already are.
func TestAppendRejectsTerminalControlCharacters(t *testing.T) {
	dir := t.TempDir()
	lock := filepath.Join(t.TempDir(), "capture.lock")
	when := time.Date(2026, 7, 22, 14, 32, 0, 0, time.UTC)

	for name, text := range map[string]string{
		"osc 52 clipboard write": "note \x1b]52;c;aGFja2Vk\a",
		"osc 0 window title":     "note \x1b]0;pwned\a",
		"csi erase display":      "note \x1b[2J",
		"bare escape":            "note \x1b",
		"bell":                   "note \a",
		"carriage return":        "note \r overwritten",
		"c1 control":             "note \u009b2J",
	} {
		if _, err := Append(Request{
			InboxDir: dir, LockPath: lock,
			Entry: Entry{Time: when, Text: text}, MaxBytes: 1024, LockTimeout: time.Second,
		}); err == nil {
			t.Fatalf("%s: Append accepted control characters", name)
		}
	}
	if entries, err := os.ReadDir(dir); err != nil || len(entries) != 0 {
		t.Fatalf("a rejected capture created files: %#v, %v", entries, err)
	}
}

// Newlines and tabs are ordinary Markdown, not terminal control: a multi-line
// capture and an indented code block must still go through.
func TestAppendAcceptsNewlinesAndTabs(t *testing.T) {
	dir := t.TempDir()
	when := time.Date(2026, 7, 22, 14, 32, 0, 0, time.UTC)
	path, err := Append(Request{
		InboxDir: dir, LockPath: filepath.Join(t.TempDir(), "capture.lock"),
		Entry:    Entry{Time: when, Text: "first line\nsecond line\n\tindented"},
		MaxBytes: 1024, LockTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "second line\n\tindented") {
		t.Fatalf("Append() dropped whitespace: %q", data)
	}
}

// Append is a read-modify-write, so a symlink planted where the monthly inbox
// goes would copy its target into a note and then replace the link with the
// result. index.Scan and app.LoadNotes already refuse symlinks on the way in;
// the write path has to agree.
func TestAppendRefusesToWriteThroughASymlink(t *testing.T) {
	dir := t.TempDir()
	secret := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(secret, []byte("root:x:0:0:secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	when := time.Date(2026, 7, 22, 14, 32, 0, 0, time.UTC)
	if err := os.Symlink(secret, filepath.Join(dir, "2026-07.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	_, err := Append(Request{
		InboxDir: dir, LockPath: filepath.Join(t.TempDir(), "capture.lock"),
		Entry: Entry{Time: when, Text: "note"}, MaxBytes: 1024, LockTimeout: time.Second,
	})
	if err == nil {
		t.Fatal("Append followed a symlink")
	}
	data, readErr := os.ReadFile(secret)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(data) != "root:x:0:0:secret\n" {
		t.Fatalf("the symlink target was modified: %q", data)
	}
}

// The inbox is one Markdown file per month, so appends must keep exactly one blank
// line between entries no matter how the previous write ended.
func TestAppendSeparatesEntriesRegardlessOfTrailingNewlines(t *testing.T) {
	when := time.Date(2026, 7, 22, 14, 32, 0, 0, time.UTC)
	for name, existing := range map[string]string{
		"no trailing newline":  "# Inbox — 2026-07\n\n## earlier\n\nprevious",
		"one trailing newline": "# Inbox — 2026-07\n\n## earlier\n\nprevious\n",
		"blank line already":   "# Inbox — 2026-07\n\n## earlier\n\nprevious\n\n",
	} {
		dir := t.TempDir()
		path := filepath.Join(dir, "2026-07.md")
		if err := os.WriteFile(path, []byte(existing), 0o600); err != nil {
			t.Fatal(err)
		}
		// A zero LockTimeout must fall back to the default rather than fail.
		if _, err := Append(Request{
			InboxDir: dir, LockPath: filepath.Join(t.TempDir(), "capture.lock"),
			Entry: Entry{Time: when, Text: "later"}, MaxBytes: 1024,
		}); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "previous\n\n\n") || strings.Contains(string(data), "previous##") {
			t.Fatalf("%s: entry separation = %q", name, data)
		}
		if !strings.Contains(string(data), "previous") || !strings.Contains(string(data), "later") {
			t.Fatalf("%s: an entry was lost: %q", name, data)
		}
	}
}

// An interrupted write or a stray `touch` can leave a zero-byte inbox behind.
// Appending to it must still produce a file with its month heading.
func TestAppendWritesTheHeaderIntoAnEmptyInbox(t *testing.T) {
	dir := t.TempDir()
	when := time.Date(2026, 7, 22, 14, 32, 0, 0, time.UTC)
	if err := os.WriteFile(filepath.Join(dir, "2026-07.md"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	path, err := Append(Request{
		InboxDir: dir, LockPath: filepath.Join(t.TempDir(), "capture.lock"),
		Entry: Entry{Time: when, Text: "First note"}, MaxBytes: 1024, LockTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "# Inbox — 2026-07\n\n") {
		t.Fatalf("empty inbox did not get its heading: %q", data)
	}
}
