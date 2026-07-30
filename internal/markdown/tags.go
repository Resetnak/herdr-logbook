package markdown

import (
	"sort"
	"strings"
)

// Tags extracts a small, deliberately conservative subset of YAML front matter.
func Tags(content string) []string {
	lines := strings.Split(content, "\n")
	seen := make(map[string]bool)
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		inTags := false
		for _, line := range lines[1:] {
			trimmed := strings.TrimSpace(line)
			if trimmed == "---" {
				break
			}
			if strings.HasPrefix(trimmed, "tags:") {
				inTags = true
				value := strings.TrimSpace(strings.TrimPrefix(trimmed, "tags:"))
				value = strings.Trim(value, "[]")
				for _, tag := range strings.Split(value, ",") {
					addTag(seen, tag)
				}
				continue
			}
			if inTags && strings.HasPrefix(trimmed, "-") {
				addTag(seen, strings.TrimPrefix(trimmed, "-"))
				continue
			}
			if inTags && trimmed != "" {
				inTags = false
			}
		}
	}
	inFence := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		for _, field := range strings.Fields(line) {
			// A field of nothing but hashes is a heading marker, not a tag.
			if strings.HasPrefix(field, "#") && strings.Trim(field, "#") != "" {
				addTag(seen, strings.TrimRight(strings.TrimPrefix(field, "#"), ".,;:!?)]}"))
			}
		}
	}
	tags := make([]string, 0, len(seen))
	for tag := range seen {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
}

func addTag(seen map[string]bool, raw string) {
	tag := strings.ToLower(strings.Trim(strings.TrimSpace(raw), "'\""))
	if tag != "" {
		seen[tag] = true
	}
}
