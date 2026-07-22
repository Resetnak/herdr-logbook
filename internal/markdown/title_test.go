package markdown

import "testing"

func TestTitleUsesFirstH1OutsideFencedCode(t *testing.T) {
	content := "```md\n# Not this\n```\n\n# Real title\n"
	if got := Title(content, "fallback.md"); got != "Real title" {
		t.Fatalf("Title() = %q", got)
	}
}

func TestTitleFallsBackToFilename(t *testing.T) {
	if got := Title("No heading\n", "auth-notes.md"); got != "auth-notes" {
		t.Fatalf("Title() = %q", got)
	}
}
