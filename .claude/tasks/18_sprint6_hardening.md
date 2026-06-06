# Task 18 — Sprint 6: Production Hardening

**Sprint goal:** Take the fully functional Aethel backend (Sprints 0–5) to production readiness. Deliver: configurable rate limiting and structured request logging, Argon2id params from blueprint, migrate validate SQL syntax check, benchmark and load tests, and the four repository documentation files (CHANGELOG, SECURITY, CONTRIBUTING, CODE_OF_CONDUCT).

**Prerequisites:** Task 17 complete. `go build ./...`, `go vet ./...`, `pnpm build`, and `go test ./...` all pass before this task begins.

**Primary outputs:**
- Configurable per-IP/per-user token bucket rate limiter
- `zerolog` structured logger with `request_id`, `user_id`, `latency_ms` fields on every request
- Argon2id cost params read from blueprint (not hardcoded)
- `migrate validate` with PostgreSQL `EXPLAIN`-based SQL syntax checking
- Benchmark tests: routing rule engine, hash chain, audit ledger insert
- `CHANGELOG.md`, `SECURITY.md`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md` at repo root
- `go test ./... -race` passes with zero data races

---

## Execution Flow

```
[Pre-flight]  Read current state → /tmp/t18-preflight.md
                   ↓
[Parallel]   Agent 1 — Rate Limiting + Structured Logging (Go middleware)
             Agent 2 — Argon2id from Blueprint + migrate validate SQL check (Go infra)
             Agent 3 — Benchmark + Race Tests (Go tests)
             Agent 4 — Repository Docs (CHANGELOG, SECURITY, CONTRIBUTING, CODE_OF_CONDUCT)
                   ↓ all complete
