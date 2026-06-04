# Task 15 — Full-Project Audit & Progress Report

**Purpose:** Scan every file in the repository, check against defined criteria, and produce a structured statistics report. No code is written or modified in this task — read-only audit only.
**Output:** A single consolidated markdown report written to `docs/reports/audit-$(date +%Y-%m-%d).md`

---

## ⚠️ Important: This Task Is State-Aware

The file lists, expected package names, and route counts written below were accurate at the time this task was authored. By the time you run this task, **Tasks 11, 12, 13, and 14 will have been executed**, adding new files and modifying existing ones. Every agent **must discover the actual current state** from the filesystem and git history — never trust the hardcoded lists below as exhaustive. Use them only as a minimum baseline; any files found beyond the list must also be audited.

---

## Step 0 — Pre-flight Sync (Run This First, Before Spawning Any Agent)

Before spawning any parallel agent, the orchestrating session must run the following and write the output to `/tmp/audit-preflight.md`. Every agent will read this file at the start of their work.

```bash
REPO=/Users/hoangharry/mh_code/internships/Bravo/aethel-workspace
cd "$REPO"

# 1. Pull the latest state from the remote
git pull origin dev

# 2. Capture recent commit history (last 30 commits)
echo "## Recent Commits" > /tmp/audit-preflight.md
git log --oneline -30 >> /tmp/audit-preflight.md

# 3. Capture all files changed in the last 30 commits
echo -e "\n## Files Changed in Last 30 Commits" >> /tmp/audit-preflight.md
git diff HEAD~30..HEAD --stat 2>/dev/null >> /tmp/audit-preflight.md || git diff $(git rev-list --max-parents=0 HEAD)..HEAD --stat >> /tmp/audit-preflight.md

# 4. Capture all new files added (untracked + recently committed)
echo -e "\n## All Go Files in aethel-core (current)" >> /tmp/audit-preflight.md
find aethel-core -name "*.go" | sort >> /tmp/audit-preflight.md

echo -e "\n## All Vue/TS Files in aethel-view (current)" >> /tmp/audit-preflight.md
find aethel-view/app -name "*.vue" -o -name "*.ts" | sort >> /tmp/audit-preflight.md

echo -e "\n## All Task Files" >> /tmp/audit-preflight.md
ls .claude/tasks/ >> /tmp/audit-preflight.md

echo -e "\n## Git Status" >> /tmp/audit-preflight.md
git status --short >> /tmp/audit-preflight.md

echo "Pre-flight complete. Context written to /tmp/audit-preflight.md"
cat /tmp/audit-preflight.md
```

Only after this completes successfully, spawn Agents 1–5 in parallel.

---

## How This Task Runs

```
[Step 0]  Pre-flight sync → /tmp/audit-preflight.md
              ↓
[Parallel] Agent 1 — NuxtJS Auditor       → aethel-view/
           Agent 2 — Go Backend Auditor   → aethel-core/
           Agent 3 — Database Auditor     → migrations + schema + queries
           Agent 4 — Security Auditor     → auth, middleware, secrets, headers
           Agent 5 — DevOps Auditor       → Docker, K8s, CI/CD, scripts
              ↓ all complete
[Serial]   Agent 6 — Synthesis Agent      → merge + overall score → final report
```

Each agent writes its findings to `/tmp/audit-agent-N.md` (N = 1..5).
Agent 6 reads all five and writes `docs/reports/audit-YYYY-MM-DD.md`.

---

## Audit Criteria Reference (All Agents Must Apply)

### Completion Scoring

| Score | Meaning |
|-------|---------|
| ✅ Done | Fully implemented, tested, no known gaps |
| 🔄 Partial | Core logic exists but missing tests, edge cases, or wiring |
| ⚠️ Stub | File exists with placeholder / "under construction" content only |
| ❌ Missing | File or feature expected but not present at all |
| 🚫 Blocked | Cannot complete without a prerequisite that is not done |

### Percentage formula

```
completion % = (Done×1.0 + Partial×0.5 + Stub×0.1) / total_items × 100
```

Apply per section and for the overall score. `total_items` = everything you actually find on disk, not the baseline list in this file.

---

## Agent 1 — NuxtJS / Frontend Auditor

**Scope:** `aethel-view/` — all pages, components, composables, plugins, middleware, assets

