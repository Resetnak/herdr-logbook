package index

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScanFiltersFilesAndBuildsMetadata(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "notes", "cache.md"), "---\ntags: [Go, perf]\n---\n# Cache policy\nUse bounded entries.")
	mustWrite(t, filepath.Join(root, "notes", "skip.txt"), "skip")
	mustWrite(t, filepath.Join(root, ".hidden", "secret.md"), "# Secret")
	if err := os.Symlink(filepath.Join(root, "notes", "cache.md"), filepath.Join(root, "alias.md")); err != nil {
		t.Fatal(err)
	}

	entries, err := Scan([]Store{{ProjectID: "p1", ProjectName: "api", Root: root}}, 256*1024)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("Scan() returned %d entries: %#v", len(entries), entries)
	}
	entry := entries[0]
	if entry.ProjectID != "p1" || entry.Title != "Cache policy" || len(entry.Tags) != 2 || entry.Fingerprint == "" {
		t.Fatalf("entry = %#v", entry)
	}
}

func TestSearchRankingIsDeterministic(t *testing.T) {
	now := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	entries := []Entry{
		{Path: "/z.md", Title: "Token cache", Content: "other", Modified: now.Add(-time.Hour)},
		{Path: "/a.md", Title: "cache", Content: "other", Modified: now.Add(-24 * time.Hour)},
		{Path: "/b.md", Title: "Cache invalidation", Content: "other", Modified: now},
		{Path: "/c.md", Title: "Architecture", Tags: []string{"cache"}, Modified: now},
		{Path: "/d.md", Title: "Operations", Content: "Flush the cache safely", Modified: now},
	}

	results := Search(entries, "cache", 10)
	want := []string{"/a.md", "/b.md", "/z.md", "/c.md", "/d.md"}
	if len(results) != len(want) {
		t.Fatalf("Search() = %#v", results)
	}
	for i := range want {
		if results[i].Entry.Path != want[i] {
			t.Fatalf("result %d path = %q, want %q", i, results[i].Entry.Path, want[i])
		}
	}
}

func TestCacheRoundTripAndCorruptionRecovery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache", "index-v1.json")
	want := Cache{Version: 1, Entries: []Entry{{Path: "/a.md", Title: "A"}}}
	if err := SaveCache(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := LoadCache(path)
	if err != nil || len(got.Entries) != 1 || got.Entries[0].Title != "A" {
		t.Fatalf("LoadCache() = %#v, %v", got, err)
	}
	if err := os.WriteFile(path, []byte("not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err = LoadCache(path)
	if err != nil || got.Version != 1 || len(got.Entries) != 0 {
		t.Fatalf("corrupt LoadCache() = %#v, %v", got, err)
	}
}

func BenchmarkSearch10000(b *testing.B) {
	entries := make([]Entry, 10000)
	for i := range entries {
		entries[i] = Entry{Path: filepath.Join("notes", time.Unix(int64(i), 0).Format("150405")+".md"), Title: "Token rotation", Content: "cache invalidation replay detection"}
	}
	b.ResetTimer()
	for range b.N {
		Search(entries, "rotation", 50)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
