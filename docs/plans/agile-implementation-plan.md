# Agile Implementation Plan — Phase 2: Go Backend

**Audience:** Project leads, Go engineers
**Status:** Active

---

## Overview

This plan covers Phase 2 of Aethel Workspace: the implementation of the Go backend (`aethel-core/`). Phase 1 (Nuxt 4 UI/UX prototype, 17 pages, 20 user stories) is complete. The database design is also complete: 20 migration SQL files are written and validated.

Phase 2 is organized into 7 sprints of approximately 2 weeks each (14 calendar weeks total). Sprints are sequential; each sprint's deliverables are prerequisites for the next.

The **definition of done** for a sprint is: all listed deliverables are merged to the `main` branch, all tests pass in CI, and no item is left in a partially implemented state.

---

## Sprint 0 — Foundation (Weeks 1–2)

**Goal:** Establish the Go module, CLI skeleton, blueprint loading, database connectivity, and CI. Every subsequent sprint depends on this being solid.

### Deliverables

- `go.mod` initialized with module path `aethel-core`; direct dependencies pinned: `cobra`, `gopkg.in/yaml.v3`, `github.com/lib/pq`, `github.com/go-chi/chi/v5`, `github.com/rs/zerolog`, `golang.org/x/crypto` (Argon2id), `github.com/golang-jwt/jwt/v5`, `github.com/google/uuid`
- `cmd/aethel/main.go` with cobra root command and subcommands: `serve`, `migrate up`, `migrate down`, `migrate status`, `migrate validate`
- `internal/blueprint/loader.go` — `LoadDatabaseConfig`, `LoadQueriesConfig`; typed structs for both YAML files; validation that fails fast on missing required fields
- `internal/database/connect.go` — `Open(*blueprint.EnvironmentConfig) (*sql.DB, error)`; DSN from env var or individual fields; pool configuration applied; `db.Ping()` on connect
- `internal/database/blueprint_context.go` — `BlueprintContext` struct; `T()` and `E()` registered as `template.FuncMap`
- `internal/database/migrator.go` — `Migrator.Up`, `Migrator.Down`, `Migrator.Status`, `Migrator.Validate`; advisory lock; history table creation; SHA-256 checksum recorded
- Startup sequence in `cmd/aethel/main.go` following the documented order: blueprint → env select → DB open → migrations → query registry → HTTP server
- GitHub Actions CI: build + `go vet` + `go test ./...` on every push; fails if `go build` fails

### Definition of Done

- `go run ./cmd/aethel migrate up` applies all 20 migrations to a local PostgreSQL 16 database without error
- `go run ./cmd/aethel migrate status` shows all 20 versions as applied
- `go run ./cmd/aethel migrate validate` succeeds on a fresh checkout without a database connection
- `go run ./cmd/aethel serve` starts an HTTP server on `:8080` that responds to `GET /healthz` with `200 OK`
- CI is green

### Dependencies

- PostgreSQL 16 running locally or in CI (GitHub Actions `services: postgres`)
- Migration SQL files already written (complete as of Phase 1 database design)

---

## Sprint 1 — Core Domain and Auth (Weeks 3–4)

**Goal:** Implement domain types, repository interfaces, user authentication, session management, and RBAC middleware. No pillar-specific logic yet — just the auth and permission infrastructure that every subsequent endpoint requires.

### Deliverables

- Seed loader: on first boot, load `blueprints/ui-theme.yaml` → insert into `branding_configs`; load `blueprints/ui-layouts.yaml` nav_seed → insert into `system_settings` key `nav_config`. Seeding is idempotent — skip if the org already has a row.
- `internal/domain/` package: `Dispatch`, `DispatchEvent`, `MinuteSheet`, `GreenNote`, `AuditEntry`, `User`, `Session`, `Permission`, `RoutingRule`, `EscalationRule` types; `ErrNotFound`, `ErrForbidden`, `ErrConflict`, `ErrHashChainBroken` sentinel errors; repository interfaces for `User`, `Session`
- `internal/database/query_registry.go` — `BuildQueryRegistry`; prepares all named statements from `internal/database/queries/queries.yaml` at startup; panics on missing key (programmer error)
- `internal/service/auth_service.go` — `Register`, `Login` (Argon2id hash verify), `IssueAccessToken` (JWT), `IssueRefreshToken` (opaque random, stored in `user_sessions`), `RefreshSession`, `RevokeSession`, `RequestPasswordReset`, `ConfirmPasswordReset`
- `internal/rbac/middleware.go` — `Require(permission string)` middleware factory; reads role from context; checks against hardcoded role-permission table; writes `403` and logs `PERMISSION_DENIED` audit event on failure
- Auth handler `internal/api/handlers/auth.go` — POST `/auth/login`, POST `/auth/refresh`, POST `/auth/logout`, POST `/auth/password-reset/request`, POST `/auth/password-reset/confirm`
- Full middleware stack wired in `api/server.go`: Recovery → RequestID → StructuredLogger → RateLimiter → CORS → Auth → RBAC → Handler
- Health and readiness endpoints: `GET /healthz`, `GET /readyz`
- Unit tests for `auth_service.go`: register/login happy path, wrong password, expired token, revoked session

