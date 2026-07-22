package editor

import (
	"fmt"
	"strings"
	"unicode"
)

type Resolution struct {
	Command []string `json:"command,omitempty"`
	Source  string   `json:"source,omitempty"`
}

func Resolve(configured []string, getenv func(string) string, goos string, lookPath func(string) (string, error)) (Resolution, error) {
	if len(configured) > 0 {
		return resolveCommand(configured, "config", lookPath)
	}
	for _, key := range []string{"HERDR_MEMORY_EDITOR", "VISUAL", "EDITOR"} {
		if value := strings.TrimSpace(getenv(key)); value != "" {
			command, err := parseCommand(value)
			if err != nil {
				return Resolution{}, fmt.Errorf("parse %s: %w", key, err)
			}
			return resolveCommand(command, key, lookPath)
		}
	}
	candidates := []string{"nvim", "vim", "vi", "nano"}
	if goos == "windows" {
		candidates = []string{"nvim.exe", "vim.exe", "notepad.exe"}
	}
	for _, candidate := range candidates {
		if path, err := lookPath(candidate); err == nil {
			return Resolution{Command: []string{path}, Source: "platform-default"}, nil
		}
	}
	return Resolution{}, fmt.Errorf("no external editor found")
}

func resolveCommand(command []string, source string, lookPath func(string) (string, error)) (Resolution, error) {
	if len(command) == 0 || command[0] == "" {
		return Resolution{}, fmt.Errorf("%s editor command is empty", source)
	}
	path, err := lookPath(command[0])
	if err != nil {
		return Resolution{}, fmt.Errorf("%s editor %q not found: %w", source, command[0], err)
	}
	resolved := append([]string{path}, command[1:]...)
	return Resolution{Command: resolved, Source: source}, nil
}

func parseCommand(value string) ([]string, error) {
	var result []string
	var token []rune
	var quote rune
	started := false
	runes := []rune(value)
	flush := func() {
		if started {
			result = append(result, string(token))
			token = nil
			started = false
		}
	}
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if quote == 0 {
			switch {
			case unicode.IsSpace(r):
				flush()
			case r == '\'' || r == '"':
				quote = r
				started = true
			default:
				token = append(token, r)
				started = true
			}
			continue
		}
		if r == quote {
			quote = 0
			continue
		}
		if quote == '"' && r == '\\' && i+1 < len(runes) && (runes[i+1] == '"' || runes[i+1] == '\\') {
			i++
			r = runes[i]
		}
		token = append(token, r)
	}
	if quote != 0 {
		return nil, fmt.Errorf("unclosed quote")
	}
	flush()
	if len(result) == 0 {
		return nil, fmt.Errorf("empty command")
	}
	return result, nil
}
