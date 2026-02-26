#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
# Build release and run harness for 60s by default
cargo build --release
# Allow passing a duration as first arg
DUR=${1:-60}
# Run the harness with the requested duration
cargo run --release -- --duration-secs "$DUR"
