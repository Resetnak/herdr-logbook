package app

import (
	"fmt"
	"strings"
	"time"

	searchindex "github.com/Resetnak/herdr-logbook/internal/index"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

const (
	panelScopes = iota
	panelNotes
	panelPreview
)

type scopeItem struct {
	name         string
	types        map[NoteType]bool
	emptyMessage string
}

// CaptureFunc and AuthorFunc report the path they wrote alongside the reloaded
// notes. The Hub cannot infer it: a capture appends to an existing monthly inbox
// and setting the current task rewrites now.md, so neither produces a note path
// the Hub has not seen before.
type CaptureFunc func(text string, global bool) (string, []Note, error)
type ReloadFunc func() ([]Note, error)
type SearchLoadFunc func() ([]searchindex.Entry, error)
type AuthorFunc func(kind, title string) (string, []Note, error)
type EditFunc func(note Note) tea.Cmd

type searchLoadedMsg struct {
	entries []searchindex.Entry
	err     error
}

type NotesReloadedMsg struct {
	Notes         []Note
	Err           error
	RefreshSearch bool
}

type captureSavedMsg struct {
	written    string
	notes      []Note
	kind       string
	openEditor bool
	err        error
}

type flashExpiredMsg struct{ tick int }

type HubModel struct {
	notes            []Note
	scopes           []scopeItem
	scopeIndex       int
	noteIndex        int
	panel            int
	width            int
	height           int
	projectName      string
	branch           string
	storageMode      string
	help             bool
	preview          viewport.Model
	captureBox       textarea.Model
	capturing        bool
	captureGlobal    bool
	captureErr       string
	quitAfterCapture bool
	captureFn        CaptureFunc
	reloadFn         ReloadFunc
	searchBox        textinput.Model
	searching        bool
	searchQuery      string
	projectSearch    bool
	searchEntries    []searchindex.Entry
	searchResults    []Note
	searchLoadFn     SearchLoadFunc
	searchRefreshing bool
	searchDirty      bool
	searchErr        string
	authorKind       string
	saving           bool
	authorFn         AuthorFunc
	editFn           EditFunc
	previewRenderer  *glamour.TermRenderer
	previewWidth     int
	previewCache     string
	previewKey       previewCacheKey
	glamourStyle     string
	flashMsg         string
	flashTick        int
	scopeWidth       int
	showBranch       bool
	uiTheme          string
}

// previewCacheKey holds everything the rendered Markdown depends on. Content is
// compared by value, but notes keep their backing string across a refresh, so the
// common case is a pointer-equality hit rather than a byte scan.
type previewCacheKey struct {
	path    string
	content string
	width   int
	style   string
}

func NewHub(notes []Note, projectName, branch, storageMode string) HubModel {
	captureBox := textarea.New()
	captureBox.Placeholder = "What should future you remember?"
	captureBox.ShowLineNumbers = false
	captureBox.SetWidth(72)
	captureBox.SetHeight(8)
	searchBox := textinput.New()
	searchBox.Prompt = "/ "
	searchBox.Placeholder = "Search all projects"

	model := HubModel{
		notes:       notes,
		projectName: projectName,
		branch:      branch,
		storageMode: storageMode,
		scopeWidth:  24,
		showBranch:  true,
		uiTheme:     "auto",
		width:       80,
		height:      24,
		panel:       panelScopes,
		preview:     viewport.New(40, 18),
		captureBox:  captureBox,
		searchBox:   searchBox,
		scopes: []scopeItem{
			{name: "Now", types: map[NoteType]bool{NoteNow: true}, emptyMessage: "Current context is unavailable. Reopen Logbook to restore now.md."},
			{name: "Project Inbox", types: map[NoteType]bool{NoteProjectInbox: true}, emptyMessage: "No project inbox captures yet. Press c to capture something."},
			{name: "Project Notes", types: map[NoteType]bool{NoteProjectNote: true}, emptyMessage: "No project notes yet. Press n to create one."},
			{name: "Decisions", types: map[NoteType]bool{NoteDecision: true}, emptyMessage: "No decisions yet. Press d to create one."},
			{name: "Global Inbox", types: map[NoteType]bool{NoteGlobalInbox: true}, emptyMessage: "No global inbox captures yet. Press C to capture something."},
			{name: "All Notes", emptyMessage: "No notes yet. Press c to capture or n to create a project note."},
		},
	}
	// No render here: WithAutoStyle would probe the terminal background for a
	// result WithStyle discards moments later. The first WindowSizeMsg renders.
	return model
}

func flashCmd(tick int) tea.Cmd {
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return flashExpiredMsg{tick: tick}
	})
}

