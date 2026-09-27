#!/usr/bin/env bash
# Apply gofmt to every tracked Go file.
#
# Uses `git ls-files` instead of `gofmt -w .` so files under .worktrees/ are
# left alone. The script works from any directory.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

mapfile -t files < <(git ls-files '*.go')
if [ "${#files[@]}" -eq 0 ]; then
	echo "no tracked Go files" >&2
	exit 0
fi

gofmt -w "${files[@]}"
echo "formatted ${#files[@]} file(s)"
