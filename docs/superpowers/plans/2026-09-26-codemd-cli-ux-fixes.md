# codemd CLI UX Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix five user-experience defects in the `codemd` CLI: errors must name the file, writes must be all-or-nothing (with a `--force` override), `-h` must exit 0 with a real usage description, `-d` must emit a real unified diff, and whitespace-only link text must error.

**Architecture:** All changes are localized to `internal/cli` and `internal/mdref`. The diff engine moves to a new focused file `internal/cli/diff.go` (replacing the whole-file dump currently in `cli.go`). The per-file processing loop in `cli.go` gains a file-name parameter for errors and a skip-on-error guard for `-w`/`-o`. The parser in `mdref/ref.go` tightens the empty-label check.

**Tech Stack:** Go 1.22+, stdlib only. Tests with `testing`.

**Spec:** `docs/superpowers/specs/2026-09-26-codemd-design.md`

## Global Constraints

- Go module path: `github.com/seercle/codemd`. Go version floor: `1.22`.
- Only external dependency allowed: `gopkg.in/yaml.v3`. Everything else is stdlib.
- Line numbers are 1-based everywhere.
- Content outside managed regions is preserved byte-for-byte.
- Errors are collected, not fatal: continue processing remaining references and files, print a summary to stderr, exit non-zero if any error occurred.
- Multiple input files are supported: `codemd [flags] file.md...`.
- Verification commands for every task: `gofmt -l .`, `go vet ./...`, `go test ./...`.
- Build/test environment: `export PATH=/nix/store/3yh8g6m3balbhzxsx77jlyyx443bxqii-go-1.26.6/bin:$PATH CGO_ENABLED=0` (Go is not on the default PATH; the box has no gcc, so `CGO_ENABLED=0` is required for vet/test).
- `-w`, `-o`, `-d`, and `--check` remain mutually exclusive.

---

### Task 1: Whitespace-only link text is an error

**Files:**
- Modify: `internal/mdref/ref.go:91`
- Test: `internal/mdref/ref_test.go:62-81`

**Interfaces:**
- Consumes: `mdref.ParseRef` (existing).
- Produces: no signature change. `ParseRef` now returns an error for a link label that is empty or contains only whitespace. Error string is unchanged: `empty link text in %q`.

- [ ] **Step 1: Write the failing test**

In `internal/mdref/ref_test.go`, add this entry to the `bad` slice in `TestParseErrors`, immediately after the `empty link text` line:

```go
		`(link a src/x.go "   ")`,         // whitespace-only link text
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/mdref/ -run TestParseErrors -v`
Expected: FAIL — `expected error for "(link a src/x.go \"   \")"`.

- [ ] **Step 3: Implement the fix**

In `internal/mdref/ref.go`, change the label emptiness check:

```go
			label := tok[1 : len(tok)-1]
			if strings.TrimSpace(label) == "" {
				return Ref{}, fmt.Errorf("empty link text in %q", comment)
			}
```

(The label itself is still stored verbatim, so `" My Label "` keeps its surrounding spaces.)

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/mdref/ -run TestParse -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
gofmt -l . && go vet ./... && go test ./...
git add internal/mdref/ref.go internal/mdref/ref_test.go
git commit -m "fix: reject whitespace-only link text"
```

---

### Task 2: Errors name the file

**Files:**
- Modify: `internal/cli/cli.go:102,164,204-216`
- Test: `internal/cli/cli_test.go`

**Interfaces:**
- Consumes: `RefError` (existing), `reportErrors` (existing).
- Produces: `reportErrors` gains a leading `name string` parameter: `func reportErrors(name string, errs []RefError, stderr io.Writer) int`. Output format becomes `codemd: <name>: line <n>: <err>` (or `codemd: <name>: <err>` when the line is 0). File mode passes the file path; stdin mode passes the literal `<stdin>`.

- [ ] **Step 1: Write the failing tests**

Add to `internal/cli/cli_test.go`:

```go
func TestRunErrorNamesFile(t *testing.T) {
	dir := t.TempDir()
	md := filepath.Join(dir, "doc.md")
	os.WriteFile(md, []byte("[codemd]:# (import a..b missing.go go)\n"), 0o644)
	var out, errb bytes.Buffer
	if code := Run([]string{md}, strings.NewReader(""), &out, &errb); code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(errb.String(), md+": line 1:") {
		t.Fatalf("stderr should name the file:\n%s", errb.String())
	}
}

