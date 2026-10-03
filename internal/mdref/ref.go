// Package mdref scans Markdown for codemd reference definitions and parses
// them into structured references. References use the HTML-comment form
// "<!-- codemd: (...) -->".
package mdref

import (
	"fmt"
	"strings"

	"github.com/seercle/codemd/internal/extract"
)

// Mode selects how a reference is resolved: Import inlines the referenced
// source, while Link emits a Markdown link to it.
type Mode int

const (
	// Import inlines the referenced lines as a fenced code block.
	Import Mode = iota
	// Link emits a Markdown link to the referenced line.
	Link
)

// Ref is a parsed reference: its resolution Mode, the source Range it selects,
// the target Path, an optional fence Lang, a Strip flag, a link Label, and
// whether the reference token was a ".." range (IsRange).
type Ref struct {
	Mode    Mode
	Range   extract.Range
	Path    string
	Lang    string
	Strip   bool
	Label   string
	IsRange bool
}

// ParseRef parses a reference argument list (the text inside the parentheses
// of a reference definition) into a Ref. It returns an error when the mode,
// range, path, or remaining tokens are malformed.
func ParseRef(comment string) (Ref, error) {
	if strings.Contains(comment, refClose) {
		return Ref{}, fmt.Errorf("reference must not contain %q", refClose)
	}
	s := strings.TrimSpace(comment)
	s = strings.TrimPrefix(s, "(")
	s = strings.TrimSuffix(s, ")")
	s = strings.TrimSpace(s)
	i := skipSpace(s, 0)
	modeStart := i
	for i < len(s) && !isSpace(s[i]) {
		i++
	}
	mode := s[modeStart:i]
	if mode == "" {
		return Ref{}, fmt.Errorf("reference too short: %q", comment)
	}
	i = skipSpace(s, i)
	rangeTok, i := scanRangeToken(s, i)
	if rangeTok == "" {
		return Ref{}, fmt.Errorf("reference too short: %q", comment)
	}
	rest, err := splitTokens(s[i:])
	if err != nil {
		return Ref{}, fmt.Errorf("%v in %q", err, comment)
	}
	var r Ref
	switch mode {
	case "import":
		r.Mode = Import
	case "link":
		r.Mode = Link
	default:
		return Ref{}, fmt.Errorf("unknown mode %q", mode)
	}
	left, right, isRange := splitRange(rangeTok)
	switch {
	case isRange:
		start, err := parseToken(left)
		if err != nil {
			return Ref{}, err
		}
		end, err := parseToken(right)
		if err != nil {
			return Ref{}, err
		}
		r.Range = extract.Range{Start: start, End: end}
		r.IsRange = r.Mode == Link
	case r.Mode == Import:
		return Ref{}, fmt.Errorf("import range must contain '..': %q", rangeTok)
	default:
		b, err := parseToken(rangeTok)
		if err != nil {
			return Ref{}, err
		}
		r.Range = extract.Range{Start: b, End: extract.Bound{Open: true}}
	}
	if len(rest) == 0 {
		return Ref{}, fmt.Errorf("missing path in %q", comment)
	}
	if strings.HasPrefix(rest[0], `"`) {
		return Ref{}, fmt.Errorf("path must not be quoted in %q", comment)
	}
	r.Path = rest[0]
	rest = rest[1:]
	for _, tok := range rest {
		switch {
		case tok == "strip":
			r.Strip = true
		case strings.HasPrefix(tok, `"`):
			if r.Mode != Link {
				return Ref{}, fmt.Errorf("link text is only valid for link mode in %q", comment)
			}
			label, err := unquote(tok)
			if err != nil {
				return Ref{}, fmt.Errorf("%v in %q", err, comment)
			}
			if strings.TrimSpace(label) == "" {
				return Ref{}, fmt.Errorf("empty link text in %q", comment)
			}
			if r.Label != "" {
				return Ref{}, fmt.Errorf("multiple link labels in %q", comment)
			}
			r.Label = label
		case r.Lang == "":
			r.Lang = tok
		default:
			return Ref{}, fmt.Errorf("unexpected token %q in %q", tok, comment)
		}
	}
	if r.Strip && r.Range.Start.Regex == "" && r.Range.End.Regex == "" {
		return Ref{}, fmt.Errorf("strip requires a regex token in %q", comment)
	}
	return r, nil
}

