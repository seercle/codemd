#!/usr/bin/env bash
# Build the codemd binary into the repository root.
#
# The output path (./codemd) is gitignored. The script works from any directory.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

go build -o codemd ./cmd/codemd
echo "built $root/codemd"