func TestRunErrorNamesStdin(t *testing.T) {
	md := "[codemd]:# (import a..b missing.go go)\n"
	var out, errb bytes.Buffer
	if code := Run(nil, strings.NewReader(md), &out, &errb); code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(errb.String(), "<stdin>: line 1:") {
		t.Fatalf("stderr should name stdin:\n%s", errb.String())
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/cli/ -run 'TestRunErrorNames' -v`
Expected: FAIL — stderr currently reads `codemd: line 1: ...` with no file name.

- [ ] **Step 3: Update `reportErrors`**

In `internal/cli/cli.go`, replace the whole `reportErrors` function:

```go
func reportErrors(name string, errs []RefError, stderr io.Writer) int {
	for _, e := range errs {
		if e.Line > 0 {
			fmt.Fprintf(stderr, "codemd: %s: line %d: %v\n", name, e.Line, e.Err)
		} else {
			fmt.Fprintf(stderr, "codemd: %s: %v\n", name, e.Err)
		}
	}
	if len(errs) > 0 {
		return len(errs)
	}
	return 0
}
```

- [ ] **Step 4: Update both call sites**

In the stdin branch (around `cli.go:102`), change:

```go
		errCount := reportErrors(errs, stderr)
```

to:

```go
		errCount := reportErrors("<stdin>", errs, stderr)
```

In the file loop (around `cli.go:164`), change:

```go
		if n := reportErrors(errs, stderr); n > 0 {
```

to:

```go
		if n := reportErrors(file, errs, stderr); n > 0 {
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./internal/cli/ -run 'TestRunError' -v`
Expected: PASS.

- [ ] **Step 6: Run the full suite and commit**

```bash
gofmt -l . && go vet ./... && go test ./...
git add internal/cli/cli.go internal/cli/cli_test.go
git commit -m "fix: include file name in error output"
```

---

### Task 3: `-h` exits 0 with a usage description

**Files:**
- Modify: `internal/cli/cli.go:1-35`
- Test: `internal/cli/cli_test.go`

**Interfaces:**
- Consumes: `Run` (existing).
- Produces: `Run` returns 0 for `-h`/`--help` (instead of 2), and prints a custom usage block (description, usage lines including the stdin form, the reference grammar, then the flag list). Unknown flags still return 2.

- [ ] **Step 1: Write the failing tests**

Add to `internal/cli/cli_test.go`:

```go
func TestRunHelpExitsZero(t *testing.T) {
	var out, errb bytes.Buffer
	if code := Run([]string{"-h"}, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("help should exit 0, got %d", code)
	}
	if !strings.Contains(errb.String(), "Usage:") {
		t.Fatalf("help should print usage:\n%s", errb.String())
	}
	if !strings.Contains(errb.String(), "read stdin") {
		t.Fatalf("help should mention the stdin form:\n%s", errb.String())
	}
	if !strings.Contains(errb.String(), "[codemd]:#") {
		t.Fatalf("help should show the reference grammar:\n%s", errb.String())
	}
}

func TestRunUnknownFlagExitsTwo(t *testing.T) {
	var out, errb bytes.Buffer
	if code := Run([]string{"--bogus"}, strings.NewReader(""), &out, &errb); code != 2 {
		t.Fatalf("unknown flag should exit 2, got %d", code)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/cli/ -run 'TestRunHelp|TestRunUnknownFlag' -v`
Expected: FAIL — `-h` currently exits 2 and prints only `Usage of codemd:`.

- [ ] **Step 3: Add the `errors` import**

In `internal/cli/cli.go`, add `"errors"` to the import block:

```go
import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/seercle/codemd/internal/lang"
	"github.com/seercle/codemd/internal/srcfile"
)
```

- [ ] **Step 4: Install the custom usage and ErrHelp handling**

Replace the flag-setup and parse block at the top of `Run` (currently `cli.go:24-35`):

```go
	fs := flag.NewFlagSet("codemd", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprint(stderr, "codemd resolves code references embedded in Markdown.\n\n")
		fmt.Fprint(stderr, "Usage:\n")
		fmt.Fprint(stderr, "  codemd [flags] file.md...\n")
		fmt.Fprint(stderr, "  codemd [flags]              (no files: read stdin, write stdout)\n\n")
		fmt.Fprint(stderr, "Reference: [codemd]:# (MODE RANGE PATH [LANG] [strip] [\"LINK-TEXT\"])\n\n")
		fmt.Fprint(stderr, "Flags:\n")
		fs.PrintDefaults()
	}
	var opt Options
	fs.BoolVar(&opt.Write, "w", false, "write result in place")
	fs.StringVar(&opt.Output, "o", "", "write result to a new file")
	fs.BoolVar(&opt.Diff, "d", false, "print a unified diff")
	fs.BoolVar(&opt.Check, "check", false, "exit non-zero if any file would change")
	fs.StringVar(&opt.Config, "config", "", "path to config file")
	languages := fs.Bool("languages", false, "list supported languages and exit")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./internal/cli/ -run 'TestRunHelp|TestRunUnknownFlag' -v`
Expected: PASS.

- [ ] **Step 6: Run the full suite and commit**

```bash
gofmt -l . && go vet ./... && go test ./...
git add internal/cli/cli.go internal/cli/cli_test.go
git commit -m "fix: -h exits 0 and prints usage description"
```

---

### Task 4: Real unified diff

**Files:**
- Create: `internal/cli/diff.go`
- Modify: `internal/cli/cli.go` (remove `unifiedDiff` and `splitDiffLines`, currently `cli.go:218-243`)
- Test: `internal/cli/cli_test.go:125-163`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: `func unifiedDiff(name, old, new string) string` — signature unchanged, but now emits one or more hunks with 3 lines of context and correct `@@ -a,b +c,d @@` headers, instead of dumping the whole file. Helpers `diffOps(a, b []string) []diffOp`, `diffHunks(a, b []string, context int) []diffHunk`, and `splitDiffLines(s string) []string` live in `diff.go`.

- [ ] **Step 1: Write the failing tests**

Replace `TestUnifiedDiffShape` in `internal/cli/cli_test.go` with the following. (It keeps the original small-case assertions and the `patch --dry-run` check, and adds exact-output and multi-hunk cases.)

```go
func TestUnifiedDiffShape(t *testing.T) {
	d := unifiedDiff("doc.md", "a\nb\n", "a\nc\n")
	want := "--- a/doc.md\n+++ b/doc.md\n@@ -1,2 +1,2 @@\n a\n-b\n+c\n"
	if d != want {
		t.Fatalf("got:\n%q\nwant:\n%q", d, want)
	}

	if _, err := exec.LookPath("patch"); err != nil {
		t.Skip("patch not available")
	}
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "doc.md")
	if err := os.WriteFile(oldPath, []byte("a\nb\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("patch", "--dry-run", oldPath)
	cmd.Stdin = strings.NewReader(d)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("patch --dry-run failed: %v\n%s", err, out)
	}
}

func TestUnifiedDiffContext(t *testing.T) {
	old := "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n"
	new := "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\nCHANGED\n"
	d := unifiedDiff("doc.md", old, new)
	want := "--- a/doc.md\n+++ b/doc.md\n@@ -9,4 +9,4 @@\n 9\n 10\n 11\n-12\n+CHANGED\n"
	if d != want {
		t.Fatalf("got:\n%q\nwant:\n%q", d, want)
	}
}

func TestUnifiedDiffMultipleHunks(t *testing.T) {
	var oldB, newB strings.Builder
	for i := 1; i <= 30; i++ {
		fmt.Fprintf(&oldB, "%d\n", i)
		if i == 5 {
			newB.WriteString("5X\n")
		} else if i == 25 {
			newB.WriteString("25Y\n")
		} else {
			fmt.Fprintf(&newB, "%d\n", i)
		}
	}
	d := unifiedDiff("doc.md", oldB.String(), newB.String())
	if strings.Count(d, "@@") != 4 {
		t.Fatalf("expected two hunks (4 @@ markers):\n%s", d)
	}
	if !strings.Contains(d, "-5\n+5X\n") {
		t.Fatalf("first change missing:\n%s", d)
	}
	if !strings.Contains(d, "-25\n+25Y\n") {
		t.Fatalf("second change missing:\n%s", d)
	}
	if strings.Contains(d, " 15\n") {
		t.Fatalf("unchanged middle line should not appear:\n%s", d)
	}
}

func TestUnifiedDiffNoChange(t *testing.T) {
	d := unifiedDiff("doc.md", "a\nb\n", "a\nb\n")
	want := "--- a/doc.md\n+++ b/doc.md\n"
	if d != want {
		t.Fatalf("got:\n%q\nwant:\n%q", d, want)
	}
}
```

Add `"fmt"` to the imports of `internal/cli/cli_test.go` if it is not already present.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/cli/ -run TestUnifiedDiff -v`
Expected: FAIL — the current implementation dumps the whole file and emits one `@@ -1,N +1,M @@` header.

- [ ] **Step 3: Create `internal/cli/diff.go`**

```go
package cli

import (
	"fmt"
	"path/filepath"
	"strings"
)

// unifiedDiff renders a unified diff between old and new with 3 lines of
// context per hunk. name is the file path used in the ---/+++ headers.
func unifiedDiff(name, old, new string) string {
	oldLines := splitDiffLines(old)
	newLines := splitDiffLines(new)
	clean := strings.TrimPrefix(name, string(filepath.Separator))
	var b strings.Builder
	fmt.Fprintf(&b, "--- a/%s\n+++ b/%s\n", clean, clean)
	for _, h := range diffHunks(oldLines, newLines, 3) {
		fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", h.oldStart, h.oldCount, h.newStart, h.newCount)
		for _, l := range h.lines {
			b.WriteString(l)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func splitDiffLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

type diffOp struct {
	kind byte // ' ' equal, '-' delete, '+' insert
	text string
}

type diffHunk struct {
	oldStart, oldCount int
	newStart, newCount int
	lines              []string
}

// diffOps computes an LCS-based edit script from a to b.
func diffOps(a, b []string) []diffOp {
	n, m := len(a), len(b)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			switch {
			case a[i] == b[j]:
				dp[i][j] = dp[i+1][j+1] + 1
			case dp[i+1][j] >= dp[i][j+1]:
				dp[i][j] = dp[i+1][j]
			default:
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	var ops []diffOp
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			ops = append(ops, diffOp{' ', a[i]})
			i++
			j++
		case dp[i+1][j] >= dp[i][j+1]:
			ops = append(ops, diffOp{'-', a[i]})
			i++
		default:
			ops = append(ops, diffOp{'+', b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		ops = append(ops, diffOp{'-', a[i]})
	}
	for ; j < m; j++ {
		ops = append(ops, diffOp{'+', b[j]})
	}
	return ops
}

// diffHunks groups the edit script into hunks with up to context equal lines
// on either side of each change, merging hunks whose context windows touch.
func diffHunks(a, b []string, context int) []diffHunk {
	ops := diffOps(a, b)
	oldAt := make([]int, len(ops)+1)
	newAt := make([]int, len(ops)+1)
	ol, nl := 1, 1
	for i, op := range ops {
		oldAt[i] = ol
		newAt[i] = nl
		switch op.kind {
		case ' ':
			ol++
			nl++
		case '-':
			ol++
		case '+':
			nl++
		}
	}
	oldAt[len(ops)] = ol
	newAt[len(ops)] = nl

	type span struct{ lo, hi int }
	var spans []span
	for i, op := range ops {
		if op.kind == ' ' {
			continue
		}
		lo := i - context
		if lo < 0 {
			lo = 0
		}
		hi := i + context
		if hi > len(ops)-1 {
			hi = len(ops) - 1
		}
		if n := len(spans); n > 0 && lo <= spans[n-1].hi+1 {
			if hi > spans[n-1].hi {
				spans[n-1].hi = hi
			}
		} else {
			spans = append(spans, span{lo, hi})
		}
	}

	hunks := make([]diffHunk, 0, len(spans))
	for _, s := range spans {
		h := diffHunk{oldStart: oldAt[s.lo], newStart: newAt[s.lo]}
		for _, op := range ops[s.lo : s.hi+1] {
			h.lines = append(h.lines, string(op.kind)+op.text)
			switch op.kind {
			case ' ':
				h.oldCount++
				h.newCount++
			case '-':
				h.oldCount++
			case '+':
				h.newCount++
			}
		}
		if h.oldCount == 0 {
			h.oldStart--
		}
		if h.newCount == 0 {
			h.newStart--
		}
		hunks = append(hunks, h)
	}
	return hunks
}
```

- [ ] **Step 4: Remove the old implementation from `cli.go`**

Delete the `unifiedDiff` and `splitDiffLines` functions from `internal/cli/cli.go` (the block currently at `cli.go:218-243`, from `func unifiedDiff(` through the closing brace of `splitDiffLines`).

Then fix the import block: `strings` was used only by those two functions, so remove `"strings"` from the import list or the build will fail with `"strings" imported and not used`. Keep `"filepath"` — it is still used at `cli.go:122` (`filepath.Dir(file)`). The import block becomes:

```go
import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/seercle/codemd/internal/lang"
	"github.com/seercle/codemd/internal/srcfile"
)
```

(Note: `"errors"` is added by Task 3; if you are executing Task 4 after Task 3, it is already present.)

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./internal/cli/ -run TestUnifiedDiff -v`
Expected: PASS.

- [ ] **Step 6: Run the full suite and commit**

```bash
gofmt -l . && go vet ./... && go test ./...
git add internal/cli/diff.go internal/cli/cli.go internal/cli/cli_test.go
git commit -m "feat: emit real unified diff with context hunks"
```

---

### Task 5: All-or-nothing writes with `--force`/`-f`

**Files:**
- Modify: `internal/cli/cli.go` (`Options`, flag registration, file loop)
- Test: `internal/cli/cli_test.go`

**Interfaces:**
- Consumes: `reportErrors(file, errs, stderr)` and the file-name parameter from Task 2.
- Produces: `Options` gains `Force bool`. New flags `-f` and `--force` both bind to it. When a file has one or more reference errors, `-w` and `-o` skip writing that file (printing `codemd: <file>: not written due to errors (use --force to write anyway)`) unless `Force` is set. `-d`, `--check`, and stdout mode are unaffected (they never persist).

- [ ] **Step 1: Write the failing tests**

Add to `internal/cli/cli_test.go`:

```go
func TestRunWriteAllOrNothing(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"s.go":   "package x\n//codemd:a\nfunc A() {}\n//codemd:b\n",
		"doc.md": "[codemd]:# (import a..b s.go go)\n[codemd]:# (import a..zzz s.go go)\n",
	})
	docPath := filepath.Join(dir, "doc.md")
	before, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	if code := Run([]string{"-w", docPath}, strings.NewReader(""), &out, &errb); code == 0 {
		t.Fatal("expected non-zero exit")
	}
	after, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("file should be untouched on error:\n%s", after)
	}
	if !strings.Contains(errb.String(), "not written") {
		t.Fatalf("stderr should explain the skip:\n%s", errb.String())
	}
}

func TestRunWriteForce(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"s.go":   "package x\n//codemd:a\nfunc A() {}\n//codemd:b\n",
		"doc.md": "[codemd]:# (import a..b s.go go)\n[codemd]:# (import a..zzz s.go go)\n",
	})
	docPath := filepath.Join(dir, "doc.md")
	var out, errb bytes.Buffer
	if code := Run([]string{"-w", "--force", docPath}, strings.NewReader(""), &out, &errb); code == 0 {
		t.Fatal("expected non-zero exit even with --force")
	}
	after, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "```go\nfunc A() {}\n```") {
		t.Fatalf("--force should write the good region:\n%s", after)
	}
}

