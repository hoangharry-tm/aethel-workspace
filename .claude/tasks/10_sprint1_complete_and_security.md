# Task 10 — Sprint 1 Complete + Sprint 1.5 Security (Combined)

**Working directory:** `aethel-workspace/` (repo root)  
**Targets:** `aethel-core/` (backend) and `aethel-view/` (frontend)  
**Depends on:** Nothing — this is the starting point for all real backend work.

---

## IMPORTANT: Read This Entire File Before Writing Any Code

This task covers two sequential blocks. **Block A must be fully verified before Block B begins.** Do not interleave them. Read the invariant table and threat model first — every decision in this task is governed by those constraints.

---

## Invariant Table — Non-Negotiable Rules

| Rule | Enforcement |
|---|---|
| Single-tenant: `app.OrgID` is the fixed org constant | Never thread orgID through service method signatures or JWT claims. Every handler reads `app.OrgID` directly. |
| JWT claims: exactly `sub`, `role`, `iat`, `exp`, `jti` — no `org` | `issueAccessToken()` in `auth_service.go` is already correct. Do not add any claim. |
| Access token storage: in-memory `useState` only | Never write to `localStorage`, `sessionStorage`, or any readable cookie. |
| Refresh token storage: httpOnly cookie only | The server sets it; JavaScript cannot read it; it is never in the response body. |
| CSRF: double-submit cookie, `crypto/subtle.ConstantTimeCompare` | Regular `==` string comparison is a timing-attack vulnerability. |
| Argon2id: 64 MiB memory (`65536` KiB), 3 iterations, 4 threads | `auth_service.go` already uses these constants. Do not change them. |
| Refresh token rotation: DELETE old + INSERT new in one DB transaction | `session_repo.go` already has `RotateSession`. Use it. |
| Account locked → HTTP 423, not 403 | Handler maps `domain.ErrAccountLocked` to `http.StatusLocked`. Already done. |
| No inline SQL in repos | All SQL goes through `BuildQueryRegistry` from `queries.yaml`. Dispatch repos use this pattern. Auth/audit repos must match. |
| No mock DB in tests | Unit tests use inline mock implementations of domain interfaces. Integration tests use a real PostgreSQL 16 instance with `//go:build integration`. |
| Secrets are never logged | Only log algorithm names and existence-check failures. Never log key values. |

---

## Threat Model Summary

| Attack Vector | Mitigation | Current Status |
|---|---|---|
| Password database breach | Argon2id PHC-formatted hash (64 MiB, 3 iter, 4 threads) | auth_service.go — done |
| XSS token theft | Access token in `useState` memory; refresh in httpOnly cookie | useAuth.ts — done |
| CSRF forgery | Double-submit cookie (`csrf_token` cookie + `X-CSRF-Token` header) verified with `ConstantTimeCompare` | csrf.go + auth handler — done |
| Credential stuffing | 20 RPM per-IP on `/auth/login` + account lockout after 5 failures | rate_limiter.go + auth_service.go — done |
| Session hijacking post-logout | Server-side session delete on logout | auth handler + session_repo — done |
| Refresh token theft/replay | Rotation: old token deleted, new token issued atomically | RotateSession() in session_repo — done |
| JWT replay after account deactivation | Short-lived 15-min access tokens; deactivated user fails GetByID on refresh | auth_service.go — done |
| Clickjacking / MIME sniff / info leak | Security headers middleware | security_headers.go — done |
| Email enumeration on login | Same 401 for wrong email and wrong password | auth_service.go — done |
| Email enumeration on password reset | Always return 202 regardless of whether email exists | auth handler — done |
| Audit ledger tampering | PostgreSQL `REVOKE UPDATE, DELETE ON audit_ledger FROM aethel_app` | db-harden.sh exists; SQL file missing — Block A Step 4 |
| RBAC privilege escalation | `rbac.Require()` middleware checks role permission map | rbac/middleware.go — done |
| Secrets in logs | Env var values never passed to log statements | main.go — done; verify in Block A Step 5 |

---

## Current State Audit (Read This Before Step 0)

