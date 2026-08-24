package app

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
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

func TestListLineAddsModifiedDateInAllNotes(t *testing.T) {
	note := Note{
		Type:     NoteProjectNote,
		Title:    "Collect more metrics",
		Modified: time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC),
	}
	if got := listLine(note, true); got != "Collect more metrics · note · 2026-08-12" {
		t.Fatalf("listLine() with date = %q", got)
	}
	// Scoped lists stay date-free; the scope already narrows the context.
	if got := listLine(note, false); got != "Collect more metrics" {
		t.Fatalf("listLine() outside All notes = %q", got)
	}
}

func TestKindLabelDescribesEveryNoteType(t *testing.T) {
	tests := []struct {
		noteType NoteType
		want     string
	}{
		{NoteNow, "current task"},
		{NoteProjectInbox, "project journal"},
		{NoteGlobalInbox, "global journal"},
		{NoteProjectNote, "note"},
		{NoteDecision, "decision"},
		{NoteGlobalNote, "global note"},
		{NoteGlobalDecision, "global decision"},
		{NoteType("future-type"), "future-type"},
	}
	for _, test := range tests {
		if got := kindLabel(test.noteType); got != test.want {
			t.Errorf("kindLabel(%q) = %q, want %q", test.noteType, got, test.want)
		}
	}
}

func TestListTitleFallsBackToSafeReadableLabels(t *testing.T) {
	if got := listTitle(Note{Type: NoteProjectNote, Path: "/store/notes/cache-policy.md"}); got != "cache-policy" {
		t.Fatalf("untitled note = %q", got)
	}
	if got := listTitle(Note{Type: NoteProjectInbox, Path: "/store/inbox/2026-07.md"}); got != "July 2026 journal" {
		t.Fatalf("journal path month = %q", got)
	}
	if got := listTitle(Note{Type: NoteProjectInbox, Path: "/store/inbox/unknown.md"}); got != "Journal" {
		t.Fatalf("journal without month = %q", got)
	}

	task := strings.Repeat("ž", 81)
	got := listTitle(Note{Type: NoteNow, Content: "# Now\n\n## Current task\n\n" + task + "\n"})
	if utf8.RuneCountInString(got) != 80 {
		t.Fatalf("long current task has %d runes, want 80", utf8.RuneCountInString(got))
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
