package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Resetnak/herdr-logbook/internal/app"
	"github.com/Resetnak/herdr-logbook/internal/author"
	"github.com/Resetnak/herdr-logbook/internal/capture"
	"github.com/Resetnak/herdr-logbook/internal/config"
	"github.com/Resetnak/herdr-logbook/internal/digest"
	"github.com/Resetnak/herdr-logbook/internal/editor"
	"github.com/Resetnak/herdr-logbook/internal/herdr"
	searchindex "github.com/Resetnak/herdr-logbook/internal/index"
	"github.com/Resetnak/herdr-logbook/internal/nowfile"
	"github.com/Resetnak/herdr-logbook/internal/project"
	"github.com/Resetnak/herdr-logbook/internal/storage"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
	"github.com/pelletier/go-toml/v2"
	"golang.org/x/term"
)

var version = "dev"

type diagnostic struct {
	PluginVersion    string        `json:"plugin_version"`
	OS               string        `json:"os"`
	Architecture     string        `json:"architecture"`
	WorkingDirectory string        `json:"working_directory,omitempty"`
	Context          herdr.Context `json:"context"`
}

type coreState struct {
	Config         config.Config
	ConfigWarnings []string
	ConfigFile     string
	StateDir       string
	Context        herdr.Context
	Project        project.Project
	Layout         storage.Layout
}

type pathsReport struct {
	ConfigFile   string `json:"config_file,omitempty"`
	StateDir     string `json:"state_dir"`
	Registry     string `json:"registry"`
	Cache        string `json:"cache"`
	ProjectStore string `json:"project_store"`
	Now          string `json:"now"`
	Inbox        string `json:"inbox"`
	Notes        string `json:"notes"`
	Decisions    string `json:"decisions"`
	Lock         string `json:"lock"`
}

type doctorReport struct {
	PluginVersion     string            `json:"plugin_version"`
	OS                string            `json:"os"`
	Architecture      string            `json:"architecture"`
	HerdrBinary       string            `json:"herdr_binary,omitempty"`
	HerdrVersion      string            `json:"herdr_version,omitempty"`
	PluginRoot        string            `json:"plugin_root,omitempty"`
	ConfigDir         string            `json:"config_dir,omitempty"`
	StateDir          string            `json:"state_dir"`
	Context           herdr.Context     `json:"context"`
	SelectedDirectory string            `json:"selected_working_directory"`
	Project           project.Project   `json:"project"`
	StorageMode       string            `json:"storage_mode"`
	StorePath         string            `json:"store_path"`
	Editor            editor.Resolution `json:"editor"`
	GlowAvailable     bool              `json:"glow_available"`
	StoreState        string            `json:"store_state"`
	CacheState        string            `json:"cache_state"`
	Warnings          []string          `json:"warnings,omitempty"`
}

type commandError struct {
	code int
	err  error
}

func main() {
	os.Exit(run(os.Args[1:], os.Getenv, os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, getenv func(string) string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "help", "--help", "-h":
		printUsage(stdout)
		return 0
	case "version":
		if len(args) != 1 {
			printUsage(stderr)
			return 2
		}
		fmt.Fprintf(stdout, "herdr-logbook %s\n", version)
		return 0
	case "resolve-cwd":
		if len(args) != 1 {
			printUsage(stderr)
			return 2
		}
		ctx, err := herdr.ReadContext(getenv)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 5
		}
		fallback, _ := os.Getwd()
		fmt.Fprintln(stdout, herdr.ResolveWorkingDirectory(ctx, fallback))
		return 0
	case "compatibility", "compat":
		return runCompatibility(args[1:], getenv, stdin, stdout, stderr)
	case "init":
		return runInit(args[1:], getenv, stdout, stderr)
	case "paths":
		return runPaths(args[1:], getenv, stdout, stderr)
	case "doctor":
		return runDoctor(args[1:], getenv, stdout, stderr)
	case "capture":
		return runCapture(args[1:], getenv, stdin, stdout, stderr)
	case "decision":
		return runDecision(args[1:], getenv, stdin, stdout, stderr)
	case "now":
		return runNow(args[1:], getenv, stdout, stderr)
	case "digest":
		return runDigest(args[1:], getenv, stdout, stderr)
	case "tui":
		return runTUI(args[1:], getenv, stdin, stdout, stderr)
	case "index":
		return runIndex(args[1:], getenv, stdout, stderr)
	case "keybinds":
		if len(args) != 1 {
			printUsage(stderr)
			return 2
		}
		printKeybinds(stdout)
		return 0
	default:
		printUsage(stderr)
		return 2
	}
}

func printKeybinds(writer io.Writer) {
	fmt.Fprintln(writer, `Navigation:
  j/k or arrows       move
  h/l, Tab/Shift+Tab  change panel
  g/G                 first/last
  Enter, v            open preview
  /, Esc              search/clear
  p                   filter search by project

Actions:
  c/C  project/global capture
  n    create project note
  d    create decision
  t    set the current task in now.md
  e    edit in external editor
  s    standup digest & activity heatmap
  r    refresh
  ?    help
  q    close

Capture / Authoring Modal:
  Ctrl+S   save note
  Ctrl+E   save note & open external editor
  Esc      cancel capture

Digest View:
  y    copy standup Markdown to clipboard
  t    toggle today / this week
  Esc  back to hub`)
}