**You are a senior NuxtJS 4 + Vue 3 + TypeScript engineer.**

### Step 1: Read pre-flight context

```bash
cat /tmp/audit-preflight.md
```

Note every `.vue` and `.ts` file listed under "All Vue/TS Files in aethel-view (current)" — these are the actual files that exist right now. Audit all of them, not just the baseline list below.

### Step 2: Discover all pages

```bash
find aethel-view/app/pages -name "*.vue" | sort
```

For **every file found** (not just the baseline), report:

| Column | What to record |
|--------|----------------|
| File path | Relative to `aethel-view/` |
| Line count | `wc -l` |
| Status | ✅ / 🔄 / ⚠️ / ❌ using completion scoring above |
| definePageMeta | Correct layout + middleware set? |
| Data source | Mock (`useMockData`) / Real (`$fetch`) / Both / None |
| Semantic CSS | Zero palette violations? |
| Notes | Any obvious issues |

**Baseline pages** (written at task-author time — treat as minimum; audit any additional pages found):
```
pages/index.vue                       pages/search.vue
pages/auth/login.vue                  pages/admin/users.vue
pages/dashboard.vue                   pages/admin/routing-rules.vue
pages/dispatch/inbound/index.vue      pages/admin/document-types.vue
pages/dispatch/inbound/new.vue        pages/admin/escalation.vue
pages/dispatch/outbound/index.vue     pages/admin/audit-log.vue
pages/documents/[id].vue              pages/admin/reports.vue
pages/my-documents.vue                pages/admin/settings.vue
pages/outgoing/new.vue                pages/admin/branding.vue
                                      pages/admin/navigation.vue
```

For any page **not in this baseline** that you find on disk, mark it as a bonus and audit it with the same criteria.

### Step 3: Discover all components

```bash
find aethel-view/app/components -name "*.vue" | sort
```

For each component found, check:
- Hardcodes palette classes? (`grep -l "text-slate\|text-indigo\|bg-white\|bg-slate\|text-gray\|text-zinc"`)
- Uses raw `<button>` or `<table>` instead of Nuxt UI primitives?

### Step 4: Composables, plugins, middleware

```bash
find aethel-view/app/composables aethel-view/app/plugins aethel-view/app/middleware -type f | sort
```

For each file found, determine its purpose and check:

**Composables minimum expected** (audit any additional ones found too):
- `useAuth.ts` — access token in `useState` not `localStorage`?
- `useRuntimeConfig.ts` — calls real `$fetch('/api/v1/config')`?
- `useMockData.ts` — prototype data layer, expected to exist

**Plugins:**
- `auth.client.ts` — silent token recovery on page load?

**Middleware:**
- `auth.ts` — redirects unauthenticated?
- `role.ts` — gates by `requiredRole`?

### Step 5: Design system compliance

Run from `aethel-view/`:

```bash
# Must be 0 — palette violations
echo "=== PALETTE VIOLATIONS ===" && grep -rn "text-slate\|text-indigo\|bg-white\|bg-slate\|text-gray\|text-zinc" app/pages/ app/components/ | wc -l
grep -rn "text-slate\|text-indigo\|bg-white\|bg-slate\|text-gray\|text-zinc" app/pages/ app/components/

# Must be > 0 — semantic tokens are being used
echo "=== SEMANTIC TOKEN USAGE ===" && grep -rn "text-body\|text-muted\|text-accent\|bg-surface\|bg-subtle" app/pages/ app/components/ | wc -l

# Must be 0 — localStorage forbidden for auth tokens
echo "=== LOCALSTORAGE USAGE ===" && grep -rn "localStorage" app/ | wc -l
grep -rn "localStorage" app/
```

### Step 6: Build check

```bash
cd aethel-view && pnpm build 2>&1 | tail -30
```

### Output format — write to `/tmp/audit-agent-1.md`

```markdown
## Frontend Audit (aethel-view/)
_Audited at: [timestamp]_
_Files discovered: N .vue files, M .ts files_

### Pages Summary
| Page | Lines | Status | Data Source | Semantic CSS | Notes |
|------|-------|--------|-------------|--------------|-------|

**Total pages found: N (baseline expected: 19)**
**Completion: X Done + Y Partial + Z Stubs = XX%**

### Components Found
- Total: N components
- Palette violations: N files → [list]
- Raw HTML interactive elements: N files → [list]

### Composables / Plugins / Middleware
| File | Found | Purpose | Correct impl | Notes |
|------|-------|---------|-------------|-------|

### Design System Compliance
- Palette violations: N matches (must be 0)
- Semantic token usage: N matches (healthy if > 0)
- localStorage usage: N matches (must be 0)

### Build Status
- pnpm build: ✅ Success / ❌ Failed
- Error (if any): ...

### Frontend Completion Score: XX%
```

