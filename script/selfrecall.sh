#!/bin/bash
set -euo pipefail

# Run the self-recall measurement at N=150 -- the fixed size every prior
# result in ROADMAP.md's Known-problems section was measured at -- without
# depending on remembering WINZE_SELFRECALL_N by hand. It's been forgotten
# at least once already (a 20-session default run got reported as if it were
# comparable to the N=150 baselines; it wasn't, and had to be killed and
# rerun).
#
# Usage: script/selfrecall.sh <tag> [--claims]
#   <tag>      short label for this run (e.g. "baseline", "surfaceforms") --
#              names the store dir and log file so concurrent runs never
#              collide.
#   --claims   sets WINZE_NOTE_SHAPE=claims (needs ANTHROPIC_API_KEY, pulled
#              from .env if not already exported).
#
# Everything else this project's diagnostics have gated on --
# WINZE_EMBED_MODEL, WINZE_SURFACE_FORMS, WINZE_SURFACE_FORMS_MAX_CALLS --
# passes through from your shell unchanged. This script only fixes the parts
# that are easy to forget: N, the store path, .env sourcing, and the log
# destination.
#
# Runs in the background and returns immediately. Prints PID, LOG, and STORE.
# Tail the log yourself, or (agents) use Monitor with
#   tail -f --pid=<PID> -n +1 <LOG> | grep -E --line-buffered "TITLE PROBE|LATER PROBE|PASS$|FAIL"

if [ $# -lt 1 ]; then
	echo "usage: $0 <tag> [--claims]" >&2
	exit 1
fi

tag="$1"
shift
claims=0
for arg in "$@"; do
	case "$arg" in
	--claims) claims=1 ;;
	*)
		echo "unknown argument: $arg" >&2
		exit 1
		;;
	esac
done

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root"

# Out-of-repo and absolute, on purpose: a Go test binary's CWD is the
# package directory (cmd/longmemeval/), not wherever this script was
# invoked from, so a relative store path resolves somewhere unexpected --
# and if that somewhere is still inside this repo, defndb's walk-up-to-find-
# .defn/ logic can latch onto winze's OWN corpus index instead of falling
# back cleanly for the scratch store, which failed outright when the real
# .defn/ database was also locked by this repo's live winze-mcp process
# (confirmed live: every --semantic call inside the scratch store errored
# with "defndb: corpus not available", 0 link edges created, 30-second
# silent dedup fail-open on every write).
out_dir="${SELFRECALL_OUT_DIR:-$HOME/.cache/winze-selfrecall}"
mkdir -p "$out_dir"

ts="$(date +%s)"
store="$out_dir/store-$tag-$ts"
log="$out_dir/selfrecall-$tag-n150-$ts.log"

if [ -z "${ANTHROPIC_API_KEY:-}" ] && [ -f .env ]; then
	set -a
	# shellcheck disable=SC1091
	source .env
	set +a
fi

export WINZE_SELFRECALL_N=150
export WINZE_SELFRECALL_STORE="$store"
if [ "$claims" -eq 1 ]; then
	export WINZE_NOTE_SHAPE=claims
fi

nohup go test ./cmd/longmemeval/ -run TestSelfRecallDecaysWithCorpusGrowth -v -timeout 90m \
	>"$log" 2>&1 &
pid=$!

echo "PID=$pid"
echo "LOG=$log"
echo "STORE=$store"
