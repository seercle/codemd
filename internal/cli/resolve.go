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

// Resolver rewrites codemd references in a Markdown document: it loads each
// referenced source with Loader and interprets file extensions using Table.
type Resolver struct {
	Loader *srcfile.Loader
	Table  map[string]lang.Language
}

// RefError pairs a reference error with the 1-based line it occurred on. Line
// is 0 when the error is not tied to a specific line.
type RefError struct {
	Line int
	Err  error
}

// Warning is a non-fatal condition encountered while resolving, such as an
// unmanaged fenced block left in place below an import reference.
type Warning struct {
	Line int
	Msg  string
}

// ResolveDocument resolves every codemd reference in the Markdown content,
// reading referenced files relative to baseDir. It returns the rewritten
// document, the reference errors encountered, and any non-fatal warnings,
// each ordered by line. References that fail are left unchanged so the rest of
// the document can still be processed.
func (r Resolver) ResolveDocument(content, baseDir string) (string, []RefError, []Warning) {
	lines := lineutil.Split(content)
	refs, scanErrs := mdref.Scan(content)
	var errs []RefError
	for _, se := range scanErrs {
		errs = append(errs, RefError{Line: se.Line, Err: se.Err})
	}
	var warns []Warning
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
	sort.SliceStable(warns, func(i, j int) bool { return warns[i].Line < warns[j].Line })
	return lines.Join(), errs, warns
}

func (r Resolver) resolveOne(ref mdref.Reference, baseDir string) ([]string, error) {
	content, _, err := r.Loader.Load(ref.Ref.Path, baseDir)
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
	set := srcfile.ExtractMarkersMulti(content, forms)
	if err := checkDuplicate(set, ref.Ref.Range.Start); err != nil {
		return nil, err
	}
	if err := checkDuplicate(set, ref.Ref.Range.End); err != nil {
		return nil, err
	}
	markers := set.Markers
	if ref.Ref.Mode == mdref.Link {
		line, err := extract.ResolveLink(content, markers, ref.Ref.Range.Start)
		if err != nil {
			return nil, err
		}
		label := ref.Ref.Label
		if label == "" {
			label = render.LinkLabel(ref.Ref.Path, line)
		}
		label = render.EscapeLabel(label)
		return []string{fmt.Sprintf("[%s](%s)", label, render.LinkTarget(ref.Ref.Path, line))}, nil
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

func checkDuplicate(set srcfile.MarkerSet, b extract.Bound) error {
	if b.Name == "" {
		return nil
	}
	lines := set.Duplicates[b.Name]
	if len(lines) < 2 {
		return nil
	}
	if len(lines) == 2 {
		return fmt.Errorf("duplicate marker %q on lines %d and %d", b.Name, lines[0], lines[1])
	}
	parts := make([]string, len(lines))
	for i, n := range lines {
		parts[i] = fmt.Sprint(n)
	}
	return fmt.Errorf("duplicate marker %q on lines %s", b.Name, strings.Join(parts, ", "))
}

var generatedLink = regexp.MustCompile(`^\[(?:\\.|[^\]\\])*\]\([^)]*#L\d+\)$`)

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
	indent := leadingIndent(lines[refLine-1])
	replacement = indentLines(replacement, indent)
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

func leadingIndent(line string) string {
	trimmed := strings.TrimLeft(line, " \t")
	return line[:len(line)-len(trimmed)]
}

func indentLines(lines []string, indent string) []string {
	if indent == "" {
		return lines
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		if l == "" {
			out[i] = l
			continue
		}
		out[i] = indent + l
	}
	return out
}
