# Command-line reference

This page is the authoritative reference for codemd's flags, inputs, output
modes, and exit codes. It uses the terms defined in
[Getting started](getting-started.md): **reference comment**, **import**,
**link**, **source marker**, **managed region**, and **idempotent**.

## Synopsis

```console-norun
codemd [flags] file.md...
codemd [flags]              (no files: read stdin, write stdout)
```

With one or more file arguments, codemd resolves each input and reports on it.
With no file arguments, it reads all of standard input and writes the result to
standard output.

Run `codemd --help` to print usage to standard output and exit `0`:

<!-- codemd: (import .. ../testdata/console/help/transcript.console console) -->
```console
$ codemd --help
codemd resolves code references embedded in Markdown.

Usage:
  codemd [flags] file.md...
  codemd [flags]              (no files: read stdin, write stdout)

Reference: <!-- codemd: (MODE RANGE PATH [LANG] [strip] ["LINK-TEXT"]) -->

Flags:
  -check
    	exit non-zero if any file would change
  -config string
    	path to config file
  -d	print a unified diff
  -f	write even if some references failed
  -force
    	write even if some references failed
  -languages
    	list supported languages and exit
  -o string
    	write result to a new file
  -version
    	print version and exit
  -w	write result in place
```

Go's flag syntax accepts a single or double dash, so `-w` and `--w`,
`-check` and `--check`, and so on are equivalent. The help listing prints the
single-dash form.

## Flags

| Flag | Description |
| --- | --- |
| `-w` | Write the result in place. |
| `-o out.md` | Write to a new file (exactly one input). |
| `-d` | Print a unified diff. |
| `--check` | Exit non-zero if any file would change (CI). |
| `--config path` | Use an explicit config file. |
| `-f`, `--force` | Write even if some references failed. |
| `--languages` | List supported languages and exit. |
| `--version` | Print the version and exit. |

An unknown flag is a usage error (exit `2`). Unlike `--help`, an unknown flag
prints its usage message to standard error.

## Mutually exclusive flags

`-w`, `-o`, `-d`, and `--check` are mutually exclusive. Passing more than one
is a usage error (exit `2`) with the message:

```text
codemd: -w, -o, -d and --check are mutually exclusive
```

`--config`, `-f`/`--force`, `--languages`, and `--version` can be combined with
the modes above as documented; `--languages` and `--version` short-circuit
before the mode checks.

## Inputs

Flags may appear before or after positional arguments; `--` ends flag parsing.

Positional arguments may be files, directories, or globs.

- **Files** are used as given.
- **Directories** are walked recursively for files whose extension is `.md` or
  `.markdown` (case-insensitive). Directories whose name starts with `.` are
  skipped, and the results are sorted.
- **Globs** are expanded when the argument contains `*`, `?`, or `[`. Globs
  support `**` to match any number of whole path segments, and cleaned-relative
  prefixes work: `codemd './**/*.md'` is equivalent to `codemd '**/*.md'`.
  Unlike directory walks, glob expansion does not skip hidden directories.

Arguments are de-duplicated in order after cleaning: the first occurrence of a
path wins, so `codemd a.md a.md ./a.md` processes `a.md` once. An argument that
matches nothing is a usage error (exit `2`):

<!-- codemd: (import .. ../testdata/console/err-no-files-match/transcript.console console) -->
```console
$ codemd 'no-such-*.md'
codemd: no files match "no-such-*.md"
```

The same applies to a directory that contains no Markdown files
(`codemd: no Markdown files in "dir"`) and to an explicit path that does not
exist (`codemd: path: stat path: no such file or directory`).

## Standard input

With no file arguments, codemd reads the whole of standard input, resolves it,
and writes the result to standard output. Config discovery and relative
source-path resolution start from the current working directory, and errors are
reported with the name `<stdin>`.

In-place flags (`-w`, `-o`, `-d`) require a file input; using them with stdin is
a usage error:

<!-- codemd: (import .. ../testdata/console/cli-stdin-inplace/transcript.console console) -->
```console
$ printf 'x\n' | codemd -w
codemd: in-place flags require an input file
$ printf 'x\n' | codemd -o out.md
codemd: -o requires exactly one input file
```

