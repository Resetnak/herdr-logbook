package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	got, warnings, err := Load(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("Load() warnings = %v", warnings)
	}
	if got.Version != 1 || got.Storage.ProjectMode != "central" || got.Storage.RepoDirectory != ".herdr/logbook" {
		t.Fatalf("Load() = %#v", got)
	}
	if got.Search.MaxIndexFileBytes != 262144 || got.Capture.MaxSelectionBytes != 524288 {
		t.Fatalf("Load() limits = %#v %#v", got.Search, got.Capture)
	}
}

func TestLoadOverridesDefaultsAndWarnsUnknownKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	data := "version = 1\nunknown = true\n[storage]\nproject_mode = \"repo\"\n[editor]\ncommand = [\"nvim\", \"-f\"]\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	got, warnings, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Storage.ProjectMode != "repo" || got.Storage.RepoDirectory != ".herdr/logbook" {
		t.Fatalf("Load() storage = %#v", got.Storage)
	}
	if len(got.Editor.Command) != 2 || got.Editor.Command[0] != "nvim" {
		t.Fatalf("Load() editor = %#v", got.Editor)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "unknown") {
		t.Fatalf("Load() warnings = %#v", warnings)
	}
}

func TestLoadWarnsAboutRemovedNoOpFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	data := `version = 1
[project]
root_strategy = "git"
share_across_worktrees = false
[ui]
theme = "nord"
popup_width = "80%"
popup_height = "70%"
[search]
follow_symlinks = true
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	got, warnings, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.UI.Theme != "nord" {
		t.Fatalf("Load() theme = %q", got.UI.Theme)
	}
	for _, field := range []string{
		"root_strategy", "share_across_worktrees", "popup_width",
		"popup_height", "follow_symlinks",
	} {
		if len(warnings) != 1 || !strings.Contains(warnings[0], field) {
			t.Fatalf("Load() warnings %q do not mention %q", warnings, field)
		}
	}
}

func TestLoadRejectsInvalidTypeAndValue(t *testing.T) {
	dir := t.TempDir()
	typePath := filepath.Join(dir, "type.toml")
	if err := os.WriteFile(typePath, []byte("[search]\nmax_index_file_bytes = \"large\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Load(typePath); err == nil {
		t.Fatal("Load() invalid type error = nil")
	}

	valuePath := filepath.Join(dir, "value.toml")
	if err := os.WriteFile(valuePath, []byte("[storage]\nproject_mode = \"somewhere\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Load(valuePath); err == nil || !strings.Contains(err.Error(), "project_mode") {
		t.Fatalf("Load() invalid value error = %v", err)
	}
}

func TestLoadMigratesVersionZeroInMemory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("version = 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, _, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Version != 1 {
		t.Fatalf("Load() version = %d, want 1", got.Version)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "version = 0\n" {
		t.Fatalf("Load() rewrote config: %q", data)
	}
}

func TestValidateRejectsUnsafeRepoDirectory(t *testing.T) {
	cfg := Default()
	cfg.Storage.RepoDirectory = "../outside"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "repo_directory") {
		t.Fatalf("Validate() error = %v", err)
	}
	cfg.Storage.RepoDirectory = "."
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "repo_directory") {
		t.Fatalf("Validate() root collision error = %v", err)
	}
}

func TestValidateRejectsInvalidUIConfig(t *testing.T) {
	cfg := Default()
	cfg.UI.ScopeWidth = -5
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "scope_width") {
		t.Fatalf("Validate() invalid scope_width error = %v", err)
	}
	cfg = Default()
	cfg.UI.DefaultView = "invalid"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "default_view") {
		t.Fatalf("Validate() invalid default_view error = %v", err)
	}

	cfg = Default()
	cfg.UI.Theme = "solarized"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "theme") {
		t.Fatalf("Validate() invalid theme error = %v", err)
	}

	cfg = Default()
	cfg.UI.PreviewStyle = "sepia"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "preview_style") {
		t.Fatalf("Validate() invalid preview_style error = %v", err)
	}
}

func TestValidateRejectsEveryRemainingInvalidField(t *testing.T) {
	cases := map[string]func(*Config){
		"unsupported config version":   func(c *Config) { c.Version = 2 },
		"project_mode must be central": func(c *Config) { c.Storage.ProjectMode = "sqlite" },
		"repo_directory absolute":      func(c *Config) { c.Storage.RepoDirectory = string(filepath.Separator) + "etc" },
		"repo_directory empty":         func(c *Config) { c.Storage.RepoDirectory = "" },
		"repo_directory parent":        func(c *Config) { c.Storage.RepoDirectory = ".." },
		"index limit":                  func(c *Config) { c.Search.MaxIndexFileBytes = 0 },
		"preview limit":                func(c *Config) { c.Search.MaxPreviewFileBytes = -1 },
		"selection limit":              func(c *Config) { c.Capture.MaxSelectionBytes = 0 },
		"empty editor argument":        func(c *Config) { c.Editor.Command = []string{"vim", ""} },
		"NUL in editor argument":       func(c *Config) { c.Editor.Command = []string{"vim\x00-f"} },
	}
	for name, mutate := range cases {
		cfg := Default()
		mutate(&cfg)
		if err := cfg.Validate(); err == nil {
			t.Fatalf("%s: Validate() accepted the config", name)
		}
	}

	cfg := Default()
	cfg.UI.DefaultView = ""
	cfg.Editor.Command = []string{"nvim", "-f"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() rejected a valid config: %v", err)
	}
}
