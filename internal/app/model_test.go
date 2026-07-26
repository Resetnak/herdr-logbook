package app

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	searchindex "github.com/Resetnak/herdr-logbook/internal/index"
	tea "github.com/charmbracelet/bubbletea"
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

func TestHubPreviewRendererIsCachedByWidth(t *testing.T) {
	model := NewHub([]Note{
		{Title: "Now", Type: NoteNow, Content: "# Now\n\nCurrent"},
		{Title: "Later", Type: NoteNow, Content: "# Later\n\nNext"},
	}, "api", "main", "central")
	renderer := model.previewRenderer
	if renderer == nil {
		t.Fatal("preview renderer was not initialized")
	}

	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRight})
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
	if model.searchGeneration != 0 {
		t.Fatalf("initial search generation = %d, want 0", model.searchGeneration)
	}
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("missing")})
	if view := model.View(); !strings.Contains(view, "Refreshing search index...") {
		t.Fatalf("pending search did not show refresh state:\n%s", view)
	}

	loaded, ok := model.Init()().(searchLoadedMsg)
	if !ok {
		t.Fatal("Init did not load search entries")
	}
	model, _ = updateHub(model, loaded)
	if view := model.View(); !strings.Contains(view, "No matching notes") {
		t.Fatalf("completed empty search did not show no-match state:\n%s", view)
	}
}

func TestHubCaptureModalSavesProjectNote(t *testing.T) {
	var capturedText string
	var capturedGlobal bool
	model := NewHub(nil, "api", "main", "central").WithActions(
		func(text string, global bool) ([]Note, error) {
			capturedText = text
			capturedGlobal = global
			return []Note{{Title: "Captured", Type: NoteProjectInbox, Content: text}}, nil
		},
		nil,
	)

	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if !model.capturing || model.captureGlobal {
		t.Fatalf("capture state = capturing %t, global %t", model.capturing, model.captureGlobal)
	}
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("remember this")})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyCtrlS})

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
		WithActions(func(text string, global bool) ([]Note, error) {
			return []Note{{Title: "Inbox", Type: NoteProjectInbox, Content: text}}, nil
		}, nil).
		WithSearch(nil, func() ([]searchindex.Entry, error) {
			searchLoads++
			return []searchindex.Entry{{Title: "Inbox", ProjectName: "api", Content: "first ever note"}}, nil
		})
	model = completeInitialSearch(t, model)

	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("first ever note")})
	model, command := updateHub(model, tea.KeyMsg{Type: tea.KeyCtrlS})
	loaded := runSearchLoadCommand(t, command)
	model, _ = updateHub(model, loaded)
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("first ever")})

	if searchLoads != 2 || len(model.searchResults) != 1 {
		t.Fatalf("search refresh = loads %d, results %#v", searchLoads, model.searchResults)
	}
}

func TestHubCaptureModalSupportsGlobalAndCancel(t *testing.T) {
	called := false
	model := NewHub(nil, "api", "main", "central").WithActions(
		func(string, bool) ([]Note, error) {
			called = true
			return nil, nil
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
		func(string, bool) ([]Note, error) { return nil, errors.New("disk full") },
		nil,
	)
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("note")})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyCtrlS})

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
	runSearchLoadCommand(t, command)
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
	runSearchLoadCommand(t, command)
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
	view := model.View()
	if !strings.Contains(view, "Runbook · payments") || strings.Contains(view, "Runbook · api") {
		t.Fatalf("project search view was not filtered:\n%s", view)
	}
}

