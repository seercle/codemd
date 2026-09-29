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

<!-- codemd: (import .. ../testdata/console/err-mutually-exclusive/transcript.console console) -->
```console
$ codemd -w -d doc.md
codemd: -w, -o, -d and --check are mutually exclusive
```

Pick one output mode.

**`-o requires exactly one input file`.** `-o` needs one input, so it fails
with zero inputs (stdin) or with more than one:

<!-- codemd: (import .. ../testdata/console/err-output-needs-one/transcript.console console) -->
```console
$ printf 'x\n' | codemd -o out.md
codemd: -o requires exactly one input file
```

Pass exactly one file.

**`in-place flags require an input file`.** `-w`, `-o`, and `-d` need a file to
read; with no file arguments codemd reads stdin and cannot write in place:

<!-- codemd: (import .. ../testdata/console/err-inplace-needs-file/transcript.console console) -->
```console
$ printf 'x\n' | codemd -w
codemd: in-place flags require an input file
```

Supply a file argument, or drop the in-place flag and capture stdout instead.

**`no files match %q`.** A glob argument expanded to nothing:

<!-- codemd: (import .. ../testdata/console/err-no-files-match/transcript.console console) -->
```console
$ codemd 'no-such-*.md'
codemd: no files match "no-such-*.md"
```

Check the pattern, or quote it so your shell passes it through unchanged.

**`no Markdown files in %q`.** A directory argument contains no `.md` or
`.markdown` files. Only directories whose name starts with `.` are skipped, so
hidden Markdown files are still processed; this error means the directory holds
no Markdown files at any visited depth:

<!-- codemd: (import .. ../testdata/console/err-no-md-in-dir/transcript.console console) -->
```console
$ codemd docs
codemd: no Markdown files in "docs"
```

Point codemd at a directory that contains Markdown files.

**`<argument>: stat <argument>: no such file or directory`.** An explicit path
does not exist:

<!-- codemd: (import .. ../testdata/console/err-explicit-missing/transcript.console console) -->
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

