package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeStore builds a store from raw inbox entries and an optional current task.
func writeStore(t *testing.T, currentTask string, entries ...string) string {
	t.Helper()
	root := t.TempDir()
	inbox := filepath.Join(root, "inbox")
	if err := os.MkdirAll(inbox, 0o755); err != nil {
		t.Fatal(err)
	}
	var body strings.Builder
	body.WriteString("# Inbox\n\n")
	for _, entry := range entries {
		body.WriteString(entry)
	}
	if err := os.WriteFile(filepath.Join(inbox, "2026-01.md"), []byte(body.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	if currentTask != "" {
		now := "# Now\n\n## Current task\n\n" + currentTask + "\n\n## Next steps\n"
		if err := os.WriteFile(filepath.Join(root, "now.md"), []byte(now), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func entry(at time.Time, line string) string {
	return "## " + at.Format("2006-01-02 15:04") + "\n\n" + line + "\n\n"
}

func tookEntry(at time.Time, line, took string) string {
	return "## " + at.Format("2006-01-02 15:04") + " — Took: " + took + "\n\n" + line + "\n\n"
}

func TestCollectAll_MergesProjects(t *testing.T) {
	now := time.Now()
	older := now.Add(-2 * time.Hour)
	if older.Day() != now.Day() {
		older = now // keep both entries on today's calendar day near midnight
	}
	api := writeStore(t, "Rotate tokens", tookEntry(older, "Task done: api work", "30m"))
	web := writeStore(t, "", entry(now, "web capture"))
	idle := writeStore(t, "")

	multi, err := CollectAll([]Store{
		{Name: "api", Root: api},
		{Name: "web", Root: web},
		{Name: "idle", Root: idle},
	}, 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(multi.Projects) != 2 || multi.Projects[0].Project != "api" || multi.Projects[1].Project != "web" {
		t.Fatalf("projects = %+v, want api and web only", multi.Projects)
	}
	if got := len(multi.Combined.Items); got != 2 {
		t.Fatalf("combined items = %d, want 2", got)
	}
	if multi.Combined.Items[0].Time.Before(multi.Combined.Items[1].Time) {
		t.Errorf("combined items are not newest first: %+v", multi.Combined.Items)
	}
	if multi.Combined.Project != CombinedName {
		t.Errorf("combined project = %q", multi.Combined.Project)
	}
	today := multi.Combined.Heatmap[len(multi.Combined.Heatmap)-1]
	if today.Count != 2 {
		t.Errorf("combined heatmap today = %d, want the sum of both projects (2)", today.Count)
	}
	if multi.Combined.Streak != 1 {
		t.Errorf("combined streak = %d, want 1", multi.Combined.Streak)
	}
	if multi.Projects[0].CurrentTask != "Rotate tokens" {
		t.Errorf("per-project current task = %q", multi.Projects[0].CurrentTask)
	}
	if multi.Combined.CurrentTask != "" {
		t.Errorf("combined current task = %q, want empty", multi.Combined.CurrentTask)
	}
}

func TestCollectAll_KeepAndDuplicates(t *testing.T) {
	idle := writeStore(t, "")
	multi, err := CollectAll([]Store{
		{Name: "idle", Root: idle, Keep: true},
		{Name: "idle again", Root: idle, Keep: true},
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(multi.Projects) != 1 || multi.Projects[0].Project != "idle" {
		t.Fatalf("projects = %+v, want the one kept store read once", multi.Projects)
	}
}

func TestCollectAll_StreakSpansProjects(t *testing.T) {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	api := writeStore(t, "", entry(now, "api today"))
	web := writeStore(t, "", entry(yesterday, "web yesterday"))

	multi, err := CollectAll([]Store{{Name: "api", Root: api}, {Name: "web", Root: web}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if multi.Combined.Streak != 2 {
		t.Errorf("combined streak = %d, want 2 (api today, web yesterday)", multi.Combined.Streak)
	}
	// Each project alone only has one active day; only the merge reaches two.
	if multi.Projects[0].Streak != 1 || multi.Projects[1].Streak != 1 {
		t.Errorf("per-project streaks = %d, %d, want 1, 1", multi.Projects[0].Streak, multi.Projects[1].Streak)
	}
}

func TestFormatMarkdownAll_GroupsByProject(t *testing.T) {
	now := time.Now()
	api := writeStore(t, "Rotate tokens", tookEntry(now, "Task done: api work", "30m"))
	web := writeStore(t, "", entry(now, "web capture"))
	multi, _ := CollectAll([]Store{{Name: "api", Root: api}, {Name: "web", Root: web}}, 1)

	out := FormatMarkdownAll(multi, false)
	for _, want := range []string{
		"### api\n", "### web\n",
		"- Task done: api work — 30m\n",
		"- web capture\n",
		"#### ⚡ Currently Working On\n- Rotate tokens\n",
		"⏱ Total tracked: 30m",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("grouped markdown is missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "(api)") || strings.Contains(out, "(web)") {
		t.Errorf("grouped markdown repeats the project under its own heading:\n%s", out)
	}
	if strings.Index(out, "### api") > strings.Index(out, "### web") {
		t.Errorf("projects are out of order:\n%s", out)
	}
}

func TestFormatMarkdownAll_ByTime(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-time.Hour)
	if earlier.Day() != now.Day() {
		earlier = now
	}
	api := writeStore(t, "Rotate tokens", entry(earlier, "api capture"))
	web := writeStore(t, "Ship docs", entry(now, "web capture"))
	multi, _ := CollectAll([]Store{{Name: "api", Root: api}, {Name: "web", Root: web}}, 1)

	out := FormatMarkdownAll(multi, true)
	for _, want := range []string{
		"### ✅ Completed\n",
		"- api capture (api)\n",
		"- web capture (web)\n",
		"### ⚡ Currently Working On\n- api: Rotate tokens\n- web: Ship docs\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("time-sorted markdown is missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "### api\n") {
		t.Errorf("time-sorted markdown still has per-project sections:\n%s", out)
	}
	if !earlier.Equal(now) && strings.Index(out, "web capture") > strings.Index(out, "api capture") {
		t.Errorf("newest item is not first:\n%s", out)
	}
}

func TestFormatMarkdownAll_Empty(t *testing.T) {
	multi, _ := CollectAll([]Store{{Name: "idle", Root: writeStore(t, ""), Keep: true}}, 1)
	for _, byTime := range []bool{false, true} {
		out := FormatMarkdownAll(multi, byTime)
		if !strings.Contains(out, "No activity recorded in this period.") {
			t.Errorf("byTime=%v empty report:\n%s", byTime, out)
		}
	}
}
