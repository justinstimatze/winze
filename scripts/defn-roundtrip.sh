#!/usr/bin/env bash
# defn-roundtrip.sh — winze<->defn emit-fidelity acceptance gate.
#
# The dogfooding bar for moving winze's write path onto defn: what defn emits
# from its Dolt DB must be the SAME CODE that went in — not merely code that
# compiles. Go's gofmt canonical form makes this an exact, checkable property:
#
#     gofmt(source)  ==  gofmt(emit(ingest(source)))     byte-for-byte
#
# "It builds" is too weak — a body truncated at a statement boundary, or a
# dropped/altered declaration, can still compile while silently losing code
# (exactly the fossil class: a body stored short by an old ingest). Byte
# identity after gofmt normalization catches the whole class and subsumes the
# build check (if the bytes equal source, the tree builds because source does).
#
# GRANULARITY CAVEAT: defn appears to reorder top-level declarations within a
# file (fossil emit had main() far from its source position). If so, a naive
# whole-file diff false-positives on pure reordering, and the correct bar is
# byte-identity at DECLARATION granularity (each gofmt'd top-level decl present
# identically, order-independent) — see cmd/defn-roundtrip once built. This
# script runs the whole-file form first; a failure that is ONLY reordering is
# the signal to switch to the decl-level checker, not a fidelity loss.
#
# Usage: scripts/defn-roundtrip.sh [defn-binary]   (default: defn on PATH)
set -euo pipefail

DEFN="${1:-defn}"
command -v "$DEFN" >/dev/null 2>&1 || { echo "defn binary not found: $DEFN" >&2; exit 2; }
SRC="$(pwd)"

OUT="$(mktemp -d)"
trap 'rm -rf "$OUT"' EXIT

echo "==> emitting DB -> $OUT"
"$DEFN" emit "$OUT" >/dev/null

echo "==> stage 1: parse-check emitted files (gofmt -e) — fast corruption catch"
bad=0
while IFS= read -r f; do
  if ! gofmt -e "$f" >/dev/null 2>"$OUT/.gofmt.err"; then
    bad=$((bad + 1)); echo "    UNPARSEABLE ${f#"$OUT"/}: $(head -1 "$OUT/.gofmt.err")"
  fi
done < <(find "$OUT" -name '*.go')
[ "$bad" -eq 0 ] || { echo "FAIL: $bad emitted file(s) do not parse — corrupted bodies (re-ingest) or emit regression"; exit 1; }
echo "    ok: all emitted files parse"

echo "==> stage 2: byte-identity vs source (gofmt-normalized) — the real bar"
diffs=0; missing=0
while IFS= read -r ef; do
  rel="${ef#"$OUT"/}"
  sf="$SRC/$rel"
  [ -f "$sf" ] || { echo "    EMIT-ONLY $rel (not in source tree)"; continue; }
  if ! diff -q <(gofmt "$sf") <(gofmt "$ef") >/dev/null 2>&1; then
    diffs=$((diffs + 1))
    echo "    DIFFERS  $rel  ($(diff <(gofmt "$sf") <(gofmt "$ef") | grep -c '^[<>]') changed lines)"
  fi
done < <(find "$OUT" -name '*.go')
# Source .go files defn never emitted (dropped decls / un-ingested files).
# Exclude polecats/ (separate worktree clones defn never ingested) and .defn/.
while IFS= read -r sf; do
  rel="${sf#"$SRC"/}"
  [ -f "$OUT/$rel" ] || { echo "    SOURCE-ONLY $rel (not emitted)"; missing=$((missing + 1)); }
done < <(find "$SRC" -name '*.go' -not -path '*/.defn/*' -not -path '*/polecats/*')

if [ "$diffs" -eq 0 ] && [ "$missing" -eq 0 ]; then
  echo "PASS: every emitted file is byte-identical to source (gofmt) — winze round-trips losslessly"
else
  echo "RESULT: $diffs file(s) differ, $missing source file(s) not emitted."
  echo "  If the differences are ONLY declaration reordering, switch to the decl-level bar."
  echo "  If any difference drops/alters/truncates a declaration, that is a real fidelity loss."
  exit 1
fi
