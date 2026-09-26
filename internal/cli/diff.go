package cli

import (
	"fmt"
	"path/filepath"
	"strings"
)

// unifiedDiff renders a unified diff between old and new with 3 lines of
// context per hunk. name is the file path used in the ---/+++ headers.
func unifiedDiff(name, old, new string) string {
	oldLines := splitDiffLines(old)
	newLines := splitDiffLines(new)
	clean := strings.TrimPrefix(name, string(filepath.Separator))
	var b strings.Builder
	fmt.Fprintf(&b, "--- a/%s\n+++ b/%s\n", clean, clean)
	for _, h := range diffHunks(oldLines, newLines, 3) {
		fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", h.oldStart, h.oldCount, h.newStart, h.newCount)
		for _, l := range h.lines {
			b.WriteString(l)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func splitDiffLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

type diffOp struct {
	kind byte // ' ' equal, '-' delete, '+' insert
	text string
}

type diffHunk struct {
	oldStart, oldCount int
	newStart, newCount int
	lines              []string
}

// diffOps computes an LCS-based edit script from a to b.
func diffOps(a, b []string) []diffOp {
	n, m := len(a), len(b)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			switch {
			case a[i] == b[j]:
				dp[i][j] = dp[i+1][j+1] + 1
			case dp[i+1][j] >= dp[i][j+1]:
				dp[i][j] = dp[i+1][j]
			default:
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	var ops []diffOp
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			ops = append(ops, diffOp{' ', a[i]})
			i++
			j++
		case dp[i+1][j] >= dp[i][j+1]:
			ops = append(ops, diffOp{'-', a[i]})
			i++
		default:
			ops = append(ops, diffOp{'+', b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		ops = append(ops, diffOp{'-', a[i]})
	}
	for ; j < m; j++ {
		ops = append(ops, diffOp{'+', b[j]})
	}
	return ops
}

// diffHunks groups the edit script into hunks with up to context equal lines
// on either side of each change, merging hunks whose context windows touch.
func diffHunks(a, b []string, context int) []diffHunk {
	ops := diffOps(a, b)
	oldAt := make([]int, len(ops)+1)
	newAt := make([]int, len(ops)+1)
	ol, nl := 1, 1
	for i, op := range ops {
		oldAt[i] = ol
		newAt[i] = nl
		switch op.kind {
		case ' ':
			ol++
			nl++
		case '-':
			ol++
		case '+':
			nl++
		}
	}
	oldAt[len(ops)] = ol
	newAt[len(ops)] = nl

	type span struct{ lo, hi int }
	var spans []span
	for i, op := range ops {
		if op.kind == ' ' {
			continue
		}
		lo := i - context
		if lo < 0 {
			lo = 0
		}
		hi := i + context
		if hi > len(ops)-1 {
			hi = len(ops) - 1
		}
		if n := len(spans); n > 0 && lo <= spans[n-1].hi+1 {
			if hi > spans[n-1].hi {
				spans[n-1].hi = hi
			}
		} else {
			spans = append(spans, span{lo, hi})
		}
	}

	hunks := make([]diffHunk, 0, len(spans))
	for _, s := range spans {
		h := diffHunk{oldStart: oldAt[s.lo], newStart: newAt[s.lo]}
		for _, op := range ops[s.lo : s.hi+1] {
			h.lines = append(h.lines, string(op.kind)+op.text)
			switch op.kind {
			case ' ':
				h.oldCount++
				h.newCount++
			case '-':
				h.oldCount++
			case '+':
				h.newCount++
			}
		}
		if h.oldCount == 0 {
			h.oldStart--
		}
		if h.newCount == 0 {
			h.newStart--
		}
		hunks = append(hunks, h)
	}
	return hunks
}