func (m HubModel) WithView(view string) HubModel {
	switch view {
	case "project":
		m.scopeIndex = 2
	case "global":
		m.scopeIndex = 4
	case "all":
		m.scopeIndex = 5
	default:
		m.scopeIndex = 0
	}
	m.refreshPreview()
	return m
}

// WithStyle pins the Glamour preview style ("dark"/"light"/"notty"). Detect it
// once before the program starts: WithAutoStyle queries the terminal background
// on every renderer build, and once Bubble Tea owns stdin that query blocks for
// seconds waiting on a reply it never receives.
func (m HubModel) WithStyle(style string) HubModel {
	m.glamourStyle = style
	m.previewRenderer = nil
	m.refreshPreview()
	return m
}

func (m HubModel) WithActions(captureFn CaptureFunc, reloadFn ReloadFunc) HubModel {
	m.captureFn = captureFn
	m.reloadFn = reloadFn
	return m
}

func (m HubModel) WithSearch(entries []searchindex.Entry, loadFn SearchLoadFunc) HubModel {
	m.searchEntries = entries
	m.searchLoadFn = loadFn
	if loadFn != nil {
		m.searchRefreshing = true
	}
	return m
}

func (m HubModel) WithAuthoring(authorFn AuthorFunc, editFn EditFunc) HubModel {
	m.authorFn = authorFn
	m.editFn = editFn
	return m
}

func (m HubModel) BeginCapture(global, quitAfterCapture bool) (HubModel, tea.Cmd) {
	m.quitAfterCapture = quitAfterCapture
	return m.openCapture(global)
}

func (m HubModel) Init() tea.Cmd {
	var commands []tea.Cmd
	// The capture-only TUI opens the modal before the program starts, so the
	// command BeginCapture returned never reaches Bubble Tea.
	if m.capturing {
		commands = append(commands, textarea.Blink)
	}
	if m.searchLoadFn != nil {
		commands = append(commands, m.loadSearchCmd())
	}
	return tea.Batch(commands...)
}

