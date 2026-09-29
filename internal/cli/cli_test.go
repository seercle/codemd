package cli

import (
	"bytes"
	"fmt"
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
	os.WriteFile(md, []byte("<!-- codemd: (import a..b s.go go) -->\n"), 0o644)
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
	os.WriteFile(md, []byte("<!-- codemd: (import a..b s.go go) -->\n"), 0o644)
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
	os.WriteFile(md, []byte("<!-- codemd: (import a..b missing.go go) -->\n"), 0o644)
	var out, errb bytes.Buffer
	if code := Run([]string{md}, strings.NewReader(""), &out, &errb); code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(errb.String(), "missing.go") {
		t.Fatalf("stderr:\n%s", errb.String())
	}
}

func TestRunErrorNamesFile(t *testing.T) {
	dir := t.TempDir()
	md := filepath.Join(dir, "doc.md")
	os.WriteFile(md, []byte("<!-- codemd: (import a..b missing.go go) -->\n"), 0o644)
	var out, errb bytes.Buffer
	if code := Run([]string{md}, strings.NewReader(""), &out, &errb); code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(errb.String(), md+": line 1:") {
		t.Fatalf("stderr should name the file:\n%s", errb.String())
	}
}

func TestRunErrorNamesStdin(t *testing.T) {
	md := "<!-- codemd: (import a..b missing.go go) -->\n"
	var out, errb bytes.Buffer
	if code := Run(nil, strings.NewReader(md), &out, &errb); code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(errb.String(), "<stdin>: line 1:") {
		t.Fatalf("stderr should name stdin:\n%s", errb.String())
	}
}

func TestRunCheckStdin(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "s.go")
	os.WriteFile(src, []byte("//codemd:a\nx\n//codemd:b\n"), 0o644)
	md := "<!-- codemd: (import a..b " + src + " go) -->\n"
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
	os.WriteFile(md, []byte("<!-- codemd: (import a..b s.go go) -->\n"), 0o644)
	var out, errb bytes.Buffer
	if code := Run([]string{md}, strings.NewReader(""), &out, &errb); code == 0 {
		t.Fatal("expected non-zero exit for malformed discovered config")
	}
	if !strings.Contains(errb.String(), ".codemd.yaml") {
		t.Fatalf("stderr should mention config error:\n%s", errb.String())
	}
}

func TestRunVersion(t *testing.T) {
	var out, errb bytes.Buffer
	if code := Run([]string{"--version"}, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("code %d stderr %s", code, errb.String())
	}
	if got := strings.TrimSpace(out.String()); got != "codemd "+Version {
		t.Fatalf("got %q", got)
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

func TestRunStdinDiscoversConfig(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		".codemd.yaml": "languages:\n  foo:\n    line: \";;\"\n    fence: foofence\n",
		"s.foo":        ";;codemd:a\nx\n;;codemd:b\n",
	})
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	md := "<!-- codemd: (import a..b s.foo) -->\n"
	var out, errb bytes.Buffer
	if code := Run(nil, strings.NewReader(md), &out, &errb); code != 0 {
		t.Fatalf("code %d stderr %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "```foofence\nx\n```") {
		t.Fatalf("stdin should use discovered config fence:\n%s", out.String())
	}
}

func TestRunHelpExitsZero(t *testing.T) {
	var out, errb bytes.Buffer
	code := Run([]string{"--help"}, strings.NewReader(""), &out, &errb)
	if code != 0 {
		t.Fatalf("help should exit 0, got %d", code)
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Fatalf("help should print usage to stdout, got stdout=%q stderr=%q", out.String(), errb.String())
	}
	if strings.Contains(errb.String(), "Usage:") {
		t.Fatalf("help must not print usage to stderr: %s", errb.String())
	}
}

func TestRunUnknownFlagExitsTwo(t *testing.T) {
	var out, errb bytes.Buffer
	if code := Run([]string{"--bogus"}, strings.NewReader(""), &out, &errb); code != 2 {
		t.Fatalf("unknown flag should exit 2, got %d", code)
	}
}

func TestRunWriteAllOrNothing(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"s.go":   "package x\n//codemd:a\nfunc A() {}\n//codemd:b\n",
		"doc.md": "<!-- codemd: (import a..b s.go go) -->\n<!-- codemd: (import a..zzz s.go go) -->\n",
	})
	docPath := filepath.Join(dir, "doc.md")
	before, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	if code := Run([]string{"-w", docPath}, strings.NewReader(""), &out, &errb); code == 0 {
		t.Fatal("expected non-zero exit")
	}
	after, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("file should be untouched on error:\n%s", after)
	}
	if !strings.Contains(errb.String(), "not written") {
		t.Fatalf("stderr should explain the skip:\n%s", errb.String())
	}
}

