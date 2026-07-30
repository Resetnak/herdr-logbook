package author

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/Resetnak/herdr-logbook/internal/storage"
)

// maxSlugRunes keeps the longest slug under 160 bytes even for two-byte runes.
const maxSlugRunes = 80

func Slug(title string) (string, error) {
	var slug strings.Builder
	separator := false
	for _, char := range strings.ToLower(strings.TrimSpace(title)) {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			if separator && slug.Len() > 0 {
				slug.WriteByte('-')
			}
			slug.WriteRune(char)
			separator = false
		} else {
			separator = true
		}
	}
	value := strings.Trim(slug.String(), "-")
	if value == "" {
		return "", fmt.Errorf("title does not contain a usable filename")
	}
	// Titles are free-form text, filenames are not: mainstream filesystems stop at
	// 255 bytes, and a decision adds a date prefix on top of the slug.
	if runes := []rune(value); len(runes) > maxSlugRunes {
		value = strings.TrimRight(string(runes[:maxSlugRunes]), "-")
	}
	if windowsReserved(value) {
		value = "note-" + value
	}
	return value, nil
}

func CreateNote(storeRoot, title string) (string, error) {
	slug, err := Slug(title)
	if err != nil {
		return "", err
	}
	path, err := uniquePath(filepath.Join(storeRoot, "notes"), slug, ".md")
	if err != nil {
		return "", err
	}
	return path, storage.AtomicWrite(path, []byte("# "+strings.TrimSpace(title)+"\n"), 0o600)
}

func CreateDecision(storeRoot, title, projectName, branch string, now time.Time) (string, error) {
	slug, err := Slug(title)
	if err != nil {
		return "", err
	}
	base := now.Format("2006-01-02") + "-" + slug
	path, err := uniquePath(filepath.Join(storeRoot, "decisions"), base, ".md")
	if err != nil {
		return "", err
	}
	var content strings.Builder
	fmt.Fprintf(&content, "# Decision: %s\n\n- Date: %s\n- Status: accepted\n", strings.TrimSpace(title), now.Format("2006-01-02"))
	if projectName != "" {
		fmt.Fprintf(&content, "- Project: %s\n", projectName)
	}
	if branch != "" {
		fmt.Fprintf(&content, "- Branch: %s\n", branch)
	}
	content.WriteString("\n## Context\n\n\n## Decision\n\n\n## Consequences\n\n\n## Follow-up\n")
	return path, storage.AtomicWrite(path, []byte(content.String()), 0o600)
}

func uniquePath(directory, base, extension string) (string, error) {
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", fmt.Errorf("create authoring directory: %w", err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return "", err
	}
	existing := make(map[string]bool, len(entries))
	for _, entry := range entries {
		existing[strings.ToLower(entry.Name())] = true
	}
	for suffix := 1; ; suffix++ {
		name := base
		if suffix > 1 {
			name = fmt.Sprintf("%s-%d", base, suffix)
		}
		name += extension
		if !existing[strings.ToLower(name)] {
			return filepath.Join(directory, name), nil
		}
	}
}

func windowsReserved(name string) bool {
	upper := strings.ToUpper(strings.TrimSpace(name))
	if upper == "CON" || upper == "PRN" || upper == "AUX" || upper == "NUL" {
		return true
	}
	for index := 1; index <= 9; index++ {
		if upper == fmt.Sprintf("COM%d", index) || upper == fmt.Sprintf("LPT%d", index) {
			return true
		}
	}
	return false
}