func (m HubModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	if reloaded, ok := message.(NotesReloadedMsg); ok {
		if reloaded.Err != nil {
			m.captureErr = reloaded.Err.Error()
			if reloaded.RefreshSearch {
				command := m.refreshSearchCmd()
				return m, command
			}
			return m, nil
		}
		m.notes = reloaded.Notes
		m.captureErr = ""
		m.flashMsg = "✓ Note updated"
		m.flashTick++
		m.refreshPreview()
		var searchCommand tea.Cmd
		if reloaded.RefreshSearch {
			searchCommand = m.refreshSearchCmd()
		}
		return m, tea.Batch(searchCommand, flashCmd(m.flashTick))
	}
	if loaded, ok := message.(searchLoadedMsg); ok {
		m.searchRefreshing = false
		if loaded.err != nil {
			m.searchErr = loaded.err.Error()
		} else {
			m.searchEntries = loaded.entries
			m.searchErr = ""
			m.updateSearchResults()
		}
		if m.searchDirty {
			m.searchDirty = false
			command := m.refreshSearchCmd()
			return m, command
		}
		return m, nil
	}
	if saved, ok := message.(captureSavedMsg); ok {
		return m.applyCaptureSaved(saved)
	}
	if expired, ok := message.(flashExpiredMsg); ok {
		if expired.tick == m.flashTick {
			m.flashMsg = ""
		}
		return m, nil
	}
	if m.capturing {
		return m.updateCapture(message)
	}
	if m.searching {
		return m.updateSearch(message)
	}

	var command tea.Cmd
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
		m.refreshPreview()
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "?":
			m.help = !m.help
		case "/":
			m.projectSearch = false
			m.searchBox.Prompt = "/ "
			m.searchBox.Placeholder = "Search all projects"
			m.searching = true
			m.searchBox.SetValue(m.searchQuery)
			return m, m.searchBox.Focus()
		case "p":
			m.projectSearch = true
			m.searchBox.Prompt = "project> "
			m.searchBox.Placeholder = "Filter by project name"
			m.searching = true
			m.searchBox.SetValue("")
			m.searchQuery = ""
			return m, m.searchBox.Focus()
		case "esc":
			m.clearSearch()
		case "c":
			return m.openCapture(false)
		case "C":
			return m.openCapture(true)
		case "n":
			return m.openAuthor("note")
		case "d":
			return m.openAuthor("decision")
		case "t":
			return m.openAuthor("now")
		case "e":
			notes := m.visibleNotes()
			if m.editFn != nil && len(notes) > 0 {
				return m, m.editFn(notes[clamp(m.noteIndex, 0, len(notes)-1)])
			}
		case "r":
			if m.reloadFn != nil {
				command = m.reloadCmd()
			}
		case "tab":
			m.panel = (m.panel + 1) % 3
		case "shift+tab":
			m.panel = (m.panel + 2) % 3
		case "h", "left":
			if m.panel > panelScopes {
				m.panel--
			}
		case "l", "right":
			if m.panel < panelPreview {
				m.panel++
			}
		case "enter":
			if m.panel == panelScopes && len(m.visibleNotes()) > 0 {
				m.panel = panelNotes
			} else if m.panel == panelNotes && len(m.visibleNotes()) > 0 {
				m.panel = panelPreview
			}
		case "j", "down":
			m.move(1)
		case "k", "up":
			m.move(-1)
		case "g":
			m.moveTo(false)
		case "G":
			m.moveTo(true)
		case "v":
			m.panel = panelPreview
		}
		m.refreshPreview()
	}

	if m.panel == panelPreview {
		var viewportCommand tea.Cmd
		m.preview, viewportCommand = m.preview.Update(message)
		return m, tea.Batch(command, viewportCommand)
	}
	return m, command
}

func (m HubModel) updateCapture(message tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := message.(tea.WindowSizeMsg); ok {
		m.resize(msg.Width, msg.Height)
		return m, nil
	}
	if msg, ok := message.(tea.KeyMsg); ok {
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.capturing = false
			m.authorKind = ""
			m.captureErr = ""
			m.captureBox.Blur()
			m.captureBox.Reset()
			if m.quitAfterCapture {
				return m, tea.Quit
			}
			return m, nil
		case "ctrl+s", "ctrl+e":
			if m.saving {
				return m, nil
			}
			text := m.captureBox.Value()
			if strings.TrimSpace(text) == "" {
				m.captureErr = "Capture text cannot be empty."
				return m, nil
			}
			openEditor := msg.String() == "ctrl+e"
			kind := m.authorKind
			if kind != "" && m.authorFn == nil {
				m.captureErr = "Authoring is unavailable."
				return m, nil
			}
			if kind == "" && m.captureFn == nil {
				m.captureErr = "Capture is unavailable."
				return m, nil
			}
			// Writing takes the storage lock and rereads the store afterwards; both
			// belong off the event loop so the modal keeps painting meanwhile.
			authorFn, captureFn, global := m.authorFn, m.captureFn, m.captureGlobal
			m.saving = true
			m.captureErr = ""
			return m, func() tea.Msg {
				saved := captureSavedMsg{kind: kind, openEditor: openEditor}
				if kind != "" {
					saved.written, saved.notes, saved.err = authorFn(kind, strings.TrimSpace(text))
				} else {
					saved.written, saved.notes, saved.err = captureFn(text, global)
				}
				return saved
			}
		}
	}

	var command tea.Cmd
	m.captureBox, command = m.captureBox.Update(message)
	return m, command
}

