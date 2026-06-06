# Changelog

All notable changes to Aethel Workspace are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Aethel Workspace uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [1.0.0] — 2026-06-06

### Added

**Backend (Go)**
- CLI with `serve`, `migrate up/down/status/validate` subcommands via cobra
- 50+ REST endpoints across 9 route groups (auth, dispatch, workflow, governance, admin, config, notifications, search, health)
- 9-layer middleware stack: Recovery, MaxBodySize, RequestID, StructuredLogger, RateLimiter, CORS, Auth (JWT), CSRF, RBAC
- Structured zerolog request logging with `request_id`, `user_id`, `latency_ms`, `status_code` on every request
- Per-IP and per-user token bucket rate limiting, configurable via `server-database.yaml`

**Authentication & Security**
- JWT/HS256 access tokens with configurable TTL (default 30 min)
- httpOnly refresh cookie with rotation and configurable TTL (default 30 days)
- Argon2id password hashing with OWASP-compliant defaults; all cost params blueprint-configurable
- CSRF double-submit cookie protection on all mutating endpoints
- Account lockout after 5 failed login attempts (423 status, 15-minute window)
- Security headers: `X-Content-Type-Options`, `X-Frame-Options`, `Strict-Transport-Security`, `Content-Security-Policy`
- `db-harden.sql`: revokes default PostgreSQL public schema permissions
- `MaxBytesReader` 1 MiB body limit on all requests

**DAK Diarization Pillar**
- Dispatch CRUD: inbound and outbound tracking with priority levels (ROUTINE/PRIORITY/IMMEDIATE)
- Priority-ordered routing rule engine with document-type, sender-org, and urgency condition matching
- Unified `dispatch_events` timeline (routing decisions, handoffs, escalations, status changes)
- Escalation worker: background goroutine evaluates overdue dispatches, transitions to ESCALATED
- Unassigned inbox for manually routed dispatches

**Green Noting Canvas Pillar**
- Minute sheet auto-creation on dispatch intake
- Append-only green notes with SHA-256 cryptographic hash chain
- `SELECT FOR UPDATE` transaction lock prevents duplicate sequence numbers under concurrent appends
- Chain integrity validation on every append (detects tampered predecessor)
- Approval workflow with `MINUTE_SHEET_APPROVED` audit event

**Governance Pillar**
- Append-only audit ledger with monthly `PARTITION BY RANGE(created_at)`
- SHA-256 `previous_checksum` row chain for tamper detection
- `GET /api/v1/audit-ledger/verify` — full chain re-derivation scan (requires `sys_admin`)
- `migrate validate --check-syntax`: wraps each migration SQL in `BEGIN/EXPLAIN/ROLLBACK` to check syntax without execution

**Config API**
- `GET /api/v1/config` with 5-minute in-memory cache; SSR-embedded on first page load
- Runtime-editable: branding (colors, fonts, logo), navigation tree, feature flags, org profile
- `PATCH /api/v1/admin/config/*` invalidates cache on update

**Real-time Notifications**
- goroutine-safe `SSEBroker` with per-user subscriptions and 30-second keepalive
- `GET /api/v1/notifications/stream` — SSE stream (CSRF-exempt)
- Notification CRUD: list, mark-read, mark-all-read

**Frontend (Nuxt 4)**
- 19 pages across 3 roles: ADMIN, RECEPTION, USER
- Full i18n: English and Vietnamese (`@nuxtjs/i18n` v10, lazy-loaded locale files)
- Three-layer dynamic theming: CSS variables → runtime org config → component semantic tokens
- Admin panel: branding editor, navigation tree editor, routing rules, escalation rules, user management, audit log, reports
- Real-time notification drawer via SSE

**DevOps**
- 2-stage Docker images (distroless Go, multi-stage Nuxt)
- Docker Compose for local dev (`docker-compose.yml`) and production overrides (`docker-compose.prod.yml`)
- Kubernetes manifests: StatefulSet (PostgreSQL), Deployments with HPA, nginx Ingress
- GitHub Actions: `ci.yml` (test + lint, parallel), `cd.yml` (build + push GHCR → staging), `security.yml` (weekly Trivy + govulncheck + gosec)

**Observability**
- `/health` and `/ready` probes (excluded from request logs)
- zerolog structured logging wired through all service layers
- `BenchmarkRoutingRuleEngine`: 14.5 µs for 1000-rule worst-case scan, 0 allocations
- `BenchmarkHashChainVerify`: 48 µs to re-derive 100-note chain

**Documentation**
- OpenAPI 3.1 spec (58 operationIds), Scalar UI at `/api/docs`
- `docs/architecture/`: code, server, API routes, security architecture
- `docs/guides/`: IT customization guide, Go developer guide, blueprint conventions
- `docs/plans/`: agile implementation plan, migration strategy

### Blueprint schema versions
- `server-database.yaml`: v1.0
- `ui-theme.yaml`: v1.0
- `ui-layouts.yaml`: v1.0

---

[1.0.0]: https://github.com/hoangharry-tm/aethel-workspace/releases/tag/v1.0.0
