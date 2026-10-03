package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/seercle/codemd/internal/lang"
	"github.com/seercle/codemd/internal/mdref"
	"github.com/seercle/codemd/internal/srcfile"
)

func resolver() Resolver {
	return Resolver{Loader: srcfile.NewLoader(), Table: lang.Builtins()}
}

func writeSource(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestSpliceInsertsBelowComment(t *testing.T) {
	lines := []string{"<!-- codemd: (link a s.go) -->", "", "keep me"}
	got := splice(lines, 1, mdref.Link, []string{"[s.go:2](s.go#L2)"})
	want := []string{"<!-- codemd: (link a s.go) -->", "[s.go:2](s.go#L2)", "", "keep me"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v", got)
	}
}

func TestSpliceReplacesGeneratedLink(t *testing.T) {
	lines := []string{"<!-- codemd: (link a s.go) -->", "", "[s.go:9](s.go#L9)"}
	got := splice(lines, 1, mdref.Link, []string{"[s.go:2](s.go#L2)"})
	want := []string{"<!-- codemd: (link a s.go) -->", "", "[s.go:2](s.go#L2)"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v", got)
	}
}

func TestSpliceBackToBackComments(t *testing.T) {
	lines := []string{"<!-- codemd: (import a..b s.go go) -->", "<!-- codemd: (link a s.go) -->"}
	got := splice(lines, 1, mdref.Import, []string{"```go", "x", "```"})
	want := []string{"<!-- codemd: (import a..b s.go go) -->", "```go", "x", "```", "<!-- codemd: (link a s.go) -->"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v", got)
	}
}

func TestResolveDocumentImportAndLink(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "server.go"), []byte("package x\n//codemd:a\nfunc f() {}\n//codemd:b\n"), 0o644)
	md := "<!-- codemd: (import a..b server.go go) -->\n\n```go\nstale\n```\n\n<!-- codemd: (link a server.go) -->\n\n[server.go:9](server.go#L9)\n"
	out, errs, _ := resolver().ResolveDocument(md, dir)
	if len(errs) != 0 {
		t.Fatalf("errs %+v", errs)
	}
	if !strings.Contains(out, "```go\nfunc f() {}\n```") {
		t.Fatalf("import not resolved:\n%s", out)
	}
	if !strings.Contains(out, "[server.go:2](server.go#L2)") {
		t.Fatalf("link not resolved:\n%s", out)
	}
	if strings.Contains(out, "stale") || strings.Contains(out, "[server.go:9](server.go#L9)") {
		t.Fatalf("stale content kept:\n%s", out)
	}
}

func TestResolveDocumentLinkInsertsUnderNormalLine(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "server.go"), []byte("package x\n//codemd:a\nfunc f() {}\n"), 0o644)
	md := "<!-- codemd: (link a server.go) -->\n\nkeep me\n"
	out, errs, _ := resolver().ResolveDocument(md, dir)
	if len(errs) != 0 {
		t.Fatalf("errs %+v", errs)
	}
	want := "<!-- codemd: (link a server.go) -->\n[server.go:2](server.go#L2)\n\nkeep me\n"
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestResolveDocumentIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "s.go"), []byte("//codemd:a\nx\n//codemd:b\n"), 0o644)
	md := "<!-- codemd: (import a..b s.go go) -->\n"
	once, _, _ := resolver().ResolveDocument(md, dir)
	twice, _, _ := resolver().ResolveDocument(once, dir)
	if once != twice {
		t.Fatalf("not idempotent:\n%s\n---\n%s", once, twice)
	}
}

func TestResolveDocumentUnknownExtensionFallback(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("#codemd:a\nhello\n#codemd:b\n"), 0o644)
	md := "<!-- codemd: (import a..b notes.txt) -->\n"
	out, errs, _ := resolver().ResolveDocument(md, dir)
	if len(errs) != 0 {
		t.Fatalf("errs %+v", errs)
	}
	if !strings.Contains(out, "```text\nhello\n```") {
		t.Fatalf("fallback not applied:\n%s", out)
	}
}

func TestResolveDocumentContinuesPastBadRef(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "s.go"), []byte("//codemd:a\nx\n//codemd:b\n"), 0o644)
	md := "<!-- codemd: (bogus ref) -->\n<!-- codemd: (import a..b s.go go) -->\n"
	out, errs, _ := resolver().ResolveDocument(md, dir)
	if len(errs) != 1 {
		t.Fatalf("want 1 error, got %+v", errs)
	}
	if !strings.Contains(out, "```go\nx\n```") {
		t.Fatalf("good ref not resolved:\n%s", out)
	}
}

func TestResolveDocumentIdempotentNestedTildeFence(t *testing.T) {
	dir := t.TempDir()
	src := "<!--codemd:a-->\nhello\n~~~\nworld\n<!--codemd:b-->\n"
	os.WriteFile(filepath.Join(dir, "doc.md"), []byte(src), 0o644)
	md := "<!-- codemd: (import a..b doc.md) -->\n"
	once, errs, _ := resolver().ResolveDocument(md, dir)
	if len(errs) != 0 {
		t.Fatalf("errs %+v", errs)
	}
	if !strings.Contains(once, "~~~") {
		t.Fatalf("tilde fence body lost:\n%s", once)
	}
	twice, _, _ := resolver().ResolveDocument(once, dir)
	if once != twice {
		t.Fatalf("not idempotent:\n--- once ---\n%s\n--- twice ---\n%s", once, twice)
	}
}

