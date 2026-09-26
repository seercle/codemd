package render

import "fmt"

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

func LinkLabel(path string, line int) string {
	return fmt.Sprintf("%s:%d", path, line)
}

func LinkTarget(path string, line int, isRemote bool) string {
	return fmt.Sprintf("%s#L%d", path, line)
}
