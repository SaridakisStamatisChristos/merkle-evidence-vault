#!/usr/bin/env bash
set -euo pipefail

# Replay minimized corpus with cargo-fuzz using deterministic single-run mode.
# Adjust FUZZ_TARGET to match your cargo-fuzz target name.
FUZZ_TARGET=tree_append_leaf
ROOT_DIR=$(dirname "$0")/..
cd "$ROOT_DIR/services/merkle-engine" || exit 1

if [ ! -d ../../fuzz/corpus/minimized ]; then
  echo "No minimized corpus found at ../../fuzz/corpus/minimized"
  exit 1
fi

for f in ../../fuzz/corpus/minimized/*; do
  echo "Replaying $f"
  # cargo-fuzz run target with single-run and provided input file
  cargo fuzz run "$FUZZ_TARGET" "$f" -runs=1 -dict="../../fuzz/dict.txt" || true
done

echo "Replay complete"
