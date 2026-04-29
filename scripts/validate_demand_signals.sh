#!/usr/bin/env bash
# Validation battery for bias-state and calibration-marker context injections.
# Runs 6 questions × TRIALS × 2 conditions (signals present / stashed).
#
# Signal files tested:
#   .metabolism-bias-state.json      (AvailabilityHeuristic + SurvivorshipBias triggered)
#   .metabolism-calibration-state.json  (DimaraTaskBasedClassification + LakeChekoIsCrater challenged)
#
# Question design:
#   bias-benefit (2):     questions where Wikipedia concentration should prompt hedging
#   cal-benefit (3):      questions about challenged/corroborated hypotheses
#   negative-control (1): question where neither signal should change the answer
#
# Scoring rubric (manual, after run):
#   bias-benefit WITH: mentions provenance concentration or hedges on confidence
#   cal-benefit WITH:  mentions challenge signal or adjusts confidence accordingly
#   negative-control:  content equivalent in both conditions (tie = instrument OK)
set -euo pipefail

OUT=/tmp/demand-signals-$(date +%s)
mkdir -p "$OUT"
TRIALS=${TRIALS:-2}
echo "battery dir: $OUT  trials: $TRIALS"

BIAS_FILE=".metabolism-bias-state.json"
CAL_FILE=".metabolism-calibration-state.json"
STASH_DIR="/tmp/demand-signals-stash-$$"
mkdir -p "$STASH_DIR"

# tag|class|question
QUESTIONS=(
  "consciousness-reliability|bias-benefit|How reliable is the KB's coverage of consciousness theories? Are there gaps or source concentration issues?"
  "predictive-processing-evidence|bias-benefit|What is the evidence base for predictive processing in the KB? How confident should we be in these claims?"
  "dimara-classification|cal-benefit|What does Dimara's task-based classification say about cognitive bias taxonomy? How well-established is it?"
  "lake-cheko|cal-benefit|Is Lake Cheko evidence for the Tunguska event being a comet or asteroid impact? What does the KB say?"
  "hard-problem-support|cal-benefit|How well-supported is the hard problem of consciousness in external literature, based on what the KB knows?"
  "luhmann-zettelkasten|negative-control|What is Luhmann's Zettelkasten and how does it relate to the KB's design?"
)

run_condition() {
  local condition=$1  # "with" or "without"
  echo ""
  echo "=== Condition: $condition signals ==="

  if [ "$condition" = "without" ]; then
    [ -f "$BIAS_FILE" ] && cp "$BIAS_FILE" "$STASH_DIR/" && rm "$BIAS_FILE"
    [ -f "$CAL_FILE" ]  && cp "$CAL_FILE"  "$STASH_DIR/" && rm "$CAL_FILE"
  fi

  for entry in "${QUESTIONS[@]}"; do
    IFS='|' read -r tag class q <<< "$entry"
    for i in $(seq 1 "$TRIALS"); do
      f="$OUT/${tag}__${class}__signals-${condition}__trial-${i}.txt"
      echo "  [$tag $class trial $i]"
      {
        echo "QUESTION: $q"
        echo "CLASS: $class"
        echo "SIGNALS: $condition"
        echo "---"
        go run ./cmd/query --ask "$q" . 2>&1
      } > "$f" || echo "  (errored)"
    done
  done

  if [ "$condition" = "without" ]; then
    [ -f "$STASH_DIR/$BIAS_FILE" ] && cp "$STASH_DIR/$BIAS_FILE" .
    [ -f "$STASH_DIR/$(basename $CAL_FILE)" ] && cp "$STASH_DIR/$(basename $CAL_FILE)" .
  fi
}

run_condition "with"
run_condition "without"

rm -rf "$STASH_DIR"
echo ""
echo "saved: $OUT"
echo "$OUT" > /tmp/demand-signals-latest.txt
