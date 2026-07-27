package herdr

import (
	"reflect"
	"strings"
	"testing"
)

func TestReadContextSummarizesHerdrEnvironmentWithoutSensitiveContent(t *testing.T) {
	env := map[string]string{
		"HERDR_BIN_PATH":            "/opt/herdr/bin/herdr",
		"HERDR_PLUGIN_ROOT":         "/tmp/plugin with spaces",
		"HERDR_PLUGIN_CONFIG_DIR":   "/tmp/config",
		"HERDR_PLUGIN_STATE_DIR":    "/tmp/state",
		"HERDR_WORKSPACE_ID":        "workspace-1",
		"HERDR_TAB_ID":              "tab-1",
		"HERDR_PANE_ID":             "pane-1",
		"HERDR_PLUGIN_CONTEXT_JSON": `{"workspace_cwd":"/work/žluťoučký kůň","focused_pane_cwd":"/work/api","selected_text":"secret output","clicked_urls":["https://example.test/?token=secret"],"invocation_source":"action"}`,
	}

	got, err := ReadContext(func(key string) string { return env[key] })
	if err != nil {
		t.Fatalf("ReadContext() error = %v", err)
	}

	if got.Paths.PluginRoot != "/tmp/plugin with spaces" || got.FocusedPaneCWD != "/work/api" {
		t.Fatalf("ReadContext() paths = %#v, focused cwd = %q", got.Paths, got.FocusedPaneCWD)
	}
	if got.SelectedTextPresent != true {
		t.Fatal("ReadContext() did not report selected text presence")
	}
	wantKeys := []string{"clicked_urls", "focused_pane_cwd", "invocation_source", "selected_text", "workspace_cwd"}
	if !reflect.DeepEqual(got.ContextKeys, wantKeys) {
		t.Fatalf("ReadContext() keys = %#v, want %#v", got.ContextKeys, wantKeys)
	}
	if got.SelectedText != "" || got.ClickedURLs != nil {
		t.Fatalf("ReadContext() retained sensitive context: %#v", got)
	}
}

func TestReadContextRejectsMalformedJSON(t *testing.T) {
	_, err := ReadContext(func(key string) string {
		if key == "HERDR_PLUGIN_CONTEXT_JSON" {
			return "{broken"
		}
		return ""
	})
	if err == nil {
		t.Fatal("ReadContext() error = nil, want malformed JSON error")
	}
}

func TestResolveWorkingDirectoryPriority(t *testing.T) {
	ctx := Context{
		WorktreePath:   "/work/worktree",
		FocusedPaneCWD: "/work/focused",
		WorkspaceCWD:   "/work/workspace",
	}

	if got := ResolveWorkingDirectory(ctx, "/fallback"); got != "/work/worktree" {
		t.Fatalf("ResolveWorkingDirectory() = %q, want worktree", got)
	}
	ctx.WorktreePath = ""
	if got := ResolveWorkingDirectory(ctx, "/fallback"); got != "/work/focused" {
		t.Fatalf("ResolveWorkingDirectory() = %q, want focused pane", got)
	}
	ctx.FocusedPaneCWD = ""
	if got := ResolveWorkingDirectory(ctx, "/fallback"); got != "/work/workspace" {
		t.Fatalf("ResolveWorkingDirectory() = %q, want workspace", got)
	}
	ctx.WorkspaceCWD = ""
	if got := ResolveWorkingDirectory(ctx, "/fallback"); got != "/fallback" {
		t.Fatalf("ResolveWorkingDirectory() = %q, want fallback", got)
	}
}

func TestSelectedTextReadsOnlyExplicitCaptureValue(t *testing.T) {
	got, err := SelectedText(func(key string) string {
		if key == "HERDR_PLUGIN_CONTEXT_JSON" {
			return `{"selected_text":"line one\nline two","clicked_urls":["https://secret"]}`
		}
		return ""
	})
	if err != nil || got != "line one\nline two" {
		t.Fatalf("SelectedText() = %q, %v", got, err)
	}
}

func TestSelectedTextRejectsMalformedContext(t *testing.T) {
	_, err := SelectedText(func(key string) string {
		if key == "HERDR_PLUGIN_CONTEXT_JSON" {
			return "{broken"
		}
		return ""
	})
	if err == nil {
		t.Fatal("SelectedText() error = nil")
	}
}

func TestContextHandlesAnAbsentOrNullContextJSON(t *testing.T) {
	empty := func(key string) string {
		if key == "HERDR_PANE_ID" {
			return "pane-1"
		}
		return "" // no HERDR_PLUGIN_CONTEXT_JSON at all
	}
	ctx, err := ReadContext(empty)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.PaneID != "pane-1" || ctx.ContextKeys != nil || ctx.SelectedTextPresent {
		t.Fatalf("ReadContext() without context JSON = %#v", ctx)
	}
	if text, err := SelectedText(empty); err != nil || text != "" {
		t.Fatalf("SelectedText() without context JSON = %q, %v", text, err)
	}
	if dir := ResolveWorkingDirectory(ctx, ""); dir != "" {
		t.Fatalf("ResolveWorkingDirectory() with nothing to fall back to = %q", dir)
	}

	// Valid JSON that is not an object must be refused, not silently treated as empty.
	null := func(key string) string {
		if key == "HERDR_PLUGIN_CONTEXT_JSON" {
			return "null"
		}
		return ""
	}
	if _, err := ReadContext(null); err == nil || !strings.Contains(err.Error(), "expected an object") {
		t.Fatalf("ReadContext() null error = %v", err)
	}
}
