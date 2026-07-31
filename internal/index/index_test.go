package index

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
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

// Saving one note used to re-read every note in every registered project. The
// rescan is keyed on size and modification time, so an unchanged file is carried
// over from the previous cache instead of being read, hashed and parsed again.
func TestRefreshReusesUnchangedEntriesWithoutReadingThem(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "notes", "cache.md")
	const original = "# Cache policy\noriginal body"
	const rewritten = "# Cache policy\nREWRITTEN!!!!" // same length, so only size+mtime match
	mustWrite(t, path, original)

	stores := []Store{{ProjectID: "p1", ProjectName: "api", Root: root}}
	first, err := Scan(stores, 256*1024)
	if err != nil {
		t.Fatal(err)
	}

	// Rewrite the body in place, keeping size and mtime identical. Only a reader
	// that actually opened the file could notice — which is exactly what Refresh
	// must not do for an unchanged entry.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(rewritten) != len(original) {
		t.Fatalf("test setup: rewrite changes the size (%d vs %d)", len(rewritten), len(original))
	}
	if err := os.WriteFile(path, []byte(rewritten), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}

	refreshed, err := Refresh(stores, 256*1024, first)
	if err != nil {
		t.Fatal(err)
	}
	if len(refreshed) != 1 {
		t.Fatalf("Refresh() returned %d entries", len(refreshed))
	}
	if !strings.Contains(refreshed[0].Content, "original body") {
		t.Fatalf("Refresh() re-read an unchanged file: %q", refreshed[0].Content)
	}
}

func TestRefreshRereadsChangedAndDropsDeletedNotes(t *testing.T) {
	root := t.TempDir()
	kept := filepath.Join(root, "notes", "kept.md")
	changed := filepath.Join(root, "notes", "changed.md")
	removed := filepath.Join(root, "notes", "removed.md")
	mustWrite(t, kept, "# Kept\nbody")
	mustWrite(t, changed, "# Changed\nold")
	mustWrite(t, removed, "# Removed\nbody")

	stores := []Store{{ProjectID: "p1", ProjectName: "api", Root: root}}
	first, err := Scan(stores, 256*1024)
	if err != nil {
		t.Fatal(err)
	}

	mustWrite(t, changed, "# Changed\nnew body that is a different length")
	if err := os.Remove(removed); err != nil {
		t.Fatal(err)
	}

	refreshed, err := Refresh(stores, 256*1024, first)
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string]Entry{}
	for _, entry := range refreshed {
		byPath[filepath.Base(entry.Path)] = entry
	}
	if len(byPath) != 2 {
		t.Fatalf("Refresh() returned %d entries: %#v", len(byPath), byPath)
	}
	if _, gone := byPath["removed.md"]; gone {
		t.Fatal("Refresh() kept a deleted note")
	}
	if !strings.Contains(byPath["changed.md"].Content, "new body") {
		t.Fatalf("Refresh() served a stale body: %q", byPath["changed.md"].Content)
	}
	if byPath["changed.md"].Fingerprint == "" {
		t.Fatal("Refresh() left the changed entry without a fingerprint")
	}
}

// Refresh with no previous cache must behave exactly like a cold Scan.
func TestRefreshWithoutPreviousEntriesMatchesScan(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "notes", "cache.md"), "# Cache policy\nbody")
	stores := []Store{{ProjectID: "p1", ProjectName: "api", Root: root}}

	scanned, err := Scan(stores, 256*1024)
	if err != nil {
		t.Fatal(err)
	}
	refreshed, err := Refresh(stores, 256*1024, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(scanned) != len(refreshed) || scanned[0].Fingerprint != refreshed[0].Fingerprint {
		t.Fatalf("Refresh() = %#v, Scan() = %#v", refreshed, scanned)
	}
}

func TestScanClassifiesNotesByStoreLayout(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "now.md"), "# Now")
	mustWrite(t, filepath.Join(root, "inbox", "2026-07.md"), "# Inbox")
	mustWrite(t, filepath.Join(root, "decisions", "2026-07-22-use-redis.md"), "# Decision: Use Redis")
	mustWrite(t, filepath.Join(root, "notes", "cache.md"), "# Cache policy")

	entries, err := Scan([]Store{{ProjectID: "p1", ProjectName: "api", Root: root}}, 256*1024)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, entry := range entries {
		got[filepath.Base(entry.Path)] = entry.NoteType
	}
	want := map[string]string{
		"now.md":                  "now",
		"2026-07.md":              "inbox",
		"2026-07-22-use-redis.md": "decision",
		"cache.md":                "note",
	}
	for name, noteType := range want {
		if got[name] != noteType {
			t.Fatalf("%s note type = %q, want %q", name, got[name], noteType)
		}
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

func TestScanRejectsANonPositiveLimitAndSkipsEmptyOrMissingStores(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "notes", "cache.md"), "# Cache policy")

	if _, err := Scan([]Store{{Root: root}}, 0); err == nil {
		t.Fatal("Scan accepted a non-positive byte limit")
	}
	entries, err := Scan([]Store{
		{ProjectID: "p1", Root: ""},
		{ProjectID: "p2", Root: filepath.Join(t.TempDir(), "never-created")},
		{ProjectID: "p3", Root: root},
	}, 256*1024)
	if err != nil || len(entries) != 1 || entries[0].ProjectID != "p3" {
		t.Fatalf("Scan across empty and missing stores = %#v, %v", entries, err)
	}
}

