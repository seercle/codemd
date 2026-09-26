package mdref

import "testing"

func TestScanFindsReferences(t *testing.T) {
	md := "# Title\n\n[codemd]:# (import a..b src/x.go go)\n\n```go\nold\n```\n\n[codemd]:# (link a src/x.go)\n[x:3](x#L3)\n"
	refs, err := Scan(md)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 2 {
		t.Fatalf("got %d refs", len(refs))
	}
	if refs[0].Line != 3 || refs[0].Ref.Mode != Import || refs[1].Line != 9 || refs[1].Ref.Mode != Link {
		t.Fatalf("got %+v", refs)
	}
}

func TestScanIgnoresReferencesInFences(t *testing.T) {
	md := "```\n[codemd]:# (import a..b src/x.go go)\n```\n"
	refs, err := Scan(md)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 0 {
		t.Fatalf("got %+v", refs)
	}
}

func TestScanReportsBadReference(t *testing.T) {
	md := "[codemd]:# (bogus a..b src/x.go)\n"
	if _, err := Scan(md); err == nil {
		t.Fatal("expected parse error")
	}
}
