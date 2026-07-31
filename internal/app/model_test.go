package app

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Resetnak/herdr-logbook/internal/digest"
	searchindex "github.com/Resetnak/herdr-logbook/internal/index"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestHubWideViewShowsThreePanesAndActionableEmptyState(t *testing.T) {
	model := NewHub(nil, "api", "feature/login", "central")
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	view := updated.(HubModel).View()
	for _, want := range []string{"Scopes", "Notes", "Preview", "Current context is unavailable", "Reopen Logbook"} {
		if !strings.Contains(view, want) {
			t.Fatalf("wide View() missing %q:\n%s", want, view)
		}
	}
}

func TestHubNarrowViewShowsOneActivePanel(t *testing.T) {
	model := NewHub([]Note{{Title: "Now", Type: NoteNow, Content: "# Now\n\nWork"}}, "api", "main", "central")
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 60, Height: 24})
	view := updated.(HubModel).View()
	if !strings.Contains(view, "Scopes") || strings.Contains(view, "Notes\n") || strings.Contains(view, "Preview\n") {
		t.Fatalf("narrow View() did not isolate active panel:\n%s", view)
	}
}

func TestHubNavigationSelectsNoteAndPreview(t *testing.T) {
	model := NewHub([]Note{
		{Title: "Now", Type: NoteNow, Content: "# Now\n\nCurrent"},
		{Title: "Inbox", Type: NoteProjectInbox, Content: "# Inbox\n\nSaved"},
	}, "api", "main", "central")
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRight})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.panel != panelPreview || !strings.Contains(model.View(), "Current") {
		t.Fatalf("navigation panel=%d view=%s", model.panel, model.View())
	}
}

// Glamour passes OSC sequences through untouched, so a note body carrying one
// reaches the terminal on preview: OSC 52 rewrites the reader's system
// clipboard. Capture rejects control characters, but a note written in an
// external editor never went through capture, so the preview must strip them.
func TestHubPreviewStripsTerminalControlCharacters(t *testing.T) {
	model := NewHub([]Note{
		{Title: "Now", Type: NoteNow, Content: "# Now\n\nbody \x1b]52;c;aGFja2Vk\a and \x1b]0;pwned\a\n"},
	}, "api", "main", "central").WithStyle("notty")
	model, _ = updateHub(model, tea.WindowSizeMsg{Width: 120, Height: 30})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRight})

	// lipgloss emits its own ANSI for borders and colour; what must not survive
	// is an escape that came out of the note.
	view := model.View()
	for _, leaked := range []string{"\x1b]52;", "\x1b]0;", "\a"} {
		if strings.Contains(view, leaked) {
			t.Fatalf("preview leaked %q from the note body:\n%q", leaked, view)
		}
	}
	if !strings.Contains(view, "body") {
		t.Fatalf("preview dropped the note body:\n%s", view)
	}
}

func TestHubPreviewRendererIsCachedByWidth(t *testing.T) {
	model := NewHub([]Note{
		{Title: "Now", Type: NoteNow, Content: "# Now\n\nCurrent"},
		{Title: "Later", Type: NoteNow, Content: "# Later\n\nNext"},
	}, "api", "main", "central")
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRight})
	renderer := model.previewRenderer
	if renderer == nil {
		t.Fatal("preview renderer was not initialized")
	}

	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyDown})
	if model.previewRenderer != renderer {
		t.Fatal("preview renderer was rebuilt for an unchanged width")
	}

	model, _ = updateHub(model, tea.WindowSizeMsg{Width: 120, Height: 30})
	if model.previewRenderer == renderer {
		t.Fatalf("preview renderer %p was reused after wrap width changed to %d", renderer, model.previewWidth)
	}
}

func TestHubScopeNavigationFiltersNotes(t *testing.T) {
	model := NewHub([]Note{
		{Title: "Now", Type: NoteNow, Content: "# Now"},
		{Title: "Inbox", Type: NoteProjectInbox, Content: "# Inbox"},
	}, "api", "main", "central")
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyDown})
	if got := model.visibleNotes(); len(got) != 1 || got[0].Title != "Inbox" {
		t.Fatalf("visibleNotes() = %#v", got)
	}
}

func TestHubEmptyStatesDescribeScopeActions(t *testing.T) {
	wants := map[string]string{
		"Now":           "Reopen Logbook",
		"Project Inbox": "Press c to capture",
		"Project Notes": "Press n to create",
		"Decisions":     "Press d to create",
		"Global Inbox":  "Press C to capture",
		"All Notes":     "Press c to capture or n to create",
	}
	model := NewHub(nil, "api", "main", "central")
	for index, scope := range model.scopes {
		t.Run(scope.name, func(t *testing.T) {
			want, ok := wants[scope.name]
			if !ok {
				t.Fatalf("scope %q has no expected empty-state guidance", scope.name)
			}
			model.scopeIndex = index
			if view := model.View(); !strings.Contains(view, want) {
				t.Fatalf("empty state missing %q:\n%s", want, view)
			}
		})
	}
}

func TestHubEmptySearchDoesNotSuggestCapturing(t *testing.T) {
	model := NewHub(nil, "api", "main", "central")
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("missing")})
	view := model.View()
	if !strings.Contains(view, "No matching notes") || strings.Contains(view, "Press c to capture") {
		t.Fatalf("empty search showed the wrong guidance:\n%s", view)
	}
}

func TestHubSearchShowsRefreshingUntilInitialIndexLoads(t *testing.T) {
	model := NewHub(nil, "api", "main", "central").WithSearch(nil, func() ([]searchindex.Entry, error) {
		return nil, nil
	})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("missing")})
	if view := model.View(); !strings.Contains(view, "Refreshing search index...") {
		t.Fatalf("pending search did not show refresh state:\n%s", view)
	}

	model, _ = updateHub(model, runSearchLoadCommand(t, model.Init()))
	if view := model.View(); !strings.Contains(view, "No matching notes") {
		t.Fatalf("completed empty search did not show no-match state:\n%s", view)
	}
}

func TestHubCaptureModalSavesProjectNote(t *testing.T) {
	var capturedText string
	var capturedGlobal bool
	model := NewHub(nil, "api", "main", "central").WithActions(
		func(text string, global bool) (string, []Note, error) {
			capturedText = text
			capturedGlobal = global
			return "/notes/inbox/2026-07.md", []Note{{Title: "Captured", Type: NoteProjectInbox, Content: text}}, nil
		},
		nil,
	)

	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if !model.capturing || model.captureGlobal {
		t.Fatalf("capture state = capturing %t, global %t", model.capturing, model.captureGlobal)
	}
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("remember this")})
	model, _ = runCaptureSave(t, model, tea.KeyMsg{Type: tea.KeyCtrlS})

	if capturedText != "remember this" || capturedGlobal {
		t.Fatalf("capture callback = (%q, %t)", capturedText, capturedGlobal)
	}
	if model.capturing {
		t.Fatal("capture modal remained open after a successful save")
	}
	if len(model.notes) != 1 || model.notes[0].Title != "Captured" {
		t.Fatalf("notes after capture = %#v", model.notes)
	}
}

