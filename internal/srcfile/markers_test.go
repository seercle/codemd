package srcfile

import (
	"testing"

	"github.com/seercle/codemd/internal/lang"
)

func TestExtractMarkersLineForm(t *testing.T) {
	src := "package x\n\ntype lv int //codemd:lv-def\nfunc f() {}\n//codemd:lv-end\n"
	ms := ExtractMarkers(src, lang.Builtins()["go"]).Markers
	if len(ms) != 2 || ms[0].Name != "lv-def" || ms[0].Line != 3 || ms[1].Name != "lv-end" || ms[1].Line != 5 {
		t.Fatalf("got %+v", ms)
	}
}

func TestExtractMarkersBlockForm(t *testing.T) {
	src := "<div>\n<!-- codemd:a -->\n</div>\n<!--codemd:b-->\n"
	ms := ExtractMarkers(src, lang.Builtins()["html"]).Markers
	if len(ms) != 2 || ms[0].Name != "a" || ms[0].Line != 2 || ms[1].Name != "b" || ms[1].Line != 4 {
		t.Fatalf("got %+v", ms)
	}
}

func TestExtractMarkersBlockFormDualFormLanguage(t *testing.T) {
	src := "int x;\n/*codemd:a*/\n"
	ms := ExtractMarkers(src, lang.Builtins()["c"]).Markers
	if len(ms) != 1 || ms[0].Name != "a" || ms[0].Line != 2 {
		t.Fatalf("got %+v", ms)
	}
}

func TestExtractMarkersRecordsDuplicates(t *testing.T) {
	src := "//codemd:a\n//codemd:a\n//codemd:b\n"
	set := ExtractMarkers(src, lang.Builtins()["go"])
	if len(set.Markers) != 2 || set.Markers[0].Name != "a" || set.Markers[0].Line != 1 || set.Markers[1].Name != "b" || set.Markers[1].Line != 3 {
		t.Fatalf("markers %+v", set.Markers)
	}
	lines := set.Duplicates["a"]
	if len(lines) != 2 || lines[0] != 1 || lines[1] != 2 {
		t.Fatalf("duplicates %+v", set.Duplicates)
	}
}

func TestExtractMarkersMultiGenericForms(t *testing.T) {
	forms := []lang.CommentForm{
		{Line: "//", Block: [2]string{"/*", "*/"}},
		{Line: "#"},
		{Block: [2]string{"<!--", "-->"}},
	}
	src := "#codemd:a\n<!--codemd:b-->\n//codemd:c\n"
	ms := ExtractMarkersMulti(src, forms).Markers
	if len(ms) != 3 || ms[0].Name != "a" || ms[1].Name != "b" || ms[2].Name != "c" {
		t.Fatalf("got %+v", ms)
	}
}

func TestExtractMarkersIgnoresNonMarkers(t *testing.T) {
	src := "// just a comment\n//codemd:\n//codemd:has space\n//codemd:ok\n"
	ms := ExtractMarkers(src, lang.Builtins()["go"]).Markers
	if len(ms) != 1 || ms[0].Name != "ok" {
		t.Fatalf("got %+v", ms)
	}
}
