# Getting started

## What codemd does

codemd resolves code references embedded in Markdown. A **reference comment** is
a line of the form `[codemd]:# (MODE RANGE PATH [LANG] [strip] ["LINK-TEXT"])`.
A reference either **imports** a snippet from a source file or generates a
**link** to a source line. The content generated directly below the comment is
the **managed region**: codemd owns that block and rewrites it in place. The
reference comment itself is preserved, so re-running codemd refreshes the
snippet or the link's line number. That property is what makes codemd
**idempotent**: running it on unchanged sources produces no further changes.

## Install

```bash
go install github.com/seercle/codemd/cmd/codemd@latest
```

Release builds set the version with
`-ldflags "-X github.com/seercle/codemd/internal/cli.Version=vX.Y.Z"`; source
builds report `dev`.

```console
$ codemd --version
codemd dev
```

## Your first import

The canonical source file has a **source marker**:

[codemd]:# (import .. ../internal/cli/testdata/integration/server.go go)
```go
package server

//codemd:handler-start
func handler() string {
	return "ok"
}

//codemd:handler-end
```

The repository's canonical copy is `starter.md`; create your own `doc.md` with
the same contents. It holds a reference comment that imports the region between
the two markers:

[codemd]:# (import .. ../internal/cli/testdata/integration/starter.md markdown)
```markdown
# My docs

[codemd]:# (import handler-start..handler-end server.go go)
```

Running `codemd doc.md` inserts the snippet below the comment (the transcript
that follows is hand-written; codemd cannot generate console output):

````console
$ codemd doc.md
# My docs

[codemd]:# (import handler-start..handler-end server.go go)
```go
func handler() string {
	return "ok"
}

```
````

Pass `-w` to write the result back to the file in place:

```console
$ codemd -w doc.md
codemd: 1 file(s) checked, 1 updated
```

The reference comment is kept, and the generated fence is inserted directly
below it. The `go` token after the path sets the fence language; without it,
codemd resolves the language from the file extension.

## Your first link

A `link` reference generates a Markdown link to a source line instead of a
snippet. Add this line to `doc.md`:

```markdown
[codemd]:# (link handler-start server.go go)
```

Run codemd to write the link:

```console
$ codemd -w doc.md
codemd: 1 file(s) checked, 1 updated
```

Below the reference comment the file now contains:

```markdown
[server.go:3](server.go#L3)
```

The label defaults to `path:line`, and the target anchors that line in the
source file. Pass a quoted label after the path to override it.

## A source marker

A **source marker** is a one-line comment that declares a named point in a
source file:

```go
//codemd:handler-start
```

Ranges and links refer to markers by name (`handler-start`). The marker line
itself is excluded from an imported snippet but remains a valid link target:
the example above links to line 3, the marker line, while the import begins on
the following `func` line. A range can also be open (`handler-start..`) or
defined by a line regex (`/^func handler/`) instead of a named point.

## Idempotency

Because the reference comment is preserved and the generated content is
replaced in place, a second run changes nothing. Run codemd with `-w` again and
the summary reports no updates:

```console
$ codemd -w doc.md
codemd: 1 file(s) checked, 0 updated
```

Only a run that has something to generate reports `1 updated`. Once the
document is current, no further changes are written, which makes `codemd
--check` safe to use in CI to fail a build when generated content is stale.

## Where next

- [Command-line reference](cli-reference.md) — flags, input expansion, and
  output modes.
- [Reference syntax](reference-syntax.md) — ranges, markers, regexes, config,
  and language handling.
- [Recipes](recipes.md) — common patterns and worked examples.