func TestHubCaptureRefreshesSearchIndex(t *testing.T) {
	searchLoads := 0
	model := NewHub(nil, "api", "main", "central").
		WithActions(func(text string, global bool) (string, []Note, error) {
			return "/notes/inbox/2026-07.md", []Note{{Title: "Inbox", Type: NoteProjectInbox, Content: text}}, nil
		}, nil).
		WithSearch(nil, func() ([]searchindex.Entry, error) {
			searchLoads++
			return []searchindex.Entry{{Title: "Inbox", ProjectName: "api", Content: "first ever note"}}, nil
		})
	model = completeInitialSearch(t, model)

	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("first ever note")})
	model, command := runCaptureSave(t, model, tea.KeyMsg{Type: tea.KeyCtrlS})
	loaded := runSearchLoadCommand(t, command)
	model, _ = updateHub(model, loaded)
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("first ever")})
	model = applySearchDebounce(model)

	if searchLoads != 2 || len(model.searchResults) != 1 {
		t.Fatalf("search refresh = loads %d, results %#v", searchLoads, model.searchResults)
	}
}

func TestHubCaptureModalSupportsGlobalAndCancel(t *testing.T) {
	called := false
	model := NewHub(nil, "api", "main", "central").WithActions(
		func(string, bool) (string, []Note, error) {
			called = true
			return "", nil, nil
		},
		nil,
	)

	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})
	if !model.capturing || !model.captureGlobal {
		t.Fatalf("global capture state = capturing %t, global %t", model.capturing, model.captureGlobal)
	}
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.capturing || called {
		t.Fatalf("cancel state = capturing %t, callback called %t", model.capturing, called)
	}
}

func TestHubCaptureErrorKeepsModalOpen(t *testing.T) {
	model := NewHub(nil, "api", "main", "central").WithActions(
		func(string, bool) (string, []Note, error) { return "", nil, errors.New("disk full") },
		nil,
	)
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("note")})
	model, _ = runCaptureSave(t, model, tea.KeyMsg{Type: tea.KeyCtrlS})

	if !model.capturing || !strings.Contains(model.View(), "disk full") {
		t.Fatalf("capture error was not visible in an open modal:\n%s", model.View())
	}
}

func TestHubRefreshReloadsNotes(t *testing.T) {
	reloads := 0
	searchLoads := 0
	model := NewHub(nil, "api", "main", "central").
		WithActions(nil, func() ([]Note, error) {
			reloads++
			return []Note{{Title: "Fresh", Type: NoteNow, Content: "new"}}, nil
		}).
		WithSearch(nil, func() ([]searchindex.Entry, error) {
			searchLoads++
			return nil, nil
		})
	model = completeInitialSearch(t, model)

	model, command := updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model, searchCommand := updateHub(model, runReloadCommand(t, command))
	runSearchLoadCommand(t, searchCommand)
	if reloads != 1 || searchLoads != 2 || len(model.notes) != 1 || model.notes[0].Title != "Fresh" {
		t.Fatalf("refresh = reloads %d, search loads %d, notes %#v", reloads, searchLoads, model.notes)
	}
}

func TestHubRefreshPreservesPreviewViewportInput(t *testing.T) {
	searchLoads := 0
	notes := []Note{{Title: "Long", Type: NoteNow, Content: strings.Repeat("line\n", 100)}}
	model := NewHub(notes, "api", "main", "central").
		WithActions(nil, func() ([]Note, error) { return notes, nil }).
		WithSearch(nil, func() ([]searchindex.Entry, error) {
			searchLoads++
			return nil, nil
		})
	model = completeInitialSearch(t, model)
	model.width = 120
	model.height = 12
	model.panel = panelPreview
	model.refreshPreview()
	model.preview.KeyMap.PageDown.SetKeys("r")

	model, command := updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model, searchCommand := updateHub(model, runReloadCommand(t, command))
	runSearchLoadCommand(t, searchCommand)
	if model.preview.YOffset == 0 || searchLoads != 2 {
		t.Fatalf("refresh = preview offset %d, search loads %d", model.preview.YOffset, searchLoads)
	}
}

func TestHubStatusPrioritizesActionError(t *testing.T) {
	model := NewHub(nil, "api", "main", "central")
	model.captureErr = "editor failed"
	model.searchErr = "index failed"
	view := model.View()
	if !strings.Contains(view, "editor failed") || strings.Contains(view, "index failed") {
		t.Fatalf("status did not prioritize the action error:\n%s", view)
	}
}

func TestHubSearchShowsCrossProjectResults(t *testing.T) {
	model := NewHub(nil, "api", "main", "central").WithSearch([]searchindex.Entry{
		{Path: "/payments/notes/cache.md", ProjectID: "payments", ProjectName: "payments", Title: "Cache policy", Content: "Bound entries."},
	}, nil)
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("cache")})
	model = applySearchDebounce(model)

	view := model.View()
	if !model.searching || !strings.Contains(view, "Cache policy") || !strings.Contains(view, "payments") {
		t.Fatalf("search view missing cross-project result:\n%s", view)
	}
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.searching || len(model.searchResults) != 0 {
		t.Fatalf("Esc did not clear search: searching %t, results %#v", model.searching, model.searchResults)
	}
}

func TestHubProjectSearchFiltersByProjectName(t *testing.T) {
	model := NewHub(nil, "api", "main", "central").WithSearch([]searchindex.Entry{
		{Path: "/payments/a.md", ProjectName: "payments", Title: "Runbook"},
		{Path: "/api/b.md", ProjectName: "api", Title: "Runbook"},
	}, nil)
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("pay")})
	model = applySearchDebounce(model)
	view := model.View()
	if !strings.Contains(view, "Runbook · payments") || strings.Contains(view, "Runbook · api") {
		t.Fatalf("project search view was not filtered:\n%s", view)
	}
}