### Definition of Done

- `POST /api/v1/auth/login` with valid credentials returns a signed JWT access token and opaque refresh token
- `POST /api/v1/auth/refresh` with a valid refresh token returns a new access token
- A request to a protected endpoint with no token returns `401`; with a token for the wrong role returns `403`
- `PERMISSION_DENIED` events appear in `audit_ledger` when access is denied
- All unit tests pass

### Dependencies

- Sprint 0 complete (migrator, DB connect, blueprint loading)

---

## Sprint 2 — Dispatch Pillar (Weeks 5–6)

**Goal:** Implement the full DAK Diarization pillar: dispatch CRUD, routing rule engine, dispatch event log, intake API endpoints, queue/inbox endpoints, and the runtime config API.

### Deliverables

- Config API endpoints: `GET /api/v1/config`, `GET /api/v1/config/branding`, `GET /api/v1/config/nav`, `GET /api/v1/config/features`; `PATCH /api/v1/admin/config/branding`, `PATCH /api/v1/admin/config/nav`, `PATCH /api/v1/admin/config/features`, `PATCH /api/v1/admin/config/org`; single `ConfigCache` struct (5-min TTL) in `internal/config/`; cache invalidation on admin PATCH.
- `internal/domain/` — `DispatchRepository`, `RoutingRuleRepository`, `DispatchEventRepository` interfaces; `RoutingRule`, `RoutingRuleCondition`, `RoutingRuleDestination`, `DispatchEvent` domain types
- `internal/database/` — concrete implementations of `DispatchRepository`, `RoutingRuleRepository`, `DispatchEventRepository`; all queries scoped by `organization_id`
- `internal/service/dispatch_service.go` — `CreateDispatch` (runs routing rule engine, appends `ROUTING_APPLIED` event), `GetDispatchByID`, `ListInbox` (calls named query from registry), `ListOutbound`, `AssignDispatch` (manual routing, appends `MANUALLY_ASSIGNED` event), `AcknowledgeDelivery`, `UpdateStatus`, `EvaluateRoutingRules` (priority-ordered rule matching: document_type + sender_org + urgency conditions)
- `internal/api/handlers/dispatch.go` — all dispatch endpoints from the route table in `architecture-api-routes.md`
- Audit events written to `audit_ledger` on: `DISPATCH_CREATED`, `DISPATCH_ASSIGNED`, `DISPATCH_DELIVERED`
- Named queries added to `internal/database/queries/queries.yaml`: `dispatch.fetch_active_inbox`, `dispatch.fetch_outbound_queue`, `dispatch.search_by_tracking_number`, `dispatch.list_by_user`, `dispatch.timeline_events`
- Integration tests using a test database: create dispatch → routing rule matches → inbox query returns the dispatch → acknowledge delivery → status is DELIVERED

### Definition of Done

- `POST /api/v1/dispatches` creates a dispatch, evaluates routing rules, assigns to the matched department, and logs a `DISPATCH_CREATED` audit event
- `GET /api/v1/dispatches` returns the active inbox for the authenticated RECEPTION user's department
- `POST /api/v1/dispatches/{id}/acknowledge` transitions status to DELIVERED and logs `DISPATCH_DELIVERED`
- Integration tests pass against a real PostgreSQL 16 instance in CI

### Dependencies

- Sprint 1 complete (auth, RBAC middleware, domain types)

---

## Sprint 3 — Workflow Pillar (Weeks 7–8)

**Goal:** Implement the Green Noting Canvas pillar: minute sheet creation (triggered automatically by dispatch creation), green note appending with hash chain validation, and workflow approval.

### Deliverables

- `internal/domain/` — `MinuteSheetRepository`, `GreenNoteRepository` interfaces; `MinuteSheet`, `GreenNote` domain types (immutable after insert)
- `internal/database/` — concrete implementations; `GreenNote` insert includes hash computation and chain validation in the same transaction
- `internal/service/workflow_service.go` — `GetMinuteSheet`, `AppendGreenNote` (validates chain before insert using `pgcrypto.digest`), `ListGreenNotes`, `ApproveMinuteSheet`
- Hash chain validation logic: fetch the last note in the minute sheet; compute expected hash as SHA-256 of `(new_content || new_sequence || author_id || previous_hash)`; compare against the stored `previous_hash` in the new note; reject with `ErrHashChainBroken` if mismatch
- `internal/api/handlers/workflow.go` — all workflow endpoints
- Named queries added to `internal/database/queries/queries.yaml`: `workflow.fetch_minute_sheet_with_notes`, `workflow.fetch_last_green_note`, `workflow.verify_hash_chain`
- Audit event written to `audit_ledger` on: `GREEN_NOTE_APPENDED`
- Unit tests for hash chain: append 5 notes, verify chain is intact; tamper with note 3, verify chain breaks at note 4

