package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunCompatibilityPrintsRedactedJSON(t *testing.T) {
	env := map[string]string{
		"HERDR_PLUGIN_ROOT":         "/tmp/plugin",
		"HERDR_PLUGIN_CONTEXT_JSON": `{"focused_pane_cwd":"/tmp/project","selected_text":"do not print me"}`,
	}
	var stdout, stderr bytes.Buffer

	code := run([]string{"compatibility"}, func(key string) string { return env[key] }, strings.NewReader(""), &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run() code = %d, stderr = %q", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "do not print me") {
		t.Fatalf("run() leaked selected text: %s", stdout.String())
	}
	for _, want := range []string{`"focused_pane_cwd": "/tmp/project"`, `"selected_text_present": true`, `"working_directory": "/tmp/project"`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("run() output = %s, want %s", stdout.String(), want)
		}
	}
}

func TestRunCompatibilityRejectsMalformedContext(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"compatibility"}, func(key string) string {
		if key == "HERDR_PLUGIN_CONTEXT_JSON" {
			return "{broken"
		}
		return ""
	}, strings.NewReader(""), &stdout, &stderr)

	if code != 5 || !strings.Contains(stderr.String(), "HERDR_PLUGIN_CONTEXT_JSON") {
		t.Fatalf("run() code = %d, stderr = %q", code, stderr.String())
	}
}

func TestRunVersionAndInvalidCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"version"}, func(string) string { return "" }, strings.NewReader(""), &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "herdr-logbook dev") {
		t.Fatalf("version code = %d, stdout = %q", code, stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"unknown"}, func(string) string { return "" }, strings.NewReader(""), &stdout, &stderr); code != 2 {
		t.Fatalf("invalid command code = %d, want 2", code)
	}
}

