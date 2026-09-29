package extract

import (
	"testing"

	"github.com/seercle/codemd/internal/lang"
	"github.com/seercle/codemd/internal/srcfile"
)

const goSrc = "package x\n\ntype lv int //codemd:a\nfunc f() {}\nfunc g() {}\n//codemd:b\n"

func markers(t *testing.T) []srcfile.Marker {
	t.Helper()
	return srcfile.ExtractMarkers(goSrc, mustGo()).Markers
}

func mustGo() lang.Language { return lang.Builtins()["go"] }

func TestResolveNamedRange(t *testing.T) {
	res, err := Resolve(goSrc, markers(t), Range{Start: Bound{Name: "a"}, End: Bound{Name: "b"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.StartLine != 4 || res.EndLine != 5 {
		t.Fatalf("bounds %d-%d", res.StartLine, res.EndLine)
	}
	if len(res.Lines) != 2 || res.Lines[0] != "func f() {}" || res.Lines[1] != "func g() {}" {
		t.Fatalf("lines %+v", res.Lines)
	}
}

func TestResolveOpenRanges(t *testing.T) {
	res, err := Resolve(goSrc, markers(t), Range{Start: Bound{Name: "a"}, End: Bound{Open: true}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.StartLine != 4 || res.EndLine != 6 || len(res.Lines) != 3 {
		t.Fatalf("got %+v", res)
	}
	res2, err := Resolve(goSrc, markers(t), Range{Start: Bound{Open: true}, End: Bound{Name: "a"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if res2.StartLine != 1 || res2.EndLine != 2 || len(res2.Lines) != 2 {
		t.Fatalf("got %+v", res2)
	}
}

func TestResolveRegexRange(t *testing.T) {
	res, err := Resolve(goSrc, markers(t), Range{Start: Bound{Regex: `func f`}, End: Bound{Regex: `func g`}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.StartLine != 4 || res.EndLine != 5 {
		t.Fatalf("bounds %d-%d", res.StartLine, res.EndLine)
	}
}

func TestResolveStrip(t *testing.T) {
	src := "// codemd:a\ncode\n// codemd:b\n"
	res, err := Resolve(src, nil, Range{Start: Bound{Regex: `^// codemd:a$`}, End: Bound{Regex: `^// codemd:b$`}}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Lines) != 1 || res.Lines[0] != "code" {
		t.Fatalf("lines %+v", res.Lines)
	}
}

func TestResolveLink(t *testing.T) {
	line, err := ResolveLink(goSrc, markers(t), Bound{Name: "a"})
	if err != nil || line != 3 {
		t.Fatalf("link = %d, %v", line, err)
	}
	line, err = ResolveLink(goSrc, markers(t), Bound{Regex: `func g`})
	if err != nil || line != 5 {
		t.Fatalf("link regex = %d, %v", line, err)
	}
}

func TestResolveUnmatchedIsError(t *testing.T) {
	if _, err := Resolve(goSrc, markers(t), Range{Start: Bound{Name: "nope"}, End: Bound{Name: "b"}}, false); err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveEndAtOrAfterStart(t *testing.T) {
	src := "func g() {}\nfunc f() {}\nfunc g() {}\n"
	res, err := Resolve(src, nil, Range{Start: Bound{Regex: "func f"}, End: Bound{Regex: "func g"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.StartLine != 2 || res.EndLine != 3 {
		t.Fatalf("bounds %d-%d", res.StartLine, res.EndLine)
	}
	if len(res.Lines) != 2 || res.Lines[0] != "func f() {}" || res.Lines[1] != "func g() {}" {
		t.Fatalf("lines %+v", res.Lines)
	}
}

func TestResolveEndBeforeStartIsError(t *testing.T) {
	src := "func g() {}\nfunc f() {}\n"
	if _, err := Resolve(src, nil, Range{Start: Bound{Regex: "func f"}, End: Bound{Regex: "func g"}}, false); err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveEmptyRangeIsError(t *testing.T) {
	src := "//codemd:a\n//codemd:b\nx\n"
	ms := srcfile.ExtractMarkers(src, mustGo()).Markers
	_, err := Resolve(src, ms, Range{Start: Bound{Name: "a"}, End: Bound{Name: "b"}}, false)
	if err == nil {
		t.Fatal("expected an error for a range with no lines between the markers")
	}
	_, err = Resolve(src, ms, Range{Start: Bound{Name: "a"}, End: Bound{Name: "a"}}, false)
	if err == nil {
		t.Fatal("expected an error for a self range")
	}
}

func TestResolveEmptyOpenRangeIsError(t *testing.T) {
	_, err := Resolve(goSrc, markers(t), Range{Start: Bound{Name: "b"}, End: Bound{Open: true}}, false)
	if err == nil {
		t.Fatal("expected an error for an open range with no lines after the final marker")
	}
}

func TestResolveStripLeftmostSingleMatch(t *testing.T) {
	src := "// a a\ncode\n"
	res, err := Resolve(src, nil, Range{Start: Bound{Regex: "a"}, End: Bound{Open: true}}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Lines) != 2 || res.Lines[0] != "//  a" || res.Lines[1] != "code" {
		t.Fatalf("lines %+v", res.Lines)
	}
}