---

## Agent 2 — Go Backend Auditor

**Scope:** `aethel-core/` — all `.go` files, `go.mod`, `go.sum`

**You are a senior Go engineer familiar with chi, zerolog, Argon2id, JWT, and PostgreSQL.**

### Step 1: Read pre-flight context

```bash
cat /tmp/audit-preflight.md
```

Note every `.go` file listed under "All Go Files in aethel-core (current)" — audit all of them.

### Step 2: Build and vet

```bash
cd aethel-core
go build ./... 2>&1
go vet ./... 2>&1
```

### Step 3: Discover all packages

```bash
find aethel-core -type d | sort
find aethel-core -name "*.go" | sort
```

For every package directory found, report its status. **Baseline packages** (minimum expected; audit any additional ones found):

```
cmd/aethel/               internal/audit/           internal/api/docs/
internal/app/             internal/blueprint/       internal/rbac/
internal/config/          internal/database/        internal/service/
internal/domain/          internal/api/             internal/worker/
internal/database/repos/  internal/api/handlers/    internal/transport/
```

### Step 4: Repository implementations

```bash
find aethel-core/internal/database/repos -name "*.go" | sort
```

For **every repo file found** (not just the baseline below), check:
- Real implementation or noop/stub? (look for actual SQL calls vs `return nil`)
- Uses `qr.Get("group.name")`? (zero inline SQL strings?)
- References `app.OrgID` instead of orgID method param?

**Baseline repos** (minimum expected):
```
user_repo.go            dispatch_repo.go        minute_sheet_repo.go
session_repo.go         dispatch_event_repo.go  green_note_repo.go
password_reset_repo.go  routing_rule_repo.go    escalation_rule_repo.go
audit_repo.go
```

### Step 5: Service layer

```bash
find aethel-core/internal/service -name "*.go" | sort
```

For each service file found:
- Injects `audit.Writer` (not `domain.AuditRepository` directly)?
- Methods with empty bodies or `// TODO`?

### Step 6: Route coverage

Read `docs/architecture/architecture-api-routes.md` to get the **authoritative** list of expected routes — do not rely on any number written in this task file, as routes may have been added by Tasks 11–14. Then count routes actually registered in `internal/api/server.go`. Report: registered vs expected, list any missing.

### Step 7: Test results

```bash
cd aethel-core
go test ./... 2>&1 | grep -E "^ok|FAIL|no test"
go test ./internal/service/... -v 2>&1 | grep -E "^=== RUN|--- PASS|--- FAIL|--- SKIP"
```

### Step 8: Violation greps

```bash
cd aethel-core

# Must be 0 — no inline SQL in repos or services
echo "=== INLINE SQL ===" && grep -rn "\"SELECT\|\"INSERT\|\"UPDATE\|\"DELETE" internal/database/repos/ internal/service/ | wc -l
grep -rn "\"SELECT\|\"INSERT\|\"UPDATE\|\"DELETE" internal/database/repos/ internal/service/

# Must be 0 — no orgID threaded through methods
echo "=== ORGID PARAMS ===" && grep -rn "orgID\b" internal/service/ internal/database/repos/ | wc -l

# Must be 0 — no org claim in JWT
echo "=== ORG JWT CLAIM ===" && grep -rn "org.*claim\|claims\[.org.\]\|\"org\"" internal/ | wc -l

# Check for noops (unfilled stubs)
echo "=== NOOP REPOS ===" && grep -rn "return nil, nil\|// noop\|// TODO" internal/database/repos/ | wc -l
grep -rn "return nil, nil\|// noop\|// TODO" internal/database/repos/
```

### Output format — write to `/tmp/audit-agent-2.md`