### Definition of Done

- `POST /api/v1/dispatches/{id}/green-notes` appends a note, computes the hash chain, and rejects out-of-order inserts
- `GET /api/v1/dispatches/{id}/minute-sheet` returns the full minute sheet with all notes in sequence order
- Attempting to insert a note with an incorrect `previous_hash` returns `409 Conflict`
- Unit tests for hash chain validation pass

### Dependencies

- Sprint 2 complete (dispatch creation triggers minute sheet creation via `CreateDispatch`)

---

## Sprint 4 — Governance Pillar (Weeks 9–10)

**Goal:** Implement the RBAC Audit Ledger pillar: centralize the audit writer used in all previous sprints, add the tamper detection verification endpoint, implement the escalation rule evaluator worker.

### Deliverables

- `internal/domain/` — `AuditRepository` interface; `AuditEntry` domain type; `EscalationRuleRepository` interface; `EscalationRule` domain type
- `internal/database/` — `AuditRepository` implementation (insert only, `organization_id` is plain UUID without FK per schema design); `EscalationRuleRepository` implementation
- `internal/service/auth_service.go` and `dispatch_service.go` updated to call a shared `audit.Writer` interface instead of writing directly — all audit writes go through one path
- Tamper detection: `governance.VerifyChain(ctx, orgID, from, to)` fetches audit rows in the date range and re-computes the `previous_checksum` chain; returns a `VerificationResult` with any broken links
- `internal/api/handlers/governance.go` — `GET /api/v1/audit-log`, `GET /api/v1/audit-log/verify`
- `internal/service/escalation_service.go` — `EvaluateEscalationRules(ctx)`: queries dispatches that have exceeded their escalation threshold, applies the configured action (update status to ESCALATED, send notification, write audit event)
- `internal/worker/escalation_worker.go` — goroutine launched in `main.go`; ticks on blueprint interval; calls `escalation_service.EvaluateEscalationRules`; logs results; respects context cancellation
- Named queries added to `internal/database/queries/queries.yaml`: `governance.query_audit_ledger_paged`, `governance.fetch_overdue_dispatches_for_escalation`
- Integration test: insert 10 audit rows, verify chain passes; corrupt one row in the database directly, verify the verification endpoint reports the break at the correct row

### Definition of Done

- `GET /api/v1/audit-log` returns paginated audit events filtered by date range (requires `SYS_ADMIN` role)
- `GET /api/v1/audit-log/verify` returns a pass result for an unmodified chain and a specific broken-link report for a tampered chain
- The escalation worker fires, finds an overdue dispatch, sets its status to ESCALATED, and writes an audit event
- Integration tests pass

### Dependencies

- Sprint 3 complete (audit events from dispatches and green notes are already being written)

---

## Sprint 5 — API Completeness and Integration (Weeks 11–12)

**Goal:** Complete the remaining API surface: admin endpoints, notifications, SSE stream, route blueprint registry, and outbound dispatch support.

### Deliverables

- `internal/api/handlers/admin.go` — all admin endpoints: users CRUD, document types CRUD, routing rules CRUD, escalation rules CRUD, reports, settings, branding
- `internal/transport/sse.go` — `SSEBroker` with `Publish(userID uuid.UUID, event Event)` and `ServeHTTP(w, r)`; goroutine-safe; cleans up closed connections
- Notification endpoints: `GET /api/v1/notifications`, `PATCH /api/v1/notifications/{id}/read`, `GET /api/v1/notifications/stream` (SSE)
- Route registry finalized in `api/server.go`: all routes from the route table in `docs/architecture/architecture-api-routes.md` are registered; `permission` field validation enforced at startup; no blueprint overrides (routes are convention-driven)
- Outbound dispatch endpoints wired (handler already exists from Sprint 2, this sprint adds the missing admin-only attachment deletion and report aggregation queries)
- Named queries added to `internal/database/queries/queries.yaml`: `admin.dispatch_volume_by_period`, `admin.user_activity_summary`, `search.fulltext_dispatches`
- CSRF middleware added to the middleware stack for browser clients
- **i18n**: install `@nuxtjs/i18n`; create `locales/en.json` and `locales/vi.json`; replace all hardcoded UI strings with `$t('key')` calls; wire active locale to `config.value.org.locale` from `useAppRuntimeConfig` so the admin's chosen locale (set via `PATCH /api/v1/admin/config/org`) takes effect immediately without a page reload; add locale key naming conventions to `aethel-view/CLAUDE.md`
- End-to-end test with the Nuxt frontend: login → view inbox → create dispatch → view document detail → append green note — all using the real backend

