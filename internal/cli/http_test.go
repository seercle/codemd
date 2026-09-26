package cli

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntegrationHTTP(t *testing.T) {
	const src = "package x\n//codemd:a\nfunc A() {}\n//codemd:b\n"
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		io.WriteString(w, src)
	}))
	defer srv.Close()

	dir := t.TempDir()
	url := srv.URL + "/s.go"
	doc := "[codemd]:# (import a..b " + url + " go)\n\n[codemd]:# (link a " + url + ")\n"
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
	want := "[codemd]:# (import a..b " + url + " go)\n```go\nfunc A() {}\n```\n\n[codemd]:# (link a " + url + ")\n[" + url + ":2](" + url + "#L2)\n"
	if string(got) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	if hits != 1 {
		t.Fatalf("expected the URL to be fetched once (cache), got %d hits", hits)
	}
}

func TestIntegrationHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	dir := t.TempDir()
	url := srv.URL + "/missing.go"
	doc := "[codemd]:# (import a..b " + url + " go)\n"
	writeTree(t, dir, map[string]string{"doc.md": doc})

	var out, errb bytes.Buffer
	code := Run([]string{"-w", filepath.Join(dir, "doc.md")}, strings.NewReader(""), &out, &errb)
	if code == 0 {
		t.Fatal("expected non-zero exit for HTTP 404")
	}
	got, err := os.ReadFile(filepath.Join(dir, "doc.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != doc {
		t.Fatalf("region should be untouched on error:\n%s", got)
	}
}
