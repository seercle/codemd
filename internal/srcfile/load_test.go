package srcfile

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLocal(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "x.go"), []byte("package x\n"), 0o644)
	l := NewLoader()
	got, remote, err := l.Load("x.go", dir)
	if err != nil || remote || got != "package x\n" {
		t.Fatalf("got %q remote=%v err=%v", got, remote, err)
	}
}

func TestLoadHTTPAndCache(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Write([]byte("remote body"))
	}))
	defer srv.Close()
	l := NewLoader()
	got, remote, err := l.Load(srv.URL, "")
	if err != nil || !remote || got != "remote body" {
		t.Fatalf("got %q remote=%v err=%v", got, remote, err)
	}
	l.Load(srv.URL, "")
	if hits != 1 {
		t.Fatalf("expected 1 hit, got %d", hits)
	}
}

func TestLoadHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()
	if _, _, err := NewLoader().Load(srv.URL, ""); err == nil {
		t.Fatal("expected error")
	}
}