func (m HubModel) View() string {
	if m.capturing {
		return m.captureView()
	}
	if m.help {
		helpText := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render("📓 Herdr Logbook — Quick Start & Help") + "\n\n" +
			lipgloss.NewStyle().Bold(true).Render("Navigation:") + "\n" +
			"  Tab / Shift+Tab or h / l   Switch panel (Scopes → Notes → Preview)\n" +
			"  j / k or ↑ / ↓             Move selection\n" +
			"  g / G                      Jump to top / bottom\n" +
			"  Enter / v                  Open Markdown preview\n" +
			"  /                          Fuzzy search all projects\n" +
			"  p                          Filter search by project name\n\n" +
			lipgloss.NewStyle().Bold(true).Render("Actions:") + "\n" +
			"  c / C                      Quick capture (project / global inbox)\n" +
			"  n / d                      New project note / decision record\n" +
			"  t                          Set current task (previous one is archived)\n" +
			"  e                          Edit note in external editor ($EDITOR / vi)\n" +
			"  r                          Reload / refresh notes\n" +
			"  ? / q                      Toggle help / quit\n\n" +
			lipgloss.NewStyle().Bold(true).Render("Capture Modal Shortcuts:") + "\n" +
			"  Ctrl+S                     Save note\n" +
			"  Ctrl+E                     Save note & open external editor\n" +
			"  Esc                        Cancel capture\n\n" +
			lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Markdown supported: # title · **bold** · - list · `code` · #tag") + "\n" +
			lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Editor saves: :wq (vim/nvim) or editor's normal save+quit.")

		return m.pane(helpText, max(30, m.width), max(3, m.height-2), true)
	}

	availableHeight := max(3, m.height-2)
	var body string
	scopeW := max(16, m.scopeWidth)
	if m.width >= 110 {
		noteWidth := max(28, m.width/3)
		previewWidth := max(30, m.width-scopeW-noteWidth-4)
		body = lipgloss.JoinHorizontal(lipgloss.Top,
			m.pane(m.scopesView(), scopeW, availableHeight, m.panel == panelScopes),
			m.pane(m.notesView(), noteWidth, availableHeight, m.panel == panelNotes),
			m.pane(m.previewView(), previewWidth, availableHeight, m.panel == panelPreview),
		)
	} else if m.width >= 70 {
		content := m.notesView()
		if m.panel == panelPreview {
			content = m.previewView()
		}
		body = lipgloss.JoinHorizontal(lipgloss.Top,
			m.pane(m.scopesView(), scopeW, availableHeight, m.panel == panelScopes),
			m.pane(content, max(30, m.width-scopeW-2), availableHeight, m.panel != panelScopes),
		)
	} else {
		content := m.scopesView()
		if m.panel == panelNotes {
			content = m.notesView()
		} else if m.panel == panelPreview {
			content = m.previewView()
		}
		body = m.pane(content, max(20, m.width), availableHeight, true)
	}

	var statusParts []string
	statusParts = append(statusParts, m.projectName)
	if m.showBranch && m.branch != "" {
		statusParts = append(statusParts, m.branch)
	}
	statusParts = append(statusParts, m.storageMode+" store", "/ search", "? help")
	status := strings.Join(statusParts, " · ")

	if m.searching {
		status = m.searchBox.View() + " · Esc clear · Enter results"
	} else if m.searchQuery != "" {
		status = fmt.Sprintf("search: %s · Esc clear · %s", m.searchQuery, status)
	}
	if m.flashMsg != "" {
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render(m.flashMsg) + " · " + status
	}
	errorMessage := m.captureErr
	if errorMessage == "" {
		errorMessage = m.searchErr
	}
	if errorMessage != "" {
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(errorMessage) + " · " + status
	}
	return body + "\n" + lipgloss.NewStyle().Faint(true).Render(status)
}

func (m HubModel) applyCaptureSaved(saved captureSavedMsg) (tea.Model, tea.Cmd) {
	m.saving = false
	if saved.err != nil {
		m.captureErr = saved.err.Error()
		return m, nil
	}
	m.notes = saved.notes
	m.capturing = false
	m.captureErr = ""
	m.captureBox.Blur()
	m.captureBox.Reset()
	m.authorKind = ""
	m.refreshPreview()
	if saved.openEditor && m.editFn != nil && saved.written != "" {
		for _, note := range saved.notes {
			if note.Path == saved.written {
				return m, m.editFn(note)
			}
		}
	}
	if m.quitAfterCapture {
		return m, tea.Quit
	}
	m.flashMsg = "✓ Saved to inbox"
	if saved.kind == "now" {
		m.flashMsg = "✓ Current task updated"
	}
	m.flashTick++
	return m, tea.Batch(m.refreshSearchCmd(), flashCmd(m.flashTick))
}

