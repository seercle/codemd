package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunStdout(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "s.go"), []byte("//codemd:a\nx\n//codemd:b\n"), 0o644)
	md := filepath.Join(dir, "doc.md")
	os.WriteFile(md, []byte("[codemd]:# (import a..b s.go go)\n"), 0o644)
	var out, errb bytes.Buffer
	code := Run([]string{md}, strings.NewReader(""), &out, &errb)
	if code != 0 {
		t.Fatalf("code %d stderr %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "```go\nx\n```") {
		t.Fatalf("out:\n%s", out.String())
	}
}

func TestRunWriteAndCheck(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "s.go"), []byte("//codemd:a\nx\n//codemd:b\n"), 0o644)
	md := filepath.Join(dir, "doc.md")
	os.WriteFile(md, []byte("[codemd]:# (import a..b s.go go)\n"), 0o644)
	var out, errb bytes.Buffer
	if code := Run([]string{"--check", md}, strings.NewReader(""), &out, &errb); code == 0 {
		t.Fatal("check should fail when out of date")
	}
	if code := Run([]string{"-w", md}, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("write code %d stderr %s", code, errb.String())
	}
	if code := Run([]string{"--check", md}, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatal("check should pass after write")
	}
}

func TestRunErrorExit(t *testing.T) {
	dir := t.TempDir()
	md := filepath.Join(dir, "doc.md")
	os.WriteFile(md, []byte("[codemd]:# (import a..b missing.go go)\n"), 0o644)
	var out, errb bytes.Buffer
	if code := Run([]string{md}, strings.NewReader(""), &out, &errb); code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(errb.String(), "missing.go") {
		t.Fatalf("stderr:\n%s", errb.String())
	}
}

func TestRunCheckStdin(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "s.go")
	os.WriteFile(src, []byte("//codemd:a\nx\n//codemd:b\n"), 0o644)
	md := "[codemd]:# (import a..b " + src + " go)\n"
	var out, errb bytes.Buffer
	if code := Run([]string{"--check"}, strings.NewReader(md), &out, &errb); code == 0 {
		t.Fatal("check should fail when stdin is out of date")
	}
	if out.Len() != 0 {
		t.Fatalf("stdout should be empty, got:\n%s", out.String())
	}

	// Resolve the document first, then check the up-to-date version.
	var resolved bytes.Buffer
	var resErr bytes.Buffer
	if code := Run([]string{}, strings.NewReader(md), &resolved, &resErr); code != 0 {
		t.Fatalf("resolve code %d stderr %s", code, resErr.String())
	}
	out.Reset()
	errb.Reset()
	if code := Run([]string{"--check"}, strings.NewReader(resolved.String()), &out, &errb); code != 0 {
		t.Fatalf("check should pass for up-to-date stdin, stderr:\n%s", errb.String())
	}
}

func TestRunDiscoveredConfigError(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".codemd.yaml"), []byte("languages: [not a map]\n"), 0o644)
	md := filepath.Join(dir, "doc.md")
	os.WriteFile(md, []byte("[codemd]:# (import a..b s.go go)\n"), 0o644)
	var out, errb bytes.Buffer
	if code := Run([]string{md}, strings.NewReader(""), &out, &errb); code == 0 {
		t.Fatal("expected non-zero exit for malformed discovered config")
	}
	if !strings.Contains(errb.String(), ".codemd.yaml") {
		t.Fatalf("stderr should mention config error:\n%s", errb.String())
	}
}

func TestRunLanguages(t *testing.T) {
	var out, errb bytes.Buffer
	if code := Run([]string{"--languages"}, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("code %d stderr %s", code, errb.String())
	}
	if !strings.Contains(out.String(), `go -> go (line "//")`) {
		t.Fatalf("out:\n%s", out.String())
	}
	if !strings.Contains(out.String(), `html -> html (block "<!--" "-->")`) {
		t.Fatalf("out:\n%s", out.String())
	}
}

func TestRunLanguagesWithConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "c.yaml")
	if err := os.WriteFile(cfg, []byte("languages:\n  foo:\n    line: \";;\"\n    fence: foofence\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	if code := Run([]string{"--languages", "--config", cfg}, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("code %d stderr %s", code, errb.String())
	}
	if !strings.Contains(out.String(), `foo -> foofence (line ";;")`) {
		t.Fatalf("out:\n%s", out.String())
	}
}

func TestUnifiedDiffShape(t *testing.T) {
	d := unifiedDiff("doc.md", "a\nb\n", "a\nc\n")
	if !strings.HasPrefix(d, "--- ") {
		t.Fatalf("missing old header:\n%s", d)
	}
	if !strings.Contains(d, "+++ ") {
		t.Fatalf("missing new header:\n%s", d)
	}
	if !strings.Contains(d, "@@ -1,2 +1,2 @@") {
		t.Fatalf("wrong hunk header:\n%s", d)
	}
	minus, plus := 0, 0
	for _, l := range strings.Split(d, "\n") {
		switch {
		case strings.HasPrefix(l, "--- "), strings.HasPrefix(l, "+++ "):
		case strings.HasPrefix(l, "-"):
			minus++
		case strings.HasPrefix(l, "+"):
			plus++
		}
	}
	if minus != 2 || plus != 2 {
		t.Fatalf("expected 2 '-' and 2 '+' lines, got %d and %d:\n%s", minus, plus, d)
	}

	if _, err := exec.LookPath("patch"); err != nil {
		t.Skip("patch not available")
	}
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "doc.md")
	if err := os.WriteFile(oldPath, []byte("a\nb\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("patch", "--dry-run", oldPath)
	cmd.Stdin = strings.NewReader(d)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("patch --dry-run failed: %v\n%s", err, out)
	}
}
