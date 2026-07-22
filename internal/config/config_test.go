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