func TestRunWriteForceShorthand(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"s.go":   "package x\n//codemd:a\nfunc A() {}\n//codemd:b\n",
		"doc.md": "[codemd]:# (import a..b s.go go)\n[codemd]:# (import a..zzz s.go go)\n",
	})
	docPath := filepath.Join(dir, "doc.md")
	var out, errb bytes.Buffer
	if code := Run([]string{"-w", "-f", docPath}, strings.NewReader(""), &out, &errb); code == 0 {
		t.Fatal("expected non-zero exit even with -f")
	}
	after, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "```go\nfunc A() {}\n```") {
		t.Fatalf("-f should write the good region:\n%s", after)
	}
}

func TestRunOutputAllOrNothing(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"s.go":   "package x\n//codemd:a\nfunc A() {}\n//codemd:b\n",
		"doc.md": "[codemd]:# (import a..b s.go go)\n[codemd]:# (import a..zzz s.go go)\n",
	})
	outPath := filepath.Join(dir, "out.md")
	var out, errb bytes.Buffer
	if code := Run([]string{"-o", outPath, filepath.Join(dir, "doc.md")}, strings.NewReader(""), &out, &errb); code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("output file should not be created on error")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/cli/ -run 'TestRunWrite|TestRunOutput' -v`
Expected: FAIL — `-w` currently writes the good region despite the error, and `--force`/`-f` are unknown flags (exit 2).

- [ ] **Step 3: Add the `Force` option and flags**

In `internal/cli/cli.go`, add `Force` to `Options`:

```go
type Options struct {
	Write  bool
	Output string
	Diff   bool
	Check  bool
	Config string
	Force  bool
}
```

Register both flag spellings next to the others (after the `--config` line):

```go
	fs.StringVar(&opt.Config, "config", "", "path to config file")
	fs.BoolVar(&opt.Force, "f", false, "write even if some references failed")
	fs.BoolVar(&opt.Force, "force", false, "write even if some references failed")
	languages := fs.Bool("languages", false, "list supported languages and exit")