func TestHubSetsCurrentTask(t *testing.T) {
	got := ""
	model := NewHub([]Note{{Path: "/notes/now.md", Title: "Now", Type: NoteNow}}, "api", "main", "central").
		WithAuthoring(func(kind, title string) (string, []Note, error) {
			got = kind + ":" + title
			return "/notes/now.md", nil, nil
		}, nil)

	m, _ := updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if !strings.Contains(m.View(), "Current task") {
		t.Fatalf("t did not open the current-task modal:\n%s", m.View())
	}
	m, _ = updateHub(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Rotate the signing tokens")})
	m, _ = runCaptureSave(t, m, tea.KeyMsg{Type: tea.KeyCtrlS})
	if got != "now:Rotate the signing tokens" {
		t.Fatalf("author action = %q", got)
	}
	if m.capturing {
		t.Fatal("Ctrl+S should close the modal")
	}
	if m.flashMsg != "✓ Current task updated" {
		t.Fatalf("flash = %q", m.flashMsg)
	}
}

func TestHubSaveAndEditCurrentTaskOpensNowFile(t *testing.T) {
	edited := ""
	now := Note{Path: "/notes/now.md", Title: "Now", Type: NoteNow}
	model := NewHub([]Note{now}, "api", "main", "central").
		WithAuthoring(func(string, string) (string, []Note, error) {
			return now.Path, []Note{
				now,
				{Path: "/notes/inbox/2026-07.md", Title: "Inbox", Type: NoteProjectInbox},
			}, nil
		}, func(note Note) tea.Cmd {
			return func() tea.Msg {
				edited = note.Path
				return nil
			}
		})

	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Rotate the signing tokens")})
	_, command := runCaptureSave(t, model, tea.KeyMsg{Type: tea.KeyCtrlE})
	if command == nil {
		t.Fatal("Ctrl+E did not open an editor")
	}
	command()
	if edited != now.Path {
		t.Fatalf("Ctrl+E edited %q, want %q", edited, now.Path)
	}
}

func TestHubAuthoringAndEditorActions(t *testing.T) {
	authorKind := ""
	edited := ""
	model := NewHub([]Note{{Path: "/notes/a.md", Title: "A", Type: NoteNow}}, "api", "main", "central").
		WithAuthoring(func(kind, title string) (string, []Note, error) {
			authorKind = kind + ":" + title
			return "/notes/new.md", []Note{{Path: "/notes/new.md", Title: title, Type: NoteProjectNote}}, nil
		}, func(note Note) tea.Cmd {
			return func() tea.Msg {
				edited = note.Path
				return nil
			}
		})

	// 1. Test Ctrl+S (saves note, does NOT open editor)
	m1, _ := updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m1, _ = updateHub(m1, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Runbook")})
	m1, cmd1 := runCaptureSave(t, m1, tea.KeyMsg{Type: tea.KeyCtrlS})
	if authorKind != "note:Runbook" {
		t.Fatalf("author action = %q", authorKind)
	}
	if cmd1 != nil {
		cmd1()
	}
	if edited != "" {
		t.Fatalf("Ctrl+S should not open editor, but opened: %q", edited)
	}
	if m1.capturing {
		t.Fatal("Ctrl+S should close capture modal")
	}

	// 2. Test Ctrl+E (saves note AND opens editor)
	m2, _ := updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m2, _ = updateHub(m2, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Runbook 2")})
	_, cmd2 := runCaptureSave(t, m2, tea.KeyMsg{Type: tea.KeyCtrlE})
	if cmd2 == nil {
		t.Fatal("Ctrl+E should return edit command")
	}
	cmd2()
	if edited != "/notes/new.md" {
		t.Fatalf("Ctrl+E edited path = %q, want /notes/new.md", edited)
	}
}

func TestHubEditorReturnRefreshesSearchIndex(t *testing.T) {
	searchLoads := 0
	model := NewHub(nil, "api", "main", "central").WithSearch(nil, func() ([]searchindex.Entry, error) {
		searchLoads++
		return nil, nil
	})
	model = completeInitialSearch(t, model)

	model, command := updateHub(model, NotesReloadedMsg{Notes: []Note{{Title: "Edited", Type: NoteProjectNote}}, RefreshSearch: true})
	runSearchLoadCommand(t, command)
	if searchLoads != 2 || len(model.notes) != 1 || model.notes[0].Title != "Edited" {
		t.Fatalf("editor return = search loads %d, notes %#v", searchLoads, model.notes)
	}
}

func TestHubEditorErrorStillRefreshesSearchIndex(t *testing.T) {
	searchLoads := 0
	model := NewHub(nil, "api", "main", "central").WithSearch(nil, func() ([]searchindex.Entry, error) {
		searchLoads++
		return nil, nil
	})
	model = completeInitialSearch(t, model)

	model, command := updateHub(model, NotesReloadedMsg{Err: errors.New("editor failed"), RefreshSearch: true})
	loaded := runSearchLoadCommand(t, command)
	model, _ = updateHub(model, loaded)
	if searchLoads != 2 || !strings.Contains(model.captureErr, "editor failed") {
		t.Fatalf("editor error = search loads %d, capture error %q", searchLoads, model.captureErr)
	}
}

func TestHubEditorSetupErrorDoesNotRefreshSearchIndex(t *testing.T) {
	searchLoads := 0
	model := NewHub(nil, "api", "main", "central").WithSearch(nil, func() ([]searchindex.Entry, error) {
		searchLoads++
		return nil, nil
	})
	model = completeInitialSearch(t, model)

	model, command := updateHub(model, NotesReloadedMsg{Err: errors.New("editor unavailable")})
	if command != nil || searchLoads != 1 || !strings.Contains(model.captureErr, "editor unavailable") {
		t.Fatalf("editor setup error = command %v, search loads %d, capture error %q", command != nil, searchLoads, model.captureErr)
	}
}

func TestHubFailedSearchRefreshDoesNotReportNoMatches(t *testing.T) {
	model := NewHub(nil, "api", "main", "central").WithSearch(nil, func() ([]searchindex.Entry, error) {
		return nil, errors.New("scan failed")
	})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("missing")})
	loaded := runSearchLoadCommand(t, model.Init())
	model, _ = updateHub(model, loaded)

	view := model.View()
	if !strings.Contains(view, "Search index refresh failed") || strings.Contains(view, "No matching notes") {
		t.Fatalf("failed search showed the wrong empty state:\n%s", view)
	}
}

func TestHubCoalescesOverlappingSearchRefreshes(t *testing.T) {
	searchLoads := 0
	model := NewHub(nil, "api", "main", "central").
		WithActions(nil, func() ([]Note, error) { return nil, nil }).
		WithSearch(nil, func() ([]searchindex.Entry, error) {
			searchLoads++
			return []searchindex.Entry{{Title: fmt.Sprintf("refresh-%d", searchLoads)}}, nil
		})

	initialCommand := model.Init()
	model, firstReload := updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model, secondReload := updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model, _ = updateHub(model, runReloadCommand(t, firstReload))
	model, _ = updateHub(model, runReloadCommand(t, secondReload))
	if searchLoads != 0 || !model.searchDirty {
		t.Fatalf("overlapping refreshes were not coalesced: loads %d, dirty %t", searchLoads, model.searchDirty)
	}

	initialResult := runSearchLoadCommand(t, initialCommand)
	model, followUpCommand := updateHub(model, initialResult)
	followUpResult := runSearchLoadCommand(t, followUpCommand)
	model, _ = updateHub(model, followUpResult)

	if searchLoads != 2 || model.searchRefreshing || model.searchDirty || len(model.searchEntries) != 1 || model.searchEntries[0].Title != "refresh-2" {
		t.Fatalf("coalesced refresh = loads %d, refreshing %t, dirty %t, entries %#v", searchLoads, model.searchRefreshing, model.searchDirty, model.searchEntries)
	}
}

func TestHubPanelAndScopeKeysMoveFocusAndSelection(t *testing.T) {
	notes := []Note{
		{Title: "Now", Type: NoteNow, Content: "# Now"},
		{Title: "Cache", Type: NoteProjectNote, Content: "# Cache"},
	}
	model := NewHub(notes, "api", "main", "central")

	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyTab})
	if model.panel != panelNotes {
		t.Fatalf("Tab panel = %d", model.panel)
	}
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyShiftTab})
	if model.panel != panelScopes {
		t.Fatalf("Shift+Tab panel = %d", model.panel)
	}
	// h at the leftmost panel must stay put rather than wrap around.
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if model.panel != panelScopes {
		t.Fatalf("h at the left edge = %d", model.panel)
	}
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if model.panel != panelScopes {
		t.Fatalf("l then h = %d", model.panel)
	}

	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	if model.panel != panelPreview {
		t.Fatalf("v panel = %d", model.panel)
	}

	// Enter walks scopes → notes → preview only while the scope has notes.
	stepping := NewHub(notes, "api", "main", "central").WithView("all")
	stepping, _ = updateHub(stepping, tea.KeyMsg{Type: tea.KeyEnter})
	if stepping.panel != panelNotes {
		t.Fatalf("first Enter = %d", stepping.panel)
	}
	stepping, _ = updateHub(stepping, tea.KeyMsg{Type: tea.KeyEnter})
	if stepping.panel != panelPreview {
		t.Fatalf("second Enter = %d", stepping.panel)
	}

	// j/k move the selection inside the notes panel and clamp at both ends.
	selecting := NewHub(notes, "api", "main", "central").WithView("all")
	selecting, _ = updateHub(selecting, tea.KeyMsg{Type: tea.KeyRight})
	selecting, _ = updateHub(selecting, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if selecting.noteIndex != 1 {
		t.Fatalf("j noteIndex = %d", selecting.noteIndex)
	}
	selecting, _ = updateHub(selecting, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	selecting, _ = updateHub(selecting, tea.KeyMsg{Type: tea.KeyUp})
	if selecting.noteIndex != 0 {
		t.Fatalf("k noteIndex = %d", selecting.noteIndex)
	}
}

func TestHubQuitKeysAndHelpToggle(t *testing.T) {
	model := NewHub(nil, "api", "main", "central")
	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'q'}},
		{Type: tea.KeyCtrlC},
	} {
		if _, cmd := updateHub(model, key); cmd == nil {
			t.Fatalf("%v did not return a quit command", key)
		}
	}

	helped, _ := updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if !helped.help {
		t.Fatal("? did not open help")
	}
	view := helped.View()
	for _, want := range []string{"Quick Start", "Navigation:", "Set current task"} {
		if !strings.Contains(view, want) {
			t.Fatalf("help view missing %q:\n%s", want, view)
		}
	}
	closed, _ := updateHub(helped, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if closed.help {
		t.Fatal("? did not close help")
	}
}

