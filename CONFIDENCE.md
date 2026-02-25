# CONFIDENCE — Merkle Evidence Vault v0.1.1

System effective confidence: 0.82 (development -> hardened test harness)

Summary:

- Integration and end-to-end test suites (property, integration, e2e) now pass
	locally using the provided `docker-compose` stack or the local `vault-api` server.
- Added minimal in-memory implementations in `services/vault-api` to satisfy
	test flows: ingest, proof, audit listing, and a simple checkpoint endpoint.

Key remaining blockers (prevent production claim):

- I4 HIGH: frontend proof display still renders proof data without a CSP
	+ DOMPurify sanitization audit and remediation — critical for removal of the
	HIGH risk (XSS) item.
- merkle-engine fuzz coverage incomplete — cargo-fuzz targets should be run
	and results verified (see CONFIDENCE.yaml -> confidence_path_to_production).
- Authentication & RBAC are currently test-shims (substring checks). Replace
	with JWKS/JWT validation and role enforcement before production.

Notes on what we built:

- Developer ergonomics: `go.work` and local module wiring to ease editor imports.
- Dev infra: `ops/docker/docker-compose.yml` updated to public images (Kafka
	+ Zookeeper), merkle-engine Dockerfile updated for reproducible builds.
- Tests: property + integration + e2e added/updated; e2e exercises ingest →
	commit → proof → checkpoints → audit flow and passes with example tokens.

See `CONFIDENCE.yaml` for artifact-level scores and `NEXT_STEPS.md` for
prioritized remediation and follow-ups.
