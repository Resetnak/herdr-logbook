package nowfile

import (
	"strings"
	"testing"
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
	updated, err := SetCurrentTask(template, "  Rotate the signing tokens  ")
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
	updated, err := SetCurrentTask("# Now\n\n## Blockers\n\n- None.\n", "Ship the release")
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
	updated, err := SetCurrentTask("", "Ship the release")
	if err != nil {
		t.Fatalf("SetCurrentTask: %v", err)
	}
	if task := CurrentTask(updated); task != "Ship the release" {
		t.Fatalf("round trip failed, got %q", task)
	}
}

func TestSetCurrentTaskIsIdempotentlyRepeatable(t *testing.T) {
	first, err := SetCurrentTask(template, "First task")
	if err != nil {
		t.Fatalf("SetCurrentTask: %v", err)
	}
	second, err := SetCurrentTask(first, "Second task")
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

func TestValidateTaskRejectsBadInput(t *testing.T) {
	cases := map[string]string{
		"empty":       "   ",
		"heading":     "Fix it\n## Blockers",
		"nul byte":    "Fix\x00it",
		"lone hash":   "#",
		"leading hash": "  # Now",
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
	updated, err := SetCurrentTask("# Now\r\n\r\n## Current task\r\n\r\nOld\r\n", "New")
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
