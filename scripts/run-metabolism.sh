#!/usr/bin/env bash
# Wrapper for any metabolism phase that logs full output to /tmp and only
# surfaces key status lines to stdout. Keeps Claude Code session context small.
#
# Usage: ./scripts/run-metabolism.sh --dream [extra flags]
#        ./scripts/run-metabolism.sh --dream --fix --tighten
#        ./scripts/run-metabolism.sh --calibrate
#        ./scripts/run-metabolism.sh --trip --pairs 5
#        ./scripts/run-metabolism.sh --bias
#        ./scripts/run-metabolism.sh --evolve --phases=cheap
#        ./scripts/run-metabolism.sh --reify
#        ./scripts/run-metabolism.sh --durability
set -euo pipefail

if [ $# -eq 0 ]; then
  echo "usage: $0 --<phase> [flags]" >&2
  exit 1
fi

# Derive a short label from the first flag for the log filename.
LABEL=$(echo "$1" | sed 's/^--//' | sed 's/[^a-z0-9]/-/g')
LOG=/tmp/metabolism-${LABEL}-$(date +%s).log
echo "metabolism log: $LOG"

go run ./cmd/metabolism "$@" . 2>&1 | tee "$LOG" \
  | grep -E '^\[|^===|^  [!*]|error:|^build |exit status|fixed|tightened|triggered|ALLOW|SKIP|corroborated|challenged|no_signal' \
  | head -120

echo "---"
echo "full log: $LOG"
