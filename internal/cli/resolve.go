package cli

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/seercle/codemd/internal/extract"
	"github.com/seercle/codemd/internal/lang"
	"github.com/seercle/codemd/internal/lineutil"
	"github.com/seercle/codemd/internal/mdref"
	"github.com/seercle/codemd/internal/render"
	"github.com/seercle/codemd/internal/srcfile"
)

var genericForms = []lang.CommentForm{
	{Line: "//", Block: [2]string{"/*", "*/"}},
	{Line: "#"},
	{Block: [2]string{"<!--", "-->"}},
}

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
	refs, scanErrs := mdref.Scan(content)
	var errs []RefError
	for _, se := range scanErrs {
		errs = append(errs, RefError{Line: se.Line, Err: se.Err})
	}
	// Apply from the bottom up so earlier line numbers stay valid.
	for i := len(refs) - 1; i >= 0; i-- {
		ref := refs[i]
		replacement, err := r.resolveOne(ref, baseDir)
		if err != nil {
			errs = append(errs, RefError{Line: ref.Line, Err: err})
			continue
		}
		lines.Content = splice(lines.Content, ref.Line, ref.Ref.Mode, replacement)
	}
	sort.SliceStable(errs, func(i, j int) bool { return errs[i].Line < errs[j].Line })
	return lines.Join(), errs
}

func (r Resolver) resolveOne(ref mdref.Reference, baseDir string) ([]string, error) {
	content, isRemote, err := r.Loader.Load(ref.Ref.Path, baseDir)
	if err != nil {
		return nil, err
	}
	ext := filepath.Ext(ref.Ref.Path)
	l, ok := lang.Resolve(ext, r.Table)
	var forms []lang.CommentForm
	if ok {
		forms = []lang.CommentForm{l.Form}
	} else {
		forms = genericForms
	}
	markers, err := srcfile.ExtractMarkersMulti(content, forms)
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
	if end, ok := mdref.FenceBlockEnd(lines, idx); ok {
		out := append([]string{}, lines[:idx]...)
		out = append(out, replacement...)
		return append(out, lines[end:]...)
	}
	out := append([]string{}, lines[:idx]...)
	out = append(out, replacement...)
	return append(out, lines[idx:]...)
}
