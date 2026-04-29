#!/usr/bin/env bash
# Thin alias — delegates to run-metabolism.sh --evolve.
# Kept for backwards compatibility with documented usage.
exec "$(dirname "$0")/run-metabolism.sh" --evolve "$@"
