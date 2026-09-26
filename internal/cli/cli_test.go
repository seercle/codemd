package cli

import (
	"bytes"
	"os"
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
