# Architecture — Merkle Evidence Vault

## System overview

The diagram below is the primary GitHub-rendered view of the runtime and verification boundaries. It complements the detailed C4 views that follow.

```mermaid
flowchart LR
    %% External actors
    subgraph ACTORS["External actors"]
        CI["CI/CD pipeline"]
        CO["Compliance officer"]
        AU["External auditor"]
    end

    %% External dependencies
    subgraph EXT["External dependencies"]
        OIDC["OIDC / JWKS provider"]
        OBJ[("Object storage\nS3 / GCS / local")]
    end

    %% Main product boundary
    subgraph VAULT["Merkle Evidence Vault"]
        direction TB

        UI["audit-dashboard\nReact / TypeScript"]

        subgraph ACCESS["Access & policy"]
            API["vault-api\nGo · REST / gRPC"]
            REDIS[("Redis\nrate limiting")]
        end

        subgraph CORE["Evidence & Merkle core"]
            DB[("PostgreSQL\nappend-only evidence schema")]
            BUS[("Redpanda\nevent stream")]
            CONSUMER["pipeline consumer\nidempotent appender"]
            MERKLE["merkle-engine\nRust · RFC 6962"]
            CHECKPOINT["checkpoint-svc\nperiodic signed tree heads"]
        end

        subgraph VERIFY["Portable verification"]
            BUNDLE[".evb evidence bundle\nmanifest + proofs + checkpoint"]
            CLI["verifier-cli\nGo · offline / no network"]
        end

        subgraph OBS["Operations & observability"]
            PROM["Prometheus"]
            GRAF["Grafana"]
        end
    end

    %% Ingest / query path
    CI -->|"submit build evidence"| API
    CO -->|"query / export"| UI
    UI <-->|"HTTPS"| API
    OIDC -->|"JWT keys / identity"| API
    API -->|"rate-limit state"| REDIS
    API -->|"evidence metadata"| DB
    API -->|"publish ingest event"| BUS
    BUS -->|"consume"| CONSUMER
    CONSUMER -->|"append leaf"| MERKLE
    API <-->|"proof / tree RPC"| MERKLE
    MERKLE -->|"tree state"| CHECKPOINT
    CHECKPOINT -->|"signed checkpoints"| DB

    %% Export / offline verification path
    API -->|"export"| BUNDLE
    BUNDLE -->|"archive / retrieve"| OBJ
    AU -->|"verify independently"| CLI
    OBJ -.->|"download bundle"| CLI
    BUNDLE -.->|"local input"| CLI

    %% Observability
    API -.->|"metrics"| PROM
    MERKLE -.->|"metrics"| PROM
    CHECKPOINT -.->|"metrics"| PROM
    PROM --> GRAF

    classDef actor fill:#f8fafc,stroke:#475569,stroke-width:1.5px,color:#0f172a;
    classDef access fill:#e0f2fe,stroke:#0284c7,stroke-width:1.5px,color:#0c4a6e;
    classDef core fill:#ecfdf5,stroke:#059669,stroke-width:1.5px,color:#064e3b;
    classDef verify fill:#f5f3ff,stroke:#7c3aed,stroke-width:1.5px,color:#4c1d95;
    classDef external fill:#fff7ed,stroke:#ea580c,stroke-width:1.5px,color:#7c2d12;
    classDef ops fill:#f8fafc,stroke:#64748b,stroke-width:1.5px,color:#334155;

    class CI,CO,AU actor;
    class API,REDIS,UI access;
    class DB,BUS,CONSUMER,MERKLE,CHECKPOINT core;
    class BUNDLE,CLI verify;
    class OIDC,OBJ external;
    class PROM,GRAF ops;
```

### Architectural intent

- **Append-only evidence path:** ingestion flows through `vault-api`, the event stream, the idempotent consumer, and `merkle-engine`; evidence metadata is retained in PostgreSQL.
- **Cryptographic checkpointing:** `checkpoint-svc` periodically signs tree state so proofs can be bound to a signed checkpoint.
- **Portable verification:** exported `.evb` bundles are designed to be checked by `verifier-cli` without network access or trust in the running service.
- **Separated trust boundaries:** identity comes from the configured OIDC/JWKS provider, object storage is an external availability layer, and offline verification remains out-of-band.
- **Operational visibility:** Prometheus and Grafana observe the runtime but are not part of the proof trust root.

