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
		if idx := strings.Index(line, form.Line); idx >= 0 {
			return line[idx+len(form.Line):], true
		}
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
	return ExtractMarkersMulti(content, []lang.CommentForm{l.Form})
}

// ExtractMarkersMulti finds markers using any of the given comment forms.
// For each line, the first form that yields a valid marker wins.
func ExtractMarkersMulti(content string, forms []lang.CommentForm) ([]Marker, error) {
	lines := lineutil.Split(content)
	var out []Marker
	seen := map[string]int{}
	for i, line := range lines.Content {
		for _, form := range forms {
			text, ok := CommentText(line, form)
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
			break
		}
	}
	return out, nil
}
