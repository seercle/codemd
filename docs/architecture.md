# Architecture

This page is the design record for codemd. It describes how the current
implementation is organized and how a Markdown document flows through it. For
the reference-comment grammar itself, see [Reference syntax](reference-syntax.md).

## Pipeline

Resolving a document is a single pass:

1. Read the input into a string.
2. `mdref.Scan` walks the document line by line and returns each reference
   comment with its 1-based line number, plus any syntax errors.
3. For each reference, `Resolver.resolveOne` loads the source, extracts source
   markers, resolves the range or link target, and renders the replacement.
4. `splice` rewrites the managed region for that reference.
5. `lineutil.Join` reassembles the lines using the detected line ending.

References are applied from the bottom up, so the line numbers recorded during
scanning stay valid as earlier replacements shift the text below them.

## Package layout

```text
cmd/codemd/          main
internal/cli/        flags, orchestration, input expansion, diff
internal/mdref/      markdown scan + reference parsing
internal/srcfile/    source loading (local/http) + marker parsing
internal/extract/    range resolution + strip
internal/lang/       builtin table + YAML merge
internal/render/     snippet/link rendering
internal/lineutil/   line splitting/joining
```

Each package has one job. `cli` wires the others together; the remaining
packages are independent of `cli` and each is testable in isolation. The builtin
language table and its configurable overrides are documented in
[Languages](languages.md) and [Configuration](configuration.md).

## Scanning

Scanning is line-based; codemd builds no Markdown AST. It tracks fenced code
blocks so that a reference comment inside a code block is ignored. A line opens
or closes a fence when it begins, after any leading whitespace, with three or
more backticks or three or more tildes. An opening fence is
closed only by a fence of the same character whose run is at least as long;
an unterminated fence runs to the end of the document.

Outside fenced blocks, a reference is recognized only when the trimmed line is
an HTML comment of the form `<!-- codemd: (...) -->`. Content outside managed
regions is left untouched, except for the line-ending normalization described
below.

## Managed regions

The managed region is the content directly below a reference comment. `splice`
skips blank lines to find the first non-blank line, then:

- `import`: if that line opens a marked fenced block, the whole region is
  replaced in place. If it opens an unmarked fenced block, that block is
  preserved (with the marked snippet inserted above it and a warning) or adopted
  by appending the marker, as described below. Otherwise the generated fence is
  inserted directly below the comment.
- `link`: if that line is a generated link, it is replaced; otherwise the
  generated link is inserted directly below the comment.

A generated `import` region is a fenced block followed by the marker
`<!-- codemd:generated -->`. A marked region is replaced in place. An unmarked
fenced block directly below an import reference is treated as hand-written:
`splice` leaves it in place, inserts the marked region above it, and returns a
warning. An unmarked block whose bytes equal the generated snippet is adopted
by appending the marker.

The reference line's leading whitespace is copied onto every non-blank
generated line, so a reference inside a list item keeps its snippet inside that
item.

A following reference comment is never consumed: codemd inserts below the
current comment and leaves the next reference in place. Because the reference
comment is preserved and the region is re-derived on every run, repeated runs
are idempotent. See [Reference syntax](reference-syntax.md) for the full rules.

## Line endings

`lineutil.Split` records the document's line ending and whether it ends with a
newline. A document with consistent LF or CRLF endings is reproduced with the
same ending, and the trailing-newline presence is preserved exactly. If a
document mixes LF and CRLF, the mixed endings are normalized: every line is
stripped of its `\r` and the file is rejoined with CRLF. Line numbers used in
references, ranges, and link anchors are 1-based.

## Sources and caching

A local `PATH` that is not absolute is resolved relative to the directory of
the Markdown file. An `http://` or `https://` path is fetched with a plain GET
using a client with a 30-second timeout; any response other than HTTP 200 is an
error for that reference.

Each source is loaded at most once per run. The loader keeps an in-memory cache
keyed by the resolved local path or the URL, so a document that references the
same file or URL many times reads or fetches it a single time.

## Errors

Error handling is collect-and-continue. A scan error or a failure while
resolving a reference records the file, the line, and the reason, and
processing continues with the remaining references and files. Errors are
reported in line order, and stdin is named `<stdin>`.

When a reference fails, its managed region is left untouched. The process exits
non-zero if any error occurred, and a `-w` or `-o` write is skipped for a file
with errors unless `--force` is given.

## Testing

Unit tests live beside each package and are table-driven, covering parsing,
extraction, rendering, line handling, and CLI behavior. Integration tests in
`internal/cli` call the `cli.Run` entry point in-process against fixtures under
`testdata` and compare against golden output; the HTTP paths are exercised with
`httptest`.

The verification gate is:

```bash
gofmt -l . && go vet ./... && go test ./...
```

The documentation pages in `docs/` are codemd documents: `TestDocsCurrent`
resolves them and fails if any page is stale, so the examples double as a
regression check on the tool.

## Non-goals

These are deliberate design boundaries, not gaps:

- SSH sources. The loader supports local paths and `http(s)://` only.
- GitHub commit-SHA permalinks. Link targets are `PATH#L<line>`, with no
  pinned revision.
- Multi-line markers in block-comment languages. Block comment markers must
  open and close on the same line.
- Inline references. A reference comment must be the whole line; references
  embedded in prose are ignored.

## Where next

- [Reference syntax](reference-syntax.md) — the reference-comment grammar and
  managed-region rules.
- [Languages](languages.md) — the builtin extension table, comment forms, and
  fences.
- [Configuration](configuration.md) — adding languages and discovery.
