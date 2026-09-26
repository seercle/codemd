# codemd Managed Regions and Integration Matrix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Change the managed-region rule so a reference inserts its generated block directly below the comment unless the first non-blank line below is already the corresponding generated block (then replace it in place), and add a curated end-to-end integration matrix covering ranges, languages, regions, HTTP, config, and errors.

**Architecture:** `cli.splice` is rewritten: it skips blank lines to locate the first non-blank line, replaces it in place when it is the corresponding generated artifact (a fenced block for `import`, a generated link for `link`), and otherwise inserts the replacement directly below the comment. This makes back-to-back references safe (a reference never consumes the next reference comment) and keeps runs idempotent. A new table-driven integration matrix exercises the CLI end-to-end via `cli.Run`.

**Tech Stack:** Go 1.22+, stdlib only. Tests with `testing` + `net/http/httptest`.

**Spec:** `docs/superpowers/specs/2026-09-26-codemd-design.md`

## Global Constraints

- Go module path: `github.com/seercle/codemd`. Go version floor: `1.22`.
- Only external dependency allowed: `gopkg.in/yaml.v3`. Everything else is stdlib.
- Line numbers are 1-based everywhere.
- Named-point boundary lines are excluded from imports; regex-matched boundary lines are included.
- `import` generated block is a triple-backtick fence with the language info string; `link` generated line is `[PATH:LINE](PATH#LLINE)`.
- Content outside managed regions is preserved byte-for-byte.
- Errors are collected, not fatal: continue processing, print a summary to stderr, exit non-zero if any error occurred.
- A reference comment is never consumed by a neighbouring reference.
- Verification commands for every task: `gofmt -l .`, `go vet ./...`, `go test ./...`.
- Build/test environment: `export PATH=/nix/store/3yh8g6m3balbhzxsx77jlyyx443bxqii-go-1.26.6/bin:$PATH CGO_ENABLED=0` (Go is not on the default PATH; the box has no gcc, so `CGO_ENABLED=0` is required for vet/test).

---

### Task 1: New managed-region rule in `splice`

**Files:**
- Modify: `internal/cli/resolve.go`
- Test: `internal/cli/resolve_test.go`

**Interfaces:**
- Consumes: `mdref.FenceBlockEnd(lines []string, start int) (end int, ok bool)`, `mdref.Mode`, `mdref.Link`, `mdref.Import`.
- Produces:
  - `func splice(lines []string, refLine int, mode mdref.Mode, replacement []string) []string` — unchanged signature, new behavior: skip blanks to the first non-blank line; replace in place if it is the corresponding generated artifact, else insert directly below the comment.
  - `func isGeneratedLink(line string) bool` — reports whether a line is a generated link (`[label](target#L<digits>)`).

- [ ] **Step 1: Write the failing tests**

Add these unit tests to `internal/cli/resolve_test.go`. Add `"reflect"` and `"github.com/seercle/codemd/internal/mdref"` to the import block.

```go
func TestSpliceInsertsBelowComment(t *testing.T) {
	lines := []string{"[codemd]:# (link a s.go)", "", "keep me"}
	got := splice(lines, 1, mdref.Link, []string{"[s.go:2](s.go#L2)"})
	want := []string{"[codemd]:# (link a s.go)", "[s.go:2](s.go#L2)", "", "keep me"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v", got)
	}
}

func TestSpliceReplacesGeneratedLink(t *testing.T) {
	lines := []string{"[codemd]:# (link a s.go)", "", "[s.go:9](s.go#L9)"}
	got := splice(lines, 1, mdref.Link, []string{"[s.go:2](s.go#L2)"})
	want := []string{"[codemd]:# (link a s.go)", "", "[s.go:2](s.go#L2)"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v", got)
	}
}

func TestSpliceBackToBackComments(t *testing.T) {
	lines := []string{"[codemd]:# (import a..b s.go go)", "[codemd]:# (link a s.go)"}
	got := splice(lines, 1, mdref.Import, []string{"```go", "x", "```"})
	want := []string{"[codemd]:# (import a..b s.go go)", "```go", "x", "```", "[codemd]:# (link a s.go)"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v", got)
	}
}
```

Also replace `TestResolveDocumentImportAndLink` with this version (the old assertions expected the link to overwrite arbitrary content, which the new rule no longer does):

