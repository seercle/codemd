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
		updated, warn := splice(lines.Content, ref.Line, ref.Ref.Mode, replacement)
		lines.Content = updated
		if warn != "" {
			warns = append(warns, Warning{Line: ref.Line, Msg: warn})
		}
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
	for _, b := range []extract.Bound{ref.Ref.Range.Start, ref.Ref.Range.End} {
		if err := checkDuplicate(set, b); err != nil {
			return nil, err
		}
	}
	markers := set.Markers
	if ref.Ref.Mode == mdref.Link {
		start, end := 0, 0
		if ref.Ref.IsRange {
			res, err := extract.Resolve(content, markers, ref.Ref.Range, false)
			if err != nil {
				return nil, err
			}
			start, end = res.StartLine, res.EndLine
		} else {
			line, err := extract.ResolveLink(content, markers, ref.Ref.Range.Start)
			if err != nil {
				return nil, err
			}
			start, end = line, line
		}
		single := start == end
		label := ref.Ref.Label
		if label == "" {
			if single {
				label = render.LinkLabel(ref.Ref.Path, start)
			} else {
				label = render.LinkLabelRange(ref.Ref.Path, start, end)
			}
		}
		label = render.EscapeLabel(label)
		target := render.LinkTarget(ref.Ref.Path, start)
		if !single {
			target = render.LinkTargetRange(ref.Ref.Path, start, end)
		}
		return []string{fmt.Sprintf("[%s](%s)", label, target)}, nil
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

// generatedLink matches a link line codemd previously emitted, either a
// single-line anchor (#Lline) or a range anchor (#Lstart-Lend). A matching
// line directly below a link reference is replaced in place, so a
// user-authored link in that position is adopted as if generated.
var generatedLink = regexp.MustCompile(`^\[(?:\\.|[^\]\\])*\]\([^)]*#L\d+(?:-L\d+)?\)$`)

func isGeneratedLink(line string) bool {
	return generatedLink.MatchString(strings.TrimSpace(line))
}

// splice updates the managed region for the reference on line refLine
// (1-based) and returns a warning when an unmanaged block was preserved. For
// Import the managed region is the generated fence followed by the hidden
// marker; for Link it is the generated link line. It skips blank lines after
// the reference to find the first non-blank line:
//
//   - a marked region (marker after the fence) is replaced in place;
//   - a marker immediately before a fence whose bytes equal the generated
//     fence is normalized in place; a marker before a differing fence keeps
//     that fence, inserts the generated region above it, and warns;
//   - an unmarked fenced block whose bytes equal the generated fence is
//     adopted by appending the marker;
//   - any other unmarked fenced block is left untouched, the marked generated
//     region is inserted above it, and a warning is returned.
//
// When no block is present the region is inserted directly below the
// reference.
func splice(lines []string, refLine int, mode mdref.Mode, replacement []string) ([]string, string) {
	indent := lineutil.LeadingSpace(lines[refLine-1])
	replacement = indentLines(replacement, indent)
	region := replacement
	if mode == mdref.Import {
		region = append(append([]string{}, replacement...), indent+mdref.GeneratedMarker)
	}
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
				return append(out, lines[idx+1:]...), ""
			}
		} else {
			if mdref.IsGeneratedMarker(lines[idx]) {
				k := idx + 1
				for k < len(lines) && strings.TrimSpace(lines[k]) == "" {
					k++
				}
				end, hasFence := 0, false
				if k < len(lines) {
					end, hasFence = mdref.FenceBlockEnd(lines, k)
				}
				if hasFence && !equalLines(lines[k:end], replacement) {
					out := append([]string{}, lines[:idx]...)
					out = append(out, region...)
					out = append(out, lines[idx+1:]...)
					return out, "unmanaged fenced block below reference; inserted generated snippet above it"
				}
				regionEnd := idx + 1
				if hasFence {
					regionEnd = end
				}
				out := append([]string{}, lines[:idx]...)
				out = append(out, region...)
				return append(out, lines[regionEnd:]...), ""
			}
			if end, ok := mdref.FenceBlockEnd(lines, idx); ok {
				j := end
				for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
					j++
				}
				if j < len(lines) && mdref.IsGeneratedMarker(lines[j]) {
					out := append([]string{}, lines[:idx]...)
					out = append(out, region...)
					return append(out, lines[j+1:]...), ""
				}
				if equalLines(lines[idx:end], replacement) {
					out := append([]string{}, lines[:end]...)
					out = append(out, indent+mdref.GeneratedMarker)
					return append(out, lines[end:]...), ""
				}
				out := append([]string{}, lines[:idx]...)
				out = append(out, region...)
				out = append(out, lines[idx:]...)
				return out, "unmanaged fenced block below reference; inserted generated snippet above it"
			}
		}
	}
	out := append([]string{}, lines[:insertAt]...)
	out = append(out, region...)
	return append(out, lines[insertAt:]...), ""
}

// equalLines reports whether a and b have the same length and contents.
func equalLines(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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
