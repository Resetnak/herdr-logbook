package app

import (
	"fmt"
	"strings"

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
	name  string
	types map[NoteType]bool
}

type CaptureFunc func(text string, global bool) ([]Note, error)
type ReloadFunc func() ([]Note, error)
type SearchLoadFunc func() ([]searchindex.Entry, error)
type AuthorFunc func(kind, title string) ([]Note, error)
type EditFunc func(note Note) tea.Cmd

type searchLoadedMsg struct {
	entries []searchindex.Entry
	err     error
}

type NotesReloadedMsg struct {
	Notes []Note
	Err   error
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
	authorKind       string
	authorFn         AuthorFunc
	editFn           EditFunc
	previewRenderer  *glamour.TermRenderer
	previewWidth     int
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
		width:       80,
		height:      24,
		panel:       panelScopes,
		preview:     viewport.New(40, 18),
		captureBox:  captureBox,
		searchBox:   searchBox,
		scopes: []scopeItem{
			{"Now", map[NoteType]bool{NoteNow: true}},
			{"Project Inbox", map[NoteType]bool{NoteProjectInbox: true}},
			{"Project Notes", map[NoteType]bool{NoteProjectNote: true}},
			{"Decisions", map[NoteType]bool{NoteDecision: true}},
			{"Global Inbox", map[NoteType]bool{NoteGlobalInbox: true}},
			{"All Notes", nil},
		},
	}
	model.refreshPreview()
	return model
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

func (m HubModel) WithActions(captureFn CaptureFunc, reloadFn ReloadFunc) HubModel {
	m.captureFn = captureFn
	m.reloadFn = reloadFn
	return m
}

func (m HubModel) WithSearch(entries []searchindex.Entry, loadFn SearchLoadFunc) HubModel {
	m.searchEntries = entries
	m.searchLoadFn = loadFn
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
	if m.searchLoadFn == nil {
		return nil
	}
	return m.loadSearchCmd()
}

func (m HubModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	if reloaded, ok := message.(NotesReloadedMsg); ok {
		if reloaded.Err != nil {
			m.captureErr = reloaded.Err.Error()
		} else {
			m.notes = reloaded.Notes
			m.captureErr = ""
			m.refreshPreview()
		}
		return m, nil
	}
	if loaded, ok := message.(searchLoadedMsg); ok {
		if loaded.err != nil {
			m.captureErr = loaded.err.Error()
		} else {
			m.searchEntries = loaded.entries
			m.captureErr = ""
			m.updateSearchResults()
		}
		return m, nil
	}
	if m.capturing {
		return m.updateCapture(message)
	}
	if m.searching {
		return m.updateSearch(message)
	}

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
		case "e":
			notes := m.visibleNotes()
			if m.editFn != nil && len(notes) > 0 {
				return m, m.editFn(notes[clamp(m.noteIndex, 0, len(notes)-1)])
			}
		case "r":
			if m.reloadFn != nil {
				notes, err := m.reloadFn()
				if err != nil {
					m.captureErr = err.Error()
				} else {
					m.notes = notes
					m.captureErr = ""
					m.refreshPreview()
				}
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
		var command tea.Cmd
		m.preview, command = m.preview.Update(message)
		return m, command
	}
	return m, nil
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
		case "ctrl+s":
			if strings.TrimSpace(m.captureBox.Value()) == "" {
				m.captureErr = "Capture text cannot be empty."
				return m, nil
			}
			oldPaths := make(map[string]bool, len(m.notes))
			for _, note := range m.notes {
				oldPaths[note.Path] = true
			}
			var notes []Note
			var err error
			if m.authorKind != "" {
				if m.authorFn == nil {
					m.captureErr = "Authoring is unavailable."
					return m, nil
				}
				notes, err = m.authorFn(m.authorKind, strings.TrimSpace(m.captureBox.Value()))
			} else {
				if m.captureFn == nil {
					m.captureErr = "Capture is unavailable."
					return m, nil
				}
				notes, err = m.captureFn(m.captureBox.Value(), m.captureGlobal)
			}
			if err != nil {
				m.captureErr = err.Error()
				return m, nil
			}
			m.notes = notes
			m.capturing = false
			m.captureErr = ""
			m.captureBox.Blur()
			m.captureBox.Reset()
			kind := m.authorKind
			m.authorKind = ""
			m.refreshPreview()
			if kind != "" && m.editFn != nil {
				for _, note := range notes {
					if !oldPaths[note.Path] {
						return m, m.editFn(note)
					}
				}
			}
			if m.quitAfterCapture {
				return m, tea.Quit
			}
			return m, nil
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
		return pane("Herdr Logbook\n\nTab/Shift+Tab or h/l: move panels\nj/k or arrows: move\ng/G: first/last\nEnter/v: preview\n/: search notes · p: find project\nc/C: project/global capture\nn/d: note/decision · e: edit\nr: refresh\nq: quit\n?: close help", max(30, m.width), max(3, m.height-2))
	}

	availableHeight := max(3, m.height-2)
	var body string
	if m.width >= 110 {
		scopeWidth := 22
		noteWidth := max(28, m.width/3)
		previewWidth := max(30, m.width-scopeWidth-noteWidth-4)
		body = lipgloss.JoinHorizontal(lipgloss.Top,
			pane(m.scopesView(), scopeWidth, availableHeight),
			pane(m.notesView(), noteWidth, availableHeight),
			pane(m.previewView(), previewWidth, availableHeight),
		)
	} else if m.width >= 70 {
		content := m.notesView()
		if m.panel == panelPreview {
			content = m.previewView()
		}
		body = lipgloss.JoinHorizontal(lipgloss.Top,
			pane(m.scopesView(), 24, availableHeight),
			pane(content, max(30, m.width-26), availableHeight),
		)
	} else {
		content := m.scopesView()
		if m.panel == panelNotes {
			content = m.notesView()
		} else if m.panel == panelPreview {
			content = m.previewView()
		}
		body = pane(content, max(20, m.width), availableHeight)
	}

	status := fmt.Sprintf("%s · %s · %s store · / search · ? help", m.projectName, m.branch, m.storageMode)
	if m.searching {
		status = m.searchBox.View() + " · Esc clear · Enter results"
	} else if m.searchQuery != "" {
		status = fmt.Sprintf("search: %s · Esc clear · %s", m.searchQuery, status)
	}
	if m.captureErr != "" {
		status = m.captureErr + " · " + status
	}
	return body + "\n" + lipgloss.NewStyle().Faint(true).Render(status)
}

