package srcfile

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/seercle/codemd/internal/lang"
	"github.com/seercle/codemd/internal/lineutil"
)

const markerPrefix = "codemd:"

var markerName = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

type Marker struct {
	Name string
	Line int
}

func CommentText(line string, form lang.CommentForm) (string, bool) {
	if form.Line != "" {
		idx := strings.Index(line, form.Line)
		if idx < 0 {
			return "", false
		}
		return line[idx+len(form.Line):], true
	}
	if form.Block != [2]string{} {
		start := strings.Index(line, form.Block[0])
		if start < 0 {
			return "", false
		}
		rest := line[start+len(form.Block[0]):]
		end := strings.Index(rest, form.Block[1])
		if end < 0 {
			return "", false
		}
		return rest[:end], true
	}
	return "", false
}

func ExtractMarkers(content string, l lang.Language) ([]Marker, error) {
	lines := lineutil.Split(content)
	var out []Marker
	seen := map[string]int{}
	for i, line := range lines.Content {
		text, ok := CommentText(line, l.Form)
		if !ok {
			continue
		}
		text = strings.TrimSpace(text)
		if !strings.HasPrefix(text, markerPrefix) {
			continue
		}
		name := strings.TrimSpace(strings.TrimPrefix(text, markerPrefix))
		if !markerName.MatchString(name) || strings.Contains(name, "..") {
			continue
		}
		if prev, dup := seen[name]; dup {
			return nil, fmt.Errorf("duplicate marker %q on lines %d and %d", name, prev, i+1)
		}
		seen[name] = i + 1
		out = append(out, Marker{Name: name, Line: i + 1})
	}
	return out, nil
}
