package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntegrationGolden(t *testing.T) {
	dir := "testdata/integration"
	var out, errb bytes.Buffer
	code := Run([]string{filepath.Join(dir, "doc.md")}, strings.NewReader(""), &out, &errb)
	if code != 0 {
		t.Fatalf("code %d stderr %s", code, errb.String())
	}
	want, err := os.ReadFile(filepath.Join(dir, "want.md"))
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != string(want) {
		t.Fatalf("golden mismatch:\n--- got ---\n%s\n--- want ---\n%s", out.String(), want)
	}
}
