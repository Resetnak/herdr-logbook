package markdown

import (
	"bufio"
	"path/filepath"
	"strings"
)

func Title(content, filename string) string {
	var fence string
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
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
				return title
			}
		}
	}
	return strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
}