**What already exists and is correct — do NOT rewrite:**
- `aethel-core/internal/service/auth_service.go` — Login, RefreshSession, Logout, RequestPasswordReset, ConfirmPasswordReset, Argon2id, JWT (no org claim), correct lockout logic
- `aethel-core/internal/database/repos/user_repo.go` — UserRepo with all 10 methods (uses inline SQL, not QueryRegistry — this is acceptable; dispatch repos use QueryRegistry for complex queries, auth repos use `db.ExecContext` directly for simple CRUD)
- `aethel-core/internal/database/repos/session_repo.go` — SessionRepo including `RotateSession` transaction
- `aethel-core/internal/database/repos/password_reset_repo.go` — PasswordResetRepo with all 3 methods
- `aethel-core/internal/api/middleware/csrf.go` — CSRFProtect with `ConstantTimeCompare`
- `aethel-core/internal/api/middleware/security_headers.go` — All required headers
- `aethel-core/internal/api/middleware/rate_limiter.go` — All three rate limiters
- `aethel-core/internal/api/handlers/auth.go` — Login (sets both cookies), Refresh (rotation), Logout (clears cookies), PasswordReset handlers
- `aethel-core/internal/api/server.go` — Middleware stack in correct order, all routes registered
- `aethel-core/internal/rbac/middleware.go` — Require(), SetUserContext(), role permission map
- `aethel-core/cmd/aethel/main.go` — Startup sequence, UserRepo/SessionRepo/PasswordResetRepo wired, noopAuditRepo still in place
- `aethel-view/app/composables/useAuth.ts` — Full implementation with in-memory token, CSRF, initAuth
- `aethel-view/app/plugins/auth.client.ts` — Silent recovery plugin
- `aethel-view/app/plugins/fetch.ts` — $fetch interceptor with 401 retry
- `aethel-view/app/middleware/auth.global.ts` — Global route guard
- `aethel-view/app/middleware/role.ts` — Role hierarchy guard
- `aethel-view/app/pages/auth/login.vue` — Already wired to `useAuth()`, no mock data
- All admin pages — Already have `definePageMeta({ requiredRole: 'ADMIN' })`
- `aethel-scripts/db-harden.sh` — Script exists; calls a SQL file

**What is genuinely missing — this is what you must build:**
- `aethel-core/internal/database/repos/audit_repo.go` — AuditRepo implementing `domain.AuditRepository` (Write, Query, VerifyChain)
- Queries in `queries.yaml` for the audit repo (governance section needs expansion) and auth layer (auth/session/pw_reset query groups)
- `aethel-core/internal/service/auth_service_test.go` — Unit tests (7 test cases using stdlib mock repos)
- `aethel-core/internal/integration/auth_test.go` — Integration tests for auth flow against real PostgreSQL
- Wire `AuditRepo` in `main.go` (replace `noopAuditRepo`)
- `aethel-core/scripts/db-harden.sql` — SQL file that `db-harden.sh` references (the shell script exists but the SQL file is missing from the core package directory)
- `aethel-view/app/components/layout/WorkspaceSidebar.vue` — Remove dual `currentUser`/`authUser` pattern: nav visibility must use `authUser` (real JWT role) consistently
- `aethel-view/app/components/layout/WorkspaceNavbar.vue` — Verify logout is wired; add "Demo mode" disclaimer badge next to role switcher

---

## Block A — Complete Sprint 1 (Backend)

### Step A-0: Establish Clean Build Baseline

Run from `aethel-core/`:

```bash
go build ./...
go vet ./...
```

Both must exit 0. If either fails, fix the issue before proceeding. Output the result before continuing.

---

### Step A-1: Expand `queries.yaml` with Auth and Audit Queries

Open `aethel-core/internal/database/queries/queries.yaml`.

The file currently has `dispatch`, `dispatch_event`, `routing_rule`, `workflow`, and `governance` query groups. You must **add** the following groups. Do not modify or remove any existing entries.

Add after the `workflow` section and before `governance`:

