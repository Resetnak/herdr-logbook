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

// A title is painted into the notes list as a plain row, with no Glamour between
// it and the terminal. Capture refuses control characters on the way in, but a
// note written in an external editor never passed through that check, so the
// title has to be safe to print on its own.
func TestTitleStripsTerminalControlCharacters(t *testing.T) {
	content := "# \x1b]52;c;aGFja2Vk\aInnocent note\n"
	got := Title(content, "fallback.md")
	if strings.ContainsAny(got, "\x1b\a") {
		t.Fatalf("Title() kept control characters: %q", got)
	}
	if got != "]52;c;aGFja2VkInnocent note" {
		t.Fatalf("Title() = %q", got)
	}

	// A filename is not a safer source than a heading: every byte except / and
	// NUL is legal in one, so the fallback needs the same treatment.
	if got := Title("no heading\n", "\x1b]0;pwned\a.md"); strings.ContainsAny(got, "\x1b\a") {
		t.Fatalf("Title() fallback kept control characters: %q", got)
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
