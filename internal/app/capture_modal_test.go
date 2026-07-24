package app

import (
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var stripANSI = regexp.MustCompile("\x1b\\[[0-9;]*m")

// Regression: the capture modal box used to be two columns narrower than the
// textarea's rendered rows, so lipgloss re-wrapped them and reflow broke the
// line at hyphens — typing "fix - rotate keys" rendered as two lines.
func TestCaptureModalKeepsHyphenatedTextOnOneLine(t *testing.T) {
	// A real terminal renders with colors; without them the textarea does not
	// pad rows with styled spaces and the wrap bug cannot surface.
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI256)
	defer lipgloss.SetColorProfile(previous)

	m := NewHub(nil, "project", "branch", "central")
	begun, _ := m.BeginCapture(false, false)
	var model tea.Model = begun
	model, _ = model.Update(tea.WindowSizeMsg{Width: 129, Height: 33})
	for _, r := range "fix - rotate keys" {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		if r == ' ' {
			msg = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
		}
		model, _ = model.Update(msg)
	}
	hub := model.(HubModel)
	if value := hub.captureBox.Value(); strings.Contains(value, "\n") {
		t.Fatalf("captured value gained a newline: %q", value)
	}
	view := stripANSI.ReplaceAllString(hub.View(), "")
	if !strings.Contains(view, "fix - rotate keys") {
		t.Fatalf("modal view split the hyphenated line:\n%s", view)
	}
}
