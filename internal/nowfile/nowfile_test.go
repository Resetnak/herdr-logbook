package nowfile

import (
	"strings"
	"testing"
	"time"
)

const template = `# Now

## Current task

Describe the task currently in progress.

## Next steps

- [ ] Add the next concrete action.

## Blockers

- None.

## Context
`

func TestCurrentTaskTreatsPlaceholderAsUnset(t *testing.T) {
	if task := CurrentTask(template); task != "" {
		t.Fatalf("expected empty task for a fresh now.md, got %q", task)
	}
}

func TestCurrentTaskReadsBody(t *testing.T) {
	content := "# Now\n\n## Current task\n\nRotate the signing tokens\n\n## Blockers\n\n- None.\n"
	if task := CurrentTask(content); task != "Rotate the signing tokens" {
		t.Fatalf("unexpected task: %q", task)
	}
}

func TestCurrentTaskMissingSection(t *testing.T) {
	if task := CurrentTask("# Now\n\n## Blockers\n\n- None.\n"); task != "" {
		t.Fatalf("expected empty task, got %q", task)
	}
	if task := CurrentTask(""); task != "" {
		t.Fatalf("expected empty task for empty content, got %q", task)
	}
}

func TestCurrentTaskIgnoresHeadingInsideFence(t *testing.T) {
	content := "# Now\n\n## Current task\n\nFix the parser:\n\n```md\n## Blockers\n```\n\n## Blockers\n\n- None.\n"
	want := "Fix the parser:\n\n```md\n## Blockers\n```"
	if task := CurrentTask(content); task != want {
		t.Fatalf("unexpected task: %q", task)
	}
}

func TestSetCurrentTaskPreservesOtherSections(t *testing.T) {
	updated, err := SetCurrentTask(template, "  Rotate the signing tokens  ", time.Time{})
	if err != nil {
		t.Fatalf("SetCurrentTask: %v", err)
	}
	if task := CurrentTask(updated); task != "Rotate the signing tokens" {
		t.Fatalf("round trip failed, got %q", task)
	}
	for _, section := range []string{"## Next steps", "- [ ] Add the next concrete action.", "## Blockers", "- None.", "## Context"} {
		if !strings.Contains(updated, section) {
			t.Fatalf("section %q was dropped:\n%s", section, updated)
		}
	}
	if strings.Contains(updated, placeholder) {
		t.Fatalf("placeholder survived the write:\n%s", updated)
	}
}

func TestSetCurrentTaskCreatesMissingSection(t *testing.T) {
	updated, err := SetCurrentTask("# Now\n\n## Blockers\n\n- None.\n", "Ship the release", time.Time{})
	if err != nil {
		t.Fatalf("SetCurrentTask: %v", err)
	}
	if task := CurrentTask(updated); task != "Ship the release" {
		t.Fatalf("round trip failed, got %q", task)
	}
	if !strings.Contains(updated, "## Blockers") {
		t.Fatalf("existing section was dropped:\n%s", updated)
	}
}

func TestSetCurrentTaskOnEmptyContent(t *testing.T) {
	updated, err := SetCurrentTask("", "Ship the release", time.Time{})
	if err != nil {
		t.Fatalf("SetCurrentTask: %v", err)
	}
	if task := CurrentTask(updated); task != "Ship the release" {
		t.Fatalf("round trip failed, got %q", task)
	}
}

func TestSetCurrentTaskIsIdempotentlyRepeatable(t *testing.T) {
	first, err := SetCurrentTask(template, "First task", time.Time{})
	if err != nil {
		t.Fatalf("SetCurrentTask: %v", err)
	}
	second, err := SetCurrentTask(first, "Second task", time.Time{})
	if err != nil {
		t.Fatalf("SetCurrentTask: %v", err)
	}
	if task := CurrentTask(second); task != "Second task" {
		t.Fatalf("unexpected task: %q", task)
	}
	if strings.Contains(second, "First task") {
		t.Fatalf("previous task was left behind:\n%s", second)
	}
}

