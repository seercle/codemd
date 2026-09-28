#!/usr/bin/env bash
# Regenerate the docs/ pages that contain codemd references.
#
# Builds a throwaway codemd binary and runs it with -w against docs/. The pages
# under docs/ import examples from testdata/snippets/, so run
# this after changing those examples or the pages. The script works from any
# directory.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

"$root/scripts/update-objectives.sh"

go build -o "$tmp/codemd" ./cmd/codemd
"$tmp/codemd" -w docs
echo "docs updated"