func runIndex(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "rebuild" {
		fmt.Fprintln(stderr, "usage: herdr-logbook index rebuild [--project-root PATH]")
		return 2
	}
	flags := flag.NewFlagSet("index rebuild", flag.ContinueOnError)
	flags.SetOutput(stderr)
	projectRoot := flags.String("project-root", "", "project root")
	cwd := flags.String("cwd", "", "fallback working directory")
	if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 {
		return 2
	}
	state, failure := loadCore(*projectRoot, *cwd, "", getenv)
	if failure != nil {
		fmt.Fprintln(stderr, failure.err)
		return failure.code
	}
	registryPath, registryLock := registryPaths(state.StateDir)
	if err := project.UpdateRegistry(registryPath, registryLock, 2*time.Second, state.Project, state.Layout.Mode, state.Layout.Root, time.Now().UTC()); err != nil {
		fmt.Fprintln(stderr, err)
		return 4
	}
	registry, err := project.LoadRegistry(registryPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		// Registry read failure is a storage problem, same as the lock below.
		return 4
	}
	stores := make([]searchindex.Store, 0, len(registry.Projects)+1)
	for _, record := range registry.Projects {
		stores = append(stores, searchindex.Store{ProjectID: record.ID, ProjectName: record.Name, Root: record.StorePath})
	}
	stores = append(stores, searchindex.Store{ProjectID: "global", ProjectName: "Global", Root: filepath.Join(state.StateDir, "store", "global")})
	cachePath := filepath.Join(state.StateDir, "cache", "index-v1.json")
	lockPath := filepath.Join(state.StateDir, "locks", "index.lock")
	var count int
	err = storage.WithLock(lockPath, 5*time.Second, func() error {
		entries, scanErr := searchindex.Refresh(stores, state.Config.Search.MaxIndexFileBytes, nil)
		if scanErr != nil {
			return scanErr
		}
		count = len(entries)
		return searchindex.SaveCache(cachePath, searchindex.Cache{Entries: entries})
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 4
	}
	fmt.Fprintf(stdout, "indexed %d notes\n", count)
	return 0
}

func rebuildSearchIndex(state coreState) ([]searchindex.Entry, error) {
	registry, err := project.LoadRegistry(filepath.Join(state.StateDir, "registry", "projects.toml"))
	if err != nil {
		return nil, err
	}
	stores := make([]searchindex.Store, 0, len(registry.Projects)+1)
	for _, record := range registry.Projects {
		stores = append(stores, searchindex.Store{ProjectID: record.ID, ProjectName: record.Name, Root: record.StorePath})
	}
	stores = append(stores, searchindex.Store{ProjectID: "global", ProjectName: "Global", Root: filepath.Join(state.StateDir, "store", "global")})
	cachePath := filepath.Join(state.StateDir, "cache", "index-v1.json")
	var entries []searchindex.Entry
	err = storage.WithLock(filepath.Join(state.StateDir, "locks", "index.lock"), 5*time.Second, func() error {
		// Reading the cache under the lock keeps this the only place the Hub
		// touches it, and lets the rescan carry unchanged notes over instead of
		// re-reading every note in every project for a one-file change.
		cache, cacheErr := searchindex.LoadCache(cachePath)
		if cacheErr != nil {
			return cacheErr
		}
		var scanErr error
		entries, scanErr = searchindex.Refresh(stores, state.Config.Search.MaxIndexFileBytes, cache.Entries)
		if scanErr != nil {
			return scanErr
		}
		return searchindex.SaveCache(cachePath, searchindex.Cache{Entries: entries})
	})
	return entries, err
}

// authorFromHub performs the write behind the Hub's n, d, and t keys. It is a
// plain function rather than a closure so the three write paths stay testable
// without a terminal.
func authorFromHub(state coreState, kind, title string) (string, error) {
	// setNowTask takes state.Layout.Lock itself, so it must stay outside the
	// lock the note/decision authors run under.
	if kind == "now" {
		if err := nowfile.ValidateTask(title); err != nil {
			return "", err
		}
		if _, err := setNowTask(state, title); err != nil {
			return "", err
		}
		return state.Layout.Now, nil
	}
	var path string
	err := storage.WithLock(state.Layout.Lock, 2*time.Second, func() error {
		var createErr error
		if kind == "decision" {
			path, createErr = author.CreateDecision(state.Layout.Root, title, state.Project.Name, state.Project.Branch, time.Now())
		} else {
			path, createErr = author.CreateNote(state.Layout.Root, title)
		}
		return createErr
	})
	if err != nil {
		return "", err
	}
	return path, nil
}

// editorCommandFor builds the argv command the Hub's e key executes. Splitting
// it out of the tea.Cmd closure keeps the guard and the editor resolution — the
// two parts that can refuse — testable on their own.
func editorCommandFor(state coreState, editorArgv []string, getenv func(string) string, path string) (*exec.Cmd, error) {
	if err := validateEditableNote(state, path); err != nil {
		return nil, err
	}
	resolved, err := editor.Resolve(editorArgv, getenv, runtime.GOOS, exec.LookPath)
	if err != nil {
		return nil, err
	}
	return exec.Command(resolved.Command[0], append(resolved.Command[1:], path)...), nil
}

func validateEditableNote(state coreState, path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("open note for editing: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || info.IsDir() || !strings.EqualFold(filepath.Ext(path), ".md") {
		return fmt.Errorf("note path is not a regular Markdown file")
	}
	roots := []string{state.Layout.Root, filepath.Join(state.StateDir, "store", "global")}
	registry, err := project.LoadRegistry(filepath.Join(state.StateDir, "registry", "projects.toml"))
	if err == nil {
		for _, record := range registry.Projects {
			roots = append(roots, record.StorePath)
		}
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	for _, root := range roots {
		absoluteRoot, rootErr := filepath.Abs(root)
		if rootErr != nil {
			continue
		}
		relative, relErr := filepath.Rel(absoluteRoot, absolutePath)
		if relErr == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return nil
		}
	}
	return fmt.Errorf("note path is outside registered memory stores")
}

func runTUI(args []string, getenv func(string) string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("tui", flag.ContinueOnError)
	flags.SetOutput(stderr)
	view := flags.String("view", "", "now, project, global, or all (defaults to config ui.default_view)")
	projectRoot := flags.String("project-root", "", "project root")
	cwd := flags.String("cwd", "", "fallback working directory")
	contextJSON := flags.String("context-json", "", "Herdr invocation context JSON")
	editorCmd := flags.String("editor", "", "editor command for e, e.g. vim, nano (overrides config/$EDITOR)")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return 2
	}
	effectiveView := *view
	if effectiveView == "" {
		effectiveView = "now"
	}
	effectiveGetenv := getenv
	if *contextJSON != "" {
		effectiveGetenv = func(key string) string {
			if key == "HERDR_PLUGIN_CONTEXT_JSON" {
				return *contextJSON
			}
			return getenv(key)
		}
	}
	state, failure := loadCore(*projectRoot, *cwd, "", effectiveGetenv)
	if failure != nil {
		fmt.Fprintln(stderr, failure.err)
		return failure.code
	}
	if *view == "" && state.Config.UI.DefaultView != "" {
		effectiveView = state.Config.UI.DefaultView
	}
	if effectiveView != "now" && effectiveView != "project" && effectiveView != "global" && effectiveView != "all" {
		fmt.Fprintln(stderr, "tui view must be now, project, global, or all")
		return 2
	}
	if err := storage.WithLock(state.Layout.Lock, 2*time.Second, func() error { return storage.Initialize(state.Layout) }); err != nil {
		fmt.Fprintln(stderr, err)
		return 4
	}
	registryPath, registryLock := registryPaths(state.StateDir)
	if err := project.UpdateRegistry(registryPath, registryLock, 2*time.Second, state.Project, state.Layout.Mode, state.Layout.Root, time.Now().UTC()); err != nil {
		fmt.Fprintln(stderr, err)
		return 4
	}
	globalRoot := filepath.Join(state.StateDir, "store", "global")
	reloadNotes := func() ([]app.Note, error) {
		return app.LoadNotes(state.Layout.Root, globalRoot, state.Config.Search.MaxPreviewFileBytes)
	}
	notes, err := reloadNotes()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	// The index is not loaded here: parsing the cache is tens of milliseconds of
	// JSON before the first frame, and Init() schedules the same rebuild as a
	// command anyway. The Hub opens on now.md and shows "indexing" until it lands.
	refreshSearch := func() ([]searchindex.Entry, error) {
		return rebuildSearchIndex(state)
	}
	captureNote := func(text string, global bool) (string, []app.Note, error) {
		path, err := appendCapture(state, text, global, false, "", "")
		if err != nil {
			return "", nil, err
		}
		notes, loadErr := reloadNotes()
		return path, notes, loadErr
	}
	authorNote := func(kind, title string) (string, []app.Note, error) {
		path, err := authorFromHub(state, kind, title)
		if err != nil {
			return "", nil, err
		}
		notes, loadErr := reloadNotes()
		return path, notes, loadErr
	}
	editorArgv := state.Config.Editor.Command
	if *editorCmd != "" {
		editorArgv = strings.Fields(*editorCmd) // ponytail: simple split; HERDR_LOGBOOK_EDITOR handles quoted paths
	}
	editNote := func(note app.Note) tea.Cmd {
		command, err := editorCommandFor(state, editorArgv, getenv, note.Path)
		if err != nil {
			return func() tea.Msg { return app.NotesReloadedMsg{Err: err} }
		}
		return tea.ExecProcess(command, func(runErr error) tea.Msg {
			if runErr != nil {
				return app.NotesReloadedMsg{Err: fmt.Errorf("editor exited with error: %w (check the note file on disk)", runErr), RefreshSearch: true}
			}
			notes, loadErr := reloadNotes()
			return app.NotesReloadedMsg{Notes: notes, Err: loadErr, RefreshSearch: true}
		})
	}
	previewStyle := state.Config.UI.PreviewStyle
	if previewStyle == "" || previewStyle == "auto" {
		previewStyle = detectGlamourStyle(stdout)
	}
	digestReport := func(days int) (digest.DigestReport, error) {
		return digest.Collect(state.Layout.Root, state.Project.Name, state.Project.Branch, days)
	}
	model := app.NewHub(notes, state.Project.Name, state.Project.Branch, state.Layout.Mode).
		WithView(effectiveView).
		WithUIConfig(state.Config.UI.ScopeWidth, state.Config.UI.ShowBranch, state.Config.UI.Theme).
		WithStyle(previewStyle).
		WithActions(captureNote, reloadNotes).
		WithSearch(nil, refreshSearch).
		WithAuthoring(authorNote, editNote).
		WithDigest(digestReport)
	program := tea.NewProgram(model, tea.WithInput(stdin), tea.WithOutput(stdout), tea.WithAltScreen(), tea.WithoutSignalHandler())
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

// detectGlamourStyle resolves the Glamour preview style once, before Bubble Tea
// takes over stdin. It mirrors glamour's WithAutoStyle decision (background probe
// via termenv) but runs while the terminal can still answer the query, so the
// TUI never blocks on it later. See HubModel.WithStyle.
func detectGlamourStyle(stdout io.Writer) string {
	f, ok := stdout.(*os.File)
	if !ok || !term.IsTerminal(int(f.Fd())) {
		return "notty"
	}
	if termenv.HasDarkBackground() {
		return "dark"
	}
	return "light"
}

func runDecision(args []string, getenv func(string) string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("decision", flag.ContinueOnError)
	flags.SetOutput(stderr)
	title := flags.String("title", "", "decision title")
	projectRoot := flags.String("project-root", "", "project root")
	cwd := flags.String("cwd", "", "fallback working directory")
	noEdit := flags.Bool("no-edit", false, "create without opening an editor")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return 2
	}
	if strings.TrimSpace(*title) == "" {
		fmt.Fprint(stdout, "Decision title: ")
		line, err := bufio.NewReader(stdin).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			fmt.Fprintln(stderr, "read decision title")
			// A broken stdin is bad input, same as capture --stdin.
			return 2
		}
		*title = strings.TrimSpace(line)
	}
	if strings.TrimSpace(*title) == "" {
		fmt.Fprintln(stderr, "decision title is required")
		return 2
	}
	state, failure := loadCore(*projectRoot, *cwd, "", getenv)
	if failure != nil {
		fmt.Fprintln(stderr, failure.err)
		return failure.code
	}
	var path string
	err := storage.WithLock(state.Layout.Lock, 2*time.Second, func() error {
		if err := storage.Initialize(state.Layout); err != nil {
			return err
		}
		var createErr error
		path, createErr = author.CreateDecision(state.Layout.Root, *title, state.Project.Name, state.Project.Branch, time.Now())
		return createErr
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 4
	}
	registryPath, registryLock := registryPaths(state.StateDir)
	if err := project.UpdateRegistry(registryPath, registryLock, 2*time.Second, state.Project, state.Layout.Mode, state.Layout.Root, time.Now().UTC()); err != nil {
		fmt.Fprintln(stderr, err)
		return 4
	}
	// The file already exists; print its path before the editor gets a chance
	// to fail, so an exit 6 never hides where the decision landed.
	fmt.Fprintln(stdout, path)
	if !*noEdit {
		resolved, err := editor.Resolve(state.Config.Editor.Command, getenv, runtime.GOOS, exec.LookPath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 6
		}
		command := exec.Command(resolved.Command[0], append(resolved.Command[1:], path)...)
		command.Stdin, command.Stdout, command.Stderr = stdin, stdout, stderr
		if err := command.Run(); err != nil {
			fmt.Fprintln(stderr, err)
			return 6
		}
	}
	return 0
}

