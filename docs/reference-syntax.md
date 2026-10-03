# Reference syntax

This page is the authoritative reference for the reference-comment grammar and
the managed-region rules. It uses the terms defined in
[Getting started](getting-started.md): **reference comment**, **import**,
**link**, **source marker**, **managed region**, and **idempotent**.

## Grammar

```text
<!-- codemd: (MODE RANGE PATH [LANG] [strip] ["LINK-TEXT"]) -->
```

A reference is an HTML comment. `<!-- codemd: (...) -->` is hidden by
renderers, so the reference does not appear in the rendered document. It must
sit on its own line; leading indentation is allowed. A reference inside a
fenced code block is ignored by the scanner.

## MODE

`MODE` is `import` or `link`. Any other value is an error:

<!-- codemd: (import .. ../testdata/console/err-unknown-mode/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: unknown mode "bogus"
<!-- codemd: (bogus a) -->
codemd: 1 error(s)
```
<!-- codemd:generated -->

## RANGE

For `import`, `RANGE` is exactly two tokens joined by `..`. A single token
without `..` is an error:

<!-- codemd: (import .. ../testdata/console/err-import-range/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: import range must contain '..': "a"
<!-- codemd: (import a src.go go) -->
codemd: 1 error(s)
```
<!-- codemd:generated -->

For `link`, `RANGE` is exactly one token; a `..` range is rejected:

<!-- codemd: (import .. ../testdata/console/err-link-token/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: link takes a single token, got "a..b"
<!-- codemd: (link a..b src.go go) -->
codemd: 1 error(s)
```
<!-- codemd:generated -->

A token is a **named point** (`handler-start`) or a **line regex**
(`/^func handler/`). A token is a regex if and only if it begins with `/`; it
ends at the first unescaped `/`, and a literal `/` is written `\/`. An
unterminated regex is an error:

<!-- codemd: (import .. ../testdata/console/syn-unterminated-regex/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: unterminated regex "/foo x.go"
<!-- codemd: (link /foo x.go) -->
codemd: 1 error(s)
```
<!-- codemd:generated -->

Ranges may be open: `a..` runs to the end of the file, and `..b` starts at the
beginning of the file. Named points and regexes may be mixed (`func1../end/`,
`/start/..func2`). A regex may contain spaces and `..`, because each token is
read left to right, so `/a..b/../c/` is the regex `a..b`, then `..`, then the
regex `c`.

## PATH

`PATH` is a local path or an `http://` or `https://` URL. A relative local path
is resolved against the directory of the Markdown file. Paths with spaces are
not supported, because tokens are whitespace-separated; a token after the path
is read as `LANG`.

## LANG

`LANG` is the optional fence language. When it is absent, codemd resolves the
language from the source file extension (see `languages.md`); an unknown
extension falls back to `text`.

## strip

`strip` is optional and is only meaningful when a range token is a regex. It
removes the matched substring (Go's leftmost match) from the boundary lines; a
boundary line that becomes empty (whitespace only) is dropped. `strip` with no
regex token is an error:

<!-- codemd: (import .. ../testdata/console/syn-strip-regex/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: strip requires a regex token in "(import a..b x.go strip)"
<!-- codemd: (import a..b x.go strip) -->
codemd: 1 error(s)
```
<!-- codemd:generated -->

## LINK-TEXT

`LINK-TEXT` is valid in `link` mode only. A double-quoted token after `PATH`
sets the rendered link label and may contain spaces. Inside the quotes a
backslash escapes the next character: `\\` becomes `\`, `\"` becomes `"`, `\]`
becomes `]`, and `\c` becomes `c` for any other character `c`. A bare `]` is
also accepted. codemd escapes `\` and `]` in the generated label.

Each of the following is an error.

A quoted token in `import` mode:

<!-- codemd: (import .. ../testdata/console/syn-link-text-import/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: link text is only valid for link mode in "(import a..b x.go \"Label\")"
<!-- codemd: (import a..b x.go "Label") -->
codemd: 1 error(s)
```
<!-- codemd:generated -->

An empty or whitespace-only label:

<!-- codemd: (import .. ../testdata/console/syn-empty-link-text/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: empty link text in "(link a x.go \"\")"
<!-- codemd: (link a x.go "") -->
codemd: 1 error(s)
```
<!-- codemd:generated -->

More than one label:

<!-- codemd: (import .. ../testdata/console/syn-multiple-labels/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: multiple link labels in "(link a x.go \"A\" \"B\")"
<!-- codemd: (link a x.go "A" "B") -->
codemd: 1 error(s)
```
<!-- codemd:generated -->

An unterminated quote:

<!-- codemd: (import .. ../testdata/console/syn-unterminated-quote/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: unterminated quoted string in "(link a x.go \"A)"
<!-- codemd: (link a x.go "A) -->
codemd: 1 error(s)
```
<!-- codemd:generated -->

A quoted `PATH`, in either mode:

<!-- codemd: (import .. ../testdata/console/syn-quoted-path/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: path must not be quoted in "(link a \"x.go\")"
<!-- codemd: (link a "x.go") -->
codemd: 1 error(s)
```
<!-- codemd:generated -->

## Managed region

The content directly below the reference is the managed region. codemd skips
blank lines to find the first non-blank line, then applies the rules for the
mode:

- `import`: if that line opens a marked fenced block, the whole region (opening
  fence, body, closing fence, and marker) is replaced in place. If it opens an
  unmarked fenced block, that block is preserved or adopted as described below.
  Otherwise the generated fence is inserted directly below the comment, leaving
  the existing lines untouched.
- `link`: if that line is a generated link (`[label](target#L<n>)`), it is
  replaced. Otherwise the generated link is inserted directly below the
  comment.

For `import`, codemd writes the fenced snippet followed by a hidden marker:

````markdown
<!-- codemd: (import a..b server.go go) -->
```go
func main() {}
```
<!-- codemd:generated -->
````

The marker identifies the block as managed, so codemd rewrites the marked
region in place on each run. An unmarked fenced block directly below the
reference is treated as hand-written: codemd leaves it untouched, inserts its
own marked snippet directly above it, and prints a warning. If the unmarked
block already matches the generated snippet, codemd marks it in place instead
of duplicating it. The marker is an HTML comment, so Markdown renderers hide
it.

The reference line's leading whitespace is copied onto every non-blank
generated line, so a reference inside a list item keeps its snippet inside that
item.

A following reference comment is never consumed: if the first non-blank line is
another reference comment, codemd inserts the generated content directly below
the current comment and leaves the next reference in place.

Re-running re-derives the region from the comment, so repeated runs are
idempotent. Content outside managed regions is preserved as written; a document
with a consistent LF or CRLF line ending is reproduced byte-for-byte, while a
document that mixes the two is normalized as described in
[Architecture](architecture.md#line-endings).

## Links rendering

A `link` reference resolves a single line. The anchor is `#L<line>`, and lines
are 1-based.

- **Local source**: the label is `PATH:LINE` and the target is `PATH#LLINE`.
  codemd loads `PATH` relative to the Markdown file's directory and emits it as
  written.
- **HTTP source**: the label is `PATH:LINE` and the target is the URL with
  `#LLINE` appended.

`LINK-TEXT` replaces the label when present.

## Boundary semantics

Named-point marker lines are excluded from imported content; regex-matched
boundary lines are included. codemd locates the start first, then locates the
end at or after the start. A token that matches nothing is an error for that
reference:

<!-- codemd: (import .. ../testdata/console/syn-marker-missing/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: marker "nope" not found at or after line 1
<!-- codemd: (import nope..end x.go go) -->
codemd: 1 error(s)
```
<!-- codemd:generated -->

<!-- codemd: (import .. ../testdata/console/syn-regex-no-match/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: regex "zzz" matched no line at or after 1
<!-- codemd: (link /zzz/ x.go go) -->
codemd: 1 error(s)
```
<!-- codemd:generated -->

## Worked example

The example is the repository fixture in `testdata/snippets`.
It exercises a named import, a regex import, `strip`, an open range, a default
link, a custom link label, and a regex link.

`server.go`:

<!-- codemd: (import .. ../testdata/snippets/server.go go) -->
```go
package server

//codemd:handler-start
func handler() string {
	return "ok"
}

//codemd:handler-end
```
<!-- codemd:generated -->

`worker.py`:

<!-- codemd: (import .. ../testdata/snippets/worker.py python) -->
```python
import os

#codemd:worker-start
def work(x):
    return x * 2

#codemd:worker-end
```
<!-- codemd:generated -->

Input `doc.md`:

<!-- codemd: (import .. ../testdata/snippets/doc.md markdown) -->
```markdown
# Docs

<!-- codemd: (import handler-start..handler-end server.go go) -->

<!-- codemd: (import worker-start..worker-end worker.py) -->

<!-- codemd: (import /#codemd:worker-start/../#codemd:worker-end/ worker.py python strip) -->

<!-- codemd: (import worker-start.. worker.py) -->

<!-- codemd: (link handler-start server.go go) -->

<!-- codemd: (link handler-start server.go go "Handler") -->

<!-- codemd: (link /^func handler/ server.go go) -->

<!-- codemd: (link worker-start worker.py) -->
```
<!-- codemd:generated -->

The output is reproduced here for reading; the integration test
`TestIntegrationGolden` verifies it byte-for-byte against `codemd doc.md`.

Output `want.md`, produced by `codemd doc.md`:

````markdown
# Docs

<!-- codemd: (import handler-start..handler-end server.go go) -->
```go
func handler() string {
	return "ok"
}

```

<!-- codemd: (import worker-start..worker-end worker.py) -->
```python
def work(x):
    return x * 2

```

<!-- codemd: (import /#codemd:worker-start/../#codemd:worker-end/ worker.py python strip) -->
```python
def work(x):
    return x * 2
```

<!-- codemd: (import worker-start.. worker.py) -->
```python
def work(x):
    return x * 2

#codemd:worker-end
```

<!-- codemd: (link handler-start server.go go) -->
[server.go:3](server.go#L3)

<!-- codemd: (link handler-start server.go go "Handler") -->
[Handler](server.go#L3)

<!-- codemd: (link /^func handler/ server.go go) -->
[server.go:4](server.go#L4)

<!-- codemd: (link worker-start worker.py) -->
[worker.py:3](worker.py#L3)
````
