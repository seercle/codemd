# Source markers

A **source marker** declares a named point in a source file. A marker is a
one-line comment whose text, after surrounding whitespace is trimmed, begins
with the fixed literal `codemd:` followed by a name. Ranges and links refer to
that name; see [Reference syntax](reference-syntax.md) for the reference
comment grammar.

## Declaration

Write the marker with the comment form of the source's language. Space between
the comment delimiter and `codemd:` is allowed, and the name is trimmed of
surrounding whitespace.

| Language | Comment form | Example |
| --- | --- | --- |
| Go, JavaScript, TypeScript | `//` line | `//codemd:name` |
| Python, Shell, YAML | `#` line | `#codemd:name` |
| Lua, SQL | `--` line | `--codemd:name` |
| C (dual-form), CSS | `/* */` block (C also `//`) | `/* codemd:name */` |
| HTML, XML, Markdown | `<!-- -->` block | `<!-- codemd:name -->` |

A block marker must open and close on the same line. Some languages accept more
than one form (for example, C accepts both `//` and `/* */`); see
[Languages](languages.md) for the complete per-extension table.

## Rules

- A marker may trail code on the same line. The line is still a marker line:
  `func f() int { return 42 } //codemd:end` declares `end` on that line.
- The name matches `[A-Za-z0-9_.-]+` and must not contain `..`. A comment whose
  name fails either rule is not a marker: `//codemd:has space`,
  `//codemd:a..b`, and `//codemd:` declare nothing.
- Markers are recognized according to the source's comment form. A file whose
  extension is unknown uses the fallback forms described below.

A marker name used by a reference must be unique; an unreferenced duplicate does
not fail the file. Duplicating a referenced name is an error:

<!-- codemd: (import .. ../testdata/console/marker-duplicate/transcript.console console) -->
```console
$ codemd doc.md
codemd: doc.md: line 1: duplicate marker "a" on lines 1 and 2
<!-- codemd: (import a..b src.txt) -->
codemd: 1 error(s)
```

## Imports and links

A marker line is excluded from an imported snippet but remains a valid link
target. The whole line is excluded, including any trailing code on it. Given
this source:

<!-- codemd: (import .. ../testdata/snippets/server.go go) -->
```go
package server

//codemd:handler-start
func handler() string {
	return "ok"
}

//codemd:handler-end
```

`import handler-start..handler-end` begins on the `func` line and drops both
marker lines, while the link:

```markdown
<!-- codemd: (link handler-start server.go go) -->
```

resolves to the marker's own line:

```markdown
[server.go:3](server.go#L3)
```

## Unknown extensions

When a source's extension is not a known language, codemd tries the generic
comment forms in order: `//` with `/* */`, then `#`, then `<!-- -->`. The first
form that yields a valid marker on a line wins. A marker in a `.txt` file
recognized through this fallback behaves like any other marker.

## Configuration

To add a language or override a comment form, see
[Configuration](configuration.md).