func runCapture(args []string, getenv func(string) string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("capture", flag.ContinueOnError)
	flags.SetOutput(stderr)
	scope := flags.String("scope", "project", "project or global")
	textValue := flags.String("text", "", "capture text")
	readStdin := flags.Bool("stdin", false, "read capture from stdin")
	selected := flags.Bool("selected", false, "capture selected terminal text")
	projectRoot := flags.String("project-root", "", "project root")
	cwd := flags.String("cwd", "", "fallback working directory")
	branch := flags.String("branch", "", "branch metadata")
	sourceCWD := flags.String("source-cwd", "", "source cwd metadata")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return 2
	}
	if *scope != "project" && *scope != "global" {
		fmt.Fprintln(stderr, "capture scope must be project or global")
		return 2
	}
	inputCount := 0
	if *textValue != "" {
		inputCount++
	}
	if *readStdin {
		inputCount++
	}
	if *selected {
		inputCount++
	}
	if inputCount > 1 {
		fmt.Fprintln(stderr, "capture requires exactly one of --text, --stdin, or --selected")
		return 2
	}
	state, failure := loadCore(*projectRoot, *cwd, "", getenv)
	if failure != nil {
		fmt.Fprintln(stderr, failure.err)
		return failure.code
	}
	if inputCount == 0 {
		return runCaptureTUI(state, *scope == "global", stdin, stdout, stderr)
	}
	content := *textValue
	if *readStdin {
		data, err := io.ReadAll(io.LimitReader(stdin, state.Config.Capture.MaxSelectionBytes+1))
		if err != nil {
			fmt.Fprintln(stderr, "read capture stdin")
			// A broken stdin pipe is bad input, not an internal failure.
			return 2
		}
		content = string(data)
	}
	if *selected {
		var err error
		content, err = herdr.SelectedText(getenv)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 5
		}
	}
	if strings.TrimSpace(content) == "" {
		fmt.Fprintln(stderr, "capture text is empty")
		return 2
	}
	if int64(len(content)) > state.Config.Capture.MaxSelectionBytes {
		fmt.Fprintf(stderr, "capture size exceeds limit of %d bytes\n", state.Config.Capture.MaxSelectionBytes)
		return 2
	}
	if *branch == "" && state.Config.Capture.IncludeBranch {
		*branch = state.Project.Branch
	}
	if *sourceCWD == "" && state.Config.Capture.IncludeSourceCWD {
		*sourceCWD = herdr.ResolveWorkingDirectory(state.Context, state.Project.Root)
	}
	inboxDir, lockPath := state.Layout.Inbox, state.Layout.Lock
	if *scope == "global" {
		inboxDir = filepath.Join(state.StateDir, "store", "global", "inbox")
		lockPath = filepath.Join(state.StateDir, "locks", "global.lock")
	}
	path, err := capture.Append(capture.Request{
		InboxDir: inboxDir, LockPath: lockPath, MaxBytes: state.Config.Capture.MaxSelectionBytes,
		LockTimeout: 2 * time.Second,
		Entry: capture.Entry{
			Time: time.Now(), Text: content, Branch: *branch, SourceCWD: *sourceCWD, Selection: *selected,
		},
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 4
	}
	if *scope == "project" {
		registryPath, registryLock := registryPaths(state.StateDir)
		if err := project.UpdateRegistry(registryPath, registryLock, 2*time.Second, state.Project, state.Layout.Mode, state.Layout.Root, time.Now().UTC()); err != nil {
			fmt.Fprintln(stderr, err)
			return 4
		}
	}
	fmt.Fprintln(stdout, path)
	return 0
}