// splitTokens splits s on whitespace, treating a double-quoted run as a single
// token (quotes included). It returns an error for an unterminated quote.
func splitTokens(s string) ([]string, error) {
	var toks []string
	i := 0
	for i < len(s) {
		for i < len(s) && isSpace(s[i]) {
			i++
		}
		if i >= len(s) {
			break
		}
		if s[i] == '"' {
			end, ok := scanQuote(s, i)
			if !ok {
				return nil, fmt.Errorf("unterminated quoted string")
			}
			toks = append(toks, s[i:end])
			i = end
			continue
		}
		start := i
		for i < len(s) && !isSpace(s[i]) {
			i++
		}
		toks = append(toks, s[start:i])
	}
	return toks, nil
}

// unquote strips the surrounding quotes from a quoted token and interprets
// backslash escapes: \\, \", \] and any \c yield the following byte.
func unquote(tok string) (string, error) {
	if len(tok) < 2 || tok[0] != '"' || tok[len(tok)-1] != '"' {
		return "", fmt.Errorf("malformed quoted string %q", tok)
	}
	body := tok[1 : len(tok)-1]
	var b strings.Builder
	for i := 0; i < len(body); i++ {
		if body[i] == '\\' && i+1 < len(body) {
			i++
		}
		b.WriteByte(body[i])
	}
	return b.String(), nil
}

// scanQuote returns the index just past the closing '"' of the quoted token
// starting at s[i] (which must be '"'), and whether it was terminated. A
// backslash escapes the next byte, so an escaped '"' does not close the token.
func scanQuote(s string, i int) (end int, ok bool) {
	for j := i + 1; j < len(s); j++ {
		switch s[j] {
		case '\\':
			if j+1 < len(s) {
				j++
			}
		case '"':
			return j + 1, true
		}
	}
	return len(s), false
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func skipSpace(s string, i int) int {
	for i < len(s) && isSpace(s[i]) {
		i++
	}
	return i
}

// scanRangeToken reads a range token: a bound optionally followed by ".." and
// a second bound. A bound is either a /regex/ (which may contain spaces or
// escaped slashes) or a name run. It returns the token and the index just past
// it.
func scanRangeToken(s string, i int) (string, int) {
	start := i
	i = scanBound(s, i, true)
	if i+1 < len(s) && s[i] == '.' && s[i+1] == '.' {
		i += 2
		i = scanBound(s, i, false)
	}
	return s[start:i], i
}

func scanBound(s string, i int, left bool) int {
	if i >= len(s) {
		return i
	}
	if s[i] == '/' {
		end, _ := scanRegex(s, i)
		return end
	}
	for i < len(s) && !isSpace(s[i]) {
		if left && i+1 < len(s) && s[i] == '.' && s[i+1] == '.' {
			break
		}
		i++
	}
	return i
}

// scanRegex returns the index just past the closing '/' of the regex token
// starting at s[i] (which must be '/'), and whether it was terminated. A
// backslash escapes the next byte, so an escaped '/' does not close the token.
func scanRegex(s string, i int) (end int, ok bool) {
	for j := i + 1; j < len(s); j++ {
		switch s[j] {
		case '\\':
			j++
		case '/':
			return j + 1, true
		}
	}
	return len(s), false
}

// splitRange splits on the first ".." that is not inside a regex token.
func splitRange(tok string) (string, string, bool) {
	i := 0
	for i < len(tok) {
		if tok[i] == '/' {
			end, ok := scanRegex(tok, i)
			if !ok {
				return "", "", false
			}
			i = end
			continue
		}
		if strings.HasPrefix(tok[i:], "..") {
			return tok[:i], tok[i+2:], true
		}
		i++
	}
	return "", "", false
}

func parseToken(tok string) (extract.Bound, error) {
	if tok == "" {
		return extract.Bound{Open: true}, nil
	}
	if strings.HasPrefix(tok, "/") {
		if len(tok) < 2 || !strings.HasSuffix(tok, "/") {
			return extract.Bound{}, fmt.Errorf("unterminated regex %q", tok)
		}
		body := tok[1 : len(tok)-1]
		body = strings.ReplaceAll(body, `\/`, `/`)
		return extract.Bound{Regex: body}, nil
	}
	return extract.Bound{Name: tok}, nil
}