func (m HubModel) captureView() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	title := titleStyle.Render("📝 Project capture")
	if m.captureGlobal {
		title = titleStyle.Render("🌐 Global capture")
	}
	switch m.authorKind {
	case "note":
		title = titleStyle.Render("📄 New project note — enter a title")
	case "decision":
		title = titleStyle.Render("⚖️ New decision — enter a title")
	case "now":
		title = titleStyle.Render("🎯 Current task — what are you working on?")
	}
	body := title + "\n\n" + m.captureBox.View()
	if m.captureErr != "" {
		body += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(m.captureErr)
	}
	mdHint := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Markdown supported: # title · **bold** · - list · #tag")
	keyHint := lipgloss.NewStyle().Foreground(lipgloss.Color("248")).Render("Ctrl+S save · Ctrl+E save & edit in editor · Esc cancel")
	if m.saving {
		keyHint = lipgloss.NewStyle().Foreground(lipgloss.Color("248")).Render("Saving…")
	}
	body += "\n\n" + mdHint + "\n" + keyHint
	// The box must be at least as wide as the textarea's rendered rows plus
	// horizontal padding; anything narrower makes lipgloss re-wrap those rows
	// (reflow breaks at hyphens), visually splitting lines that contain "-".
	boxWidth := max(24, lipgloss.Width(m.captureBox.View())+4)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2).
		Width(boxWidth).
		Render(body)
}

func (m HubModel) openCapture(global bool) (HubModel, tea.Cmd) {
	m.authorKind = ""
	m.captureGlobal = global
	m.capturing = true
	m.captureErr = ""
	m.captureBox.Reset()
	return m, m.captureBox.Focus()
}

func (m HubModel) openAuthor(kind string) (HubModel, tea.Cmd) {
	m.authorKind = kind
	m.captureGlobal = false
	m.capturing = true
	m.captureErr = ""
	m.captureBox.Reset()
	return m, m.captureBox.Focus()
}

func (m HubModel) updateSearch(message tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := message.(tea.KeyMsg); ok {
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.clearSearch()
			return m, nil
		case "enter":
			m.searching = false
			m.searchBox.Blur()
			m.panel = panelNotes
			return m, nil
		}
	}
	var command tea.Cmd
	m.searchBox, command = m.searchBox.Update(message)
	m.searchQuery = strings.TrimSpace(m.searchBox.Value())
	m.updateSearchResults()
	m.noteIndex = 0
	m.refreshPreview()
	return m, command
}

func (m *HubModel) updateSearchResults() {
	m.searchResults = nil
	if m.searchQuery == "" {
		return
	}
	if m.projectSearch {
		for _, entry := range m.searchEntries {
			if !strings.Contains(strings.ToLower(entry.ProjectName), strings.ToLower(m.searchQuery)) {
				continue
			}
			m.searchResults = append(m.searchResults, searchNote(entry))
			if len(m.searchResults) == 100 {
				break
			}
		}
		return
	}
	for _, result := range searchindex.Search(m.searchEntries, m.searchQuery, 100) {
		m.searchResults = append(m.searchResults, searchNote(result.Entry))
	}
}

func searchNote(entry searchindex.Entry) Note {
	title := entry.Title
	if entry.ProjectName != "" {
		title += " · " + entry.ProjectName
	}
	return Note{
		Path: entry.Path, Title: title, Type: NoteProjectNote,
		Content: entry.Content, Modified: entry.Modified, Size: entry.Size,
	}
}

func (m *HubModel) clearSearch() {
	m.searching = false
	m.searchQuery = ""
	m.projectSearch = false
	m.searchResults = nil
	m.searchBox.Blur()
	m.searchBox.Reset()
	m.noteIndex = 0
	m.refreshPreview()
}

func (m HubModel) loadSearchCmd() tea.Cmd {
	return func() tea.Msg {
		entries, err := m.searchLoadFn()
		return searchLoadedMsg{entries: entries, err: err}
	}
}

