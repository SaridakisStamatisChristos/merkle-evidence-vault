# Changelog

All notable changes to Merkle Evidence Vault are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versioning follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Planned (v0.2)
- HSM integration for signing key (P1 — ADR-002)
- mTLS enforcement between vault-api and merkle-engine (P1 — threat S2)
- CSP headers + DOMPurify on frontend proof display (P0 — threat I4)
- Per-IP rate limiting in addition to per-subject (P2)
- Leader election for checkpoint-svc (currently single-replica)
- RFC 3161 TSA external checkpoint anchoring

---

## [0.1.0] — 2026-02-24

### Added

**Core Merkle Engine (Rust)**
- RFC 6962 Binary Merkle Tree with SHA-256 leaf and node hashing
- Append-only leaf ingestion with 4 MiB payload cap
- Inclusion proof generation and offline verification
- Consistency proof generation and verification
- Ed25519 Signed Tree Head (STH) with deterministic signing
- gRPC service layer (tonic) exposing all tree operations
- Property-based tests via proptest (6 properties × 500 trials)
- Fuzz target skeleton for `tree::append_leaf`

... (truncated for brevity in repo) ...
