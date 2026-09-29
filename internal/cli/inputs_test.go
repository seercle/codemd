package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestExpandInputsDirectory(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"a.md":           "a\n",
		"sub/b.markdown": "b\n",
		"sub/c.txt":      "c\n",
		"sub/s.go":       "package x\n",
		".hidden/d.md":   "d\n",
	})
	got, err := expandInputs([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		filepath.Join(dir, "a.md"),
		filepath.Join(dir, "sub", "b.markdown"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestExpandInputsExplicitFileKept(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{"notes.txt": "x\n"})
	p := filepath.Join(dir, "notes.txt")
	got, err := expandInputs([]string{p})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{p}) {
		t.Fatalf("got %#v", got)
	}
}

func TestExpandInputsDedupes(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{"a.md": "a\n"})
	p := filepath.Join(dir, "a.md")
	got, err := expandInputs([]string{p, p, dir})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{p}) {
		t.Fatalf("got %#v", got)
	}
}

func TestExpandInputsGlobDoubleStar(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"docs/a.md":    "a\n",
		"docs/x/b.md":  "b\n",
		"docs/x/c.txt": "c\n",
	})
	got, err := expandInputs([]string{filepath.Join(dir, "docs", "**", "*.md")})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		filepath.Join(dir, "docs", "a.md"),
		filepath.Join(dir, "docs", "x", "b.md"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestExpandInputsDotSlashGlob(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"a.md":      "a\n",
		"sub/b.md":  "b\n",
		"sub/c.txt": "c\n",
	})
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	got, err := expandInputs([]string{"./**/*.md"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a.md", filepath.Join("sub", "b.md")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestExpandInputsNoMatch(t *testing.T) {
	dir := t.TempDir()
	if _, err := expandInputs([]string{filepath.Join(dir, "*.md")}); err == nil {
		t.Fatal("expected error for a pattern matching nothing")
	}
	if _, err := expandInputs([]string{filepath.Join(dir, "missing.md")}); err == nil {
		t.Fatal("expected error for a missing file")
	}
	if _, err := os.Stat(filepath.Join(dir, "missing.md")); err == nil {
		t.Fatal("precondition: file must not exist")
	}
}

func TestRunWriteDirectory(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"s.go":            "//codemd:a\nx\n//codemd:b\n",
		"docs/a.md":       "<!-- codemd: (import a..b ../s.go go) -->\n",
		"docs/sub/b.md":   "<!-- codemd: (import a..b ../../s.go go) -->\n",
		"docs/ignore.txt": "no refs\n",
	})
	var out, errb bytes.Buffer
	if code := Run([]string{"-w", filepath.Join(dir, "docs")}, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("code %d stderr %s", code, errb.String())
	}
	for _, name := range []string{"a.md", filepath.Join("sub", "b.md")} {
		got, err := os.ReadFile(filepath.Join(dir, "docs", name))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(got), "```go\nx\n```") {
			t.Fatalf("%s not resolved:\n%s", name, got)
		}
	}
}
