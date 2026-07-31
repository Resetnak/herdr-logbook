package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCollect_EmptyStore(t *testing.T) {
	tempDir := t.TempDir()

	report, err := Collect(tempDir, "test-project", "main", 7)
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if len(report.Items) != 0 {
		t.Errorf("Expected 0 items, got %d", len(report.Items))
	}
}

func TestCollect_InboxParsing(t *testing.T) {
	tempDir := t.TempDir()
	inboxDir := filepath.Join(tempDir, "inbox")
	if err := os.MkdirAll(inboxDir, 0755); err != nil {
		t.Fatalf("Failed to create inbox dir: %v", err)
	}

	day := time.Now().Format("2006-01-02")
	content := "# Inbox — " + time.Now().Format("2006-01") + `

## ` + day + ` 14:20

Some note text

## ` + day + ` 10:15 — Branch: ` + "`main`" + `

Task done: Implement token rotation
`
	if err := os.WriteFile(filepath.Join(inboxDir, time.Now().Format("2006-01")+".md"), []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write inbox file: %v", err)
	}

	report, err := Collect(tempDir, "test-project", "main", 1)
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	if len(report.Items) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(report.Items))
	}

	// Because of sorting by time descending, 14:20 should be first, 10:15 second
	if report.Items[0].Kind != "capture" {
		t.Errorf("Expected first item to be capture, got %s", report.Items[0].Kind)
	}
	if report.Items[0].Summary != "Some note text" {
		t.Errorf("Expected 'Some note text', got '%s'", report.Items[0].Summary)
	}

	if report.Items[1].Kind != "task" {
		t.Errorf("Expected second item to be task, got %s", report.Items[1].Kind)
	}
	if report.Items[1].Summary != "Implement token rotation" {
		t.Errorf("Expected 'Implement token rotation', got '%s'", report.Items[1].Summary)
	}
}

func TestCollect_InboxMetadataSuffix(t *testing.T) {
	tempDir := t.TempDir()
	inboxDir := filepath.Join(tempDir, "inbox")
	if err := os.MkdirAll(inboxDir, 0755); err != nil {
		t.Fatalf("Failed to create inbox dir: %v", err)
	}

	stamp := time.Now().Format("2006-01-02") + " 14:20"
	content := `## ` + stamp + ` — Branch: ` + "`main`" + ` — Source: ` + "`/path`" + `

Note with metadata
`
	if err := os.WriteFile(filepath.Join(inboxDir, "meta.md"), []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write inbox file: %v", err)
	}

	report, err := Collect(tempDir, "test-project", "main", 1)
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	if len(report.Items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(report.Items))
	}

	if report.Items[0].Time.Format("2006-01-02 15:04") != stamp {
		t.Errorf("Failed to parse correct time, got: %s", report.Items[0].Time)
	}
}

// A terminal capture wraps its text in a code fence and lists its metadata
// first; neither may become the summary.
func TestCollect_TerminalCaptureSummary(t *testing.T) {
	tempDir := t.TempDir()
	inboxDir := filepath.Join(tempDir, "inbox")
	if err := os.MkdirAll(inboxDir, 0755); err != nil {
		t.Fatalf("Failed to create inbox dir: %v", err)
	}

	content := "## " + time.Now().Format("2006-01-02") + " 09:00 — Terminal capture\n\n" +
		"- Branch: `main`\n- Source: `/tmp/api`\n\n" +
		"```text\npanic: connection refused\n\tat pool.go:42\n```\n"
	if err := os.WriteFile(filepath.Join(inboxDir, "term.md"), []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write inbox file: %v", err)
	}

	report, err := Collect(tempDir, "test-project", "main", 1)
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if len(report.Items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(report.Items))
	}
	if report.Items[0].Summary != "panic: connection refused" {
		t.Errorf("Expected fenced text as summary, got %q", report.Items[0].Summary)
	}
}

// --days 1 means today, not the last 24 hours.
func TestCollect_CutoffIsCalendarDay(t *testing.T) {
	tempDir := t.TempDir()
	inboxDir := filepath.Join(tempDir, "inbox")
	if err := os.MkdirAll(inboxDir, 0755); err != nil {
		t.Fatalf("Failed to create inbox dir: %v", err)
	}

	content := "## " + time.Now().Format("2006-01-02") + " 00:01\n\nEarly today\n" +
		"## " + time.Now().AddDate(0, 0, -1).Format("2006-01-02") + " 23:59\n\nLate yesterday\n"
	if err := os.WriteFile(filepath.Join(inboxDir, "cutoff.md"), []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write inbox file: %v", err)
	}

	report, err := Collect(tempDir, "test-project", "main", 1)
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if len(report.Items) != 1 || report.Items[0].Summary != "Early today" {
		t.Fatalf("Expected only today's entry, got %+v", report.Items)
	}

	week, err := Collect(tempDir, "test-project", "main", 7)
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if len(week.Items) != 2 {
		t.Errorf("Expected 2 items over 7 days, got %d", len(week.Items))
	}
}

