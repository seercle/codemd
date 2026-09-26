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
[codemd]:# (MODE RANGE PATH [LANG] [strip] ["LINK-TEXT"])
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
- **LINK-TEXT**: `link` only; a quoted token after PATH sets the link label. It
  may contain spaces and must not contain `]`.

The reference manages the content directly below it. Blank lines are skipped to
find the first non-blank line: an existing fence (for `import`) or generated
link (for `link`) is replaced in place; otherwise the generated block is
inserted directly below the comment, leaving existing lines untouched.

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
| `--languages` | List supported languages and exit. |

`-w`, `-o`, `-d`, and `--check` are mutually exclusive.

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
