package markdown

import "strings"

// IsTerminalControl reports whether r is a control character that must never
// reach a terminal from note content.
//
// Note bodies and titles are painted straight into the user's terminal: titles
// as plain list rows without Glamour, bodies through Glamour, which passes OSC
// sequences through untouched. An escape stored in a note is therefore an escape
// executed on display — OSC 52 rewrites the reader's system clipboard, OSC 0
// retitles their window. Newlines and tabs are ordinary Markdown and stay.
func IsTerminalControl(r rune) bool {
	if r == '\n' || r == '\t' {
		return false
	}
	return r < 0x20 || (r >= 0x7F && r <= 0x9F)
}

// StripTerminalControl removes the characters IsTerminalControl rejects. Use it
// on text that is already stored: capture refuses control characters on the way
// in, but a note written by an external editor never passed through that check.
func StripTerminalControl(value string) string {
	if !strings.ContainsFunc(value, IsTerminalControl) {
		return value
	}
	return strings.Map(func(r rune) rune {
		if IsTerminalControl(r) {
			return -1
		}
		return r
	}, value)
}
