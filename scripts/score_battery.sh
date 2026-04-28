#!/usr/bin/env bash
# Pair up with-JSONL and without-JSONL outputs from two battery runs.
# For each question, side-by-side compare the answer body. Manual scoring after.
#
# Usage: score_battery.sh <with-dir> <without-dir>
set -e
WITH=$1
WITHOUT=$2
if [ -z "$WITH" ] || [ -z "$WITHOUT" ]; then
  echo "usage: $0 <with-jsonl-dir> <without-jsonl-dir>"
  exit 1
fi

OUT=/tmp/battery-paired-$(date +%s).md
echo "# JSONL validation battery — paired comparison" > "$OUT"
echo "" >> "$OUT"
echo "with: $WITH" >> "$OUT"
echo "without: $WITHOUT" >> "$OUT"
echo "" >> "$OUT"

# extract unique tag__class prefixes from with-dir
prefixes=$(ls "$WITH" 2>/dev/null | sed 's/__jsonl-.*//' | sort -u)

for p in $prefixes; do
  echo "" >> "$OUT"
  echo "---" >> "$OUT"
  echo "## $p" >> "$OUT"
  echo "" >> "$OUT"
  for f in "$WITH"/${p}__jsonl-yes__trial-*.txt; do
    [ -f "$f" ] || continue
    echo "" >> "$OUT"
    echo "### WITH ($(basename "$f"))" >> "$OUT"
    echo '```' >> "$OUT"
    # skip the leading metadata block (QUESTION/CLASS/JSONL_PRESENT/---)
    awk '/^---$/{flag=1; next} flag' "$f" >> "$OUT"
    echo '```' >> "$OUT"
  done
  for f in "$WITHOUT"/${p}__jsonl-no__trial-*.txt; do
    [ -f "$f" ] || continue
    echo "" >> "$OUT"
    echo "### WITHOUT ($(basename "$f"))" >> "$OUT"
    echo '```' >> "$OUT"
    awk '/^---$/{flag=1; next} flag' "$f" >> "$OUT"
    echo '```' >> "$OUT"
  done
done

echo "" >> "$OUT"
echo "wrote: $OUT"
echo "$OUT"