// refreshSearchCmd schedules a rebuild, or marks one as pending when a rebuild is
// already in flight. searchRefreshing is the only gate: a second load can never be
// dispatched while one is outstanding, so a searchLoadedMsg is always the answer to
// the newest request and needs no generation tag to be recognised as current.
func (m *HubModel) refreshSearchCmd() tea.Cmd {
	if m.searchLoadFn == nil {
		return nil
	}
	if m.searchRefreshing {
		m.searchDirty = true
		return nil
	}
	m.searchRefreshing = true
	return m.loadSearchCmd()
}

func (m *HubModel) resize(width, height int) {
	m.width, m.height = width, height
	m.captureBox.SetWidth(max(20, width-8))
	m.captureBox.SetHeight(min(12, max(3, height-8)))
}

type themePalette struct {
	headerFg lipgloss.Color
	activeFg lipgloss.Color
	focusBd  lipgloss.Color
	dimBd    lipgloss.Color
}

func getThemePalette(theme string) themePalette {
	switch strings.ToLower(theme) {
	case "dracula":
		return themePalette{headerFg: lipgloss.Color("141"), activeFg: lipgloss.Color("212"), focusBd: lipgloss.Color("141"), dimBd: lipgloss.Color("236")}
	case "tokyo-night":
		return themePalette{headerFg: lipgloss.Color("73"), activeFg: lipgloss.Color("111"), focusBd: lipgloss.Color("73"), dimBd: lipgloss.Color("237")}
	case "nord":
		return themePalette{headerFg: lipgloss.Color("81"), activeFg: lipgloss.Color("255"), focusBd: lipgloss.Color("81"), dimBd: lipgloss.Color("239")}
	default:
		return themePalette{headerFg: lipgloss.Color("39"), activeFg: lipgloss.Color("212"), focusBd: lipgloss.Color("39"), dimBd: lipgloss.Color("238")}
	}
}

func (m HubModel) WithUIConfig(scopeWidth int, showBranch bool, theme string) HubModel {
	if scopeWidth > 0 {
		m.scopeWidth = scopeWidth
	}
	m.showBranch = showBranch
	if theme != "" {
		m.uiTheme = theme
	}
	return m
}

