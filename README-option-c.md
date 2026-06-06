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

> [!IMPORTANT]
> **Designed for regulated environments.**
> - Tamper-evident audit log with SHA-256 hash chain verification — any altered record is identified by row ID and timestamp
> - Chain-of-custody tracking on every document transition, from intake to final delivery
> - RBAC with `sys_admin` role gating on all security-critical operations

---

## Who Uses Aethel

<table>
  <thead>
    <tr>
      <th>Role</th>
      <th>What They Do</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>ADMIN</strong></td>
      <td>Manages routing rules, reviews audit logs, configures branding and navigation, controls user access and escalation thresholds — all through the admin panel without touching config files</td>
    </tr>
    <tr>
      <td><strong>RECEPTION</strong></td>
      <td>Receives inbound correspondence, assigns dispatches to departments, tracks delivery status, and manages outbound document flow with a full timeline per item</td>
    </tr>
    <tr>
      <td><strong>USER</strong></td>
      <td>Views assigned documents, submits green notes (countersigned decisions), monitors workflow status, and submits outgoing document requests</td>
    </tr>
  </tbody>
</table>

Aethel Workspace replaces manual, paper-based correspondence with a structured digital system. Every inbound letter is tracked from receipt to delivery, annotated on a cryptographically-chained minute sheet, and permanently recorded in a tamper-evident audit ledger — all without touching a config file after first boot.

It is built for institutions where accountability and chain-of-custody are operational requirements, not features on a checklist: government offices, law firms, compliance-heavy enterprises, and any organization that signs and countersigns paper today.

> [!NOTE]
> **Aethel is self-hosted by design.** There is no cloud service, no SaaS tier, no vendor holding your documents. You deploy it to your own server. You own the database. You control who has access.

---

## Three Pillars

### DAK Diarization — Dispatch Tracking

A reception clerk receives an inbound letter from the Ministry of Finance — Aethel tracks it from intake to final delivery with a full audit trail.

Every dispatch is stamped with a tracking number and routed automatically by configurable rules (document type × sender × urgency level). Every routing decision — automatic or manual — is recorded on the document's timeline. An escalation worker monitors deadlines and fires notifications when items approach overdue thresholds.

**Dispatch lifecycle:**
```
PENDING_ASSIGNMENT → UNDER_REVIEW → IN_TRANSIT → DELIVERED
                                               ↘ ESCALATED (worker fires on deadline)
                                               ↘ REJECTED
```

---

### Green Noting Canvas — Minute Sheets

An approver appends a decision note — each note is cryptographically linked to the previous, creating an immutable chain of custody.

Every dispatch carries a minute sheet where staff append sequential notes. Each note is chained with a SHA-256 hash: `hash(content ‖ sequence ‖ author ‖ previous_hash)`. No note can be silently altered or deleted without breaking the chain. Approvals are countersigned with role-gated write access — attempted approvals by under-privileged users are rejected with `403` and logged to the audit ledger.

---

### RBAC Audit Ledger — Tamper Detection

A security officer verifies the audit log — a single API call returns whether the chain is intact or identifies which record was tampered with.

An append-only, monthly range-partitioned event log with a checksum chain spanning the entire ledger. The `GET /api/v1/audit-log/verify` endpoint re-computes the chain on demand and reports exactly which row was tampered with — down to the row ID and timestamp. Access requires `sys_admin` role.

<details>
<summary><strong>Tamper detection response format and event types</strong></summary>
<br/>

**Verification response:**
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

**Captured event types:** `DISPATCH_CREATED`, `DISPATCH_ASSIGNED`, `DISPATCH_DELIVERED`, `GREEN_NOTE_APPENDED`, `PERMISSION_DENIED`, `SECURITY_BREACH_ATTEMPT`, `UNAUTHORIZED_ACCESS_BYPASSED`, `RBAC_ELEVATION_ATTEMPT`.

**Schema choices:** `bigserial` PK (not UUID) for partition-efficient sequential access; `organization_id` stored as plain `uuid` without FK so audit records survive organization deletion; monthly `PARTITION BY RANGE(created_at)` so old partitions can be archived without touching the active partition.

</details>

---