func TestRunWriteForce(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"s.go":   "package x\n//codemd:a\nfunc A() {}\n//codemd:b\n",
		"doc.md": "<!-- codemd: (import a..b s.go go) -->\n<!-- codemd: (import a..zzz s.go go) -->\n",
	})
	docPath := filepath.Join(dir, "doc.md")
	var out, errb bytes.Buffer
	if code := Run([]string{"-w", "--force", docPath}, strings.NewReader(""), &out, &errb); code == 0 {
		t.Fatal("expected non-zero exit even with --force")
	}
	after, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "```go\nfunc A() {}\n```") {
		t.Fatalf("--force should write the good region:\n%s", after)
	}
}

func TestRunWriteForceShorthand(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"s.go":   "package x\n//codemd:a\nfunc A() {}\n//codemd:b\n",
		"doc.md": "<!-- codemd: (import a..b s.go go) -->\n<!-- codemd: (import a..zzz s.go go) -->\n",
	})
	docPath := filepath.Join(dir, "doc.md")
	var out, errb bytes.Buffer
	if code := Run([]string{"-w", "-f", docPath}, strings.NewReader(""), &out, &errb); code == 0 {
		t.Fatal("expected non-zero exit even with -f")
	}
	after, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "```go\nfunc A() {}\n```") {
		t.Fatalf("-f should write the good region:\n%s", after)
	}
}

func TestRunOutputAllOrNothing(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"s.go":   "package x\n//codemd:a\nfunc A() {}\n//codemd:b\n",
		"doc.md": "<!-- codemd: (import a..b s.go go) -->\n<!-- codemd: (import a..zzz s.go go) -->\n",
	})
	outPath := filepath.Join(dir, "out.md")
	var out, errb bytes.Buffer
	if code := Run([]string{"-o", outPath, filepath.Join(dir, "doc.md")}, strings.NewReader(""), &out, &errb); code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("output file should not be created on error")
	}
}

