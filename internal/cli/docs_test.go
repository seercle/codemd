package cli

import (
	"bytes"
	"os"
	"path/filepath"
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

// isConsoleImport reports whether line is a codemd import directive whose
// requested fence language is `console`.
func isConsoleImport(line string) bool {
	line = strings.TrimSpace(line)
	return strings.HasPrefix(line, "[codemd]:# (import ") && strings.HasSuffix(line, " console)")
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
		lines := strings.Split(string(data), "\n")
		inFence := false
		for i, raw := range lines {
			trimmed := strings.TrimSpace(raw)
			if !strings.HasPrefix(trimmed, "```") {
				continue
			}
			info := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
			if inFence {
				if info == "" {
					inFence = false
				}
				continue
			}
			inFence = true
			if info != "console" {
				continue
			}
			j := i - 1
			for j >= 0 && strings.TrimSpace(lines[j]) == "" {
				j--
			}
			if j < 0 || !isConsoleImport(lines[j]) {
				t.Errorf(`%s:%d: console fence must be preceded by a codemd import directive ending in " console)"`, e.Name(), i+1)
			}
		}
	}
}
