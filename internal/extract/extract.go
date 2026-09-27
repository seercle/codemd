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

func findLine(content string, markers []srcfile.Marker, b Bound, from int) (int, error) {
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
	lines := lineutil.Split(content)
	for i := from - 1; i < len(lines.Content); i++ {
		if re.MatchString(lines.Content[i]) {
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
// bound cannot be found or a regex fails to compile.
func Resolve(content string, markers []srcfile.Marker, r Range, strip bool) (Result, error) {
	lines := lineutil.Split(content)
	start, err := findLine(content, markers, r.Start, 1)
	if err != nil {
		return Result{}, err
	}
	if r.Start.Open {
		start = 1
	}
	end, err := findLine(content, markers, r.End, start)
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
		return Result{StartLine: first, EndLine: last}, nil
	}
	out := make([]string, 0, last-first+1)
	for i := first; i <= last; i++ {
		line := lines.Content[i-1]
		if strip {
			line = stripBound(line, r.Start, i == start)
			line = stripBound(line, r.End, i == end)
		}
		out = append(out, line)
	}
	if strip {
		out = dropEmptyBoundaries(out)
	}
	return Result{StartLine: first, EndLine: last, Lines: out}, nil
}

func stripBound(line string, b Bound, isBound bool) string {
	if !isBound || b.Regex == "" {
		return line
	}
	re, err := regexp.Compile(b.Regex)
	if err != nil {
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

// ResolveLink returns the 1-based line number in content matched by bound b,
// used to build a link to a single source location. It returns an error when
// the bound cannot be found.
func ResolveLink(content string, markers []srcfile.Marker, b Bound) (int, error) {
	return findLine(content, markers, b, 1)
}
