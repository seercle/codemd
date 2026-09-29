package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDocsCurrent(t *testing.T) {
	dir := filepath.Join("..", "..", "docs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read docs: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		var out, errBuf bytes.Buffer
		if code := Run([]string{path}, strings.NewReader(""), &out, &errBuf); code != 0 {
			t.Errorf("%s: resolving failed (exit %d): %s", e.Name(), code, strings.TrimSpace(errBuf.String()))
			continue
		}
		if out.String() != string(want) {
			t.Errorf("%s is out of date; regenerate with `codemd -w docs`", e.Name())
		}
	}
}

func TestDocsUseHiddenReferenceSyntax(t *testing.T) {
	var files []string
	dir := filepath.Join("..", "..", "docs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read docs: %v", err)
	}
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".md" {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	files = append(files, filepath.Join("..", "..", "README.md"))
	if err := filepath.Walk(filepath.Join("..", "..", "testdata"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".md" {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		t.Fatalf("walk testdata: %v", err)
	}

	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, "[codemd]:#") {
				t.Errorf("%s:%d: legacy reference syntax; use <!-- codemd: (...) -->", path, i+1)
			}
		}
	}
}

// isConsoleImport reports whether line is a codemd import directive whose
// requested fence language is `console`.
func isConsoleImport(line string) bool {
	line = strings.TrimSpace(line)
	return strings.HasPrefix(line, "<!-- codemd: (import ") && strings.HasSuffix(line, " console) -->")
}

// fence is an opened backtick code fence: its 1-based line and info string.
type fence struct {
	line int
	info string
}

// fenceOpen reports whether line opens a backtick fence, returning its run
// length and info string. Up to three leading spaces are allowed, and the info
// string must not itself contain a backtick.
func fenceOpen(line string) (run int, info string, ok bool) {
	trimmed := strings.TrimLeft(line, " ")
	if len(line)-len(trimmed) > 3 {
		return 0, "", false
	}
	n := 0
	for n < len(trimmed) && trimmed[n] == '`' {
		n++
	}
	if n < 3 {
		return 0, "", false
	}
	rest := trimmed[n:]
	if strings.ContainsRune(rest, '`') {
		return 0, "", false
	}
	return n, strings.TrimSpace(rest), true
}

// fenceClose reports whether line closes a fence whose opening run is run: at
// least run backticks followed only by whitespace.
func fenceClose(line string, run int) bool {
	trimmed := strings.TrimLeft(line, " ")
	if len(line)-len(trimmed) > 3 {
		return false
	}
	n := 0
	for n < len(trimmed) && trimmed[n] == '`' {
		n++
	}
	return n >= run && strings.TrimSpace(trimmed[n:]) == ""
}

// scanFences returns every backtick code fence opened in doc, skipping fence
// contents so embedded fences are not themselves treated as openers.
func scanFences(doc string) []fence {
	lines := strings.Split(doc, "\n")
	var out []fence
	for i := 0; i < len(lines); {
		run, info, ok := fenceOpen(lines[i])
		if !ok {
			i++
			continue
		}
		out = append(out, fence{line: i + 1, info: info})
		i++
		for i < len(lines) && !fenceClose(lines[i], run) {
			i++
		}
		i++
	}
	return out
}

// consoleFenceViolations returns the 1-based line numbers of `console` fences
// in doc whose preceding non-empty line is not a codemd console import. Fences
// tagged console-norun and every non-console fence are ignored.
func consoleFenceViolations(doc string) []int {
	lines := strings.Split(doc, "\n")
	var bad []int
	for _, f := range scanFences(doc) {
		if f.info != "console" {
			continue
		}
		j := f.line - 2
		for j >= 0 && strings.TrimSpace(lines[j]) == "" {
			j--
		}
		if j < 0 || !isConsoleImport(lines[j]) {
			bad = append(bad, f.line)
		}
	}
	return bad
}

// TestConsoleFenceScan checks the fence scanner against the cases the flat
// parser used to miss: four-backtick openers and fences that embed other fence
// lines.
func TestConsoleFenceScan(t *testing.T) {
	imp := "<!-- codemd: (import .. ../testdata/console/x/transcript.console console) -->"
	cases := []struct {
		name string
		doc  string
		want []int
	}{
		{"imported console", imp + "\n```console\n$ x\n```\n", nil},
		{"imported four-backtick with embedded fence", imp + "\n````console\n$ x\n```go\ny\n```\n````\n", nil},
		{"hand-written three", "# hi\n\n```console\n$ x\n```\n", []int{3}},
		{"hand-written four", "````console\n$ x\n```\n````\n", []int{1}},
		{"norun ignored", "```console-norun\n$ x\n```\n", nil},
		{"other language ignored", "```go\nx\n```\n", nil},
		{"checked after norun", "```console-norun\nx\n```\n\n```console\ny\n```\n", []int{5}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := consoleFenceViolations(tc.doc); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("violations = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestConsoleFencesAreImported enforces that console fences in the docs are
// backed by a replayed objective: the previous non-empty line before a
// ```console fence must be a codemd import directive ending in " console)".
// Fences labelled console-norun (and every other language) are unrestricted.
func TestConsoleFencesAreImported(t *testing.T) {
	dir := filepath.Join("..", "..", "docs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read docs: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		for _, line := range consoleFenceViolations(string(data)) {
			t.Errorf(`%s:%d: console fence must be preceded by a codemd import directive ending in " console)"`, e.Name(), line)
		}
	}
}
