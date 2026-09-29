package srcfile

import (
	"regexp"
	"strings"

	"github.com/seercle/codemd/internal/lang"
	"github.com/seercle/codemd/internal/lineutil"
)

const markerPrefix = "codemd:"

var markerName = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

// Marker is a named anchor in a source file, recorded at the 1-based line of
// its codemd comment.
type Marker struct {
	Name string
	Line int
}

// CommentText returns the text following the comment delimiter of form on line,
// or the content between a block comment's delimiters. ok is false when line
// contains no comment in that form.
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

// MarkerSet is the markers found in a source plus the names that occurred more
// than once. Markers holds the first occurrence of each name in line order.
type MarkerSet struct {
	Markers    []Marker
	Duplicates map[string][]int
}

// ExtractMarkers returns the markers found in content using l's single comment
// form.
func ExtractMarkers(content string, l lang.Language) MarkerSet {
	return ExtractMarkersMulti(content, []lang.CommentForm{l.Form})
}

// ExtractMarkersMulti finds markers using any of the given comment forms. For
// each line, the first form that yields a valid marker wins. A name that
// appears more than once is recorded in Duplicates with every line number;
// Markers keeps the first occurrence. Callers decide whether the reference they
// resolve is ambiguous, so an unrelated duplicate does not fail the file.
func ExtractMarkersMulti(content string, forms []lang.CommentForm) MarkerSet {
	lines := lineutil.Split(content)
	set := MarkerSet{Duplicates: map[string][]int{}}
	first := map[string]int{}
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
			if prev, dup := first[name]; dup {
				if _, seen := set.Duplicates[name]; !seen {
					set.Duplicates[name] = []int{prev}
				}
				set.Duplicates[name] = append(set.Duplicates[name], i+1)
			} else {
				first[name] = i + 1
				set.Markers = append(set.Markers, Marker{Name: name, Line: i + 1})
			}
			break
		}
	}
	return set
}
