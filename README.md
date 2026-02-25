# Merkle Evidence Vault

This repository contains the Merkle Evidence Vault: an append-only evidence
store backed by an RFC 6962 binary Merkle tree, Ed25519-signed checkpoints,
and an offline-verifiable `.evb` bundle format.

Quick start (local dev):

```bash
export MERKLE_SIGNING_KEY_HEX=$(openssl rand -hex 32)
make dev-up          # Postgres + Redis + Redpanda
make migrate         # Run DB migrations
make build-all       # Rust + Go + React
make test            # Full test suite
make confidence      # Confidence gate (must pass)
```

See `CONFIDENCE.md` for the confidence assessment and production readiness.
