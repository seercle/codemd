package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTree(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

type matrixCase struct {
	name    string
	files   map[string]string // extra files written into the temp dir
	doc     string            // doc.md content
	want    string            // expected doc.md after -w
	wantErr bool
}

func runMatrixCase(t *testing.T, tc matrixCase) {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{}
	for k, v := range tc.files {
		files[k] = v
	}
	files["doc.md"] = tc.doc
	writeTree(t, dir, files)
	var out, errb bytes.Buffer
	code := Run([]string{"-w", filepath.Join(dir, "doc.md")}, strings.NewReader(""), &out, &errb)
	if tc.wantErr {
		if code == 0 {
			t.Fatalf("expected non-zero exit; stderr=%s", errb.String())
		}
	} else if code != 0 {
		t.Fatalf("code %d stderr %s", code, errb.String())
	}
	got, err := os.ReadFile(filepath.Join(dir, "doc.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != tc.want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, tc.want)
	}
}

const rangeSrc = "package x\n\n//codemd:a\nfunc A() {\n\treturn\n}\n\n//codemd:b\nfunc B() {}\n"

func TestIntegrationMatrixRanges(t *testing.T) {
	cases := []matrixCase{
		{
			name:  "named",
			files: map[string]string{"s.go": rangeSrc},
			doc:   "[codemd]:# (import a..b s.go go)\n",
			want:  "[codemd]:# (import a..b s.go go)\n```go\nfunc A() {\n\treturn\n}\n\n```\n",
		},
		{
			name:  "regex",
			files: map[string]string{"s.go": rangeSrc},
			doc:   "[codemd]:# (import /func A/../^}/ s.go go)\n",
			want:  "[codemd]:# (import /func A/../^}/ s.go go)\n```go\nfunc A() {\n\treturn\n}\n```\n",
		},
		{
			name:  "mixed",
			files: map[string]string{"s.go": rangeSrc},
			doc:   "[codemd]:# (import a../^}/ s.go go)\n",
			want:  "[codemd]:# (import a../^}/ s.go go)\n```go\nfunc A() {\n\treturn\n}\n```\n",
		},
		{
			name:  "open end",
			files: map[string]string{"s.go": rangeSrc},
			doc:   "[codemd]:# (import a.. s.go go)\n",
			want:  "[codemd]:# (import a.. s.go go)\n```go\nfunc A() {\n\treturn\n}\n\n//codemd:b\nfunc B() {}\n```\n",
		},
		{
			name:  "open start",
			files: map[string]string{"s.go": rangeSrc},
			doc:   "[codemd]:# (import ..b s.go go)\n",
			want:  "[codemd]:# (import ..b s.go go)\n```go\npackage x\n\n//codemd:a\nfunc A() {\n\treturn\n}\n\n```\n",
		},
		{
			name:  "strip",
			files: map[string]string{"s.txt": "AAA\ncode\nBBB\n"},
			doc:   "[codemd]:# (import /AAA/../BBB/ s.txt text strip)\n",
			want:  "[codemd]:# (import /AAA/../BBB/ s.txt text strip)\n```text\ncode\n```\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { runMatrixCase(t, tc) })
	}
}

func TestIntegrationMatrixLanguages(t *testing.T) {
	cases := []matrixCase{
		{
			name:  "go",
			files: map[string]string{"s.go": "package x\n//codemd:a\nfunc A() {}\n//codemd:b\n"},
			doc:   "[codemd]:# (import a..b s.go)\n",
			want:  "[codemd]:# (import a..b s.go)\n```go\nfunc A() {}\n```\n",
		},
		{
			name:  "python",
			files: map[string]string{"s.py": "x = 1\n#codemd:a\ny = 2\n#codemd:b\n"},
			doc:   "[codemd]:# (import a..b s.py)\n",
			want:  "[codemd]:# (import a..b s.py)\n```python\ny = 2\n```\n",
		},
		{
			name:  "html",
			files: map[string]string{"s.html": "<div>\n<!--codemd:a-->\n<p>hi</p>\n<!--codemd:b-->\n"},
			doc:   "[codemd]:# (import a..b s.html)\n",
			want:  "[codemd]:# (import a..b s.html)\n```html\n<p>hi</p>\n```\n",
		},
		{
			name:  "c dual form block marker",
			files: map[string]string{"s.c": "int x;\n/*codemd:a*/\nint y;\n/*codemd:b*/\n"},
			doc:   "[codemd]:# (import a..b s.c)\n",
			want:  "[codemd]:# (import a..b s.c)\n```c\nint y;\n```\n",
		},
		{
			name:  "unknown extension fallback",
			files: map[string]string{"s.txt": "#codemd:a\nhello\n#codemd:b\n"},
			doc:   "[codemd]:# (import a..b s.txt)\n",
			want:  "[codemd]:# (import a..b s.txt)\n```text\nhello\n```\n",
		},
		{
			name:  "explicit lang token overrides extension",
			files: map[string]string{"s.go": "package x\n//codemd:a\nfunc A() {}\n//codemd:b\n"},
			doc:   "[codemd]:# (import a..b s.go rust)\n",
			want:  "[codemd]:# (import a..b s.go rust)\n```rust\nfunc A() {}\n```\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { runMatrixCase(t, tc) })
	}
}

func TestIntegrationMatrixRegions(t *testing.T) {
	src := map[string]string{"s.go": "package x\n//codemd:a\nfunc A() {}\n//codemd:b\n"}
	cases := []matrixCase{
		{
			name:  "insert below comment when absent",
			files: src,
			doc:   "[codemd]:# (import a..b s.go go)\n",
			want:  "[codemd]:# (import a..b s.go go)\n```go\nfunc A() {}\n```\n",
		},
		{
			name:  "replace existing fence",
			files: src,
			doc:   "[codemd]:# (import a..b s.go go)\n\n```go\nstale\n```\n",
			want:  "[codemd]:# (import a..b s.go go)\n\n```go\nfunc A() {}\n```\n",
		},
		{
			name:  "insert below comment before normal line",
			files: src,
			doc:   "[codemd]:# (import a..b s.go go)\n\ntext\n",
			want:  "[codemd]:# (import a..b s.go go)\n```go\nfunc A() {}\n```\n\ntext\n",
		},
		{
			name:  "back to back references",
			files: src,
			doc:   "[codemd]:# (import a..b s.go go)\n[codemd]:# (link a s.go)\n",
			want:  "[codemd]:# (import a..b s.go go)\n```go\nfunc A() {}\n```\n[codemd]:# (link a s.go)\n[s.go:2](s.go#L2)\n",
		},
		{
			name:  "link inserts under normal line",
			files: src,
			doc:   "[codemd]:# (link a s.go)\n\nkeep\n",
			want:  "[codemd]:# (link a s.go)\n[s.go:2](s.go#L2)\n\nkeep\n",
		},
		{
			name:  "link replaces generated link",
			files: src,
			doc:   "[codemd]:# (link a s.go)\n\n[s.go:9](s.go#L9)\n",
			want:  "[codemd]:# (link a s.go)\n\n[s.go:2](s.go#L2)\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { runMatrixCase(t, tc) })
	}
}

func TestIntegrationMatrixErrors(t *testing.T) {
	cases := []matrixCase{
		{
			name:    "missing marker",
			files:   map[string]string{"s.go": "package x\n//codemd:a\nfunc A() {}\n"},
			doc:     "[codemd]:# (import nope..a s.go go)\n",
			want:    "[codemd]:# (import nope..a s.go go)\n",
			wantErr: true,
		},
		{
			name:    "bad regex",
			files:   map[string]string{"s.go": "package x\n"},
			doc:     "[codemd]:# (import /(/../x/ s.go go)\n",
			want:    "[codemd]:# (import /(/../x/ s.go go)\n",
			wantErr: true,
		},
		{
			name:    "missing file",
			files:   map[string]string{},
			doc:     "[codemd]:# (import a..b missing.go go)\n",
			want:    "[codemd]:# (import a..b missing.go go)\n",
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { runMatrixCase(t, tc) })
	}
}

func TestIntegrationMatrixIdempotent(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"s.go":   "package x\n//codemd:a\nfunc A() {}\n//codemd:b\n",
		"doc.md": "[codemd]:# (import a..b s.go go)\n\ntext\n\n[codemd]:# (link a s.go)\n",
	})
	docPath := filepath.Join(dir, "doc.md")
	var out, errb bytes.Buffer
	if code := Run([]string{"-w", docPath}, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("first run code %d stderr %s", code, errb.String())
	}
	first, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errb.Reset()
	if code := Run([]string{"-w", docPath}, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("second run code %d stderr %s", code, errb.String())
	}
	second, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatalf("not idempotent:\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
}

func TestIntegrationMatrixLinkLabel(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"s.go":   "package x\n//codemd:a\nfunc A() {}\n//codemd:b\n",
		"doc.md": "[codemd]:# (link a s.go \"My Label\")\n",
	})
	docPath := filepath.Join(dir, "doc.md")
	want := "[codemd]:# (link a s.go \"My Label\")\n[My Label](s.go#L2)\n"
	var out, errb bytes.Buffer
	if code := Run([]string{"-w", docPath}, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("first run code %d stderr %s", code, errb.String())
	}
	first, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", first, want)
	}
	out.Reset()
	errb.Reset()
	if code := Run([]string{"-w", docPath}, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("second run code %d stderr %s", code, errb.String())
	}
	second, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(second) != string(first) {
		t.Fatalf("not idempotent:\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
}

func TestIntegrationMatrixConfig(t *testing.T) {
	files := map[string]string{
		".codemd.yaml": "languages:\n  foo:\n    line: \";;\"\n    fence: foofence\n",
		"s.foo":        "aaa\n;;codemd:a\nbbb\n;;codemd:b\nccc\n",
	}
	cases := []matrixCase{
		{
			name:  "discovered custom line form and fence",
			files: files,
			doc:   "[codemd]:# (import a..b s.foo)\n",
			want:  "[codemd]:# (import a..b s.foo)\n```foofence\nbbb\n```\n",
		},
		{
			name:  "explicit lang token overrides configured fence",
			files: files,
			doc:   "[codemd]:# (import a..b s.foo custom)\n",
			want:  "[codemd]:# (import a..b s.foo custom)\n```custom\nbbb\n```\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { runMatrixCase(t, tc) })
	}
}