func runCaptureTUI(state coreState, global bool, stdin io.Reader, stdout, stderr io.Writer) int {
	if err := storage.WithLock(state.Layout.Lock, 2*time.Second, func() error {
		return storage.Initialize(state.Layout)
	}); err != nil {
		fmt.Fprintln(stderr, err)
		return 4
	}

	saved := false
	captureNote := func(text string, captureGlobal bool) (string, []app.Note, error) {
		path, err := appendCapture(state, text, captureGlobal, false, "", "")
		if err != nil {
			return "", nil, err
		}
		saved = true
		return path, nil, nil
	}
	model := app.NewHub(nil, state.Project.Name, state.Project.Branch, state.Layout.Mode).
		WithActions(captureNote, nil)
	model, _ = model.BeginCapture(global, true)
	program := tea.NewProgram(model,
		tea.WithInput(stdin),
		tea.WithOutput(stdout),
		tea.WithAltScreen(),
		tea.WithoutSignalHandler(),
	)
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if saved && !global {
		registryPath, registryLock := registryPaths(state.StateDir)
		if err := project.UpdateRegistry(registryPath, registryLock, 2*time.Second, state.Project, state.Layout.Mode, state.Layout.Root, time.Now().UTC()); err != nil {
			fmt.Fprintln(stderr, err)
			return 4
		}
	}
	return 0
}

