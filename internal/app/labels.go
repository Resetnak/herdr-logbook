package app

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"

	"github.com/Resetnak/herdr-logbook/internal/nowfile"
)

func listTitle(note Note) string {
	switch note.Type {
	case NoteNow:
		if task := nowfile.CurrentTask(note.Content); task != "" {
			return firstLine(task)
		}
		return "Current task"
	case NoteProjectInbox, NoteGlobalInbox:
		return journalListTitle(note)
	default:
		title := strings.TrimSpace(note.Title)
		if note.Type == NoteDecision || note.Type == NoteGlobalDecision {
			// The template H1 is "# Decision: X" so the file reads well on its
			// own; lists already label the kind, so the prefix is noise there.
			title = strings.TrimSpace(strings.TrimPrefix(title, "Decision:"))
		}
		if title != "" {
			return title
		}
		return strings.TrimSuffix(filepath.Base(note.Path), filepath.Ext(note.Path))
	}
}

func listLine(note Note, showKind bool) string {
	title := listTitle(note)
	var extra []string
	if showKind {
		extra = append(extra, kindLabel(note.Type))
	}
	if name := strings.TrimSpace(note.ProjectName); name != "" {
		extra = append(extra, name)
	}
	// All notes flattens every scope and date, so the line carries the date;
	// scoped lists stay short.
	if showKind && !note.Modified.IsZero() {
		extra = append(extra, note.Modified.Format("2006-01-02"))
	}
	if len(extra) == 0 {
		return title
	}
	return title + " · " + strings.Join(extra, " · ")
}

func kindLabel(noteType NoteType) string {
	switch noteType {
	case NoteNow:
		return "current task"
	case NoteProjectInbox:
		return "project journal"
	case NoteGlobalInbox:
		return "global journal"
	case NoteProjectNote:
		return "note"
	case NoteDecision:
		return "decision"
	case NoteGlobalNote:
		return "global note"
	case NoteGlobalDecision:
		return "global decision"
	default:
		return string(noteType)
	}
}

func journalListTitle(note Note) string {
	month := inboxMonth(note)
	if month == "" {
		return "Journal"
	}
	parsed, err := time.Parse("2006-01", month)
	if err != nil {
		return "Journal · " + month
	}
	return parsed.Format("January 2006") + " journal"
}

func inboxMonth(note Note) string {
	if month := lastMonthToken(note.Title); month != "" {
		return month
	}
	base := strings.TrimSuffix(filepath.Base(note.Path), filepath.Ext(note.Path))
	if _, err := time.Parse("2006-01", base); err == nil {
		return base
	}
	return ""
}

func lastMonthToken(title string) string {
	fields := strings.Fields(title)
	if len(fields) == 0 {
		return ""
	}
	candidate := fields[len(fields)-1]
	if _, err := time.Parse("2006-01", candidate); err == nil {
		return candidate
	}
	return ""
}

func firstLine(text string) string {
	for line := range strings.SplitSeq(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		return truncateRunes(line, 80)
	}
	return ""
}

func truncateRunes(text string, limit int) string {
	if utf8.RuneCountInString(text) <= limit {
		return text
	}
	return string([]rune(text)[:limit])
}

// truncateListLabel fits one notes-list row into the pane so a long title
// cannot wrap and break the column alignment. width is the full pane width;
// border, padding, and the selection prefix eat six cells of it.
func truncateListLabel(label string, width int) string {
	limit := width - 6
	if limit < 8 {
		return label
	}
	return ansi.Truncate(label, limit, "…")
}

// CanDelete reports whether the Hub may offer a confirmed removal of note.
// now.md is the current-task singleton; replace it with t instead of unlinking it.
// The path is checked as well as Type: search results historically arrive as
// generic notes, and a Type-only guard would let x unlink now.md.
func CanDelete(note Note) error {
	if note.Type == NoteNow || isNowFile(note.Path) {
		return fmt.Errorf("the current task cannot be deleted; press t to replace it")
	}
	if strings.TrimSpace(note.Path) == "" {
		return fmt.Errorf("note path is empty")
	}
	return nil
}

func isNowFile(path string) bool {
	return strings.EqualFold(filepath.Base(path), "now.md")
}

func isJournalFile(path string) bool {
	slash := filepath.ToSlash(path)
	return strings.Contains(slash, "/inbox/") || strings.HasPrefix(slash, "inbox/")
}