```markdown
## Backend Audit (aethel-core/)
_Audited at: [timestamp]_
_Go files discovered: N_

### Build Status
- go build ./...: ✅ / ❌ (error: ...)
- go vet ./...: ✅ / ❌

### Package Inventory
| Package | Status | Key files | Notes |
|---------|--------|-----------|-------|

### Repository Layer
| Repo File | Exists | Real impl | Zero inline SQL | Uses app.OrgID | Notes |
|-----------|--------|-----------|-----------------|----------------|-------|

**Repos: X/N real, Y stubs/noops**

### Service Layer
| Service | Exists | audit.Writer | Methods complete | TODOs found |
|---------|--------|-------------|-----------------|-------------|

### Route Coverage
- Routes in architecture doc: N (read from the doc)
- Routes registered in server.go: N
- Coverage: XX%
- Missing routes: [list]

### Test Results
- Packages with tests: N
- Tests passing: N
- Tests failing: N (list)
- Packages with no tests: [list]

### Violation Report
- Inline SQL strings: N (must be 0)
- orgID method params: N (must be 0)
- Noop/TODO stubs in repos: N

### Backend Completion Score: XX%
```

---

## Agent 3 — Database Auditor

**Scope:** `aethel-core/internal/database/migrations/`, `aethel-core/internal/database/queries/queries.yaml`, `docs/db-design.mmd`

**You are a senior PostgreSQL engineer.**

### Step 1: Read pre-flight context

```bash
cat /tmp/audit-preflight.md
```

Check "Files Changed in Last 30 Commits" for any new migration files added by Tasks 11–14.

### Step 2: Discover all migration files

```bash
find aethel-core/internal/database/migrations -name "*.sql" | sort
```

Do not assume a fixed count — count what's actually there. For every `.up.sql` found, check a corresponding `.down.sql` exists.

For each migration, record: number, what it creates/alters, and whether it's referenced in `docs/db-design.mmd`.

**Baseline tables** (minimum expected in the ER diagram — check for any additions beyond these):
```
organizations           routing_rules               system_settings
departments             routing_rule_conditions     branding_configs
users                   routing_rule_destinations   audit_ledger (partitioned)
user_sessions           minute_sheets               + migration 21: ALTER branding_configs
password_reset_tokens   green_notes                   adds neutral_palette, font_family, wordmark
notification_preferences notifications
document_types          escalation_rules
dispatches              
dispatch_attachments    
dispatch_events         
```

### Step 3: queries.yaml completeness

```bash
cat aethel-core/internal/database/queries/queries.yaml
```

List every group and query key actually present. Then:

```bash
# Extract all qr.Get() calls from Go source
grep -rn 'qr\.Get(' aethel-core/internal/ | grep -oP '"[^"]+\.(?:[^"]+)"' | sort -u
```

Cross-reference: every key called in Go must exist in the YAML. Report any dangling references.

**Baseline query groups** (minimum expected — audit any additional groups found):
```
auth / users / sessions / password_reset / dispatch / dispatch_events /
routing_rules / minute_sheets / green_notes / escalation_rules /
governance / config / notifications
```

### Step 4: Template variable usage

```bash
grep -rn "{{ .Schema }}\|{{ T \|{{ E " aethel-core/internal/database/migrations/
```

### Output format — write to `/tmp/audit-agent-3.md`

```markdown
## Database Audit
_Audited at: [timestamp]_
_Migration files found: N up + N down_

### Migration Files
- Total up: N  |  Total down: N  |  Paired: ✅/❌
- Highest migration number: N
- Migration 21 (ALTER branding_configs): ✅/❌
- Unpaired migrations: [list]

### Schema vs ER Diagram
| Table | In Diagram | Migration # | Status | Discrepancies |
|-------|-----------|-------------|--------|---------------|

### queries.yaml Coverage
| Group | Exists | Query count | Missing queries |
|-------|--------|-------------|-----------------|

**Total queries defined: N**
**Groups found: N (baseline expected: 13)**

### Dangling qr.Get() Keys (in Go but not in YAML)
- N dangling keys: [list]

### Template Usage
- Migrations using templates: N / N total

### Database Completion Score: XX%
```

---

## Agent 4 — Security Auditor

**Scope:** Entire repository — focus on `aethel-core/internal/`, `aethel-view/app/`, `.env.example`, `aethel-scripts/`

**You are a senior application security engineer.** Verify that all controls in `docs/architecture/architecture-security.md` are actually implemented.

### Step 1: Read pre-flight context + security architecture

