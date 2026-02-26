Merkle-engine fuzz targets
=========================

This folder contains a libFuzzer-style fuzz target for `MerkleTree::append` and `root`.

Quick start (local):

```bash
# from services/merkle-engine
cd fuzz
./run_cargo_fuzz.sh
```

Notes:
- The script will attempt to `cargo install cargo-fuzz` if it's not available.
- For CI, prefer running `cargo fuzz` on a runner with Rust toolchain and sanitizers (ASAN/UBSAN).
