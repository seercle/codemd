package extract

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/seercle/codemd/internal/lineutil"
	"github.com/seercle/codemd/internal/srcfile"
)

type Bound struct {
	Name  string
	Regex string
	Open  bool
}

type Range struct {
	Start Bound
	End   Bound
}

type Result struct {
	StartLine int
	EndLine   int
	Lines     []string
}

func findLine(content string, markers []srcfile.Marker, b Bound) (int, error) {
	if b.Open {
		return 0, nil
	}
	if b.Name != "" {
		for _, m := range markers {
			if m.Name == b.Name {
				return m.Line, nil
			}
		}
		return 0, fmt.Errorf("marker %q not found", b.Name)
	}
	re, err := regexp.Compile(b.Regex)
	if err != nil {
		return 0, fmt.Errorf("bad regex %q: %w", b.Regex, err)
	}
	lines := lineutil.Split(content)
	for i, line := range lines.Content {
		if re.MatchString(line) {
			return i + 1, nil
		}
	}
	return 0, fmt.Errorf("regex %q matched no line", b.Regex)
}

func Resolve(content string, markers []srcfile.Marker, r Range, strip bool) (Result, error) {
	lines := lineutil.Split(content)
	start, err := findLine(content, markers, r.Start)
	if err != nil {
		return Result{}, err
	}
	end, err := findLine(content, markers, r.End)
	if err != nil {
		return Result{}, err
	}
	if r.Start.Open {
		start = 1
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

func ResolveLink(content string, markers []srcfile.Marker, b Bound) (int, error) {
	return findLine(content, markers, b)
}