```

- [ ] **Step 4: Guard `-w` and `-o` on errors**

In the file loop, capture the per-file error count and use it. Change the block that reports errors and dispatches output. The current code is:

```go
		out, errs := resolver.ResolveDocument(string(data), baseDir)
		if n := reportErrors(file, errs, stderr); n > 0 {
			errCount += n
			exit = 1
		}
		changed := out != string(data)
		switch {
		case opt.Check:
			if changed {
				fmt.Fprintf(stderr, "codemd: %s is out of date\n", file)
				errCount++
				exit = 1
			}
		case opt.Diff:
			if changed {
				io.WriteString(stdout, unifiedDiff(file, string(data), out))
			}
		case opt.Write:
			if changed {
				if err := os.WriteFile(file, []byte(out), 0o644); err != nil {
					fmt.Fprintf(stderr, "codemd: %v\n", err)
					errCount++
					exit = 1
				}
			}
		case opt.Output != "":
			if err := os.WriteFile(opt.Output, []byte(out), 0o644); err != nil {
				fmt.Fprintf(stderr, "codemd: %v\n", err)
				errCount++
				exit = 1
			}
		default:
			io.WriteString(stdout, out)
		}
```

Replace it with:

```go
		out, errs := resolver.ResolveDocument(string(data), baseDir)
		fileErrs := len(errs)
		if n := reportErrors(file, errs, stderr); n > 0 {
			errCount += n
			exit = 1
		}
		changed := out != string(data)
		switch {
		case opt.Check:
			if changed {
				fmt.Fprintf(stderr, "codemd: %s is out of date\n", file)
				errCount++
				exit = 1
			}
		case opt.Diff:
			if changed {
				io.WriteString(stdout, unifiedDiff(file, string(data), out))
			}
		case opt.Write:
			if fileErrs > 0 && !opt.Force {
				fmt.Fprintf(stderr, "codemd: %s: not written due to errors (use --force to write anyway)\n", file)
			} else if changed {
				if err := os.WriteFile(file, []byte(out), 0o644); err != nil {
					fmt.Fprintf(stderr, "codemd: %v\n", err)
					errCount++
					exit = 1
				}
			}
		case opt.Output != "":
			if fileErrs > 0 && !opt.Force {
				fmt.Fprintf(stderr, "codemd: %s: not written due to errors (use --force to write anyway)\n", file)
			} else if err := os.WriteFile(opt.Output, []byte(out), 0o644); err != nil {
				fmt.Fprintf(stderr, "codemd: %v\n", err)
				errCount++
				exit = 1
			}
		default:
			io.WriteString(stdout, out)
		}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./internal/cli/ -run 'TestRunWrite|TestRunOutput' -v`
Expected: PASS.

- [ ] **Step 6: Run the full suite and commit**

```bash
gofmt -l . && go vet ./... && go test ./...
git add internal/cli/cli.go internal/cli/cli_test.go
git commit -m "feat: all-or-nothing writes with --force override"
```

---

### Task 6: Update the spec and README

**Files:**
- Modify: `docs/superpowers/specs/2026-09-26-codemd-design.md`
- Modify: `README.md`

**Interfaces:**
- Consumes: the behavior from Tasks 1–5.
- Produces: documentation matching the code; no code change.

- [ ] **Step 1: Update the spec's LINK-TEXT bullet**

In `docs/superpowers/specs/2026-09-26-codemd-design.md`, in the **LINK-TEXT** bullet, change the phrase:

```
A quoted
  token in import mode, an empty link text, more than one link text, an
  unterminated quote, or a quoted PATH is an error.
