package mdref

import (
	"fmt"
	"strings"

	"github.com/seercle/codemd/internal/extract"
)

type Mode int

const (
	Import Mode = iota
	Link
)

type Ref struct {
	Mode  Mode
	Range extract.Range
	Path  string
	Lang  string
	Strip bool
	Label string
}

func ParseRef(comment string) (Ref, error) {
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
	if r.Mode == Import {
		start, end, err := parseRange(rangeTok)
		if err != nil {
			return Ref{}, err
		}
		r.Range = extract.Range{Start: start, End: end}
	} else {
		if _, _, ok := splitRange(rangeTok); ok {
			return Ref{}, fmt.Errorf("link takes a single token, got %q", rangeTok)
		}
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
			label := tok[1 : len(tok)-1]
			if label == "" {
				return Ref{}, fmt.Errorf("empty link text in %q", comment)
			}
			if strings.Contains(label, "]") {
				return Ref{}, fmt.Errorf("link text must not contain ']' in %q", comment)
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
			j := i + 1
			for j < len(s) && s[j] != '"' {
				j++
			}
			if j >= len(s) {
				return nil, fmt.Errorf("unterminated quoted string")
			}
			toks = append(toks, s[i:j+1])
			i = j + 1
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
		i++
		for i < len(s) {
			if s[i] == '\\' {
				if i+1 < len(s) {
					i += 2
				} else {
					i++
				}
				continue
			}
			if s[i] == '/' {
				i++
				break
			}
			i++
		}
		return i
	}
	for i < len(s) && !isSpace(s[i]) {
		if left && i+1 < len(s) && s[i] == '.' && s[i+1] == '.' {
			break
		}
		i++
	}
	return i
}

func parseRange(tok string) (extract.Bound, extract.Bound, error) {
	left, right, ok := splitRange(tok)
	if !ok {
		return extract.Bound{}, extract.Bound{}, fmt.Errorf("import range must contain '..': %q", tok)
	}
	start, err := parseToken(left)
	if err != nil {
		return extract.Bound{}, extract.Bound{}, err
	}
	end, err := parseToken(right)
	if err != nil {
		return extract.Bound{}, extract.Bound{}, err
	}
	return start, end, nil
}

// splitRange splits on the first ".." that is not inside a regex token.
func splitRange(tok string) (string, string, bool) {
	i := 0
	for i < len(tok) {
		if tok[i] == '/' {
			j := i + 1
			for j < len(tok) {
				if tok[j] == '\\' {
					j += 2
					continue
				}
				if tok[j] == '/' {
					break
				}
				j++
			}
			if j >= len(tok) {
				return "", "", false
			}
			i = j + 1
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
