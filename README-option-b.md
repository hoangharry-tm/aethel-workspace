<div align="center">
  <img src="docs/assets/banner.svg" width="100%" alt="Aethel Workspace"/>
</div>

<div align="center">
  <br/>
  <em>Document governance infrastructure for organizations that cannot afford ambiguity.</em>
  <br/><br/>

  [![Go](https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
  [![Nuxt](https://img.shields.io/badge/Nuxt-4-00DC82?style=flat-square&logo=nuxt.js&logoColor=white)](https://nuxt.com)
  [![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?style=flat-square&logo=postgresql&logoColor=white)](https://www.postgresql.org)
  [![License](https://img.shields.io/badge/License-Apache_2.0-4f46e5?style=flat-square)](./LICENSE)
  [![OpenAPI](https://img.shields.io/badge/OpenAPI-3.1_·_58_endpoints-6BA539?style=flat-square&logo=openapiinitiative&logoColor=white)](./docs/architecture/architecture-api-routes.md)
  [![Docker](https://img.shields.io/badge/Docker-ready-2496ED?style=flat-square&logo=docker&logoColor=white)](./docker-compose.yml)
  [![Self-hosted](https://img.shields.io/badge/self--hosted-by_design-0a0a0a?style=flat-square)](./docs/guides/it-customization-guide.md)

  <br/>

  <table>
    <tr>
      <td align="center"><strong>17</strong><br/><sub>UI pages</sub></td>
      <td align="center"><strong>3</strong><br/><sub>user roles</sub></td>
      <td align="center"><strong>58</strong><br/><sub>API endpoints</sub></td>
      <td align="center"><strong>42</strong><br/><sub>DB migrations</sub></td>
      <td align="center"><strong>3</strong><br/><sub>domain pillars</sub></td>
      <td align="center"><strong>&lt; 200 ms</strong><br/><sub>p99 latency target</sub></td>
    </tr>
  </table>
</div>

---

> [!NOTE]
> Go 1.26 · chi router · zerolog · cobra CLI. PostgreSQL 16 with monthly range-partitioned audit ledger, SHA-256 cryptographic green-note chain, pg_trgm full-text search, and 42 migration files rendered via Go `text/template` for configurable schema aliases. Nuxt 4 SSR with zero client-side config round-trips — runtime branding and nav tree are embedded in the initial HTML payload from a 5-minute in-memory config cache. 9-layer middleware stack: Recovery → RequestID → Logger → Rate Limit → CORS → Auth → RBAC → CSRF → Body Limit. Self-hosted only. No external services required at runtime.

---

## Architecture

```mermaid
flowchart LR
    subgraph Browser["Browser (Nuxt 4 SSR)"]
        FE["Vue 3 · Nuxt UI v4\n17 pages · 3 roles"]
        SSE_C["SSE Client\nreal-time events"]
    end

    subgraph Backend["Go Backend  (chi · zerolog)"]
        direction TB
        MW["9-layer Middleware\nRecovery · RequestID · Logger\nRate Limit · CORS · Auth · RBAC · CSRF · Body Limit"]
        API["58 REST Endpoints\nOpenAPI 3.1 · Scalar UI"]
        CFG["Config Cache\n5-min TTL · runtime branding"]
        BROKER["SSE Broker\nper-user goroutine-safe channels"]
        WORKER["Escalation Worker\nconfigurable tick interval"]
    end

    subgraph DB["PostgreSQL 16"]
        SCHEMA["20 tables · 42 migrations\nmonthly-partitioned audit ledger\ncryptographic green-note chain\npg_trgm full-text search"]
    end

    subgraph Seed["YAML Blueprints (set once)"]
        YAML["server-database.yaml\nui-theme.yaml"]
    end

    FE -->|"JWT Bearer\nhttpOnly refresh cookie"| MW
    MW --> API
    API --> DB
    CFG -->|"GET /api/v1/config\nembedded in SSR HTML"| FE
    CFG --> DB
    BROKER -->|"text/event-stream"| SSE_C
    WORKER --> DB
    Seed -->|"first-boot seed\nON CONFLICT DO NOTHING"| DB
```

---

## Three Pillars

| Pillar | Mechanism | Key table |
|---|---|---|
| **DAK Diarization** | Rule-engine routes dispatch by `(doc_type × sender × urgency)`. Escalation worker fires on configurable deadline threshold. | `dispatches`, `dispatch_events`, `routing_rules` |
| **Green Noting Canvas** | Minute sheets with append-only notes. Each note: `SHA-256(content ‖ seq ‖ author_id ‖ previous_hash)`. Role-gated `workflow.approve` permission for countersigning. | `minute_sheets`, `green_notes` |
| **RBAC Audit Ledger** | `bigserial` PK (not UUID) for partition-efficient access. `previous_checksum` chains entire ledger. `GET /api/v1/audit-log/verify` re-computes on demand and returns the first broken row ID. | `audit_ledger` (monthly partitions) |

---

## Quick Start

```bash
# 1. Clone and configure
git clone https://github.com/your-org/aethel-workspace.git
cd aethel-workspace
cp .env.example .env          # set AETHEL_JWT_SECRET and AETHEL_DB_PASSWORD

# 2. Start everything
make dev                      # PostgreSQL + Go backend + Nuxt frontend via Docker Compose

# 3. Apply database schema
make migrate-up               # 42 migrations, ~2 seconds on first run

# 4. Create initial sys_admin account
go run ./cmd/aethel bootstrap-admin   # prints temporary password — change immediately
```

| Service | URL | Notes |
|---|---|---|
| Frontend | http://localhost:3000 | Nuxt 4 · hot-reload in dev |
| Backend API | http://localhost:8080/api/v1 | JWT-authenticated |
| API documentation | http://localhost:8080/api/docs | Scalar UI — interactive |
| Health probe | http://localhost:8080/healthz | liveness |
| Readiness probe | http://localhost:8080/readyz | db.Ping() included |

---

## API at a Glance

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/config` | Runtime config (branding, nav, feature flags) — embedded in SSR HTML |
| `POST` | `/api/v1/auth/login` | Argon2id credential check; issues JWT access token + httpOnly refresh cookie |
| `POST` | `/api/v1/auth/logout` | Rotates refresh token, clears cookie |
| `GET` | `/api/v1/dispatches` | Paginated dispatch list; supports `?status=`, `?priority=`, `?q=` (pg_trgm) |
| `POST` | `/api/v1/dispatches` | Create dispatch — triggers routing rule evaluation and audit event |
| `GET` | `/api/v1/audit-ledger` | Paginated audit log (`sys_admin` only); supports time-range filters |

Full OpenAPI 3.1 spec with all 58 operationIds at [`docs/architecture/architecture-api-routes.md`](./docs/architecture/architecture-api-routes.md). Interactive Scalar UI available at `http://localhost:8080/api/docs` when the backend is running.

---

## Tech Stack

<table>
  <tr>
    <td><strong>Frontend</strong></td>
    <td>
      <img src="https://img.shields.io/badge/Nuxt_4-00DC82?style=flat-square&logo=nuxt.js&logoColor=white" alt="Nuxt 4"/>
      <img src="https://img.shields.io/badge/Vue_3-4FC08D?style=flat-square&logo=vue.js&logoColor=white" alt="Vue 3"/>
      <img src="https://img.shields.io/badge/TypeScript-3178C6?style=flat-square&logo=typescript&logoColor=white" alt="TypeScript"/>
      <img src="https://img.shields.io/badge/Tailwind_v4-06B6D4?style=flat-square&logo=tailwindcss&logoColor=white" alt="Tailwind CSS v4"/>
      <img src="https://img.shields.io/badge/Pinia-FFD859?style=flat-square&logo=pinia&logoColor=black" alt="Pinia"/>
    </td>
  </tr>
  <tr>
    <td><strong>Backend</strong></td>
    <td>
      <img src="https://img.shields.io/badge/Go_1.26-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go"/>
      <img src="https://img.shields.io/badge/chi_router-00ADD8?style=flat-square" alt="chi"/>
      <img src="https://img.shields.io/badge/zerolog-00ADD8?style=flat-square" alt="zerolog"/>
      <img src="https://img.shields.io/badge/cobra_CLI-00ADD8?style=flat-square" alt="cobra"/>
    </td>
  </tr>
  <tr>
    <td><strong>Database</strong></td>
    <td>
      <img src="https://img.shields.io/badge/PostgreSQL_16-336791?style=flat-square&logo=postgresql&logoColor=white" alt="PostgreSQL"/>
      <img src="https://img.shields.io/badge/uuid--ossp-336791?style=flat-square" alt="uuid-ossp"/>
      <img src="https://img.shields.io/badge/pgcrypto-336791?style=flat-square" alt="pgcrypto"/>
      <img src="https://img.shields.io/badge/pg__trgm-336791?style=flat-square" alt="pg_trgm"/>
    </td>
  </tr>
  <tr>
    <td><strong>Auth</strong></td>
    <td>
      <img src="https://img.shields.io/badge/Argon2id-password_hashing-dc2626?style=flat-square" alt="Argon2id"/>
      <img src="https://img.shields.io/badge/JWT_HS256-access_tokens-dc2626?style=flat-square" alt="JWT"/>
      <img src="https://img.shields.io/badge/httpOnly_cookie-refresh_tokens-dc2626?style=flat-square" alt="httpOnly"/>
      <img src="https://img.shields.io/badge/CSRF_double--submit-dc2626?style=flat-square" alt="CSRF"/>
    </td>
  </tr>
  <tr>
    <td><strong>Infrastructure</strong></td>
    <td>
      <img src="https://img.shields.io/badge/Docker-2496ED?style=flat-square&logo=docker&logoColor=white" alt="Docker"/>
      <img src="https://img.shields.io/badge/Kubernetes-326CE5?style=flat-square&logo=kubernetes&logoColor=white" alt="Kubernetes"/>
      <img src="https://img.shields.io/badge/GitHub_Actions-2088FF?style=flat-square&logo=github-actions&logoColor=white" alt="GitHub Actions"/>
    </td>
  </tr>
  <tr>
    <td><strong>Testing</strong></td>
    <td>
      <img src="https://img.shields.io/badge/Vitest-6E9F18?style=flat-square&logo=vitest&logoColor=white" alt="Vitest"/>
      <img src="https://img.shields.io/badge/Playwright-2EAD33?style=flat-square&logo=playwright&logoColor=white" alt="Playwright"/>
      <img src="https://img.shields.io/badge/go_test_--race-00ADD8?style=flat-square" alt="go test -race"/>
    </td>
  </tr>
</table>

---

## Feature Reference

<details>
<summary><strong>DAK Diarization — Dispatch Tracking (expanded)</strong></summary>
<br/>

**Routing rule engine** — Rules are evaluated in priority order at dispatch creation. Each rule matches on any combination of: document type, sender organization, urgency level (`ROUTINE` / `PRIORITY` / `IMMEDIATE`). The first matching rule assigns the dispatch to the configured department. If no rule matches, the dispatch enters the unassigned inbox for manual routing.

**Dispatch lifecycle**

```
PENDING_ASSIGNMENT → UNDER_REVIEW → IN_TRANSIT → DELIVERED
                                               ↘ ESCALATED (worker fires)
                                               ↘ REJECTED
```

**Timeline events** — A unified `dispatch_events` table aggregates all routing decisions, handoffs, escalations, and status changes. The document detail page renders this as a live timeline without multi-table UNIONs.

**Escalation worker** — A background goroutine evaluates all dispatches older than the configured threshold. When a dispatch exceeds its deadline, its status transitions to `ESCALATED`, a notification is sent to the assigned department head, and an audit event is written. The evaluation interval is blueprint-configurable.

**Config cache** — Runtime-configurable branding (colors, fonts, logo), navigation structure, and feature flags are stored in PostgreSQL and served via `GET /api/v1/config` with a 5-minute in-memory TTL. The response is embedded in the initial SSR HTML — zero client-side config round-trips.

</details>

<details>
<summary><strong>Green Noting Canvas — Minute Sheets</strong></summary>
<br/>

Every dispatch automatically gets a minute sheet on creation. The minute sheet is the official record of institutional action taken on that document.

**Hash chain integrity**

Each green note is inserted with:
```
cryptographic_hash = SHA-256(content ‖ sequence_number ‖ author_id ‖ previous_hash)
```

The database stores both the note's own hash and the `previous_hash` of its predecessor. Any attempt to alter a note's content, swap two notes, or delete a note from the middle of the chain produces a hash mismatch detectable at the next verification pass.

**Approval countersigning** — Notes flagged as approvals require a role with `workflow.approve` permission. The approval is recorded with the signer's user ID and timestamp. Attempted approvals by under-privileged users are rejected with `403` and logged to the audit ledger.

</details>

<details>
<summary><strong>RBAC Audit Ledger — Tamper Detection</strong></summary>
<br/>

**Schema design choices:**
- `bigserial` primary key (not UUID) for partition-efficient sequential access
- `organization_id` stored as plain `uuid` without FK — audit records survive organization deletion
- `previous_checksum` chains rows: each new entry hashes `(event_type ‖ actor_id ‖ resource_id ‖ created_at ‖ previous_checksum)`
- Monthly `PARTITION BY RANGE(created_at)` — old partitions can be archived or moved to cold storage without touching the active partition

**Verification endpoint** — `GET /api/v1/audit-log/verify?from=&to=` (requires `sys_admin` role) re-fetches every row in the range and recomputes the checksum chain from scratch. Response:

```json
{
  "verified": false,
  "rows_checked": 4821,
  "first_broken_at": {
    "row_id": 1337,
    "created_at": "2026-05-14T09:23:11Z",
    "expected_checksum": "a3f9...",
    "actual_checksum": "b7c2..."
  }
}
```

**Tamper event types** — The ledger captures: `DISPATCH_CREATED`, `DISPATCH_ASSIGNED`, `DISPATCH_DELIVERED`, `GREEN_NOTE_APPENDED`, `PERMISSION_DENIED`, `SECURITY_BREACH_ATTEMPT`, `UNAUTHORIZED_ACCESS_BYPASSED`, `RBAC_ELEVATION_ATTEMPT`.

</details>

---

## Configuration

<details>
<summary><strong>Environment variables</strong></summary>
<br/>

| Variable | Required | Default | Description |
|---|---|---|---|
| `AETHEL_JWT_SECRET` | **Yes (prod)** | — | HS256 signing secret — min 32 bytes, base64-encoded. Server refuses to start if unset in production. |
| `AETHEL_DB_PASSWORD` | **Yes** | — | PostgreSQL password |
| `AETHEL_ENV` | No | `development` | `development` or `production` |
| `AETHEL_PORT` | No | `8080` | HTTP listen port |
| `AETHEL_DB_DSN` | No | — | Full DSN; overrides individual DB fields if set |

See `.env.example` for the complete reference with commentary.

</details>

<details>
<summary><strong>Blueprint files (IT-facing)</strong></summary>
<br/>

Only two YAML files require IT attention at deployment. Everything else is runtime-configurable through the admin panel.

| File | Purpose |
|---|---|
| `blueprints/server-database.yaml` | DB connection, pool sizing, schema aliases, auth cost params, server port, rate limit RPM |
| `blueprints/ui-theme.yaml` | Initial branding seed — primary color, neutral palette, font, wordmark, logo path. Loaded once at first boot; subsequent changes go through `/admin/branding`. |

The blueprint JSON Schema is at `blueprints/schemas/server-database.schema.json` — wire it to your editor for validation and autocomplete.

</details>

---

## Command Reference

<details>
<summary><strong>All make targets</strong></summary>
<br/>

**Development**
```bash
make dev           # Start PostgreSQL + backend + frontend (Docker Compose)
make dev-fe        # Nuxt dev server only — hot reload at :3000
make dev-be        # Go backend with Air live reload at :8080
make dev-db        # PostgreSQL container only
make dev-down      # Stop all containers (volumes preserved)
make dev-reset     # Full reset — wipe volumes, fresh database
```

**Testing**
```bash
make test          # All tests: frontend unit + backend unit + race detector
make test-fe       # Vitest unit + Nuxt component tests
make test-be       # go test ./... -race -short
make test-e2e      # Playwright E2E against running dev stack
```

**Database**
```bash
make migrate-up          # Apply all pending migrations
make migrate-down        # Roll back the last migration
make migrate-status      # Show applied / pending
make migrate-validate    # Render templates + SQL syntax check (no writes)
make db-shell            # Open psql into the dev database
make db-dump             # Dump to /tmp/aethel-dump-<timestamp>.sql
```

**Build**
```bash
make build          # Build frontend + backend binaries
make build-docker   # Build both Docker images locally
```

</details>

---

## Documentation

| Document | Description |
|---|---|
| [Architecture overview](./docs/architecture/) | Code architecture, server design, API routes, security model |
| [ER diagram](./docs/db-design.mmd) | Full database entity-relationship diagram (Mermaid) |
| [API routes](./docs/architecture/architecture-api-routes.md) | All 58 operationIds with request/response schemas |
| [Security architecture](./docs/architecture/architecture-security.md) | Auth model, middleware stack, threat model |
| [Go developer guide](./docs/guides/go-developer-guide.md) | Conventions, repo pattern, adding endpoints |
| [DevOps tooling](./docs/devops/devops-tooling.md) | Docker, Kubernetes, GitHub Actions CI/CD reference |

---

## License

Apache 2.0 — see [`LICENSE`](./LICENSE). Contributions welcome — see [`CONTRIBUTING.md`](./CONTRIBUTING.md).

<br/>

---

<div align="center">
  <sub><a href="./SECURITY.md">Security Policy</a> · <a href="./CONTRIBUTING.md">Contributing</a></sub>
</div>