```

to:

```
A quoted
  token in import mode, an empty or whitespace-only link text, more than one
  link text, an unterminated quote, or a quoted PATH is an error.
```

- [ ] **Step 2: Update the spec's CLI section**

In the `## CLI` section, add this bullet after the `--languages` bullet:

```markdown
- `-f`, `--force`: with `-w` or `-o`, write even if some references failed.
```

Then, after the paragraph beginning "`-w`, `-o`, `-d`, and `--check` are mutually exclusive.", add:

```markdown
`-w` and `-o` are all-or-nothing per file: if any reference in a file fails,
that file is not written (other files still process). `--force` overrides this.
`-d` and stdout output are unaffected and still print the partial result.
```

- [ ] **Step 3: Update the spec's Errors section**

In the `## Errors` section, change the first sentence:

```
Collect-and-continue: every reference error is recorded with file, line, and
reason; remaining references and files still process.
```

to:

```
Collect-and-continue: every reference error is recorded with the file name,
line, and reason (stdin is reported as `<stdin>`); remaining references and
files still process.
```

- [ ] **Step 4: Update the README**

In `README.md`, in the **LINK-TEXT** bullet, change:

```
It
  may contain spaces and must not contain `]`.
```

to:

```
It
  may contain spaces and must not contain `]`; an empty or whitespace-only
  label is an error.
```