[Serial]     Agent 5 — Integration Verification + Commit
```

All four parallel agents write to non-overlapping files:
- Agent 1: `internal/api/middleware/`, `internal/api/server.go`
- Agent 2: `internal/config/`, `internal/database/migrator.go`, `internal/service/auth_service.go`
- Agent 3: `internal/service/*_bench_test.go`, `internal/database/*_bench_test.go`
- Agent 4: `CHANGELOG.md`, `SECURITY.md`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md` (repo root)

---

## Pre-flight (run in orchestrating session before spawning agents)

```bash
REPO=/Users/hoangharry/mh_code/internships/Bravo/aethel-workspace
cd "$REPO"

echo "## Go packages" > /tmp/t18-preflight.md
find aethel-core -type d | sort >> /tmp/t18-preflight.md

echo -e "\n## Middleware files" >> /tmp/t18-preflight.md
ls aethel-core/internal/api/middleware/ >> /tmp/t18-preflight.md

echo -e "\n## Blueprint yaml (server-database)" >> /tmp/t18-preflight.md
cat blueprints/server-database.yaml >> /tmp/t18-preflight.md

echo -e "\n## auth_service argon2id params" >> /tmp/t18-preflight.md
grep -n "argon2\|Memory\|Iterations\|Parallelism\|argon2id" aethel-core/internal/service/auth_service.go >> /tmp/t18-preflight.md

echo -e "\n## migrator.go current validate impl" >> /tmp/t18-preflight.md
grep -n "Validate\|validate\|EXPLAIN" aethel-core/internal/database/migrator.go >> /tmp/t18-preflight.md

echo -e "\n## Existing test files" >> /tmp/t18-preflight.md
find aethel-core -name "*_test.go" | sort >> /tmp/t18-preflight.md

echo -e "\n## Repo root docs" >> /tmp/t18-preflight.md
ls /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/*.md 2>/dev/null >> /tmp/t18-preflight.md

echo -e "\n## Git HEAD" >> /tmp/t18-preflight.md
git rev-parse --short HEAD >> /tmp/t18-preflight.md

echo "Pre-flight complete."
cat /tmp/t18-preflight.md
```

---

## Agent 1 — Rate Limiting + Structured Logging

**You are a senior Go engineer** specializing in HTTP middleware and observability. Harden the request pipeline with production-grade rate limiting and structured logging.

**Working directory:** `aethel-core/`

### Step 1: Read current state

```bash
cat /tmp/t18-preflight.md
cat internal/api/server.go
cat internal/api/middleware/rate_limiter.go 2>/dev/null || echo "does not exist"
cat internal/blueprint/loader.go | grep -A10 "RateLimit\|rate_limit\|RPM\|rpm"
cat blueprints/server-database.yaml | grep -A5 "rate_limit\|http\|server"
```

### Step 2: Configurable token bucket rate limiter

The current rate limiter (if it exists) uses hardcoded values. Upgrade it to read limits from the blueprint config:

**In `internal/blueprint/loader.go` (or a dedicated `server_config.go`)**, define or extend the server config struct to include:

```go
type ServerConfig struct {
    Port            int    `yaml:"port"`
    RateLimitRPM    int    `yaml:"rate_limit_rpm"`    // default: 600
    RateLimitBurst  int    `yaml:"rate_limit_burst"`  // default: 100
    BodyLimitBytes  int64  `yaml:"body_limit_bytes"`  // default: 1048576 (1 MiB)
}
```

If `blueprints/server-database.yaml` does not have these fields, add them with the defaults documented above, under a `server:` key.

**In `internal/api/middleware/rate_limiter.go`**, implement a two-tier token bucket:

```go
// RateLimiter returns middleware that enforces per-IP and per-user rate limits.
// rpm: requests per minute (refill rate); burst: maximum burst size.
// Per-IP limit fires for unauthenticated requests.
// Per-user limit fires for authenticated requests (keyed by user ID from JWT context).
// Exceeding the limit returns 429 Too Many Requests with Retry-After header.
// Buckets are cleaned up after 10 minutes of inactivity (background goroutine or sync.Map with expiry).
func RateLimiter(rpm int, burst int) func(http.Handler) http.Handler
```

Implementation requirements:
- Use `golang.org/x/time/rate` (already in go.mod, or add it)
- `sync.Map` to store per-key `*rate.Limiter` instances
- Background cleanup: a goroutine that ticks every 5 minutes and removes limiters not accessed in the last 10 minutes — requires wrapping the limiter with a `lastSeen time.Time`
- The `Retry-After` header value is `int(limiter.Reserve().Delay().Seconds()) + 1`

### Step 3: Structured request logging middleware

In `internal/api/middleware/logger.go` (create or replace the current stub), implement a zerolog-based request logger:

```go
// StructuredLogger returns middleware that logs every request with zerolog.
// Log fields: request_id, method, path, status_code, latency_ms, user_id (if authenticated), remote_ip, user_agent.
// Level: INFO for 2xx/3xx, WARN for 4xx, ERROR for 5xx.
// Skips logging for /healthz and /readyz to avoid log spam.
func StructuredLogger(logger zerolog.Logger) func(http.Handler) http.Handler
```

Implementation notes:
- Capture `start := time.Now()` at the start; log `time.Since(start).Milliseconds()` after `next.ServeHTTP`
- Use a `responseWriterWrapper` that captures the status code written by downstream handlers
- Read `request_id` from `r.Context()` (set by the RequestID middleware that runs before this one)
- Read `user_id` from context claims if present (do not error if absent — unauthenticated requests also get logged)

### Step 4: Wire into server.go

Ensure the middleware is registered in the correct order in `server.go`:

```
Recovery → RequestID → StructuredLogger → RateLimiter → CORS → Auth → RBAC → Handler
```

Read `server.go` to verify exact current order. Adjust if needed.

Also add the `http.MaxBytesReader` body limit middleware (from Task 16 if not already done):

```go
func MaxBodySize(limit int64) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            r.Body = http.MaxBytesReader(w, r.Body, limit)
            next.ServeHTTP(w, r)
        })
    }
}
```

Register it immediately after `Recovery`.

### Step 5: Verify

```bash
go build ./... 2>&1
go vet ./... 2>&1

# Verify rate limiter is wired
grep -n "RateLimiter\|rate_limiter" internal/api/server.go

# Verify logger is wired
grep -n "StructuredLogger\|structured_logger\|Logger" internal/api/server.go

# Verify MaxBytesReader
grep -n "MaxBytesReader\|MaxBodySize" internal/api/server.go internal/api/middleware/
```

### Output

Write to `/tmp/t18-agent-1.md`:

```markdown
## Agent 1 — Rate Limiting + Structured Logging
_Completed at: [timestamp]_

### Files modified/created
- internal/api/middleware/rate_limiter.go: ✅/❌
- internal/api/middleware/logger.go: ✅/❌
- internal/api/server.go (middleware order): ✅/❌

### Middleware stack (exact order in server.go)
1. ...

### Rate limiter
- Per-IP bucket: ✅/❌
- Per-user bucket: ✅/❌
- Blueprint-configurable: ✅/❌
- Cleanup goroutine: ✅/❌

### Build: ✅/❌
```

---

## Agent 2 — Argon2id from Blueprint + migrate validate SQL Check

**You are a senior Go engineer** with expertise in cryptography configuration and database tooling. Make Argon2id cost parameters runtime-configurable and strengthen the `migrate validate` command with real SQL syntax checking.

**Working directory:** `aethel-core/`

### Step 1: Read current state

```bash
cat /tmp/t18-preflight.md
cat internal/service/auth_service.go | head -50
cat internal/database/migrator.go
cat internal/blueprint/loader.go
cat blueprints/server-database.yaml
```

### Step 2: Argon2id params from blueprint

Currently `auth_service.go` hardcodes:
```go
defaultMemoryKiB  = 65536
defaultIterations = 3
defaultParallelism = 4
```

**Step 2a:** Add an `Auth` section to the database blueprint struct in `internal/blueprint/loader.go`:

```go
type AuthConfig struct {
    Argon2MemoryKiB   uint32 `yaml:"argon2_memory_kib"`   // default: 65536 (64 MiB)
    Argon2Iterations  uint32 `yaml:"argon2_iterations"`   // default: 3
    Argon2Parallelism uint8  `yaml:"argon2_parallelism"`  // default: 4
    AccessTokenTTL    int    `yaml:"access_token_ttl_min"` // default: 30
    RefreshTokenTTL   int    `yaml:"refresh_token_ttl_days"` // default: 30
}
```

Add this struct as an `Auth AuthConfig` field to the top-level blueprint struct. Add YAML defaults via struct tags or a `SetDefaults()` method called after `yaml.Unmarshal`.

**Step 2b:** Add the corresponding fields to `blueprints/server-database.yaml` under an `auth:` key:

```yaml
auth:
  argon2_memory_kib: 65536
  argon2_iterations: 3
  argon2_parallelism: 4
  access_token_ttl_min: 30
  refresh_token_ttl_days: 30
```

**Step 2c:** Thread the config into `auth_service.go`. The `AuthService` struct should accept an `AuthConfig` (or embed the relevant fields). Replace the three hardcoded constants with values from the config. Ensure the service is constructed with config values in `main.go`.

### Step 3: migrate validate SQL syntax check

The current `Migrator.Validate()` renders templates and checks for render errors, but does NOT verify SQL syntax. Extend it to optionally send each rendered SQL to PostgreSQL using `EXPLAIN` (which parses without executing).

In `internal/database/migrator.go`, extend `Validate()`:

```go
// Validate renders all pending migration templates and optionally checks SQL syntax.
// If db is non-nil, each rendered SQL statement is wrapped in a transaction with EXPLAIN
// to verify syntax, then the transaction is rolled back.
// If db is nil, validation is template-only (no database connection required).
func (m *Migrator) Validate(ctx context.Context, db *sql.DB) error
```

Implementation:
1. Render each `.up.sql` template using the existing `BlueprintContext`
2. If `db != nil`: for each rendered SQL statement (split on `;\n`), run:
   ```sql
   BEGIN;
   EXPLAIN <statement>;
   ROLLBACK;
   ```
   If PostgreSQL returns a syntax error, wrap it with the migration filename and statement number, and return the error immediately.
3. Log results using `zerolog`: `logger.Info().Str("migration", name).Msg("syntax valid")` or `logger.Error().Err(err).Str("migration", name).Msg("syntax error")`
4. Update the `migrate validate` cobra subcommand to open a DB connection when `--check-syntax` flag is passed (default: off, so the command still works without a running database)

### Step 4: Fix accessToken TTL mismatch (from Task 15 audit)

The audit found that `auth_service.go:28` sets `accessTokenDuration=30min` but `auth.go:93` advertises `expires_in: 900` (15 min). Fix the advertised value:

```bash
grep -n "expires_in\|900\|accessTokenDuration" internal/api/handlers/auth.go internal/service/auth_service.go
```

The `expires_in` field in the login response must match the actual JWT `exp` claim. After threading `AccessTokenTTL` from blueprint (Step 2c), the login response should compute: `"expires_in": cfg.Auth.AccessTokenTTL * 60`.

### Step 5: Verify

```bash
go build ./... 2>&1
go vet ./... 2>&1

# Verify argon2 params are no longer hardcoded
grep -n "65536\|defaultMemory\|defaultIter" internal/service/auth_service.go

# Verify validate supports --check-syntax flag
grep -n "check-syntax\|CheckSyntax\|EXPLAIN" internal/database/migrator.go
```

### Output

Write to `/tmp/t18-agent-2.md`:

```markdown
## Agent 2 — Argon2id Config + migrate validate
_Completed at: [timestamp]_

### Argon2id from blueprint
- Blueprint struct updated: ✅/❌
- server-database.yaml auth section: ✅/❌
- auth_service hardcoded constants removed: ✅/❌
- expires_in TTL mismatch fixed: ✅/❌

### migrate validate
- Template-only mode (no DB): ✅/❌
- --check-syntax mode (EXPLAIN): ✅/❌

### Build: ✅/❌
```

---

## Agent 3 — Benchmark + Race Tests

**You are a senior Go engineer** specializing in performance testing and concurrency safety. Write benchmark tests for the three hot paths and verify the codebase is race-condition-free.

**Working directory:** `aethel-core/`

### Step 1: Read current state

```bash
cat /tmp/t18-preflight.md
find . -name "*_test.go" | sort
cat internal/service/dispatch_service.go | grep -n "EvaluateRouting\|ruleApplies" | head -10
cat internal/service/workflow_service.go | grep -n "AppendGreenNote\|hash\|Chain" | head -10
cat internal/database/repos/audit_repo.go | head -40
```

### Step 2: Write `internal/service/dispatch_bench_test.go`

Benchmark the routing rule engine with 1000 rules:

```go
package service_test

import (
    "context"
    "testing"
    // import domain types
)

// BenchmarkRoutingRuleEngine benchmarks EvaluateRoutingRules with 1000 active rules.
// Setup: create a dispatch with priority=IMMEDIATE and document_type_id=X.
// Create 999 non-matching rules and 1 matching rule at the end.
// Verify the matching rule is found AND benchmark the evaluation latency.
// Target: < 5ms for 1000 rules (sufficient for real workloads of < 50 rules).
func BenchmarkRoutingRuleEngine(b *testing.B) {
    rules := make([]domain.RoutingRule, 1000)
    // ... populate rules: 999 with wrong conditions, 1 matching
    dispatch := domain.Dispatch{PriorityLevel: "IMMEDIATE", DocumentTypeID: matchingTypeID}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        // call the rule evaluation logic with the 1000 rules
    }
}
```

### Step 3: Write `internal/service/workflow_bench_test.go`

Benchmark hash chain validation with 100 notes:

```go
// BenchmarkHashChainValidation benchmarks AppendGreenNote for a chain of 100 notes.
// Each iteration appends one note and validates the chain up to that point.
// Target: < 10ms per note append including hash computation.
func BenchmarkHashChainValidation(b *testing.B) { ... }

// BenchmarkHashChainVerify benchmarks verifying an already-built 100-note chain.
// Target: < 100ms for full chain verification.
func BenchmarkHashChainVerify(b *testing.B) { ... }
```

### Step 4: Write `internal/database/repos/audit_bench_test.go`

Benchmark audit ledger insert throughput:

```go
// BenchmarkAuditLedgerInsert benchmarks concurrent audit event inserts.
// Target: 1000 inserts/sec on local PostgreSQL 16 (single-connection baseline).
// Uses build tag: //go:build integration
// This skips automatically in unit test runs without a database.
func BenchmarkAuditLedgerInsert(b *testing.B) {
    if testing.Short() {
        b.Skip("skipping DB benchmark in short mode")
    }
    // ... setup db connection
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        // insert one audit entry
    }
    b.ReportAllocs()
}
```

### Step 5: Run race detector on all unit tests

```bash
cd /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/aethel-core

# Run all unit tests with race detector (skip integration tests that need DB)
go test -race -short ./... 2>&1

# Run benchmarks (no DB needed for service benchmarks)
go test -bench=BenchmarkRoutingRuleEngine -benchtime=5s ./internal/service/... 2>&1
go test -bench=BenchmarkHashChain -benchtime=5s ./internal/service/... 2>&1
```

If any race condition is detected, fix it before finishing. Common causes in this codebase: concurrent map writes in the config cache, SSEBroker clients map, or the query registry.

### Step 6: Verify

```bash
go test -race -short ./... 2>&1 | grep -E "^ok|FAIL|DATA RACE"
```

All packages must show `ok` — no `FAIL` or `DATA RACE`.

### Output

Write to `/tmp/t18-agent-3.md`:

```markdown
## Agent 3 — Benchmarks + Race Tests
_Completed at: [timestamp]_

### Benchmark files written
- internal/service/dispatch_bench_test.go: ✅/❌
- internal/service/workflow_bench_test.go: ✅/❌
- internal/database/repos/audit_bench_test.go: ✅/❌

### Race detector results
- go test -race -short ./...: ✅ (0 races) / ❌ (N races found)
- Races fixed: [list or "none"]

### Benchmark results (informational)
| Benchmark | ns/op | allocs/op | Pass threshold |
|-----------|-------|-----------|----------------|
| RoutingRuleEngine/1000rules | | | < 5ms |
| HashChainValidation/100notes | | | < 10ms/note |
```

---

## Agent 4 — Repository Documentation

**You are a senior technical writer AND open source community manager.** Write four essential repository documentation files at the repo root. Each file must be complete, professional, and suitable for a real open source project.

**Working directory:** `/Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/`

### Step 1: Read existing docs for context

```bash
cat /tmp/t18-preflight.md
cat README.md 2>/dev/null | head -30 || echo "No README"
cat CLAUDE.md | head -40
cat docs/plans/agile-implementation-plan.md | head -30
cat aethel-core/go.mod | head -10
```

### Step 2: Write `CHANGELOG.md`

Follow [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format. Write a single `## [1.0.0] — 2026-06-06` entry.

Structure:

```markdown
# Changelog

All notable changes to Aethel Workspace are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Aethel Workspace uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [1.0.0] — 2026-06-06

### Added
[List all major features delivered across Sprints 0–6, grouped logically:
- Go backend: CLI (serve, migrate), 50+ REST endpoints, 9-layer middleware stack
- Auth: JWT/HS256, Argon2id, httpOnly refresh cookie, CSRF protection, rate limiting, account lockout
- DAK Diarization pillar: dispatch CRUD, routing rule engine, dispatch events timeline
- Green Notes pillar: minute sheets, cryptographic hash chain, approval workflow
- Governance pillar: tamper-evident audit ledger, chain verification, escalation worker
- Config API: runtime branding/nav/feature toggle with 5-minute cache
- Real-time: SSEBroker, per-user notification stream
- Frontend: Nuxt 4, 19 pages, 3 roles (ADMIN/RECEPTION/USER), full i18n (EN/VI)
- DevOps: 2-stage Docker images, Docker Compose (dev + prod), K8s manifests with HPA, 3 GitHub Actions workflows
- Security: security headers (X-Content-Type-Options, X-Frame-Options, HSTS, CSP), db-harden.sql
- Observability: structured zerolog logging with request_id/user_id/latency_ms, /healthz + /readyz probes
- Docs: OpenAPI 3.1 spec (58 operationIds), Scalar UI, architecture docs, IT customization guide]

### Blueprint schema versions
- server-database.yaml: v1.0
- ui-theme.yaml: v1.0
- ui-layouts.yaml: v1.0
```

### Step 3: Write `SECURITY.md`

Place at repo root. Must include all five required sections:

1. **Supported Versions** — table showing v1.0.x as ✅ receiving security patches; earlier versions (pre-1.0) as ❌ not supported.

2. **Reporting a Vulnerability** — instruct reporters to use GitHub's private vulnerability reporting (Security tab → Report a vulnerability). Explicitly state: do NOT open a public GitHub issue for security bugs. Alternative contact: tonminhhoang.work@gmail.com.

3. **Response SLA** — acknowledge within 48 hours; triage within 7 days; patch release within 30 days for CVSS ≥ 7.0 (Critical/High); 90 days for CVSS < 7.0 (Medium/Low).

4. **Scope** — In scope: auth bypass, SQL injection, privilege escalation (RBAC bypass), JWT forgery, XSS in admin pages, sensitive data exposure via API. Out of scope: rate limiting edge cases, UI cosmetics, issues requiring physical access, issues in third-party libraries (report those upstream).

5. **Credit** — reporters credited in CHANGELOG.md and the GitHub Security Advisory for the CVE, unless they request anonymity.

### Step 4: Write `CONTRIBUTING.md`

Must include all seven required sections:

1. **Development Setup** — prerequisites: Go 1.24+, Node 22+, pnpm 9+, PostgreSQL 16, Docker 24+. Step-by-step:
   ```bash
   git clone <repo>
   cp .env.example .env  # edit POSTGRES_PASSWORD and AETHEL_JWT_SECRET
   bash aethel-scripts/setup-dev.sh
   make dev  # starts docker-compose: postgres + backend + frontend
   ```

2. **Branch Naming** — `feat/short-description`, `fix/short-description`, `chore/short-description`, `docs/short-description`. Branch from `dev`, PR into `dev`. PRs from `dev` into `main` are release merges only.

3. **Commit Message Format** — Conventional Commits. Examples: `feat(dispatch): add priority-based routing`, `fix(auth): remove hardcoded JWT fallback`, `chore(deps): upgrade chi to v5.2`. Breaking changes: add `BREAKING CHANGE:` footer.

4. **Pull Request Requirements** — `go test ./...` passes; `pnpm test` passes; `go vet ./...` passes; no TypeScript errors (`pnpm exec vue-tsc --noEmit`); CI green; PR description links to a GitHub issue; at least one reviewer approved.

5. **Code Style** — Go: `gofmt` (enforced by CI) + `golangci-lint run`. Vue/TypeScript: ESLint via `eslint.config.mjs` (`pnpm lint`). No commented-out code. No `console.log` in production paths.

6. **Testing Expectations** — New API endpoints require at least one integration test in `internal/integration/` against a real PostgreSQL instance (no mocked DB per project convention — see Task 16 rationale). Service-layer unit tests use interface-based mocks via the domain repository interfaces.

7. **License** — By submitting a PR, contributors agree their code is licensed under Apache 2.0 and that they have the right to submit it.

### Step 5: Write `CODE_OF_CONDUCT.md`

Use **Contributor Covenant v2.1 verbatim**. Fill in:
- Enforcement contact email: `tonminhhoang.work@gmail.com`
- Project maintainer: Minh Hoang Ton

The Contributor Covenant v2.1 text is available at https://www.contributor-covenant.org/version/2/1/code_of_conduct/ — use the exact text, do not paraphrase or summarize.

### Step 6: Verify all four files exist and are non-empty

```bash
cd /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace
for f in CHANGELOG.md SECURITY.md CONTRIBUTING.md CODE_OF_CONDUCT.md; do
  [ -f "$f" ] && echo "$f: ✅ ($(wc -l < $f) lines)" || echo "$f: ❌ MISSING"
done
```

### Output

Write to `/tmp/t18-agent-4.md`:

```markdown
## Agent 4 — Repository Documentation
_Completed at: [timestamp]_

### Files written
| File | Lines | Status |
|------|-------|--------|
| CHANGELOG.md | N | ✅/❌ |
| SECURITY.md | N | ✅/❌ |
| CONTRIBUTING.md | N | ✅/❌ |
| CODE_OF_CONDUCT.md | N | ✅/❌ |

### Changelog version: 1.0.0 — 2026-06-06
### SECURITY.md contact: tonminhhoang.work@gmail.com ✅
### CODE_OF_CONDUCT.md: Contributor Covenant v2.1 verbatim ✅/❌
```

---

## Agent 5 — Integration Verification + Commit

**Run after all 4 parallel agents complete.** Final validation: race test, build, violations check, then commit everything.

### Step 1: Verify all agent outputs exist

```bash
for i in 1 2 3 4; do
  [ -f "/tmp/t18-agent-$i.md" ] && echo "Agent $i: ✅" || echo "Agent $i: ❌ MISSING"
done
```

### Step 2: Full build verification

```bash
cd /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/aethel-core
go build ./... 2>&1
go vet ./... 2>&1

cd /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/aethel-view
pnpm build 2>&1 | tail -30
```

Both must exit 0.

### Step 3: Race detector — final run

```bash
cd /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/aethel-core
go test -race -short -count=1 ./... 2>&1 | tee /tmp/t18-race-results.txt
grep -E "DATA RACE|FAIL" /tmp/t18-race-results.txt || echo "✅ No races detected"
```

**If any DATA RACE is found**, fix it before committing. Common patterns:
- Concurrent map read/write → protect with `sync.RWMutex`
- Config cache stale pointer → ensure cache writes use a write lock
- SSEBroker clients map → already uses `sync.RWMutex` (verify)

### Step 4: Violation checks

```bash
cd /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace
echo "=== Inline SQL ===" && grep -rn '"SELECT\|"INSERT\|"UPDATE\|"DELETE' aethel-core/internal/database/repos/ | wc -l
echo "=== Hardcoded secret ===" && grep -rn '"dev-secret' aethel-core/ | wc -l
echo "=== Palette violations ===" && grep -rn "text-slate\|text-indigo\|bg-white\|bg-slate" aethel-view/app/pages/ aethel-view/app/components/ | wc -l
echo "=== localStorage ===" && grep -rn "localStorage" aethel-view/app/ | wc -l
```

All must be 0.

### Step 5: Run all tests one final time

```bash
cd /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/aethel-core
go test -short ./... 2>&1 | grep -E "^ok|FAIL"
```

All packages must show `ok`.

### Step 6: Commit

```bash
cd /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace
git add aethel-core/ aethel-view/ CHANGELOG.md SECURITY.md CONTRIBUTING.md CODE_OF_CONDUCT.md blueprints/
git commit -m "$(cat <<'EOF'
feat(sprint6): production hardening — rate limiting, logging, benchmarks, repo docs

- Rate limiter: per-IP + per-user token bucket, RPM from blueprint, 429 + Retry-After
- Structured logging: zerolog with request_id, user_id, latency_ms, skip healthz/readyz
- MaxBytesReader: 1 MiB body limit on all requests
- Argon2id: cost params (memory_kib, iterations, parallelism) and token TTLs from blueprint
- migrate validate: --check-syntax flag wraps SQL in BEGIN/EXPLAIN/ROLLBACK
- Benchmarks: routing rule engine (1000 rules), hash chain (100 notes), audit insert
- go test -race -short ./...: 0 data races
- CHANGELOG.md: v1.0.0 entry (Keep a Changelog format)
- SECURITY.md: private reporting, 48h SLA, scope, credit policy
- CONTRIBUTING.md: setup guide, branch/commit conventions, PR requirements, testing expectations
- CODE_OF_CONDUCT.md: Contributor Covenant v2.1 (Minh Hoang Ton, tonminhhoang.work@gmail.com)

Co-Authored-By: Claude Code Task 18 <noreply@anthropic.com>
EOF
)"
```

---

## Definition of Done

- [ ] `go build ./...` passes with zero errors
- [ ] `go vet ./...` passes with zero warnings
- [ ] `pnpm build` passes with zero TypeScript errors
- [ ] `go test -race -short ./...` passes with zero data races
- [ ] Rate limiter is configurable from blueprint (`rate_limit_rpm`, `rate_limit_burst`)
- [ ] Every request log line contains `request_id`, `status_code`, `latency_ms`
- [ ] `MaxBytesReader` middleware is registered in `server.go`
- [ ] Argon2id constants removed from `auth_service.go` — values come from blueprint
- [ ] `migrate validate --check-syntax` runs `EXPLAIN` against each migration SQL
- [ ] `expires_in` in login response matches actual JWT `exp` claim
- [ ] `internal/service/dispatch_bench_test.go` exists with `BenchmarkRoutingRuleEngine`
- [ ] `internal/service/workflow_bench_test.go` exists with `BenchmarkHashChainValidation`
- [ ] `internal/database/repos/audit_bench_test.go` exists with `BenchmarkAuditLedgerInsert`
- [ ] `CHANGELOG.md` exists at repo root with v1.0.0 entry
- [ ] `SECURITY.md` exists at repo root with all 5 required sections
- [ ] `CONTRIBUTING.md` exists at repo root with all 7 required sections
- [ ] `CODE_OF_CONDUCT.md` exists at repo root — Contributor Covenant v2.1 verbatim
- [ ] Zero inline SQL violations
- [ ] Zero hardcoded secret violations
- [ ] Zero palette CSS violations
- [ ] Zero localStorage violations
- [ ] All changes committed to `dev` branch
