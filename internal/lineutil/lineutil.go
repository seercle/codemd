// Package lineutil splits and rejoins text while preserving its line endings
// and trailing-newline state.
package lineutil

import "strings"

// Lines is text split into its lines along with the line ending in use and
// whether the original text ended with a newline, so it can be rejoined
// losslessly.
type Lines struct {
	Content         []string
	EOL             string
	TrailingNewline bool
}

// Split splits s into lines while recording the detected line ending and
// whether the input ended with a newline. A trailing newline does not produce
// an empty final element.
func Split(s string) Lines {
	l := Lines{EOL: "\n"}
	if s == "" {
		return l
	}
	l.TrailingNewline = strings.HasSuffix(s, "\n")
	body := s
	if l.TrailingNewline {
		body = s[:len(s)-1]
	}
	l.Content = strings.Split(body, "\n")
	for i, line := range l.Content {
		if strings.HasSuffix(line, "\r") {
			l.Content[i] = line[:len(line)-1]
			l.EOL = "\r\n"
		}
	}
	return l
}

// LeadingSpace returns the run of spaces and tabs at the start of s.
func LeadingSpace(s string) string {
	trimmed := strings.TrimLeft(s, " \t")
	return s[:len(s)-len(trimmed)]
}

// LeadingRun returns the length of the run of ch bytes at the start of s,
// ignoring any leading spaces or tabs.
func LeadingRun(s string, ch byte) int {
	trimmed := strings.TrimLeft(s, " \t")
	n := 0
	for n < len(trimmed) && trimmed[n] == ch {
		n++
	}
	return n
}

// Join reassembles the lines using the recorded line ending and restores the
// trailing newline, if any.
func (l Lines) Join() string {
	if len(l.Content) == 0 {
		if l.TrailingNewline {
			return "\n"
		}
		return ""
	}
	out := strings.Join(l.Content, l.EOL)
	if l.TrailingNewline {
		out += l.EOL
	}
	return out
}
