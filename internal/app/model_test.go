package app

import (
	"errors"
	"strings"
	"testing"

	searchindex "github.com/Resetnak/herdr-logbook/internal/index"
	tea "github.com/charmbracelet/bubbletea"
)

func TestHubWideViewShowsThreePanesAndActionableEmptyState(t *testing.T) {
	model := NewHub(nil, "api", "feature/login", "central")
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	view := updated.(HubModel).View()
	for _, want := range []string{"Scopes", "Notes", "Preview", "No current context yet", "Press c"} {
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
	model := NewHub(nil, "api", "main", "central").WithActions(nil, func() ([]Note, error) {
		reloads++
		return []Note{{Title: "Fresh", Type: NoteNow, Content: "new"}}, nil
	})

	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if reloads != 1 || len(model.notes) != 1 || model.notes[0].Title != "Fresh" {
		t.Fatalf("refresh = reloads %d, notes %#v", reloads, model.notes)
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
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	model, _ = updateHub(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Runbook")})
	_, command := updateHub(model, tea.KeyMsg{Type: tea.KeyCtrlS})
	if authorKind != "note:Runbook" || command == nil {
		t.Fatalf("author action = %q, command nil %t", authorKind, command == nil)
	}
	command()
	if edited != "/notes/new.md" {
		t.Fatalf("edited path after authoring = %q", edited)
	}
}

func updateHub(model HubModel, message tea.Msg) (HubModel, tea.Cmd) {
	updated, command := model.Update(message)
	return updated.(HubModel), command
}
