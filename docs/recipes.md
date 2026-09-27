# Recipes

Prior pages describe each feature on its own; this page puts them together.
Every recipe states the input, the command, and the result observed from the
built binary. Run the commands in a scratch directory so relative source paths
resolve from there.

Cases 1–11 and 13–17 use the repository fixture copied from
`internal/cli/testdata/integration` — `server.go` and `worker.py` (case 12
uses a locally served copy and an external URL instead):

```go
package server

//codemd:handler-start
func handler() string {
	return "ok"
}

//codemd:handler-end
```

```python
import os

#codemd:worker-start
def work(x):
    return x * 2

#codemd:worker-end
```

See [Getting started](getting-started.md) for the concepts,
[Reference syntax](reference-syntax.md) for the grammar, [Source
markers](source-markers.md) for marker comments, [Languages](languages.md) for
the built-in table, [Configuration](configuration.md) for `.codemd.yaml`, and
the [Command-line reference](cli-reference.md) for flags and exit codes.

## 1. Import by named range

Import the lines between two named markers. Both marker lines are excluded.

Input `doc.md`:

```markdown
[codemd]:# (import handler-start..handler-end server.go go)
```

````console
$ codemd doc.md
[codemd]:# (import handler-start..handler-end server.go go)
```go
func handler() string {
	return "ok"
}

```
````

## 2. Import by regex

A token that begins with `/` is a line regex. Regex boundary lines are
**included**, so both marker lines appear here.

Input `doc.md`:

```markdown
[codemd]:# (import /#codemd:worker-start/../#codemd:worker-end/ worker.py python)
```

````console
$ codemd doc.md
[codemd]:# (import /#codemd:worker-start/../#codemd:worker-end/ worker.py python)
```python
#codemd:worker-start
def work(x):
    return x * 2

#codemd:worker-end
```
````

## 3. Import with `strip`

`strip` removes the matched regex text from the boundary lines. A boundary line
that becomes blank is dropped, and leading and trailing blank lines are
trimmed. The interior source lines are unchanged.

Input `doc.md`:

```markdown
[codemd]:# (import /#codemd:worker-start/../#codemd:worker-end/ worker.py python strip)
```

````console
$ codemd doc.md
[codemd]:# (import /#codemd:worker-start/../#codemd:worker-end/ worker.py python strip)
```python
def work(x):
    return x * 2
```
````

## 4. Open range to end of file

An empty end (`start..`) runs to the last line, including any trailing markers.

Input `doc.md`:

```markdown
[codemd]:# (import worker-start.. worker.py)
```

````console
$ codemd doc.md
[codemd]:# (import worker-start.. worker.py)
```python
def work(x):
    return x * 2

#codemd:worker-end
```
````

## 5. Open range from start of file

An empty start (`..end`) begins at the first line. The named end marker is
excluded.

Input `doc.md`:

```markdown
[codemd]:# (import ..handler-end server.go)
```

````console
$ codemd doc.md
[codemd]:# (import ..handler-end server.go)
```go
package server

//codemd:handler-start
func handler() string {
	return "ok"
}

```
````

## 6. Mixed named and regex range

One boundary may be a named point and the other a regex. This example keeps the
named start (excluded) and ends on a regex that matches the Go marker line
(included).

Input `doc.md`:

```markdown
[codemd]:# (import handler-start../codemd:handler-end/ server.go go)
```

````console
$ codemd doc.md
[codemd]:# (import handler-start../codemd:handler-end/ server.go go)
```go
func handler() string {
	return "ok"
}

//codemd:handler-end
```
````

## 7. Per-reference fence override

The `LANG` token after the path sets the fence. It may differ from the source's
extension.

Input `doc.md`:

```markdown
[codemd]:# (import handler-start..handler-end server.go bash)
```

````console
$ codemd doc.md
[codemd]:# (import handler-start..handler-end server.go bash)
```bash
func handler() string {
	return "ok"
}

```
````

## 8. Link by named point

A link resolves a single line. The default label is `path:line`.

Input `doc.md`:

```markdown
[codemd]:# (link handler-start server.go go)
```

```console
$ codemd doc.md
[codemd]:# (link handler-start server.go go)
[server.go:3](server.go#L3)
```

## 9. Link by regex

A regex link anchors the first matching line.

Input `doc.md`:

```markdown
[codemd]:# (link /^func handler/ server.go go)
```

```console
$ codemd doc.md
[codemd]:# (link /^func handler/ server.go go)
[server.go:4](server.go#L4)
```

## 10. Link with custom text

A quoted token after the path replaces the label.

Input `doc.md`:

```markdown
[codemd]:# (link handler-start server.go go "Handler")
```

```console
$ codemd doc.md
[codemd]:# (link handler-start server.go go "Handler")
[Handler](server.go#L3)
```

## 11. Link text with spaces and escapes

The label may contain spaces, and a backslash escapes the next character.

Input `doc.md`:

```markdown
[codemd]:# (link handler-start server.go go "the \"handler\" entry")
```

