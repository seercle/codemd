package cli

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestIntegrationExternalLink(t *testing.T) {
	const url = "https://raw.githubusercontent.com/golang/go/go1.22.0/src/errors/errors.go"

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		t.Skipf("external host unreachable, skipping: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Skipf("external host returned %d, skipping", resp.StatusCode)
	}

	dir := t.TempDir()
	doc := "[codemd]:# (link /^package errors/ " + url + " go)\n"
	writeTree(t, dir, map[string]string{"doc.md": doc})

	var out, errb bytes.Buffer
	code := Run([]string{"-w", filepath.Join(dir, "doc.md")}, strings.NewReader(""), &out, &errb)
	if code != 0 {
		t.Fatalf("code %d stderr %s", code, errb.String())
	}
	got, err := os.ReadFile(filepath.Join(dir, "doc.md"))
	if err != nil {
		t.Fatal(err)
	}
	want := doc + "[" + url + ":57](" + url + "#L57)\n"
	if string(got) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}