func TestStartedAtRoundTrip(t *testing.T) {
	started := time.Date(2026, 8, 9, 14, 20, 0, 0, time.FixedZone("CEST", 2*3600))
	updated, err := SetCurrentTask(template, "Rotate the signing tokens", started)
	if err != nil {
		t.Fatalf("SetCurrentTask: %v", err)
	}
	at, ok := StartedAt(updated)
	if !ok || !at.Equal(started) {
		t.Fatalf("StartedAt = %v, %v; want %v, true", at, ok, started)
	}
	// The marker must never leak into the task body.
	if task := CurrentTask(updated); task != "Rotate the signing tokens" {
		t.Fatalf("marker leaked into the task: %q", task)
	}
	// A later switch replaces the marker instead of accumulating them.
	next := started.Add(2 * time.Hour)
	second, err := SetCurrentTask(updated, "Ship the release", next)
	if err != nil {
		t.Fatalf("SetCurrentTask: %v", err)
	}
	if strings.Count(second, startedPrefix) != 1 {
		t.Fatalf("expected exactly one marker:\n%s", second)
	}
	if at, ok := StartedAt(second); !ok || !at.Equal(next) {
		t.Fatalf("StartedAt after switch = %v, %v; want %v, true", at, ok, next)
	}
}

func TestStartedAtMissingOrHandEdited(t *testing.T) {
	if _, ok := StartedAt(template); ok {
		t.Fatal("expected no started-at in a fresh now.md")
	}
	// A hand-edited or truncated marker reads as unknown, never as an error.
	content := "# Now\n\n## Current task\n\nOld task\n\n<!-- herdr-logbook: started not-a-time -->\n"
	if _, ok := StartedAt(content); ok {
		t.Fatal("expected an unparseable marker to read as unknown")
	}
	if task := CurrentTask(content); task != "Old task" {
		t.Fatalf("broken marker leaked into the task: %q", task)
	}
}

func TestSetCurrentTaskZeroTimeWritesNoMarker(t *testing.T) {
	updated, err := SetCurrentTask(template, "Rotate the signing tokens", time.Time{})
	if err != nil {
		t.Fatalf("SetCurrentTask: %v", err)
	}
	if strings.Contains(updated, startedPrefix) {
		t.Fatalf("zero time produced a marker:\n%s", updated)
	}
}

func TestValidateTaskRejectsBadInput(t *testing.T) {
	cases := map[string]string{
		"empty":        "   ",
		"heading":      "Fix it\n## Blockers",
		"nul byte":     "Fix\x00it",
		"lone hash":    "#",
		"leading hash": "  # Now",
		// now.md is read outside the TUI too — cat, glow, an editor — where
		// nothing strips escapes before they reach the terminal.
		"osc 52":  "Fix \x1b]52;c;aGFja2Vk\a",
		"osc 0":   "Fix \x1b]0;pwned\a",
		"bell":    "Fix\ait",
		"c1 csi":  "Fix \u009b2J",
		"del":     "Fix\x7fit",
		"lone cr": "Fix\rit",
	}
	for name, task := range cases {
		if err := ValidateTask(task); err == nil {
			t.Fatalf("%s: expected an error", name)
		}
	}
	if err := ValidateTask("Rotate the signing tokens"); err != nil {
		t.Fatalf("valid task rejected: %v", err)
	}
}

func TestSetCurrentTaskNormalizesCRLF(t *testing.T) {
	updated, err := SetCurrentTask("# Now\r\n\r\n## Current task\r\n\r\nOld\r\n", "New", time.Time{})
	if err != nil {
		t.Fatalf("SetCurrentTask: %v", err)
	}
	if strings.Contains(updated, "\r") {
		t.Fatalf("carriage returns survived:\n%q", updated)
	}
	if task := CurrentTask(updated); task != "New" {
		t.Fatalf("unexpected task: %q", task)
	}
}
