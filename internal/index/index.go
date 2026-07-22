package index

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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
	Fingerprint string    `json:"content_fingerprint"`
}

type Cache struct {
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
	Entries   []Entry   `json:"entries"`
}

type Result struct {
	Entry   Entry
	Score   int
	Snippet string
}

func Scan(stores []Store, maxFileBytes int64) ([]Entry, error) {
	if maxFileBytes <= 0 {
		return nil, fmt.Errorf("index byte limit must be positive")
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
			hash := sha256.Sum256(data)
			entries = append(entries, Entry{
				Path: path, ProjectID: store.ProjectID, ProjectName: store.ProjectName,
				NoteType: noteType(filepath.ToSlash(relative)), Title: md.Title(content, dirEntry.Name()),
				Tags: md.Tags(content), Modified: info.ModTime(), Size: info.Size(), Content: content,
				Fingerprint: hex.EncodeToString(hash[:]),
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
			} else if strings.Contains(strings.ToLower(entry.Content), query) {
				score = 200000
			}
		}
		if score > 0 {
			results = append(results, Result{Entry: entry, Score: score, Snippet: snippet(entry.Content, query)})
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
	data, err := json.MarshalIndent(cache, "", "  ")
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

func containsTag(tags []string, query string) bool {
	for _, tag := range tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}
	return false
}

func snippet(content, query string) string {
	flat := strings.Join(strings.Fields(content), " ")
	lower := strings.ToLower(flat)
	position := strings.Index(lower, query)
	if position < 0 {
		if len(flat) > 160 {
			return flat[:160] + "…"
		}
		return flat
	}
	start := max(0, position-60)
	end := min(len(flat), position+len(query)+100)
	return flat[start:end]
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

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}
