#!/usr/bin/env bash
set -euo pipefail

# Install cargo-fuzz if missing and run the tree_append_leaf fuzz target for a bounded time.
if ! command -v cargo-fuzz >/dev/null 2>&1; then
  echo "cargo-fuzz not found; installing..."
  cargo install cargo-fuzz || true
fi

# Run for 60 seconds by default, allow override via $FUZZ_TIME
TIMEOUT_SECONDS=${FUZZ_TIME:-60}

echo "Running cargo-fuzz (tree_append_leaf) for ${TIMEOUT_SECONDS}s..."
cargo fuzz run tree_append_leaf -- -max_total_time=${TIMEOUT_SECONDS}