```bash
cat /tmp/audit-preflight.md
cat docs/architecture/architecture-security.md
```

The security architecture doc is the **authoritative source** of what controls are required. If new controls were added to the doc by recent tasks, audit those too.

### Step 2: Authentication controls

```bash
# JWT algorithm
grep -rn "SigningMethod\|HS256\|RS256\|alg" aethel-core/internal/service/auth_service.go

# JWT claims — should have sub, role, iat, exp, jti — no org
grep -rn "MapClaims\|RegisteredClaims\|\"sub\"\|\"role\"\|\"org\"\|\"jti\"" aethel-core/internal/service/auth_service.go

# Argon2id params
grep -rn "argon2\|IDKey\|Memory\|Iterations\|Parallelism" aethel-core/internal/service/auth_service.go

# httpOnly cookie
grep -rn "HttpOnly\|SetCookie\|refresh_token" aethel-core/internal/api/handlers/auth.go

# localStorage forbidden
grep -rn "localStorage" aethel-view/app/

# Refresh token rotation — look for tx.Commit in session handling
grep -rn "BeginTx\|RotateSession\|DeleteSession.*InsertSession\|tx\.Commit" aethel-core/internal/service/auth_service.go aethel-core/internal/database/repos/session_repo.go
```

### Step 3: Middleware stack order

Read `aethel-core/internal/api/server.go`. List the exact middleware chain order as it appears in the code. Compare to expected:

```
Expected: Recovery → RequestID → StructuredLogger → RateLimiter → CORS → Auth → RBAC → Handler
```

### Step 4: CSRF protection

```bash
find aethel-core/internal -name "csrf*.go" -o -name "*csrf*.go" | sort
grep -rn "ConstantTimeCompare\|csrf\|X-CSRF" aethel-core/internal/
grep -rn "csrf" aethel-view/app/
```

### Step 5: Security headers

```bash
find aethel-core/internal -name "*security*header*" -o -name "*header*security*" | sort
grep -rn "X-Content-Type-Options\|X-Frame-Options\|Strict-Transport\|Content-Security-Policy\|Referrer-Policy" aethel-core/internal/
```

### Step 6: Secrets and account lockout

```bash
# No hardcoded secrets
grep -rn "AETHEL_JWT_SECRET\s*=\s*\|-----BEGIN RSA PRIVATE\|password\s*:=\s*\"" aethel-core/ aethel-view/ --include="*.go" --include="*.ts" --include="*.vue"

# .env.example exists and has no real values (only placeholders)
cat .env.example

# Account lockout — 423 status
grep -rn "423\|StatusLocked\|ErrAccountLocked" aethel-core/internal/

# db-harden.sql
ls aethel-core/scripts/db-harden.sql 2>/dev/null && grep -c "REVOKE" aethel-core/scripts/db-harden.sql
```

### Output format — write to `/tmp/audit-agent-4.md`

```markdown
## Security Audit
_Audited at: [timestamp]_
_Security architecture doc read: ✅/❌_

### Authentication Controls
| Control | Status | Evidence (file:line) | Notes |
|---------|--------|----------------------|-------|
| JWT algorithm RS256/HS256 | | | |
| JWT claims correct (no org) | | | |
| Argon2id params ≥ 64MiB/3iter/4threads | | | |
| Access token NOT in localStorage | | | |
| Refresh token httpOnly cookie | | | |
| Refresh token rotation atomic | | | |

### Middleware Stack
Expected: Recovery → RequestID → StructuredLogger → RateLimiter → CORS → Auth → RBAC
Actual:   [exact order from server.go]
Match: ✅/❌

### CSRF Protection
| Check | Status | File | Notes |
|-------|--------|------|-------|
| Middleware exists | | | |
| ConstantTimeCompare used | | | |
| Token in readable cookie | | | |
| Applied to POST/PATCH/DELETE | | | |

### Security Headers
| Header | Present | Notes |
|--------|---------|-------|
| X-Content-Type-Options | | |
| X-Frame-Options | | |
| Strict-Transport-Security | | |
| Content-Security-Policy | | |
| Referrer-Policy | | |

### Other Controls
| Control | Status | Notes |
|---------|--------|-------|
| No hardcoded secrets | | |
| .env.example placeholder-only | | |
| Account lockout → 423 | | |
| db-harden.sql exists + REVOKE | | |

### Security Score: XX% (N/M controls ✅)
### Critical findings (❌ items): [list]
```