func TestRunResolveCWDUsesFocusedPane(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"resolve-cwd"}, func(key string) string {
		if key == "HERDR_PLUGIN_CONTEXT_JSON" {
			return `{"workspace_cwd":"/tmp/workspace","focused_pane_cwd":"/tmp/focused"}`
		}
		return ""
	}, strings.NewReader(""), &stdout, &stderr)

	if code != 0 || stdout.String() != "/tmp/focused\n" {
		t.Fatalf("resolve-cwd code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
}

func TestRunInitCentralIsIdempotentAndDoesNotWriteRepository(t *testing.T) {
	repo := t.TempDir()
	stateDir := t.TempDir()
	configDir := t.TempDir()
	env := map[string]string{"HERDR_PLUGIN_STATE_DIR": stateDir, "HERDR_PLUGIN_CONFIG_DIR": configDir}
	getenv := func(key string) string { return env[key] }
	var stdout, stderr bytes.Buffer

	code := run([]string{"init", "--storage", "central", "--project-root", repo}, getenv, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("init code = %d, stderr = %q", code, stderr.String())
	}
	projects, err := os.ReadDir(filepath.Join(stateDir, "store", "projects"))
	if err != nil || len(projects) != 1 {
		t.Fatalf("central projects = %#v, %v", projects, err)
	}
	store := filepath.Join(stateDir, "store", "projects", projects[0].Name())
	nowPath := filepath.Join(store, "now.md")
	if _, err := os.Stat(filepath.Join(store, "project.toml")); err != nil {
		t.Fatalf("project.toml missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, ".herdr")); !os.IsNotExist(err) {
		t.Fatalf("central init wrote repository: %v", err)
	}
	if err := os.WriteFile(nowPath, []byte("# Keep me\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"init", "--storage", "central", "--project-root", repo}, getenv, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("second init code = %d, stderr = %q", code, stderr.String())
	}
	data, err := os.ReadFile(nowPath)
	if err != nil || string(data) != "# Keep me\n" {
		t.Fatalf("second init now.md = %q, %v", data, err)
	}
}

func TestRunInitRepoPrintsIgnoreGuidance(t *testing.T) {
	repo := t.TempDir()
	env := map[string]string{"HERDR_PLUGIN_STATE_DIR": t.TempDir(), "HERDR_PLUGIN_CONFIG_DIR": t.TempDir()}
	var stdout, stderr bytes.Buffer
	code := run([]string{"init", "--storage", "repo", "--project-root", repo}, func(key string) string { return env[key] }, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("init repo code = %d, stderr = %q", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(repo, ".herdr", "logbook", "now.md")); err != nil {
		t.Fatalf("repo now.md missing: %v", err)
	}
	if !strings.Contains(stdout.String(), ".herdr/logbook/") || !strings.Contains(stdout.String(), "may be committed") {
		t.Fatalf("repo init output = %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(repo, ".gitignore")); !os.IsNotExist(err) {
		t.Fatalf("repo init modified .gitignore: %v", err)
	}
}

func TestRunInitRepoGuidanceUsesConfiguredDirectory(t *testing.T) {
	repo := t.TempDir()
	configDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte("[storage]\nrepo_directory = \"private-memory\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"HERDR_PLUGIN_STATE_DIR": t.TempDir(), "HERDR_PLUGIN_CONFIG_DIR": configDir}
	var stdout, stderr bytes.Buffer
	code := run([]string{"init", "--storage", "repo", "--project-root", repo}, func(key string) string { return env[key] }, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("init repo code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "private-memory/") || strings.Contains(stdout.String(), ".herdr/logbook/") {
		t.Fatalf("repo init output = %q", stdout.String())
	}
}

func TestRunPathsJSONReportsResolvedStore(t *testing.T) {
	repo := t.TempDir()
	stateDir := t.TempDir()
	env := map[string]string{"HERDR_PLUGIN_STATE_DIR": stateDir, "HERDR_PLUGIN_CONFIG_DIR": t.TempDir()}
	var stdout, stderr bytes.Buffer
	code := run([]string{"paths", "--json", "--project-root", repo}, func(key string) string { return env[key] }, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("paths code = %d, stderr = %q", code, stderr.String())
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("paths JSON = %q: %v", stdout.String(), err)
	}
	if result["state_dir"] != stateDir || !strings.Contains(result["project_store"].(string), filepath.Join("store", "projects", "p_")) {
		t.Fatalf("paths result = %#v", result)
	}
}

func TestRunDoctorJSONRedactsSelectionAndRemoteCredentials(t *testing.T) {
	repo := t.TempDir()
	gitCommand(t, repo, "init")
	gitCommand(t, repo, "remote", "add", "origin", "https://user:secret@GitHub.COM/Org/Repo.git?token=secret")
	contextJSON, err := json.Marshal(map[string]string{
		"focused_pane_cwd": repo,
		"selected_text":    "private terminal output",
	})
	if err != nil {
		t.Fatal(err)
	}
	env := map[string]string{
		"HERDR_PLUGIN_STATE_DIR":    t.TempDir(),
		"HERDR_PLUGIN_CONFIG_DIR":   t.TempDir(),
		"HERDR_BIN_PATH":            filepath.Join(t.TempDir(), "missing-herdr"),
		"HERDR_PLUGIN_CONTEXT_JSON": string(contextJSON),
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{"doctor", "--json", "--project-root", repo}, func(key string) string { return env[key] }, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("doctor code = %d, stderr = %q", code, stderr.String())
	}
	for _, secret := range []string{"private terminal output", "user:secret", "token=secret"} {
		if strings.Contains(stdout.String(), secret) || strings.Contains(stderr.String(), secret) {
			t.Fatalf("doctor leaked %q: stdout=%s stderr=%s", secret, stdout.String(), stderr.String())
		}
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("doctor JSON = %q: %v", stdout.String(), err)
	}
	project := result["project"].(map[string]any)
	if project["remote_fingerprint"] != "github.com/Org/Repo" {
		t.Fatalf("doctor project = %#v", project)
	}
}

func gitCommand(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}

func TestWaitForCloseKeepsPaneAliveWhenStdinIsEOF(t *testing.T) {
	started := time.Now()
	waitForClose(strings.NewReader(""), 20*time.Millisecond)
	if elapsed := time.Since(started); elapsed < 15*time.Millisecond {
		t.Fatalf("waitForClose() returned after %s on EOF", elapsed)
	}
}

func TestWaitForCloseHonorsMinimumWhenInputArrivesImmediately(t *testing.T) {
	started := time.Now()
	waitForClose(strings.NewReader("\n"), 20*time.Millisecond)
	if elapsed := time.Since(started); elapsed < 15*time.Millisecond {
		t.Fatalf("waitForClose() returned after %s on early input", elapsed)
	}
}

func TestRunCaptureProjectTextWritesMonthlyInbox(t *testing.T) {
	repo := t.TempDir()
	stateDir := t.TempDir()
	env := map[string]string{"HERDR_PLUGIN_STATE_DIR": stateDir, "HERDR_PLUGIN_CONFIG_DIR": t.TempDir()}
	var stdout, stderr bytes.Buffer
	code := run([]string{"capture", "--scope", "project", "--text", "remember this", "--project-root", repo, "--branch", "feature/login", "--source-cwd", "/workspace/api"}, func(key string) string { return env[key] }, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("capture code = %d, stderr = %q", code, stderr.String())
	}
	path := strings.TrimSpace(stdout.String())
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"remember this", "feature/login", "/workspace/api"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("capture data = %q, want %q", data, want)
		}
	}
}

func TestRunNowSetsTaskAndArchivesThePreviousOne(t *testing.T) {
	repo := t.TempDir()
	stateDir := t.TempDir()
	env := map[string]string{"HERDR_PLUGIN_STATE_DIR": stateDir, "HERDR_PLUGIN_CONFIG_DIR": t.TempDir()}
	getenv := func(key string) string { return env[key] }
	logbook := func(args ...string) (int, string, string) {
		var stdout, stderr bytes.Buffer
		code := run(args, getenv, strings.NewReader(""), &stdout, &stderr)
		return code, stdout.String(), stderr.String()
	}

	code, _, stderr := logbook("now", "--project-root", repo)
	if code != 0 || !strings.Contains(stderr, "no current task set") {
		t.Fatalf("empty now code = %d, stderr = %q", code, stderr)
	}

	code, stdout, stderr := logbook("now", "--project-root", repo, "Rotate the signing tokens")
	if code != 0 {
		t.Fatalf("set now code = %d, stderr = %q", code, stderr)
	}
	if strings.Contains(stdout, "archived:") {
		t.Fatalf("first task should not archive anything: %q", stdout)
	}
	nowPath := strings.TrimSpace(stdout)

	code, stdout, stderr = logbook("now", "--project-root", repo)
	if code != 0 || strings.TrimSpace(stdout) != "Rotate the signing tokens" {
		t.Fatalf("read now code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}

	code, stdout, stderr = logbook("now", "--project-root", repo, "Write the release notes")
	if code != 0 || !strings.Contains(stdout, "archived: Rotate the signing tokens") {
		t.Fatalf("switch now code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}

	data, err := os.ReadFile(nowPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "Write the release notes") || strings.Contains(content, "Rotate the signing tokens") {
		t.Fatalf("now.md = %q", content)
	}
	for _, want := range []string{"## Next steps", "## Blockers", "## Context"} {
		if !strings.Contains(content, want) {
			t.Fatalf("now.md dropped %q:\n%s", want, content)
		}
	}

	inbox := filepath.Join(filepath.Dir(nowPath), "inbox", time.Now().Format("2006-01")+".md")
	journal, err := os.ReadFile(inbox)
	if err != nil {
		t.Fatalf("read work journal: %v", err)
	}
	if !strings.Contains(string(journal), "Task done: Rotate the signing tokens") {
		t.Fatalf("work journal = %q", journal)
	}
}

func TestRunNowRejectsHeadingsAndEmptyInput(t *testing.T) {
	repo := t.TempDir()
	env := map[string]string{"HERDR_PLUGIN_STATE_DIR": t.TempDir(), "HERDR_PLUGIN_CONFIG_DIR": t.TempDir()}
	var stdout, stderr bytes.Buffer
	code := run([]string{"now", "--project-root", repo, "## Blockers"}, func(key string) string { return env[key] }, strings.NewReader(""), &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "headings") {
		t.Fatalf("heading task code = %d, stderr = %q", code, stderr.String())
	}
}

func TestRunCaptureGlobalReadsStdin(t *testing.T) {
	stateDir := t.TempDir()
	env := map[string]string{"HERDR_PLUGIN_STATE_DIR": stateDir, "HERDR_PLUGIN_CONFIG_DIR": t.TempDir()}
	var stdout, stderr bytes.Buffer
	code := run([]string{"capture", "--scope", "global", "--stdin", "--project-root", t.TempDir()}, func(key string) string { return env[key] }, strings.NewReader("global note"), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("capture code = %d, stderr = %q", code, stderr.String())
	}
	path := strings.TrimSpace(stdout.String())
	if !strings.Contains(path, filepath.Join("store", "global", "inbox")) {
		t.Fatalf("global capture path = %q", path)
	}
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), "global note") {
		t.Fatalf("global capture data = %q, %v", data, err)
	}
}

func TestRunCaptureSelectedUsesRedactedContextOnlyForCapture(t *testing.T) {
	repo := t.TempDir()
	stateDir := t.TempDir()
	contextJSON, err := json.Marshal(map[string]string{
		"focused_pane_cwd": repo,
		"selected_text":    "line one\n```\nline two",
	})
	if err != nil {
		t.Fatal(err)
	}
	env := map[string]string{
		"HERDR_PLUGIN_STATE_DIR": stateDir, "HERDR_PLUGIN_CONFIG_DIR": t.TempDir(),
		"HERDR_PLUGIN_CONTEXT_JSON": string(contextJSON),
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{"capture", "--selected", "--project-root", repo}, func(key string) string { return env[key] }, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("capture code = %d, stderr = %q", code, stderr.String())
	}
	data, err := os.ReadFile(strings.TrimSpace(stdout.String()))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Terminal capture") || !strings.Contains(string(data), "````text") {
		t.Fatalf("selected capture data = %q", data)
	}
}

func TestRunCaptureRejectsConflictingInputs(t *testing.T) {
	env := map[string]string{"HERDR_PLUGIN_STATE_DIR": t.TempDir(), "HERDR_PLUGIN_CONFIG_DIR": t.TempDir()}
	var stdout, stderr bytes.Buffer
	code := run([]string{"capture", "--text", "note", "--stdin", "--project-root", t.TempDir()}, func(key string) string { return env[key] }, strings.NewReader("stdin"), &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "exactly one") {
		t.Fatalf("capture code = %d, stderr = %q", code, stderr.String())
	}
}

func TestRunCaptureWithoutSourceOpensTextareaAndSaves(t *testing.T) {
	repo := t.TempDir()
	stateDir := t.TempDir()
	env := map[string]string{"HERDR_PLUGIN_STATE_DIR": stateDir, "HERDR_PLUGIN_CONFIG_DIR": t.TempDir()}
	var stdout, stderr bytes.Buffer

	code := run(
		[]string{"capture", "--project-root", repo},
		func(key string) string { return env[key] },
		strings.NewReader("interactive note\x13"),
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf("interactive capture code = %d, stderr = %q", code, stderr.String())
	}

	matches, err := filepath.Glob(filepath.Join(stateDir, "store", "projects", "*", "inbox", "*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("interactive capture files = %v, %v", matches, err)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil || !strings.Contains(string(data), "interactive note") {
		t.Fatalf("interactive capture data = %q, %v", data, err)
	}
}

func TestRunDecisionCreatesTemplate(t *testing.T) {
	repo := t.TempDir()
	stateDir := t.TempDir()
	env := map[string]string{"HERDR_PLUGIN_STATE_DIR": stateDir, "HERDR_PLUGIN_CONFIG_DIR": t.TempDir()}
	var stdout, stderr bytes.Buffer
	code := run(
		[]string{"decision", "--title", "Use Redis", "--no-edit", "--project-root", repo},
		func(key string) string { return env[key] },
		strings.NewReader(""), &stdout, &stderr,
	)
	if code != 0 {
		t.Fatalf("decision code = %d, stderr = %q", code, stderr.String())
	}
	path := strings.TrimSpace(stdout.String())
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), "# Decision: Use Redis") {
		t.Fatalf("decision data = %q, %v", data, err)
	}
}

func TestRunTUIRejectsInvalidView(t *testing.T) {
	env := map[string]string{"HERDR_PLUGIN_STATE_DIR": t.TempDir(), "HERDR_PLUGIN_CONFIG_DIR": t.TempDir()}
	var stdout, stderr bytes.Buffer
	code := run([]string{"tui", "--view", "sideways", "--project-root", t.TempDir()}, func(key string) string { return env[key] }, strings.NewReader("q"), &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "now, project, global, or all") {
		t.Fatalf("tui code = %d, stderr = %q", code, stderr.String())
	}
}

func TestRunTUIStartsAndExitsOnQ(t *testing.T) {
	env := map[string]string{"HERDR_PLUGIN_STATE_DIR": t.TempDir(), "HERDR_PLUGIN_CONFIG_DIR": t.TempDir()}
	var stdout, stderr bytes.Buffer
	code := run([]string{"tui", "--view", "now", "--project-root", t.TempDir()}, func(key string) string { return env[key] }, strings.NewReader("q"), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("tui code = %d, stderr = %q", code, stderr.String())
	}
}

func TestRunIndexRebuildScansRegisteredStores(t *testing.T) {
	repo := t.TempDir()
	stateDir := t.TempDir()
	env := map[string]string{"HERDR_PLUGIN_STATE_DIR": stateDir, "HERDR_PLUGIN_CONFIG_DIR": t.TempDir()}
	getenv := func(key string) string { return env[key] }
	var stdout, stderr bytes.Buffer
	if code := run([]string{"init", "--project-root", repo}, getenv, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("init code = %d, stderr = %q", code, stderr.String())
	}
	projects, err := os.ReadDir(filepath.Join(stateDir, "store", "projects"))
	if err != nil || len(projects) != 1 {
		t.Fatalf("project stores = %v, %v", projects, err)
	}
	notePath := filepath.Join(stateDir, "store", "projects", projects[0].Name(), "notes", "cache.md")
	if err := os.WriteFile(notePath, []byte("# Cache policy\nBound entries."), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"index", "rebuild", "--project-root", repo}, getenv, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("index rebuild code = %d, stderr = %q", code, stderr.String())
	}
	cachePath := filepath.Join(stateDir, "cache", "index-v1.json")
	data, err := os.ReadFile(cachePath)
	if err != nil || !strings.Contains(string(data), "Cache policy") {
		t.Fatalf("cache data = %q, %v", data, err)
	}
}