func TestSearchIgnoresEmptyQueriesAndHonoursTheLimit(t *testing.T) {
	entries := []Entry{
		{Path: "/a.md", Title: "Cache policy", Content: "bounded"},
		{Path: "/b.md", Title: "Cache plan", Content: "bounded"},
	}
	if results := Search(entries, "   ", 10); results != nil {
		t.Fatalf("empty query returned %#v", results)
	}
	if results := Search(entries, "cache", 0); results != nil {
		t.Fatalf("zero limit returned %#v", results)
	}
	if results := Search(entries, "cache", 1); len(results) != 1 {
		t.Fatalf("limit 1 returned %d results", len(results))
	}
}

// Equal scores must fall back to newest-first and then path, so repeated searches
// return the same order.
func TestSearchBreaksScoreTiesByModifiedTimeThenPath(t *testing.T) {
	older := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC)
	entries := []Entry{
		{Path: "/b.md", Title: "Runbook", Modified: older},
		{Path: "/c.md", Title: "Runbook", Modified: newer},
		{Path: "/a.md", Title: "Runbook", Modified: newer},
	}
	results := Search(entries, "runbook", 10)
	if len(results) != 3 {
		t.Fatalf("results = %#v", results)
	}
	want := []string{"/a.md", "/c.md", "/b.md"}
	for i, path := range want {
		if results[i].Entry.Path != path {
			t.Fatalf("order = %q, want %q", []string{results[0].Entry.Path, results[1].Entry.Path, results[2].Entry.Path}, want)
		}
	}
}

func TestSnippetTruncatesContentThatDoesNotContainTheQuery(t *testing.T) {
	long := strings.Repeat("word ", 100)
	results := Search([]Entry{{Path: "/a.md", Title: "Cache policy", Content: long}}, "cache", 10)
	if len(results) != 1 {
		t.Fatalf("results = %#v", results)
	}
	if !strings.HasSuffix(results[0].Snippet, "…") || len([]rune(results[0].Snippet)) != 161 {
		t.Fatalf("snippet = %q (%d runes)", results[0].Snippet, len([]rune(results[0].Snippet)))
	}
}

func TestLoadCacheTreatsAMissingFileAsEmpty(t *testing.T) {
	cache, err := LoadCache(filepath.Join(t.TempDir(), "index-v1.json"))
	if err != nil || cache.Version != CacheVersion || len(cache.Entries) != 0 {
		t.Fatalf("LoadCache on a missing file = %#v, %v", cache, err)
	}
	if _, err := LoadCache(t.TempDir()); err == nil {
		t.Fatal("LoadCache accepted a directory")
	}
}

func TestIndexFoldFindsTheQueryRegardlessOfCase(t *testing.T) {
	cases := []struct {
		name     string
		haystack string
		needle   string
		want     int
	}{
		{"exact", "cache invalidation", "cache", 0},
		{"uppercase haystack", "CACHE INVALIDATION", "cache", 0},
		{"mixed case in the middle", "replay DeTeCtIoN here", "detection", 7},
		{"at the very end", "rotate the Token", "token", 11},
		{"false start before the real match", "aaab", "aab", 1},
		{"absent", "cache invalidation", "redis", -1},
		{"needle longer than haystack", "no", "nope", -1},
		{"empty haystack", "", "token", -1},
		{"diacritics", "Přehled Změn v Modulu", "změn", 9},
		{"diacritics absent", "prehled zmen", "změn", -1},
		{"multi-byte first rune with an ASCII tail", "ŽÁDNÉ zprávy", "žádné", 0},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := indexFold(testCase.haystack, testCase.needle); got != testCase.want {
				t.Fatalf("indexFold(%q, %q) = %d, want %d", testCase.haystack, testCase.needle, got, testCase.want)
			}
		})
	}
}

// Snippets are sliced by byte offset, so a window edge can land inside a
// multi-byte rune and paint a replacement character into the search list.
func TestSnippetStaysValidUTF8(t *testing.T) {
	// The leading "abc " shifts the 160th byte into the middle of a two-byte rune.
	long := "abc " + strings.Repeat("žluťoučký kůň ", 40)
	truncated := Search([]Entry{{Path: "/a.md", Title: "Cache policy", Content: long}}, "cache", 10)
	if len(truncated) != 1 || !utf8.ValidString(truncated[0].Snippet) {
		t.Fatalf("truncated snippet = %q", truncated[0].Snippet)
	}
	if !strings.HasSuffix(truncated[0].Snippet, "…") || len([]rune(truncated[0].Snippet)) != 161 {
		t.Fatalf("truncated snippet = %q (%d runes)", truncated[0].Snippet, len([]rune(truncated[0].Snippet)))
	}

	windowed := Search([]Entry{{Path: "/b.md", Title: "Poznámky", Content: long + "cache " + long}}, "cache", 10)
	if len(windowed) != 1 || !utf8.ValidString(windowed[0].Snippet) {
		t.Fatalf("windowed snippet = %q", windowed[0].Snippet)
	}
	if !strings.Contains(strings.ToLower(windowed[0].Snippet), "cache") {
		t.Fatalf("windowed snippet lost the match: %q", windowed[0].Snippet)
	}
}
