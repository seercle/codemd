package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seercle/codemd/internal/lang"
	"github.com/seercle/codemd/internal/srcfile"
)

func resolver() Resolver {
	return Resolver{Loader: srcfile.NewLoader(), Table: lang.Builtins()}
}

func TestResolveDocumentImportAndLink(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "server.go"), []byte("package x\n//codemd:a\nfunc f() {}\n//codemd:b\n"), 0o644)
	md := "[codemd]:# (import a..b server.go go)\n\n```go\nstale\n```\n\n[codemd]:# (link a server.go)\n\nold link\n"
	out, errs := resolver().ResolveDocument(md, dir)
	if len(errs) != 0 {
		t.Fatalf("errs %+v", errs)
	}
	if !strings.Contains(out, "```go\nfunc f() {}\n```") {
		t.Fatalf("import not resolved:\n%s", out)
	}
	if !strings.Contains(out, "[server.go:2](server.go#L2)") {
		t.Fatalf("link not resolved:\n%s", out)
	}
	if strings.Contains(out, "stale") || strings.Contains(out, "old link") {
		t.Fatalf("stale content kept:\n%s", out)
	}
}

func TestResolveDocumentIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "s.go"), []byte("//codemd:a\nx\n//codemd:b\n"), 0o644)
	md := "[codemd]:# (import a..b s.go go)\n"
	once, _ := resolver().ResolveDocument(md, dir)
	twice, _ := resolver().ResolveDocument(once, dir)
	if once != twice {
		t.Fatalf("not idempotent:\n%s\n---\n%s", once, twice)
	}
}

func TestResolveDocumentUnknownExtensionFallback(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("#codemd:a\nhello\n#codemd:b\n"), 0o644)
	md := "[codemd]:# (import a..b notes.txt)\n"
	out, errs := resolver().ResolveDocument(md, dir)
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
	md := "[codemd]:# (bogus ref)\n[codemd]:# (import a..b s.go go)\n"
	out, errs := resolver().ResolveDocument(md, dir)
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
	md := "[codemd]:# (import a..b doc.md)\n"
	once, errs := resolver().ResolveDocument(md, dir)
	if len(errs) != 0 {
		t.Fatalf("errs %+v", errs)
	}
	if !strings.Contains(once, "~~~") {
		t.Fatalf("tilde fence body lost:\n%s", once)
	}
	twice, _ := resolver().ResolveDocument(once, dir)
	if once != twice {
		t.Fatalf("not idempotent:\n--- once ---\n%s\n--- twice ---\n%s", once, twice)
	}
}

func TestResolveDocumentCollectsErrors(t *testing.T) {
	dir := t.TempDir()
	md := "[codemd]:# (import a..b missing.go go)\n"
	out, errs := resolver().ResolveDocument(md, dir)
	if len(errs) != 1 || out != md {
		t.Fatalf("out=%q errs=%+v", out, errs)
	}
}
