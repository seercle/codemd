package mdref

import (
	"strings"

	"github.com/seercle/codemd/internal/lineutil"
)

type Reference struct {
	Line    int
	Comment string
	Ref     Ref
}

const refPrefix = "[codemd]:#"

func ExtractComment(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, refPrefix) {
		return "", false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(trimmed, refPrefix))
	if !strings.HasPrefix(rest, "(") {
		return "", false
	}
	end := strings.LastIndex(rest, ")")
	if end < 0 {
		return "", false
	}
	return rest[:end+1], true
}

type ScanError struct {
	Line int
	Err  error
}

// fenceInfo reports the fence character and run length if line is a fence
// (``` or ~~~ with <=3 leading spaces and a run of at least 3).
func fenceInfo(line string) (ch byte, n int, ok bool) {
	trimmed := strings.TrimLeft(line, " \t")
	if indent := len(line) - len(trimmed); indent > 3 {
		return 0, 0, false
	}
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