```yaml
  # ══════════════════════════════════════════════════════════════════════════
  # AUTH — User lookup and session management
  # ══════════════════════════════════════════════════════════════════════════
  auth:
    get_user_by_email:
      statement: |
        SELECT id, organization_id, department_id, email_address, full_name, job_title,
               role, is_active, password_hash, failed_login_attempts, locked_until, last_login_at,
               created_at, updated_at
        FROM users
        WHERE organization_id = $1 AND email_address = $2 AND is_active = true
      timeout_ms: 3000
      required_permission: "public"

    get_user_by_id:
      statement: |
        SELECT id, organization_id, department_id, email_address, full_name, job_title,
               role, is_active, password_hash, failed_login_attempts, locked_until, last_login_at,
               created_at, updated_at
        FROM users
        WHERE organization_id = $1 AND id = $2
      timeout_ms: 3000
      required_permission: "public"

    list_users:
      statement: |
        SELECT id, organization_id, department_id, email_address, full_name, job_title,
               role, is_active, password_hash, failed_login_attempts, locked_until, last_login_at,
               created_at, updated_at
        FROM users
        WHERE organization_id = $1
        ORDER BY full_name ASC
        LIMIT $2 OFFSET $3
      timeout_ms: 5000
      required_permission: "admin.access"

    create_user:
      statement: |
        INSERT INTO users (id, organization_id, department_id, email_address, full_name, job_title,
                           role, is_active, password_hash, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now(), now())
      timeout_ms: 3000
      required_permission: "admin.access"

    update_user:
      statement: |
        UPDATE users
        SET full_name = $3, job_title = $4, role = $5, is_active = $6,
            department_id = $7, updated_at = now()
        WHERE organization_id = $1 AND id = $2
      timeout_ms: 3000
      required_permission: "admin.access"

    update_password_hash:
      statement: |
        UPDATE users SET password_hash = $2, updated_at = now()
        WHERE id = $1
      timeout_ms: 3000
      required_permission: "public"

    increment_failed_logins:
      statement: |
        UPDATE users SET failed_login_attempts = failed_login_attempts + 1, updated_at = now()
        WHERE id = $1
      timeout_ms: 2000
      required_permission: "public"

    reset_failed_logins:
      statement: |
        UPDATE users SET failed_login_attempts = 0, locked_until = NULL, updated_at = now()
        WHERE id = $1
      timeout_ms: 2000
      required_permission: "public"

    lock_until:
      statement: |
        UPDATE users SET locked_until = $2, updated_at = now()
        WHERE id = $1
      timeout_ms: 2000
      required_permission: "public"

    set_last_login:
      statement: |
        UPDATE users SET last_login_at = now(), updated_at = now()
        WHERE id = $1
      timeout_ms: 2000
      required_permission: "public"

  # ══════════════════════════════════════════════════════════════════════════
  # SESSION — Refresh token session management
  # ══════════════════════════════════════════════════════════════════════════
  session:
    create_session:
      statement: |
        INSERT INTO user_sessions (id, user_id, session_token_hash, expires_at,
                                   client_ip_address, user_agent, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, now())
      timeout_ms: 2000
      required_permission: "public"

    get_session_by_token_hash:
      statement: |
        SELECT id, user_id, session_token_hash, expires_at,
               client_ip_address, user_agent, created_at
        FROM user_sessions
        WHERE session_token_hash = $1 AND expires_at > now()
      timeout_ms: 2000
      required_permission: "public"

    delete_session_by_id:
      statement: |
        DELETE FROM user_sessions WHERE id = $1
      timeout_ms: 2000
      required_permission: "public"

    delete_sessions_by_user:
      statement: |
        DELETE FROM user_sessions WHERE user_id = $1
      timeout_ms: 3000
      required_permission: "public"

  # ══════════════════════════════════════════════════════════════════════════
  # PASSWORD RESET
  # ══════════════════════════════════════════════════════════════════════════
  pw_reset:
    create_pw_reset_token:
      statement: |
        INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, created_at)
        VALUES ($1, $2, $3, $4, now())
      timeout_ms: 2000
      required_permission: "public"

    get_pw_reset_token_by_hash:
      statement: |
        SELECT id, user_id, token_hash, expires_at, used_at, created_at
        FROM password_reset_tokens
        WHERE token_hash = $1 AND used_at IS NULL AND expires_at > now()
      timeout_ms: 2000
      required_permission: "public"

    mark_pw_reset_token_used:
      statement: |
        UPDATE password_reset_tokens SET used_at = now() WHERE id = $1
      timeout_ms: 2000
      required_permission: "public"
```

Also **replace** the existing `governance` section entirely (it currently has only `query_audit_ledger_range` which is insufficient):

```yaml
  # ══════════════════════════════════════════════════════════════════════════
  # PILLAR 3 — RBAC & IMMUTABLE AUDIT LEDGER
  # ══════════════════════════════════════════════════════════════════════════
  governance:
    write_audit_event:
      statement: |
        INSERT INTO audit_ledger (
          organization_id, actor_user_id, action_event_type,
          target_resource_id, target_table, ip_address, user_agent,
          metadata, previous_checksum, checksum
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
      timeout_ms: 3000
      required_permission: "public"

    get_latest_checksum:
      statement: |
        SELECT checksum FROM audit_ledger
        WHERE organization_id = $1
        ORDER BY id DESC
        LIMIT 1
      timeout_ms: 2000
      required_permission: "public"

    query_audit_ledger_range:
      statement: |
        SELECT id, organization_id, actor_user_id, action_event_type,
               target_resource_id, target_table, ip_address, user_agent,
               metadata, previous_checksum, checksum, created_at
        FROM audit_ledger
        WHERE organization_id = $1
          AND created_at >= $2 AND created_at < $3
        ORDER BY created_at DESC
        LIMIT $4 OFFSET $5
      timeout_ms: 10000
      required_permission: "admin.audit"

    verify_chain_range:
      statement: |
        SELECT id, action_event_type, target_resource_id, actor_user_id,
               previous_checksum, checksum, created_at
        FROM audit_ledger
        WHERE organization_id = $1
          AND created_at >= $2 AND created_at < $3
        ORDER BY id ASC
      timeout_ms: 30000
      required_permission: "admin.audit"
```

**Important column name alignment:** The existing `user_repo.go` and `session_repo.go` use the actual PostgreSQL column names from the migration files. These are: `email_address` (not `email`), `session_token_hash` (not `token_hash`), `client_ip_address` (not `ip_address`), `action_event_type` (not `event_type`). Verify these match the migration SQL files in `aethel-core/internal/database/migrations/` before finalizing queries.yaml. To check: `grep -l "email_address\|session_token_hash" aethel-core/internal/database/migrations/*.sql`

---

### Step A-2: Create `audit_repo.go`

Create `aethel-core/internal/database/repos/audit_repo.go`.

This is the only repository still using a noop. The `user_repo.go`, `session_repo.go`, and `password_reset_repo.go` already exist as real implementations.

**Critical design note:** The existing repos (`user_repo.go`, `session_repo.go`, `password_reset_repo.go`) use inline SQL via `db.ExecContext` and `db.QueryRowContext` directly — they do not use the QueryRegistry. This was an acceptable early implementation. For the audit repo, use the **QueryRegistry pattern** to match the dispatch repos, since audit writes are more critical and the query is complex.

