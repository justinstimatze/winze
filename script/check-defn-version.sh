#!/bin/bash
# SessionStart hook: winze is defn's own dogfooding consumer, so a session
# working here should know immediately if go.mod is pinned behind the newest
# defn release (see docs/defn-migration.md and the "track latest" policy).
#
# Best-effort and silent on success or failure: a proxy timeout or offline
# session must never block or spam session start. Prints one line only when
# a newer version genuinely exists.
set -u
cd "$(dirname "$0")/.." || exit 0

pinned=$(grep -oP 'justinstimatze/defn v\K\S+' go.mod 2>/dev/null) || exit 0
[ -z "$pinned" ] && exit 0

latest=$(timeout 3 go list -m -versions github.com/justinstimatze/defn 2>/dev/null | awk '{print $NF}' | sed 's/^v//')
[ -z "$latest" ] && exit 0

if [ "$pinned" != "$latest" ]; then
	echo "[defn] pinned v$pinned, latest is v$latest — run: go get github.com/justinstimatze/defn@v$latest && go mod tidy"
fi
exit 0