<!-- codemd: (import .. ../testdata/console/err-reference-too-short/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: reference too short: "()"
<!-- codemd: () -->
codemd: 1 error(s)
```

Supply a mode and a range.

**`unknown mode %q`.** The first token is neither `import` nor `link`:

<!-- codemd: (import .. ../testdata/console/err-unknown-mode/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: unknown mode "bogus"
<!-- codemd: (bogus a) -->
codemd: 1 error(s)
```

Use `import` or `link`.

**`import range must contain '..': %q`.** An `import` range is a single token
with no `..`:

<!-- codemd: (import .. ../testdata/console/err-import-range/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: import range must contain '..': "a"
<!-- codemd: (import a src.go go) -->
codemd: 1 error(s)
```

Write two bounds joined by `..`, such as `a..b`.

**`link takes a single token, got %q`.** A `link` range contains `..`:

<!-- codemd: (import .. ../testdata/console/err-link-token/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: link takes a single token, got "a..b"
<!-- codemd: (link a..b src.go go) -->
codemd: 1 error(s)
```

A link targets one line; pass a single named point or regex.

**`missing path in %q`.** The reference has a mode and range but no path:

<!-- codemd: (import .. ../testdata/console/err-missing-path/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: missing path in "(import a..b)"
<!-- codemd: (import a..b) -->
codemd: 1 error(s)
```

Add a source path or URL after the range.

**`path must not be quoted in %q`.** The path token is double-quoted:

<!-- codemd: (import .. ../testdata/console/err-quoted-path/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: path must not be quoted in "(import a..b \"src.go\")"
<!-- codemd: (import a..b "src.go") -->
codemd: 1 error(s)
```

Drop the quotes. See also [the FAQ on paths with spaces](#faq).

**`unterminated regex %q`.** A regex token begins with `/` but has no closing
`/`:

<!-- codemd: (import .. ../testdata/console/err-unterminated-regex/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: unterminated regex "/foo src.go"
<!-- codemd: (link /foo src.go) -->
codemd: 1 error(s)
```

Close the regex, and escape a literal slash as `\/`.

**`strip requires a regex token in %q`.** `strip` is present but neither range
bound is a regex:

<!-- codemd: (import .. ../testdata/console/err-strip-needs-regex/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: strip requires a regex token in "(import a..b src.go strip)"
<!-- codemd: (import a..b src.go strip) -->
codemd: 1 error(s)
```

Give at least one bound as a `/regex/`, or remove `strip`.

**`link text is only valid for link mode in %q`.** A quoted label follows a
path in `import` mode:

<!-- codemd: (import .. ../testdata/console/err-link-text-import/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: link text is only valid for link mode in "(import a..b src.go \"x\")"
<!-- codemd: (import a..b src.go "x") -->
codemd: 1 error(s)
```

Use `link` mode for a custom label, or remove the label.

**`empty link text in %q`.** The label is empty or whitespace only:

<!-- codemd: (import .. ../testdata/console/err-empty-link-text/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: empty link text in "(link a src.go \"\")"
<!-- codemd: (link a src.go "") -->
codemd: 1 error(s)
```

Put non-blank text in the quotes, or omit the label.

**`multiple link labels in %q`.** More than one quoted label is present:

<!-- codemd: (import .. ../testdata/console/err-multiple-labels/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: multiple link labels in "(link a src.go \"x\" \"y\")"
<!-- codemd: (link a src.go "x" "y") -->
codemd: 1 error(s)
```

Keep a single label.

**`unexpected token %q in %q`.** A token appears after the optional `LANG`
position; codemd has nowhere left to put it:

<!-- codemd: (import .. ../testdata/console/err-unexpected-token/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: unexpected token "extra" in "(import a..b src.go go extra)"
<!-- codemd: (import a..b src.go go extra) -->
codemd: 1 error(s)
```

Remove the extra token, or fold it into a valid position.

**`unterminated quoted string in %q`.** A double quote is not closed:

<!-- codemd: (import .. ../testdata/console/err-unterminated-quote/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: unterminated quoted string in "(import a..b src.go \"x)"
<!-- codemd: (import a..b src.go "x) -->
codemd: 1 error(s)
```

Close the quote.

## Resolution errors (exit 1)

These appear when a well-formed reference cannot be resolved. The message is
prefixed with `codemd: <file>: line <n>: `, where the line is the reference
comment.

**`marker %q not found at or after line %d`.** No source marker with that name
exists at or after the search position:

<!-- codemd: (import .. ../testdata/console/err-marker-missing/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: marker "foo" not found at or after line 1
<!-- codemd: (import foo..bar src.go go) -->
codemd: 1 error(s)
```

Check the marker name and spelling (see
[Source markers](source-markers.md)), or widen the range.

**`regex %q matched no line at or after %d`.** The line regex matched nothing:

<!-- codemd: (import .. ../testdata/console/err-regex-no-match/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: regex "nomatch" matched no line at or after 1
<!-- codemd: (link /nomatch/ src.go go) -->
codemd: 1 error(s)
```

Adjust the regex, remembering it matches a whole line.

**`bad regex %q: <regexp error>`.** The regex does not compile:

<!-- codemd: (import .. ../testdata/console/err-bad-regex/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: bad regex "[": error parsing regexp: missing closing ]: `[`
<!-- codemd: (link /[/ src.go go) -->
codemd: 1 error(s)
```

Fix the Go regular expression.

**`empty range a..b`.** The two bounds select no lines: named markers with
nothing between them, or the same marker on both ends:

<!-- codemd: (import .. ../testdata/console/err-empty-range/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: empty range a..b
<!-- codemd: (import a..b src.go go) -->
codemd: 1 error(s)
```

**`open <path>: no such file or directory`.** The reference's source file does
not exist relative to the Markdown file's directory:

<!-- codemd: (import .. ../testdata/console/err-open-missing/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: open no-such.go: no such file or directory
<!-- codemd: (import a..b no-such.go go) -->
codemd: 1 error(s)
```

Fix the path or add the file.

**`<url>: HTTP <code>`.** The remote source returned a non-200 status:

<!-- codemd: (import .. ../testdata/console/err-http-404/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: https://raw.githubusercontent.com/golang/go/a10e42f219abb9c5bc4e7d86d9464700a42c7d57/src/errors/does-not-exist.txt: HTTP 404
<!-- codemd: (link a https://raw.githubusercontent.com/golang/go/a10e42f219abb9c5bc4e7d86d9464700a42c7d57/src/errors/does-not-exist.txt go) -->
codemd: 1 error(s)
```

Check the URL and that the file is reachable. Network failures surface the
underlying error instead, for example
`Get "http://...": dial tcp ...: connect: connection refused`.

## Marker errors (exit 1)

**`duplicate marker %q on lines %d and %d`.** A source file defines the same
marker name twice:

<!-- codemd: (import .. ../testdata/console/err-duplicate-marker/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: duplicate marker "a" on lines 3 and 6
<!-- codemd: (import a..b src.go go) -->
codemd: 1 error(s)
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

<!-- codemd: (import .. ../testdata/console/err-config-yaml/transcript.console console) -->
```console
$ codemd --config bad.yaml doc.md
codemd: bad.yaml: yaml: did not find expected key
```

Fix the YAML syntax.

**`<path>: language %q defines both line and block`.** A language entry sets
both `line` and a non-empty `block`:

<!-- codemd: (import .. ../testdata/console/err-config-both/transcript.console console) -->
```console
$ codemd --config both.yaml doc.md
codemd: both.yaml: language "foo" defines both line and block
```

Keep exactly one comment form.

**`<path>: language %q must define exactly one of line or block`.** A language
entry sets neither `line` nor a two-element `block`:

<!-- codemd: (import .. ../testdata/console/err-config-neither/transcript.console console) -->
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

<!-- codemd: (import .. ../testdata/console/err-flag-after-file/transcript.console console) -->
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
