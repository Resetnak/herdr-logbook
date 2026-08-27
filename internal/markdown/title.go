package markdown

import (
	"path/filepath"
	"strings"
)

// maxTitleRunes bounds titles built from a plain first line; a heading is
// already one deliberate line, but a heading-less note can open with a
// pasted paragraph.
const maxTitleRunes = 80

func Title(content, filename string) string {
	var fence string
	frontmatter := false
	first := true
	fallback := ""
	// strings.Lines, not bufio.Scanner: a note can hold a pasted line longer than
	// the scanner's 64 KB token limit, and Scanner would stop reading right there.
	for raw := range strings.Lines(content) {
		line := strings.TrimSpace(raw)
		if first {
			first = false
			if line == "---" {
				frontmatter = true
				continue
			}
		}
		if frontmatter {
			if line == "---" || line == "..." {
				frontmatter = false
			}
			continue
		}
		if fence != "" {
			if strings.HasPrefix(line, fence) {
				fence = ""
			}
			continue
		}
		if strings.HasPrefix(line, "```") {
			fence = "```"
			continue
		}
		if strings.HasPrefix(line, "~~~") {
			fence = "~~~"
			continue
		}
		if strings.HasPrefix(line, "# ") {
			title := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(strings.TrimPrefix(line, "# ")), "#"))
			if title != "" {
				return StripTerminalControl(title)
			}
			continue
		}
		if fallback == "" && line != "" && !strings.HasPrefix(line, "#") {
			fallback = line
		}
	}
	// A note without any heading still has to be found in a list; its first
	// line beats the immutable filename slug (discussion #18: a title froze at
	// "something-very-important" after the heading was edited away).
	if fallback != "" {
		for _, marker := range []string{"- ", "* ", "+ ", "> "} {
			fallback = strings.TrimPrefix(fallback, marker)
		}
		if runes := []rune(fallback); len(runes) > maxTitleRunes {
			fallback = string(runes[:maxTitleRunes])
		}
		if clean := strings.TrimSpace(StripTerminalControl(fallback)); clean != "" {
			return clean
		}
	}
	return StripTerminalControl(strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)))
}
