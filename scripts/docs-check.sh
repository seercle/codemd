#!/usr/bin/env bash
# Fail if the docs/ pages are out of date, without writing anything.
#
# Builds a throwaway codemd binary and runs `--check` over docs/. Use this as a
# fast pre-commit or CI check; scripts/update-docs.sh regenerates the pages. The
# script works from any directory.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

go build -o "$tmp/codemd" ./cmd/codemd
"$tmp/codemd" --check docs
echo "docs are up to date"
