package index

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	md "github.com/Resetnak/herdr-logbook/internal/markdown"
	"github.com/Resetnak/herdr-logbook/internal/storage"
	"github.com/sahilm/fuzzy"
)

const CacheVersion = 1

type Store struct {
	ProjectID   string
	ProjectName string
	Root        string
}

type Entry struct {
	Path        string    `json:"path"`
	ProjectID   string    `json:"project_id"`
	ProjectName string    `json:"project_name"`
	NoteType    string    `json:"note_type"`
	Title       string    `json:"title"`
	Tags        []string  `json:"tags,omitempty"`
	Modified    time.Time `json:"modified"`
	Size        int64     `json:"size"`
	Content     string    `json:"content"`
}

type Cache struct {
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
	Entries   []Entry   `json:"entries"`
}

type Result struct {
	Entry Entry
	Score int
}

// Refresh rebuilds the index, carrying over any entry from previous whose file
// still has the same size and modification time. Saving a single note otherwise
// costs a full re-read of every note in every registered project.
//
// ponytail: size plus mtime, not a content hash — hashing would mean reading the
// file, which is the cost being avoided. A write that preserves both is missed;
// `index rebuild` exists for that, and Scan still forces a cold read.
func Refresh(stores []Store, maxFileBytes int64, previous []Entry) ([]Entry, error) {
	if maxFileBytes <= 0 {
		return nil, fmt.Errorf("index byte limit must be positive")
	}
	known := make(map[string]Entry, len(previous))
	for _, entry := range previous {
		known[entry.Path] = entry
	}
	var entries []Entry
	for _, store := range stores {
		if store.Root == "" {
			continue
		}
		err := filepath.WalkDir(store.Root, func(path string, dirEntry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				if errors.Is(walkErr, os.ErrNotExist) && path == store.Root {
					return filepath.SkipDir
				}
				return walkErr
			}
			if path != store.Root && dirEntry.IsDir() && strings.HasPrefix(dirEntry.Name(), ".") {
				return filepath.SkipDir
			}
			if dirEntry.Type()&os.ModeSymlink != 0 || dirEntry.IsDir() || !strings.EqualFold(filepath.Ext(dirEntry.Name()), ".md") {
				return nil
			}
			info, err := dirEntry.Info()
			if err != nil {
				return err
			}
			// The project a store belongs to can be renamed in the registry, so
			// the carried-over entry has to pick up the current store identity.
			if cached, ok := known[path]; ok && cached.Size == info.Size() && cached.Modified.Equal(info.ModTime()) {
				cached.ProjectID, cached.ProjectName = store.ProjectID, store.ProjectName
				entries = append(entries, cached)
				return nil
			}
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			data, readErr := io.ReadAll(io.LimitReader(file, maxFileBytes))
			closeErr := file.Close()
			if readErr != nil {
				return readErr
			}
			if closeErr != nil {
				return closeErr
			}
			content := string(data)
			relative, err := filepath.Rel(store.Root, path)
			if err != nil {
				return err
			}
			entries = append(entries, Entry{
				Path: path, ProjectID: store.ProjectID, ProjectName: store.ProjectName,
				NoteType: noteType(filepath.ToSlash(relative)), Title: md.Title(content, dirEntry.Name()),
				Tags: md.Tags(content), Modified: info.ModTime(), Size: info.Size(), Content: content,
			})
			return nil
		})
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("scan store %q: %w", store.Root, err)
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, nil
}

func Search(entries []Entry, query string, limit int) []Result {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" || limit <= 0 {
		return nil
	}
	results := make([]Result, 0)
	for _, entry := range entries {
		title := strings.ToLower(entry.Title)
		path := strings.ToLower(filepath.ToSlash(entry.Path))
		score := 0
		switch {
		case title == query:
			score = 600000
		case strings.HasPrefix(title, query):
			score = 500000
		default:
			if fuzzyScore := bestFuzzyScore(query, title, path); fuzzyScore != 0 {
				score = 400000 + fuzzyScore
			} else if containsTag(entry.Tags, query) {
				score = 300000
			} else if indexFold(entry.Content, query) >= 0 {
				score = 200000
			}
		}
		if score > 0 {
			results = append(results, Result{Entry: entry, Score: score})
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		if !results[i].Entry.Modified.Equal(results[j].Entry.Modified) {
			return results[i].Entry.Modified.After(results[j].Entry.Modified)
		}
		return results[i].Entry.Path < results[j].Entry.Path
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results
}

func LoadCache(path string) (Cache, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Cache{Version: CacheVersion}, nil
	}
	if err != nil {
		return Cache{}, fmt.Errorf("read index cache: %w", err)
	}
	var cache Cache
	if err := json.Unmarshal(data, &cache); err != nil || cache.Version != CacheVersion {
		return Cache{Version: CacheVersion}, nil
	}
	return cache, nil
}

func SaveCache(path string, cache Cache) error {
	cache.Version = CacheVersion
	cache.UpdatedAt = time.Now().UTC()
	// Not MarshalIndent: the cache is machine-read, disposable, and holds every
	// note's body, so the indentation is megabytes nobody reads.
	data, err := json.Marshal(cache)
	if err != nil {
		return fmt.Errorf("encode index cache: %w", err)
	}
	data = append(data, '\n')
	return storage.AtomicWrite(path, data, 0o600)
}

func bestFuzzyScore(query string, values ...string) int {
	best := 0
	for _, match := range fuzzy.FindNoSort(query, values) {
		score := 1000 + match.Score
		if score > best {
			best = score
		}
	}
	return best
}

// indexFold returns the byte offset of the already-lowercased needle in haystack,
// ignoring case, or -1. strings.Index(strings.ToLower(...)) would copy every note
// body on every keystroke; this scans in place for either case of the first rune
// and only compares a needle-sized window. It also keeps offsets usable against
// the original string, which lowercasing does not guarantee for every rune.
func indexFold(haystack, needle string) int {
	if needle == "" {
		return 0
	}
	first, _ := utf8.DecodeRuneInString(needle)
	targets := string([]rune{unicode.ToLower(first), unicode.ToUpper(first)})
	for offset := 0; offset+len(needle) <= len(haystack); {
		index := strings.IndexAny(haystack[offset:], targets)
		if index < 0 {
			return -1
		}
		start := offset + index
		if start+len(needle) > len(haystack) {
			return -1
		}
		if strings.EqualFold(haystack[start:start+len(needle)], needle) {
			return start
		}
		_, width := utf8.DecodeRuneInString(haystack[start:])
		offset = start + width
	}
	return -1
}

func containsTag(tags []string, query string) bool {
	for _, tag := range tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}
	return false
}

func noteType(relative string) string {
	if relative == "now.md" {
		return "now"
	}
	switch strings.Split(relative, "/")[0] {
	case "inbox":
		return "inbox"
	case "decisions":
		return "decision"
	default:
		return "note"
	}
}
