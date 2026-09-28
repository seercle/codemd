#!/usr/bin/env bash
# Regenerate the console objective transcripts under testdata/objectives/console/.
#
# Replays each scenario with the built binary and rewrites transcript.console.
# Run scripts/update-docs.sh afterwards, or on its own via that script. Works
# from any directory.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

OBJECTIVES_UPDATE=1 go test ./internal/cli -run TestConsoleObjectives -count=1
echo "objectives updated"
