package app

import (
	"os"
	"path/filepath"
	"strings"
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

func TestLoadNotesClassifiesEveryScopeDirectory(t *testing.T) {
	projectRoot := t.TempDir()
	globalRoot := t.TempDir()
	writeNote(t, filepath.Join(projectRoot, "decisions", "2026-07-22-use-redis.md"), "# Decision\n")
	// now.md is only special in a project store; in the global store it is a note.
	writeNote(t, filepath.Join(globalRoot, "now.md"), "# Global Now\n")
	writeNote(t, filepath.Join(globalRoot, "inbox", "2026-07.md"), "# Global Inbox\n")
	writeNote(t, filepath.Join(globalRoot, "decisions", "2026-07-22-global.md"), "# Global Decision\n")

	notes, err := LoadNotes(projectRoot, globalRoot, 1024)
	if err != nil {
		t.Fatal(err)
	}
	got := map[NoteType]bool{}
	for _, note := range notes {
		got[note.Type] = true
	}
	for _, want := range []NoteType{NoteDecision, NoteGlobalNote, NoteGlobalInbox, NoteGlobalDecision} {
		if !got[want] {
			t.Fatalf("note type %q was never produced: %#v", want, notes)
		}
	}
}

func TestLoadNotesMarksOversizeNotesAndRejectsANonPositiveLimit(t *testing.T) {
	root := t.TempDir()
	writeNote(t, filepath.Join(root, "notes", "huge.md"), strings.Repeat("x", 64))

	notes, err := LoadNotes(root, "", 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 || !notes[0].TooLarge || notes[0].Content != "Note is too large to preview." {
		t.Fatalf("oversize note = %#v", notes)
	}

	if _, err := LoadNotes(root, "", 0); err == nil {
		t.Fatal("LoadNotes accepted a non-positive preview limit")
	}
}

func TestLoadNotesTreatsAMissingRootAsEmpty(t *testing.T) {
	notes, err := LoadNotes(filepath.Join(t.TempDir(), "never-created"), "", 1024)
	if err != nil || len(notes) != 0 {
		t.Fatalf("LoadNotes on a missing root = %#v, %v", notes, err)
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