// A heading that is not a timestamp, and a timestamp with no body, must not
// take the rest of the file down with them.
func TestCollect_MalformedInboxEntries(t *testing.T) {
	tempDir := t.TempDir()
	inboxDir := filepath.Join(tempDir, "inbox")
	if err := os.MkdirAll(inboxDir, 0755); err != nil {
		t.Fatalf("Failed to create inbox dir: %v", err)
	}

	day := time.Now().Format("2006-01-02")
	content := "## not a timestamp\n\nOrphan body\n\n" +
		"## " + day + " 08:00\n\n" +
		"## " + day + " 09:00\n\nReal note\n"
	if err := os.WriteFile(filepath.Join(inboxDir, "malformed.md"), []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write inbox file: %v", err)
	}

	report, err := Collect(tempDir, "test-project", "main", 1)
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if len(report.Items) != 2 {
		t.Fatalf("Expected 2 items, got %d: %+v", len(report.Items), report.Items)
	}
	if report.Items[0].Summary != "Real note" {
		t.Errorf("Expected 'Real note' first, got %q", report.Items[0].Summary)
	}
	if report.Items[1].Summary != "(empty entry)" {
		t.Errorf("Expected the body-less heading to report '(empty entry)', got %q", report.Items[1].Summary)
	}
}

// --days 0 means today, not an empty window.
func TestCollect_DaysClampedToOne(t *testing.T) {
	report, err := Collect(t.TempDir(), "test-project", "main", 0)
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if report.Days != 1 {
		t.Errorf("Expected Days to be clamped to 1, got %d", report.Days)
	}
}

func TestCollect_DecisionWithoutDateIsSkipped(t *testing.T) {
	tempDir := t.TempDir()
	decisionsDir := filepath.Join(tempDir, "decisions")
	if err := os.MkdirAll(decisionsDir, 0755); err != nil {
		t.Fatalf("Failed to create decisions dir: %v", err)
	}

	content := "# Decision: Undated draft\n\n- Status: proposed\n"
	if err := os.WriteFile(filepath.Join(decisionsDir, "draft.md"), []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write decision file: %v", err)
	}

	report, err := Collect(tempDir, "test-project", "main", 7)
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if len(report.Items) != 0 {
		t.Errorf("Expected an undated decision to be skipped, got %+v", report.Items)
	}
}

func TestCollect_Decisions(t *testing.T) {
	tempDir := t.TempDir()
	decisionsDir := filepath.Join(tempDir, "decisions")
	if err := os.MkdirAll(decisionsDir, 0755); err != nil {
		t.Fatalf("Failed to create decisions dir: %v", err)
	}

	day := time.Now().AddDate(0, 0, -2).Format("2006-01-02")
	content := `# Decision: Auth rewrite strategy

- Date: ` + day + `
- Status: accepted
- Project: api-gateway
- Branch: main
`
	if err := os.WriteFile(filepath.Join(decisionsDir, day+"-auth-rewrite.md"), []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write decision file: %v", err)
	}

	report, err := Collect(tempDir, "test-project", "main", 7)
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	if len(report.Items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(report.Items))
	}

	if report.Items[0].Kind != "decision" {
		t.Errorf("Expected kind decision, got %s", report.Items[0].Kind)
	}
	if report.Items[0].Summary != "Auth rewrite strategy" {
		t.Errorf("Expected 'Auth rewrite strategy', got '%s'", report.Items[0].Summary)
	}
}

func TestCollect_NowFile(t *testing.T) {
	tempDir := t.TempDir()

	content := `# Now

## Current task

Implement heatmap rendering

## Next steps

- Add tests
`
	if err := os.WriteFile(filepath.Join(tempDir, "now.md"), []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write now.md: %v", err)
	}

	report, err := Collect(tempDir, "test-project", "main", 7)
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	if report.CurrentTask != "Implement heatmap rendering" {
		t.Errorf("Expected 'Implement heatmap rendering', got '%s'", report.CurrentTask)
	}
}

func TestCollect_Heatmap(t *testing.T) {
	tempDir := t.TempDir()
	report, err := Collect(tempDir, "test-project", "main", 7)
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	if len(report.Heatmap) != 28 {
		t.Errorf("Expected Heatmap to have 28 elements, got %d", len(report.Heatmap))
	}
}