func appendCapture(state coreState, content string, global, selection bool, branch, sourceCWD string) (string, error) {
	request, err := buildCaptureRequest(state, content, global, selection, branch, sourceCWD)
	if err != nil {
		return "", err
	}
	return capture.Append(request)
}

func buildCaptureRequest(state coreState, content string, global, selection bool, branch, sourceCWD string) (capture.Request, error) {
	if strings.TrimSpace(content) == "" {
		return capture.Request{}, fmt.Errorf("capture text is empty")
	}
	if int64(len(content)) > state.Config.Capture.MaxSelectionBytes {
		return capture.Request{}, fmt.Errorf("capture size exceeds limit of %d bytes", state.Config.Capture.MaxSelectionBytes)
	}
	if branch == "" && state.Config.Capture.IncludeBranch {
		branch = state.Project.Branch
	}
	if sourceCWD == "" && state.Config.Capture.IncludeSourceCWD {
		sourceCWD = herdr.ResolveWorkingDirectory(state.Context, state.Project.Root)
	}

	inboxDir, lockPath := state.Layout.Inbox, state.Layout.Lock
	if global {
		inboxDir = filepath.Join(state.StateDir, "store", "global", "inbox")
		lockPath = filepath.Join(state.StateDir, "locks", "global.lock")
	}
	return capture.Request{
		InboxDir:    inboxDir,
		LockPath:    lockPath,
		MaxBytes:    state.Config.Capture.MaxSelectionBytes,
		LockTimeout: 2 * time.Second,
		Entry: capture.Entry{
			Time:      time.Now(),
			Text:      content,
			Branch:    branch,
			SourceCWD: sourceCWD,
			Selection: selection,
		},
	}, nil
}

