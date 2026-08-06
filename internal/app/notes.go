package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	md "github.com/Resetnak/herdr-logbook/internal/markdown"
)

// NoteType classifies a note by where it lives in the store layout; the Hub's
// scope tabs filter on it.
type NoteType string

const (
	NoteNow            NoteType = "now"
	NoteProjectInbox   NoteType = "project-inbox"
	NoteProjectNote    NoteType = "project-note"
	NoteDecision       NoteType = "decision"
	NoteGlobalInbox    NoteType = "global-inbox"
	NoteGlobalNote     NoteType = "global-note"
	NoteGlobalDecision NoteType = "global-decision"
)

// Note is one Markdown file loaded for the Hub, body included (bounded by
// maxPreviewBytes).
type Note struct {
	Path     string
	Title    string
	Type     NoteType
	Content  string
	Modified time.Time
	Size     int64
}

// LoadNotes reads every note in the project and global stores, skipping
// symlinks and replacing oversized bodies with a sentinel message.
func LoadNotes(projectRoot, globalRoot string, maxPreviewBytes int64) ([]Note, error) {
	if maxPreviewBytes <= 0 {
		return nil, fmt.Errorf("preview byte limit must be positive")
	}
	var notes []Note
	for _, source := range []struct {
		root   string
		global bool
	}{{projectRoot, false}, {globalRoot, true}} {
		if source.root == "" {
			continue
		}
		err := filepath.WalkDir(source.root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				if errors.Is(walkErr, os.ErrNotExist) && path == source.root {
					return filepath.SkipDir
				}
				return walkErr
			}
			if path != source.root && entry.IsDir() && strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			if entry.Type()&os.ModeSymlink != 0 || entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			note := Note{Path: path, Modified: info.ModTime(), Size: info.Size()}
			if info.Size() > maxPreviewBytes {
				note.Content = "Note is too large to preview."
			} else {
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				note.Content = string(data)
			}
			note.Title = md.Title(note.Content, entry.Name())
			rel, err := filepath.Rel(source.root, path)
			if err != nil {
				return err
			}
			note.Type = classifyNote(filepath.ToSlash(rel), source.global)
			notes = append(notes, note)
			return nil
		})
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("load notes from %q: %w", source.root, err)
		}
	}
	sort.Slice(notes, func(i, j int) bool {
		left, right := noteOrder(notes[i].Type), noteOrder(notes[j].Type)
		if left != right {
			return left < right
		}
		return notes[i].Path < notes[j].Path
	})
	return notes, nil
}

func classifyNote(relative string, global bool) NoteType {
	directory := strings.Split(relative, "/")[0]
	if !global && relative == "now.md" {
		return NoteNow
	}
	if global {
		switch directory {
		case "inbox":
			return NoteGlobalInbox
		case "decisions":
			return NoteGlobalDecision
		default:
			return NoteGlobalNote
		}
	}
	switch directory {
	case "inbox":
		return NoteProjectInbox
	case "decisions":
		return NoteDecision
	default:
		return NoteProjectNote
	}
}

func noteOrder(noteType NoteType) int {
	order := map[NoteType]int{
		NoteNow: 0, NoteProjectInbox: 1, NoteProjectNote: 2, NoteDecision: 3,
		NoteGlobalInbox: 4, NoteGlobalNote: 5, NoteGlobalDecision: 6,
	}
	return order[noteType]
}