```go
func TestResolveDocumentImportAndLink(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "server.go"), []byte("package x\n//codemd:a\nfunc f() {}\n//codemd:b\n"), 0o644)
	md := "[codemd]:# (import a..b server.go go)\n\n```go\nstale\n```\n\n[codemd]:# (link a server.go)\n\n[server.go:2](server.go#L2)\n"
	out, errs := resolver().ResolveDocument(md, dir)
	if len(errs) != 0 {
		t.Fatalf("errs %+v", errs)
	}
	if !strings.Contains(out, "```go\nfunc f() {}\n```") {
		t.Fatalf("import not resolved:\n%s", out)
	}
	if !strings.Contains(out, "[server.go:2](server.go#L2)") {
		t.Fatalf("link not resolved:\n%s", out)
	}
	if strings.Contains(out, "stale") {
		t.Fatalf("stale fence kept:\n%s", out)
	}
}
```

And add this document-level test for link insertion under a normal line:

```go
func TestResolveDocumentLinkInsertsUnderNormalLine(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "server.go"), []byte("package x\n//codemd:a\nfunc f() {}\n"), 0o644)
	md := "[codemd]:# (link a server.go)\n\nkeep me\n"
	out, errs := resolver().ResolveDocument(md, dir)
	if len(errs) != 0 {
		t.Fatalf("errs %+v", errs)
	}
	want := "[codemd]:# (link a server.go)\n[server.go:2](server.go#L2)\n\nkeep me\n"
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/ -run 'TestSplice|TestResolveDocumentImportAndLink|TestResolveDocumentLinkInserts' -v`
Expected: FAIL — `TestSpliceInsertsBelowComment` fails (old `splice` overwrites the `keep me` line) and `TestResolveDocumentLinkInsertsUnderNormalLine` fails (old rule overwrites `keep me`). The back-to-back and generated-link-replacement cases already pass under the old rule (no blank lines to skip, and the generated link is replaced either way); they are included as regression guards for the new rule.

- [ ] **Step 3: Rewrite `splice` and add `isGeneratedLink`**

In `internal/cli/resolve.go`, add `"regexp"` to the imports, then replace the `splice` function with:

```go
var generatedLink = regexp.MustCompile(`^\[[^\]]*\]\([^)]*#L\d+\)$`)

func isGeneratedLink(line string) bool {
	return generatedLink.MatchString(strings.TrimSpace(line))
}

// splice updates the managed region for the reference on line refLine
// (1-based). It skips blank lines after the comment to find the first
// non-blank line. If that line is the corresponding generated artifact (a
// fenced block for import, a generated link for link) it is replaced in place;
// otherwise the replacement is inserted directly below the comment, leaving
// any existing line untouched.
func splice(lines []string, refLine int, mode mdref.Mode, replacement []string) []string {
	insertAt := refLine // 0-based index of the line directly below the comment
	idx := insertAt
	for idx < len(lines) && strings.TrimSpace(lines[idx]) == "" {
		idx++
	}
	if idx < len(lines) {
		if mode == mdref.Link {
			if isGeneratedLink(lines[idx]) {
				out := append([]string{}, lines[:idx]...)
				out = append(out, replacement...)
				return append(out, lines[idx+1:]...)
			}
		} else if end, ok := mdref.FenceBlockEnd(lines, idx); ok {
			out := append([]string{}, lines[:idx]...)
			out = append(out, replacement...)
			return append(out, lines[end:]...)
		}
	}
	out := append([]string{}, lines[:insertAt]...)
	out = append(out, replacement...)
	return append(out, lines[insertAt:]...)
}
```

- [ ] **Step 4: Run tests and verify they pass**

Run: `go test ./internal/cli/ -v`
Expected: PASS (all cli tests, including the pre-existing golden integration test).

- [ ] **Step 5: Commit**

```bash
gofmt -l . && go vet ./... && go test ./...
git add internal/cli/resolve.go internal/cli/resolve_test.go
git commit -m "feat: insert generated blocks below the comment unless replacing"
```

---

### Task 2: Regenerate the golden fixture without placeholders

**Files:**
- Modify: `internal/cli/testdata/integration/doc.md`
- Modify: `internal/cli/testdata/integration/want.md`
- Test: `internal/cli/integration_test.go` (existing, unchanged)

**Interfaces:**
- Consumes: the new `splice` behavior from Task 1.
- Produces: a canonical golden fixture whose references sit back-to-back with no placeholder blocks.

- [ ] **Step 1: Replace `doc.md` with the placeholder-free fixture**

Write `internal/cli/testdata/integration/doc.md` exactly:

```
# Docs