func TestHubNotesReloadedAndFlashExpiry(t *testing.T) {
	model := NewHub(nil, "api", "main", "central")

	reloaded, cmd := updateHub(model, NotesReloadedMsg{Notes: []Note{{Title: "Cache", Type: NoteProjectNote, Content: "# Cache"}}})
	if len(reloaded.notes) != 1 || reloaded.flashMsg != "✓ Note updated" || cmd == nil {
		t.Fatalf("reload = %d notes, flash %q, cmd %v", len(reloaded.notes), reloaded.flashMsg, cmd)
	}
	if !strings.Contains(reloaded.View(), "✓ Note updated") {
		t.Fatalf("flash is not visible in the status line:\n%s", reloaded.View())
	}

	// A stale tick must not decay or clear a newer flash.
	stale, _ := updateHub(reloaded, flashDecayMsg{tick: reloaded.flashTick - 1, step: 2})
	if stale.flashMsg == "" || stale.flashStep != 0 {
		t.Fatalf("a stale flashDecayMsg touched the flash: %q step %d", stale.flashMsg, stale.flashStep)
	}
	current, _ := updateHub(reloaded, flashDecayMsg{tick: reloaded.flashTick, step: 2})
	if current.flashMsg != "" {
		t.Fatalf("flash survived its own final tick: %q", current.flashMsg)
	}

	failed, _ := updateHub(model, NotesReloadedMsg{Err: errors.New("editor exited with error")})
	if failed.captureErr != "editor exited with error" {
		t.Fatalf("reload error = %q", failed.captureErr)
	}
	if !strings.Contains(failed.View(), "editor exited with error") {
		t.Fatalf("reload error is not visible in the status line:\n%s", failed.View())
	}
}

func TestHubEditKeyPassesTheSelectedNote(t *testing.T) {
	edited := ""
	model := NewHub([]Note{{Path: "/store/notes/cache.md", Title: "Cache", Type: NoteProjectNote}}, "api", "main", "central").
		WithAuthoring(nil, func(note Note) tea.Cmd {
			edited = note.Path
			return nil
		}).WithView("all")

	if _, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}); edited != "/store/notes/cache.md" {
		t.Fatalf("e edited %q", edited)
	}

	// Without an edit function, or with nothing selected, e must be a no-op.
	if _, cmd := updateHub(NewHub(nil, "api", "main", "central"), tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}); cmd != nil {
		t.Fatal("e on an empty Hub returned a command")
	}
}

func TestHubAuthoringRefusesWhenNoActionIsWired(t *testing.T) {
	model := NewHub(nil, "api", "main", "central")

	authoring, _ := updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if !strings.Contains(authoring.View(), "New decision") {
		t.Fatalf("d did not open the decision modal:\n%s", authoring.View())
	}
	authoring, _ = updateHub(authoring, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Use Redis")})
	authoring, _ = updateHub(authoring, tea.KeyMsg{Type: tea.KeyCtrlS})
	if authoring.captureErr != "Authoring is unavailable." {
		t.Fatalf("author error = %q", authoring.captureErr)
	}

	capturing, _ := updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})
	if !strings.Contains(capturing.View(), "Global capture") {
		t.Fatalf("C did not open the global capture modal:\n%s", capturing.View())
	}
	capturing, _ = updateHub(capturing, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("note")})
	capturing, _ = updateHub(capturing, tea.KeyMsg{Type: tea.KeyCtrlS})
	if capturing.captureErr != "Capture is unavailable." {
		t.Fatalf("capture error = %q", capturing.captureErr)
	}

	// An empty modal is rejected before any action is called.
	empty, _ := updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	empty, _ = updateHub(empty, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("   ")})
	empty, _ = updateHub(empty, tea.KeyMsg{Type: tea.KeyCtrlS})
	if empty.captureErr != "Capture text cannot be empty." || !empty.capturing {
		t.Fatalf("empty capture error = %q, capturing = %t", empty.captureErr, empty.capturing)
	}
	if _, cmd := updateHub(empty, tea.KeyMsg{Type: tea.KeyCtrlC}); cmd == nil {
		t.Fatal("Ctrl+C in the modal did not quit")
	}
}

func TestBeginCaptureQuitsAfterSaving(t *testing.T) {
	saved := ""
	model, cmd := NewHub(nil, "api", "main", "central").
		WithActions(func(text string, global bool) (string, []Note, error) {
			saved = text
			return "/notes/inbox/2026-07.md", nil, nil
		}, nil).
		BeginCapture(false, true)
	if cmd == nil || !model.capturing {
		t.Fatalf("BeginCapture did not focus the modal: capturing = %t", model.capturing)
	}
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("standalone capture")})
	model, cmd = runCaptureSave(t, model, tea.KeyMsg{Type: tea.KeyCtrlS})
	if saved != "standalone capture" {
		t.Fatalf("captured %q", saved)
	}
	if cmd == nil {
		t.Fatal("quitAfterCapture did not quit after saving")
	}
}

