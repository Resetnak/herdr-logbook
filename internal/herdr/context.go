package herdr

import (
	"encoding/json"
	"fmt"
	"sort"
)

// Paths contains the filesystem locations Herdr supplies to a plugin process.
type Paths struct {
	HerdrBinary string `json:"herdr_binary,omitempty"`
	PluginRoot  string `json:"plugin_root,omitempty"`
	ConfigDir   string `json:"config_dir,omitempty"`
	StateDir    string `json:"state_dir,omitempty"`
}

// Context is the non-sensitive invocation summary used by compatibility diagnostics.
type Context struct {
	Paths               Paths    `json:"paths"`
	WorkspaceID         string   `json:"workspace_id,omitempty"`
	TabID               string   `json:"tab_id,omitempty"`
	PaneID              string   `json:"pane_id,omitempty"`
	WorkspaceCWD        string   `json:"workspace_cwd,omitempty"`
	FocusedPaneCWD      string   `json:"focused_pane_cwd,omitempty"`
	WorktreePath        string   `json:"worktree_path,omitempty"`
	InvocationSource    string   `json:"invocation_source,omitempty"`
	ContextKeys         []string `json:"context_keys,omitempty"`
	SelectedTextPresent bool     `json:"selected_text_present"`

	SelectedText string   `json:"-"`
	ClickedURLs  []string `json:"-"`
}

// ReadContext reads Herdr's environment without retaining selected text or clicked URLs.
func ReadContext(getenv func(string) string) (Context, error) {
	ctx := Context{
		Paths: Paths{
			HerdrBinary: getenv("HERDR_BIN_PATH"),
			PluginRoot:  getenv("HERDR_PLUGIN_ROOT"),
			ConfigDir:   getenv("HERDR_PLUGIN_CONFIG_DIR"),
			StateDir:    getenv("HERDR_PLUGIN_STATE_DIR"),
		},
		WorkspaceID: getenv("HERDR_WORKSPACE_ID"),
		TabID:       getenv("HERDR_TAB_ID"),
		PaneID:      getenv("HERDR_PANE_ID"),
	}

	rawJSON := getenv("HERDR_PLUGIN_CONTEXT_JSON")
	if rawJSON == "" {
		return ctx, nil
	}

	var raw map[string]any
	if err := json.Unmarshal([]byte(rawJSON), &raw); err != nil {
		return Context{}, fmt.Errorf("parse HERDR_PLUGIN_CONTEXT_JSON: %w", err)
	}
	if raw == nil {
		return Context{}, fmt.Errorf("parse HERDR_PLUGIN_CONTEXT_JSON: expected an object")
	}

	ctx.WorkspaceCWD = stringValue(raw, "workspace_cwd")
	ctx.FocusedPaneCWD = stringValue(raw, "focused_pane_cwd")
	ctx.WorktreePath = stringValue(raw, "worktree_path")
	ctx.InvocationSource = stringValue(raw, "invocation_source")
	_, ctx.SelectedTextPresent = raw["selected_text"]
	for key := range raw {
		ctx.ContextKeys = append(ctx.ContextKeys, key)
	}
	sort.Strings(ctx.ContextKeys)

	return ctx, nil
}

// ResolveWorkingDirectory follows Herdr's documented context priority for the spike.
func ResolveWorkingDirectory(ctx Context, fallback string) string {
	for _, path := range []string{ctx.WorktreePath, ctx.FocusedPaneCWD, ctx.WorkspaceCWD, fallback} {
		if path != "" {
			return path
		}
	}
	return ""
}

// SelectedText reads terminal selection only for an explicit capture command.
func SelectedText(getenv func(string) string) (string, error) {
	rawJSON := getenv("HERDR_PLUGIN_CONTEXT_JSON")
	if rawJSON == "" {
		return "", nil
	}
	var selected struct {
		Text string `json:"selected_text"`
	}
	if err := json.Unmarshal([]byte(rawJSON), &selected); err != nil {
		return "", fmt.Errorf("parse HERDR_PLUGIN_CONTEXT_JSON: %w", err)
	}
	return selected.Text, nil
}

func stringValue(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return value
}