func TestIsGeneratedLink(t *testing.T) {
	cases := []struct {
		name string
		line string
		want bool
	}{
		{"plain generated link", "[s.go:2](s.go#L2)", true},
		{"custom label with parens", "[f(x) #L9)](s.go#L2)", true},
		{"custom label with spaces", "[My Label](s.go#L2)", true},
		{"escaped bracket label", `[a\]b](s.go#L2)`, true},
		{"surrounding whitespace", "  [s.go:2](s.go#L2)  ", true},
		{"non-link line", "text", false},
		{"reference comment", "<!-- codemd: (link a s.go) -->", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isGeneratedLink(tc.line); got != tc.want {
				t.Fatalf("isGeneratedLink(%q) = %v, want %v", tc.line, got, tc.want)
			}
		})
	}
}

func TestResolveIndentedReferenceIndentsOutput(t *testing.T) {
	dir := writeSource(t, "s.go", "//codemd:a\nx\n//codemd:b\n")
	r := Resolver{Loader: srcfile.NewLoader(), Table: lang.Builtins()}
	in := "- docs:\n\n  <!-- codemd: (import a..b s.go go) -->\n"
	out, errs, _ := r.ResolveDocument(in, dir)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	want := "- docs:\n\n  <!-- codemd: (import a..b s.go go) -->\n  ```go\n  x\n  ```\n"
	if out != want {
		t.Fatalf("got:\n%q\nwant:\n%q", out, want)
	}
}

func TestDuplicateMarkerDoesNotBlockOtherReferences(t *testing.T) {
	dir := writeSource(t, "s.go", "//codemd:a\n//codemd:a\nx\n//codemd:b\ny\n//codemd:c\n")
	r := Resolver{Loader: srcfile.NewLoader(), Table: lang.Builtins()}
	doc := "<!-- codemd: (import b..c s.go go) -->\n"
	out, errs, _ := r.ResolveDocument(doc, dir)
	if len(errs) != 0 {
		t.Fatalf("unrelated reference failed: %v", errs)
	}
	if !strings.Contains(out, "y") {
		t.Fatalf("expected snippet in output, got:\n%s", out)
	}

	bad := "<!-- codemd: (import a..c s.go go) -->\n"
	if _, errs, _ := r.ResolveDocument(bad, dir); len(errs) != 1 {
		t.Fatalf("expected one duplicate-marker error, got: %v", errs)
	}

	badEnd := "<!-- codemd: (import c..a s.go go) -->\n"
	if _, errs, _ := r.ResolveDocument(badEnd, dir); len(errs) != 1 {
		t.Fatalf("expected one duplicate-marker error on the End bound, got: %v", errs)
	}
}

func TestDuplicateMarkerMessageListsEveryLine(t *testing.T) {
	dir := writeSource(t, "s.go", "//codemd:a\n//codemd:a\n//codemd:a\nx\n")
	r := Resolver{Loader: srcfile.NewLoader(), Table: lang.Builtins()}
	doc := "<!-- codemd: (import a.. s.go go) -->\n"
	_, errs, _ := r.ResolveDocument(doc, dir)
	if len(errs) != 1 {
		t.Fatalf("expected one error, got: %v", errs)
	}
	if msg := errs[0].Err.Error(); !strings.Contains(msg, "lines 1, 2, 3") {
		t.Fatalf("message must list every occurrence, got: %q", msg)
	}
}

func TestResolveIndentedReferenceIsIdempotent(t *testing.T) {
	dir := writeSource(t, "s.go", "//codemd:a\nx\n//codemd:b\n")
	r := Resolver{Loader: srcfile.NewLoader(), Table: lang.Builtins()}
	in := "- docs:\n\n  <!-- codemd: (import a..b s.go go) -->\n"
	once, errs, _ := r.ResolveDocument(in, dir)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	twice, errs, _ := r.ResolveDocument(once, dir)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors on re-run: %v", errs)
	}
	if once != twice {
		t.Fatalf("resolve is not idempotent:\nonce:\n%q\ntwice:\n%q", once, twice)
	}
}

func TestResolveDocumentCollectsErrors(t *testing.T) {
	dir := t.TempDir()
	md := "<!-- codemd: (import a..b missing.go go) -->\n"
	out, errs, _ := resolver().ResolveDocument(md, dir)
	if len(errs) != 1 || out != md {
		t.Fatalf("out=%q errs=%+v", out, errs)
	}
}

func TestResolveDocumentReturnsNoWarningsForCleanInput(t *testing.T) {
	dir := writeSource(t, "s.go", "//codemd:a\nx\n//codemd:b\n")
	r := Resolver{Loader: srcfile.NewLoader(), Table: lang.Builtins()}
	_, errs, warns := r.ResolveDocument("<!-- codemd: (import a..b s.go go) -->\n", dir)
	if len(errs) != 0 || len(warns) != 0 {
		t.Fatalf("errs=%v warns=%v", errs, warns)
	}
}
