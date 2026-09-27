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