```console
$ codemd doc.md
[codemd]:# (link handler-start server.go go "the \"handler\" entry")
[the "handler" entry](server.go#L3)
```

## 12. HTTP and HTTPS sources

A path may be an `http://` or `https://` URL. Serve a source locally to see the
same result as a file import; `src.go` here is the fixture source served at
`http://127.0.0.1:8137/src.go`.

Import from the URL:

````console
$ cat doc.md
[codemd]:# (import handler-start..handler-end http://127.0.0.1:8137/src.go go)
$ codemd doc.md
[codemd]:# (import handler-start..handler-end http://127.0.0.1:8137/src.go go)
```go
func handler() string {
	return "ok"
}

```
````

Link from the URL; the label is `URL:line` and the target appends the anchor:

```console
$ cat doc.md
[codemd]:# (link handler-start http://127.0.0.1:8137/src.go go)
$ codemd doc.md
[codemd]:# (link handler-start http://127.0.0.1:8137/src.go go)
[http://127.0.0.1:8137/src.go:3](http://127.0.0.1:8137/src.go#L3)
```

HTTP semantics:

- Each distinct URL is fetched at most once per run and cached, so three
  references to one URL issue a single GET.
- A non-200 response is an error for that reference and exits `1`:
  `codemd: doc.md: line 1: http://127.0.0.1:8137/missing.go: HTTP 404`.
- The request times out after 30 seconds.
- `https://` URLs are fetched the same way. The example below is illustrative
  only: `https://example.com/` is a live, externally controlled page, so the
  matched line and rendered output may differ and cannot be reproduced from
  this repository.

  ```console
  $ cat doc.md
  [codemd]:# (link /^<!doctype/ https://example.com/ html)
  $ codemd doc.md
  [codemd]:# (link /^<!doctype/ https://example.com/ html)
  [https://example.com/:1](https://example.com/#L1)
  ```

## 13. Multiple references in one file

