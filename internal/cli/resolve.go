package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/seercle/codemd/internal/extract"
	"github.com/seercle/codemd/internal/lang"
	"github.com/seercle/codemd/internal/lineutil"
	"github.com/seercle/codemd/internal/mdref"
	"github.com/seercle/codemd/internal/render"
	"github.com/seercle/codemd/internal/srcfile"
)

type Resolver struct {
	Loader *srcfile.Loader
	Table  map[string]lang.Language
}

type RefError struct {
	Line int
	Err  error
}

func (r Resolver) ResolveDocument(content, baseDir string) (string, []RefError) {
	lines := lineutil.Split(content)
	refs, err := mdref.Scan(content)
	if err != nil {
		return content, []RefError{{Line: 0, Err: err}}
	}
	var errs []RefError
	// Apply from the bottom up so earlier line numbers stay valid.
	for i := len(refs) - 1; i >= 0; i-- {
		ref := refs[i]
		replacement, err := r.resolveOne(ref, baseDir)
		if err != nil {
			errs = append([]RefError{{Line: ref.Line, Err: err}}, errs...)
			continue
		}
		lines.Content = splice(lines.Content, ref.Line, ref.Ref.Mode, replacement)
	}
	return lines.Join(), errs
}

func (r Resolver) resolveOne(ref mdref.Reference, baseDir string) ([]string, error) {
	content, isRemote, err := r.Loader.Load(ref.Ref.Path, baseDir)
	if err != nil {
		return nil, err
	}
	ext := filepath.Ext(ref.Ref.Path)
	l, ok := lang.Resolve(ext, r.Table)
	if !ok {
		l = lang.Language{Form: lang.CommentForm{Line: "//", Block: [2]string{"/*", "*/"}}}
	}
	markers, err := srcfile.ExtractMarkers(content, l)
	if err != nil {
		return nil, err
	}
	if ref.Ref.Mode == mdref.Link {
		line, err := extract.ResolveLink(content, markers, ref.Ref.Range.Start)
		if err != nil {
			return nil, err
		}
		return []string{fmt.Sprintf("[%s](%s)", render.LinkLabel(ref.Ref.Path, line), render.LinkTarget(ref.Ref.Path, line, isRemote))}, nil
	}
	res, err := extract.Resolve(content, markers, ref.Ref.Range, ref.Ref.Strip)
	if err != nil {
		return nil, err
	}
	fence := ref.Ref.Lang
	if fence == "" {
		fence = lang.FenceFor(ext, r.Table)
	}
	return render.Snippet(fence, res.Lines), nil
}

// splice replaces the managed region after refLine (1-based) with replacement.
func splice(lines []string, refLine int, mode mdref.Mode, replacement []string) []string {
	idx := refLine // 0-based index of the line after the reference
	for idx < len(lines) && strings.TrimSpace(lines[idx]) == "" {
		idx++
	}
	if idx >= len(lines) {
		out := append([]string{}, lines...)
		return append(out, replacement...)
	}
	if mode == mdref.Link {
		out := append([]string{}, lines[:idx]...)
		out = append(out, replacement...)
		return append(out, lines[idx+1:]...)
	}
	if isFenceOpen(lines[idx]) {
		end := idx + 1
		for end < len(lines) && !isFenceClose(lines[end]) {
			end++
		}
		if end < len(lines) {
			end++ // include closing fence
		}
		out := append([]string{}, lines[:idx]...)
		out = append(out, replacement...)
		return append(out, lines[end:]...)
	}
	out := append([]string{}, lines[:idx]...)
	out = append(out, replacement...)
	return append(out, lines[idx:]...)
}

func isFenceOpen(line string) bool {
	t := strings.TrimLeft(line, " \t")
	return strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~")
}

func isFenceClose(line string) bool {
	return isFenceOpen(line)
}
