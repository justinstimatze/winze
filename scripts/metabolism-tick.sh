#!/usr/bin/env bash
# One metabolism cycle. Cadence belongs to the scheduler that invokes this
# (systemd timer, cron, CI); phase granularity and self-gating belong to
# winze's own gates. This script is deliberately loopless — it runs exactly
# one tick and exits, so the scheduler owns restart, catch-up, and overlap.
#
# Usage: scripts/metabolism-tick.sh
# Install as an hourly systemd user timer: scripts/install-metabolism-timer.sh
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO"

# Auto-export so ANTHROPIC_API_KEY, KAGI_API_KEY and METABOLISM_BUDGET_CENTS
# all reach the child process. The budget guard reads the cap via os.Getenv,
# so a missed export silently disables the spend cap.
if [ -f .env ]; then
  set -a
  # shellcheck disable=SC1091
  . ./.env
  set +a
fi

BIN="$REPO/metabolism"

# Rebuild only when a source file is newer than the binary. `go run` compiles
# and links on every invocation and peaks 1-2 GB RSS during link; keeping the
# toolchain off the common path is what makes an hourly tick cheap.
needs_build=0
[ -x "$BIN" ] || needs_build=1
if [ "$needs_build" -eq 0 ] && \
   [ -n "$(find ./cmd/metabolism ./internal -name '*.go' -newer "$BIN" -print -quit 2>/dev/null)" ]; then
  needs_build=1
fi
if [ "$needs_build" -eq 1 ]; then
  echo "[tick] building ./metabolism"
  go build -o "$BIN" ./cmd/metabolism
fi

"$BIN" --evolve .
"$BIN" --calibrate-trend . | tail -3

# Stage only the evolving corpus. Runtime state (.metabolism-log.json,
# .metabolism-budget.json, .metabolism-calibration.jsonl) is gitignored.
shopt -s nullglob
staged=(metabolism_cycle*.go)
shopt -u nullglob
[ -f predictions.go ] && staged+=(predictions.go)
if [ ${#staged[@]} -gt 0 ]; then
  git add -- "${staged[@]}"
fi

if git diff --cached --quiet; then
  echo "[tick] no corpus changes this cycle"
  exit 0
fi

git commit -q -m "metabolism: autonomous cycle $(date -u +%Y-%m-%dT%H:%M:%SZ)"

if ! git pull --rebase -q origin main; then
  git rebase --abort 2>/dev/null || true
  echo "[tick] rebase failed — commit left local, next tick retries"
  exit 0
fi

if ! git push -q origin HEAD:main; then
  echo "[tick] push failed — commit left local, next tick retries"
fi
