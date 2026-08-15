package app

import (
	"strings"
	"testing"
)

func TestListTitlePrefersCurrentTaskBody(t *testing.T) {
	note := Note{
		Type:    NoteNow,
		Title:   "Now",
		Content: "# Now\n\n## Current task\n\nRotate the signing tokens\n\n## Next steps\n",
	}
	if got := listTitle(note); got != "Rotate the signing tokens" {
		t.Fatalf("listTitle() = %q", got)
	}
}

func TestListTitleUsesJournalMonthNotInboxH1(t *testing.T) {
	note := Note{
		Type:  NoteProjectInbox,
		Title: "Inbox — 2026-08",
		Path:  "/store/inbox/2026-08.md",
	}
	if got := listTitle(note); got != "August 2026 journal" {
		t.Fatalf("listTitle() = %q", got)
	}
}

func TestListLineAddsKindInAllNotes(t *testing.T) {
	note := Note{Type: NoteProjectNote, Title: "Collect more metrics"}
	if got := listLine(note, true); got != "Collect more metrics · note" {
		t.Fatalf("listLine() = %q", got)
	}
	if got := listLine(note, false); got != "Collect more metrics" {
		t.Fatalf("listLine() without kind = %q", got)
	}
	note.ProjectName = "api"
	if got := listLine(note, false); got != "Collect more metrics · api" {
		t.Fatalf("listLine() with project = %q", got)
	}
}

func TestCanDeleteRejectsNowAndEmptyPath(t *testing.T) {
	if err := CanDelete(Note{Type: NoteNow, Path: "/store/now.md"}); err == nil || !strings.Contains(err.Error(), "cannot be deleted") {
		t.Fatalf("CanDelete(now) = %v", err)
	}
	if err := CanDelete(Note{Type: NoteProjectNote}); err == nil || !strings.Contains(err.Error(), "path is empty") {
		t.Fatalf("CanDelete(empty path) = %v", err)
	}
	if err := CanDelete(Note{Type: NoteProjectNote, Path: "/store/notes/a.md"}); err != nil {
		t.Fatalf("CanDelete(note) = %v", err)
	}
	// Search used to stamp every hit as a project note. Deleting now.md that way
	// would remove the current-task file.
	if err := CanDelete(Note{Type: NoteProjectNote, Path: "/store/now.md"}); err == nil {
		t.Fatal("CanDelete allowed now.md when the type was a generic note")
	}
}
