#!/bin/bash
set -euo pipefail

# Local quality gates — run before committing.
# Equivalent to what CI would run.

cd "$(dirname "$0")/.."

echo "=== go build ==="
go build ./...

echo "=== go vet ==="
go vet ./...

echo "=== staticcheck ==="
staticcheck ./...

echo "=== golangci-lint ==="
golangci-lint run ./...

echo "=== winze lint (deterministic) ==="
go run ./cmd/lint .

echo ""
echo "All checks passed."