### Definition of Done

- All 50+ API endpoints in the route table return non-500 responses for valid authenticated requests
- The SSE stream delivers a notification event to a connected browser client within 2 seconds of the triggering action
- The Nuxt frontend can be pointed at the running backend and complete the full reception workflow (US-01 through US-06)

### Dependencies

- Sprints 1–4 complete

---

## Sprint 6 — Hardening (Weeks 13–14)

**Goal:** Production readiness: rate limiting, structured logging, health probes, performance benchmarks, and blueprint validation completeness.

### Deliverables

- Per-IP and per-user token bucket rate limiter implemented in `api/server.go`; `rate_limit_rpm` from route blueprint applied per group and per route
- Request ID propagated through all log lines and error responses; `zerolog` structured logger with `request_id`, `org_id`, `user_id`, `latency_ms` fields
- `GET /healthz` (liveness) and `GET /readyz` (readiness with `db.PingContext`) endpoints finalized and documented
- Argon2id cost parameters read from blueprint (Sprint 1 used hardcoded values; this sprint makes them configurable)
- `migrate validate` performs SQL syntax checking (render templates, send to PostgreSQL with `EXPLAIN` but no `EXECUTE`) in addition to template rendering
- Benchmark tests (`go test -bench`) for: routing rule engine (1000 rules), hash chain validation (100 notes), audit ledger insert throughput (target: 1000 inserts/sec on local PostgreSQL)
- Load test with `k6` or `vegeta`: 100 concurrent users, reception workflow, p99 latency < 200 ms against a local PostgreSQL 16 instance
- `CHANGELOG.md` entry for v1.0.0 with the final blueprint schema versions, following the [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format (`## [1.0.0] — YYYY-MM-DD` header, sections: Added / Changed / Fixed / Security)
- **`SECURITY.md`** — security vulnerability disclosure policy placed at the repository root. Must include: (1) **Supported versions table** listing which versions receive security patches; (2) **Private reporting instructions** — direct reporters to GitHub's private vulnerability reporting (`Security` tab → `Report a vulnerability`) or a dedicated email, explicitly stating NOT to open a public GitHub issue for security bugs; (3) **Response SLA** — acknowledge within 48 hours, triage within 7 days, patch release within 30 days for critical CVEs; (4) **Scope** — clarify what is in scope (auth bypass, SQL injection, privilege escalation, token leakage) vs. out of scope (rate limiting edge cases, UI cosmetics); (5) **Credit** — confirm reporters will be credited in `CHANGELOG.md` and the GitHub Security Advisory unless they request anonymity
- **`CONTRIBUTING.md`** — contributor guide placed at the repository root. Must include: (1) **Development setup** — prerequisites (Go 1.26+, Node 22+, pnpm, PostgreSQL 16, Docker), step-by-step `make dev` quickstart referencing `aethel-scripts/setup-dev.sh`; (2) **Branch naming convention** — `feat/`, `fix/`, `chore/`, `docs/` prefixes; (3) **Commit message format** — Conventional Commits (`feat(scope): message`, `fix(scope): message`); (4) **PR requirements** — all tests pass (`go test ./...` + `pnpm test`), no TypeScript errors, CI green, description references a GitHub issue; (5) **Code style** — Go: `gofmt` + `golangci-lint`; Vue/TypeScript: ESLint via `eslint.config.mjs`; (6) **Testing expectations** — new endpoints require at least one integration test against a real PostgreSQL instance (no mocked DB per project convention); (7) **License agreement** — by submitting a PR, contributors agree their code is licensed under Apache 2.0
- **`CODE_OF_CONDUCT.md`** — placed at the repository root. Use the **Contributor Covenant v2.1** verbatim (https://www.contributor-covenant.org/version/2/1/code_of_conduct/) with the following fields filled in: contact email for enforcement reports (use the author's professional email or a dedicated conduct@aethel.org address); project maintainer name (Minh Hoang Ton). The Contributor Covenant v2.1 is the industry standard adopted by Linux Foundation, GitHub, and thousands of open-source projects — do not write a custom conduct policy
- Docker multi-stage build: `Dockerfile` producing a minimal image; `docker compose` for local development with PostgreSQL 16

### Definition of Done

- `go test ./... -race` passes (no data races detected)
- Benchmark results are recorded in CI output and compared against the target thresholds
- Rate limiting rejects a burst of 700 RPM with a mix of `200` and `429` responses when the limit is set to 600 RPM
- The Docker image builds cleanly and `docker compose up` brings a working development environment

### Dependencies

- Sprint 5 complete (all endpoints implemented)