func runNow(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("now", flag.ContinueOnError)
	flags.SetOutput(stderr)
	projectRoot := flags.String("project-root", "", "project root")
	cwd := flags.String("cwd", "", "fallback working directory")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	task := strings.TrimSpace(strings.Join(flags.Args(), " "))
	state, failure := loadCore(*projectRoot, *cwd, "", getenv)
	if failure != nil {
		fmt.Fprintln(stderr, failure.err)
		return failure.code
	}

	if task == "" {
		content, err := os.ReadFile(state.Layout.Now)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintln(stderr, err)
			return 4
		}
		current := nowfile.CurrentTask(string(content))
		if current == "" {
			fmt.Fprintln(stdout, "no current task set")
			return 0
		}
		fmt.Fprintln(stdout, current)
		return 0
	}

	if err := nowfile.ValidateTask(task); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if int64(len(task)) > state.Config.Capture.MaxSelectionBytes {
		fmt.Fprintf(stderr, "task size exceeds limit of %d bytes\n", state.Config.Capture.MaxSelectionBytes)
		return 2
	}
	previous, err := setNowTask(state, task)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 4
	}
	registryPath, registryLock := registryPaths(state.StateDir)
	if err := project.UpdateRegistry(registryPath, registryLock, 2*time.Second, state.Project, state.Layout.Mode, state.Layout.Root, time.Now().UTC()); err != nil {
		fmt.Fprintln(stderr, err)
		return 4
	}
	if previous != "" {
		fmt.Fprintf(stdout, "archived: %s\n", firstLine(previous))
	}
	fmt.Fprintln(stdout, state.Layout.Now)
	return 0
}

// setNowTask replaces the "## Current task" section of now.md and files the task
// it displaced into the monthly inbox, so switching tasks builds a work journal
// for free. It returns the task that was replaced.
func setNowTask(state coreState, task string) (string, error) {
	var previous string
	err := storage.WithLock(state.Layout.Lock, 2*time.Second, func() error {
		if err := storage.Initialize(state.Layout); err != nil {
			return err
		}
		content, readErr := storage.ReadForRewrite(state.Layout.Now)
		if readErr != nil {
			return readErr
		}
		previous = nowfile.CurrentTask(string(content))
		updated, setErr := nowfile.SetCurrentTask(string(content), task)
		if setErr != nil {
			return setErr
		}
		// Read, archive, and overwrite form one critical section so a concurrent
		// now.md change cannot be clobbered by a write computed from a stale
		// read. The archive still runs before the overwrite: a failed overwrite
		// duplicates a journal entry instead of losing one.
		if previous != "" {
			request, err := buildCaptureRequest(state, "Task done: "+previous, false, false, "", "")
			if err != nil {
				return err
			}
			if _, err := capture.AppendLocked(request); err != nil {
				return err
			}
		}
		return storage.AtomicWrite(state.Layout.Now, []byte(updated), 0o600)
	})
	if err != nil {
		return "", err
	}
	return previous, nil
}

func firstLine(text string) string {
	if index := strings.IndexByte(text, '\n'); index >= 0 {
		return strings.TrimSpace(text[:index]) + " …"
	}
	return text
}

func runDigest(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("digest", flag.ContinueOnError)
	flags.SetOutput(stderr)
	projectRoot := flags.String("project-root", "", "project root")
	cwd := flags.String("cwd", "", "fallback working directory")
	days := flags.Int("days", 1, "number of days to include (default 1)")
	jsonOutput := flags.Bool("json", false, "output as JSON")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return 2
	}
	if *days < 1 {
		*days = 1
	}
	state, failure := loadCore(*projectRoot, *cwd, "", getenv)
	if failure != nil {
		fmt.Fprintln(stderr, failure.err)
		return failure.code
	}
	report, err := digest.Collect(state.Layout.Root, state.Project.Name, state.Project.Branch, *days)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *jsonOutput {
		if err := writeJSON(stdout, report, true); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	fmt.Fprint(stdout, digest.FormatMarkdown(report))
	return 0
}

