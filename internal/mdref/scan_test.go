package mdref

import "testing"

func TestScanFindsCommentReferences(t *testing.T) {
	md := "# Title\n\n<!-- codemd: (import a..b src/x.go go) -->\n\n```go\nold\n```\n\n<!-- codemd: (link a src/x.go) -->\n[x:3](x#L3)\n"
	refs, errs := Scan(md)
	if len(errs) != 0 {
		t.Fatalf("errs %+v", errs)
	}
	if len(refs) != 2 {
		t.Fatalf("got %d refs", len(refs))
	}
	if refs[0].Line != 3 || refs[0].Ref.Mode != Import || refs[1].Line != 9 || refs[1].Ref.Mode != Link {
		t.Fatalf("got %+v", refs)
	}
}

func TestScanIgnoresReferencesInFences(t *testing.T) {
	md := "```\n<!-- codemd: (import a..b src/x.go go) -->\n```\n"
	refs, errs := Scan(md)
	if len(errs) != 0 {
		t.Fatalf("errs %+v", errs)
	}
	if len(refs) != 0 {
		t.Fatalf("got %+v", refs)
	}
}

func TestScanReportsBadReference(t *testing.T) {
	md := "<!-- codemd: (bogus a..b src/x.go) -->\n"
	refs, errs := Scan(md)
	if len(errs) != 1 || len(refs) != 0 {
		t.Fatalf("refs=%+v errs=%+v", refs, errs)
	}
}

func TestScanContinuesAfterBadReference(t *testing.T) {
	md := "<!-- codemd: (bogus a..b src/x.go) -->\n<!-- codemd: (import a..b src/x.go go) -->\n"
	refs, errs := Scan(md)
	if len(errs) != 1 || len(refs) != 1 || refs[0].Line != 2 {
		t.Fatalf("refs=%+v errs=%+v", refs, errs)
	}
}

func TestFenceBlockEnd(t *testing.T) {
	lines := []string{"```go", "~~~", "code", "```", "after"}
	end, ok := FenceBlockEnd(lines, 0)
	if !ok || end != 4 {
		t.Fatalf("end=%d ok=%v", end, ok)
	}
	if _, ok := FenceBlockEnd([]string{"plain"}, 0); ok {
		t.Fatal("plain line should not be a fence")
	}
}

func TestExtractCommentNewSyntax(t *testing.T) {
	cases := []struct {
		line string
		want string
		ok   bool
	}{
		{"<!-- codemd: (import a..b src.go go) -->", "(import a..b src.go go)", true},
		{"  <!-- codemd: (link a src.go) -->", "(link a src.go)", true},
		{"<!-- codemd: (import a..b src.go go)", "", false},
		{"<!-- codemd: not-parenthesized -->", "", false},
		{"plain text", "", false},
	}
	for _, tc := range cases {
		got, ok := ExtractComment(tc.line)
		if ok != tc.ok || got != tc.want {
			t.Errorf("ExtractComment(%q) = (%q, %v), want (%q, %v)", tc.line, got, ok, tc.want, tc.ok)
		}
	}
}
