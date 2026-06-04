# Task 15 — Full-Project Audit & Progress Report

**Purpose:** Scan every file in the repository, check against defined criteria, and produce a structured statistics report. No code is written or modified in this task — read-only audit only.
**Output:** A single consolidated markdown report written to `docs/reports/audit-$(date +%Y-%m-%d).md`

---

## How This Task Runs

Spawn **5 parallel subagents** immediately — one per domain. Each agent scans its scope, fills its section of a shared report template, and writes its section to a temp file. A final **synthesis agent** (Agent 6) waits for all 5 to complete, merges the sections, computes overall statistics, and writes the final report.

```
Agent 1 — NuxtJS Auditor       → aethel-view/
Agent 2 — Go Backend Auditor   → aethel-core/
Agent 3 — Database Auditor     → migrations + schema + queries
Agent 4 — Security Auditor     → auth, middleware, secrets, headers
Agent 5 — DevOps Auditor       → Docker, K8s, CI/CD, scripts
                    ↓ all complete
Agent 6 — Synthesis Agent      → merge + overall score → final report
```

Each agent writes its findings to:
```
/tmp/audit-agent-{1..5}.md
```

Agent 6 reads all five files and writes:
```
docs/reports/audit-YYYY-MM-DD.md
```

---

## Audit Criteria Reference (All Agents Must Apply)

### Completion Scoring

Rate each item on this scale — use it consistently across all agents:

| Score | Meaning |
|-------|---------|
| ✅ Done | Fully implemented, tested, no known gaps |
| 🔄 Partial | Core logic exists but missing tests, edge cases, or wiring |
| ⚠️ Stub | File exists with placeholder / "under construction" content only |
| ❌ Missing | File or feature not present at all |
| 🚫 Blocked | Cannot complete without a prerequisite that is not done |

### Percentage formula

```
completion % = (Done×1.0 + Partial×0.5 + Stub×0.1) / total_items × 100
```

Apply this formula per section and for the overall score.

---

## Agent 1 — NuxtJS / Frontend Auditor

**Scope:** `aethel-view/` — all pages, components, composables, plugins, middleware, assets

**You are a senior NuxtJS 4 + Vue 3 + TypeScript engineer.** Read every `.vue`, `.ts`, and `.css` file in scope. Do not read `aethel-core/`.

### Audit checklist

#### Pages (`app/pages/`)
For each page file, report:
- File path + line count
- Completion score (Done / Partial / Stub / Missing)
- Whether `definePageMeta` is set correctly
- Whether it uses real API data or mock data (`useMockData()`)
- Whether it uses semantic CSS tokens (zero palette classes: `text-slate-*`, `bg-white`, etc.)
- Any TypeScript errors visible in the template (look for obvious `any` casts or missing type imports)

Expected pages (check each exists and is not a stub):
```
pages/index.vue
pages/auth/login.vue
pages/dashboard.vue
pages/dispatch/inbound/index.vue
pages/dispatch/inbound/new.vue
pages/dispatch/outbound/index.vue
pages/documents/[id].vue
pages/my-documents.vue
pages/outgoing/new.vue
pages/search.vue
pages/admin/users.vue
pages/admin/routing-rules.vue
pages/admin/document-types.vue
pages/admin/escalation.vue
pages/admin/audit-log.vue
pages/admin/reports.vue
pages/admin/settings.vue
pages/admin/branding.vue
pages/admin/navigation.vue
```

#### Components (`app/components/`)
- List all components found
- Flag any component that hardcodes palette classes (grep for `text-slate`, `text-indigo`, `bg-white`, `bg-slate`, `text-gray`)
- Flag any component not using Nuxt UI (`UButton`, `UTable`, etc.) but using raw HTML buttons or tables