[codemd]:# (import handler-start..handler-end server.go go)

[codemd]:# (link handler-start server.go go)

[codemd]:# (import /func handler/../^}/ server.go go)
```

- [ ] **Step 2: Regenerate `want.md` and inspect it**

Run:
```bash
export PATH=/nix/store/3yh8g6m3balbhzxsx77jlyyx443bxqii-go-1.26.6/bin:$PATH CGO_ENABLED=0
go run ./cmd/codemd internal/cli/testdata/integration/doc.md > internal/cli/testdata/integration/want.md
cat internal/cli/testdata/integration/want.md
```

Expected content (verify each point by eye before committing):
- Each reference comment is followed on the next line by its generated block, with no `stale` text anywhere.
- The first import fence contains `func handler() string {` and `return "ok"`.
- The link line is exactly `[server.go:3](server.go#L3)`.
- The regex import fence contains `func handler() string {`, `return "ok"`, and the closing `}` line.
- The two generated blocks are ` ```go ` fenced.

- [ ] **Step 3: Run the golden test and verify it passes**

Run: `go test ./internal/cli/ -run TestIntegrationGolden -v`
Expected: PASS (golden comparison and the idempotency re-run).

- [ ] **Step 4: Commit**

```bash
gofmt -l . && go vet ./... && go test ./...
git add internal/cli/testdata/integration/doc.md internal/cli/testdata/integration/want.md
git commit -m "test: regenerate golden fixture without placeholder blocks"
```

---

### Task 3: Integration matrix (ranges, strip, languages, regions, errors, idempotency)

**Files:**
- Create: `internal/cli/matrix_test.go`

**Interfaces:**
- Consumes: `cli.Run`, the new `splice` behavior.
- Produces: `writeTree(t, dir, files)` and `runMatrixCase(t, tc)` helpers plus table-driven coverage; no exported API.

- [ ] **Step 1: Write the failing test file**

Create `internal/cli/matrix_test.go` exactly:

```go
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
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `go test ./internal/cli/ -run TestIntegrationMatrix -v`
Expected: PASS for every subtest. Task 1 already implements the behavior these tests lock in; a failure here is a defect in Task 1 — stop and report it rather than editing the expectation to match.

- [ ] **Step 3: (No implementation step)**

This task adds tests only; the behavior under test is implemented in Task 1.

- [ ] **Step 4: Run tests and verify they pass**

Run: `go test ./internal/cli/ -run TestIntegrationMatrix -v`
Expected: PASS for every subtest.

- [ ] **Step 5: Commit**

```bash
gofmt -l . && go vet ./... && go test ./...
git add internal/cli/matrix_test.go
git commit -m "test: curated end-to-end integration matrix"
```

---

### Task 4: HTTP integration test

**Files:**
- Create: `internal/cli/http_test.go`

**Interfaces:**
- Consumes: `cli.Run`, `net/http/httptest`, the loader's per-run cache.
- Produces: end-to-end coverage for `http(s)://` sources; no exported API.

- [ ] **Step 1: Write the failing test**

Create `internal/cli/http_test.go` exactly:

```go
package cli

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntegrationHTTP(t *testing.T) {
	const src = "package x\n//codemd:a\nfunc A() {}\n//codemd:b\n"
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		io.WriteString(w, src)
	}))
	defer srv.Close()

	dir := t.TempDir()
	url := srv.URL + "/s.go"
	doc := "[codemd]:# (import a..b " + url + " go)\n\n[codemd]:# (link a " + url + ")\n"
	writeTree(t, dir, map[string]string{"doc.md": doc})

	var out, errb bytes.Buffer
	code := Run([]string{"-w", filepath.Join(dir, "doc.md")}, strings.NewReader(""), &out, &errb)
	if code != 0 {
		t.Fatalf("code %d stderr %s", code, errb.String())
	}
	got, err := os.ReadFile(filepath.Join(dir, "doc.md"))
	if err != nil {
		t.Fatal(err)
	}
	want := "[codemd]:# (import a..b " + url + " go)\n```go\nfunc A() {}\n```\n\n[codemd]:# (link a " + url + ")\n[" + url + ":2](" + url + "#L2)\n"
	if string(got) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	if hits != 1 {
		t.Fatalf("expected the URL to be fetched once (cache), got %d hits", hits)
	}
}

func TestIntegrationHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	dir := t.TempDir()
	url := srv.URL + "/missing.go"
	doc := "[codemd]:# (import a..b " + url + " go)\n"
	writeTree(t, dir, map[string]string{"doc.md": doc})

	var out, errb bytes.Buffer
	code := Run([]string{"-w", filepath.Join(dir, "doc.md")}, strings.NewReader(""), &out, &errb)
	if code == 0 {
		t.Fatal("expected non-zero exit for HTTP 404")
	}
	got, err := os.ReadFile(filepath.Join(dir, "doc.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != doc {
		t.Fatalf("region should be untouched on error:\n%s", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `go test ./internal/cli/ -run TestIntegrationHTTP -v`
Expected: PASS — HTTP loading already exists; this task adds end-to-end coverage. If `TestIntegrationHTTP` fails on the cache hit count, report it as a defect (the loader must fetch each URL once per run).

- [ ] **Step 3: (No implementation step)**

Tests only.

- [ ] **Step 4: Run tests and verify they pass**

Run: `go test ./internal/cli/ -run TestIntegrationHTTP -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
gofmt -l . && go vet ./... && go test ./...
git add internal/cli/http_test.go
git commit -m "test: end-to-end http source coverage"
```

---

### Task 5: Update the spec and README for the new managed-region rule

**Files:**
- Modify: `docs/superpowers/specs/2026-09-26-codemd-design.md`
- Modify: `README.md`

**Interfaces:**
- Consumes: the behavior implemented in Task 1.
- Produces: documentation matching the code; no code change.

- [ ] **Step 1: Replace the spec's "Managed regions" section**

In `docs/superpowers/specs/2026-09-26-codemd-design.md`, replace the section starting at `### Managed regions` through the line before `## Reference grammar` with:

```markdown
### Managed regions

The reference manages the content directly below it. Blank lines after the
comment are skipped to find the first non-blank line:

- Import: if that line opens a fenced block, the whole block (opening fence,
  body, closing fence) is the managed region and is replaced. Otherwise the
  generated fence is inserted directly below the comment, leaving any existing
  line untouched.
- Link: if that line is a generated link (`[label](target#L<n>)`), it is
  replaced. Otherwise the generated link line is inserted directly below the
  comment, leaving any existing line untouched.
- A following reference comment is never consumed: when the first non-blank
  line is another reference, the generated content is inserted below the
  current comment.
- On re-run the region is re-derived from the comment, so repeated runs are
  idempotent.
```

- [ ] **Step 2: Update the README's managed-region paragraph**

In `README.md`, replace the paragraph:

```
The managed region is the first non-blank line after the comment: an existing
fence is replaced for `import`, or the single line is replaced for `link`.
```

with:

```
The reference manages the content directly below it. Blank lines are skipped to
find the first non-blank line: an existing fence (for `import`) or generated
link (for `link`) is replaced in place; otherwise the generated block is
inserted directly below the comment, leaving existing lines untouched.
```

- [ ] **Step 3: Verify and commit**

Run:
```bash
gofmt -l . && go vet ./... && go test ./...
git add docs/superpowers/specs/2026-09-26-codemd-design.md README.md
git commit -m "docs: describe insert-or-replace managed regions"
```

Expected: all commands clean.

---

## Self-Review

**Spec coverage:**
- New managed-region rule (insert unless the first non-blank line is the generated artifact) → Task 1.
- Reference comments never consumed by a neighbour → Task 1 (`TestSpliceBackToBackComments`), Task 3 (`Regions/back to back references`).
- Idempotency under the new rule → Task 1, Task 2, Task 3 (`TestIntegrationMatrixIdempotent`).
- Golden fixture with no placeholders → Task 2.
- Curated matrix: ranges (named/regex/mixed/open), strip, languages (go/py/html/c-dual/unknown-ext/explicit-lang), regions (insert/replace/back-to-back/link), errors (missing marker/bad regex/missing file) → Task 3.
- HTTP sources and per-run caching → Task 4.
- Spec and README reflect the rule → Task 5.

**Placeholder scan:** no TBD/TODO; every code and test step contains full code; every expected value is concrete.

**Type consistency:** `splice(lines []string, refLine int, mode mdref.Mode, replacement []string) []string` keeps its signature; `isGeneratedLink(line string) bool` is used only inside `splice`; `mdref.FenceBlockEnd` is the existing exported helper consumed by `splice`; `writeTree`/`runMatrixCase`/`matrixCase` are defined once in `matrix_test.go` and reused by `http_test.go` (same package).
