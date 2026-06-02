# Aethel Workspace

**Open-source e-office platform for government and institutional document workflows.**

Aethel replaces manual paper-based correspondence with a structured digital system: every inbound letter is tracked, routed to the right desk, annotated with a cryptographically-chained minute sheet, and logged to a tamper-evident audit ledger — all without touching a config file after first boot.

Designed for organizations where accountability, chain-of-custody, and audit compliance aren't optional.

---

## Why Aethel

Most "workflow" tools are generic project trackers dressed up with custom fields. Aethel is built around three real institutional processes:

**Diarization** — Inbound correspondence is received, assigned a tracking number, routed by configurable rules (document type, sender, urgency), and handed to the correct department. Every routing decision is recorded.

**Green Noting** — Each dispatch has a minute sheet where staff append sequential notes. Notes are chained with SHA-256 hashes: no note can be silently altered or deleted without breaking the chain. Approvals are countersigned.

**Audit Ledger** — An append-only, monthly-partitioned event log with a checksum chain spanning the entire ledger. The `/admin/audit-log/verify` endpoint re-computes the chain on demand and reports exactly which row was tampered with, if any.

---

## How It Works

Configuration is split between two layers: **seed** (YAML, set once by IT at deployment) and **runtime** (PostgreSQL, editable by admins in the browser — no recompilation, no restart).

```
Nuxt 4 frontend  ──►  Go backend (chi + RBAC + JWT)  ──►  PostgreSQL 16
                              │
                    YAML blueprints (seed only)
                    branding, nav, DB connection
```

The backend serves `GET /api/v1/config` with a 5-minute per-org in-memory cache embedded in the initial SSR HTML. Admin changes to branding, navigation, or feature flags propagate to all users on next page load — no CDN invalidation, no build step.

---

## Feature Highlights

- **Role-based access control** — three built-in roles (ADMIN / RECEPTION / USER) with granular permission strings; `sys_admin` gating for sensitive audit views
- **Routing rule engine** — priority-ordered rules matching on document type, sender organization, and urgency level; evaluated at dispatch creation
- **Runtime branding** — IT admins change the organization's color scheme, logo, and font from the browser; takes effect immediately for all users
- **Escalation worker** — background goroutine evaluates overdue dispatches on a configurable interval and transitions them to ESCALATED with a full audit trail
- **SSE notifications** — real-time delivery to connected browser clients via a goroutine-safe SSE broker; no WebSocket dependency
- **42 database migrations** — full schema with multi-tenancy (`organization_id` on every table), RLS-ready, monthly-partitioned audit ledger

---

## Tech Stack

| Layer | Technology |
|---|---|
| Frontend | Nuxt 4, Vue 3, Nuxt UI v4, Pinia, Tailwind CSS v4 |
| Backend | Go, chi router, cobra CLI, zerolog, Argon2id, JWT |
| Database | PostgreSQL 16 (uuid-ossp, pgcrypto, pg_trgm) |
| Auth | Argon2id password hashing, JWT access + opaque refresh tokens |
| Infrastructure | Docker Compose, Kubernetes (k8s/), GitHub Actions CI/CD |
| Testing | Vitest + Playwright (frontend), Go race detector + integration tests (backend) |

---

## Prerequisites

| Tool | Version |
|---|---|
| Go | 1.22+ |
| Node.js | 20+ |
| pnpm | 9+ |
| Docker + Docker Compose | v2+ |
| PostgreSQL | 16+ (or use Docker) |

---

## Quick Start

```bash
# 1. Copy and configure environment
cp .env.example .env

# 2. Start all services (PostgreSQL + backend + frontend)
make dev

# 3. Apply database migrations
make migrate-up
```

| Service | URL |
|---|---|
| Frontend | http://localhost:3000 |
| Backend API | http://localhost:8080/api/v1 |
| Health probe | http://localhost:8080/healthz |

---

## Commands

### Development

```bash
make dev                # Start all services via Docker Compose
make dev-fe             # Nuxt dev server only (hot reload)
make dev-be             # Go backend with Air hot reload
make dev-db             # PostgreSQL container only
make dev-down           # Stop all containers (volumes preserved)
make dev-reset          # Full reset — wipe volumes, fresh database
```