func TestHubAuthoringAndEditorActions(t *testing.T) {
	authorKind := ""
	edited := ""
	model := NewHub([]Note{{Path: "/notes/a.md", Title: "A", Type: NoteNow}}, "api", "main", "central").
		WithAuthoring(func(kind, title string) ([]Note, error) {
			authorKind = kind + ":" + title
			return []Note{{Path: "/notes/new.md", Title: title, Type: NoteProjectNote}}, nil
		}, func(note Note) tea.Cmd {
			return func() tea.Msg {
				edited = note.Path
				return nil
			}
		})

	// 1. Test Ctrl+S (saves note, does NOT open editor)
	m1, _ := updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m1, _ = updateHub(m1, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Runbook")})
	m1, cmd1 := updateHub(m1, tea.KeyMsg{Type: tea.KeyCtrlS})
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
	_, cmd2 := updateHub(m2, tea.KeyMsg{Type: tea.KeyCtrlE})
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
	model, firstCommand := updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model, secondCommand := updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if firstCommand != nil || secondCommand != nil || !model.searchDirty {
		t.Fatalf("overlapping refreshes were not coalesced: first %v, second %v, dirty %t", firstCommand != nil, secondCommand != nil, model.searchDirty)
	}

	initialResult := runSearchLoadCommand(t, initialCommand)
	model, followUpCommand := updateHub(model, initialResult)
	followUpResult := runSearchLoadCommand(t, followUpCommand)
	model, _ = updateHub(model, followUpResult)

	if searchLoads != 2 || model.searchRefreshing || model.searchDirty || len(model.searchEntries) != 1 || model.searchEntries[0].Title != "refresh-2" {
		t.Fatalf("coalesced refresh = loads %d, refreshing %t, dirty %t, entries %#v", searchLoads, model.searchRefreshing, model.searchDirty, model.searchEntries)
	}
}

func TestHubIgnoresSupersededSearchRefresh(t *testing.T) {
	searchLoads := 0
	model := NewHub(nil, "api", "main", "central").
		WithActions(nil, func() ([]Note, error) { return nil, nil }).
		WithSearch(nil, func() ([]searchindex.Entry, error) {
			searchLoads++
			return []searchindex.Entry{{Title: fmt.Sprintf("refresh-%d", searchLoads)}}, nil
		})
	model = completeInitialSearch(t, model)

	model, currentCommand := updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	currentResult := runSearchLoadCommand(t, currentCommand)
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model, nextCommand := updateHub(model, currentResult)

	currentResult.entries = []searchindex.Entry{{Title: "stale"}}
	model, _ = updateHub(model, currentResult)
	if len(model.searchEntries) != 1 || model.searchEntries[0].Title != "refresh-2" {
		t.Fatalf("search entries were replaced by a stale refresh: %#v", model.searchEntries)
	}

	nextResult := runSearchLoadCommand(t, nextCommand)
	model, _ = updateHub(model, nextResult)
	if len(model.searchEntries) != 1 || model.searchEntries[0].Title != "refresh-3" {
		t.Fatalf("latest search refresh was not applied: %#v", model.searchEntries)
	}
}

func updateHub(model HubModel, message tea.Msg) (HubModel, tea.Cmd) {
	updated, command := model.Update(message)
	return updated.(HubModel), command
}

func completeInitialSearch(t *testing.T, model HubModel) HubModel {
	t.Helper()
	loaded := runSearchLoadCommand(t, model.Init())
	model, _ = updateHub(model, loaded)
	return model
}

func runSearchLoadCommand(t *testing.T, command tea.Cmd) searchLoadedMsg {
	t.Helper()
	if command == nil {
		t.Fatal("search-index refresh was not scheduled")
	}
	switch message := command().(type) {
	case searchLoadedMsg:
		return message
	case tea.BatchMsg:
		results := make(chan tea.Msg, len(message))
		for _, batched := range message {
			go func(command tea.Cmd) {
				results <- command()
			}(batched)
		}
		timeout := time.NewTimer(time.Second)
		defer timeout.Stop()
		for range len(message) {
			select {
			case result := <-results:
				if loaded, ok := result.(searchLoadedMsg); ok {
					return loaded
				}
			case <-timeout.C:
				t.Fatal("timed out waiting for search entries")
			}
		}
	}
	t.Fatal("command did not load search entries")
	return searchLoadedMsg{}
}
