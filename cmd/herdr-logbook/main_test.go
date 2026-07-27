package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Resetnak/herdr-logbook/internal/project"
	"github.com/Resetnak/herdr-logbook/internal/storage"
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

func TestRunDecisionPromptsForATitleAndReportsEditorFailures(t *testing.T) {
	newEnv := func(extra map[string]string) func(string) string {
		env := map[string]string{"HERDR_PLUGIN_STATE_DIR": t.TempDir(), "HERDR_PLUGIN_CONFIG_DIR": t.TempDir()}
		maps.Copy(env, extra)
		return func(key string) string { return env[key] }
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"decision", "--no-edit", "--project-root", t.TempDir()},
		newEnv(nil), strings.NewReader("Use Redis\n"), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("prompted decision code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Decision title: ") {
		t.Fatalf("no title prompt in stdout: %q", stdout.String())
	}
	path := strings.TrimSpace(strings.TrimPrefix(stdout.String(), "Decision title: "))
	if data, err := os.ReadFile(path); err != nil || !strings.Contains(string(data), "# Decision: Use Redis") {
		t.Fatalf("decision data = %q, %v", data, err)
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"decision", "--no-edit", "--project-root", t.TempDir()},
		newEnv(nil), strings.NewReader("   \n"), &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "decision title is required") {
		t.Fatalf("blank title code = %d, stderr = %q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"decision", "--title", "Use Redis", "leftover"}, newEnv(nil), strings.NewReader(""), &stdout, &stderr)
	if code != 2 {
		t.Fatalf("positional argument code = %d, want 2", code)
	}

	// Without --no-edit the resolved editor is executed; an unresolvable one is exit 6.
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"decision", "--title", "Use Redis", "--project-root", t.TempDir()},
		newEnv(map[string]string{"EDITOR": "herdr-logbook-no-such-editor"}), strings.NewReader(""), &stdout, &stderr)
	if code != 6 {
		t.Fatalf("missing editor code = %d, stderr = %q", code, stderr.String())
	}

	if runtime.GOOS != "windows" {
		stdout.Reset()
		stderr.Reset()
		code = run([]string{"decision", "--title", "Use Redis", "--project-root", t.TempDir()},
			newEnv(map[string]string{"EDITOR": "true"}), strings.NewReader(""), &stdout, &stderr)
		if code != 0 {
			t.Fatalf("editor decision code = %d, stderr = %q", code, stderr.String())
		}
		if data, err := os.ReadFile(strings.TrimSpace(stdout.String())); err != nil || !strings.Contains(string(data), "# Decision: Use Redis") {
			t.Fatalf("edited decision data = %q, %v", data, err)
		}
	}
}

func TestPrintKeybindsListsEveryHubAction(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"keybinds"}, func(string) string { return "" }, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("keybinds code = %d, stderr = %q", code, stderr.String())
	}
	for _, want := range []string{"Navigation:", "Actions:", "t    set the current task", "Ctrl+S", "e    edit in external editor"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("keybinds output missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestPathStateDistinguishesMissingAvailableAndReadOnly(t *testing.T) {
	dir := t.TempDir()
	if got := pathState(filepath.Join(dir, "gone")); got != "missing" {
		t.Fatalf("missing path = %q", got)
	}
	if got := pathState(dir); got != "available" {
		t.Fatalf("writable dir = %q", got)
	}

	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		return // directory permission bits are not enforced here
	}
	readOnly := filepath.Join(dir, "read-only")
	if err := os.Mkdir(readOnly, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnly, 0o700) })
	if got := pathState(readOnly); got != "read-only" {
		t.Fatalf("read-only dir = %q", got)
	}
}

