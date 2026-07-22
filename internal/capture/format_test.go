package capture

import (
	"strings"
	"testing"
	"time"
)

func TestFormatProjectCaptureIncludesOnlyPresentMetadata(t *testing.T) {
	entry := Entry{
		Time: time.Date(2026, 7, 22, 14, 32, 0, 0, time.UTC), Text: "Remember this.",
		Branch: "feature/login", SourceCWD: "/workspace/api",
	}
	got, err := Format(entry)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"## 2026-07-22 14:32", "Branch: `feature/login`", "Source: `/workspace/api`", "Remember this."} {
		if !strings.Contains(got, want) {
			t.Fatalf("Format() = %q, want %q", got, want)
		}
	}
}

func TestFormatOmitsEmptyMetadata(t *testing.T) {
	got, err := Format(Entry{Time: time.Date(2026, 7, 22, 14, 32, 0, 0, time.UTC), Text: "Note"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "Branch:") || strings.Contains(got, "Source:") {
		t.Fatalf("Format() included empty metadata: %q", got)
	}
}

func TestFormatMultilineSelectionUsesSafeFence(t *testing.T) {
	text := "first\n```\ninside\n````\nlast"
	got, err := Format(Entry{Time: time.Date(2026, 7, 22, 14, 35, 0, 0, time.UTC), Text: text, Selection: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "— Terminal capture") || !strings.Contains(got, "`````text\n"+text+"\n`````\n") {
		t.Fatalf("Format() unsafe selection fence: %q", got)
	}
}

func TestFormatOneLineSelectionRemainsPlainText(t *testing.T) {
	got, err := Format(Entry{Time: time.Date(2026, 7, 22, 14, 35, 0, 0, time.UTC), Text: "single line", Selection: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "```text") || !strings.Contains(got, "\nsingle line\n") {
		t.Fatalf("Format() one-line selection = %q", got)
	}
}

func TestFormatRejectsBlankText(t *testing.T) {
	if _, err := Format(Entry{Time: time.Now(), Text: " \n\t"}); err == nil {
		t.Fatal("Format() blank error = nil")
	}
}
