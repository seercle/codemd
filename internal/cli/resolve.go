package cli

import (
	"fmt"
	"path/filepath"
	"regexp"
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
		label := ref.Ref.Label
		if label == "" {
			label = render.LinkLabel(ref.Ref.Path, line)
		}
		return []string{fmt.Sprintf("[%s](%s)", label, render.LinkTarget(ref.Ref.Path, line, isRemote))}, nil
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

var generatedLink = regexp.MustCompile(`^\[[^\]]*\]\([^)]*#L\d+\)$`)

func isGeneratedLink(line string) bool {
	return generatedLink.MatchString(strings.TrimSpace(line))
}

// splice updates the managed region for the reference on line refLine
// (1-based). It skips blank lines after the comment to find the first
// non-blank line. If that line is the corresponding generated artifact (a
// fenced block for import, a generated link for link) it is replaced in place;
// otherwise the replacement is inserted directly below the comment, leaving
// any existing line untouched.
func splice(lines []string, refLine int, mode mdref.Mode, replacement []string) []string {
	insertAt := refLine // 0-based index of the line directly below the comment
	idx := insertAt
	for idx < len(lines) && strings.TrimSpace(lines[idx]) == "" {
		idx++
	}
	if idx < len(lines) {
		if mode == mdref.Link {
			if isGeneratedLink(lines[idx]) {
				out := append([]string{}, lines[:idx]...)
				out = append(out, replacement...)
				return append(out, lines[idx+1:]...)
			}
		} else if end, ok := mdref.FenceBlockEnd(lines, idx); ok {
			out := append([]string{}, lines[:idx]...)
			out = append(out, replacement...)
			return append(out, lines[end:]...)
		}
	}
	out := append([]string{}, lines[:insertAt]...)
	out = append(out, replacement...)
	return append(out, lines[insertAt:]...)
}