// hubState builds the coreState the Hub's action closures are given, so the n/d/t
// and e paths can be exercised without a terminal.
func hubState(t *testing.T, extraEnv map[string]string) (coreState, func(string) string) {
	t.Helper()
	env := map[string]string{"HERDR_PLUGIN_STATE_DIR": t.TempDir(), "HERDR_PLUGIN_CONFIG_DIR": t.TempDir()}
	maps.Copy(env, extraEnv)
	getenv := func(key string) string { return env[key] }
	state, failure := loadCore("", t.TempDir(), "", getenv)
	if failure != nil {
		t.Fatalf("loadCore: %v", failure.err)
	}
	if err := storage.WithLock(state.Layout.Lock, 2*time.Second, func() error { return storage.Initialize(state.Layout) }); err != nil {
		t.Fatal(err)
	}
	return state, getenv
}

func TestAuthorFromHubWritesNotesDecisionsAndTheCurrentTask(t *testing.T) {
	state, _ := hubState(t, nil)

	if err := authorFromHub(state, "note", "Cache Policy"); err != nil {
		t.Fatalf("note: %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(state.Layout.Notes, "cache-policy.md")); err != nil || !strings.Contains(string(data), "# Cache Policy") {
		t.Fatalf("note = %q, %v", data, err)
	}

	if err := authorFromHub(state, "decision", "Use Redis"); err != nil {
		t.Fatalf("decision: %v", err)
	}
	decisions, err := filepath.Glob(filepath.Join(state.Layout.Decisions, "*use-redis.md"))
	if err != nil || len(decisions) != 1 {
		t.Fatalf("decisions = %v, %v", decisions, err)
	}

	if err := authorFromHub(state, "now", "Rotate the signing tokens"); err != nil {
		t.Fatalf("now: %v", err)
	}
	if err := authorFromHub(state, "now", "Ship the release"); err != nil {
		t.Fatalf("second now: %v", err)
	}
	nowData, err := os.ReadFile(state.Layout.Now)
	if err != nil || !strings.Contains(string(nowData), "Ship the release") {
		t.Fatalf("now.md = %q, %v", nowData, err)
	}
	// Switching tasks must have filed the displaced one into the monthly inbox.
	inbox := readAll(t, mustGlob(t, filepath.Join(state.Layout.Inbox, "*.md")))
	if !strings.Contains(inbox, "Task done: Rotate the signing tokens") {
		t.Fatalf("previous task was not archived:\n%s", inbox)
	}

	if err := authorFromHub(state, "now", "  "); err == nil {
		t.Fatal("authorFromHub accepted an empty task")
	}
	if err := authorFromHub(state, "note", "###"); err == nil {
		t.Fatal("authorFromHub accepted an unusable note title")
	}
}

func TestEditorCommandForGuardsThePathAndResolvesTheEditor(t *testing.T) {
	state, getenv := hubState(t, map[string]string{"EDITOR": "logbook-test-editor"})
	editorPath := fakeEditor(t, "logbook-test-editor")

	note := filepath.Join(state.Layout.Notes, "cache.md")
	if err := os.WriteFile(note, []byte("# Cache\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	command, err := editorCommandFor(state, nil, getenv, note)
	if err != nil {
		t.Fatalf("editorCommandFor: %v", err)
	}
	if command.Path != editorPath || command.Args[len(command.Args)-1] != note {
		t.Fatalf("command = %q %q", command.Path, command.Args)
	}

	// An explicit argv overrides $EDITOR and keeps its own arguments.
	command, err = editorCommandFor(state, []string{editorPath, "--wait"}, getenv, note)
	if err != nil {
		t.Fatalf("explicit argv: %v", err)
	}
	if len(command.Args) != 3 || command.Args[1] != "--wait" || command.Args[2] != note {
		t.Fatalf("explicit argv command = %q", command.Args)
	}

	if _, err := editorCommandFor(state, nil, getenv, filepath.Join(t.TempDir(), "outside.md")); err == nil {
		t.Fatal("editorCommandFor accepted a note outside every store")
	}
	if _, err := editorCommandFor(state, []string{"herdr-logbook-no-such-editor"}, getenv, note); err == nil {
		t.Fatal("editorCommandFor accepted an unresolvable editor")
	}
}

// fakeEditor puts a no-op executable named name on PATH and returns its path.
func fakeEditor(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	script := "#!/bin/sh\nexit 0\n"
	if runtime.GOOS == "windows" {
		path += ".bat"
		script = "@echo off\r\n"
	}
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	resolved, err := exec.LookPath(name)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func mustGlob(t *testing.T, pattern string) []string {
	t.Helper()
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		t.Fatalf("glob %q = %v, %v", pattern, matches, err)
	}
	return matches
}

func readAll(t *testing.T, paths []string) string {
	t.Helper()
	var combined strings.Builder
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		combined.Write(data)
	}
	return combined.String()
}

// Exit codes are part of the contract Herdr reads: 2 usage, 3 state resolution,
// 5 Herdr context.
func TestRunRejectsBadInvocationsWithTheDocumentedExitCodes(t *testing.T) {
	stateDir := t.TempDir()
	env := map[string]string{"HERDR_PLUGIN_STATE_DIR": stateDir, "HERDR_PLUGIN_CONFIG_DIR": t.TempDir()}
	getenv := func(key string) string { return env[key] }

	usage := [][]string{
		{},
		{"bogus"},
		{"version", "extra"},
		{"resolve-cwd", "extra"},
		{"keybinds", "extra"},
		{"index"},
		{"index", "bogus"},
		{"index", "rebuild", "--nope"},
		{"capture", "--nope"},
		{"capture", "--scope", "sideways"},
		{"capture", "--text", "   "},
		{"decision", "--nope"},
		{"init", "--nope"},
		{"paths", "extra"},
		{"doctor", "extra"},
		{"now", "--nope"},
		{"tui", "extra"},
		{"compatibility", "--bogus"},
	}
	for _, args := range usage {
		var stdout, stderr bytes.Buffer
		if code := run(args, getenv, strings.NewReader(""), &stdout, &stderr); code != 2 {
			t.Fatalf("run(%q) = %d, want 2 (stderr %q)", args, code, stderr.String())
		}
	}

	// loadCore cannot resolve a store without the state directory Herdr provides.
	for _, args := range [][]string{{"paths"}, {"doctor"}, {"now"}} {
		var stdout, stderr bytes.Buffer
		code := run(args, func(string) string { return "" }, strings.NewReader(""), &stdout, &stderr)
		if code != 3 || !strings.Contains(stderr.String(), "HERDR_PLUGIN_STATE_DIR") {
			t.Fatalf("run(%q) without a state dir = %d, stderr %q", args, code, stderr.String())
		}
	}

	brokenContext := func(key string) string {
		if key == "HERDR_PLUGIN_CONTEXT_JSON" {
			return "{broken"
		}
		return env[key]
	}
	for _, args := range [][]string{{"resolve-cwd"}, {"capture", "--selected"}} {
		var stdout, stderr bytes.Buffer
		if code := run(args, brokenContext, strings.NewReader(""), &stdout, &stderr); code != 5 {
			t.Fatalf("run(%q) with a broken context = %d, want 5 (stderr %q)", args, code, stderr.String())
		}
	}
}

func TestRunEnforcesTheConfiguredCaptureSizeLimit(t *testing.T) {
	configDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte("[capture]\nmax_selection_bytes = 8\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"HERDR_PLUGIN_STATE_DIR": t.TempDir(), "HERDR_PLUGIN_CONFIG_DIR": configDir}
	getenv := func(key string) string { return env[key] }
	repo := t.TempDir()

	var stdout, stderr bytes.Buffer
	code := run([]string{"capture", "--text", "far too long to fit", "--project-root", repo}, getenv, strings.NewReader(""), &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "exceeds limit") {
		t.Fatalf("oversize capture = %d, stderr %q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"now", "--project-root", repo, "far too long to fit"}, getenv, strings.NewReader(""), &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "exceeds limit") {
		t.Fatalf("oversize task = %d, stderr %q", code, stderr.String())
	}
}

func TestRunCaptureReadsStdinAndRunIndexRebuildsTheCache(t *testing.T) {
	stateDir := t.TempDir()
	repo := t.TempDir()
	env := map[string]string{"HERDR_PLUGIN_STATE_DIR": stateDir, "HERDR_PLUGIN_CONFIG_DIR": t.TempDir()}
	getenv := func(key string) string { return env[key] }

	var stdout, stderr bytes.Buffer
	code := run([]string{"capture", "--stdin", "--project-root", repo}, getenv, strings.NewReader("piped from a pipeline\n"), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("stdin capture = %d, stderr %q", code, stderr.String())
	}
	inbox := readAll(t, mustGlob(t, filepath.Join(stateDir, "store", "projects", "*", "inbox", "*.md")))
	if !strings.Contains(inbox, "piped from a pipeline") {
		t.Fatalf("stdin capture missing from the inbox:\n%s", inbox)
	}

	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"index", "rebuild", "--project-root", repo}, getenv, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("index rebuild = %d, stderr %q", code, stderr.String())
	}
	cache, err := os.ReadFile(filepath.Join(stateDir, "cache", "index-v1.json"))
	if err != nil || !strings.Contains(string(cache), "piped from a pipeline") {
		t.Fatalf("rebuilt cache = %v", err)
	}
}

func TestRunPathsAndDoctorRenderPlainTextReports(t *testing.T) {
	repo := t.TempDir()
	emptyPath := t.TempDir() // no herdr on PATH, so inspectHerdr must report it as missing
	t.Setenv("PATH", emptyPath)
	env := map[string]string{"HERDR_PLUGIN_STATE_DIR": t.TempDir(), "HERDR_PLUGIN_CONFIG_DIR": t.TempDir()}
	getenv := func(key string) string { return env[key] }

	var stdout, stderr bytes.Buffer
	if code := run([]string{"paths", "--project-root", repo}, getenv, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("paths = %d, stderr %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), filepath.Join("store", "projects")) {
		t.Fatalf("paths text report = %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"doctor", "--project-root", repo}, getenv, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("doctor = %d, stderr %q", code, stderr.String())
	}
	for _, want := range []string{"Herdr Logbook", "project:", "store:", "editor:", "cache:"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("doctor text report missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestWaitForCloseReturnsAfterTheTimeout(t *testing.T) {
	start := time.Now()
	waitForClose(strings.NewReader("\n"), 20*time.Millisecond)
	if elapsed := time.Since(start); elapsed < 20*time.Millisecond {
		t.Fatalf("waitForClose returned after %v, want at least the full timeout", elapsed)
	}
	// A reader that never delivers a line must still be released by the timer.
	waitForClose(strings.NewReader(""), 20*time.Millisecond)
}

// firstLine shortens a multi-line task for the "archived: …" confirmation, marking
// the elision with an ellipsis; single-line input is passed through untouched.
func TestFirstLineTruncatesAtTheFirstNewline(t *testing.T) {
	cases := map[string]string{
		"Rotate the signing tokens":        "Rotate the signing tokens",
		"Rotate the tokens\nand the certs": "Rotate the tokens …",
		"\nleading newline":                " …",
	}
	for input, want := range cases {
		if got := firstLine(input); got != want {
			t.Fatalf("firstLine(%q) = %q, want %q", input, got, want)
		}
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

func TestValidateEditableNoteAcceptsOnlyMarkdownInsideRegisteredStores(t *testing.T) {
	stateDir := t.TempDir()
	projectRoot := filepath.Join(stateDir, "store", "projects", "api")
	globalRoot := filepath.Join(stateDir, "store", "global")
	registered := filepath.Join(t.TempDir(), "external-store")
	outside := t.TempDir()

	for _, dir := range []string{filepath.Join(projectRoot, "notes"), globalRoot, registered, outside} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}

	registryPath, registryLock := registryPaths(stateDir)
	err := project.UpdateRegistry(registryPath, registryLock, 2*time.Second,
		project.Project{ID: "external", Name: "external", Root: registered},
		"central", registered, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	state := coreState{StateDir: stateDir, Layout: storage.Layout{Root: projectRoot}}

	write := func(path string) string {
		if err := os.WriteFile(path, []byte("# Note\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}

	accepted := map[string]string{
		"project store": write(filepath.Join(projectRoot, "notes", "note.md")),
		"global store":  write(filepath.Join(globalRoot, "note.md")),
		"registered":    write(filepath.Join(registered, "note.md")),
		"uppercase ext": write(filepath.Join(projectRoot, "notes", "note.MD")),
	}
	for name, path := range accepted {
		if err := validateEditableNote(state, path); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}

	rejected := map[string]struct{ path, want string }{
		"missing":       {filepath.Join(projectRoot, "notes", "gone.md"), "open note for editing"},
		"directory":     {filepath.Join(projectRoot, "notes"), "not a regular Markdown file"},
		"not markdown":  {write(filepath.Join(projectRoot, "notes", "note.txt")), "not a regular Markdown file"},
		"outside roots": {write(filepath.Join(outside, "note.md")), "outside registered memory stores"},
		"traversal":     {filepath.Join(projectRoot, "notes", "..", "..", "..", "..", "escape.md"), "open note for editing"},
	}
	for name, test := range rejected {
		err := validateEditableNote(state, test.path)
		if err == nil {
			t.Fatalf("%s: expected an error", name)
		}
		if !strings.Contains(err.Error(), test.want) {
			t.Fatalf("%s: error = %v, want %q", name, err, test.want)
		}
	}

	if runtime.GOOS != "windows" {
		link := filepath.Join(projectRoot, "notes", "link.md")
		if err := os.Symlink(accepted["project store"], link); err != nil {
			t.Fatal(err)
		}
		if err := validateEditableNote(state, link); err == nil {
			t.Fatal("symlink was accepted for editing")
		}
	}
}

func TestAppendCaptureGuardsEmptyTextAndRoutesTheGlobalInbox(t *testing.T) {
	state, _ := hubState(t, nil)

	if _, err := appendCapture(state, "   \n\t", false, false, "", ""); err == nil ||
		!strings.Contains(err.Error(), "capture text is empty") {
		t.Fatalf("appendCapture() empty error = %v", err)
	}

	path, err := appendCapture(state, "global thought", true, false, "", "")
	if err != nil {
		t.Fatal(err)
	}
	globalInbox := filepath.Join(state.StateDir, "store", "global", "inbox")
	if filepath.Dir(path) != globalInbox {
		t.Fatalf("global capture landed in %q, want a file under %q", path, globalInbox)
	}
	if !strings.Contains(readAll(t, []string{path}), "global thought") {
		t.Fatalf("global inbox %q is missing the capture", path)
	}
}

// failingWriter stands in for a closed pipe: every diagnostic must report the
// broken stdout instead of exiting successfully.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("stdout is closed") }

func TestJSONReportsFailLoudlyWhenStdoutIsBroken(t *testing.T) {
	stateDir := t.TempDir()
	getenv := func(key string) string {
		if key == "HERDR_PLUGIN_STATE_DIR" {
			return stateDir
		}
		return ""
	}
	root := t.TempDir()

	for _, args := range [][]string{
		{"compatibility"},
		{"paths", "--json", "--project-root", root},
		{"doctor", "--json", "--project-root", root},
	} {
		var stderr bytes.Buffer
		if code := run(args, getenv, strings.NewReader(""), failingWriter{}, &stderr); code != 1 {
			t.Fatalf("%v with a broken stdout = %d, stderr = %q", args, code, stderr.String())
		}
	}

	// index rebuild must surface the missing state directory as a context failure.
	var stderr bytes.Buffer
	if code := run([]string{"index", "rebuild", "--project-root", root},
		func(string) string { return "" }, strings.NewReader(""), &bytes.Buffer{}, &stderr); code != 3 {
		t.Fatalf("index rebuild without a state directory = %d", code)
	}
}
