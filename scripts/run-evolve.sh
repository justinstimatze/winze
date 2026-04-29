#!/usr/bin/env bash
# Thin wrapper around --evolve that logs full output to /tmp and only surfaces
# key status lines to stdout. Keeps Claude Code transcript size down.
#
# Usage: ./scripts/run-evolve.sh [extra flags]
#   e.g. ./scripts/run-evolve.sh --phases=cheap
#        ./scripts/run-evolve.sh --dry-run
set -euo pipefail

LOG=/tmp/evolve-$(date +%s).log
echo "evolve log: $LOG"

go run ./cmd/metabolism --evolve "$@" . 2>&1 | tee "$LOG" \
  | grep -E '^\[cycle\]|^\[gate\]|^\[dream\]|^===|^  [!]|error:|^build |exit status' \
  | head -80

echo "---"
echo "full log: $LOG"
