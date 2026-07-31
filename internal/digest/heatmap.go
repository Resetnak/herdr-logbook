package digest

import (
	"strings"
	"time"
)

// RenderHeatmap draws a GitHub-style contribution grid: one column per week,
// one row per weekday, oldest week on the left. data holds daily counts ending
// today; days past today stay blank.
func RenderHeatmap(data []DayActivity, weeks int) string {
	if weeks <= 0 {
		weeks = 4
	}

	counts := make(map[string]int, len(data))
	for _, day := range data {
		counts[day.Date] = day.Count
	}

	today := time.Now()
	if len(data) > 0 {
		if parsed, err := time.ParseInLocation("2006-01-02", data[len(data)-1].Date, time.Local); err == nil {
			today = parsed
		}
	}

	// Only every other row is labelled, the way GitHub does it.
	labels := map[time.Weekday]string{time.Monday: "Mon ", time.Wednesday: "Wed ", time.Friday: "Fri "}

	var out strings.Builder
	for row := range 7 {
		if label, ok := labels[time.Weekday(row)]; ok {
			out.WriteString(label)
		} else {
			out.WriteString("    ")
		}
		for week := range weeks {
			offset := (week-(weeks-1))*7 + (row - int(today.Weekday()))
			if offset > 0 {
				out.WriteString("  ")
				continue
			}
			out.WriteString(block(counts[today.AddDate(0, 0, offset).Format("2006-01-02")]) + " ")
		}
		out.WriteString("\n")
	}

	out.WriteString("\nLess ")
	for _, count := range []int{0, 1, 2, 4, 6} {
		out.WriteString(block(count) + " ")
	}
	out.WriteString("More")
	return out.String()
}

func block(count int) string {
	switch {
	case count <= 0:
		return "·"
	case count == 1:
		return "░"
	case count <= 3:
		return "▒"
	case count <= 5:
		return "▓"
	default:
		return "█"
	}
}
