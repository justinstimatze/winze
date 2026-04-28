#!/usr/bin/env bash
# Validation battery: run a fixed set of --ask questions and capture outputs.
# Run twice (with .metabolism-trip-isolated.jsonl present, then absent) and
# diff per-question. Manual scoring after — see compact brief for rubric.
set -e
OUT=/tmp/jsonl-battery-$(date +%s)
mkdir -p "$OUT"
TRIALS=${TRIALS:-3}
HAS_JSONL=$([ -f .metabolism-trip-isolated.jsonl ] && echo "yes" || echo "no")
echo "battery dir: $OUT"
echo "JSONL present: $HAS_JSONL  trials: $TRIALS"

# tag|class|question
QUESTIONS=(
  "godel-bias|should-help|What does Gödel's incompleteness theorem have in common with cognitive bias detection?"
  "hilbert-consciousness|should-help|Are there structural parallels between Hilbert's Program and consciousness research?"
  "dual-process-pp|should-help|How does dual-process theory relate to predictive coding in schizophrenia?"
  "bias-failure|should-help|Is cognitive bias really a failure?"
  "godel-baloney|should-help|What do Gödel's theorem and Sagan's Baloney Detection Kit have in common?"
  "hard-problem|negative-control|Who proposed the Hard Problem of Consciousness?"
  "tunguska|negative-control|What is the Tunguska event?"
  "disputes|negative-control|List active disputes in the corpus."
  "apophenia-contested|mixed|What is contested about apophenia?"
  "searle-kahneman|mixed|How does Searle relate to Kahneman?"
)

for entry in "${QUESTIONS[@]}"; do
  IFS='|' read -r tag class q <<< "$entry"
  for i in $(seq 1 "$TRIALS"); do
    f="$OUT/${tag}__${class}__jsonl-${HAS_JSONL}__trial-${i}.txt"
    echo "[$tag $class trial $i] -> $(basename "$f")"
    {
      echo "QUESTION: $q"
      echo "CLASS: $class"
      echo "JSONL_PRESENT: $HAS_JSONL"
      echo "---"
      go run ./cmd/query --ask "$q" . 2>&1
    } > "$f" || echo "  (errored, see file)"
  done
done

echo "saved: $OUT"
echo "$OUT" > /tmp/jsonl-battery-latest.txt