func runCompatibility(args []string, getenv func(string) string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) > 1 || (len(args) == 1 && args[0] != "--wait") {
		fmt.Fprintln(stderr, "usage: herdr-logbook compatibility [--wait]")
		return 2
	}
	ctx, err := herdr.ReadContext(getenv)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 5
	}
	fallback, _ := os.Getwd()
	result := diagnostic{
		PluginVersion: version, OS: runtime.GOOS, Architecture: runtime.GOARCH,
		WorkingDirectory: herdr.ResolveWorkingDirectory(ctx, fallback), Context: ctx,
	}
	if err := writeJSON(stdout, result, true); err != nil {
		fmt.Fprintf(stderr, "write compatibility diagnostic: %v\n", err)
		return 1
	}
	if len(args) == 1 {
		fmt.Fprintln(stdout, "Diagnostics stay open for 30 seconds; press Enter to close afterward.")
		waitForClose(stdin, 30*time.Second, 10*time.Minute)
	}
	return 0
}

func runInit(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("init", flag.ContinueOnError)
	flags.SetOutput(stderr)
	mode := flags.String("storage", "", "central or repo")
	projectRoot := flags.String("project-root", "", "project root")
	cwd := flags.String("cwd", "", "fallback working directory")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return 2
	}
	state, failure := loadCore(*projectRoot, *cwd, *mode, getenv)
	if failure != nil {
		fmt.Fprintln(stderr, failure.err)
		return failure.code
	}
	if err := storage.WithLock(state.Layout.Lock, 2*time.Second, func() error {
		if err := storage.Initialize(state.Layout); err != nil {
			return err
		}
		if state.Layout.ProjectFile == "" {
			return nil
		}
		metadata := struct {
			Version           int      `toml:"version"`
			ID                string   `toml:"id"`
			Name              string   `toml:"name"`
			RemoteFingerprint string   `toml:"remote_fingerprint,omitempty"`
			DefaultRoot       string   `toml:"default_root"`
			Roots             []string `toml:"roots"`
		}{1, state.Project.ID, state.Project.Name, state.Project.Fingerprint, state.Project.Root, state.Project.Roots}
		data, err := toml.Marshal(metadata)
		if err != nil {
			return err
		}
		return storage.AtomicWrite(state.Layout.ProjectFile, data, 0o600)
	}); err != nil {
		fmt.Fprintln(stderr, err)
		return 4
	}
	registryPath, registryLock := registryPaths(state.StateDir)
	if err := project.UpdateRegistry(registryPath, registryLock, 2*time.Second, state.Project, state.Layout.Mode, state.Layout.Root, time.Now().UTC()); err != nil {
		fmt.Fprintln(stderr, err)
		return 4
	}
	fmt.Fprintf(stdout, "initialized %s (%s)\n", state.Layout.Root, state.Layout.Mode)
	if state.Layout.Mode == "repo" {
		ignoreRule := strings.TrimSuffix(filepath.ToSlash(state.Config.Storage.RepoDirectory), "/") + "/"
		fmt.Fprintf(stdout, "Warning: these files may be committed. Recommended ignore rule: %s\n", ignoreRule)
	}
	return 0
}

func runPaths(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("paths", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOutput := flags.Bool("json", false, "print JSON")
	mode := flags.String("storage", "", "central or repo")
	projectRoot := flags.String("project-root", "", "project root")
	cwd := flags.String("cwd", "", "fallback working directory")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return 2
	}
	state, failure := loadCore(*projectRoot, *cwd, *mode, getenv)
	if failure != nil {
		fmt.Fprintln(stderr, failure.err)
		return failure.code
	}
	registryPath, _ := registryPaths(state.StateDir)
	report := pathsReport{
		ConfigFile: state.ConfigFile, StateDir: state.StateDir, Registry: registryPath,
		Cache:        filepath.Join(state.StateDir, "cache", "index-v1.json"),
		ProjectStore: state.Layout.Root, Now: state.Layout.Now, Inbox: state.Layout.Inbox,
		Notes: state.Layout.Notes, Decisions: state.Layout.Decisions, Lock: state.Layout.Lock,
	}
	if *jsonOutput {
		if err := writeJSON(stdout, report, true); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "config: %s\nstate: %s\nregistry: %s\nproject: %s\nnow: %s\n", report.ConfigFile, report.StateDir, report.Registry, report.ProjectStore, report.Now)
	return 0
}

