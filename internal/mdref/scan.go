package mdref

import (
	"strings"

	"github.com/seercle/codemd/internal/lineutil"
)

// Reference is a codemd reference definition found in a Markdown document at a
// 1-based Line, with its raw Comment text and parsed Ref.
type Reference struct {
	Line    int
	Comment string
	Ref     Ref
}

const (
	refOpen  = "<!-- codemd:"
	refClose = "-->"
)

// ExtractComment returns the reference argument list from a reference
// definition, for example "(import a..b path)": the inner text of the HTML
// comment "<!-- codemd: (...) -->". ok is false when line is not a reference
// definition.
func ExtractComment(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, refOpen) || !strings.HasSuffix(trimmed, refClose) {
		return "", false
	}
	inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, refOpen), refClose))
	if strings.HasPrefix(inner, "(") && strings.HasSuffix(inner, ")") {
		return inner, true
	}
	return "", false
}

// GeneratedMarker is the HTML comment codemd writes next to a generated
// fenced block to mark that region as managed. Markdown renderers hide it, and
// Scan ignores it because its body is not a parenthesized reference.
const GeneratedMarker = "<!-- codemd:generated -->"

// IsGeneratedMarker reports whether line is the managed-region marker.
func IsGeneratedMarker(line string) bool {
	return strings.TrimSpace(line) == GeneratedMarker
}

// ScanError pairs a reference parse error with the 1-based line it occurred on.
type ScanError struct {
	Line int
	Err  error
}

// fenceInfo reports the fence character and run length if line is a fence
// (``` or ~~~ after any leading whitespace and a run of at least 3).
func fenceInfo(line string) (ch byte, n int, ok bool) {
	trimmed := strings.TrimLeft(line, " \t")
	if len(trimmed) == 0 || (trimmed[0] != '`' && trimmed[0] != '~') {
		return 0, 0, false
	}
	ch = trimmed[0]
	for n < len(trimmed) && trimmed[n] == ch {
		n++
	}
	if n < 3 {
		return 0, 0, false
	}
	return ch, n, true
}

// FenceBlockEnd returns the index just past the fenced block that starts at
// lines[start], or len(lines) if it is unterminated. ok is false when
// lines[start] is not an opening fence.
func FenceBlockEnd(lines []string, start int) (end int, ok bool) {
	ch, n, ok := fenceInfo(lines[start])
	if !ok {
		return start, false
	}
	for i := start + 1; i < len(lines); i++ {
		if cch, cn, isFence := fenceInfo(lines[i]); isFence && cch == ch && cn >= n {
			return i + 1, true
		}
	}
	return len(lines), true
}

// Scan finds every codemd reference definition in content that lies outside
// fenced code blocks. It returns the parsed references in document order and
// the per-line parse errors, allowing valid references to still be resolved.
func Scan(content string) ([]Reference, []ScanError) {
	lines := lineutil.Split(content)
	var refs []Reference
	var errs []ScanError
	var fenceChar byte
	var fenceLen int
	for i, line := range lines.Content {
		if ch, n, ok := fenceInfo(line); ok {
			if fenceChar == 0 {
				fenceChar, fenceLen = ch, n
			} else if ch == fenceChar && n >= fenceLen {
				fenceChar, fenceLen = 0, 0
			}
			continue
		}
		if fenceChar != 0 {
			continue
		}
		comment, ok := ExtractComment(line)
		if !ok {
			continue
		}
		ref, err := ParseRef(comment)
		if err != nil {
			errs = append(errs, ScanError{Line: i + 1, Err: err})
			continue
		}
		refs = append(refs, Reference{Line: i + 1, Comment: comment, Ref: ref})
	}
	return refs, errs
}
