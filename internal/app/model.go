package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/Resetnak/herdr-logbook/internal/digest"
	searchindex "github.com/Resetnak/herdr-logbook/internal/index"
	md "github.com/Resetnak/herdr-logbook/internal/markdown"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
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
type DigestFunc func(days int) (digest.DigestReport, error)

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

type flashDecayMsg struct {
	tick int
	step int
}

type digestStaggerMsg struct {
	step int
}

// searchDebounceMsg asks for the query typed at generation gen to be matched.
type searchDebounceMsg struct {
	gen int
}

// searchDebounce is how long typing has to pause before the query is matched.
// Search scans the body of every indexed note — milliseconds per keystroke,
// growing with the corpus — and it runs on the event loop, so without this the
// cost lands between the key and the character appearing.
//
// ponytail: a debounce, not a goroutine. It cuts the work by roughly the length
// of a typed word and keeps the results a plain function of the model. Upgrade
// path if a corpus ever makes one match visibly slow: move updateSearchResults
// into a tea.Cmd and key the reply by gen, the way searchLoadedMsg already works.
const searchDebounce = 90 * time.Millisecond

func searchDebounceCmd(gen int) tea.Cmd {
	return tea.Tick(searchDebounce, func(time.Time) tea.Msg {
		return searchDebounceMsg{gen: gen}
	})
}

// The heatmap is four weeks wide and reveals one column per step, slow enough
// that the wipe is visible without holding up a reader who came for the numbers.
const (
	heatmapWeeks       = 4
	heatmapStaggerStep = 60 * time.Millisecond
)

func digestStaggerCmd(step int) tea.Cmd {
	return tea.Tick(heatmapStaggerStep, func(time.Time) tea.Msg {
		return digestStaggerMsg{step: step}
	})
}

