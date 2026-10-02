#!/bin/bash
# The research behind "how long would one sorter take to collect a set", and
# the facts beside it: runs pile's commands and facts.py over a data folder
# and writes everything to an output folder. Run from the repo, after
# scripts/build.sh. LOTS names the lots (streams) in the data folder's
# lots.json, comma separated.
#
#   LOTS=mine,everything scripts/research/run.sh DATA OUT
set -euo pipefail
cd "$(dirname "$0")/../.."
DATA=${1:?the data folder}
OUT=${2:?the output folder}
LOTS=${LOTS:-mine,everything}
mkdir -p "$OUT"
SETS=$(grep -v '^#' scripts/research/sets.txt | cut -d' ' -f1 | paste -sd, -)
lotargs=()
for lot in ${LOTS//,/ }; do
	bin/export-pieces -data "$DATA" -lot "$lot" > "$OUT/$lot.csv"
	lotargs+=(--lot "$lot=$OUT/$lot.csv")
done
python3 scripts/research/facts.py --data "$DATA" "${lotargs[@]}" --out "$OUT/facts.json" | tee "$OUT/facts.txt"
bin/collect-time -data "$DATA" -lots "$LOTS" -sets "$SETS" -resamples "${RESAMPLES:-200}" -json "$OUT/collect-time.json" | tee "$OUT/collect-time.md"