### Testing

```bash
make test               # All tests (frontend + backend)
make test-fe            # Vitest unit + component tests
make test-be            # Go tests with race detector
make test-e2e           # Playwright end-to-end
```

### Database

```bash
make migrate-up         # Apply all pending migrations
make migrate-down       # Roll back the last migration
make migrate-status     # Show applied / pending migrations
make migrate-validate   # Dry-run: render templates, validate SQL (no DB writes)
make db-shell           # Open psql into the dev database
make db-dump            # Dump dev database to /tmp/aethel-dump-<timestamp>.sql
```

### Build

```bash
make build              # Build frontend + backend
make build-docker       # Build both Docker images locally
```

### Backend CLI (direct)

```bash
cd aethel-core
go run ./cmd/aethel serve              # Start HTTP server on :8080
go run ./cmd/aethel migrate up         # Apply all pending migrations
go run ./cmd/aethel migrate status     # List applied / pending
go run ./cmd/aethel migrate validate   # Validate without writing
```

---

## Configuration

### Environment variables

| Variable | Required | Description |
|---|---|---|
| `AETHEL_DB_PASSWORD` | Yes | PostgreSQL password |
| `AETHEL_JWT_SECRET` | Yes (prod) | HS256 signing secret |
| `AETHEL_ENV` | No | `development` (default) or `production` |
| `AETHEL_PORT` | No | HTTP listen port (default `8080`) |
| `AETHEL_DB_DSN` | No | Full DSN (overrides individual fields) |
| `AETHEL_ARGON2_MEMORY_KIB` | No | Argon2id memory cost in KiB (default `65536`) |

See `.env.example` for the full reference.

### Blueprint files (IT-facing)

Only two YAML files need to be touched by IT at deployment:

| File | Purpose |
|---|---|
| `blueprints/server-database.yaml` | DB connection, pool config, schema aliases |
| `blueprints/ui-theme.yaml` | Initial branding seed — loaded once at first boot |

Everything else (navigation, feature flags, branding overrides, org profile) is managed at runtime through `/admin/*` and stored in PostgreSQL.

---

## Project Structure

```
aethel-workspace/
├── aethel-view/          # Nuxt 4 frontend (17 pages, 3 roles)
├── aethel-core/          # Go backend
│   ├── cmd/aethel/       # CLI entry point (cobra)
│   └── internal/
│       ├── api/          # HTTP server, routes, handlers
│       ├── config/       # Runtime config cache (5-min TTL per org)
│       ├── database/     # Migrator, query registry, connection pool
│       ├── domain/       # Domain types and repository interfaces
│       ├── rbac/         # Permission middleware
│       ├── service/      # Business logic (auth, dispatch, workflow)
│       ├── transport/    # SSE broker
│       └── worker/       # Background workers (escalation)
├── blueprints/           # IT-facing YAML seed files + schemas
├── docs/                 # Architecture docs, ER diagram, guides
├── k8s/                  # Kubernetes manifests (namespace: aethel-workspace)
├── aethel-scripts/       # Dev/ops shell scripts
├── docker-compose.yml    # Local dev stack
└── Makefile              # All dev/build/test/deploy commands (make help)
```

---

## Documentation

| Document | Location |
|---|---|
| Architecture overview | `docs/architecture/` |
| ER diagram | `docs/db-design.mmd` |
| API routes | `docs/architecture/architecture-api-routes.md` |
| Security architecture | `docs/architecture/architecture-security.md` |
| IT customization guide | `docs/guides/it-customization-guide.md` |
| Go developer guide | `docs/guides/go-developer-guide.md` |
| DevOps tooling | `docs/devops/devops-tooling.md` |

---

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) for setup instructions, branch conventions, and testing requirements. All PRs must pass `go test ./...` and `pnpm test` with CI green.

---

## License

Apache 2.0. See [LICENSE](./LICENSE) and [NOTICE](./NOTICE) for terms.