> [!TIP]
> **Deployment footprint.** Aethel requires: 2 Docker containers (backend + frontend), 2 YAML config files to edit at deployment, 1 PostgreSQL database, 0 external services required at runtime. No message queue, no object store, no third-party auth provider. Everything runs on a single server.

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
go run ./cmd/aethel bootstrap-admin   # prints temporary password — change immediately via /admin/users
```

### What You'll See

After `make dev` succeeds and migrations are applied:

- **Login page** at `http://localhost:3000` — sign in with the `sys_admin` account created in step 4
- **Dashboard** showing your dispatch queue, urgency breakdown, and live notification feed via SSE
- **Admin panel** at `/admin/branding` where you can change the org name, colors, logo, and navigation structure — changes take effect on the next page load without a rebuild

| Service | URL |
|---|---|
| Frontend | http://localhost:3000 |
| Backend API | http://localhost:8080/api/v1 |
| Interactive API docs | http://localhost:8080/api/docs |

---

## Configuration for IT Administrators

Only two YAML files require attention at deployment. Everything else is managed at runtime through the admin panel.

| File | What to Edit |
|---|---|
| `blueprints/server-database.yaml` | Database connection details, connection pool size, rate limit RPM, JWT parameters, server port |
| `blueprints/ui-theme.yaml` | Initial branding seed — primary color, neutral palette, font, wordmark, logo path |

The JSON Schema at `blueprints/schemas/server-database.schema.json` enables editor validation and autocomplete when editing the database blueprint.

**Runtime admin panel** — after first boot, all further configuration changes happen through `/admin/*`:
- `/admin/branding` — live color picker, font selector, logo upload, preview panel
- `/admin/navigation` — nav tree editor (reorder, visibility, rename, add items)
- `/admin/routing-rules` — dispatch routing rule engine
- `/admin/users` — user management and role assignment
- `/admin/escalation` — deadline thresholds and notification targets
- `/admin/audit-log` — tamper-evident event log (`sys_admin` only)

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

---

## Security

Aethel takes security seriously. The full threat model and implemented controls are documented in [`docs/architecture/architecture-security.md`](./docs/architecture/architecture-security.md).

**Key controls at a glance:**

<details>
<summary><strong>Authentication and password security</strong></summary>
<br/>

Aethel uses **Argon2id** (RFC 9106 winner) for password hashing with configurable memory, iteration, and parallelism parameters. Defaults: 64 MiB memory · 3 iterations · 4 parallel threads. Passwords are never logged, never returned by the API. Access tokens are short-lived JWTs (HS256); refresh tokens are stored in httpOnly cookies with automatic rotation. Account lockout triggers after 5 failed attempts.

</details>

**Security surface:** Argon2id · JWT with short-lived access tokens · httpOnly refresh cookie with rotation · CSRF double-submit · account lockout · Content-Security-Policy · per-IP + per-user rate limiting · append-only audit ledger · `db-harden.sql` revokes default public schema permissions.

To report a vulnerability, see [`SECURITY.md`](./SECURITY.md). Please use GitHub's private vulnerability reporting — do not open a public issue for security bugs.

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

## Documentation

| Document | Description |
|---|---|
| [IT customization guide](./docs/guides/it-customization-guide.md) | Step-by-step deployment and first-boot configuration |
| [Architecture overview](./docs/architecture/) | Code architecture, server design, API routes, security model |
| [ER diagram](./docs/db-design.mmd) | Full database entity-relationship diagram (Mermaid) |
| [API routes](./docs/architecture/architecture-api-routes.md) | All 58 operationIds with request/response schemas |
| [Security architecture](./docs/architecture/architecture-security.md) | Auth model, middleware stack, threat model |
| [Go developer guide](./docs/guides/go-developer-guide.md) | Conventions, repo pattern, adding endpoints |
| [DevOps tooling](./docs/devops/devops-tooling.md) | Docker, Kubernetes, GitHub Actions CI/CD reference |
| [Interactive API docs](http://localhost:8080/api/docs) | Scalar UI — available when the backend is running |

---

## Contributing

See [`CONTRIBUTING.md`](./CONTRIBUTING.md) for the full guide. The short version: fork → branch from `dev` → `make test` passes → CI green → PR description links to an issue.

## License

Apache 2.0 — see [`LICENSE`](./LICENSE).

<br/>

---

<div align="center">
  <sub>Built with care · <a href="./SECURITY.md">Security Policy</a> · <a href="./CODE_OF_CONDUCT.md">Code of Conduct</a> · <a href="./CHANGELOG.md">Changelog</a> · <a href="./docs/guides/it-customization-guide.md">IT Customization Guide</a></sub>
</div>
