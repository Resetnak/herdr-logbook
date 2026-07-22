package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadNotesOrdersNowFirstAndClassifiesScopes(t *testing.T) {
	projectRoot := t.TempDir()
	globalRoot := t.TempDir()
	writeNote(t, filepath.Join(projectRoot, "notes", "zeta.md"), "# Zeta\n")
	writeNote(t, filepath.Join(projectRoot, "now.md"), "# Now\n")
	writeNote(t, filepath.Join(projectRoot, "inbox", "2026-07.md"), "# Inbox\n")
	writeNote(t, filepath.Join(globalRoot, "notes", "global.md"), "# Global\n")
	writeNote(t, filepath.Join(projectRoot, "notes", "ignore.txt"), "# Ignore\n")

	notes, err := LoadNotes(projectRoot, globalRoot, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 4 || notes[0].Type != NoteNow || notes[0].Title != "Now" {
		t.Fatalf("LoadNotes() = %#v", notes)
	}
	if notes[1].Type != NoteProjectInbox || notes[3].Type != NoteGlobalNote {
		t.Fatalf("LoadNotes() classification = %#v", notes)
	}
}

func TestLoadNotesSkipsHiddenDirectoriesAndSymlinks(t *testing.T) {
	root := t.TempDir()
	writeNote(t, filepath.Join(root, ".hidden", "secret.md"), "# Secret\n")
	target := filepath.Join(t.TempDir(), "outside.md")
	writeNote(t, target, "# Outside\n")
	if err := os.Symlink(target, filepath.Join(root, "linked.md")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	notes, err := LoadNotes(root, "", 1024)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 0 {
		t.Fatalf("LoadNotes() followed hidden/symlink entries: %#v", notes)
	}
}

func writeNote(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
