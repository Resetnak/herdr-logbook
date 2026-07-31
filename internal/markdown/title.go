package markdown

import (
	"path/filepath"
	"strings"
)

func Title(content, filename string) string {
	var fence string
	// strings.Lines, not bufio.Scanner: a note can hold a pasted line longer than
	// the scanner's 64 KB token limit, and Scanner would stop reading right there.
	for raw := range strings.Lines(content) {
		line := strings.TrimSpace(raw)
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
		}
	}
	return StripTerminalControl(strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)))
}
