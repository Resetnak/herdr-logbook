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
		"HERDR_MEMORY_EDITOR": `code --wait "folder with spaces"`,
		"VISUAL":              "vim",
		"EDITOR":              "nano",
	}
	got, err := Resolve(nil, func(key string) string { return env[key] }, "linux", fakeLookup("code", "vim", "nano"))
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	want := []string{"/bin/code", "--wait", "folder with spaces"}
	if !reflect.DeepEqual(got.Command, want) || got.Source != "HERDR_MEMORY_EDITOR" {
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