func TestCollect_Streak(t *testing.T) {
	tempDir := t.TempDir()
	inboxDir := filepath.Join(tempDir, "inbox")
	if err := os.MkdirAll(inboxDir, 0755); err != nil {
		t.Fatalf("Failed to create inbox dir: %v", err)
	}

	today := time.Now()
	yesterday := today.AddDate(0, 0, -1)
	dayBefore := today.AddDate(0, 0, -2)

	content := "## " + today.Format("2006-01-02 15:04") + "\n\nToday entry\n" +
		"## " + yesterday.Format("2006-01-02 15:04") + "\n\nYesterday entry\n" +
		"## " + dayBefore.Format("2006-01-02 15:04") + "\n\nDay before entry\n"

	if err := os.WriteFile(filepath.Join(inboxDir, "streak.md"), []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write inbox file: %v", err)
	}

	report, err := Collect(tempDir, "test-project", "main", 7)
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	if report.Streak != 3 {
		t.Errorf("Expected streak to be 3, got %d", report.Streak)
	}
}

func TestFormatMarkdown_TasksAndDecisions(t *testing.T) {
	report := DigestReport{
		Generated: time.Now(),
		Project:   "test-project",
		Items: []ActivityItem{
			{Kind: "task", Summary: "Do something"},
			{Kind: "capture", Summary: "A thought"},
			{Kind: "decision", Summary: "Use Postgres"},
			{Kind: "task", Summary: "Cross-project task", Project: "other-project"},
		},
		CurrentTask: "Working on tests",
	}

	md := FormatMarkdown(report)

	if !strings.Contains(md, "## Standup —") {
		t.Error("Missing '## Standup —'")
	}
	if !strings.Contains(md, "### ✅ Completed") {
		t.Error("Missing '### ✅ Completed'")
	}
	if !strings.Contains(md, "- Task done: Do something") {
		t.Error("Missing '- Task done: Do something'")
	}
	if !strings.Contains(md, "- A thought") {
		t.Error("Missing '- A thought'")
	}
	if !strings.Contains(md, "### 🏗️ Decisions") {
		t.Error("Missing '### 🏗️ Decisions'")
	}
	if !strings.Contains(md, "- ADR: Use Postgres") {
		t.Error("Missing '- ADR: Use Postgres'")
	}
	if !strings.Contains(md, "- Task done: Cross-project task (other-project)") {
		t.Error("Missing cross-project task notation")
	}
	if !strings.Contains(md, "### ⚡ Currently Working On") {
		t.Error("Missing '### ⚡ Currently Working On'")
	}
	if !strings.Contains(md, "- Working on tests") {
		t.Error("Missing '- Working on tests'")
	}
}

// Decisions carry the same cross-project suffix as captures, and a current task
// that is already a bullet list must not get a second dash per line.
func TestFormatMarkdown_CrossProjectDecisionAndBulletedTask(t *testing.T) {
	md := FormatMarkdown(DigestReport{
		Generated:   time.Now(),
		Project:     "api-gateway",
		Items:       []ActivityItem{{Kind: "decision", Summary: "Adopt opaque tokens", Project: "auth-service"}},
		CurrentTask: "- Ship the rotation PR\n\n- Backfill the changelog",
	})

	if !strings.Contains(md, "- ADR: Adopt opaque tokens (auth-service)") {
		t.Errorf("Missing cross-project decision notation in:\n%s", md)
	}
	if strings.Contains(md, "- - ") {
		t.Errorf("Bulleted current task was double-dashed in:\n%s", md)
	}
	for _, want := range []string{"- Ship the rotation PR", "- Backfill the changelog"} {
		if !strings.Contains(md, want) {
			t.Errorf("Missing %q in:\n%s", want, md)
		}
	}
}

func TestFormatMarkdown_Empty(t *testing.T) {
	report := DigestReport{
		Generated: time.Now(),
		Project:   "test-project",
	}

	md := FormatMarkdown(report)

	if !strings.Contains(md, "## Standup —") {
		t.Error("Missing '## Standup —'")
	}
	if strings.Contains(md, "### ✅ Completed") {
		t.Error("Should not contain '### ✅ Completed'")
	}
	if strings.Contains(md, "### 🏗️ Decisions") {
		t.Error("Should not contain '### 🏗️ Decisions'")
	}
	if strings.Contains(md, "### ⚡ Currently Working On") {
		t.Error("Should not contain '### ⚡ Currently Working On'")
	}
}
