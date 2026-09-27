# Contributing

This page covers how to build codemd, how to run its tests, and where to make
common changes. Read [Architecture](architecture.md) first for the package
layout and the resolution pipeline.

## Prerequisites

- Go 1.22 or newer, as declared by `go.mod`.
- No cgo. The project builds and tests with `CGO_ENABLED=0`.

## Build and run

Build the binary and inspect its usage:

```console
$ go build ./cmd/codemd
$ go run ./cmd/codemd --help
```

The commands above work with or without the `CGO_ENABLED=0` prefix; the gate
below sets it explicitly.

## The gate

Run all three checks before you consider a change done:

```bash
export CGO_ENABLED=0
gofmt -l .
go vet ./...
go test ./...
```

`gofmt -l .` must print nothing; `go vet` and `go test` must pass. A non-empty
`gofmt` listing means one or more files need formatting.

## Test conventions

- **Unit tests** live beside the package they cover and are table-driven. Each
  case names its input and expected output, so adding a case is a one-line edit.
  See the `_test.go` files in each `internal/` package.
- **Integration tests** in `internal/cli/` call `cli.Run` in-process against
  fixtures under `internal/cli/testdata/integration/`. A fixture pairs an input
  `doc.md` with an expected `want.md`; `integration_test.go` compares the
  resolved output byte for byte. HTTP paths use `httptest` in `http_test.go`.
- **Temp fixtures** are created with the `writeTree` helper in
  `internal/cli/matrix_test.go`, which writes a set of files into a temporary
  directory. The `Run`-level tests in `internal/cli/` (`cli_test.go`,
  `matrix_test.go`, `integration_test.go`, `http_test.go`) drive the CLI
  end to end.
- **TDD.** Write the failing test first, then make it pass. A bug fix starts
  with a test that reproduces the bug.

## Adding or overriding a language

The built-in table lives in `Builtins()` in `internal/lang/lang.go`. To add a
built-in language, add an entry keyed by its extension:

```go
"coffee": {Fence: "coffee", Form: CommentForm{Line: "#"}},
```

A `Language` defines exactly one comment form: `Line` for a single-line prefix
or `Block` for a two-element open/close pair. `Fence` is the tag written on a
generated code block and defaults to the extension when empty.

For a project-specific extension, do not touch the source. Document a
`.codemd.yaml` entry instead:

```yaml
languages:
  coffee:
    line: "#"
    fence: coffee
```

Config entries merge onto the built-ins and replace a built-in in full. See
[Configuration](configuration.md) for discovery and merge semantics.

Whichever route you take, add a table entry test in `internal/lang/` covering the
new extension so the comment form and fence are pinned.

## Adding a flag

1. Wire the flag in `internal/cli/cli.go`. Follow the existing `flag` declarations
   and validate combinations alongside the current mutual-exclusion checks.
2. Update the flag table and the `--help` listing in
   [Command-line reference](cli-reference.md).
3. Add a `Run`-level test in `internal/cli/`. Use the existing `TestRun*` tests
   in `cli_test.go` as templates, and `writeTree` when the test needs inputs on
   disk.

## Documentation

Keep the pages under `docs/` in sync with behavior: a change to output, flags,
config, or language handling usually needs a documentation edit in the same
change. Verify every example against the binary before committing it; wording
like "prints" or "exits non-zero" must match what you observe.

## Where next

- [Architecture](architecture.md) — package layout and the resolution pipeline.
- [Command-line reference](cli-reference.md) — flags, inputs, and exit codes.
- [Configuration](configuration.md) — the `.codemd.yaml` schema and merge rules.