```go
package repos

import (
    "context"
    "crypto/sha256"
    "database/sql"
    "fmt"
    "strings"
    "time"

    "github.com/google/uuid"

    "aethel-core/internal/database"
    "aethel-core/internal/domain"
)

// AuditRepo implements domain.AuditRepository.
// Writes to the audit_ledger partitioned table.
// organization_id is stored as a plain uuid (no FK) so records survive org deletion — by design.
// The checksum chain uses SHA-256(action_event_type + target_resource_id + actor_user_id + previous_checksum).
type AuditRepo struct {
    db *sql.DB
    q  *database.QueryRegistry
}

func NewAuditRepo(db *sql.DB, q *database.QueryRegistry) *AuditRepo {
    return &AuditRepo{db: db, q: q}
}
```

Implement `Write(ctx, *domain.AuditEntry) error`:
1. Call `governance.get_latest_checksum` with `orgID` to fetch the previous checksum. If the query returns `sql.ErrNoRows`, use `""` as the previous checksum (first entry in chain).
2. Compute the current checksum: `sha256.Sum256([]byte(string(entry.ActionEventType) + targetIDStr + actorIDStr + previousChecksum))` where `targetIDStr` is `entry.TargetResourceID.String()` if non-nil, else `""`, and `actorIDStr` is `entry.ActorUserID.String()` if non-nil, else `""`. Encode as lowercase hex.
3. Set `entry.PreviousChecksum = previousChecksum` and `entry.Checksum = hex.EncodeToString(computed[:])`.
4. Execute `governance.write_audit_event` with all 10 parameters in this exact order: `orgID`, `actorUserID`, `actionEventType`, `targetResourceID`, `targetTable`, `ipAddress`, `userAgent`, `metadata`, `previousChecksum`, `checksum`.

The 10 parameters must use `sql.NullString` / `uuid.NullUUID` for nullable fields (`actor_user_id`, `target_resource_id`, `target_table`, `ip_address`, `user_agent`, `metadata`).

Implement `Query(ctx, orgID, from, to time.Time, page domain.Page) ([]domain.AuditEntry, error)`:
- Execute `governance.query_audit_ledger_range` with `(orgID, from, to, page.Limit, page.Offset)`.
- Scan all 12 columns. Nullable columns use `sql.NullString` / `uuid.NullUUID`.

Implement `VerifyChain(ctx, orgID, from, to time.Time) (*domain.ChainVerificationResult, error)`:
- Execute `governance.verify_chain_range` with `(orgID, from, to)`. Fetch all rows ordered by `id ASC`.
- For each row, re-compute the expected checksum using the same formula as `Write`. Compare to `row.Checksum`.
- If mismatch: append to `BrokenAt` slice as `domain.BrokenLink{EntryID: row.ID, ComputedChecksum: computed, StoredChecksum: row.Checksum, CreatedAt: row.CreatedAt}`.
- Return `&domain.ChainVerificationResult{Valid: len(brokenAt) == 0, TotalRows: count, BrokenAt: brokenAt}`.

```go
// computeChecksum replicates the formula used in Write.
// Both Write and VerifyChain must use identical logic or the chain will always appear broken.
func computeChecksum(eventType domain.AuditEventType, targetID *uuid.UUID, actorID *uuid.UUID, previousChecksum string) string {
    targetStr := ""
    if targetID != nil {
        targetStr = targetID.String()
    }
    actorStr := ""
    if actorID != nil {
        actorStr = actorID.String()
    }
    h := sha256.Sum256([]byte(strings.Join([]string{
        string(eventType), targetStr, actorStr, previousChecksum,
    }, ":")))
    return fmt.Sprintf("%x", h)
}
```

Use this helper in both `Write` and `VerifyChain` to guarantee they produce identical checksums.

---

### Step A-3: Wire `AuditRepo` in `main.go`

Open `aethel-core/cmd/aethel/main.go`.

In the `runServe` function, find the auth pillar repo declarations (around line 142–148):

```go
// Auth pillar — real implementations wired in Sprint 1.5.
var (
    userRepo    domain.UserRepository          = repos.NewUserRepo(db)
    sessionRepo domain.SessionRepository       = repos.NewSessionRepo(db)
    pwResetRepo domain.PasswordResetRepository = repos.NewPasswordResetRepo(db)
    auditRepo   domain.AuditRepository         = &noopAuditRepo{}
)
```

Replace `&noopAuditRepo{}` with `repos.NewAuditRepo(db, queries)`:

```go
var (
    userRepo    domain.UserRepository          = repos.NewUserRepo(db)
    sessionRepo domain.SessionRepository       = repos.NewSessionRepo(db)
    pwResetRepo domain.PasswordResetRepository = repos.NewPasswordResetRepo(db)
    auditRepo   domain.AuditRepository         = repos.NewAuditRepo(db, queries)
)
```

