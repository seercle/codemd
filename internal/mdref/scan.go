package mdref

import (
	"fmt"
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

func Scan(content string) ([]Reference, error) {
	lines := lineutil.Split(content)
	var refs []Reference
	var fenceChar byte
	var fenceLen int
	for i, line := range lines.Content {
		trimmed := strings.TrimLeft(line, " \t")
		if indent := len(line) - len(trimmed); indent <= 3 && (strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")) {
			ch := trimmed[0]
			n := 0
			for n < len(trimmed) && trimmed[n] == ch {
				n++
			}
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
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		refs = append(refs, Reference{Line: i + 1, Comment: comment, Ref: ref})
	}
	return refs, nil
}
