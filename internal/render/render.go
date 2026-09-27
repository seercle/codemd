// Package render builds the Markdown artifacts (fenced snippets and links)
// inserted for resolved references.
package render

import (
	"fmt"
	"strings"
)

// Snippet wraps lines in a Markdown fenced code block, using lang as the fence
// info string when it is non-empty.
func Snippet(lang string, lines []string) []string {
	open := "```"
	if lang != "" {
		open += lang
	}
	out := make([]string, 0, len(lines)+2)
	out = append(out, open)
	out = append(out, lines...)
	out = append(out, "```")
	return out
}

// LinkLabel returns the default link label for a source location, formatted as
// "path:line".
func LinkLabel(path string, line int) string {
	return fmt.Sprintf("%s:%d", path, line)
}

// LinkTarget returns the Markdown link target "path#Lline" for a source
// location. isRemote notes that path is a remote URL; the target format is the
// same for local and remote paths.
func LinkTarget(path string, line int, isRemote bool) string {
	return fmt.Sprintf("%s#L%d", path, line)
}

// EscapeLabel escapes a link label for use inside Markdown link brackets.
// Backslashes and closing brackets are backslash-escaped.
func EscapeLabel(label string) string {
	label = strings.ReplaceAll(label, `\`, `\\`)
	label = strings.ReplaceAll(label, `]`, `\]`)
	return label
}
