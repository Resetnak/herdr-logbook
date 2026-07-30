package markdown

import (
	"reflect"
	"testing"
)

func TestTagsExtractsFrontMatterLists(t *testing.T) {
	content := "---\ntitle: Cache\ntags: [Go, performance]\n---\n# Cache\n"
	if got, want := Tags(content), []string{"go", "performance"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Tags() = %v, want %v", got, want)
	}
}

func TestTagsExtractsMultilineAndDeduplicates(t *testing.T) {
	content := "---\ntags:\n  - Backend\n  - auth\n  - backend\n---\n"
	if got, want := Tags(content), []string{"auth", "backend"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Tags() = %v, want %v", got, want)
	}
}

func TestTagsExtractsHashTagsButIgnoresFencedCode(t *testing.T) {
	content := "# Cache\nUse #backend and #perf.\n```sh\necho #secret\n```\n"
	if got, want := Tags(content), []string{"backend", "perf"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Tags() = %v, want %v", got, want)
	}
}

// Every note written by the plugin starts with Markdown headings, and "##" is a
// field starting with "#" — it must not register as a tag named "#".
func TestTagsIgnoresHeadingMarkers(t *testing.T) {
	content := "# Now\n\n## Current task\n\n### Details\n\nShip #release today.\n"
	if got, want := Tags(content), []string{"release"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Tags() = %v, want %v", got, want)
	}
}