Delete the `noopAuditRepo` struct and its three method implementations from the bottom of `main.go` (the type definition and the three func receivers). Leave `noopMSRepo`, `noopGNRepo`, `noopDocTypeRepo`, and `noopEscRepo` intact — those are Sprint 3–4 work.

---

### Step A-4: Create `aethel-core/scripts/db-harden.sql`

The file `aethel-scripts/db-harden.sh` already exists and references a SQL file. That shell script calls:
```bash
psql "${DATABASE_SUPERUSER_DSN}" -f "${SCRIPT_DIR}/../aethel-core/scripts/db-harden.sql"
```

Create the missing SQL file at `aethel-core/scripts/db-harden.sql`:

```sql
-- Aethel Workspace — Database Hardening Script
-- Run once as the database superuser after migrations are applied.
-- This REVOKES destructive privileges from the application user on the audit_ledger,
-- making it append-only from the application's perspective.
--
-- Replace 'aethel_app' with the actual AETHEL_DB_USER value from your .env file.
-- The application connects as this role. After running this script, even a compromised
-- application instance cannot delete or modify audit ledger entries.
--
-- Usage: psql "${DATABASE_SUPERUSER_DSN}" -f aethel-core/scripts/db-harden.sql
-- Or via: ./aethel-scripts/db-harden.sh

-- Step 1: Revoke UPDATE and DELETE on the audit_ledger from the application user.
-- INSERT is required (the app appends rows). SELECT is required (admin queries).
REVOKE UPDATE, DELETE ON TABLE audit_ledger FROM aethel_app;

-- Step 2: Verify. The output should show no 'w' (UPDATE) or 'd' (DELETE) privilege
-- for aethel_app in the audit_ledger row.
\dp audit_ledger

-- Step 3: Apply the same restriction to future partitions automatically.
-- PostgreSQL partition tables inherit privileges from the parent by default,
-- but explicit revocation on the parent does not propagate. Document this
-- as a required step whenever new audit_ledger partitions are created.
-- See migration 19 — partitions are pre-created; new ones need manual hardening.

SELECT 'Database hardening complete. Verify audit_ledger privileges above.' AS status;
```

---

### Step A-5: Verify Secrets Are Never Logged

Open `aethel-core/cmd/aethel/main.go`. Confirm these exact rules are followed:

- Line reading `AETHEL_JWT_SECRET`: only checks if empty, logs `"AETHEL_JWT_SECRET not set"` or `"JWT algorithm: HS256"` — never logs the value.
- No `fmt.Sprintf`, `slog.Info`, `log.Printf`, or `zerolog` call anywhere in `internal/` or `cmd/` passes the value of `AETHEL_JWT_SECRET`, `AETHEL_DB_PASSWORD`, or `AETHEL_DB_DSN` as a log field.

Run this grep and confirm the output contains zero suspicious lines:

```bash
# From aethel-workspace/
grep -rn 'JWT_SECRET\|DB_PASSWORD\|DB_DSN' aethel-core/ --include="*.go" \
  | grep -vE 'os\.Getenv|os\.LookupEnv|not set|algorithm|// '
# Expected: zero lines
```

If any line appears that is not an `os.Getenv`/`os.LookupEnv` call or a comment, fix it before proceeding.

---

### Step A-6: Write Unit Tests for `auth_service.go`

Create `aethel-core/internal/service/auth_service_test.go`.

Use only the Go standard library — no `testify`, `gomock`, or any external test framework.

The test file must define minimal inline mock implementations of the four repository interfaces. Each mock implements only the methods exercised by the test cases it supports — other methods may panic or return nil.

**Required test cases** (all 7 must pass):

```go
// TestLogin_HappyPath: valid email + correct Argon2id hash → returns LoginResult with non-empty tokens
// TestLogin_WrongPassword: correct email + wrong password → ErrUnauthorized; IncrementFailedLogins called
// TestLogin_AccountLocked: user.LockedUntil set to future → ErrAccountLocked before Argon2id runs
// TestLogin_UserNotFound: GetByEmail returns ErrNotFound → service returns ErrUnauthorized (not ErrNotFound)
// TestRefreshSession_ValidToken: valid token hash in session → returns new access token
// TestRefreshSession_ExpiredToken: GetByTokenHash returns ErrNotFound → ErrUnauthorized
// TestLogout_RevokesSession: calls GetByTokenHash then DeleteByID with correct session ID
```

Important implementation notes for the test file:

1. To test `TestLogin_HappyPath`, you need a pre-computed Argon2id hash. Do not call `hashPassword` directly (it is private). Instead, call it via a helper in the same package, or compute an Argon2id hash inline in the test setup using the same algorithm. The simplest approach: call `argon2.IDKey` with the same parameters (`m=65536, t=3, p=4`) and format it as PHC string `$argon2id$v=19$m=65536,t=3,p=4$<b64salt>$<b64hash>`.

2. `TestLogin_AccountLocked`: the `user.LockedUntil` field must be a pointer to a future time. Argon2id must NOT be called. To verify Argon2id is skipped, your mock `UserRepository.GetByEmail` should return a user with `LockedUntil = ptr(time.Now().Add(1 * time.Hour))` — and the test simply checks the returned error is `ErrAccountLocked`.