func TestUnifiedDiffShape(t *testing.T) {
	d := unifiedDiff("doc.md", "a\nb\n", "a\nc\n")
	want := "--- a/doc.md\n+++ b/doc.md\n@@ -1,2 +1,2 @@\n a\n-b\n+c\n"
	if d != want {
		t.Fatalf("got:\n%q\nwant:\n%q", d, want)
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

func TestUnifiedDiffContext(t *testing.T) {
	old := "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n"
	new := "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\nCHANGED\n"
	d := unifiedDiff("doc.md", old, new)
	want := "--- a/doc.md\n+++ b/doc.md\n@@ -9,4 +9,4 @@\n 9\n 10\n 11\n-12\n+CHANGED\n"
	if d != want {
		t.Fatalf("got:\n%q\nwant:\n%q", d, want)
	}
}

func TestUnifiedDiffMultipleHunks(t *testing.T) {
	var oldB, newB strings.Builder
	for i := 1; i <= 30; i++ {
		fmt.Fprintf(&oldB, "%d\n", i)
		if i == 5 {
			newB.WriteString("5X\n")
		} else if i == 25 {
			newB.WriteString("25Y\n")
		} else {
			fmt.Fprintf(&newB, "%d\n", i)
		}
	}
	d := unifiedDiff("doc.md", oldB.String(), newB.String())
	if strings.Count(d, "@@") != 4 {
		t.Fatalf("expected two hunks (4 @@ markers):\n%s", d)
	}
	if !strings.Contains(d, "-5\n+5X\n") {
		t.Fatalf("first change missing:\n%s", d)
	}
	if !strings.Contains(d, "-25\n+25Y\n") {
		t.Fatalf("second change missing:\n%s", d)
	}
	if strings.Contains(d, " 15\n") {
		t.Fatalf("unchanged middle line should not appear:\n%s", d)
	}
}

func TestUnifiedDiffNoChange(t *testing.T) {
	d := unifiedDiff("doc.md", "a\nb\n", "a\nb\n")
	want := "--- a/doc.md\n+++ b/doc.md\n"
	if d != want {
		t.Fatalf("got:\n%q\nwant:\n%q", d, want)
	}
}

func TestUnifiedDiffNoTrailingNewline(t *testing.T) {
	old := "<!-- codemd: (import a..b s.go go) -->"
	new := "<!-- codemd: (import a..b s.go go) -->\n\n```go\nfunc A() {}\n```"
	d := unifiedDiff("doc.md", old, new)
	want := "--- a/doc.md\n+++ b/doc.md\n@@ -1,1 +1,5 @@\n-<!-- codemd: (import a..b s.go go) -->\n\\ No newline at end of file\n+<!-- codemd: (import a..b s.go go) -->\n+\n+```go\n+func A() {}\n+```\n\\ No newline at end of file\n"
	if d != want {
		t.Fatalf("got:\n%q\nwant:\n%q", d, want)
	}
}

func TestUnifiedDiffNewlineOnlyChange(t *testing.T) {
	got := unifiedDiff("doc.md", "x", "x\n")
	want := "--- a/doc.md\n+++ b/doc.md\n@@ -1,1 +1,1 @@\n-x\n\\ No newline at end of file\n+x\n"
	if got != want {
		t.Fatalf("old no NL, new NL:\ngot:\n%q\nwant:\n%q", got, want)
	}
	got = unifiedDiff("doc.md", "x\n", "x")
	want = "--- a/doc.md\n+++ b/doc.md\n@@ -1,1 +1,1 @@\n-x\n+x\n\\ No newline at end of file\n"
	if got != want {
		t.Fatalf("old NL, new no NL:\ngot:\n%q\nwant:\n%q", got, want)
	}
}

func TestUnifiedDiffNoTrailingNewlineContext(t *testing.T) {
	d := unifiedDiff("doc.md", "a\nb\nc", "a\nX\nc")
	want := "--- a/doc.md\n+++ b/doc.md\n@@ -1,3 +1,3 @@\n a\n-b\n+X\n c\n\\ No newline at end of file\n"
	if d != want {
		t.Fatalf("got:\n%q\nwant:\n%q", d, want)
	}
}

func TestFlagsAfterPositionalArgs(t *testing.T) {
	dir := writeSource(t, "doc.md", "<!-- codemd: (link a src.go) -->\n")
	src := "// a comment\nline\n//codemd:a\n"
	if err := os.WriteFile(filepath.Join(dir, "src.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	code := Run([]string{filepath.Join(dir, "doc.md"), "--check"}, strings.NewReader(""), &out, &errb)
	if code != 1 || strings.Contains(errb.String(), "no such file") {
		t.Fatalf("--check after a path must be parsed as a flag: code=%d stderr=%s", code, errb.String())
	}
	if !strings.Contains(errb.String(), "out of date") {
		t.Fatalf("expected an out-of-date report, got: %s", errb.String())
	}
}

func TestDoubleDashEndsFlagParsing(t *testing.T) {
	var out, errb bytes.Buffer
	code := Run([]string{"--", "--version"}, strings.NewReader(""), &out, &errb)
	if code == 0 {
		t.Fatalf("--version after -- must not be parsed as a flag: stdout=%q", out.String())
	}
	if strings.Contains(out.String(), "codemd "+Version) {
		t.Fatalf("version banner must not be printed: %q", out.String())
	}
	if !strings.Contains(errb.String(), "no such file") {
		t.Fatalf("expected --version to be treated as an input path, got: %s", errb.String())
	}
}

func TestRunCheckSummary(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"s.go": "//codemd:a\nx\n//codemd:b\n",
		"a.md": "<!-- codemd: (import a..b s.go go) -->\n",
	})
	var out, errb bytes.Buffer
	if code := Run([]string{"--check", dir}, strings.NewReader(""), &out, &errb); code == 0 {
		t.Fatal("expected out-of-date")
	}
	if !strings.Contains(errb.String(), "1 file(s) checked, 1 out of date") {
		t.Fatalf("stderr:\n%s", errb.String())
	}
}

func TestRunWriteSummary(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"s.go": "//codemd:a\nx\n//codemd:b\n",
		"a.md": "<!-- codemd: (import a..b s.go go) -->\n",
	})
	var out, errb bytes.Buffer
	if code := Run([]string{"-w", dir}, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("code %d stderr %s", code, errb.String())
	}
	if !strings.Contains(errb.String(), "1 file(s) checked, 1 updated") {
		t.Fatalf("stderr:\n%s", errb.String())
	}
}

func TestRunLinkLabelEscaping(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"s.go":   "package x\n//codemd:a\nfunc A() {}\n",
		"doc.md": "<!-- codemd: (link a s.go go \"a]b\") -->\n",
	})
	docPath := filepath.Join(dir, "doc.md")
	var out, errb bytes.Buffer
	if code := Run([]string{"-w", docPath}, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("code %d stderr %s", code, errb.String())
	}
	got, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatal(err)
	}
	want := "<!-- codemd: (link a s.go go \"a]b\") -->\n[a\\]b](s.go#L2)\n"
	if string(got) != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
	var out2, errb2 bytes.Buffer
	if code := Run([]string{"--check", docPath}, strings.NewReader(""), &out2, &errb2); code != 0 {
		t.Fatalf("re-run not idempotent: %s", errb2.String())
	}
}

func TestRunDiffNoTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"s.go":   "package x\n//codemd:a\nfunc A() {}\n//codemd:b\n",
		"doc.md": "<!-- codemd: (import a..b s.go go) -->",
	})
	var out, errb bytes.Buffer
	if code := Run([]string{"-d", filepath.Join(dir, "doc.md")}, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	got := out.String()
	if strings.Count(got, "\\ No newline at end of file\n") != 2 {
		t.Fatalf("expected two newline markers:\n%s", got)
	}
	if !strings.Contains(got, "-<!-- codemd: (import a..b s.go go) -->\n\\ No newline at end of file\n") {
		t.Fatalf("old last line should be marked:\n%s", got)
	}
	if !strings.HasSuffix(got, "+```\n\\ No newline at end of file\n") {
		t.Fatalf("new last line should be marked:\n%s", got)
	}
}