func (m HubModel) captureView() string {
	title := "Project capture"
	if m.captureGlobal {
		title = "Global capture"
	}
	if m.authorKind == "note" {
		title = "New project note — enter a title"
	} else if m.authorKind == "decision" {
		title = "New decision — enter a title"
	}
	body := title + "\n\n" + m.captureBox.View()
	if m.captureErr != "" {
		body += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(m.captureErr)
	}
	body += "\n" + lipgloss.NewStyle().Faint(true).Render("Ctrl+S save · Esc cancel")
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Width(max(24, m.captureBox.Width()+2)).
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

func (m *HubModel) resize(width, height int) {
	m.width, m.height = width, height
	m.captureBox.SetWidth(max(20, width-8))
	m.captureBox.SetHeight(min(12, max(3, height-8)))
}

func (m HubModel) scopesView() string {
	var output strings.Builder
	output.WriteString("Scopes\n\n")
	for index, scope := range m.scopes {
		prefix := "  "
		if index == m.scopeIndex {
			prefix = "● "
		}
		output.WriteString(prefix + scope.name + "\n")
	}
	return output.String()
}

func (m HubModel) notesView() string {
	notes := m.visibleNotes()
	var output strings.Builder
	if m.searchQuery != "" {
		output.WriteString("Search results\n\n")
	} else {
		output.WriteString("Notes\n\n")
	}
	if len(notes) == 0 {
		if m.scopeIndex == 0 {
			output.WriteString("No current context yet. Press c to capture something.\n")
		} else {
			output.WriteString("No notes in this scope yet. Press c to capture something.\n")
		}
		return output.String()
	}
	for index, note := range notes {
		prefix := "  "
		if index == m.noteIndex {
			prefix = "› "
		}
		output.WriteString(prefix + note.Title + "\n")
	}
	return output.String()
}

func (m HubModel) previewView() string {
	if len(m.visibleNotes()) == 0 {
		return "Preview\n\nSelect a scope with notes or press c to capture something.\n"
	}
	return "Preview\n\n" + m.preview.View()
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

func (m *HubModel) move(delta int) {
	if m.panel == panelScopes {
		m.scopeIndex = clamp(m.scopeIndex+delta, 0, len(m.scopes)-1)
		m.noteIndex = 0
		return
	}
	m.noteIndex = clamp(m.noteIndex+delta, 0, len(m.visibleNotes())-1)
}

func (m *HubModel) moveTo(end bool) {
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
		m.preview.SetContent("")
		return
	}
	m.noteIndex = clamp(m.noteIndex, 0, len(notes)-1)
	width := max(20, m.width/2-4)
	if m.width < 70 {
		width = max(20, m.width-4)
	}
	rendered := notes[m.noteIndex].Content
	// ponytail: cache the renderer; NewTermRenderer loads chroma styles and is the
	// expensive part, so only rebuild it when the wrap width actually changes.
	if m.previewRenderer == nil || m.previewWidth != width {
		if renderer, err := glamour.NewTermRenderer(glamour.WithAutoStyle(), glamour.WithWordWrap(width)); err == nil {
			m.previewRenderer = renderer
			m.previewWidth = width
		}
	}
	if m.previewRenderer != nil {
		if output, renderErr := m.previewRenderer.Render(rendered); renderErr == nil {
			rendered = output
		}
	}
	m.preview.Width = width
	m.preview.Height = max(3, m.height-6)
	m.preview.SetContent(rendered)
}

func pane(content string, width, height int) string {
	return lipgloss.NewStyle().Width(max(1, width)).Height(max(1, height)).Padding(0, 1).Render(content)
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
