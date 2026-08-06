package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Config mirrors the user's config.toml (see config.example.toml). Missing
// keys keep the Default values; unknown or invalid ones surface as warnings
// or errors from Load.
type Config struct {
	Version int           `toml:"version" json:"version"`
	Storage StorageConfig `toml:"storage" json:"storage"`
	Editor  EditorConfig  `toml:"editor" json:"editor"`
	UI      UIConfig      `toml:"ui" json:"ui"`
	Search  SearchConfig  `toml:"search" json:"search"`
	Capture CaptureConfig `toml:"capture" json:"capture"`
}

type StorageConfig struct {
	ProjectMode   string `toml:"project_mode" json:"project_mode"`
	RepoDirectory string `toml:"repo_directory" json:"repo_directory"`
}

type EditorConfig struct {
	Command []string `toml:"command" json:"command"`
}

type UIConfig struct {
	Theme        string `toml:"theme" json:"theme"`
	PreviewStyle string `toml:"preview_style" json:"preview_style"`
	ShowBranch   bool   `toml:"show_branch" json:"show_branch"`
	ScopeWidth   int    `toml:"scope_width" json:"scope_width"`
	DefaultView  string `toml:"default_view" json:"default_view"`
}

type SearchConfig struct {
	MaxIndexFileBytes   int64 `toml:"max_index_file_bytes" json:"max_index_file_bytes"`
	MaxPreviewFileBytes int64 `toml:"max_preview_file_bytes" json:"max_preview_file_bytes"`
}

type CaptureConfig struct {
	MaxSelectionBytes int64 `toml:"max_selection_bytes" json:"max_selection_bytes"`
	IncludeBranch     bool  `toml:"include_branch" json:"include_branch"`
	IncludeSourceCWD  bool  `toml:"include_source_cwd" json:"include_source_cwd"`
}

// Default is the configuration used when no config.toml exists.
func Default() Config {
	return Config{
		Version: 1,
		Storage: StorageConfig{ProjectMode: "central", RepoDirectory: ".herdr/logbook"},
		Editor:  EditorConfig{Command: []string{}},
		UI: UIConfig{
			Theme: "auto", PreviewStyle: "auto", ShowBranch: true,
			ScopeWidth: 24, DefaultView: "now",
		},
		Search: SearchConfig{
			MaxIndexFileBytes: 262144, MaxPreviewFileBytes: 2097152,
		},
		Capture: CaptureConfig{
			MaxSelectionBytes: 524288, IncludeBranch: true, IncludeSourceCWD: true,
		},
	}
}

// Load reads the config at path on top of Default. A missing file is not an
// error; unknown keys come back as warnings so a typo never fails silently.
func Load(path string) (Config, []string, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil, nil
	}
	if err != nil {
		return Config{}, nil, fmt.Errorf("read config %q: %w", path, err)
	}

	err = toml.NewDecoder(bytes.NewReader(data)).DisallowUnknownFields().Decode(&cfg)
	var warnings []string
	var unknown *toml.StrictMissingError
	if errors.As(err, &unknown) {
		warnings = append(warnings, unknown.String())
	} else if err != nil {
		return Config{}, nil, fmt.Errorf("decode config %q: %w", path, err)
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, warnings, fmt.Errorf("validate config %q: %w", path, err)
	}
	return cfg, warnings, nil
}

func (c Config) Validate() error {
	if c.Version != 1 {
		return fmt.Errorf("unsupported config version %d", c.Version)
	}
	if c.Storage.ProjectMode != "central" && c.Storage.ProjectMode != "repo" {
		return fmt.Errorf("storage.project_mode must be central or repo")
	}
	repoDir := filepath.Clean(c.Storage.RepoDirectory)
	if c.Storage.RepoDirectory == "" || filepath.IsAbs(repoDir) || os.IsPathSeparator(repoDir[0]) || repoDir == "." || repoDir == ".." || strings.HasPrefix(repoDir, ".."+string(filepath.Separator)) {
		return fmt.Errorf("storage.repo_directory must stay inside the project")
	}
	switch c.UI.Theme {
	case "", "auto", "dracula", "tokyo-night", "nord", "default":
	default:
		return fmt.Errorf("ui.theme must be auto, dracula, tokyo-night, nord, or default")
	}
	switch c.UI.PreviewStyle {
	case "", "auto", "dark", "light", "notty":
	default:
		return fmt.Errorf("ui.preview_style must be auto, dark, light, or notty")
	}
	if c.UI.ScopeWidth <= 0 {
		return fmt.Errorf("ui.scope_width must be positive")
	}
	if c.UI.DefaultView != "" && c.UI.DefaultView != "now" && c.UI.DefaultView != "project" && c.UI.DefaultView != "global" && c.UI.DefaultView != "all" {
		return fmt.Errorf("ui.default_view must be now, project, global, or all")
	}
	if c.Search.MaxIndexFileBytes <= 0 || c.Search.MaxPreviewFileBytes <= 0 {
		return fmt.Errorf("search byte limits must be positive")
	}
	if c.Capture.MaxSelectionBytes <= 0 {
		return fmt.Errorf("capture.max_selection_bytes must be positive")
	}
	for _, arg := range c.Editor.Command {
		if arg == "" || strings.ContainsRune(arg, '\x00') {
			return fmt.Errorf("editor.command contains an invalid argument")
		}
	}
	return nil
}