The fixture `doc.md` combines a named import, a regex import, `strip`, an open
range, a default link, a custom-label link, and a regex link. Its input and full
output are listed in [Reference syntax](reference-syntax.md#worked-example).
Verify the whole document in one run:

```console
$ diff <(codemd doc.md) want.md && echo FIXTURE_OK
FIXTURE_OK
```

## 14. Resolve a file to stdout

With no output flag, codemd prints the resolved document and writes nothing to
disk.

Input `link.md`:

```markdown
[codemd]:# (link handler-start server.go go)
```

```console
$ codemd link.md
[codemd]:# (link handler-start server.go go)
[server.go:3](server.go#L3)
$ cat link.md
[codemd]:# (link handler-start server.go go)
```

## 15. Write in place

`-w` rewrites each file whose resolved output differs and prints a summary to
standard error. A second run changes nothing.

````console
$ codemd -w doc.md
codemd: 1 file(s) checked, 1 updated
$ cat doc.md
[codemd]:# (import handler-start..handler-end server.go go)
```go
func handler() string {
	return "ok"
}

```
$ codemd -w doc.md
codemd: 1 file(s) checked, 0 updated
````

## 16. Write to a new file

`-o out.md` writes the result to `out.md` and requires exactly one input.

````console
$ codemd -o out.md doc.md
$ diff out.md want.md && echo MATCHES_WANT
MATCHES_WANT
$ codemd -o out.md a.md b.md
codemd: -o requires exactly one input file
$ echo $?
2
````

The `-o` and `-w` forms are equivalent in content: `-o out.md doc.md` produces
the same bytes as `-w` writes back.

## 17. Preview a diff

`-d` prints a unified diff with up to three lines of context and `a/` and `b/`
headers, and writes nothing to disk.

Input `link.md`:

```markdown
[codemd]:# (link handler-start server.go go)
```

```console
$ codemd -d link.md
--- a/link.md
+++ b/link.md
@@ -1,1 +1,2 @@
 [codemd]:# (link handler-start server.go go)
+[server.go:3](server.go#L3)
```

Exit code is `0` even when the diff is non-empty; use `--check` to fail a build.

## 18. CI check

`--check` reports each stale file and exits `1`; once the files are current it
exits `0`.

````console
$ codemd --check '**/*.md'
codemd: doc.md is out of date
codemd: 1 file(s) checked, 1 out of date
codemd: 1 error(s)
$ echo $?
1
$ codemd -w doc.md
codemd: 1 file(s) checked, 1 updated
$ codemd --check '**/*.md'
codemd: 1 file(s) checked, 0 out of date
$ echo $?
0
````

## 19. Input expansion

Positional arguments may be a file, a directory, or a glob. A directory is
walked recursively for `.md` and `.markdown` files; a bare glob matches only the
given depth; `**` matches whole path segments, and a `./` prefix is accepted.

For this layout:

```text
doc.md
sub/nested.md
```

```console
$ codemd --check doc.md
codemd: doc.md is out of date
codemd: 1 file(s) checked, 1 out of date
codemd: 1 error(s)
$ codemd --check .
codemd: doc.md is out of date
codemd: sub/nested.md is out of date
codemd: 2 file(s) checked, 2 out of date
codemd: 2 error(s)
$ codemd --check '*.md'
codemd: doc.md is out of date
codemd: 1 file(s) checked, 1 out of date
codemd: 1 error(s)
$ codemd --check './**/*.md'
codemd: doc.md is out of date
codemd: sub/nested.md is out of date
codemd: 2 file(s) checked, 2 out of date
codemd: 2 error(s)
```

Each command exits `1`. See the [Command-line reference](cli-reference.md#inputs)
for de-duplication and hidden-directory rules.

## 20. Read from stdin

With no file argument, codemd reads all of standard input and writes the result
to standard output. Relative source paths resolve from the current directory,
and errors are named `<stdin>`.

Input `link.md`:

```markdown
[codemd]:# (link handler-start server.go go)
```

```console
$ cat link.md | codemd
[codemd]:# (link handler-start server.go go)
[server.go:3](server.go#L3)
$ cat link.md | codemd --check
codemd: <stdin> is out of date
codemd: 1 error(s)
$ echo $?
1
```

The stdin form of `--check` prints no file-count summary, and in-place flags are
rejected with no file input.

## 21. Force a write past errors

Writing is all-or-nothing per file. If a reference fails, the file is left
untouched and the error is reported. `--force` writes the partial result; it
does not suppress the error, and the exit code stays `1`.

Input `broken.md`:

```markdown
[codemd]:# (import handler-start..handler-end server.go go)

[codemd]:# (import nope server.go go)
```

````console
$ codemd -w broken.md
codemd: broken.md: line 3: import range must contain '..': "nope"
codemd: broken.md: not written due to errors (use --force to write anyway)
codemd: 1 file(s) checked, 0 updated
codemd: 1 error(s)
$ codemd -w --force broken.md
codemd: broken.md: line 3: import range must contain '..': "nope"
codemd: 1 file(s) checked, 1 updated
codemd: 1 error(s)
$ cat broken.md
[codemd]:# (import handler-start..handler-end server.go go)
```go
func handler() string {
	return "ok"
}

```

[codemd]:# (import nope server.go go)
````

## 22. Custom language via config

Add an extension with a `.codemd.yaml` next to the Markdown file. This `coffee`
entry maps `#` comments and a `coffee` fence; see
[Configuration](configuration.md) for discovery and merge rules.

`.codemd.yaml`:

```yaml
languages:
  coffee:
    line: "#"
    fence: coffee
```

`src.coffee`:

```coffee
#codemd:s
x = 1
#codemd:e
```

Input `doc.md`:

```markdown
[codemd]:# (import s..e src.coffee)

[codemd]:# (link s src.coffee)
```

````console
$ codemd doc.md
[codemd]:# (import s..e src.coffee)
```coffee
x = 1
```

[codemd]:# (link s src.coffee)
[src.coffee:1](src.coffee#L1)
````

## 23. Unknown-extension source

When the extension is unknown, codemd tries the generic comment forms (`//` with
`/* */`, then `#`, then `<!-- -->`) and falls back to the `text` fence. Here
`.txt` is unknown, so the `//` marker is recognized and the fence is `text`.

`src.txt`:

```text
//codemd:s
line one
line two
//codemd:e
```

Input `doc.md`:

```markdown
[codemd]:# (import s..e src.txt)
```

````console
$ codemd doc.md
[codemd]:# (import s..e src.txt)
```text
line one
line two
```
````

## 24. Markdown-in-Markdown

A `.md` source uses the `<!-- -->` comment form, and the generated fence is
`markdown`.

`src.md`:

```markdown
<!-- codemd:s -->
# Title

Body text.
<!-- codemd:e -->
```

Input `doc.md`:

```markdown
[codemd]:# (import s..e src.md)

[codemd]:# (link s src.md)
```

````console
$ codemd doc.md
[codemd]:# (import s..e src.md)
```markdown
# Title

Body text.
```

[codemd]:# (link s src.md)
[src.md:1](src.md#L1)
````

## 25. CRLF files

codemd detects the document's line ending and preserves it. If any line ends
with CRLF, the whole resolved document is written with CRLF; otherwise LF is
used. A file without a trailing newline stays without one. Imported source
content is normalized to the host document's line endings.

A CRLF document importing from `src.go`:

`src.go`:

```go
package p

//codemd:s
a := 1
//codemd:e
```

```console
$ printf '[codemd]:# (import s..e src.go go)\r\n' > doc.md
$ codemd doc.md > out.md
$ file out.md
out.md: ASCII text, with CRLF line terminators
```

The generated fence and snippet are CRLF, like the reference line. Once the
document is current, `-d` prints nothing: preserved CRLF endings are not
reported as a change. A document with no trailing newline keeps that property:

```console
$ printf '[codemd]:# (import s..e src.go go)' > doc.md
$ codemd doc.md | od -An -c | tail -1
   `   `   `
```

The last three bytes are the closing fence, with no trailing newline. A
document whose lines mix LF and CRLF is normalized to CRLF, because any CRLF
line selects CRLF for the whole document.
