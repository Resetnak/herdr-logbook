package author

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestSlugHandlesUnicodeReservedNamesAndEmptyTitles(t *testing.T) {
	tests := []struct{ title, want string }{
		{"Token Rotation", "token-rotation"},
		{"Příliš žluťoučký", "příliš-žluťoučký"},
		{"CON", "note-con"},
	}
	for _, test := range tests {
		got, err := Slug(test.title)
		if err != nil || got != test.want {
			t.Fatalf("Slug(%q) = %q, %v; want %q", test.title, got, err, test.want)
		}
	}
	if _, err := Slug("../"); err == nil {
		t.Fatal("Slug traversal-only title succeeded")
	}
}

func TestCreateNoteWritesTitleAndDeduplicatesFilenames(t *testing.T) {
	root := t.TempDir()
	first, err := CreateNote(root, "  Token Rotation  ")
	if err != nil {
		t.Fatal(err)
	}
	second, err := CreateNote(root, "Token Rotation")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(first) != "token-rotation.md" || filepath.Base(second) != "token-rotation-2.md" {
		t.Fatalf("note paths = %q, %q", first, second)
	}
	if filepath.Dir(first) != filepath.Join(root, "notes") {
		t.Fatalf("note was written outside notes/: %q", first)
	}
	data, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "# Token Rotation\n" {
		t.Fatalf("note body = %q", data)
	}
	info, err := os.Stat(first)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("note mode = %v, want 0600", info.Mode().Perm())
	}
	if _, err := CreateNote(root, "../../escape"); err != nil {
		t.Fatalf("slug should strip traversal, got %v", err)
	}
	if _, err := CreateNote(root, "###"); err == nil {
		t.Fatal("CreateNote accepted a title with no usable filename")
	}
}

func TestCreateDecisionUsesSafeUniqueFilenameAndTemplate(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	first, err := CreateDecision(root, "Use Redis", "api", "feature/cache", now)
	if err != nil {
		t.Fatal(err)
	}
	second, err := CreateDecision(root, "Use Redis", "api", "feature/cache", now)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(first) != "2026-07-22-use-redis.md" || filepath.Base(second) != "2026-07-22-use-redis-2.md" {
		t.Fatalf("decision paths = %q, %q", first, second)
	}
	data, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Decision: Use Redis", "- Project: api", "- Branch: feature/cache", "## Context", "## Consequences"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("decision missing %q:\n%s", want, data)
		}
	}
}

// A title is free-form text and can be a pasted paragraph. Without a cap the
// filename runs past the 255-byte limit every mainstream filesystem enforces,
// and note creation fails with ENAMETOOLONG.
func TestSlugIsCappedAndStaysOnRuneBoundaries(t *testing.T) {
	long := strings.Repeat("žluťoučký kůň úpěl ďábelské ódy ", 20)
	slug, err := Slug(long)
	if err != nil {
		t.Fatalf("Slug() error = %v", err)
	}
	if len([]rune(slug)) > 80 || !utf8.ValidString(slug) {
		t.Fatalf("Slug() = %q (%d runes)", slug, len([]rune(slug)))
	}
	if strings.HasPrefix(slug, "-") || strings.HasSuffix(slug, "-") {
		t.Fatalf("Slug() = %q, want no dangling separator", slug)
	}
}

func TestCreateNoteAndDecisionAcceptAVeryLongTitle(t *testing.T) {
	root := t.TempDir()
	long := strings.Repeat("cache invalidation and replay detection ", 20)

	notePath, err := CreateNote(root, long)
	if err != nil {
		t.Fatalf("CreateNote() error = %v", err)
	}
	if _, err := os.Stat(notePath); err != nil {
		t.Fatalf("note was not written: %v", err)
	}

	decisionPath, err := CreateDecision(root, long, "api", "main", time.Date(2026, 7, 30, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("CreateDecision() error = %v", err)
	}
	if _, err := os.Stat(decisionPath); err != nil {
		t.Fatalf("decision was not written: %v", err)
	}
}
