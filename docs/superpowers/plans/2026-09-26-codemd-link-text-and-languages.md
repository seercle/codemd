# codemd Link Text and Language Listing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let `link` references specify custom link text via a quoted token, expose the supported language table with a `--languages` flag, document the built-in languages in the README, and record inline references as a future objective.

**Architecture:** The reference tokenizer in `internal/mdref/ref.go` gains a quote-aware splitter so a `"..."` token is captured whole; `Ref` gains a `Label` field, and `internal/cli/resolve.go` uses it as the link label when present. A new `lang.Describe` helper renders the language table, and `cli.Run` gains a `--languages` flag that prints it and exits. Docs are updated to match.

**Tech Stack:** Go 1.22+, stdlib only. Tests with `testing`.

**Spec:** `docs/superpowers/specs/2026-09-26-codemd-design.md`

## Global Constraints

- Go module path: `github.com/seercle/codemd`. Go version floor: `1.22`.
- Only external dependency allowed: `gopkg.in/yaml.v3`. Everything else is stdlib.
- Line numbers are 1-based everywhere.
- Named-point boundary lines are excluded from imports; regex-matched boundary lines are included.
- `import` generated block is a triple-backtick fence with the language info string; `link` generated line is `[LABEL](PATH#LLINE)`, where LABEL is the custom link text when present, else `PATH:LINE`.
- Content outside managed regions is preserved byte-for-byte.
- Errors are collected, not fatal: continue processing, print a summary to stderr, exit non-zero if any error occurred.
- A reference comment is never consumed by a neighbouring reference.
- Verification commands for every task: `gofmt -l .`, `go vet ./...`, `go test ./...`.
- Build/test environment: `export PATH=/nix/store/3yh8g6m3balbhzxsx77jlyyx443bxqii-go-1.26.6/bin:$PATH CGO_ENABLED=0` (Go is not on the default PATH; the box has no gcc, so `CGO_ENABLED=0` is required for vet/test).

---

### Task 1: Custom link text

**Files:**
- Modify: `internal/mdref/ref.go`
- Modify: `internal/cli/resolve.go`
- Test: `internal/mdref/ref_test.go`
- Test: `internal/cli/matrix_test.go`

**Interfaces:**
- Consumes: `mdref.Ref` (existing struct), `render.LinkLabel`, `render.LinkTarget`, `extract.ResolveLink`.
- Produces:
  - `mdref.Ref` gains a `Label string` field (empty when no custom link text was given).
  - `func splitTokens(s string) ([]string, error)` — splits on whitespace, keeps a double-quoted run (quotes included) as one token, errors on an unterminated quote. Used only inside `ParseRef`.
  - Grammar: `(link RANGE PATH ["LINK-TEXT"])`.

- [ ] **Step 1: Write the failing parser tests**

Add to `internal/mdref/ref_test.go`:

```go
func TestParseLinkCustomLabel(t *testing.T) {
	r, err := ParseRef(`(link a src/x.go "My Label")`)
	if err != nil {
		t.Fatal(err)
	}
	if r.Mode != Link || r.Range.Start.Name != "a" || r.Path != "src/x.go" || r.Label != "My Label" {
		t.Fatalf("got %+v", r)
	}
}

func TestParseLinkNoLabel(t *testing.T) {
	r, err := ParseRef("(link a src/x.go go)")
	if err != nil {
		t.Fatal(err)
	}
	if r.Label != "" || r.Lang != "go" {
		t.Fatalf("got %+v", r)
	}
}
```

And add these entries to the `bad` slice in `TestParseErrors`:

```go
		`(import a..b src/x.go "nope")`, // link text on import
		`(link a src/x.go "")`,          // empty link text
		`(link a src/x.go "a]b")`,       // link text with bracket
		`(link a src/x.go "A" "B")`,     // multiple link texts
		`(link a src/x.go "oops)`,       // unterminated quote
		`(link a "src/x.go")`,           // quoted path
```

- [ ] **Step 2: Run the parser tests to verify they fail**

Run: `go test ./internal/mdref/ -run 'TestParse' -v`
Expected: FAIL — `TestParseLinkCustomLabel` fails (no `Label` field / quoted token not captured) and the new `TestParseErrors` entries fail (no error returned for them).

- [ ] **Step 3: Implement the tokenizer and label parsing**

In `internal/mdref/ref.go`, add a `Label` field to `Ref`:

```go
type Ref struct {
	Mode  Mode
	Range extract.Range
	Path  string
	Lang  string
	Strip bool
	Label string
}
```

Replace the line `rest := strings.Fields(s[i:])` with:

```go
	rest, err := splitTokens(s[i:])
	if err != nil {
		return Ref{}, fmt.Errorf("%v in %q", err, comment)
	}
```

Replace the token loop (currently from `if len(rest) == 0 {` through the end of the `for _, tok := range rest` loop) with:

```go
	if len(rest) == 0 {
		return Ref{}, fmt.Errorf("missing path in %q", comment)
	}
	if strings.HasPrefix(rest[0], `"`) {
		return Ref{}, fmt.Errorf("path must not be quoted in %q", comment)
	}
	r.Path = rest[0]
	rest = rest[1:]
	for _, tok := range rest {
		switch {
		case tok == "strip":
			r.Strip = true
		case strings.HasPrefix(tok, `"`):
			if r.Mode != Link {
				return Ref{}, fmt.Errorf("link text is only valid for link mode in %q", comment)
			}
			label := tok[1 : len(tok)-1]
			if label == "" {
				return Ref{}, fmt.Errorf("empty link text in %q", comment)
			}
			if strings.Contains(label, "]") {
				return Ref{}, fmt.Errorf("link text must not contain ']' in %q", comment)
			}
			if r.Label != "" {
				return Ref{}, fmt.Errorf("multiple link labels in %q", comment)
			}
			r.Label = label
		case r.Lang == "":
			r.Lang = tok
		default:
			return Ref{}, fmt.Errorf("unexpected token %q in %q", tok, comment)
		}
	}
```

Add `splitTokens` immediately before `func isSpace`:

```go
// splitTokens splits s on whitespace, treating a double-quoted run as a single
// token (quotes included). It returns an error for an unterminated quote.
func splitTokens(s string) ([]string, error) {
	var toks []string
	i := 0
	for i < len(s) {
		for i < len(s) && isSpace(s[i]) {
			i++
		}
		if i >= len(s) {
			break
		}
		if s[i] == '"' {
			j := i + 1
			for j < len(s) && s[j] != '"' {
				j++
			}
			if j >= len(s) {
				return nil, fmt.Errorf("unterminated quoted string")
			}
			toks = append(toks, s[i:j+1])
			i = j + 1
			continue
		}
		start := i
		for i < len(s) && !isSpace(s[i]) {
			i++
		}
		toks = append(toks, s[start:i])
	}
	return toks, nil
}
```

- [ ] **Step 4: Run the parser tests to verify they pass**

Run: `go test ./internal/mdref/ -run 'TestParse' -v`
Expected: PASS.

- [ ] **Step 5: Write the failing end-to-end test**

Add to `internal/cli/matrix_test.go`:

```go
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
```

- [ ] **Step 6: Run the end-to-end test to verify it fails**

Run: `go test ./internal/cli/ -run TestIntegrationMatrixLinkLabel -v`
Expected: FAIL — the generated label is `[s.go:2](s.go#L2)`, not `[My Label](s.go#L2)`.

- [ ] **Step 7: Use the label in link rendering**

In `internal/cli/resolve.go`, replace the link-mode return in `resolveOne`:

```go
	if ref.Ref.Mode == mdref.Link {
		line, err := extract.ResolveLink(content, markers, ref.Ref.Range.Start)
		if err != nil {
			return nil, err
		}
		label := ref.Ref.Label
		if label == "" {
			label = render.LinkLabel(ref.Ref.Path, line)
		}
		return []string{fmt.Sprintf("[%s](%s)", label, render.LinkTarget(ref.Ref.Path, line, isRemote))}, nil
	}
```

- [ ] **Step 8: Run the tests to verify they pass**

Run: `go test ./internal/cli/ -run TestIntegrationMatrixLinkLabel -v`
Expected: PASS (including the idempotency re-run).

- [ ] **Step 9: Run the full suite and commit**

```bash
gofmt -l . && go vet ./... && go test ./...
git add internal/mdref/ref.go internal/mdref/ref_test.go internal/cli/resolve.go internal/cli/matrix_test.go
git commit -m "feat: custom link text via quoted token"
```

---

### Task 2: `--languages` flag

**Files:**
- Modify: `internal/lang/lang.go`
- Modify: `internal/cli/cli.go`
- Test: `internal/lang/lang_test.go`
- Test: `internal/cli/cli_test.go`

**Interfaces:**
- Consumes: `lang.Builtins`, `lang.LoadConfig`, `lang.Merge`, `cli.Options`.
- Produces:
  - `func lang.Describe(table map[string]Language) []string` — one line per language, sorted by extension. Format: `ext -> fence (line "prefix")`, `ext -> fence (block "open" "close")`, or `ext -> fence (line "prefix" or block "open" "close")` for dual-form entries.
  - `--languages` boolean flag on `cli.Run`: prints `lang.Describe` of the effective table (built-ins, or built-ins merged with `--config`) to stdout and returns 0.

- [ ] **Step 1: Write the failing `Describe` test**

Add to `internal/lang/lang_test.go` (add `"reflect"` to the import block):

```go
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
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/lang/ -run TestDescribe -v`
Expected: FAIL with "undefined: Describe".

- [ ] **Step 3: Implement `Describe`**

In `internal/lang/lang.go`, replace `import "strings"` with:

```go
import (
	"fmt"
	"sort"
	"strings"
)
```

Append to the file:

```go
// Describe returns one human-readable line per language in table, sorted by
// extension. Each line is "ext -> fence (line "prefix")",
// "ext -> fence (block "open" "close")", or both joined with " or " for
// dual-form entries.
func Describe(table map[string]Language) []string {
	exts := make([]string, 0, len(table))
	for ext := range table {
		exts = append(exts, ext)
	}
	sort.Strings(exts)
	out := make([]string, 0, len(exts))
	for _, ext := range exts {
		l := table[ext]
		var parts []string
		if l.Form.Line != "" {
			parts = append(parts, fmt.Sprintf("line %q", l.Form.Line))
		}
		if l.Form.Block != [2]string{} {
			parts = append(parts, fmt.Sprintf("block %q %q", l.Form.Block[0], l.Form.Block[1]))
		}
		out = append(out, fmt.Sprintf("%s -> %s (%s)", ext, l.Fence, strings.Join(parts, " or ")))
	}
	return out
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/lang/ -run TestDescribe -v`
Expected: PASS.

- [ ] **Step 5: Write the failing CLI tests**

Add to `internal/cli/cli_test.go`:

```go
func TestRunLanguages(t *testing.T) {
	var out, errb bytes.Buffer
	if code := Run([]string{"--languages"}, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("code %d stderr %s", code, errb.String())
	}
	if !strings.Contains(out.String(), `go -> go (line "//")`) {
		t.Fatalf("out:\n%s", out.String())
	}
	if !strings.Contains(out.String(), `html -> html (block "<!--" "-->")`) {
		t.Fatalf("out:\n%s", out.String())
	}
}

func TestRunLanguagesWithConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "c.yaml")
	if err := os.WriteFile(cfg, []byte("languages:\n  foo:\n    line: \";;\"\n    fence: foofence\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	if code := Run([]string{"--languages", "--config", cfg}, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("code %d stderr %s", code, errb.String())
	}
	if !strings.Contains(out.String(), `foo -> foofence (line ";;")`) {
		t.Fatalf("out:\n%s", out.String())
	}
}
```

- [ ] **Step 6: Run the CLI tests to verify they fail**

Run: `go test ./internal/cli/ -run TestRunLanguages -v`
Expected: FAIL — `--languages` is an unknown flag, so `Run` returns 2.

- [ ] **Step 7: Implement the flag**

In `internal/cli/cli.go`, add the flag registration after the `--config` line:

```go
	fs.StringVar(&opt.Config, "config", "", "path to config file")
	languages := fs.Bool("languages", false, "list supported languages and exit")
```

Then, immediately after the `fs.Parse` block (`if err := fs.Parse(args); err != nil { return 2 }`), insert:

```go
	if *languages {
		table := lang.Builtins()
		if opt.Config != "" {
			cfg, err := lang.LoadConfig(opt.Config)
			if err != nil {
				fmt.Fprintf(stderr, "codemd: %v\n", err)
				return 1
			}
			table, err = lang.Merge(lang.Builtins(), cfg)
			if err != nil {
				fmt.Fprintf(stderr, "codemd: %v\n", err)
				return 1
			}
		}
		for _, line := range lang.Describe(table) {
			fmt.Fprintln(stdout, line)
		}
		return 0
	}
```

- [ ] **Step 8: Run the CLI tests to verify they pass**

Run: `go test ./internal/cli/ -run TestRunLanguages -v`
Expected: PASS.

- [ ] **Step 9: Run the full suite and commit**

```bash
gofmt -l . && go vet ./... && go test ./...
git add internal/lang/lang.go internal/lang/lang_test.go internal/cli/cli.go internal/cli/cli_test.go
git commit -m "feat: --languages flag lists supported languages"
```

---

### Task 3: Update the spec and README

**Files:**
- Modify: `docs/superpowers/specs/2026-09-26-codemd-design.md`
- Modify: `README.md`

**Interfaces:**
- Consumes: the behavior from Tasks 1 and 2.
- Produces: documentation matching the code; no code change.

- [ ] **Step 1: Update the spec's reference grammar**

In `docs/superpowers/specs/2026-09-26-codemd-design.md`, change the grammar line:

```
[codemd]:# (MODE RANGE PATH [LANG] [strip])
```

to:

```
[codemd]:# (MODE RANGE PATH [LANG] [strip] ["LINK-TEXT"])
```

Then add this bullet immediately after the **strip** bullet:

```markdown
- **LINK-TEXT**: link mode only. A double-quoted token after PATH sets the
  rendered link label; it may contain spaces. It must not contain `]`. A quoted
  token in import mode, an empty link text, more than one link text, an
  unterminated quote, or a quoted PATH is an error.
```

- [ ] **Step 2: Update the spec's link rendering and non-goals**

In the `### Link rendering` section, add this bullet after the HTTP-source bullet:

```markdown
- The label is LINK-TEXT when present, otherwise `PATH:LINE`.
```

In the `## Non-Goals (future objectives)` list, add:

```markdown
- Inline references: a reference that appears mid-line and renders inline.
```

- [ ] **Step 3: Update the spec's CLI and sources sections**

In the `## CLI` section, add this bullet after the `--config` bullet:

```markdown
- `--languages`: print the supported extension → fence table and exit.
```

In the `## Sources and languages` section, add this bullet at the end of the list (after the unknown-extensions bullet):

```markdown
- `--languages` prints the effective table (built-ins plus `--config`), sorted by
  extension.
```

- [ ] **Step 4: Update the README grammar and flags**

In `README.md`, change the grammar code block line:

```
[codemd]:# (MODE RANGE PATH [LANG] [strip])
```

to:

```
[codemd]:# (MODE RANGE PATH [LANG] [strip] ["LINK-TEXT"])
```

Add this bullet after the **strip** bullet:

```markdown
- **LINK-TEXT**: `link` only; a quoted token after PATH sets the link label. It
  may contain spaces and must not contain `]`.
```

In the Flags table, add this row after the `--config` row:

```markdown
| `--languages` | List supported languages and exit. |
```

- [ ] **Step 5: Add a "Supported languages" section to the README**

Insert this section immediately before `## Config`:

```markdown
## Supported languages

The built-in table maps an extension to a comment form and a fence name.
`--languages` prints the effective table (including any `--config` additions).

| Extension | Fence | Comment form |
| --- | --- | --- |
| `go` | `go` | line `//` |
| `js`, `mjs` | `javascript` | line `//` |
| `ts` | `typescript` | line `//` |
| `tsx` | `tsx` | line `//` |
| `jsx` | `jsx` | line `//` |
| `c`, `h` | `c` | line `//` or block `/* */` |
| `cpp`, `hpp` | `cpp` | line `//` or block `/* */` |
| `java` | `java` | line `//` or block `/* */` |
| `rs` | `rust` | line `//` or block `/* */` |
| `php` | `php` | line `//` or block `/* */` |
| `py` | `python` | line `#` |
| `rb` | `ruby` | line `#` |
| `sh`, `bash` | `bash` | line `#` |
| `yaml`, `yml` | `yaml` | line `#` |
| `toml` | `toml` | line `#` |
| `lua` | `lua` | line `--` |
| `sql` | `sql` | line `--` |
| `html` | `html` | block `<!-- -->` |
| `xml` | `xml` | block `<!-- -->` |
| `md` | `markdown` | block `<!-- -->` |
| `css` | `css` | block `/* */` |

Any other extension still works: markers are recognized by trying the generic
comment forms (`//`, `#`, `/* */`, `<!-- -->`), and the fence falls back to
`text`. Use the explicit LANG token or a `.codemd.yaml` entry to override the
fence or comment form.
```

- [ ] **Step 6: Verify and commit**

Run:
```bash
gofmt -l . && go vet ./... && go test ./...
git add docs/superpowers/specs/2026-09-26-codemd-design.md README.md
git commit -m "docs: document link text, --languages, and language table"
```

Expected: all commands clean.

---

## Self-Review

**Spec coverage:**
- Custom link text grammar, parsing, rendering, and errors → Task 1.
- `--languages` flag and language table rendering → Task 2.
- Built-in language list documented in README → Task 3.
- Inline references recorded as a future objective → Task 3.
- Item 2 (stdin behavior) requires no change and is not in this plan.

**Placeholder scan:** no TBD/TODO; every code and test step contains full code; every expected value is concrete.

**Type consistency:** `mdref.Ref.Label` is defined in Task 1 and consumed by `resolveOne` in Task 1; `lang.Describe(map[string]Language) []string` is defined in Task 2 and consumed by `cli.Run` in Task 2; the grammar string `["LINK-TEXT"]` is used consistently in Task 1 and Task 3; the flag name `--languages` and the `languages` local are consistent within Task 2.