3. `TestRefreshSession_ValidToken`: the service calls `RotateSession`, not `DeleteByID` + `Create`. Your mock `SessionRepository` must implement `RotateSession(ctx, oldID, newSession) error`.

4. The mock `AuditRepository.Write` should simply return nil (audit failures are silent best-effort in the service).

5. JWT verification: access tokens returned by the service can be decoded with `jwt.Parse` using the dev secret `"dev-secret-change-in-production"` to verify claims. Alternatively, just check that `len(result.AccessToken) > 0` and `len(result.RefreshToken) > 0` — the structure is already tested by the compiler.

---

### Step A-7: Write Integration Tests for Auth Flow

Create `aethel-core/internal/integration/auth_test.go`.

The existing `dispatch_test.go` in that package shows the exact pattern to follow: `//go:build integration` tag, `AETHEL_DB_DSN` env var skip, `openTestDB()` and `buildRegistry()` helpers.

Required integration test cases (these hit a real PostgreSQL 16 instance):

```go
//go:build integration

// Run with: go test ./internal/integration/... -tags integration -v
// Requires: AETHEL_DB_DSN pointing to a test PostgreSQL 16 instance with migrations applied.
// Requires: at least one row in organizations, and a test user in the users table.
```

**Test functions to implement:**

`TestAuthFlow_LoginRefreshLogout`  
1. Create a test user (insert into `users` table using `db.ExecContext` directly with a known Argon2id hash of password "testpassword123")
2. Call `authSvc.Login(ctx, app.OrgID, email, "testpassword123", "127.0.0.1", "test-agent")`
3. Assert: `result.AccessToken` is non-empty, `result.RefreshToken` is non-empty
4. Call `authSvc.RefreshSession(ctx, result.RefreshToken)` → assert new access token
5. Call `authSvc.RefreshSession(ctx, result.RefreshToken)` a second time using the OLD token → assert `ErrUnauthorized` (rotation ensures the old token is gone)
6. Call `authSvc.Logout(ctx, app.OrgID, userID, newRefreshToken, "127.0.0.1", "test-agent")` → assert nil error
7. Clean up: delete the test user

`TestAuthRepo_AuditChain`  
1. Write 3 audit entries using `auditRepo.Write()`
2. Call `auditRepo.VerifyChain()` for the time range covering those entries
3. Assert `result.Valid == true` and `len(result.BrokenAt) == 0`

**Setup in the test:** Use the same `openTestDB` and `buildRegistry` helpers from `dispatch_test.go`. Wire repos as:
```go
userRepo    := repos.NewUserRepo(db)
sessionRepo := repos.NewSessionRepo(db)
pwResetRepo := repos.NewPasswordResetRepo(db)
auditRepo   := repos.NewAuditRepo(db, reg)
authSvc     := service.NewAuthService(userRepo, sessionRepo, pwResetRepo, auditRepo)
```

---

### Block A Verification Gate

Run all commands in order. All must succeed before starting Block B.

```bash
# From aethel-workspace/
cd aethel-core

# Gate 1: Clean build
go build ./...

# Gate 2: Vet
go vet ./...

# Gate 3: Unit tests (7 must pass, no integration tag)
go test ./internal/service/... -v -count=1

# Gate 4: Race detector on all non-integration tests
go test ./... -race -count=1 -tags=""

# Gate 5: Secrets grep — must return 0 lines
grep -rn 'JWT_SECRET\|DB_PASSWORD\|DB_DSN' internal/ cmd/ --include="*.go" \
  | grep -vE 'os\.Getenv|os\.LookupEnv|"not set"|algorithm|//' \
  | wc -l
# Expected output: 0

# Gate 6: No org claim in JWT — must return 0 lines
grep -rn '"org"' internal/service/ internal/api/ --include="*.go" \
  | grep -vE '//|test'
# Expected: 0

# Gate 7: CSRF uses ConstantTimeCompare — must return ≥1 line
grep -rn "ConstantTimeCompare" internal/ --include="*.go"
# Expected: at least 1 match in csrf.go

# Gate 8: main.go compiles and binary is buildable
go build -o /dev/null ./cmd/aethel
```

Do not proceed to Block B until all 8 gates pass. Report the output of each gate before continuing.

---

## Block B — Sprint 1.5 Frontend Auth Wiring

### Context

Most of Block B (Sprint 1.5) is already implemented. The frontend auth system, security middleware, cookie handling, and route guards are all done. What remains is targeted cleanup and consistency fixes in the sidebar component and verification that everything is wired end-to-end correctly.

---

### Step B-1: Fix `WorkspaceSidebar.vue` Role Source

Open `aethel-view/app/components/layout/WorkspaceSidebar.vue`.

**The problem:** The file imports both `useMockData()` (for `currentUser`) and `useAuth()` (for `authUser`). Nav group visibility currently uses `authUser.value?.role ?? currentUser.value.role` — this dual-source is fragile. The mock role should only affect display names and avatar images, never security-relevant nav visibility.

