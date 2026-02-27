#!/usr/bin/env bash
set -euo pipefail

TARGET="${1:-tree_append_leaf}"
ARTIFACT_DIR="${2:-services/merkle-engine/fuzz/artifacts/$TARGET}"
OUT_DIR="${3:-fuzz/corpus/minimized-from-crashes}"

if [ ! -d "$ARTIFACT_DIR" ]; then
  echo "No artifact directory found at $ARTIFACT_DIR; skipping minimization"
  exit 0
fi

mkdir -p "$OUT_DIR"

shopt -s nullglob
files=("$ARTIFACT_DIR"/*)
if [ ${#files[@]} -eq 0 ]; then
  echo "No crash artifacts found in $ARTIFACT_DIR; skipping minimization"
  exit 0
fi

for f in "${files[@]}"; do
  base=$(basename "$f")
  out="$OUT_DIR/minimized-$base"
  echo "Minimizing $f -> $out"
  (
    cd services/merkle-engine/fuzz
    cargo +nightly fuzz tmin "$TARGET" "../../../$f" -o "../../../$out" -- -dict=../../../fuzz/dict.txt
  ) || true
done

echo "Crash minimization run complete"
