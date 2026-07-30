package markdown

import (
	"strings"
	"testing"
)

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

// bufio.Scanner gives up on any line past 64 KB, and a note can easily hold one
// pasted log line or a minified payload before its heading.
func TestTitleSurvivesAVeryLongLine(t *testing.T) {
	content := strings.Repeat("x", 128*1024) + "\n\n# Real Title\n"
	if got := Title(content, "/notes/fallback.md"); got != "Real Title" {
		t.Fatalf("Title() = %q, want %q", got, "Real Title")
	}
}