func TestHubSearchEnterKeepsResultsAndCapsThemAtOneHundred(t *testing.T) {
	entries := make([]searchindex.Entry, 150)
	for i := range entries {
		entries[i] = searchindex.Entry{Path: "/payments/n.md", ProjectName: "payments", Title: "Runbook", Content: "cache"}
	}
	model := NewHub(nil, "api", "main", "central").WithSearch(entries, nil)

	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("pay")})
	model = applySearchDebounce(model)
	if len(model.searchResults) != 100 {
		t.Fatalf("project search results = %d, want the 100 cap", len(model.searchResults))
	}
	if _, cmd := updateHub(model, tea.KeyMsg{Type: tea.KeyCtrlC}); cmd == nil {
		t.Fatal("Ctrl+C while searching did not quit")
	}

	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.searching || model.panel != panelNotes || len(model.searchResults) != 100 {
		t.Fatalf("Enter left search state %t, panel %d, %d results", model.searching, model.panel, len(model.searchResults))
	}
	if !strings.Contains(model.View(), "search: pay") {
		t.Fatalf("status line does not show the active filter:\n%s", model.View())
	}
	// Esc outside the search box clears the retained query.
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.searchQuery != "" || len(model.searchResults) != 0 {
		t.Fatalf("Esc left query %q and %d results", model.searchQuery, len(model.searchResults))
	}
}

func TestNarrowViewFollowsTheFocusedPanelAndEmptyScopes(t *testing.T) {
	model := NewHub([]Note{{Title: "Cache", Type: NoteProjectNote, Content: "# Cache\n\nBody"}}, "api", "main", "central").WithView("project")
	model, _ = updateHub(model, tea.WindowSizeMsg{Width: 60, Height: 24})

	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRight})
	if !strings.Contains(model.View(), "Cache") {
		t.Fatalf("narrow notes panel:\n%s", model.View())
	}
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRight})
	if !strings.Contains(model.View(), "Body") {
		t.Fatalf("narrow preview panel:\n%s", model.View())
	}

	// A non-Now scope with nothing in it gets its own empty-state wording.
	empty := NewHub(nil, "api", "main", "central").WithView("project")
	empty, _ = updateHub(empty, tea.KeyMsg{Type: tea.KeyRight})
	if !strings.Contains(empty.View(), "No project notes yet. Press n to create one.") {
		t.Fatalf("empty scope view:\n%s", empty.View())
	}
}

func TestVisibleNotesToleratesAnOutOfRangeScope(t *testing.T) {
	model := NewHub([]Note{{Title: "Cache", Type: NoteProjectNote}}, "api", "main", "central")
	model.scopeIndex = len(model.scopes) + 3
	if notes := model.visibleNotes(); notes != nil {
		t.Fatalf("visibleNotes for an out-of-range scope = %#v", notes)
	}
}

func TestWithViewSelectsTheMatchingScope(t *testing.T) {
	model := NewHub(nil, "api", "main", "central")
	for view, want := range map[string]int{"project": 2, "global": 4, "all": 5, "now": 0, "": 0} {
		if got := model.WithView(view).scopeIndex; got != want {
			t.Fatalf("WithView(%q) scopeIndex = %d, want %d", view, got, want)
		}
	}
}

func TestWithStyleAndUIConfigOverrideDefaults(t *testing.T) {
	model := NewHub(nil, "api", "main", "central").WithStyle("notty")
	if model.glamourStyle != "notty" || model.previewRenderer != nil {
		t.Fatalf("WithStyle did not pin the style and drop the renderer: %q, %v", model.glamourStyle, model.previewRenderer)
	}

	configured := model.WithUIConfig(30, false, "nord")
	if configured.scopeWidth != 30 || configured.showBranch || configured.uiTheme != "nord" {
		t.Fatalf("WithUIConfig = %d, %t, %q", configured.scopeWidth, configured.showBranch, configured.uiTheme)
	}
	// Zero width and an empty theme must keep the constructor defaults.
	kept := model.WithUIConfig(0, true, "")
	if kept.scopeWidth != 24 || kept.uiTheme != "auto" {
		t.Fatalf("WithUIConfig overwrote defaults: %d, %q", kept.scopeWidth, kept.uiTheme)
	}
}

func TestThemePalettesAreDistinctPerTheme(t *testing.T) {
	seen := map[lipgloss.Color]string{}
	for _, theme := range []string{"Dracula", "tokyo-night", "nord", "auto"} {
		header := getThemePalette(theme).headerFg
		if other, clash := seen[header]; clash {
			t.Fatalf("themes %q and %q share headerFg %q", theme, other, header)
		}
		seen[header] = theme
	}
}

func TestInitLoadsSearchEntriesOnlyWhenALoaderIsSet(t *testing.T) {
	if cmd := NewHub(nil, "api", "main", "central").Init(); cmd != nil {
		t.Fatal("Init returned a command without a search loader")
	}

	entries := []searchindex.Entry{{Path: "/payments/a.md", ProjectName: "payments", Title: "Runbook"}}
	model := NewHub(nil, "api", "main", "central").WithSearch(nil, func() ([]searchindex.Entry, error) {
		return entries, nil
	})
	loaded := runSearchLoadCommand(t, model.Init())
	if loaded.err != nil || len(loaded.entries) != 1 {
		t.Fatalf("Init message = %#v", loaded)
	}

	model, _ = updateHub(model, loaded)
	if len(model.searchEntries) != 1 || model.captureErr != "" {
		t.Fatalf("entries = %#v, err = %q", model.searchEntries, model.captureErr)
	}

	failing := NewHub(nil, "api", "main", "central").WithSearch(nil, func() ([]searchindex.Entry, error) {
		return nil, errors.New("index unreadable")
	})
	failing, _ = updateHub(failing, runSearchLoadCommand(t, failing.Init()))
	if failing.searchErr != "index unreadable" {
		t.Fatalf("search load error = %q", failing.searchErr)
	}
}

func TestGoToFirstAndLastJumpsWithinTheFocusedPanel(t *testing.T) {
	model := NewHub([]Note{
		{Title: "One", Type: NoteProjectNote},
		{Title: "Two", Type: NoteProjectNote},
		{Title: "Three", Type: NoteProjectNote},
	}, "api", "main", "central")

	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if model.scopeIndex != len(model.scopes)-1 {
		t.Fatalf("G on scopes = %d, want %d", model.scopeIndex, len(model.scopes)-1)
	}
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if model.scopeIndex != 0 || model.noteIndex != 0 {
		t.Fatalf("g on scopes = %d/%d", model.scopeIndex, model.noteIndex)
	}

	model = model.WithView("project")
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRight})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if model.noteIndex != 2 {
		t.Fatalf("G on notes = %d, want 2", model.noteIndex)
	}
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if model.noteIndex != 0 {
		t.Fatalf("g on notes = %d, want 0", model.noteIndex)
	}

	// An empty panel must not produce a negative index.
	empty := NewHub(nil, "api", "main", "central").WithView("project")
	empty, _ = updateHub(empty, tea.KeyMsg{Type: tea.KeyRight})
	empty, _ = updateHub(empty, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if empty.noteIndex != 0 {
		t.Fatalf("G on an empty panel = %d, want 0", empty.noteIndex)
	}
}

func TestClampKeepsValuesInsideTheRange(t *testing.T) {
	cases := []struct{ value, low, high, want int }{
		{5, 0, 3, 3},
		{-2, 0, 3, 0},
		{2, 0, 3, 2},
		{7, 0, -1, 0}, // empty list: high < low collapses to low
	}
	for _, test := range cases {
		if got := clamp(test.value, test.low, test.high); got != test.want {
			t.Fatalf("clamp(%d, %d, %d) = %d, want %d", test.value, test.low, test.high, got, test.want)
		}
	}
}