Add this row to the Flags table after the `--config path` row:

```markdown
| `-f`, `--force` | Write even if some references failed. |
```

- [ ] **Step 5: Verify and commit**

Run:
```bash
gofmt -l . && go vet ./... && go test ./...
git add docs/superpowers/specs/2026-09-26-codemd-design.md README.md
git commit -m "docs: document file-named errors, --force, and diff behavior"
```

Expected: all commands clean.

---

## Self-Review

**Spec coverage:**
- Item 1 (file name in errors) → Task 2.
- Item 2 (all-or-nothing writes + force flag) → Task 5.
- Item 3 (`-h` exit 0 + usage description) → Task 3.
- Item 4 (real unified diff) → Task 4.
- Item 5 (whitespace-only link text) → Task 1.
- Docs for all of the above → Task 6.

**Placeholder scan:** no TBD/TODO; every code and test step contains full code; every expected value is concrete (exact diff strings, exact error fragments, exact exit codes).

**Type consistency:** `reportErrors(name string, errs []RefError, stderr io.Writer) int` is defined in Task 2 and consumed by Task 5; `Options.Force` is defined and consumed in Task 5; `unifiedDiff`/`splitDiffLines` keep their signatures when moved to `diff.go` in Task 4; `diffOps`/`diffHunk`/`diffHunks` are used only within `diff.go`. The `-f`/`--force` flag names and the `not written due to errors` message are consistent between Task 5 and Task 6.
