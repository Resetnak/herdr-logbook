package editor

import (
	"fmt"
	"reflect"
	"testing"
)

func TestResolveUsesConfiguredArgvBeforeEnvironment(t *testing.T) {
	got, err := Resolve([]string{"nvim", "-f"}, func(string) string { return "code --wait" }, "linux", fakeLookup("nvim", "code"))
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if !reflect.DeepEqual(got.Command, []string{"/bin/nvim", "-f"}) || got.Source != "config" {
		t.Fatalf("Resolve() = %#v", got)
	}
}

func TestResolveEnvironmentPrecedenceAndArguments(t *testing.T) {
	env := map[string]string{
		"HERDR_LOGBOOK_EDITOR": `code --wait "folder with spaces"`,
		"VISUAL":               "vim",
		"EDITOR":               "nano",
	}
	got, err := Resolve(nil, func(key string) string { return env[key] }, "linux", fakeLookup("code", "vim", "nano"))
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	want := []string{"/bin/code", "--wait", "folder with spaces"}
	if !reflect.DeepEqual(got.Command, want) || got.Source != "HERDR_LOGBOOK_EDITOR" {
		t.Fatalf("Resolve() = %#v, want %#v", got, want)
	}
}

func TestResolveUsesPlatformFallback(t *testing.T) {
	got, err := Resolve(nil, func(string) string { return "" }, "windows", fakeLookup("notepad.exe"))
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if !reflect.DeepEqual(got.Command, []string{"/bin/notepad.exe"}) || got.Source != "platform-default" {
		t.Fatalf("Resolve() = %#v", got)
	}
}

func fakeLookup(available ...string) func(string) (string, error) {
	return func(name string) (string, error) {
		for _, candidate := range available {
			if name == candidate {
				return "/bin/" + name, nil
			}
		}
		return "", fmt.Errorf("not found")
	}
}

func TestResolveRejectsUnusableEditorCommands(t *testing.T) {
	found := func(string) (string, error) { return "/usr/bin/vim", nil }
	missing := func(name string) (string, error) { return "", fmt.Errorf("%s not found in PATH", name) }
	noEnv := func(string) string { return "" }

	if _, err := Resolve(nil, func(key string) string {
		if key == "EDITOR" {
			return `vim "unterminated`
		}
		return ""
	}, "linux", found); err == nil {
		t.Fatal("Resolve accepted an unparseable $EDITOR")
	}

	if _, err := Resolve([]string{""}, noEnv, "linux", found); err == nil {
		t.Fatal("Resolve accepted an empty configured command")
	}
	if _, err := Resolve([]string{"nvim"}, noEnv, "linux", missing); err == nil {
		t.Fatal("Resolve accepted a configured editor that is not on PATH")
	}
	if _, err := Resolve(nil, noEnv, "linux", missing); err == nil {
		t.Fatal("Resolve succeeded with no editor available at all")
	}
}

func TestParseCommandHandlesQuotesAndEscapes(t *testing.T) {
	found := func(name string) (string, error) { return "/opt/bin/" + name, nil }
	resolved, err := Resolve(nil, func(key string) string {
		if key == "HERDR_LOGBOOK_EDITOR" {
			// Backslash escapes are only honoured inside double quotes.
			return `"code editor" --wait --arg="a b" "quote:\"" 'single quoted'`
		}
		return ""
	}, "linux", found)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/opt/bin/code editor", "--wait", "--arg=a b", `quote:"`, "single quoted"}
	if !reflect.DeepEqual(resolved.Command, want) {
		t.Fatalf("parsed command = %#v, want %#v", resolved.Command, want)
	}
	if resolved.Source != "HERDR_LOGBOOK_EDITOR" {
		t.Fatalf("source = %q", resolved.Source)
	}
}
