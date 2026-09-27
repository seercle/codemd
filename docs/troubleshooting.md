# Troubleshooting

This page explains codemd's exit codes and every error message it can print,
with the cause of each and how to fix it. It assumes the flags and inputs from
the [Command-line reference](cli-reference.md) and the reference-comment
grammar from [Reference syntax](reference-syntax.md).

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Success. For `--check`, nothing was out of date. |
| `1` | One or more reference, config, or I/O errors; or `--check` found changes. |
| `2` | Usage error: bad flags, mutually exclusive flags, `-o` with other than one input, `-w`/`-o`/`-d` with no file input, or an input argument that matches nothing. |

Usage messages are printed to standard error; for an unknown flag the flag
parser prints its message and then the usage text. Document errors are printed
one per line as `codemd: <message>`, or `codemd: <name>: line <n>: <message>`
when the error belongs to a reference on a specific line. A run that collects
document errors prints a final `codemd: N error(s)` line and exits `1`; a config
load failure prints only its message and exits `1`.

## Usage errors (exit 2)

**Unknown flag.** Go's flag parser rejects the flag before codemd runs, prints
`flag provided but not defined: <flag>`, prints the usage text, and exits `2`.
Use `codemd --help` to list the flags.

**Mutually exclusive flags.** `-w`, `-o`, `-d`, and `--check` may not be
combined:

```console
$ codemd -w -d doc.md
codemd: -w, -o, -d and --check are mutually exclusive
```

Pick one output mode.

**`-o requires exactly one input file`.** `-o` needs one input, so it fails
with zero inputs (stdin) or with more than one:

```console
$ printf 'x\n' | codemd -o out.md
codemd: -o requires exactly one input file
```

Pass exactly one file.

**`in-place flags require an input file`.** `-w`, `-o`, and `-d` need a file to
read; with no file arguments codemd reads stdin and cannot write in place:

```console
$ printf 'x\n' | codemd -w
codemd: in-place flags require an input file
```

Supply a file argument, or drop the in-place flag and capture stdout instead.

**`no files match %q`.** A glob argument expanded to nothing:

```console
$ codemd 'no-such-*.md'
codemd: no files match "no-such-*.md"
```

Check the pattern, or quote it so your shell passes it through unchanged.

**`no Markdown files in %q`.** A directory argument contains no `.md` or
`.markdown` files. Only directories whose name starts with `.` are skipped, so
hidden Markdown files are still processed; this error means the directory holds
no Markdown files at any visited depth:

```console
$ codemd docs
codemd: no Markdown files in "docs"
```

Point codemd at a directory that contains Markdown files.

**`<argument>: stat <argument>: no such file or directory`.** An explicit path
does not exist:

```console
$ codemd missing.md
codemd: missing.md: stat missing.md: no such file or directory
```

Correct the path.

## Reference parse errors (exit 1)

A malformed reference comment fails on its own line and the run continues with
the remaining references. The message is prefixed with
`codemd: <file>: line <n>: `.

**`reference too short: %q`.** The reference has no mode, or no range token:

```console
$ codemd doc.md
codemd: doc.md: line 1: reference too short: "()"
```

Supply a mode and a range.

**`unknown mode %q`.** The first token is neither `import` nor `link`:

```console
$ codemd doc.md
codemd: doc.md: line 1: unknown mode "bogus"
```

Use `import` or `link`.

**`import range must contain '..': %q`.** An `import` range is a single token
with no `..`:

```console
$ codemd doc.md
codemd: doc.md: line 1: import range must contain '..': "a"
```

Write two bounds joined by `..`, such as `a..b`.

**`link takes a single token, got %q`.** A `link` range contains `..`:

```console
$ codemd doc.md
codemd: doc.md: line 1: link takes a single token, got "a..b"
```

A link targets one line; pass a single named point or regex.

**`missing path in %q`.** The reference has a mode and range but no path:

```console
$ codemd doc.md
codemd: doc.md: line 1: missing path in "(import a..b)"
```

Add a source path or URL after the range.

**`path must not be quoted in %q`.** The path token is double-quoted:

```console
$ codemd doc.md
codemd: doc.md: line 1: path must not be quoted in "(import a..b \"src.go\")"
```