type digestLoadedMsg struct {
	report digest.DigestReport
	err    error
}

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
	searchGen        int
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
	flashStep        int
	scopeWidth       int
	showBranch       bool
	uiTheme          string
	digest           bool
	digestDays       int
	digestReport     digest.DigestReport
	digestErr        string
	digestLoading    bool
	digestStagger    int
	digestFn         DigestFunc
	digestViewport   viewport.Model
	spinner          spinner.Model
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

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(getThemePalette("").headerFg)

	model := HubModel{
		notes:          notes,
		projectName:    projectName,
		branch:         branch,
		storageMode:    storageMode,
		scopeWidth:     24,
		showBranch:     true,
		uiTheme:        "auto",
		width:          80,
		height:         24,
		panel:          panelScopes,
		preview:        viewport.New(40, 18),
		captureBox:     captureBox,
		searchBox:      searchBox,
		digestDays:     1,
		digestViewport: viewport.New(40, 18),
		spinner:        sp,
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

// flash shows a transient status message and starts its colour decay. Every
// flash goes through here, so the decay step can never be left behind from a
// previous message and render the new one already faded.
func (m *HubModel) flash(message string) tea.Cmd {
	m.flashMsg = message
	m.flashStep = 0
	m.flashTick++
	return flashDecayCmd(m.flashTick, 0)
}

// flashColorFor maps a decay step to its colour: bright green, then a duller
// green, then grey just before the message clears.
func flashColorFor(step int) lipgloss.Color {
	switch step {
	case 1:
		return lipgloss.Color("34")
	case 2:
		return lipgloss.Color("244")
	default:
		return lipgloss.Color("10")
	}
}

// flashDecayCmd schedules the next step of the flash colour decay: the message
// holds at full brightness, then fades over two short steps before it clears.
func flashDecayCmd(tick, step int) tea.Cmd {
	duration := 2 * time.Second
	if step > 0 {
		duration = 400 * time.Millisecond
	}
	return tea.Tick(duration, func(time.Time) tea.Msg {
		return flashDecayMsg{tick: tick, step: step}
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

func (m HubModel) WithDigest(digestFn DigestFunc) HubModel {
	m.digestFn = digestFn
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
		commands = append(commands, m.loadSearchCmd(), m.spinner.Tick)
	}
	return tea.Batch(commands...)
}

func (m HubModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	if tick, ok := message.(spinner.TickMsg); ok {
		// Drop the loop as soon as nothing is loading, so an idle Hub stops
		// waking up ten times a second.
		if !m.busy() {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(tick)
		return m, cmd
	}
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
		flashCommand := m.flash("✓ Note updated")
		m.refreshPreview()
		var searchCommand tea.Cmd
		if reloaded.RefreshSearch {
			searchCommand = m.refreshSearchCmd()
		}
		return m, tea.Batch(searchCommand, flashCommand)
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
	if decay, ok := message.(flashDecayMsg); ok {
		if decay.tick == m.flashTick {
			if decay.step >= 2 {
				m.flashMsg = ""
				m.flashStep = 0
			} else {
				m.flashStep = decay.step + 1
				return m, flashDecayCmd(m.flashTick, m.flashStep)
			}
		}
		return m, nil
	}
	if loaded, ok := message.(digestLoadedMsg); ok {
		m.digestLoading = false
		if loaded.err != nil {
			m.digestErr = loaded.err.Error()
		} else {
			m.digestReport = loaded.report
			m.digestErr = ""
			m.digestStagger = 1
			m.refreshDigestViewport()
			return m, digestStaggerCmd(m.digestStagger)
		}
		return m, nil
	}
	if debounce, ok := message.(searchDebounceMsg); ok {
		// Only the newest tick counts; the ones behind it describe a query the
		// user has already typed past.
		if debounce.gen == m.searchGen {
			m.updateSearchResults()
			m.noteIndex = 0
			m.refreshPreview()
		}
		return m, nil
	}
	if stagger, ok := message.(digestStaggerMsg); ok {
		// Only the tick that matches the current column advances the reveal, so
		// ticks left over from a digest that was closed and reopened are ignored.
		if !m.digest || stagger.step != m.digestStagger || stagger.step >= heatmapWeeks {
			return m, nil
		}
		m.digestStagger = stagger.step + 1
		m.refreshDigestViewport()
		return m, digestStaggerCmd(m.digestStagger)
	}
	if m.capturing {
		return m.updateCapture(message)
	}
	if m.digest {
		return m.updateDigest(message)
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
				return m, m.editFn(notes[max(0, min(m.noteIndex, len(notes)-1))])
			}
		case "r":
			if m.reloadFn != nil {
				command = m.reloadCmd()
			}
		case "s":
			if m.digestFn != nil {
				m.digest = true
				return m, m.startDigestCmd()
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
	if m.digest {
		return m.digestView()
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
			"  s                          Standup digest & activity heatmap\n" +
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
	if m.searchRefreshing {
		// Kept short: the status bar is not truncated and this prefix is present
		// on every launch, including terminals below 70 columns.
		status = m.spinner.View() + " indexing · " + status
	}
	if m.flashMsg != "" {
		status = lipgloss.NewStyle().Foreground(flashColorFor(m.flashStep)).Render(m.flashMsg) + " · " + status
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
	message := "✓ Saved to inbox"
	if saved.kind == "now" {
		message = "✓ Current task updated"
	}
	return m, tea.Batch(m.refreshSearchCmd(), m.flash(message))
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
			// Enter means "show me the results now", so it cannot wait out a
			// debounce that is still in flight.
			m.searchGen++
			m.updateSearchResults()
			m.noteIndex = 0
			m.refreshPreview()
			return m, nil
		}
	}
	var command tea.Cmd
	m.searchBox, command = m.searchBox.Update(message)
	query := strings.TrimSpace(m.searchBox.Value())
	if query == m.searchQuery {
		return m, command
	}
	m.searchQuery = query
	// Clearing is instant: there is nothing to scan, and leaving stale results on
	// screen until the debounce fires would look like the delete did not register.
	if query == "" {
		m.searchGen++
		m.searchResults = nil
		m.noteIndex = 0
		m.refreshPreview()
		return m, command
	}
	m.searchGen++
	return m, tea.Batch(command, searchDebounceCmd(m.searchGen))
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
	// Retire any debounce still in flight, so it cannot repopulate the results
	// the user just cleared.
	m.searchGen++
	m.searchQuery = ""
	m.projectSearch = false
	m.searchResults = nil
	m.searchBox.Blur()
	m.searchBox.Reset()
	m.noteIndex = 0
	m.refreshPreview()
}

// busy reports whether anything the spinner stands for is still running.
func (m HubModel) busy() bool {
	return m.searchRefreshing || m.digestLoading
}

// startDigestCmd loads the digest for the current range and restarts the
// heatmap reveal. ponytail: if a second tick loop is started while one is still
// running the spinner briefly spins double speed, then both loops stop together
// once busy() goes false — not worth a generation tag.
func (m *HubModel) startDigestCmd() tea.Cmd {
	m.digestLoading = true
	m.digestStagger = 1
	digestFn, days := m.digestFn, m.digestDays
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		report, err := digestFn(days)
		return digestLoadedMsg{report: report, err: err}
	})
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
	return tea.Batch(m.loadSearchCmd(), m.spinner.Tick)
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
	// The spinner is built before the theme is known, so it picks up its colour here.
	m.spinner.Style = lipgloss.NewStyle().Foreground(getThemePalette(m.uiTheme).headerFg)
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
			if m.panel == panelScopes {
				prefix = "▶ "
			} else {
				prefix = "● "
			}
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
			if m.panel == panelNotes {
				prefix = "▶ "
			} else {
				prefix = "› "
			}
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
		m.scopeIndex = max(0, min(m.scopeIndex+delta, len(m.scopes)-1))
		m.noteIndex = 0
		return
	}
	m.noteIndex = max(0, min(m.noteIndex+delta, len(m.visibleNotes())-1))
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
	m.noteIndex = max(0, min(m.noteIndex, len(notes)-1))
	width := max(20, m.width/2-4)
	if m.width < 70 {
		width = max(20, m.width-4)
	}
	note := notes[m.noteIndex]
	// Every keystroke reaches this function, so re-render only when the output
	// would actually differ. Glamour parses and highlights the whole document.
	key := previewCacheKey{path: note.Path, content: note.Content, width: width, style: m.glamourStyle}
	if key != m.previewKey {
		// Glamour passes OSC sequences straight through, so an escape stored in a
		// note is an escape executed on display. Capture refuses them on the way
		// in; a note written in an external editor never went through capture.
		rendered := md.StripTerminalControl(note.Content)
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

func (m HubModel) updateDigest(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
		m.refreshDigestViewport()
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc", "q", "s":
			m.digest = false
			return m, nil
		case "y":
			if m.digestErr == "" {
				// OSC 52 through the terminal instead of shelling out to
				// pbcopy/xclip: works over SSH and needs no helper binary.
				termenv.Copy(digest.FormatMarkdown(m.digestReport))
				m.digest = false
				return m, m.flash("✓ Standup copied to clipboard")
			}
			return m, nil
		case "t":
			if m.digestDays == 1 {
				m.digestDays = 7
			} else {
				m.digestDays = 1
			}
			if m.digestFn != nil {
				return m, m.startDigestCmd()
			}
			return m, nil
		}
	}
	var viewportCmd tea.Cmd
	m.digestViewport, viewportCmd = m.digestViewport.Update(message)
	return m, viewportCmd
}

func (m HubModel) digestView() string {
	palette := getThemePalette(m.uiTheme)
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(palette.headerFg)

	var body strings.Builder

	if m.digestLoading {
		body.WriteString(titleStyle.Render("📊 Activity Digest") + "\n\n")
		body.WriteString(m.spinner.View() + " Loading…\n")
	} else if m.digestErr != "" {
		body.WriteString(titleStyle.Render("📊 Activity Digest") + "\n\n")
		body.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(m.digestErr) + "\n")
	} else {
		body.WriteString(m.digestViewport.View())
	}

	// The hint names the range t switches to, like the other two hints beside it
	// name what their key does. The range in effect is already in the header.
	rangeLabel := "This week"
	if m.digestDays == 7 {
		rangeLabel = "Today"
	}
	keyHint := lipgloss.NewStyle().Faint(true).Render(
		"[y] Copy Markdown · [t] " + rangeLabel + " · [Esc] Back to Hub",
	)

	content := body.String() + "\n" + keyHint
	return m.pane(content, max(30, m.width), max(3, m.height-2), true)
}

func (m *HubModel) refreshDigestViewport() {
	palette := getThemePalette(m.uiTheme)
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(palette.headerFg)
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(palette.activeFg)
	faintStyle := lipgloss.NewStyle().Faint(true)

	var content strings.Builder

	// Title
	content.WriteString(titleStyle.Render("📊 Activity Heatmap (Last 4 Weeks)") + "\n\n")

	// Heatmap
	stagger := m.digestStagger
	if stagger <= 0 {
		stagger = heatmapWeeks
	}
	content.WriteString(digest.RenderHeatmapStaggered(m.digestReport.Heatmap, heatmapWeeks, stagger))
	if m.digestReport.Streak > 0 {
		content.WriteString(fmt.Sprintf("   🔥 %d day streak", m.digestReport.Streak))
	}
	content.WriteString("\n\n")

	// Standup summary
	rangeLabel := "Today"
	if m.digestDays == 7 {
		rangeLabel = "This Week"
	}
	content.WriteString(titleStyle.Render("📝 Standup Summary — "+rangeLabel) + "\n\n")

	var tasks, captures, decisions []digest.ActivityItem
	for _, item := range m.digestReport.Items {
		switch item.Kind {
		case "task":
			tasks = append(tasks, item)
		case "capture":
			captures = append(captures, item)
		case "decision":
			decisions = append(decisions, item)
		}
	}

	if len(tasks) > 0 || len(captures) > 0 {
		content.WriteString(sectionStyle.Render("✅ Completed") + "\n")
		for _, item := range tasks {
			content.WriteString(fmt.Sprintf("  ✓ %s", item.Summary))
			if item.Project != "" {
				content.WriteString(faintStyle.Render(" (" + item.Project + ")"))
			}
			content.WriteString("\n")
		}
		for _, item := range captures {
			content.WriteString(fmt.Sprintf("  • %s", item.Summary))
			if item.Project != "" {
				content.WriteString(faintStyle.Render(" (" + item.Project + ")"))
			}
			content.WriteString("\n")
		}
		content.WriteString("\n")
	}

	if len(decisions) > 0 {
		content.WriteString(sectionStyle.Render("🏗️ Decisions") + "\n")
		for _, item := range decisions {
			content.WriteString(fmt.Sprintf("  ⚖ %s", item.Summary))
			if item.Project != "" {
				content.WriteString(faintStyle.Render(" (" + item.Project + ")"))
			}
			content.WriteString("\n")
		}
		content.WriteString("\n")
	}

	if m.digestReport.CurrentTask != "" {
		content.WriteString(sectionStyle.Render("⚡ Currently Working On") + "\n")
		for _, line := range strings.Split(m.digestReport.CurrentTask, "\n") {
			if strings.TrimSpace(line) != "" {
				content.WriteString("  " + line + "\n")
			}
		}
		content.WriteString("\n")
	}

	if len(tasks) == 0 && len(captures) == 0 && len(decisions) == 0 && m.digestReport.CurrentTask == "" {
		content.WriteString(faintStyle.Render("No activity recorded in this period.") + "\n")
	}

	m.digestViewport.Width = max(20, m.width-4)
	m.digestViewport.Height = max(3, m.height-8)
	m.digestViewport.SetContent(content.String())
}
