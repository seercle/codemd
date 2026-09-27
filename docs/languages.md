# Languages

codemd maps each source extension to a **comment form** and a **fence** name.
The comment form is how codemd finds [source markers](source-markers.md) in a
source file. The fence name is the language tag codemd writes on a generated
code block. See [Reference syntax](reference-syntax.md) for the `LANG` token in
a reference comment.

## How languages are resolved

codemd resolves a language from the source file's extension:

- The extension is lowercased and a leading dot is stripped, so `Server.GO` and
  `s.go` both resolve as `go`.
- The built-in table maps that extension to a comment form and a fence name. A
  path with no extension (for example, `Makefile`) has an empty extension and is
  treated as unknown.
- An unknown extension uses the generic forms and the `text` fence described in
  [Unknown extensions](#unknown-extensions).

## Built-in table

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

Several extensions can share one entry: `js` and `mjs` both use the
`javascript` fence and the `//` line form.

## Dual-form entries

Some entries accept both a line prefix and a pair of block delimiters. codemd
checks the line prefix first and uses it when the line contains it, otherwise it
uses the block delimiters. Write a marker with one form on its own line. For
example, a `.c` file may declare markers with either form:

```c
/*codemd:start*/
int x;
//codemd:end
```

An `import start..end s.c` snippet then spans from the block marker to the line
marker. `c`, `h`, `cpp`, `hpp`, `java`, `rs`, and `php` are dual-form entries;
see the [built-in table](#built-in-table).

## Unknown extensions

For an extension that is not in the table, codemd tries the generic comment
forms in order; the first form that yields a valid marker on a line wins:

1. line `//` with block `/* */`
2. line `#`
3. block `<!-- -->`

The generated fence falls back to `text`. A `.txt` file, for example, can
declare markers with any of the generic forms:

```text
//codemd:a
x
#codemd:b
<!--codemd:c-->
```

An `import a..b notes.txt` snippet is emitted under a `text` fence.

## Overriding the fence or form

Two mechanisms override the defaults:

- **Explicit `LANG` token.** A fourth token after the path in a reference
  comment sets the fence for that one reference. It changes only the emitted
  fence; codemd still finds markers with the source extension's comment form.
  `[codemd]:# (import a..b s.c python)` emits a `python` fence while reading
  markers from `s.c` with the `c` comment form.
- **A `.codemd.yaml` entry.** Add or replace an extension's language in the
  config file. The entry replaces the built-in for that extension and must
  define exactly one of `line` or `block`; `fence` defaults to the extension
  name. See [Configuration](configuration.md) for the file format and
  discovery rules.

## `--languages`

`codemd --languages` prints the effective table and exits `0`. When `--config`
is given, its entries are merged over the built-ins first. The table is sorted
by extension; each line is `ext -> fence (form)`:

```console
$ codemd --languages
bash -> bash (line "#")
c -> c (line "//" or block "/*" "*/")
cpp -> cpp (line "//" or block "/*" "*/")
css -> css (block "/*" "*/")
go -> go (line "//")
h -> c (line "//" or block "/*" "*/")
hpp -> cpp (line "//" or block "/*" "*/")
html -> html (block "<!--" "-->")
java -> java (line "//" or block "/*" "*/")
js -> javascript (line "//")
jsx -> jsx (line "//")
lua -> lua (line "--")
md -> markdown (block "<!--" "-->")
mjs -> javascript (line "//")
php -> php (line "//" or block "/*" "*/")
py -> python (line "#")
rb -> ruby (line "#")
rs -> rust (line "//" or block "/*" "*/")
sh -> bash (line "#")
sql -> sql (line "--")
toml -> toml (line "#")
ts -> typescript (line "//")
tsx -> tsx (line "//")
xml -> xml (block "<!--" "-->")
yaml -> yaml (line "#")
yml -> yaml (line "#")
```

This output is the authoritative list of supported extensions.

## Where next

- [Source markers](source-markers.md) — the marker comment grammar.
- [Configuration](configuration.md) — add or override languages.
- [Reference syntax](reference-syntax.md) — the complete reference-comment
  grammar, including the `LANG` token.
