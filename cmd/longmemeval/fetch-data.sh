#!/usr/bin/env bash
# Fetch the LongMemEval dataset that cmd/longmemeval runs against.
#
# The data is NOT committed (see .gitignore) and is large, so this script is the
# recorded, reproducible way to re-acquire it. Source: the official cleaned
# release on HuggingFace (Wu et al., LongMemEval; repo xiaowu0162/LongMemEval).
# The 2025/09 "cleaned" variant removes history-session interference on answer
# correctness and is the one to use.
#
#   ./fetch-data.sh          # oracle only (~15 MB): answer-bearing sessions.
#                            #   Sufficient for --probe (per-session truncation
#                            #   of gold-answer turns is identical to the full set).
#   ./fetch-data.sh --full   # + longmemeval_s_cleaned.json (~265 MB): the full
#                            #   ~40-session haystack, needed for real accuracy runs.
#
# Files land in cmd/longmemeval/data/. Point the tool at them with --dataset.
set -euo pipefail

base="https://huggingface.co/datasets/xiaowu0162/longmemeval-cleaned/resolve/main"
dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/data"
mkdir -p "$dir"

fetch() {
	local name="$1" dest="$dir/$1"
	if [ -s "$dest" ]; then echo "have $name ($(du -h "$dest" | cut -f1))"; return; fi
	echo "fetching $name ..."
	curl -fL --progress-bar "$base/$name" -o "$dest.part"
	mv "$dest.part" "$dest"
	echo "  -> $dest ($(du -h "$dest" | cut -f1))"
}

fetch longmemeval_oracle.json
if [ "${1:-}" = "--full" ]; then
	fetch longmemeval_s_cleaned.json
fi
