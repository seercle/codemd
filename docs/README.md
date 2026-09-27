# codemd documentation

codemd resolves code references embedded in Markdown. A reference either
**imports** a snippet from a source file or generates a **link** to a source
line. The reference comment is preserved, so re-running codemd refreshes
snippets and line numbers in place.

## Contents

- [Getting started](getting-started.md) — install codemd, run a first import and link, and see why a second run changes nothing.
- [Command-line reference](cli-reference.md) — every flag, input form, output mode, and exit code.
- [Reference syntax](reference-syntax.md) — the reference-comment grammar and managed-region rules.
- [Source markers](source-markers.md) — declare named points in source files that imports and links can target.
- [Languages](languages.md) — how codemd maps extensions to comment forms and fence names.
- [Configuration](configuration.md) — the `.codemd.yaml` schema, discovery, and merge semantics.
- [Recipes](recipes.md) — worked examples for every use case, from a first import to CI checks.
- [Troubleshooting](troubleshooting.md) — exit codes and every error message with its cause and fix.
- [Architecture](architecture.md) — the design record and the resolution pipeline.
- [Contributing](contributing.md) — build, test, and where to make common changes.

## 30-second example

Give `server.go` a `handler-start` marker and a matching `handler-end` marker
(see [Getting started](getting-started.md)), then put this reference in
`doc.md`:

````markdown
# My docs

[codemd]:# (import handler-start..handler-end server.go go)
````

Run codemd to generate the snippet:

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

The reference comment is preserved and the generated fence is inserted directly
below it. Pass `-w` to write the result to the file in place.

## Document map

- New to codemd → [Getting started](getting-started.md)
- Look up a flag or exit code → [Command-line reference](cli-reference.md)
- Write or debug a reference comment → [Reference syntax](reference-syntax.md)
- Add a named point to source → [Source markers](source-markers.md)
- Teach codemd a new extension → [Configuration](configuration.md)
- Set up a CI check → [Command-line reference](cli-reference.md) and [Recipes](recipes.md#18-ci-check)
- Fix an error message → [Troubleshooting](troubleshooting.md)
- Change the implementation → [Architecture](architecture.md) and [Contributing](contributing.md)
