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

func TestFormatDurationRendering(t *testing.T) {
	cases := map[time.Duration]string{
		20 * time.Second:                "1m", // sub-minute rounds up instead of vanishing
		45 * time.Minute:                "45m",
		2*time.Hour + 15*time.Minute:    "2h15m",
		26*time.Hour + 90*time.Second:   "26h2m",
		59*time.Minute + 40*time.Second: "1h0m",
	}
	for d, want := range cases {
		if got := FormatDuration(d); got != want {
			t.Errorf("FormatDuration(%v) = %q, want %q", d, got, want)
		}
	}
}

func TestFormatIncludesDurationInHeading(t *testing.T) {
	entry := Entry{
		Time: time.Date(2026, 8, 9, 14, 20, 0, 0, time.UTC), Text: "Task done: Rotate tokens",
		Branch: "main", Duration: 2*time.Hour + 15*time.Minute,
	}
	got, err := Format(entry)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "## 2026-08-09 14:20 — Branch: `main` — Took: 2h15m\n") {
		t.Fatalf("Format() = %q, want the Took heading suffix", got)
	}
	// Zero duration must leave the heading untouched.
	entry.Duration = 0
	got, err = Format(entry)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "Took:") {
		t.Fatalf("Format() included Took for zero duration: %q", got)
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

func TestFormatSelectionListsMetadataAsBulletsAndFencesBackticks(t *testing.T) {
	when := time.Date(2026, 7, 22, 14, 32, 0, 0, time.UTC)
	got, err := Format(Entry{
		Time: when, Selection: true, Branch: "feature/`quoted`", SourceCWD: "/tmp/api",
		Text: "line one\nline two",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Terminal capture", "- Branch: ", "- Source: `/tmp/api`", "feature/`quoted`"} {
		if !strings.Contains(got, want) {
			t.Fatalf("selection format missing %q:\n%s", want, got)
		}
	}
	// A value containing backticks must be wrapped in a longer, padded fence.
	if !strings.Contains(got, "`` feature/`quoted` ``") {
		t.Fatalf("inline code fence was not widened:\n%s", got)
	}
}

func TestFenceAlwaysOutgrowsTheLongestBacktickRun(t *testing.T) {
	cases := map[string]string{
		"plain text":   "```",
		"one ` inside": "```",
		"``` inside":   "````",
		"````` inside": "``````",
	}
	for text, want := range cases {
		if got := Fence(text); got != want {
			t.Fatalf("Fence(%q) = %q, want %q", text, got, want)
		}
	}
}
