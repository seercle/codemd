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

	// Idempotency: resolving the golden output again must be a no-op.
	tmp, err := os.CreateTemp(dir, "golden-*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(out.String()); err != nil {
		t.Fatal(err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatal(err)
	}
	var out2, errb2 bytes.Buffer
	code2 := Run([]string{tmp.Name()}, strings.NewReader(""), &out2, &errb2)
	if code2 != 0 {
		t.Fatalf("second run code %d stderr %s", code2, errb2.String())
	}
	if out2.String() != out.String() {
		t.Fatalf("not idempotent:\n--- first ---\n%s\n--- second ---\n%s", out.String(), out2.String())
	}
}
