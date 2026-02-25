# ADR-002: Signing scheme

Decision: Use Ed25519 (RFC 8032) for Signed Tree Heads (STH).

Rationale: Ed25519 provides compact deterministic signatures, broad
library support in Rust and Go, and sufficient security for STHs.
