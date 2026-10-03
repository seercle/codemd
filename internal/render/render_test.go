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

func TestSnippetLongerFence(t *testing.T) {
	got := Snippet("console", []string{"$ codemd doc.md", "```go", "x", "```"})
	want := []string{"````console", "$ codemd doc.md", "```go", "x", "```", "````"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v", got)
	}
	// A run of four requires five.
	got = Snippet("", []string{"````"})
	want = []string{"`````", "````", "`````"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v", got)
	}
	// Leading whitespace is allowed before a fence.
	got = Snippet("", []string{"   ```"})
	want = []string{"````", "   ```", "````"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v", got)
	}
}

func TestEscapeLabel(t *testing.T) {
	cases := map[string]string{
		"plain":     "plain",
		"a]b":       `a\]b`,
		"a[b":       `a\[b`,
		"a[0]":      `a\[0\]`,
		`a\b`:       `a\\b`,
		`a\]b`:      `a\\\]b`,
		"f(x) #L9)": "f(x) #L9)",
	}
	for in, want := range cases {
		if got := EscapeLabel(in); got != want {
			t.Fatalf("EscapeLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLinks(t *testing.T) {
	if got := LinkLabel("src/server.go", 13); got != "src/server.go:13" {
		t.Fatalf("label %q", got)
	}
	if got := LinkTarget("src/server.go", 13); got != "src/server.go#L13" {
		t.Fatalf("local target %q", got)
	}
	if got := LinkTarget("https://x/y.go", 13); got != "https://x/y.go#L13" {
		t.Fatalf("remote target %q", got)
	}
}

func TestLinkRanges(t *testing.T) {
	if got := LinkLabelRange("src/server.go", 10, 20); got != "src/server.go:10-20" {
		t.Fatalf("label %q", got)
	}
	if got := LinkTargetRange("src/server.go", 10, 20); got != "src/server.go#L10-L20" {
		t.Fatalf("local target %q", got)
	}
	if got := LinkTargetRange("https://x/y.go", 10, 20); got != "https://x/y.go#L10-L20" {
		t.Fatalf("remote target %q", got)
	}
	if got := LinkLabelRange("src/server.go", 5, 5); got != "src/server.go:5-5" {
		t.Fatalf("equal-range label %q", got)
	}
	if got := LinkTargetRange("src/server.go", 5, 5); got != "src/server.go#L5-L5" {
		t.Fatalf("equal-range target %q", got)
	}
	if got := LinkLabelRange("src/server.go", 20, 10); got != "src/server.go:20-10" {
		t.Fatalf("inverted-range label %q", got)
	}
	if got := LinkTargetRange("src/server.go", 20, 10); got != "src/server.go#L20-L10" {
		t.Fatalf("inverted-range target %q", got)
	}
}