func (m HubModel) pane(content string, width, height int, focused bool) string {
	palette := getThemePalette(m.uiTheme)
	color := palette.dimBd
	if focused {
		color = palette.focusBd
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(color).
		Width(max(1, width-2)).
		Height(max(1, height-2)).
		Padding(0, 1).
		Render(content)
}

func (m HubModel) scopesView() string {
	palette := getThemePalette(m.uiTheme)
	var output strings.Builder
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(palette.headerFg)
	output.WriteString(titleStyle.Render("Scopes") + "\n\n")
	activeStyle := lipgloss.NewStyle().Bold(true).Foreground(palette.activeFg)
	for index, scope := range m.scopes {
		prefix := "  "
		line := scope.name
		if index == m.scopeIndex {
			prefix = "● "
			line = activeStyle.Render(scope.name)
		}
		output.WriteString(prefix + line + "\n")
	}
	return output.String()
}

func (m HubModel) notesView() string {
	notes := m.visibleNotes()
	palette := getThemePalette(m.uiTheme)
	var output strings.Builder
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(palette.headerFg)
	if m.searchQuery != "" {
		output.WriteString(titleStyle.Render("Search results") + "\n\n")
	} else {
		output.WriteString(titleStyle.Render("Notes") + "\n\n")
	}
	if len(notes) == 0 {
		output.WriteString(m.emptyStateMessage() + "\n")
		return output.String()
	}
	activeStyle := lipgloss.NewStyle().Bold(true).Foreground(palette.activeFg)
	for index, note := range notes {
		prefix := "  "
		line := note.Title
		if index == m.noteIndex {
			prefix = "› "
			line = activeStyle.Render(note.Title)
		}
		output.WriteString(prefix + line + "\n")
	}
	return output.String()
}

func (m HubModel) previewView() string {
	palette := getThemePalette(m.uiTheme)
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(palette.headerFg)
	if len(m.visibleNotes()) == 0 {
		return titleStyle.Render("Preview") + "\n\n" + m.emptyStateMessage() + "\n"
	}
	return titleStyle.Render("Preview") + "\n\n" + m.preview.View()
}

func (m HubModel) emptyStateMessage() string {
	if m.searchQuery != "" {
		if m.searchRefreshing {
			return "Refreshing search index..."
		}
		if m.searchErr != "" {
			return "Search index refresh failed. Results may be stale."
		}
		return "No matching notes. Press Esc to clear the search."
	}
	if m.scopeIndex >= 0 && m.scopeIndex < len(m.scopes) {
		return m.scopes[m.scopeIndex].emptyMessage
	}
	return "No notes in this scope yet."
}

func (m HubModel) visibleNotes() []Note {
	if m.searchQuery != "" {
		return m.searchResults
	}
	if m.scopeIndex < 0 || m.scopeIndex >= len(m.scopes) {
		return nil
	}
	types := m.scopes[m.scopeIndex].types
	if types == nil {
		return m.notes
	}
	result := make([]Note, 0, len(m.notes))
	for _, note := range m.notes {
		if types[note.Type] {
			result = append(result, note)
		}
	}
	return result
}

// reloadCmd rescans the store off the event loop. Reading every note back from
// disk inside Update would freeze the TUI until the scan finished.
func (m HubModel) reloadCmd() tea.Cmd {
	reloadFn := m.reloadFn
	return func() tea.Msg {
		notes, err := reloadFn()
		return NotesReloadedMsg{Notes: notes, Err: err, RefreshSearch: true}
	}
}

func (m *HubModel) move(delta int) {
	// The preview panel forwards j/k to the viewport; moving the selection here
	// would scroll one note and display another on the same keypress.
	if m.panel == panelPreview {
		return
	}
	if m.panel == panelScopes {
		m.scopeIndex = clamp(m.scopeIndex+delta, 0, len(m.scopes)-1)
		m.noteIndex = 0
		return
	}
	m.noteIndex = clamp(m.noteIndex+delta, 0, len(m.visibleNotes())-1)
}

func (m *HubModel) moveTo(end bool) {
	if m.panel == panelPreview {
		return
	}
	if m.panel == panelScopes {
		m.scopeIndex = 0
		if end {
			m.scopeIndex = len(m.scopes) - 1
		}
		m.noteIndex = 0
		return
	}
	m.noteIndex = 0
	if end && len(m.visibleNotes()) > 0 {
		m.noteIndex = len(m.visibleNotes()) - 1
	}
}

func (m *HubModel) refreshPreview() {
	notes := m.visibleNotes()
	if len(notes) == 0 {
		m.noteIndex = 0
		m.previewKey = previewCacheKey{}
		m.previewCache = ""
		m.preview.SetContent("")
		return
	}
	m.noteIndex = clamp(m.noteIndex, 0, len(notes)-1)
	width := max(20, m.width/2-4)
	if m.width < 70 {
		width = max(20, m.width-4)
	}
	note := notes[m.noteIndex]
	// Every keystroke reaches this function, so re-render only when the output
	// would actually differ. Glamour parses and highlights the whole document.
	key := previewCacheKey{path: note.Path, content: note.Content, width: width, style: m.glamourStyle}
	if key != m.previewKey {
		rendered := note.Content
		// ponytail: cache the renderer too; NewTermRenderer loads chroma styles, so
		// only rebuild it when the wrap width actually changes.
		if m.previewRenderer == nil || m.previewWidth != width {
			style := glamour.WithAutoStyle()
			if m.glamourStyle != "" {
				style = glamour.WithStandardStyle(m.glamourStyle)
			}
			if renderer, err := glamour.NewTermRenderer(style, glamour.WithWordWrap(width)); err == nil {
				m.previewRenderer = renderer
				m.previewWidth = width
			}
		}
		if m.previewRenderer != nil {
			if output, renderErr := m.previewRenderer.Render(rendered); renderErr == nil {
				rendered = output
			}
		}
		m.previewCache = rendered
		m.previewKey = key
	}
	m.preview.Width = width
	m.preview.Height = max(3, m.height-6)
	m.preview.SetContent(m.previewCache)
}

func clamp(value, low, high int) int {
	if high < low {
		return low
	}
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}
