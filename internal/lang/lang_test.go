package lang

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBuiltinsAndResolve(t *testing.T) {
	b := Builtins()
	if b["go"].Form.Line != "//" || b["go"].Fence != "go" {
		t.Fatalf("go entry wrong: %+v", b["go"])
	}
	if b["html"].Form.Block != [2]string{"<!--", "-->"} {
		t.Fatalf("html entry wrong: %+v", b["html"])
	}
	if got := FenceFor("py", b); got != "python" {
		t.Fatalf("FenceFor py = %q", got)
	}
	if got := FenceFor("unknown", b); got != "text" {
		t.Fatalf("FenceFor unknown = %q", got)
	}
}

func TestLoadMergeAndDiscover(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".codemd.yaml")
	os.WriteFile(cfgPath, []byte("languages:\n  foo:\n    line: \"--\"\n    fence: foo\n"), 0o644)
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	table, err := Merge(Builtins(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if table["foo"].Form.Line != "--" {
		t.Fatalf("foo not merged: %+v", table["foo"])
	}
	sub := filepath.Join(dir, "a", "b")
	os.MkdirAll(sub, 0o755)
	found, err := DiscoverConfig(sub)
	if err != nil || found != cfgPath {
		t.Fatalf("DiscoverConfig = %q, %v", found, err)
	}
}

func TestMergeRejectsBadEntry(t *testing.T) {
	_, err := Merge(Builtins(), Config{Languages: map[string]Language{
		"bad": {Fence: "bad"},
	}})
	if err == nil {
		t.Fatal("expected error for entry with neither line nor block")
	}
}

func TestDescribe(t *testing.T) {
	table := map[string]Language{
		"go":   {Fence: "go", Form: CommentForm{Line: "//"}},
		"c":    {Fence: "c", Form: CommentForm{Line: "//", Block: [2]string{"/*", "*/"}}},
		"html": {Fence: "html", Form: CommentForm{Block: [2]string{"<!--", "-->"}}},
	}
	got := Describe(table)
	want := []string{
		`c -> c (line "//" or block "/*" "*/")`,
		`go -> go (line "//")`,
		`html -> html (block "<!--" "-->")`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v", got)
	}
}
