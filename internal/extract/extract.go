// Package extract locates a region of a source file by named markers or regular
// expressions and returns the selected lines.
package extract

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/seercle/codemd/internal/lineutil"
	"github.com/seercle/codemd/internal/srcfile"
)

// Bound is one end of a Range. Exactly one of Name, Regex, or Open is set:
// Name matches a marker, Regex matches a line pattern, and Open leaves the end
// unbounded.
type Bound struct {
	Name  string
	Regex string
	Open  bool
}

// Range is a pair of bounds selecting a region of a source file.
type Range struct {
	Start Bound
	End   Bound
}

// Result is the outcome of resolving a Range: the 1-based StartLine and
// EndLine selected and the extracted Lines.
type Result struct {
	StartLine int
	EndLine   int
	Lines     []string
}

func findLine(lines []string, markers []srcfile.Marker, b Bound, from int) (int, error) {
	if b.Open {
		return 0, nil
	}
	if b.Name != "" {
		for _, m := range markers {
			if m.Name == b.Name && m.Line >= from {
				return m.Line, nil
			}
		}
		return 0, fmt.Errorf("marker %q not found at or after line %d", b.Name, from)
	}
	re, err := regexp.Compile(b.Regex)
	if err != nil {
		return 0, fmt.Errorf("bad regex %q: %w", b.Regex, err)
	}
	for i := from - 1; i < len(lines); i++ {
		if re.MatchString(lines[i]) {
			return i + 1, nil
		}
	}
	return 0, fmt.Errorf("regex %q matched no line at or after %d", b.Regex, from)
}

// Resolve returns the lines of content covered by r, whose bounds are marker
// names or regexes. Named bounds are excluded and regex bounds included; an
// open bound extends to the start or end of the file. When strip is true the
// regex matched by a boundary is removed from that boundary line, and blank
// lines at the edges of the result are dropped. It returns an error when a
// bound cannot be found, a regex fails to compile, or the range selects no
// lines.
func Resolve(content string, markers []srcfile.Marker, r Range, strip bool) (Result, error) {
	lines := lineutil.Split(content)
	start, err := findLine(lines.Content, markers, r.Start, 1)
	if err != nil {
		return Result{}, err
	}
	if r.Start.Open {
		start = 1
	}
	end, err := findLine(lines.Content, markers, r.End, start)
	if err != nil {
		return Result{}, err
	}
	if r.End.Open {
		end = len(lines.Content)
	}
	// Named points are excluded; regex bounds are included.
	first := start
	if !r.Start.Open && r.Start.Name != "" {
		first = start + 1
	}
	last := end
	if !r.End.Open && r.End.Name != "" {
		last = end - 1
	}
	if first > last {
		return Result{}, fmt.Errorf("empty range %s..%s", describeBound(r.Start), describeBound(r.End))
	}
	var startRe, endRe *regexp.Regexp
	if strip {
		startRe, endRe = compileBound(r.Start), compileBound(r.End)
	}
	out := make([]string, 0, last-first+1)
	for i := first; i <= last; i++ {
		line := lines.Content[i-1]
		if strip {
			line = stripBound(line, startRe, i == start)
			line = stripBound(line, endRe, i == end)
		}
		out = append(out, line)
	}
	if strip {
		out = dropEmptyBoundaries(out)
	}
	return Result{StartLine: first, EndLine: last, Lines: out}, nil
}

// compileBound compiles b's regex, or returns nil when b has none. A bound
// that reaches stripping has already been compiled by findLine, so a
// compilation error here cannot occur and yields no stripping.
func compileBound(b Bound) *regexp.Regexp {
	if b.Regex == "" {
		return nil
	}
	re, err := regexp.Compile(b.Regex)
	if err != nil {
		return nil
	}
	return re
}

func stripBound(line string, re *regexp.Regexp, isBound bool) string {
	if !isBound || re == nil {
		return line
	}
	loc := re.FindStringIndex(line)
	if loc == nil {
		return line
	}
	return line[:loc[0]] + line[loc[1]:]
}

func dropEmptyBoundaries(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func describeBound(b Bound) string {
	switch {
	case b.Open:
		return ""
	case b.Regex != "":
		return "/" + b.Regex + "/"
	default:
		return b.Name
	}
}

// ResolveLink returns the 1-based line number in content matched by bound b,
// used to build a link to a single source location. It returns an error when
// the bound cannot be found.
func ResolveLink(content string, markers []srcfile.Marker, b Bound) (int, error) {
	return findLine(lineutil.Split(content).Content, markers, b, 1)
}
