package digest

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// CombinedName labels the all-projects report.
const CombinedName = "All projects"

// Store is one project's store as CollectAll reads it.
type Store struct {
	Name   string
	Branch string
	Root   string
	// Keep retains the project even when it has no activity at all, so the
	// current project stays selectable in an otherwise quiet workspace.
	Keep bool
}

// MultiReport is the digest across projects: one report per project that has
// anything to show, in the order the stores were given, plus the merge of them.
type MultiReport struct {
	Projects []DigestReport
	// Combined merges every project's items (newest first), sums the daily
	// heatmap counts, and takes its streak from the summed days. Its
	// CurrentTask is empty; each project's own task is in Projects.
	Combined DigestReport
}

// CollectAll runs Collect over every store and merges the results. A store with
// no activity, no current task, and Keep unset is left out; a store listed twice
// is read once.
func CollectAll(stores []Store, days int) (MultiReport, error) {
	now := time.Now()
	combined := DigestReport{Generated: now, Days: max(days, 1), Project: CombinedName}
	totals := make(map[string]int)
	seen := make(map[string]bool, len(stores))
	var multi MultiReport

	for _, store := range stores {
		if seen[store.Root] {
			continue
		}
		seen[store.Root] = true
		report, counts := collect(store.Root, store.Name, store.Branch, days, now)
		if len(counts) == 0 && report.CurrentTask == "" && !store.Keep {
			continue
		}
		multi.Projects = append(multi.Projects, report)
		combined.Items = append(combined.Items, report.Items...)
		for date, count := range counts {
			totals[date] += count
		}
	}

	sort.SliceStable(combined.Items, func(i, j int) bool { return combined.Items[i].Time.After(combined.Items[j].Time) })
	for i := heatmapDays - 1; i >= 0; i-- {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		combined.Heatmap = append(combined.Heatmap, DayActivity{Date: date, Count: totals[date]})
	}
	combined.Streak = calculateStreak(totals, now)
	multi.Combined = combined
	return multi, nil
}

// FormatMarkdownAll renders the cross-project standup: one section per project
// by default, or a single newest-first list with "(project)" tags when byTime.
func FormatMarkdownAll(multi MultiReport, byTime bool) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Standup — %s\n\n", multi.Combined.Generated.Format("2006-01-02")))

	if byTime {
		writeItems(&sb, multi.Combined.Items, "", "###")
		writeCurrentTasks(&sb, multi.Projects)
	} else {
		for _, report := range multi.Projects {
			if len(report.Items) == 0 && report.CurrentTask == "" {
				continue
			}
			sb.WriteString(fmt.Sprintf("### %s\n\n", report.Project))
			writeItems(&sb, report.Items, report.Project, "####")
			if report.CurrentTask != "" {
				sb.WriteString("#### ⚡ Currently Working On\n")
				writeTaskLines(&sb, "", report.CurrentTask)
				sb.WriteString("\n")
			}
		}
	}

	if len(multi.Combined.Items) == 0 && !anyCurrentTask(multi.Projects) {
		sb.WriteString("No activity recorded in this period.\n\n")
	}
	if len(multi.Projects) > 1 {
		if total := TrackedMinutes(multi.Combined.Items); total > 0 {
			sb.WriteString(fmt.Sprintf("⏱ Total tracked: %s\n\n", FormatTook(total)))
		}
	}
	sb.WriteString(footer)
	return sb.String()
}

func writeCurrentTasks(sb *strings.Builder, reports []DigestReport) {
	if !anyCurrentTask(reports) {
		return
	}
	sb.WriteString("### ⚡ Currently Working On\n")
	for _, report := range reports {
		if report.CurrentTask != "" {
			writeTaskLines(sb, report.Project+": ", report.CurrentTask)
		}
	}
	sb.WriteString("\n")
}

func writeTaskLines(sb *strings.Builder, prefix, task string) {
	for _, line := range strings.Split(task, "\n") {
		if strings.TrimSpace(line) != "" {
			sb.WriteString("- " + prefix + strings.TrimPrefix(line, "- ") + "\n")
		}
	}
}

func anyCurrentTask(reports []DigestReport) bool {
	for _, report := range reports {
		if report.CurrentTask != "" {
			return true
		}
	}
	return false
}
