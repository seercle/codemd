package render

import (
	"reflect"
	"testing"
)

func TestSnippet(t *testing.T) {
	got := Snippet("go", []string{"a", "b"})
	want := []string{"```go", "a", "b", "```"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v", got)
	}
	if got := Snippet("", nil); !reflect.DeepEqual(got, []string{"```", "```"}) {
		t.Fatalf("empty got %#v", got)
	}
}

func TestLinks(t *testing.T) {
	if got := LinkLabel("src/server.go", 13); got != "src/server.go:13" {
		t.Fatalf("label %q", got)
	}
	if got := LinkTarget("src/server.go", 13, false); got != "src/server.go#L13" {
		t.Fatalf("local target %q", got)
	}
	if got := LinkTarget("https://x/y.go", 13, true); got != "https://x/y.go#L13" {
		t.Fatalf("remote target %q", got)
	}
}