---

## Agent 5 — DevOps Auditor

**Scope:** `Makefile`, `docker-compose.yml`, `docker-compose.prod.yml`, `aethel-core/Dockerfile`, `aethel-view/Dockerfile`, `k8s/`, `.github/workflows/`, `aethel-scripts/`

**You are a senior DevOps / platform engineer.**

### Step 1: Read pre-flight context

```bash
cat /tmp/audit-preflight.md
```

Note any new workflow or infrastructure files added by recent tasks.

### Step 2: Discover all infra files

```bash
find . -name "Dockerfile*" -o -name "docker-compose*.yml" | sort
find k8s/ -type f 2>/dev/null | sort
find .github/workflows/ -type f 2>/dev/null | sort
find aethel-scripts/ -type f 2>/dev/null | sort
ls Makefile 2>/dev/null
```

Audit **everything found**, not just the baseline below.

### Step 3: Docker

For each Dockerfile found:
- Multi-stage build? (look for multiple `FROM`)
- Production stage uses distroless or minimal base (`scratch`, `gcr.io/distroless/*`, `alpine`)?
- No secrets in ENV or ARG?

For docker-compose files:
- Services defined: postgres, backend, frontend?
- PostgreSQL port: host `5433` → container `5432`?
- Health checks present?
- Secrets from env vars, not hardcoded?

### Step 4: Kubernetes

```bash
find k8s/ -type f | sort
```

**Baseline manifests** (minimum expected — audit any additional manifests found):
```
k8s/postgres/StatefulSet    k8s/backend/Deployment     k8s/frontend/Deployment
k8s/postgres/PVC            k8s/backend/HPA            k8s/frontend/Service
k8s/postgres/Service        k8s/backend/ConfigMap      k8s/ingress.yaml
```

For each found: does it set `namespace: aethel-workspace`?

### Step 5: GitHub Actions

```bash
find .github/workflows/ -name "*.yml" | sort
cat .github/workflows/*.yml 2>/dev/null
```

For each workflow found, read it and check:
- `ci.yml` — runs `go test ./...` AND `pnpm test` AND lint?
- `cd.yml` — builds + pushes to GHCR on merge to main?
- `security.yml` — runs Trivy + govulncheck + gosec?
- Any workflow: PostgreSQL service defined for integration tests?

### Step 6: Scripts

```bash
for f in aethel-scripts/setup-dev.sh aethel-scripts/health-check.sh \
          aethel-scripts/rotate-jwt-secret.sh aethel-scripts/db-backup.sh \
          aethel-scripts/k8s-rollout.sh; do
  [ -f "$f" ] && echo "$f: $(wc -l < $f) lines" || echo "$f: MISSING"
done
```

Also check `aethel-core/scripts/` for any new scripts added:
```bash
find aethel-core/scripts/ -type f 2>/dev/null | sort
```

### Output format — write to `/tmp/audit-agent-5.md`

```markdown
## DevOps Audit
_Audited at: [timestamp]_
_Infra files discovered: N_

### Docker
| Artifact | Exists | Multi-stage | Distroless stage | No hardcoded secrets | Notes |
|----------|--------|-------------|-----------------|----------------------|-------|

### Kubernetes Manifests
| Manifest | Exists | Correct namespace | Notes |
|----------|--------|------------------|-------|

**Manifests: X/N found (baseline expected: 9)**

### GitHub Actions Workflows
| Workflow | Exists | Key steps verified | Notes |
|----------|--------|-------------------|-------|

### Scripts
| Script | Exists | Non-empty | Notes |
|--------|--------|-----------|-------|

### Makefile
- Exists: ✅/❌
- make help: ✅/❌
- Key targets found: [list]

### DevOps Completion Score: XX%
```

---

## Agent 6 — Synthesis Agent (runs after all 5 complete)

### Step 1: Verify all temp files exist

```bash
for i in 1 2 3 4 5; do
  [ -f "/tmp/audit-agent-$i.md" ] && echo "Agent $i: ✅" || echo "Agent $i: ❌ MISSING"
done
```

If any file is missing, wait or re-run the missing agent before continuing.

### Step 2: Read all findings

Read `/tmp/audit-agent-{1..5}.md` in full. Extract the completion score (`XX%`) from each.