// Search scans the body of every indexed note, which costs milliseconds per
// keystroke and grows with the corpus. Running it on the event loop means the
// cost is paid between the key and the character appearing, so it runs on a
// debounce instead: type freely, match once the typing pauses.
func TestHubSearchRunsOnADebounceNotOnEveryKeystroke(t *testing.T) {
	entries := []searchindex.Entry{
		{Path: "/notes/lock.md", Title: "Storage lock", ProjectName: "api", Content: "flock bounded"},
	}
	model := NewHub(nil, "api", "main", "central").WithSearch(entries, nil)
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("lock")})

	if len(model.searchResults) != 0 {
		t.Fatalf("search ran on the keystroke: %#v", model.searchResults)
	}

	model, _ = updateHub(model, searchDebounceMsg{gen: model.searchGen})
	if len(model.searchResults) != 1 || model.searchResults[0].Path != "/notes/lock.md" {
		t.Fatalf("the debounced search did not produce results: %#v", model.searchResults)
	}
}

// A debounce tick left over from an earlier keystroke must not overwrite the
// results of a newer query.
func TestHubSearchIgnoresStaleDebounceTicks(t *testing.T) {
	entries := []searchindex.Entry{
		{Path: "/notes/lock.md", Title: "Storage lock", ProjectName: "api", Content: "flock"},
	}
	model := NewHub(nil, "api", "main", "central").WithSearch(entries, nil)
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("lock")})
	stale := model.searchGen

	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("zzz")})
	model, _ = updateHub(model, searchDebounceMsg{gen: stale})
	if len(model.searchResults) != 0 {
		t.Fatalf("a stale debounce tick applied an outdated query: %#v", model.searchResults)
	}
}

// Enter means "show me the results now" — it must not wait out the debounce.
func TestHubSearchEnterAppliesTheQueryImmediately(t *testing.T) {
	entries := []searchindex.Entry{
		{Path: "/notes/lock.md", Title: "Storage lock", ProjectName: "api", Content: "flock"},
	}
	model := NewHub(nil, "api", "main", "central").WithSearch(entries, nil)
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("lock")})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyEnter})

	if len(model.searchResults) != 1 {
		t.Fatalf("Enter did not apply the query: %#v", model.searchResults)
	}
}

func updateHub(model HubModel, message tea.Msg) (HubModel, tea.Cmd) {
	updated, command := model.Update(message)
	return updated.(HubModel), command
}

// applySearchDebounce delivers the tick the search box scheduled, standing in
// for the pause in typing that fires it at runtime.
func applySearchDebounce(model HubModel) HubModel {
	model, _ = updateHub(model, searchDebounceMsg{gen: model.searchGen})
	return model
}

func completeInitialSearch(t *testing.T, model HubModel) HubModel {
	t.Helper()
	loaded := runSearchLoadCommand(t, model.Init())
	model, _ = updateHub(model, loaded)
	return model
}

// runCaptureSave presses the save key, runs the save off the event loop, and
// applies the result — the sequence Bubble Tea performs at runtime.
func runCaptureSave(t *testing.T, model HubModel, key tea.KeyMsg) (HubModel, tea.Cmd) {
	t.Helper()
	model, command := updateHub(model, key)
	if command == nil {
		t.Fatal("save was not scheduled")
	}
	message, ok := command().(captureSavedMsg)
	if !ok {
		t.Fatalf("save command produced %T", message)
	}
	return updateHub(model, message)
}

func runReloadCommand(t *testing.T, command tea.Cmd) NotesReloadedMsg {
	t.Helper()
	if command == nil {
		t.Fatal("reload was not scheduled")
	}
	message, ok := command().(NotesReloadedMsg)
	if !ok {
		t.Fatalf("reload command produced %T", message)
	}
	return message
}

func runSearchLoadCommand(t *testing.T, command tea.Cmd) searchLoadedMsg {
	t.Helper()
	return runFor[searchLoadedMsg](t, command)
}

// runFor runs a command tree and returns the first message of type T it
// produces. Bubble Tea batches commands and batches nest, so the tree is walked
// recursively; the branches run concurrently because a batch routinely holds a
// tea.Tick that will not fire for seconds.
func runFor[T tea.Msg](t *testing.T, command tea.Cmd) T {
	t.Helper()
	var zero T
	if command == nil {
		t.Fatalf("no command was scheduled to produce %T", zero)
	}

	results := make(chan tea.Msg, 32)
	var run func(tea.Cmd)
	run = func(cmd tea.Cmd) {
		if cmd == nil {
			return
		}
		go func() {
			switch message := cmd().(type) {
			case tea.BatchMsg:
				for _, child := range message {
					run(child)
				}
			default:
				results <- message
			}
		}()
	}
	run(command)

	timeout := time.NewTimer(time.Second)
	defer timeout.Stop()
	for {
		select {
		case result := <-results:
			if found, ok := result.(T); ok {
				return found
			}
		case <-timeout.C:
			t.Fatalf("command tree never produced a %T", zero)
			return zero
		}
	}
}

// A capture appends to the existing monthly inbox, so no new note path appears in
// the reloaded list. Ctrl+E must still open the file the capture landed in.
func TestHubSaveAndEditCaptureOpensTheInboxItAppendedTo(t *testing.T) {
	inbox := Note{Path: "/notes/inbox/2026-07.md", Title: "Inbox", Type: NoteProjectInbox}
	edited := ""
	model := NewHub([]Note{inbox}, "api", "main", "central").
		WithActions(func(string, bool) (string, []Note, error) { return inbox.Path, []Note{inbox}, nil }, nil).
		WithAuthoring(nil, func(note Note) tea.Cmd {
			return func() tea.Msg {
				edited = note.Path
				return nil
			}
		})

	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("remember this")})
	_, command := runCaptureSave(t, model, tea.KeyMsg{Type: tea.KeyCtrlE})
	if command == nil {
		t.Fatal("Ctrl+E did not open an editor")
	}
	command()
	if edited != inbox.Path {
		t.Fatalf("Ctrl+E edited %q, want the inbox %q", edited, inbox.Path)
	}
}

// Every keystroke funnels through refreshPreview, so re-running Glamour on an
// unchanged note is pure waste. The rendered Markdown is cached; poisoning the
// cache proves the cached copy is what reaches the viewport.
func TestRefreshPreviewReusesTheRenderedMarkdown(t *testing.T) {
	notes := []Note{{Path: "/notes/a.md", Title: "A", Type: NoteNow, Content: "# Heading\n\nBody."}}
	model := NewHub(notes, "api", "main", "central").WithStyle("notty")
	model.refreshPreview()

	model.previewCache = "CACHED-MARKER"
	model.refreshPreview()

	if !strings.Contains(model.preview.View(), "CACHED-MARKER") {
		t.Fatalf("refreshPreview re-rendered an unchanged note: %q", model.preview.View())
	}
}