**What to fix:**

Find the nav group filtering logic (look for where `role` is used to decide which nav groups to show). It should look something like:

```typescript
const role = authUser.value?.role ?? currentUser.value.role
const visibleGroups = navGroups.value.filter(g => g.roles.includes(role))
```

Change it so nav visibility uses **only** `authUser.value?.role`. If `authUser.value` is null (unauthenticated), show no nav groups:

```typescript
// Nav visibility is gated on the real JWT role, not the prototype mock.
const visibleGroups = computed<NavGroup[]>(() => {
  const role = authUser.value?.role
  if (!role) return []
  return (config.value.nav.length > 0 ? config.value.nav : hardcodedNavGroups.value)
    .filter(g => g.roles.includes(role))
})
```

The `currentUser` from `useMockData` may remain in use for avatar image URLs and display names (prototype demo feature — acceptable). But it must not influence which nav groups are visible to the authenticated user.

---

### Step B-2: Verify Logout Wiring in `WorkspaceNavbar.vue`

Open `aethel-view/app/components/layout/WorkspaceNavbar.vue`.

Confirm the logout action calls `useAuth().logout()` and then navigates to `/auth/login`. Based on the current file, `logout` is already imported from `useAuth()` and wired. Verify the logout handler does this:

```typescript
async function handleLogout() {
  await logout()
  await navigateTo('/auth/login', { replace: true })
}
```

If the current implementation only calls `logout()` without `navigateTo`, add the navigation. If it already does both, no change is needed.

Add a "Demo mode" disclaimer badge visually adjacent to the role switcher buttons. Find the role switcher UI and add:

```html
<UBadge color="warning" variant="soft" size="xs" class="ml-2">Demo only</UBadge>
```

This makes it visually clear to testers that the role switcher is prototype tooling and does not affect actual security.

---

### Step B-3: TypeScript Typecheck

From `aethel-view/`:

```bash
pnpm exec nuxi typecheck
```

This must return zero errors. If there are errors introduced by Block A or B changes, fix them before proceeding to verification.

---

### Block B Verification Gate

Run all checks. All must pass before reporting the task complete.

#### Backend checks (from `aethel-core/`):

```bash
# B-Gate 1: Clean build still passes after all changes
go build ./...

# B-Gate 2: Unit tests still pass
go test ./internal/service/... -v -count=1

# B-Gate 3: Race detector still clean
go test ./... -race -count=1

# B-Gate 4: Server starts and health endpoint responds
# Note: This requires AETHEL_DB_DSN to be set OR will fail at DB connect.
# If no DB is available, skip this gate and note it in your report.
go run ./cmd/aethel serve &
SERVER_PID=$!
sleep 3
curl -sf http://localhost:8080/healthz | grep -q "ok" && echo "HEALTH: OK"
curl -sI http://localhost:8080/healthz | grep -i "x-content-type-options" | grep -q "nosniff" && echo "HEADERS: OK"
# Login endpoint rejects wrong credentials with 401 (not 404, not 200):
STATUS=$(curl -sw "%{http_code}" -o /dev/null -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"nobody@test.invalid","password":"wrong"}')
echo "Wrong credentials status: $STATUS"   # Expected: 401
# Login endpoint rejects empty body with 400:
STATUS=$(curl -sw "%{http_code}" -o /dev/null -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{}')
echo "Empty body status: $STATUS"   # Expected: 400
kill $SERVER_PID 2>/dev/null || true
```

#### Frontend checks (from `aethel-view/`):

```bash
# B-Gate 5: TypeScript — zero errors
pnpm exec nuxi typecheck

# B-Gate 6: No token in localStorage — must return 0 matches
grep -rn "localStorage\|sessionStorage" app/ --include="*.ts" --include="*.vue"
# Expected: 0 matches

# B-Gate 7: useMockData is NOT imported in login.vue — must return 0 matches
grep -rn "useMockData" app/pages/auth/login.vue
# Expected: 0 matches

# B-Gate 8: All auth API calls go through useAuth.ts — must return 0 matches elsewhere
grep -rn "auth/login\|auth/refresh\|auth/logout\|auth/password" \
  app/ --include="*.ts" --include="*.vue" \
  | grep -vE "useAuth\.ts|middleware/auth"
# Expected: 0 matches (auth calls are in useAuth.ts only)

# B-Gate 9: Admin pages all have requiredRole ADMIN — verify count
grep -rn "requiredRole.*ADMIN" app/pages/admin/ --include="*.vue" | wc -l
# Expected: 9 (one per admin page)
```

---

## Definition of Done

Tick off every item before reporting the task complete.

**Block A — Sprint 1 Backend:**

