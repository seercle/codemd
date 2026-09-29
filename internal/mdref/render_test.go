package mdref

import (
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

// TestReferenceLinesAreHiddenByMarkdown locks the property that a reference
// line, whatever its contents, produces no visible output when rendered.
func TestReferenceLinesAreHiddenByMarkdown(t *testing.T) {
	refs := []string{
		"<!-- codemd: (import handler-start..handler-end server.go go) -->",
		`<!-- codemd: (import /^func (Foo)\(/../^}/ server.go go) -->`,
		`<!-- codemd: (link handler-start server.go go "the (handler) entry") -->`,
		`<!-- codemd: (link a src.go go "a) b") -->`,
		`<!-- codemd: (import a..b src.go go "label] with ] brackets") -->`,
	}
	for _, ref := range refs {
		var out strings.Builder
		if err := goldmark.Convert([]byte(ref+"\n"), &out); err != nil {
			t.Fatalf("%s: %v", ref, err)
		}
		if strings.Contains(out.String(), "codemd") || strings.Contains(out.String(), "import") {
			t.Errorf("reference rendered visibly:\nref:  %s\nhtml: %s", ref, out.String())
		}
	}
}

// TestIndentedSnippetStaysInListItem locks the property that an indented
// reference's managed fence is indented with it, so its code block renders
// inside the list item rather than breaking out of it.
func TestIndentedSnippetStaysInListItem(t *testing.T) {
	doc := "- docs:\n\n  <!-- codemd: (import a..b s.go go) -->\n  ```go\n  x\n  ```\n"
	var out strings.Builder
	if err := goldmark.Convert([]byte(doc), &out); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	li := strings.Index(html, "<li>")
	code := strings.Index(html, "<code")
	closeLi := strings.Index(html, "</li>")
	if li < 0 || code < 0 || closeLi < 0 || !(li < code && code < closeLi) {
		t.Fatalf("code block is not inside the list item:\n%s", html)
	}
}

// TestLegacyReferenceRendersVisibly documents the bug the HTML-comment syntax
// fixes: the old parenthesized-title form is only hidden while it contains no
// unescaped parentheses.
func TestLegacyReferenceRendersVisibly(t *testing.T) {
	ref := "[codemd]:# (import /^func (Foo)/../^}/ server.go go)\n"
	var out strings.Builder
	if err := goldmark.Convert([]byte(ref), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "codemd") {
		t.Fatalf("expected the legacy form to render visibly, got: %s", out.String())
	}
}