Drop the quotes. See also [the FAQ on paths with spaces](#faq).

**`unterminated regex %q`.** A regex token begins with `/` but has no closing
`/`:

```console
$ codemd doc.md
codemd: doc.md: line 1: unterminated regex "/foo src.go"
```

Close the regex, and escape a literal slash as `\/`.

**`strip requires a regex token in %q`.** `strip` is present but neither range
bound is a regex:

```console
$ codemd doc.md
codemd: doc.md: line 1: strip requires a regex token in "(import a..b src.go strip)"
```

Give at least one bound as a `/regex/`, or remove `strip`.

**`link text is only valid for link mode in %q`.** A quoted label follows a
path in `import` mode:

```console
$ codemd doc.md
codemd: doc.md: line 1: link text is only valid for link mode in "(import a..b src.go \"x\")"
```

Use `link` mode for a custom label, or remove the label.

**`empty link text in %q`.** The label is empty or whitespace only:

```console
$ codemd doc.md
codemd: doc.md: line 1: empty link text in "(link a src.go \"\")"
```

Put non-blank text in the quotes, or omit the label.

**`multiple link labels in %q`.** More than one quoted label is present:

```console
$ codemd doc.md
codemd: doc.md: line 1: multiple link labels in "(link a src.go \"x\" \"y\")"
```

Keep a single label.

**`unexpected token %q in %q`.** A token appears after the optional `LANG`
position; codemd has nowhere left to put it:

```console
$ codemd doc.md
codemd: doc.md: line 1: unexpected token "extra" in "(import a..b src.go go extra)"
```

Remove the extra token, or fold it into a valid position.

**`unterminated quoted string in %q`.** A double quote is not closed:

```console
$ codemd doc.md
codemd: doc.md: line 1: unterminated quoted string in "(import a..b src.go \"x)"
```

Close the quote.

## Resolution errors (exit 1)

These appear when a well-formed reference cannot be resolved. The message is
prefixed with `codemd: <file>: line <n>: `, where the line is the reference
comment.

**`marker %q not found at or after line %d`.** No source marker with that name
exists at or after the search position:

```console
$ codemd doc.md
codemd: doc.md: line 1: marker "foo" not found at or after line 1
```

Check the marker name and spelling (see
[Source markers](source-markers.md)), or widen the range.

**`regex %q matched no line at or after %d`.** The line regex matched nothing:

```console
$ codemd doc.md
codemd: doc.md: line 1: regex "nomatch" matched no line at or after 1
```

Adjust the regex, remembering it matches a whole line.

**`bad regex %q: <regexp error>`.** The regex does not compile:

```console
$ codemd doc.md
codemd: doc.md: line 1: bad regex "[": error parsing regexp: missing closing ]: `[`
```

Fix the Go regular expression.

**`open <path>: no such file or directory`.** The reference's source file does
not exist relative to the Markdown file's directory:

```console
$ codemd doc.md
codemd: doc.md: line 1: open no-such.go: no such file or directory
```

Fix the path or add the file.

**`<url>: HTTP <code>`.** The remote source returned a non-200 status:

```console
$ codemd doc.md
codemd: doc.md: line 1: http://127.0.0.1:8099/missing.txt: HTTP 404
```

Check the URL and that the file is reachable. Network failures surface the
underlying error instead, for example
`Get "http://...": dial tcp ...: connect: connection refused`.

## Marker errors (exit 1)

**`duplicate marker %q on lines %d and %d`.** A source file defines the same
marker name twice:

```console
$ codemd doc.md
codemd: doc.md: line 1: duplicate marker "foo" on lines 1 and 3
```

Marker names must be unique within a source file. Rename one of them.

## Status and write messages

These lines describe what a run did rather than a malformed input. They are
printed to standard error.

**`<file> is out of date`** or **`<stdin> is out of date`** — `--check` found a
difference; the run exits `1`. Run `codemd -w <file>` to update the file.

**`<file>: not written due to errors (use --force to write anyway)`** — `-w` or
`-o` skipped a file because one or more of its references failed. Fix the
references, or pass `--force` to write the partial result.

**`N file(s) checked, M out of date`** / **`N file(s) checked, M updated`** —
the summary printed after a `--check` or `-w` run over file inputs.

**`N error(s)`** — the number of errors collected during the run. Any nonzero
count means the exit code is `1`.

An I/O failure while reading or writing a file prints the underlying OS error,
such as `codemd: open out.md: permission denied`. Check the file's permissions.

## Config errors (exit 1)

A `--config` file is loaded before any input is processed, so a bad config
fails the whole run. Discovered `.codemd.yaml` files produce the same messages
with the discovered path. The message is prefixed with `codemd: `.

**`<path>: yaml: <yaml error>`.** The config is not valid YAML:

```console
$ codemd --config bad.yaml doc.md
codemd: bad.yaml: yaml: line 1: did not find expected node content
```

Fix the YAML syntax.

**`<path>: language %q defines both line and block`.** A language entry sets
both `line` and a non-empty `block`:

```console
$ codemd --config both.yaml doc.md
codemd: both.yaml: language "foo" defines both line and block
```

Keep exactly one comment form.

**`<path>: language %q must define exactly one of line or block`.** A language
entry sets neither `line` nor a two-element `block`:

```console
$ codemd --config neither.yaml doc.md
codemd: neither.yaml: language "foo" must define exactly one of line or block
```

Add a `line` or a `block` with exactly two strings.

**`open <path>: no such file or directory`.** The `--config` path does not
exist. Correct the path, or omit `--config` to use discovery.

See [Configuration](configuration.md) for the config format.

## FAQ

**Why does re-running codemd produce no changes?** Resolution is idempotent:
each run re-derives the managed region from the reference comment, so once a
document is up to date, further runs leave it byte-for-byte unchanged. A clean
re-run is the expected result.

**Why does `--check` say a file is out of date?** The resolved output differs
from the file on disk. Run `codemd -w <file>` to update it, then commit the
resulting file. `--check` exits `1` when any input differs.

**Why is my reference ignored?** A reference inside a fenced code block is not
scanned, so it is treated as literal text. Place the reference outside the
fences, or remove the surrounding fence.

**Why can't I reference a path with spaces?** Tokens are whitespace-separated,
and a quoted path is rejected (`path must not be quoted`). Rename the file or
directory to remove the spaces.

**Why are my flags treated as a path?** Go's flag parser stops at the first
non-flag argument, so any flag after a positional file is read as another input
path:

```console
$ codemd doc.md --check
codemd: --check: stat --check: no such file or directory
```

Put all flags before the positional arguments: `codemd --check doc.md`.

**Does codemd change my line endings or add a newline?** No. codemd detects the
document's line ending (LF or CRLF) and reuses it, and it preserves a trailing
newline exactly: a file without a final newline stays without one.

**Do HTTP sources get retried or cached?** Each URL is fetched at most once per
run and cached in memory, so repeated references to the same URL share one
request. The HTTP client uses a 30-second timeout; a request that exceeds it
fails that reference.

## Where next

- [Command-line reference](cli-reference.md) — flags, inputs, output modes, and
  exit codes.
- [Reference syntax](reference-syntax.md) — the complete reference-comment
  grammar.
- [Configuration](configuration.md) — language entries and discovery.