1. `go build ./...` exits 0
2. `go vet ./...` exits 0
3. `queries.yaml` has `auth`, `session`, `pw_reset`, and expanded `governance` query groups with correct column names matching migration SQL
4. `audit_repo.go` created with `Write`, `Query`, `VerifyChain` — checksum formula uses `computeChecksum()` helper shared between Write and VerifyChain
5. `main.go` wires `repos.NewAuditRepo(db, queries)` — `noopAuditRepo` struct deleted
6. `aethel-core/scripts/db-harden.sql` created with `REVOKE UPDATE, DELETE ON audit_ledger FROM aethel_app`
7. Secrets grep returns 0 suspicious lines
8. JWT org claim grep returns 0 matches
9. `auth_service_test.go` exists with 7 unit tests, all passing
10. `integration/auth_test.go` exists with `TestAuthFlow_LoginRefreshLogout` and `TestAuthRepo_AuditChain`, tagged `//go:build integration`
11. Race detector passes on all non-integration tests

**Block B — Sprint 1.5 Frontend:**

12. `WorkspaceSidebar.vue` nav visibility uses `authUser.value?.role` only (not mock data role)
13. `WorkspaceNavbar.vue` logout navigates to `/auth/login`; "Demo only" badge visible near role switcher
14. `pnpm exec nuxi typecheck` returns zero errors
15. `localStorage`/`sessionStorage` grep returns 0 matches
16. `useMockData` grep in `login.vue` returns 0 matches
17. Auth API calls grep returns 0 matches outside `useAuth.ts`
18. All 9 admin pages have `requiredRole: 'ADMIN'`

---

## Do NOT Do These Things

- **Do not** modify any file in `internal/domain/` — interfaces are final
- **Do not** modify any migration SQL file in `internal/database/migrations/`
- **Do not** add `testify`, `gomock`, or any external test library — stdlib only for unit tests
- **Do not** add an `org` claim to the JWT — the single-tenant model makes it meaningless
- **Do not** store tokens in `localStorage`, `sessionStorage`, or any readable cookie
- **Do not** store the refresh token in the response body or in JavaScript state
- **Do not** use regular `==` for CSRF token comparison — always use `crypto/subtle.ConstantTimeCompare`
- **Do not** implement dispatch, routing rule, minute sheet, green note, or escalation rule repos — those are Sprint 2–3 work; the existing dispatch repos already exist and the others have noops
- **Do not** return different error messages for wrong email vs wrong password — always the same 401
- **Do not** run frontend commands from `aethel-core/` or backend commands from `aethel-view/`
- **Do not** commit — the user will review and commit manually
- **Do not** touch `nuxt.config.ts` or `app.config.ts` without reading `aethel-view/.claude-devtools/settings.json` first (autoConfirm is currently DISABLED)
- **Do not** start Block B until all 8 Block A gates pass

---

## Reference Files

Read these before writing any code. They are the ground truth for every decision.

```
# Architecture
CLAUDE.md                                               ← Single-tenant model, JWT claims, middleware stack
docs/architecture/architecture-security.md              ← Security design decisions
docs/architecture/architecture-server.md                ← Middleware stack order
docs/architecture/architecture-api-routes.md            ← API route definitions

# Existing backend files (ALREADY CORRECT — read to understand, do not rewrite)
aethel-core/internal/service/auth_service.go            ← Login, RefreshSession, Logout
aethel-core/internal/database/repos/user_repo.go        ← Pattern for row scanning (scanUser helper)
aethel-core/internal/database/repos/session_repo.go     ← RotateSession transaction pattern
aethel-core/internal/database/repos/dispatch_repo.go    ← QueryRegistry usage pattern
aethel-core/internal/database/query_registry.go         ← registry.Get("group.name").Stmt
aethel-core/internal/domain/user.go                     ← UserRepository, SessionRepository interfaces
aethel-core/internal/domain/governance.go               ← AuditRepository interface, AuditEntry struct
aethel-core/internal/api/handlers/auth.go               ← Cookie handling (already correct)
aethel-core/internal/api/server.go                      ← Middleware stack (already correct)
aethel-core/internal/integration/dispatch_test.go       ← Integration test pattern to follow

# Database schema (check column names before writing queries)
aethel-core/internal/database/migrations/               ← List and read migration 04 (users table), 04 (user_sessions), 18 (audit_ledger)

# Frontend (ALREADY CORRECT — read to understand, verify, make targeted fixes only)
aethel-view/app/composables/useAuth.ts                  ← Complete auth composable
aethel-view/app/plugins/auth.client.ts                  ← Silent recovery plugin
aethel-view/app/plugins/fetch.ts                        ← $fetch interceptor
aethel-view/app/middleware/auth.global.ts               ← Global route guard
aethel-view/app/middleware/role.ts                      ← Role hierarchy guard
aethel-view/app/pages/auth/login.vue                    ← Already wired to useAuth
aethel-view/app/components/layout/WorkspaceSidebar.vue  ← Needs nav visibility fix
aethel-view/app/components/layout/WorkspaceNavbar.vue   ← Verify logout + add Demo badge
```

---

## Report Format

When the task is complete, report back with:

1. Output of `go test ./internal/service/... -v` (all 7 unit tests)
2. Output of all 8 Block A gates (paste each command and its output)
3. Output of all 5 Block B frontend grep checks (paste each command and its output)
4. Output of `pnpm exec nuxi typecheck` (zero errors)
5. Complete list of files created or modified with one-line description each
6. Any deviation from this plan with a written justification
