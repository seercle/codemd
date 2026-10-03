// Package render builds the Markdown artifacts (fenced snippets and links)
// inserted for resolved references.
package render

import (
	"fmt"
	"strings"
)

// Snippet wraps lines in a Markdown fenced code block, using lang as the fence
// info string when it is non-empty. The fence is made longer than the longest
// run of backticks at the start of any body line so embedded fences cannot
// close the block early.
func Snippet(lang string, lines []string) []string {
	fence := fenceFor(lines)
	open := fence
	if lang != "" {
		open += lang
	}
	out := make([]string, 0, len(lines)+2)
	out = append(out, open)
	out = append(out, lines...)
	out = append(out, fence)
	return out
}

// fenceFor returns the shortest backtick run (at least three) that is longer
// than every leading backtick run in lines.
func fenceFor(lines []string) string {
	longest := 0
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		n := 0
		for n < len(trimmed) && trimmed[n] == '`' {
			n++
		}
		if n > longest {
			longest = n
		}
	}
	n := 3
	if longest >= n {
		n = longest + 1
	}
	return strings.Repeat("`", n)
}

// LinkLabel returns the default link label for a source location, formatted as
// "path:line".
func LinkLabel(path string, line int) string {
	return fmt.Sprintf("%s:%d", path, line)
}

// LinkTarget returns the Markdown link target "path#Lline" for a source
// location.
func LinkTarget(path string, line int) string {
	return fmt.Sprintf("%s#L%d", path, line)
}

// LinkLabelRange returns the default link label for a source range, formatted as
// "path:start-end".
func LinkLabelRange(path string, start, end int) string {
	return fmt.Sprintf("%s:%d-%d", path, start, end)
}

// LinkTargetRange returns the Markdown link target "path#Lstart-Lend" for a
// source range.
func LinkTargetRange(path string, start, end int) string {
	return fmt.Sprintf("%s#L%d-L%d", path, start, end)
}

// EscapeLabel escapes a link label for use inside Markdown link brackets.
// Backslashes and closing brackets are backslash-escaped.
func EscapeLabel(label string) string {
	label = strings.ReplaceAll(label, `\`, `\\`)
	label = strings.ReplaceAll(label, `]`, `\]`)
	return label
}
