package capture

import (
	"fmt"
	"strings"
	"time"
)

type Entry struct {
	Time      time.Time
	Text      string
	Branch    string
	SourceCWD string
	Selection bool
}

func Format(entry Entry) (string, error) {
	if strings.TrimSpace(entry.Text) == "" {
		return "", fmt.Errorf("capture text is empty")
	}
	var output strings.Builder
	fmt.Fprintf(&output, "## %s", entry.Time.Format("2006-01-02 15:04"))
	if entry.Selection {
		output.WriteString(" — Terminal capture")
	} else {
		if entry.Branch != "" {
			output.WriteString(" — Branch: ")
			output.WriteString(inlineCode(entry.Branch))
		}
		if entry.SourceCWD != "" {
			output.WriteString(" — Source: ")
			output.WriteString(inlineCode(entry.SourceCWD))
		}
	}
	output.WriteString("\n\n")

	if entry.Selection && (entry.Branch != "" || entry.SourceCWD != "") {
		if entry.Branch != "" {
			fmt.Fprintf(&output, "- Branch: %s\n", inlineCode(entry.Branch))
		}
		if entry.SourceCWD != "" {
			fmt.Fprintf(&output, "- Source: %s\n", inlineCode(entry.SourceCWD))
		}
		output.WriteString("\n")
	}
	if entry.Selection && strings.Contains(entry.Text, "\n") {
		fence := Fence(entry.Text)
		output.WriteString(fence + "text\n")
		output.WriteString(entry.Text)
		if !strings.HasSuffix(entry.Text, "\n") {
			output.WriteByte('\n')
		}
		output.WriteString(fence + "\n")
	} else {
		output.WriteString(entry.Text)
		if !strings.HasSuffix(entry.Text, "\n") {
			output.WriteByte('\n')
		}
	}
	return output.String(), nil
}

func Fence(text string) string {
	longest := longestBacktickRun(text)
	if longest < 2 {
		longest = 2
	}
	return strings.Repeat("`", longest+1)
}

func inlineCode(value string) string {
	length := longestBacktickRun(value) + 1
	if length == 1 {
		return "`" + value + "`"
	}
	fence := strings.Repeat("`", length)
	return fence + " " + value + " " + fence
}

func longestBacktickRun(text string) int {
	longest, current := 0, 0
	for _, r := range text {
		if r == '`' {
			current++
			if current > longest {
				longest = current
			}
		} else {
			current = 0
		}
	}
	return longest
}
