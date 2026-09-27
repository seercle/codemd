# Configuration

codemd resolves each [source marker](source-markers.md) with the
[comment form](languages.md) of the source file's extension. A `.codemd.yaml`
file adds extensions or replaces the built-in entries. This page is the
authoritative reference for that file: discovery, schema, merge semantics, and
errors.

## Discovery

For each Markdown input, codemd walks up from the input file's directory looking
for `.codemd.yaml`, and uses the nearest one it finds. A marker in
`docs/guide/readme.md` therefore uses `docs/guide/.codemd.yaml` when present,
otherwise `docs/.codemd.yaml`, and so on up to the filesystem root. If no
`.codemd.yaml` is found, only the built-ins apply.

For stdin, discovery starts from the current working directory.

`--config path` skips discovery: it loads exactly that file and applies it
globally to every input, regardless of each input's directory. See
[the CLI reference](cli-reference.md#flags).

## Schema

The file is a YAML mapping with a top-level `languages:` key. Under it, each key
is a source extension and each value is an entry:

```yaml
languages:
  EXTENSION:
    line: "PREFIX"
    fence: FENCE-NAME
  OTHER:
    block: ["OPEN", "CLOSE"]
```

Each entry defines **exactly one** of:

- `line` — a single-line comment prefix, for example `"//"` or `"#"`.
- `block` — a two-element list of the opening and closing delimiters, for
  example `["<!--", "-->"]`.

`fence` is optional. It is the language tag codemd writes on a generated code
block. When omitted, it defaults to the extension name. A config entry replaces
a built-in entry in full, so an entry for `c` that defines only `line` removes
the built-in block form for `.c`.

## Example

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

## Merge semantics

Config entries are merged onto the built-in table (see the
[language reference](languages.md#built-in-table)):

- A key that already exists as a built-in **overrides** that entry.
- A key that does not exist **adds** a new extension.
- An entry with no `fence` inherits the extension name as its fence.

The result is the effective table codemd uses. A table supplied with `--config`
is merged the same way, and `codemd --languages --config path` prints that merged
table.

## Errors

A malformed config is an error: codemd prints a message to stderr and exits `1`.
The message names the config file.

An entry that defines both `line` and `block`:

```console
$ codemd --config config.yaml doc.md
codemd: config.yaml: language "coffee" defines both line and block
```

An entry that defines neither, or whose `block` does not have exactly two
elements:

```console
$ codemd --config config.yaml doc.md
codemd: config.yaml: language "coffee" must define exactly one of line or block
```

A file that is not valid YAML:

```console
$ codemd --config config.yaml doc.md
codemd: config.yaml: yaml: line 4: did not find expected node content
```

An unreadable `--config` path:

```console
$ codemd --config missing.yaml doc.md
codemd: open missing.yaml: no such file or directory
```

When the config is given with `--config`, it is loaded once and a failure is
reported on its own, as above. When the config is discovered, a failure is
reported per input with the config file's absolute path and counted in the
per-input error summary; one bad file fails every input that resolves through
it.

## Worked example

This example adds `coffee` to the table and uses it from both an `import` and a
`link`. Save the config as `.codemd.yaml` next to the Markdown file.

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

`doc.md`:

````markdown
[codemd]:# (import s..e src.coffee)

[codemd]:# (link s src.coffee)
````

`codemd doc.md` finds the config by discovery and emits a `coffee`-fenced block
and a link to the named point:

````markdown
[codemd]:# (import s..e src.coffee)
```coffee
x = 1
```

[codemd]:# (link s src.coffee)
[src.coffee:1](src.coffee#L1)
````

The `fence: coffee` line is optional here; without it the fence still defaults
to `coffee`.

## Where next

- [Languages](languages.md) — the built-in table, comment forms, and fences.
- [Source markers](source-markers.md) — the marker comment grammar.
- [Reference syntax](reference-syntax.md) — the `LANG` token and reference
  grammar.
- [Command-line reference](cli-reference.md) — `--config` and `--languages`.
