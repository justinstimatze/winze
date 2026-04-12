#!/bin/bash
set -euo pipefail

# Set up autonomous winze operation.
#
# Two modes:
#   1. Cron-based (no dependencies): runs --cycle on a schedule
#   2. Gas Town (full orchestration): uses gt/bd for multi-agent operation
#
# Prerequisites:
#   - Go 1.24+
#   - ANTHROPIC_API_KEY in .env or environment
#   - Optional: Wikipedia ZIM file for offline search

cd "$(dirname "$0")/.."
WINZE_DIR="$(pwd)"

echo "=== winze autonomous setup ==="
echo ""

# Check prerequisites
if ! command -v go &> /dev/null; then
    echo "ERROR: Go not found. Install Go 1.24+ from https://go.dev/dl/"
    exit 1
fi

if ! go build ./... 2>/dev/null; then
    echo "ERROR: go build failed. Fix compilation errors first."
    exit 1
fi

echo "✓ Go build passes"

# Check for API key
if [ -f .env ]; then
    # shellcheck disable=SC1091
    source .env 2>/dev/null || true
fi
if [ -z "${ANTHROPIC_API_KEY:-}" ]; then
    echo ""
    echo "WARNING: ANTHROPIC_API_KEY not set."
    echo "  Without it, --trip (speculative connections) and --dream --fix --tighten"
    echo "  (LLM-assisted Brief tightening) will be skipped."
    echo "  The rest of the cycle (dream, bias audit, calibrate) works without it."
    echo ""
    echo "  To set it: echo 'ANTHROPIC_API_KEY=sk-...' > .env"
    echo ""
    HAS_API_KEY=false
else
    echo "✓ ANTHROPIC_API_KEY found"
    HAS_API_KEY=true
fi

# Check for ZIM file
ZIM_PATH="${1:-}"
if [ -n "$ZIM_PATH" ] && [ -f "$ZIM_PATH" ]; then
    echo "✓ ZIM file: $ZIM_PATH"
    HAS_ZIM=true
else
    echo "  No ZIM file specified (optional — enables Wikipedia offline search)"
    echo "  Download from: https://download.kiwix.org/zim/wikipedia/"
    echo "  Then re-run: $0 /path/to/wikipedia.zim"
    HAS_ZIM=false
fi

echo ""

# Check for Gas Town
if command -v gt &> /dev/null; then
    echo "Gas Town detected ($(gt --version 2>/dev/null || echo 'unknown version'))"
    echo ""
    echo "=== Gas Town mode ==="
    echo ""
    echo "The full autonomous loop uses Gas Town for scheduling and multi-agent"
    echo "coordination. To start it:"
    echo ""
    echo "  # One-time: install formulas"
    echo "  cp .beads/formulas/mol-curate-auto.formula.toml \$(gt config get formula-dir 2>/dev/null || echo '~/.gt/formulas/')/"
    echo ""
    echo "  # Start the overnight cycle"
    if [ "$HAS_ZIM" = true ]; then
        echo "  gt mol start mol-curate-auto --var zim_path=$ZIM_PATH"
    else
        echo "  gt mol start mol-curate-auto"
    fi
    echo ""
    echo "  # Check status in the morning"
    echo "  gt mol status"
    echo ""
else
    echo "Gas Town not detected — using cron mode."
fi

echo "=== Cron mode (no dependencies) ==="
echo ""
echo "The --cycle flag runs the full sleep cycle in one command:"
echo "  metabolism → dream (with bias audit) → trip → calibrate"
echo ""

# Build the cycle command
CYCLE_CMD="cd $WINZE_DIR && go run ./cmd/metabolism --cycle"
if [ "$HAS_ZIM" = true ]; then
    CYCLE_CMD="$CYCLE_CMD --zim $ZIM_PATH"
fi
if [ "$HAS_API_KEY" = false ]; then
    CYCLE_CMD="$CYCLE_CMD --dry-run"
    echo "  (dry-run mode — no API key for trip/fix phases)"
fi
CYCLE_CMD="$CYCLE_CMD . >> .cycle-log.txt 2>&1"

echo "To run once:"
echo "  go run ./cmd/metabolism --cycle ."
echo ""
echo "To schedule nightly at 3am via cron:"
echo "  crontab -e"
echo "  # Add this line:"
echo "  0 3 * * * $CYCLE_CMD"
echo ""

# Offer to install the cron job
read -p "Install the cron job now? [y/N] " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    # Add to crontab without clobbering existing entries
    CRON_LINE="0 3 * * * $CYCLE_CMD"
    (crontab -l 2>/dev/null || true; echo "$CRON_LINE") | crontab -
    echo "✓ Cron job installed. The cycle will run nightly at 3am."
    echo "  Logs: $WINZE_DIR/.cycle-log.txt"
    echo "  Remove: crontab -e and delete the winze line"
else
    echo "Skipped. You can install it manually later."
fi

echo ""
echo "=== Verification ==="
echo ""
echo "Test the cycle (dry run):"
echo "  go run ./cmd/metabolism --cycle --dry-run ."
echo ""
echo "View the morning report:"
echo "  go run ./cmd/metabolism --dream ."
echo "  go run ./cmd/metabolism --bias ."
echo "  go run ./cmd/metabolism --calibrate --narrative ."