#### Composables (`app/composables/`)
- Does `useAuth.ts` exist? Does it store access token in `useState` (not `localStorage`)?
- Does `useRuntimeConfig.ts` exist and call real `$fetch('/api/v1/config')`?
- Does `useMockData.ts` still exist (expected — it's the prototype data layer)?

#### Plugins (`app/plugins/`)
- Does `auth.client.ts` exist? Does it perform silent token recovery on page load?

#### Middleware (`app/middleware/`)
- Does `auth.ts` exist? Does it redirect unauthenticated users?
- Does `role.ts` exist? Does it gate pages by requiredRole?

#### Design system compliance
Run these grep checks and report exact match counts:
```bash
# Should be 0 — palette violations
grep -rn "text-slate\|text-indigo\|bg-white\|bg-slate\|text-gray\|text-zinc" app/pages/ app/components/
# Should be > 0 — semantic tokens in use
grep -rn "text-body\|text-muted\|text-accent\|bg-surface\|bg-subtle" app/pages/ app/components/
# Should be 0 — localStorage forbidden for tokens
grep -rn "localStorage" app/
```

#### Build check
```bash
cd aethel-view
pnpm build 2>&1 | tail -20
```
Report: success or first error message.

### Output format for Agent 1 section

```markdown
## Frontend Audit (aethel-view/)

### Pages Summary
| Page | Lines | Status | Mock/Real | Semantic CSS | Notes |
|------|-------|--------|-----------|--------------|-------|
| ... | ... | ✅/🔄/⚠️/❌ | Mock/Real/Both | ✅/❌ | ... |

**Pages completion: X/19 done (X%), Y partial, Z stubs**

### Component Violations
- Palette class violations: N files (list them)
- Non-Nuxt-UI interactive elements: N files

### Composables & Plugins
| File | Exists | Correct impl | Notes |
|------|--------|-------------|-------|

### Design System Grep Results
- Palette violations: N matches across N files
- Semantic token usage: N matches (healthy)
- localStorage usage: N matches (must be 0)

### Build Status
- pnpm build: ✅ Success / ❌ Failed (error: ...)

### Frontend Completion Score: XX%
```

---

## Agent 2 — Go Backend Auditor

**Scope:** `aethel-core/` — all `.go` files, `go.mod`, `go.sum`

**You are a senior Go engineer familiar with chi, zerolog, Argon2id, JWT, and PostgreSQL.** Read every `.go` file. Do not read `aethel-view/`.

### Audit checklist

#### Build & vet
```bash
cd aethel-core
go build ./... 2>&1
go vet ./... 2>&1
```
Report: pass or first error.

#### Package inventory
For each package, report: exists ✅ / missing ❌ / stub (no real logic) ⚠️

Expected packages:
```
cmd/aethel/               # main.go — startup sequence
internal/app/             # org.go — LoadOrgID(), OrgID var
internal/audit/           # writer.go, db_writer.go — centralized audit interface
internal/blueprint/       # loader.go
internal/config/          # cache.go, loader.go, handler.go
internal/database/        # connect.go, migrator.go, query_registry.go
internal/database/repos/  # all repo implementations
internal/database/queries/ # queries.yaml
internal/domain/          # all domain types + interfaces
internal/api/             # server.go
internal/api/handlers/    # auth.go, dispatch.go, workflow.go, governance.go, admin.go
internal/api/docs/        # handler.go, openapi.yaml, scalar.html
internal/rbac/            # middleware.go
internal/service/         # auth_service.go, dispatch_service.go, workflow_service.go, governance_service.go, escalation_service.go
internal/worker/          # escalation_worker.go
internal/transport/       # sse.go (Sprint 5 — may be missing)
```

#### Repository implementations
For each repo file, check:
- Is it a real implementation or a noop/stub?
- Does it use `qr.Get("group.name")` — zero inline SQL strings?
- Does it reference `app.OrgID` instead of taking orgID as a parameter?

Expected repos:
```
repos/user_repo.go
repos/session_repo.go
repos/password_reset_repo.go
repos/audit_repo.go
repos/dispatch_repo.go
repos/dispatch_event_repo.go
repos/routing_rule_repo.go
repos/minute_sheet_repo.go
repos/green_note_repo.go
repos/escalation_rule_repo.go
```

Grep check:
```bash
# Should be 0 — no inline SQL
grep -rn "db\.Query\|db\.Exec\|db\.QueryRow" internal/database/repos/
# Should be 0 — no orgID method params
grep -rn "orgID\b" internal/service/ internal/database/repos/
# Should be 0 — no org claim in JWT
grep -rn "org.*claim\|claims\[.org.\]" internal/
```

#### Service layer
For each service, check:
- Does it inject `audit.Writer` (not `domain.AuditRepository` directly)?
- Are all business logic methods implemented (not empty or TODO)?

#### Routes registered in server.go
Count the total registered routes and compare against the expected 50+ from `docs/architecture/architecture-api-routes.md`. List any routes in the architecture doc that are NOT registered.

#### Test coverage
```bash
go test ./... 2>&1 | grep -E "^ok|FAIL|---"
go test ./internal/service/... -v 2>&1 | grep -E "^=== RUN|--- PASS|--- FAIL"
```
Report: X/Y tests passing, list failures.

#### Inline SQL check
```bash
grep -rn "\"SELECT\|\"INSERT\|\"UPDATE\|\"DELETE" internal/database/repos/ internal/service/
```
Report: exact count — must be 0.

### Output format for Agent 2 section

```markdown
## Backend Audit (aethel-core/)

### Build Status
- go build ./...: ✅ / ❌ (error)
- go vet ./...: ✅ / ❌

### Package Inventory
| Package | Status | Notes |
|---------|--------|-------|

### Repository Layer
| Repo File | Exists | Real impl | Zero inline SQL | Uses app.OrgID |
|-----------|--------|-----------|-----------------|----------------|

### Service Layer
| Service | Exists | audit.Writer injected | All methods implemented |
|---------|--------|----------------------|-------------------------|

### Route Coverage
- Routes registered: N
- Routes in architecture doc: 50+
- Missing routes: [list]

### Test Results
- Passing: N
- Failing: N (list)

### Grep Violations
- Inline SQL strings: N (must be 0)
- orgID method params: N (must be 0)

### Backend Completion Score: XX%
```

---

## Agent 3 — Database Auditor

**Scope:** `aethel-core/internal/database/migrations/`, `aethel-core/internal/database/queries/queries.yaml`, `docs/db-design.mmd`

**You are a senior PostgreSQL engineer.** Read every migration SQL file and the queries YAML. Do not read application Go code.

### Audit checklist

#### Migration files
- Count total `.up.sql` and `.down.sql` files — should be equal
- List each migration number and table/object created
- Check that migration 21 exists (ALTER branding_configs — neutral_palette, font_family, wordmark)
- Flag any migration that references a column not in `docs/db-design.mmd`
- Flag any migration with `DROP TABLE` or destructive operations not in a `.down.sql` file

#### Schema consistency vs ER diagram
Read `docs/db-design.mmd`. For each table defined in the diagram, verify a migration creates it. Report discrepancies.

Expected tables (20 + migration 21 objects):
```
organizations, departments, users, user_sessions, password_reset_tokens,
notification_preferences, document_types, dispatches, dispatch_attachments,
dispatch_events, routing_rules, routing_rule_conditions, routing_rule_destinations,
minute_sheets, green_notes, notifications, escalation_rules, system_settings,
branding_configs, audit_ledger (partitioned)
```

#### queries.yaml completeness
Read `internal/database/queries/queries.yaml`. List every query group and the named queries within each. Check for:
- Groups expected: `auth`, `users`, `sessions`, `password_reset`, `dispatch`, `dispatch_events`, `routing_rules`, `minute_sheets`, `green_notes`, `escalation_rules`, `governance`, `config`, `notifications`
- Missing groups
- Queries referenced in Go service code (`qr.Get("X.Y")`) but not present in the YAML — grep for `qr.Get(` in all `.go` files, extract the key names, verify each exists in YAML

#### Template variable usage
Check migration files for Go template syntax (`{{ .Schema }}`, `{{ T "..." }}`, `{{ E "..." }}`). List which migrations use templates and which use hardcoded names.

### Output format for Agent 3 section

```markdown
## Database Audit

### Migration Files
- Total up migrations: N
- Total down migrations: N
- Paired correctly: ✅ / ❌ (list unpaired)
- Migration 21 (branding_configs ALTER): ✅ / ❌

### Schema vs ER Diagram
| Table | In Diagram | Migration # | Discrepancies |
|-------|-----------|-------------|---------------|

### queries.yaml Coverage
| Group | Exists | Query Count | Missing queries |
|-------|--------|-------------|-----------------|

### Dangling qr.Get() calls (keys in Go but not in YAML)
- N dangling keys (list them)

### Database Completion Score: XX%
```

---

## Agent 4 — Security Auditor

**Scope:** entire repository — focus on `aethel-core/internal/`, `aethel-view/app/`, `.env.example`, `aethel-scripts/`

**You are a senior application security engineer.** Your job is to verify that all security controls documented in `docs/architecture/architecture-security.md` are actually implemented in code.

### Audit checklist

#### Authentication
- [ ] JWT algorithm is RS256 or HS256 (read `internal/service/auth_service.go`) — no `alg: none`
- [ ] JWT claims: `sub`, `role`, `iat`, `exp`, `jti` present — no `org` claim (single-tenant)
- [ ] Argon2id parameters: memory ≥ 64MiB, iterations ≥ 3, threads ≥ 4 (grep for `argon2.IDKey`)
- [ ] Access token NOT stored in `localStorage` on the frontend (grep `aethel-view/`)
- [ ] Refresh token set as `httpOnly` cookie (grep `SetCookie` + `HttpOnly`)
- [ ] Refresh token rotation is atomic (single DB transaction deletes old + inserts new)

#### Middleware stack (read `internal/api/server.go`)
Verify these middleware are applied in this exact order:
1. Recovery
2. RequestID
3. StructuredLogger
4. RateLimiter
5. CORS
6. Auth
7. RBAC
8. Handler

Report: exact order found vs expected.

#### CSRF protection
- [ ] `CSRFProtect` middleware exists in `internal/api/` or `internal/middleware/`
- [ ] Uses `crypto/subtle.ConstantTimeCompare` (not `==` for token comparison)
- [ ] CSRF token set as a readable cookie (not httpOnly) so the JS frontend can read it
- [ ] CSRF token validated on all state-changing requests (POST, PATCH, DELETE)

#### Security headers
Read `internal/api/` for a `SecurityHeaders` middleware. Verify these headers are set:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Strict-Transport-Security` (HSTS — only over HTTPS)
- `Content-Security-Policy`
- `Referrer-Policy`

#### Secrets audit
```bash
# Must return 0 — no secrets hardcoded
grep -rn "AETHEL_JWT_SECRET\|-----BEGIN RSA\|password.*=.*['\"]" aethel-core/ aethel-view/ --include="*.go" --include="*.ts" --include="*.vue"
# .env.example should exist and contain no real values
cat .env.example | grep -v "^#" | grep "="
```

#### Account lockout
- [ ] Failed login attempts tracked (grep `failed_login_attempts` or equivalent)
- [ ] Locked account returns HTTP 423 (not 403 or 401) — grep for `423` in handlers

#### db-harden.sql
- [ ] `aethel-core/scripts/db-harden.sql` exists
- [ ] Contains REVOKE statements on `audit_ledger` for non-superuser roles

### Output format for Agent 4 section

```markdown
## Security Audit

### Authentication Controls
| Control | Status | Evidence (file:line) |
|---------|--------|----------------------|

### Middleware Stack Order
Expected: Recovery → RequestID → StructuredLogger → RateLimiter → CORS → Auth → RBAC
Actual:   [list what's found in server.go]
Match: ✅ / ❌

### CSRF Protection
| Check | Status | Notes |
|-------|--------|-------|

### Security Headers
| Header | Present | Value |
|--------|---------|-------|

### Secrets Audit
- Hardcoded secrets found: N (must be 0)
- .env.example real values: N (must be 0)

### Other Controls
| Control | Status | Notes |
|---------|--------|-------|

### Security Score: XX% (N/M controls passing)
### Critical findings: [list any ❌ items]
```

---

## Agent 5 — DevOps Auditor

**Scope:** `Makefile`, `docker-compose.yml`, `docker-compose.prod.yml`, `aethel-core/Dockerfile`, `aethel-view/Dockerfile`, `k8s/`, `.github/workflows/`, `aethel-scripts/`

**You are a senior DevOps / platform engineer.** Read every infrastructure file. Do not read application source code.

### Audit checklist

#### Docker
- [ ] `aethel-core/Dockerfile` exists — is it a multi-stage build? Does stage 2 use a distroless or minimal base?
- [ ] `aethel-view/Dockerfile` exists — is it a multi-stage build?
- [ ] `docker-compose.yml` defines: `postgres`, `backend`, `frontend` services
- [ ] PostgreSQL port mapping: host `5433` → container `5432` (non-default to avoid collision)
- [ ] `docker-compose.prod.yml` exists with production overrides
- [ ] Health checks defined on backend + postgres services
- [ ] No secrets hardcoded in compose files — all from environment variables

#### Kubernetes
- [ ] `k8s/` directory exists
- Expected manifests: `postgres/StatefulSet`, `postgres/PVC`, `backend/Deployment`, `backend/HPA`, `backend/ConfigMap`, `frontend/Deployment`, `frontend/Service`, `ingress.yaml`
- List which manifests exist and which are missing
- [ ] Namespace set to `aethel-workspace` in all manifests

#### GitHub Actions
- [ ] `ci.yml` exists — does it run `go test ./...` + `pnpm test` + lint in parallel?
- [ ] `cd.yml` exists — does it build + push to GHCR on merge to main?
- [ ] `security.yml` exists — does it run Trivy + govulncheck + gosec?
- [ ] PostgreSQL service defined in CI for integration tests

#### Scripts (`aethel-scripts/`)
Expected scripts:
```
setup-dev.sh
health-check.sh
rotate-jwt-secret.sh
db-backup.sh
k8s-rollout.sh
```
For each: exists ✅ / missing ❌ / exists but empty ⚠️

#### Makefile
- [ ] `make help` target exists
- [ ] Common targets present: `dev`, `build`, `test`, `migrate`, `deploy`

### Output format for Agent 5 section

```markdown
## DevOps Audit

### Docker
| Artifact | Exists | Multi-stage | Notes |
|----------|--------|-------------|-------|

### Kubernetes Manifests
| Manifest | Exists | Correct namespace | Notes |
|----------|--------|------------------|-------|

### GitHub Actions Workflows
| Workflow | Exists | Key steps verified | Notes |
|----------|--------|-------------------|-------|

### Scripts
| Script | Exists | Non-empty | Notes |
|--------|--------|-----------|-------|

### Makefile
- make help: ✅ / ❌
- Key targets: [list found]

### DevOps Completion Score: XX%
```

---

## Agent 6 — Synthesis Agent (runs after all 5 complete)

**Wait for all 5 temp files to exist:**
```bash
ls /tmp/audit-agent-{1,2,3,4,5}.md
```

**Read all five files.** Merge into a single report. Compute the overall score using the formula:

```
overall = (frontend_pct × 0.25) + (backend_pct × 0.30) + (database_pct × 0.15) + (security_pct × 0.20) + (devops_pct × 0.10)
```

Weights rationale: backend is the most work-in-progress (0.30); security is critical (0.20); frontend prototype is mostly done (0.25); database is stable (0.15); DevOps is scaffolded (0.10).

### Final report structure

Write to `docs/reports/audit-YYYY-MM-DD.md` (use the actual date):

```markdown
# Aethel Workspace — Project Audit Report
**Date:** YYYY-MM-DD
**Audited by:** 5-agent parallel audit team (Claude Code)

---

## Executive Summary

| Domain | Weight | Score | Weighted |
|--------|--------|-------|---------|
| Frontend (NuxtJS) | 25% | XX% | XX% |
| Backend (Go) | 30% | XX% | XX% |
| Database | 15% | XX% | XX% |
| Security | 20% | XX% | XX% |
| DevOps | 10% | XX% | XX% |
| **OVERALL** | **100%** | | **XX%** |

---

## Critical Findings (must fix before Sprint 5)

List any ❌ items across all domains that block forward progress.

---

## Sprint Progress vs Agile Plan

| Sprint | Plan Status | Actual Status | Delta |
|--------|-------------|---------------|-------|
| Sprint 0 | Complete | [from audit] | |
| Sprint 1 | Complete | [from audit] | |
| Sprint 1.5 | Complete | [from audit] | |
| Sprint 2 | Running | [from audit] | |
| Sprint 3 | Queued | [from audit] | |
| Sprint 4 | Not started | [from audit] | |
| Sprint 5 | Not started | [from audit] | |
| Sprint 6 | Not started | [from audit] | |

---

## Detailed Findings

[Paste each agent's full section here verbatim]

---

## Recommendations for Next Session

Top 5 highest-impact items to address, ordered by priority.
```

After writing the report, output the Executive Summary table to the console so the user sees the results immediately without opening the file.

---

## Execution Instructions

```
1. Spawn Agents 1–5 in parallel (single message, 5 Agent tool calls)
2. Each agent writes its section to /tmp/audit-agent-N.md
3. Wait for all 5 completion notifications
4. Spawn Agent 6 to synthesize
5. Agent 6 writes the final report to docs/reports/audit-YYYY-MM-DD.md
6. Agent 6 prints the Executive Summary table to console
7. Commit the report: git add docs/reports/ && git commit -m "chore(audit): project progress report YYYY-MM-DD"
```

## Definition of Done

- [ ] All 5 audit temp files written to `/tmp/audit-agent-{1..5}.md`
- [ ] Final report written to `docs/reports/audit-YYYY-MM-DD.md`
- [ ] Overall completion percentage computed and displayed
- [ ] Critical findings section lists all ❌ items
- [ ] Report committed to `dev` branch
- [ ] No code was modified — this task is read-only