`--check` is valid with stdin for CI pipes; see below.

## Output modes

| Mode | Destination | Effect |
| --- | --- | --- |
| default | stdout | Resolved document(s) written to stdout; nothing is written to disk. |
| `-w` | the input files | Each file rewritten in place when its resolved output differs. |
| `-o out.md` | `out.md` | Result written to `out.md`; requires exactly one input. |
| `-d` | stdout | Unified diff per changed file: 3 lines of context, `a/` and `b/` headers. Nothing is written to disk. |
| `--check` | stderr | No document is written; prints status and exits non-zero if any input differs. |

With multiple inputs, the default mode concatenates each resolved document to
stdout in argument order, with no separator. `-d` emits a separate diff only for
each file that changed; unchanged files produce no output.

## `--check` and `-w` summaries

For file inputs, `--check` and `-w` print a summary line to standard error after
all files are processed:

```text
codemd: N file(s) checked, M out of date
codemd: N file(s) checked, M updated
```

`N` is the number of files processed and `M` is the number that differed
(`--check`) or were rewritten (`-w`). For example:

<!-- codemd: (import .. ../testdata/console/cli-check-summary/transcript.console console) -->
```console
$ codemd --check doc.md
codemd: doc.md is out of date
codemd: 1 file(s) checked, 1 out of date
codemd: 1 error(s)
$ codemd -w doc.md
codemd: 1 file(s) checked, 1 updated
$ codemd -w doc.md
codemd: 1 file(s) checked, 0 updated
```

The file-count summary is not printed for stdin. A stdin `--check` that differs
prints the per-input line `codemd: <stdin> is out of date`, counts it as an
error, and exits `1`:

<!-- codemd: (import .. ../testdata/console/cli-stdin-check/transcript.console console) -->
```console
$ codemd --check < doc.md
codemd: <stdin> is out of date
codemd: 1 error(s)
```

When a file `--check` finds changes, it prints `codemd: <file> is out of date`
before the summary, counts it as an error, and exits `1`.

## Failure and `--force`

Writing with `-w` and `-o` is all-or-nothing per file: if any reference in a
file fails to resolve, that file is not written, while other files still
process. The failure is reported and the file is left untouched:

<!-- codemd: (import .. ../testdata/console/cli-force-mixed/transcript.console console) -->
```console
$ codemd -w mixed.md
codemd: mixed.md: line 5: open missing.go: no such file or directory
codemd: mixed.md: not written due to errors (use --force to write anyway)
codemd: 1 file(s) checked, 0 updated
codemd: 1 error(s)
```

`--force` (or `-f`) bypasses this guard and writes the partial result when the
document still changed; it does not suppress the error or change the exit code,
which remains `1`. If the failed references leave the resolved document
identical to the input, there is nothing to write and the file stays as it was,
even with `--force`.

`-d` and default stdout output are unaffected by write failures: they still
print the partial result and exit `1`.

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Success. For `--check`, nothing was out of date. |
| `1` | One or more reference, config, or I/O errors; or `--check` found changes. |
| `2` | Usage error: bad flags, mutually exclusive flags, `-o` with other than one input, `-w`/`-o`/`-d` with no file input, or an input argument that matches nothing. |

## `--version`

Prints the version to stdout as `codemd <version>` and exits `0`. Source builds
report `dev`; release builds set the version at link time.

<!-- codemd: (import .. ../testdata/console/version/transcript.console console) -->
```console
$ codemd --version
codemd dev
```

## `--languages`

Prints the effective language table and exits `0`. When `--config` is given, its
entries are merged over the built-ins first. The table is sorted by extension and
each line is `ext -> fence (form)`:

```console-norun
$ codemd --languages
bash -> bash (line "#")
c -> c (line "//" or block "/*" "*/")
cpp -> cpp (line "//" or block "/*" "*/")
go -> go (line "//")
...
```

A `--config` that fails to load or merge is a config error (exit `1`) and prints
`codemd: <error>` to stderr.

## Where next

- [Reference syntax](reference-syntax.md) — ranges, markers, regexes, config,
  and language handling.
- [Recipes](recipes.md) — common patterns and worked examples.
