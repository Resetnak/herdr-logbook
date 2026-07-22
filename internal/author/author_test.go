package author

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