func TestRefreshPreviewRerendersWhenTheWidthChanges(t *testing.T) {
	notes := []Note{{Path: "/notes/a.md", Title: "A", Type: NoteNow, Content: "# Heading\n\nBody."}}
	model := NewHub(notes, "api", "main", "central").WithStyle("notty")
	model.refreshPreview()

	model.previewCache = "CACHED-MARKER"
	model.resize(140, 40)
	model.refreshPreview()

	if strings.Contains(model.preview.View(), "CACHED-MARKER") {
		t.Fatal("refreshPreview served a cached render at the old width")
	}
}

func TestRefreshPreviewRerendersWhenTheNoteChangesOnDisk(t *testing.T) {
	written := time.Now()
	notes := []Note{{Path: "/notes/a.md", Title: "A", Type: NoteNow, Content: "# Heading", Modified: written}}
	model := NewHub(notes, "api", "main", "central").WithStyle("notty")
	model.refreshPreview()

	model.previewCache = "CACHED-MARKER"
	model.notes = []Note{{Path: "/notes/a.md", Title: "A", Type: NoteNow, Content: "# Rewritten", Modified: written.Add(time.Second)}}
	model.refreshPreview()

	if strings.Contains(model.preview.View(), "CACHED-MARKER") {
		t.Fatal("refreshPreview served a stale render after the note changed")
	}
}

// WithAutoStyle probes the terminal background. NewHub must not trigger that
// probe: the caller pins the real style with WithStyle a moment later, which
// throws any renderer built here away.
func TestNewHubDoesNotBuildARendererBeforeTheStyleIsPinned(t *testing.T) {
	model := NewHub([]Note{{Path: "/notes/a.md", Title: "A", Type: NoteNow, Content: "# Heading"}}, "api", "main", "central")
	if model.previewRenderer != nil {
		t.Fatal("NewHub built a Glamour renderer before the style was pinned")
	}
}

// In the preview panel j/k belong to the viewport. Moving the selection there
// scrolls the old note and swaps in a different one on the same keypress.
func TestHubPreviewPanelScrollsWithoutChangingTheSelection(t *testing.T) {
	notes := []Note{
		{Path: "/a.md", Title: "First", Type: NoteNow, Content: strings.Repeat("line\n", 100)},
		{Path: "/b.md", Title: "Second", Type: NoteNow, Content: "# Second"},
	}
	model := NewHub(notes, "api", "main", "central").WithStyle("notty")
	model.panel = panelPreview
	model.refreshPreview()

	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if model.noteIndex != 0 {
		t.Fatalf("j in the preview moved the selection to note %d", model.noteIndex)
	}
	if model.preview.YOffset == 0 {
		t.Fatal("j in the preview did not scroll the viewport")
	}

	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if model.noteIndex != 0 {
		t.Fatalf("G in the preview moved the selection to note %d", model.noteIndex)
	}
}

// Reloading rescans the whole store. Doing it inside Update freezes the TUI for
// the duration: no repaint, no input, no way to tell the user anything.
func TestHubReloadRunsOffTheEventLoop(t *testing.T) {
	reloaded := false
	notes := []Note{{Path: "/a.md", Title: "Cache", Type: NoteProjectNote, Content: "# Cache"}}
	model := NewHub(nil, "api", "main", "central").
		WithActions(nil, func() ([]Note, error) {
			reloaded = true
			return notes, nil
		})

	_, command := updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if reloaded {
		t.Fatal("r reloaded notes inside Update")
	}
	if command == nil {
		t.Fatal("r produced no reload command")
	}

	message, ok := command().(NotesReloadedMsg)
	if !ok || !reloaded || len(message.Notes) != 1 || !message.RefreshSearch {
		t.Fatalf("reload command produced %#v (reloaded=%v)", message, reloaded)
	}
}

// Saving takes the storage lock (bounded at two seconds) and then rereads the
// store. Running that inside Update freezes the modal with no sign of progress.
func TestHubCaptureSaveRunsOffTheEventLoop(t *testing.T) {
	captured := false
	note := Note{Path: "/inbox/2026-07.md", Title: "Inbox", Type: NoteProjectInbox}
	model := NewHub(nil, "api", "main", "central").
		WithActions(func(string, bool) (string, []Note, error) {
			captured = true
			return note.Path, []Note{note}, nil
		}, nil)

	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("remember this")})
	model, command := updateHub(model, tea.KeyMsg{Type: tea.KeyCtrlS})
	if captured {
		t.Fatal("the capture was written inside Update")
	}
	if command == nil {
		t.Fatal("Ctrl+S produced no save command")
	}
	if !model.capturing || !strings.Contains(model.View(), "Saving") {
		t.Fatalf("the modal did not report that a save is in flight:\n%s", model.View())
	}

	model, _ = updateHub(model, command())
	if !captured || model.capturing || len(model.notes) != 1 {
		t.Fatalf("save message left capturing=%v notes=%#v (captured=%v)", model.capturing, model.notes, captured)
	}
}

// The capture-only TUI hands the model straight to Bubble Tea, so whatever
// BeginCapture returns is dropped. Init has to ask for the cursor itself.
func TestInitBlinksTheCursorWhenTheModalIsAlreadyOpen(t *testing.T) {
	model, _ := NewHub(nil, "api", "main", "central").BeginCapture(false, true)
	if model.Init() == nil {
		t.Fatal("Init did not start the capture cursor")
	}
	if NewHub(nil, "api", "main", "central").Init() != nil {
		t.Fatal("Init scheduled work for a Hub with nothing to do")
	}
}

// The digest opens before its store scan finishes, so both the loading state
// and a failed scan need a view of their own.
func TestDigestViewShowsLoadingThenError(t *testing.T) {
	model := NewHub(nil, "api", "main", "central").WithDigest(func(int) (digest.DigestReport, error) {
		return digest.DigestReport{}, errors.New("store is unreadable")
	})
	model, _ = updateHub(model, tea.WindowSizeMsg{Width: 120, Height: 30})

	model, command := updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if !strings.Contains(model.View(), "Loading") {
		t.Fatalf("digest view did not show a loading state:\n%s", model.View())
	}

	model, _ = updateHub(model, runFor[digestLoadedMsg](t, command))
	if !strings.Contains(model.View(), "store is unreadable") {
		t.Fatalf("digest view did not surface the error:\n%s", model.View())
	}

	// A failed digest must not leave a stale report on the clipboard path.
	if _, command = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}); command != nil {
		t.Error("y copied while the digest was in an error state")
	}
}

