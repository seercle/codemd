# codemd — Design

## Summary

`codemd` is a Go CLI that resolves references embedded in Markdown comments. A
reference either **imports** a code snippet from a source file (local or HTTP)
or generates a **clickable link** to a specific line in that source. The
reference comment is preserved so the operation is idempotent: re-running the
tool refreshes imported snippets and link line numbers.

## Goals

- Import code snippets from any text source file into Markdown.
- Generate clickable links to source lines (relative links for local files,
  HTTP links for HTTP sources).
- Define snippet boundaries in source files with line-based comment markers.
- Support regex-based boundaries for sources that cannot be edited.
- Work in place, to a new file, as a diff, or as a CI check.
- Support many languages out of the box, extensible via a config file.

## Non-Goals (future objectives)

- SSH sources.
- Range links (`link a..b`); link mode supports a single point only.
- GitHub commit-SHA permalinks.
- Markers spanning multiple lines in block-comment languages.
- Inline references: a reference that appears mid-line and renders inline.

## Vocabulary

### Source markers

Markers are named points, declared as comments fully contained on one line:

```go
type lv int //codemd:lv-def
func (l lv) double() lv { return l * 2 }
//codemd:lv-end
```

- Marker text is the fixed literal `codemd:` followed by a name matching
  `[A-Za-z0-9_.-]+` that does not contain `..` (which would be ambiguous with
  the range separator).
- A marker may trail code on the same line.
- Marker lines are **always excluded** from imported content.
- Marker lines **are** link targets: link mode points at the marker's own line.
- Duplicate marker names within one source file are an error.
- There is no `begin`/`end` keyword and no pairing: any two points may delimit a
  range.

### Markdown reference

A Markdown link-reference definition that is never referenced, so renderers
(including GitHub) hide it. It sits on its own line and manages the content
directly below it, skipping blank lines.

Import mode — replaces the fenced code block directly below when one is
present, otherwise inserts the generated fence directly below the comment:

````
[codemd]:# (import lv-def..lv-end src/server.go go)
```go
...generated...
```
````

Link mode — inserts the generated link directly below the comment, replacing an
existing generated link in place:

```
[codemd]:# (link lv-def src/server.go go)
[src/server.go:13](src/server.go#L13)
```

The generated fence uses the reference's LANG token when present, otherwise the
language resolved from the source extension. The fence is always a triple
backtick fence with the language info string.

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

## Reference grammar

```
[codemd]:# (MODE RANGE PATH [LANG] [strip] ["LINK-TEXT"])
```

- **MODE**: `import` or `link`.
- **RANGE**:
  - Import: exactly two tokens joined by `..`. Each token is a named point
    (`func1`) or a line regex (`/re/`). Open ranges are allowed: `a..` (to end
    of file), `..b` (from start of file). A single token without `..` is an
    error for import.
  - Link: exactly one token (name or `/re/`).
  - Mixing named points and regexes in one range is allowed
    (`func1../end/`, `/start/..func2`).
  - A token is a regex iff it starts with `/`; otherwise it is a name. A regex
    ends at the first unescaped `/` after its opening `/`; a literal `/` inside
    a regex is written `\/`. Parsing is left to right: read the left token (if
    it starts with `/`, consume through its closing `/`), then require `..`,
    then read the right token the same way. This allows regexes containing `..`
    (e.g. `/a..b/../c/`).
- **PATH**: a local path or `http(s)://` URL. Local paths are resolved relative
  to the directory of the Markdown file. Paths containing spaces are not
  supported (tokens are whitespace-separated).
- **LANG**: optional fence language. Falls back to the extension table.
- **strip**: optional modifier, only meaningful for regex tokens. Removes the
  matched substring from the boundary lines (Go leftmost match). If a boundary
  line becomes empty (whitespace only) after stripping, it is dropped. `strip`
  with no regex token is an error.
- **LINK-TEXT**: link mode only. A double-quoted token after PATH sets the
  rendered link label; it may contain spaces. It must not contain `]`. A quoted
  token in import mode, an empty link text, more than one link text, an
  unterminated quote, or a quoted PATH is an error.

### Link rendering

- Local source: label `PATH:LINE`, target `PATH#LLINE` (relative to the
  Markdown file's directory).
- HTTP source: label `PATH:LINE`, target the source URL with `#LLINE` appended.
- The label is LINK-TEXT when present, otherwise `PATH:LINE`.
- Single line only; the anchor is `#L<line>`.

### Range semantics

- Named points are excluded from imported content.
- Regex-matched boundary lines are included (embedmd-style).
- Start is located first; end is located at or after the start match.
- Line numbers are 1-based.
- An unmatched token is an error for that reference.

## Markdown scanning

Line-based text scanner (no Markdown AST). It tracks fenced-code state (``` and
~~~ fences, leading indentation ≤ 3 spaces, CommonMark-ish) so that reference
comments inside code blocks are ignored. All content outside managed regions is
preserved byte-for-byte.

Line endings: the dominant line ending of each file (`\n` or `\r\n`) and the
presence of a trailing newline are detected and reproduced on output. Files with
mixed line endings are normalized to the dominant ending.

## Sources and languages

- Local files are read from disk; `http(s)://` sources are fetched with a plain
  GET via `net/http`.
- Built-in table (~20 common languages) maps extension → comment form, canonical
  fence name, and aliases:
  - `line` form: a prefix terminated by end-of-line (e.g. `go: //`,
    `python: #`).
  - `block` form: a prefix/suffix pair (e.g. `html: <!--` / `-->`,
    `c: /*` / `*/`).
- Config file `.codemd.yaml`, discovered by walking up from the Markdown file,
  overridable with `--config`. It adds or overrides extensions, comment forms,
  fence names, and aliases. A config entry keyed by an extension overrides the
  built-in entry for that extension; a new key adds an extension. Each entry
  must define exactly one of `line` or `block`.
- Unknown extensions fall back to trying generic comment prefixes per line
  (`//`, `#`, `/* */`, `<!-- -->`). A marker is recognized when the trimmed
  line, after removing the comment form, equals `codemd:NAME`.
- `--languages` prints the effective table (built-ins plus `--config`), sorted by
  extension.

Example config:

```yaml
languages:
  go:
    line: "//"
    fence: go
  python:
    line: "#"
    fence: python
    aliases: [py]
  html:
    block: ["<!--", "-->"]
    fence: html
```

## CLI

Single command: `codemd [flags] file.md...`

- default: resolved output to stdout
- `-w`: write in place
- `-o out.md`: write to a new file (requires exactly one input file)
- `-d`: print a unified diff
- `--check`: exit non-zero if any file would change (CI)
- `--config path`: explicit config file
- `--languages`: print the supported extension → fence table and exit.

`-w`, `-o`, `-d`, and `--check` are mutually exclusive. With no input files,
input is read from stdin and written to stdout (in-place flags are then
invalid); config discovery and relative-path resolution then start from the
current working directory.

## Errors

Collect-and-continue: every reference error is recorded with file, line, and
reason; remaining references and files still process. A summary is printed to
stderr, and the process exits non-zero if any error occurred. On error, the
managed region is left untouched. Sources are fetched once per run (cache).

## Package layout

```
cmd/codemd/          main
internal/cli/        flags, orchestration
internal/mdref/      markdown scan + reference parsing
internal/srcfile/    source loading (local/http) + marker parsing
internal/extract/    range resolution + strip
internal/lang/       builtin table + YAML merge
internal/render/     snippet/link rendering
```

## Testing

- Table-driven unit tests per package.
- Golden-file integration tests running the CLI against fixture directories.
- Verification: `gofmt`, `go vet`, `go test ./...`.
