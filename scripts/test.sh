#!/usr/bin/env bash
# Run the full pre-commit gate: format check, go vet, and go test.
#
# Mirrors "The gate" in docs/contributing.md with CGO_ENABLED=0. gofmt is
# restricted to tracked Go files so a checked-out .worktrees/ cannot cause
# false failures. The script works from any directory.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

export CGO_ENABLED=0

mapfile -t files < <(git ls-files '*.go')
if [ "${#files[@]}" -eq 0 ]; then
	echo "gofmt: no tracked Go files" >&2
	exit 1
fi

unformatted="$(gofmt -l "${files[@]}")"
if [ -n "$unformatted" ]; then
	echo "gofmt: these files need formatting (run scripts/format.sh):" >&2
	echo "$unformatted" >&2
	exit 1
fi
echo "gofmt: clean"

echo "go vet ./..."
go vet ./...

echo "go test ./..."
go test ./...