func runDoctor(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("doctor", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOutput := flags.Bool("json", false, "print JSON")
	mode := flags.String("storage", "", "central or repo")
	projectRoot := flags.String("project-root", "", "project root")
	cwd := flags.String("cwd", "", "fallback working directory")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return 2
	}
	state, failure := loadCore(*projectRoot, *cwd, *mode, getenv)
	if failure != nil {
		fmt.Fprintln(stderr, failure.err)
		return failure.code
	}
	report := doctorReport{
		PluginVersion: version, OS: runtime.GOOS, Architecture: runtime.GOARCH,
		HerdrBinary: state.Context.Paths.HerdrBinary, PluginRoot: state.Context.Paths.PluginRoot,
		ConfigDir: filepath.Dir(state.ConfigFile), StateDir: state.StateDir, Context: state.Context,
		SelectedDirectory: state.Project.Root, Project: state.Project,
		StorageMode: state.Layout.Mode, StorePath: state.Layout.Root,
		StoreState: pathState(state.Layout.Root),
		CacheState: pathState(filepath.Join(state.StateDir, "cache", "index-v1.json")),
		Warnings:   append([]string{}, state.ConfigWarnings...),
	}
	if report.ConfigDir == "." && state.ConfigFile == "" {
		report.ConfigDir = ""
	}
	if resolved, err := editor.Resolve(state.Config.Editor.Command, getenv, runtime.GOOS, exec.LookPath); err == nil {
		report.Editor = resolved
	} else {
		report.Warnings = append(report.Warnings, err.Error())
	}
	if _, err := exec.LookPath("glow"); err == nil {
		report.GlowAvailable = true
	}
	if herdrVersion, binary, err := inspectHerdr(report.HerdrBinary); err == nil {
		report.HerdrVersion = herdrVersion
		report.HerdrBinary = binary
	} else {
		report.Warnings = append(report.Warnings, err.Error())
	}
	if *jsonOutput {
		if err := writeJSON(stdout, report, true); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "Herdr Logbook %s · %s/%s\nproject: %s (%s)\nstore: %s [%s]\neditor: %s\ncache: %s\n", report.PluginVersion, report.OS, report.Architecture, report.Project.Name, report.Project.ID, report.StorePath, report.StorageMode, strings.Join(report.Editor.Command, " "), report.CacheState)
	for _, warning := range report.Warnings {
		fmt.Fprintf(stdout, "warning: %s\n", warning)
	}
	return 0
}

func loadCore(projectRoot, cwd, requestedMode string, getenv func(string) string) (coreState, *commandError) {
	ctx, err := herdr.ReadContext(getenv)
	if err != nil {
		return coreState{}, &commandError{5, err}
	}
	configDir := getenv("HERDR_PLUGIN_CONFIG_DIR")
	configFile := ""
	cfg := config.Default()
	var warnings []string
	if configDir != "" {
		configFile = filepath.Join(configDir, "config.toml")
		cfg, warnings, err = config.Load(configFile)
		if err != nil {
			return coreState{}, &commandError{2, err}
		}
	}
	stateDir := getenv("HERDR_PLUGIN_STATE_DIR")
	if stateDir == "" {
		return coreState{}, &commandError{3, fmt.Errorf("HERDR_PLUGIN_STATE_DIR is required")}
	}
	resolved, err := project.Resolve(project.ResolveOptions{
		ExplicitRoot: projectRoot, WorktreePath: ctx.WorktreePath, FocusedPaneCWD: ctx.FocusedPaneCWD,
		WorkspaceCWD: ctx.WorkspaceCWD, CWD: cwd,
	})
	if err != nil {
		return coreState{}, &commandError{3, err}
	}
	// The storage mode comes from the user only: --storage, or storage.project_mode
	// in their own config. A repository cannot vote — see project.overrideConfig.
	layout, err := storage.Resolve(stateDir, resolved.Root, resolved.ID, cfg, requestedMode)
	if err != nil {
		return coreState{}, &commandError{2, err}
	}
	return coreState{
		Config: cfg, ConfigWarnings: warnings, ConfigFile: configFile, StateDir: stateDir,
		Context: ctx, Project: resolved, Layout: layout,
	}, nil
}

func inspectHerdr(binary string) (string, string, error) {
	var err error
	if binary == "" {
		binary, err = exec.LookPath("herdr")
		if err != nil {
			return "", "", fmt.Errorf("herdr binary not found")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, binary, "--version").Output()
	if err != nil {
		return "", binary, fmt.Errorf("herdr version unavailable: %w", err)
	}
	return strings.TrimSpace(string(output)), binary, nil
}

func registryPaths(stateDir string) (string, string) {
	return filepath.Join(stateDir, "registry", "projects.toml"), filepath.Join(stateDir, "locks", "registry.lock")
}

func pathState(path string) string {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "missing"
	}
	if err != nil {
		return "unavailable"
	}
	if info.IsDir() && info.Mode().Perm()&0o200 == 0 {
		return "read-only"
	}
	return "available"
}

func writeJSON(writer io.Writer, value any, indent bool) error {
	encoder := json.NewEncoder(writer)
	if indent {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(value)
}

// waitForClose keeps the pane open for at least `timeout`, then closes on
// Enter/EOF — or after `grace`, so a supervisor holding stdin open without
// ever writing cannot park the process forever.
func waitForClose(reader io.Reader, timeout, grace time.Duration) {
	result := make(chan struct{}, 1)
	go func() {
		_, _ = bufio.NewReader(reader).ReadString('\n')
		result <- struct{}{}
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-result:
		<-timer.C
	case <-timer.C:
		graceTimer := time.NewTimer(grace)
		defer graceTimer.Stop()
		select {
		case <-result:
		case <-graceTimer.C:
		}
	}
}

func printUsage(writer io.Writer) {
	fmt.Fprintln(writer, "usage: herdr-logbook tui | capture | decision | now [TASK] | digest [--days N] [--json] | init | paths | doctor [--json] | index rebuild | keybinds | compatibility [--wait] | resolve-cwd | version")
}
