package mdref

import "testing"

func TestParseImportNamedRange(t *testing.T) {
	r, err := ParseRef("(import a..b src/server.go go)")
	if err != nil {
		t.Fatal(err)
	}
	if r.Mode != Import || r.Range.Start.Name != "a" || r.Range.End.Name != "b" || r.Path != "src/server.go" || r.Lang != "go" || r.Strip {
		t.Fatalf("got %+v", r)
	}
}

func TestParseImportOpenAndRegex(t *testing.T) {
	r, err := ParseRef("(import a.. /src/server.go)")
	if err != nil {
		t.Fatal(err)
	}
	if !r.Range.End.Open || r.Path != "/src/server.go" {
		t.Fatalf("got %+v", r)
	}
	r, err = ParseRef(`(import /func main/../^}/ x.go go strip)`)
	if err != nil {
		t.Fatal(err)
	}
	if r.Range.Start.Regex != "func main" || r.Range.End.Regex != "^}" || !r.Strip || r.Lang != "go" {
		t.Fatalf("got %+v", r)
	}
}

func TestParseLink(t *testing.T) {
	r, err := ParseRef("(link a src/x.go)")
	if err != nil {
		t.Fatal(err)
	}
	if r.Mode != Link || r.Range.Start.Name != "a" || !r.Range.End.Open {
		t.Fatalf("got %+v", r)
	}
}

func TestParseErrors(t *testing.T) {
	bad := []string{
		"(import a src/x.go)",             // import needs range
		"(link a..b src/x.go)",            // link takes one token
		"(import a..b src/x.go go strip)", // strip without regex
		"(nope a..b src/x.go)",            // bad mode
		"(import a..b)",                   // missing path
	}
	for _, s := range bad {
		if _, err := ParseRef(s); err == nil {
			t.Errorf("expected error for %q", s)
		}
	}
}
