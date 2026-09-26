package srcfile

import (
	"testing"

	"github.com/seercle/codemd/internal/lang"
)

func TestExtractMarkersLineForm(t *testing.T) {
	src := "package x\n\ntype lv int //codemd:lv-def\nfunc f() {}\n//codemd:lv-end\n"
	ms, err := ExtractMarkers(src, lang.Builtins()["go"])
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 2 || ms[0].Name != "lv-def" || ms[0].Line != 3 || ms[1].Name != "lv-end" || ms[1].Line != 5 {
		t.Fatalf("got %+v", ms)
	}
}

func TestExtractMarkersBlockForm(t *testing.T) {
	src := "<div>\n<!-- codemd:a -->\n</div>\n<!--codemd:b-->\n"
	ms, err := ExtractMarkers(src, lang.Builtins()["html"])
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 2 || ms[0].Name != "a" || ms[0].Line != 2 || ms[1].Name != "b" || ms[1].Line != 4 {
		t.Fatalf("got %+v", ms)
	}
}

func TestExtractMarkersBlockFormDualFormLanguage(t *testing.T) {
	src := "int x;\n/*codemd:a*/\n"
	ms, err := ExtractMarkers(src, lang.Builtins()["c"])
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 1 || ms[0].Name != "a" || ms[0].Line != 2 {
		t.Fatalf("got %+v", ms)
	}
}

func TestExtractMarkersDuplicateIsError(t *testing.T) {
	src := "//codemd:a\n//codemd:a\n"
	if _, err := ExtractMarkers(src, lang.Builtins()["go"]); err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestExtractMarkersMultiGenericForms(t *testing.T) {
	forms := []lang.CommentForm{
		{Line: "//", Block: [2]string{"/*", "*/"}},
		{Line: "#"},
		{Block: [2]string{"<!--", "-->"}},
	}
	src := "#codemd:a\n<!--codemd:b-->\n//codemd:c\n"
	ms, err := ExtractMarkersMulti(src, forms)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 3 || ms[0].Name != "a" || ms[1].Name != "b" || ms[2].Name != "c" {
		t.Fatalf("got %+v", ms)
	}
}

func TestExtractMarkersIgnoresNonMarkers(t *testing.T) {
	src := "// just a comment\n//codemd:\n//codemd:has space\n//codemd:ok\n"
	ms, err := ExtractMarkers(src, lang.Builtins()["go"])
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 1 || ms[0].Name != "ok" {
		t.Fatalf("got %+v", ms)
	}
}
