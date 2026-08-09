#!/bin/bash
set -euo pipefail

# Local quality gates — run before committing.
#
# The gating section runs what .github/workflows/ci.yml gates on, minus the
# defn ingest (CI builds .defn/ from scratch; locally it is already warm) and
# the topology pass that depends on it. Keep the two in step: a local gate
# stricter than CI goes red, stops being run, and then stops catching anything.
# That is what happened here — the script asserted "equivalent to what CI would
# run" while running two linters CI did not, died at the first of them under
# `set -e`, and never reached the winze lint line below.
#
# golangci-lint is the one deliberate divergence and it is advisory. It reports
# roughly 46 findings CI does not gate on, most of them errcheck; making those
# fatal is a separate decision from adopting staticcheck. Printed, never fatal.

cd "$(dirname "$0")/.."

echo "=== go build ==="
go build ./...

echo "=== go vet ==="
go vet ./...

echo "=== staticcheck ==="
staticcheck ./...

echo "=== go test ==="
go test ./...

echo "=== winze lint (deterministic) ==="
go run ./cmd/lint corpus

echo ""
echo "All gates passed."

echo ""
echo "=== golangci-lint (advisory — not gated in CI) ==="
golangci-lint run ./... || true
