# codemd

Resolve code references embedded in Markdown. A reference either **imports** a
snippet from a source file or generates a **link** to a source line. The
reference comment is preserved, so re-running refreshes snippets and line
numbers.

## Install

```bash
go install github.com/seercle/codemd/cmd/codemd@latest
```

## Usage

`codemd [flags] file.md...` — with no files, reads stdin and writes stdout.

## Grammar

```
[codemd]:# (MODE RANGE PATH [LANG] [strip])
```

- **MODE**: `import` or `link`.
- **RANGE**: `import` takes two tokens joined by `..`; `link` takes one token.
  A token is a named point (`handler-start`) or a line regex (`/^func/`).
  Open ranges are allowed: `a..` and `..b`. Named points and regexes may be
  mixed (`func1../end/`). Regex boundaries are included; named-point marker
  lines are excluded.
- **PATH**: a local path (relative to the Markdown file) or `http(s)://` URL.
- **LANG**: optional fence language; otherwise resolved from the extension.
- **strip**: optional; removes the regex match from the boundary line.

The managed region is the first non-blank line after the comment: an existing
fence is replaced for `import`, or the single line is replaced for `link`.

## Source markers

Named points are declared as one-line comments. Marker lines are excluded from
imports but are valid link targets.

```go
//codemd:handler-start
func handler() string {
	return "ok"
}
//codemd:handler-end
```

## Flags

| Flag | Description |
| --- | --- |
| `-w` | Write the result in place. |
| `-o out.md` | Write to a new file (exactly one input). |
| `-d` | Print a unified diff. |
| `--check` | Exit non-zero if any file would change (CI). |
| `--config path` | Use an explicit config file. |

`-w`, `-o`, `-d`, and `--check` are mutually exclusive.

## Config

`.codemd.yaml` is discovered by walking up from the Markdown file, or passed
with `--config`. Each entry defines exactly one of `line` or `block`.

```yaml
languages:
  go:
    line: "//"
    fence: go
  python:
    line: "#"
    fence: python
  html:
    block: ["<!--", "-->"]
    fence: html
```