### Step 3: Compute overall score

```
overall = (frontend × 0.25) + (backend × 0.30) + (database × 0.15) + (security × 0.20) + (devops × 0.10)
```

### Step 4: Read the agile plan for sprint status comparison

```bash
cat docs/plans/agile-implementation-plan.md | head -40
```

Use the Implementation Status table in the plan doc as the expected baseline. For each sprint, compare what the plan says vs what the audit findings show.

### Step 5: Write the final report

```bash
mkdir -p docs/reports
REPORT="docs/reports/audit-$(date +%Y-%m-%d).md"
```

Report structure:

```markdown
# Aethel Workspace — Project Audit Report
**Date:** YYYY-MM-DD
**Git HEAD:** [git rev-parse --short HEAD]
**Recent changes:** [paste the 10 most recent commits from preflight]
**Audited by:** 5-agent parallel audit team (Claude Code Task 15)

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

## Critical Findings (block Sprint 5 if unresolved)

[List every ❌ item from all 5 agents, grouped by domain. Be specific: file path, what's wrong, what's needed.]

---

## Sprint Progress vs Agile Plan

| Sprint | Plan Status | Audit Actual | Gap |
|--------|-------------|--------------|-----|
| Sprint 0 — Foundation | Complete | [from audit] | [delta] |
| Sprint 1 — Auth | Complete | [from audit] | [delta] |
| Sprint 1.5 — Security | Complete | [from audit] | [delta] |
| Sprint 2 — Dispatch | [from plan doc] | [from audit] | [delta] |
| Sprint 3 — Workflow | [from plan doc] | [from audit] | [delta] |
| Sprint 4 — Governance | [from plan doc] | [from audit] | [delta] |
| Sprint 5 — API Complete | Not started | [from audit] | [delta] |
| Sprint 6 — Hardening | Not started | [from audit] | [delta] |

---

## Statistics Summary

| Metric | Count |
|--------|-------|
| Total Vue pages found | N |
| Pages fully complete | N |
| Pages stub/incomplete | N |
| Go packages found | N |
| Repo implementations (real) | N |
| Repo implementations (stub/noop) | N |
| Total unit tests | N |
| Passing tests | N |
| SQL migrations | N |
| query.yaml groups | N |
| Routes registered | N |
| Security controls ✅ | N/M |
| Inline SQL violations | N (must be 0) |
| Palette CSS violations | N (must be 0) |
| localStorage violations | N (must be 0) |

---

## Detailed Findings

[Paste each agent's full `/tmp/audit-agent-N.md` verbatim, separated by `---`]

---

## Recommendations for Next Session

Top 5 highest-impact items, ordered by priority:

1. [item]
2. [item]
3. [item]
4. [item]
5. [item]
```

### Step 6: Print summary to console

After writing the file, print the Executive Summary table and Critical Findings section directly to the terminal so the user sees results immediately.

### Step 7: Commit

```bash
git add docs/reports/
git commit -m "chore(audit): project progress report $(date +%Y-%m-%d)

Overall score: XX%
Frontend: XX% | Backend: XX% | Database: XX% | Security: XX% | DevOps: XX%"
git push origin dev
```

---

## Execution Instructions

```
1. Run Step 0 pre-flight (orchestrating session, not a subagent)
2. Verify /tmp/audit-preflight.md was written successfully
3. Spawn Agents 1–5 in a single message (5 parallel Agent tool calls)
   — pass /tmp/audit-preflight.md path in each agent's prompt
4. Wait for all 5 completion notifications
5. Spawn Agent 6 to synthesize
6. Agent 6 writes + commits the final report
7. Print Executive Summary to console
```

## Definition of Done

- [ ] `/tmp/audit-preflight.md` written with current git state
- [ ] All 5 audit temp files written to `/tmp/audit-agent-{1..5}.md`
- [ ] Final report written to `docs/reports/audit-YYYY-MM-DD.md`
- [ ] Overall completion percentage computed with correct weighting
- [ ] Executive Summary printed to console
- [ ] Critical findings section lists every ❌ item with file paths
- [ ] Sprint progress table shows plan vs actual for all 8 sprints
- [ ] Statistics summary table populated with real counts
- [ ] Report committed and pushed to `dev` branch
- [ ] Zero files were modified (read-only audit)
