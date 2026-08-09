// Package nowfile edits the "## Current task" section of a project's now.md.
// Markdown stays canonical: the file is read and rewritten as text, and every
// section the user wrote by hand is preserved untouched.
package nowfile

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	md "github.com/Resetnak/herdr-logbook/internal/markdown"
)

const (
	heading = "## Current task"
	// placeholder is the body storage.Initialize writes into a fresh now.md. It
	// means "no task set", so reading it back reports an empty task rather than
	// archiving the template text as work that was done.
	placeholder = "Describe the task currently in progress."
	// startedPrefix and startedSuffix delimit the marker SetCurrentTask leaves
	// under the task. An HTML comment stays invisible in rendered Markdown, and
	// CurrentTask strips it so plain readers of the section body never see it.
	startedPrefix = "<!-- herdr-logbook: started "
	startedSuffix = " -->"
)

// CurrentTask returns the body of the "## Current task" section, or an empty
// string when the section is missing, blank, or still holds the template text.
func CurrentTask(content string) string {
	lines := splitLines(content)
	start, end, found := section(lines)
	if !found {
		return ""
	}
	body := make([]string, 0, end-start)
	for _, line := range lines[start:end] {
		if strings.HasPrefix(strings.TrimSpace(line), startedPrefix) {
			continue
		}
		body = append(body, line)
	}
	task := strings.TrimSpace(strings.Join(body, "\n"))
	if task == placeholder {
		return ""
	}
	return task
}

// StartedAt returns when the current task was set, read from the marker
// SetCurrentTask stores in the "## Current task" section. It reports false for
// files written before the marker existed or edited by hand.
func StartedAt(content string) (time.Time, bool) {
	lines := splitLines(content)
	start, end, found := section(lines)
	if !found {
		return time.Time{}, false
	}
	for _, line := range lines[start:end] {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, startedPrefix) || !strings.HasSuffix(trimmed, startedSuffix) {
			continue
		}
		stamp := strings.TrimSuffix(strings.TrimPrefix(trimmed, startedPrefix), startedSuffix)
		if at, err := time.Parse(time.RFC3339, stamp); err == nil {
			return at, true
		}
	}
	return time.Time{}, false
}

// SetCurrentTask returns content with the "## Current task" body replaced by
// task. A missing section is inserted after the document title. A non-zero
// startedAt is stored under the task as an invisible marker so the next switch
// can record how long the task took.
func SetCurrentTask(content, task string, startedAt time.Time) (string, error) {
	if err := ValidateTask(task); err != nil {
		return "", err
	}
	task = strings.TrimSpace(task)

	lines := splitLines(content)
	start, end, found := section(lines)
	if !found {
		at := insertionPoint(lines)
		with := make([]string, 0, len(lines)+1)
		with = append(with, lines[:at]...)
		with = append(with, heading)
		lines = append(with, lines[at:]...)
		start, end = at+1, at+1
	}

	updated := make([]string, 0, len(lines)+5)
	updated = append(updated, lines[:start]...)
	updated = append(updated, "", task, "")
	if !startedAt.IsZero() {
		updated = append(updated, startedPrefix+startedAt.Format(time.RFC3339)+startedSuffix, "")
	}
	updated = append(updated, lines[end:]...)
	for len(updated) > 0 && strings.TrimSpace(updated[len(updated)-1]) == "" {
		updated = updated[:len(updated)-1]
	}
	return strings.Join(updated, "\n") + "\n", nil
}

// ValidateTask reports whether task can be stored as a "## Current task" body.
// Headings are rejected because they would silently end the section and make the
// tail of the task read back as a separate part of the document.
func ValidateTask(task string) error {
	trimmed := strings.TrimSpace(task)
	if trimmed == "" {
		return fmt.Errorf("task text is empty")
	}
	if !utf8.ValidString(trimmed) || strings.ContainsRune(trimmed, '\x00') {
		return fmt.Errorf("task text must be valid UTF-8 without NUL bytes")
	}
	// now.md is read outside the TUI too — cat, glow, an editor — none of which
	// strip escapes before they reach the terminal.
	if strings.ContainsFunc(trimmed, md.IsTerminalControl) {
		return fmt.Errorf("task text must not contain terminal control characters")
	}
	for line := range strings.SplitSeq(trimmed, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			return fmt.Errorf("task text must not contain Markdown headings")
		}
	}
	return nil
}

// section locates the body of the "## Current task" section as a [start, end)
// line range. Headings inside fenced code blocks are ignored so a task that
// quotes Markdown cannot move the section boundary.
func section(lines []string) (int, int, bool) {
	start, fence := -1, ""
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if fence != "" {
			if strings.HasPrefix(trimmed, fence) {
				fence = ""
			}
			continue
		}
		if strings.HasPrefix(trimmed, "```") {
			fence = "```"
			continue
		}
		if strings.HasPrefix(trimmed, "~~~") {
			fence = "~~~"
			continue
		}
		if start >= 0 && strings.HasPrefix(trimmed, "#") {
			return start, index, true
		}
		if start < 0 && strings.EqualFold(trimmed, heading) {
			start = index + 1
		}
	}
	if start >= 0 {
		return start, len(lines), true
	}
	return 0, 0, false
}

// insertionPoint returns the line index the section should be created at: right
// after the document title, or at the top when there is none.
func insertionPoint(lines []string) int {
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			break
		}
		if strings.HasPrefix(trimmed, "# ") {
			return index + 1
		}
	}
	return 0
}

// splitLines drops the empty element a trailing newline produces, so callers do
// not have to special-case the end of the file.
func splitLines(content string) []string {
	content = strings.TrimSuffix(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	if content == "" {
		return nil
	}
	return strings.Split(content, "\n")
}