## C4 Level 1 — System Context
```
┌─────────────────────────────────────────────────────────────────────┐
│  External Actors                                                    │
│                                                                     │
│  [CI/CD Pipeline]  [Compliance Officer]  [External Auditor]        │
│         │                  │                      │                 │
│         └──────────────────┴──────────────────────┘                │
│                            │ HTTPS / JWT                            │
│                            ▼                                        │
│              ┌─────────────────────────┐                           │
│              │  MERKLE EVIDENCE VAULT  │                           │
│              │    (this system)        │                           │
│              └─────────────────────────┘                           │
│                            │                                        │
│              ┌─────────────┴──────────┐                            │
│          [OIDC Provider]    [Object Storage]                        │
│          (external)         (S3/GCS/local)                         │
└─────────────────────────────────────────────────────────────────────┘
```

## C4 Level 2 — Container Diagram
```
┌──────────────────────────────────────────────────────────────────────────┐
│  MERKLE EVIDENCE VAULT                                                   │
│                                                                          │
│  ┌─────────────────┐    REST/gRPC      ┌──────────────────────────┐     │
│  │  audit-dashboard│ ◄────HTTPS──────► │       vault-api          │     │
│  │  (React/TS)     │                   │  (Go, chi + tonic)       │     │
│  │  :5173 dev      │                   │  :8443 HTTPS :9443 gRPC  │     │
│  └─────────────────┘                   └────────────┬─────────────┘     │
│                                                     │                    │
│                          ┌──────────────────────────┼──────────────┐    │
│                          │                          │              │    │
│                     Unix/mTLS gRPC            TCP+TLS         Kafka│    │
│                          │                          │              │    │
│                          ▼                          ▼              ▼    │
│              ┌───────────────────┐   ┌──────────────────┐  ┌──────────┐│
│              │  merkle-engine    │   │    postgres:5432  │  │redpanda  ││
│              │  (Rust, RFC 6962) │   │  append-only      │  │:9092     ││
│              │  :9444 gRPC       │   │  Merkle schema    │  │          ││
│              └────────┬──────────┘   └──────────────────┘  └──────────┘│
│                       │                                          │       │
│              ┌────────▼──────────┐              ┌───────────────▼─────┐ │
│              │  checkpoint-svc   │              │  pipeline consumer  │ │
│              │  (Go, periodic    │              │  (Go, idempotent    │ │
│              │   STH signing)    │              │   Merkle appender)  │ │
│              └───────────────────┘              └─────────────────────┘ │
│                                                                          │
│              ┌──────────────────────────────────────────────────────┐   │
│              │  verifier-cli (Go, offline, no network, ships as     │   │
│              │  standalone binary — used by auditors out-of-band)   │   │
│              └──────────────────────────────────────────────────────┘   │
│                                                                          │
│  Supporting: Redis:6379 (rate-limit), Prometheus:9090, Grafana:3000     │
└──────────────────────────────────────────────────────────────────────────┘
```

## C4 Level 3 — vault-api Component Diagram
```
vault-api
│
├── cmd/server/main.go          Entry point, dependency wiring
│
├── config/                     YAML + env-var config loading
│
├── middleware/
│   ├── auth.go                 JWKS cache → JWT validation → role extraction
│   ├── ratelimit.go            Redis sliding window, fail-open
│   ├── logging.go              Structured zerolog per-request
│   └── audit.go                Append-only audit log write on every request
│
├── handler/
│   ├── ingest.go               Validate → dedup → persist → enqueue
│   ├── query.go                Evidence lookup, proof delegation, checkpoints
│   ├── audit.go                Paginated audit trail read
│   ├── export.go               Bundle creation + download
│   └── health.go               Liveness + readiness probes
│
├── internal/
│   ├── merklerpc/client.go     gRPC client → merkle-engine (implements merkle.Engine)
│   └── pipeline/
│       ├── publisher.go        Redpanda ingest event writer
│       └── consumer.go         Idempotent Merkle append consumer
│
└── domain/
    ├── evidence/types.go       Evidence value object, LeafData() binding
    ├── merkle/interface.go     Engine interface (testable, mockable)
    ├── checkpoint/types.go     STH types + policy
    └── bundle/format.go        .evb manifest + archive layout constants
```