func TestDigestViewLoadsRendersAndTogglesRange(t *testing.T) {
	var requested []int
	model := NewHub(nil, "api", "main", "central").WithDigest(func(days int) (digest.DigestReport, error) {
		requested = append(requested, days)
		return digest.DigestReport{
			Generated: time.Now(),
			Days:      days,
			Streak:    3,
			Items: []digest.ActivityItem{
				{Kind: "task", Summary: "Rotate the signing tokens", Time: time.Now()},
				{Kind: "decision", Summary: "Use opaque refresh tokens", Time: time.Now()},
			},
			CurrentTask: "Implement token rotation",
			Heatmap:     []digest.DayActivity{{Date: time.Now().Format("2006-01-02"), Count: 2}},
		}, nil
	})
	model, _ = updateHub(model, tea.WindowSizeMsg{Width: 120, Height: 30})

	model, command := updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if !model.digest || command == nil {
		t.Fatalf("s did not open the digest view (digest=%v, command=%v)", model.digest, command != nil)
	}
	model, _ = updateHub(model, runFor[digestLoadedMsg](t, command))

	view := model.View()
	for _, want := range []string{"Activity Heatmap", "Rotate the signing tokens", "Use opaque refresh tokens", "Implement token rotation", "3 day streak", "Copy Markdown"} {
		if !strings.Contains(view, want) {
			t.Fatalf("digest view missing %q:\n%s", want, view)
		}
	}
	// The hint names where t goes, not where we already are.
	if !strings.Contains(view, "[t] This week") {
		t.Fatalf("today's digest should offer the week in its hint:\n%s", view)
	}

	model, command = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	if command == nil {
		t.Fatal("t did not reload the digest")
	}
	model, _ = updateHub(model, runFor[digestLoadedMsg](t, command))
	view = model.View()
	if !strings.Contains(view, "This Week") {
		t.Fatalf("t did not switch the range:\n%s", view)
	}
	if !strings.Contains(view, "[t] Today") {
		t.Fatalf("the week's digest should offer today in its hint:\n%s", view)
	}

	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.digest {
		t.Fatal("Esc did not return to the Hub")
	}
	if len(requested) != 2 || requested[0] != 1 || requested[1] != 7 {
		t.Fatalf("digest requested days = %v, want [1 7]", requested)
	}
}

func TestFlashMessageColorDecay(t *testing.T) {
	model := NewHub(nil, "api", "main", "central")
	command := model.flash("✓ Saved to inbox")
	if command == nil || model.flashStep != 0 {
		t.Fatalf("flash = step %d, cmd %v", model.flashStep, command)
	}

	seen := []lipgloss.Color{flashColorFor(model.flashStep)}
	for step := range 2 {
		var cmd tea.Cmd
		model, cmd = updateHub(model, flashDecayMsg{tick: model.flashTick, step: step})
		if model.flashMsg == "" || model.flashStep != step+1 || cmd == nil {
			t.Fatalf("decay step %d = msg %q, step %d, cmd %v", step, model.flashMsg, model.flashStep, cmd)
		}
		seen = append(seen, flashColorFor(model.flashStep))
	}

	// Each step must render in its own colour, otherwise the decay is invisible.
	if seen[0] == seen[1] || seen[1] == seen[2] || seen[0] == seen[2] {
		t.Fatalf("decay steps reuse colours: %v", seen)
	}

	model, _ = updateHub(model, flashDecayMsg{tick: model.flashTick, step: 2})
	if model.flashMsg != "" || model.flashStep != 0 {
		t.Fatalf("flash outlived its last step: %q step %d", model.flashMsg, model.flashStep)
	}
}

func TestFlashResetsDecayStepOfThePreviousMessage(t *testing.T) {
	model := NewHub(nil, "api", "main", "central")
	model.flash("✓ Saved to inbox")
	model, _ = updateHub(model, flashDecayMsg{tick: model.flashTick, step: 0})
	if model.flashStep != 1 {
		t.Fatalf("setup: expected a faded flash, got step %d", model.flashStep)
	}

	model.flash("✓ Standup copied to clipboard")
	if model.flashStep != 0 {
		t.Fatalf("a new flash inherited the old decay step %d", model.flashStep)
	}
}

func TestHeatmapStaggerRevealsEveryColumnThenStops(t *testing.T) {
	model := NewHub(nil, "api", "main", "central")
	model.digest = true
	model.digestStagger = 1

	for step := 1; step < heatmapWeeks; step++ {
		var cmd tea.Cmd
		model, cmd = updateHub(model, digestStaggerMsg{step: step})
		if model.digestStagger != step+1 || cmd == nil {
			t.Fatalf("stagger step %d = %d, cmd %v", step, model.digestStagger, cmd)
		}
	}

	final, cmd := updateHub(model, digestStaggerMsg{step: heatmapWeeks})
	if final.digestStagger != heatmapWeeks || cmd != nil {
		t.Fatalf("reveal kept ticking past the last column: %d, cmd %v", final.digestStagger, cmd)
	}

	// A tick left over from a digest that was closed and reopened must not
	// rewind the reveal of the current one.
	stale, _ := updateHub(final, digestStaggerMsg{step: 1})
	if stale.digestStagger != heatmapWeeks {
		t.Fatalf("a stale stagger tick rewound the reveal to %d", stale.digestStagger)
	}
}

func TestSpinnerAnimatesWhileBusyAndStopsWhenIdle(t *testing.T) {
	model := NewHub(nil, "api", "main", "central").
		WithSearch(nil, func() ([]searchindex.Entry, error) { return nil, nil })

	// Something has to emit the first tick, or the spinner is frozen on frame 0.
	tick := findSpinnerTick(model.Init())
	if tick == nil {
		t.Fatal("Init never starts the spinner")
	}

	frame := model.spinner.View()
	model, cmd := updateHub(model, tick())
	if model.spinner.View() == frame {
		t.Fatalf("spinner did not advance a frame while busy: still %q", frame)
	}
	if cmd == nil {
		t.Fatal("spinner stopped ticking while search was still refreshing")
	}

	// Once nothing is loading the loop must end instead of waking the Hub forever.
	model.searchRefreshing = false
	idle, cmd := updateHub(model, tick())
	if cmd != nil {
		t.Fatalf("spinner kept ticking after loading finished: %v", cmd)
	}
	if strings.Contains(idle.View(), "indexing") {
		t.Fatalf("idle status bar still advertises indexing:\n%s", idle.View())
	}
}

func TestSpinnerRendersInStatusBarAndDigest(t *testing.T) {
	model := NewHub(nil, "api", "main", "central")
	model.searchRefreshing = true
	view := model.View()
	if !strings.Contains(view, model.spinner.View()+" indexing") {
		t.Fatalf("expected the spinner beside the indexing hint:\n%s", view)
	}

	model.searchRefreshing = false
	model.digest = true
	model.digestLoading = true
	digestView := model.View()
	if !strings.Contains(digestView, model.spinner.View()+" Loading…") {
		t.Fatalf("expected the spinner beside the digest loading hint:\n%s", digestView)
	}
}

// findSpinnerTick unwraps the command tree Init returns and hands back the
// spinner's own tick command, or nil when nothing starts the spinner.
func findSpinnerTick(command tea.Cmd) tea.Cmd {
	if command == nil {
		return nil
	}
	switch msg := command().(type) {
	case tea.BatchMsg:
		for _, child := range msg {
			if found := findSpinnerTick(child); found != nil {
				return found
			}
		}
	case spinner.TickMsg:
		return command
	}
	return nil
}

func TestSpinnerFollowsTheConfiguredTheme(t *testing.T) {
	model := NewHub(nil, "api", "main", "central").WithUIConfig(0, false, "dracula")
	model.searchRefreshing = true
	if !strings.Contains(model.View(), model.spinner.View()) {
		t.Fatalf("spinner is missing from the status bar:\n%s", model.View())
	}
	if model.spinner.Style.GetForeground() != getThemePalette("dracula").headerFg {
		t.Errorf("spinner keeps the default colour under a custom theme: %v", model.spinner.Style.GetForeground())
	}
}
