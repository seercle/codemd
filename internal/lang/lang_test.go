package lang

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

func TestLoadConfigRejectsUnknownKeys(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"toplevel": "languagez:\n  foo:\n    line: \"//\"\n",
		"entry":    "languages:\n  foo:\n    line: \"//\"\n    fencee: foo\n",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(dir, name+".yaml")
			if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadConfig(path); err == nil {
				t.Fatalf("expected an error for unknown key in %q", body)
			}
		})
	}
}

func TestLoadConfigUnknownKeyMessage(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]struct {
		body string
		want string
	}{
		"toplevel": {
			body: "languagez:\n  foo:\n    line: \"//\"\n",
			want: `: line 1: unknown key "languagez"`,
		},
		"entry": {
			body: "languages:\n  foo:\n    line: \"//\"\n    fencee: foo\n",
			want: `: line 4: unknown key "fencee"`,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(dir, name+".yaml")
			if err := os.WriteFile(path, []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := LoadConfig(path)
			if err == nil {
				t.Fatalf("expected an error for %q", tc.body)
			}
			if !strings.HasSuffix(err.Error(), tc.want) {
				t.Fatalf("error = %q, want suffix %q", err, tc.want)
			}
			if strings.Contains(err.Error(), "struct {") || strings.Contains(err.Error(), "yamlLanguage") {
				t.Fatalf("error leaks Go type details: %q", err)
			}
		})
	}
}

func TestLoadConfigTypeErrorsDoNotLeakGoTypes(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"languages-not-map": "languages: [a]\n",
		"block-not-list":    "languages:\n  coffee:\n    block: \"/*\"\n",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(dir, name+".yaml")
			if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := LoadConfig(path)
			if err == nil {
				t.Fatalf("expected an error for %q", body)
			}
			for _, leak := range []string{"[]string", "yamlLanguage", "map[string", "struct {"} {
				if strings.Contains(err.Error(), leak) {
					t.Fatalf("error leaks Go type %q: %v", leak, err)
				}
			}
		})
	}
}

func TestDualFormConfigEntry(t *testing.T) {
	cfg := Config{Languages: map[string]Language{
		"coffee": {Fence: "coffee", Form: CommentForm{
			Line:  "#",
			Block: [2]string{"/*", "*/"},
		}},
	}}
	table, err := Merge(Builtins(), cfg)
	if err != nil {
		t.Fatalf("both forms must be accepted: %v", err)
	}
	got := table["coffee"].Form
	if got.Line != "#" || got.Block != [2]string{"/*", "*/"} {
		t.Fatalf("dual form not preserved: %+v", got)
	}
	if _, err := Merge(Builtins(), Config{Languages: map[string]Language{
		"bad": {Form: CommentForm{}},
	}}); err == nil {
		t.Fatal("an entry with neither form must still be rejected")
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
