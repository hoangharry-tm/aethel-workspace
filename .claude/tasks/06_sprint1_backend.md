# Task 06 — Sprint 1: Real Repository Implementations & Auth Wiring

**Working directory:** `aethel-workspace/` (repo root)  
**Primary target:** `aethel-core/`  
**Sprint:** 1 of 6 (see `docs/plans/agile-implementation-plan.md`)

---

## Your Role

You are a senior Go backend engineer completing Sprint 1 of the Aethel Workspace backend. The architecture, domain types, services, handlers, and middleware are already written. Your job is to replace the noop repository stubs with real PostgreSQL implementations for the auth layer, create the queries YAML, write the seed loader, wire it all together, and write unit tests. Nothing more.

---

## Step 0 — Load Context Before Writing Anything

Read these files in full before touching any code. Do not skip this step.

```
aethel-core/cmd/aethel/main.go                          ← noop stubs at lines 119–131 and 254–391
aethel-core/internal/domain/user.go                     ← UserRepository, SessionRepository, PasswordResetRepository interfaces
aethel-core/internal/domain/governance.go               ← AuditRepository interface
aethel-core/internal/service/auth_service.go            ← what the repos must satisfy at runtime
aethel-core/internal/database/query_registry.go         ← how named queries are looked up
aethel-core/internal/database/blueprint_context.go      ← T(), E() for migrations (read-only reference)
aethel-core/internal/database/migrations/               ← list files; understand the schema (users, user_sessions, password_reset_tokens, branding_configs, system_settings)
aethel-core/internal/database/connect.go               ← how *sql.DB is created
aethel-core/internal/blueprint/loader.go               ← LoadDatabaseConfig, LoadQueriesConfig
aethel-core/internal/blueprint/queries_config.go       ← QueriesConfig struct shape
blueprints/server-database.yaml                        ← schema name and table aliases
blueprints/ui-theme.yaml                               ← branding seed fields
blueprints/ui-layouts.yaml                             ← nav_config seed structure
docs/guides/go-developer-guide.md                      ← sections 4 (migrations), 5 (queries), 6 (startup), 8 (SQL placement rules)
docs/plans/agile-implementation-plan.md                ← Sprint 1 Definition of Done
```

Then, **before writing any code**, output a brief numbered list of exactly what files you will create or modify. This is your commit to the user. Do not deviate from it.

---

## Step 1 — Verify Sprint 0 State

Run these commands to confirm the foundation is intact:

```bash
cd aethel-core
go build ./...
go vet ./...
```

Both must exit 0. If either fails, fix the compilation error before proceeding. Do not continue to Step 2 until this is clean.

---

## Step 2 — Create `internal/database/queries/queries.yaml`

Create the file at `aethel-core/internal/database/queries/queries.yaml`.

This file must define all named SQL queries required by the auth layer. Follow the schema established in `aethel-core/internal/blueprint/queries_config.go` for the YAML structure.

**Required query keys and their SQL** (key format is `group.name` → looked up as `registry.Get("group.name")`):

```yaml
global_query_defaults:
  timeout_ms: 5000
  max_rows_per_page: 50

queries:
  auth:
    get_user_by_email:
      statement: |
        SELECT id, organization_id, email, password_hash, role, department_id,
               full_name, is_active, failed_login_attempts, locked_until, last_login_at,
               created_at, updated_at
        FROM users
        WHERE organization_id = $1 AND email = $2 AND is_active = true
      timeout_ms: 3000
      required_permission: "public"

    get_user_by_id:
      statement: |
        SELECT id, organization_id, email, password_hash, role, department_id,
               full_name, is_active, failed_login_attempts, locked_until, last_login_at,
               created_at, updated_at
        FROM users
        WHERE organization_id = $1 AND id = $2
      timeout_ms: 3000
      required_permission: "public"

    create_user:
      statement: |
        INSERT INTO users (id, organization_id, email, password_hash, role, department_id,
                           full_name, is_active, failed_login_attempts)
        VALUES ($1, $2, $3, $4, $5, $6, $7, true, 0)
      timeout_ms: 3000
      required_permission: "admin.access"

    update_user:
      statement: |
        UPDATE users SET full_name = $3, role = $4, department_id = $5,
               is_active = $6, updated_at = now()
        WHERE organization_id = $1 AND id = $2
      timeout_ms: 3000
      required_permission: "admin.access"

    update_password_hash:
      statement: |
        UPDATE users SET password_hash = $3, updated_at = now()
        WHERE organization_id = $1 AND id = $2
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
        UPDATE users SET failed_login_attempts = 0, updated_at = now()
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

    list_users:
      statement: |
        SELECT id, organization_id, email, role, department_id, full_name,
               is_active, created_at, updated_at
        FROM users
        WHERE organization_id = $1
        ORDER BY full_name ASC
        LIMIT $2 OFFSET $3
      timeout_ms: 5000
      required_permission: "admin.access"

  session:
    create_session:
      statement: |
        INSERT INTO user_sessions (id, organization_id, user_id, token_hash,
                                   expires_at, ip_address, user_agent)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
      timeout_ms: 2000
      required_permission: "public"

    get_session_by_token_hash:
      statement: |
        SELECT id, organization_id, user_id, token_hash, expires_at,
               ip_address, user_agent, created_at
        FROM user_sessions
        WHERE token_hash = $1 AND expires_at > now()
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

  pw_reset:
    create_pw_reset_token:
      statement: |
        INSERT INTO password_reset_tokens (id, organization_id, user_id, token_hash, expires_at)
        VALUES ($1, $2, $3, $4, $5)
      timeout_ms: 2000
      required_permission: "public"

    get_pw_reset_token_by_hash:
      statement: |
        SELECT id, organization_id, user_id, token_hash, expires_at, used_at, created_at
        FROM password_reset_tokens
        WHERE token_hash = $1 AND used_at IS NULL AND expires_at > now()
      timeout_ms: 2000
      required_permission: "public"

    mark_pw_reset_token_used:
      statement: |
        UPDATE password_reset_tokens SET used_at = now() WHERE id = $1
      timeout_ms: 2000
      required_permission: "public"

  audit:
    write_audit_event:
      statement: |
        INSERT INTO audit_ledger (organization_id, actor_user_id, actor_role,
                                  event_type, target_type, target_id,
                                  ip_address, payload, previous_checksum, checksum)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
      timeout_ms: 3000
      required_permission: "public"

    query_audit_log:
      statement: |
        SELECT id, organization_id, actor_user_id, actor_role, event_type,
               target_type, target_id, ip_address, payload,
               previous_checksum, checksum, created_at
        FROM audit_ledger
        WHERE organization_id = $1
          AND created_at >= $2 AND created_at < $3
        ORDER BY created_at DESC
        LIMIT $4 OFFSET $5
      timeout_ms: 10000
      required_permission: "admin.audit"
```

**Cross-check:** After writing this file, run `go run ./cmd/aethel migrate validate` — it must still exit 0 (this command validates migration templates only, not queries.yaml, but it confirms the blueprint loader still works).

---

## Step 3 — Create Real Repository Implementations

Create the directory `aethel-core/internal/database/repos/` and write four files.

### Rules for all repo files

- Accept `*sql.DB` and `*database.QueryRegistry` as constructor args
- Use `registry.Get("group.name").Stmt` to execute prepared statements
- Always scope queries by `organization_id` for the multi-tenant invariant
- Map `sql.ErrNoRows` → `domain.ErrNotFound`
- Wrap errors with `fmt.Errorf("...: %w", err)` at every DB call boundary
- Never log inside a repository — return errors up to the service layer

---

### `internal/database/repos/user_repo.go`

```go
package repos

// UserRepo implements domain.UserRepository using prepared statements from QueryRegistry.
// All methods scope by organization_id to enforce multi-tenancy.
type UserRepo struct { ... }

func NewUserRepo(db *sql.DB, q *database.QueryRegistry) *UserRepo
```

Implement every method on `domain.UserRepository`:

- `GetByID(ctx, orgID, userID) (*domain.User, error)` → query `auth.get_user_by_id`
- `GetByEmail(ctx, orgID, email) (*domain.User, error)` → query `auth.get_user_by_email`
- `List(ctx, orgID, page) ([]domain.User, error)` → query `auth.list_users`
- `Create(ctx, *domain.User) error` → query `auth.create_user`
- `Update(ctx, *domain.User) error` → query `auth.update_user`
- `UpdatePasswordHash(ctx, userID, hash) error` → query `auth.update_password_hash`
- `IncrementFailedLogins(ctx, userID) error` → query `auth.increment_failed_logins`
- `ResetFailedLogins(ctx, userID) error` → query `auth.reset_failed_logins`
- `LockUntil(ctx, userID, until time.Time) error` → query `auth.lock_until`
- `SetLastLogin(ctx, userID) error` → query `auth.set_last_login`

Scan helper: write a private `scanUser(row)` function to avoid repeating the 13-field scan.

---

### `internal/database/repos/session_repo.go`

```go
// SessionRepo implements domain.SessionRepository.
type SessionRepo struct { ... }

func NewSessionRepo(db *sql.DB, q *database.QueryRegistry) *SessionRepo
```

Implement every method on `domain.SessionRepository`:

- `Create(ctx, *domain.Session) error` → query `session.create_session`
- `GetByTokenHash(ctx, tokenHash) (*domain.Session, error)` → query `session.get_session_by_token_hash`
- `DeleteByID(ctx, sessionID) error` → query `session.delete_session_by_id`
- `DeleteByUserID(ctx, userID) error` → query `session.delete_sessions_by_user`

---

### `internal/database/repos/pw_reset_repo.go`

```go
// PWResetRepo implements domain.PasswordResetRepository.
type PWResetRepo struct { ... }

func NewPWResetRepo(db *sql.DB, q *database.QueryRegistry) *PWResetRepo
```

Implement:

- `Create(ctx, *domain.PasswordResetToken) error` → query `pw_reset.create_pw_reset_token`
- `GetByTokenHash(ctx, hash) (*domain.PasswordResetToken, error)` → query `pw_reset.get_pw_reset_token_by_hash`
- `MarkUsed(ctx, tokenID) error` → query `pw_reset.mark_pw_reset_token_used`

---

### `internal/database/repos/audit_repo.go`

```go
// AuditRepo implements domain.AuditRepository.
// Writes to the audit_ledger partitioned table. organization_id stored as plain uuid (no FK — by design).
type AuditRepo struct { ... }

func NewAuditRepo(db *sql.DB, q *database.QueryRegistry) *AuditRepo
```

Implement:

- `Write(ctx, *domain.AuditEntry) error` → query `audit.write_audit_event`
  - The `checksum` field = SHA-256 of `(event_type + target_id + actor_user_id + previous_checksum)` concatenated as strings
  - `previous_checksum`: fetch the most recent row's checksum for this org before inserting; empty string if first row
- `Query(ctx, orgID, from, to time.Time, page domain.Page) ([]domain.AuditEntry, error)` → query `audit.query_audit_log`
- `VerifyChain(ctx, orgID, from, to time.Time) (*domain.ChainVerificationResult, error)`
  - Fetch all rows ordered by `created_at ASC` in the date range
  - Re-compute each row's expected checksum using the same formula
  - Return `ChainVerificationResult{Valid: true}` if all match; return `Valid: false` with `BrokenAtID` set to the first mismatched row's ID

---

## Step 4 — Create the Seed Loader

Create `aethel-core/internal/database/seed.go`.

```go
package database

// SeedDefaults performs idempotent first-boot seeding.
// Loads branding from blueprints/ui-theme.yaml → inserts into branding_configs if no row exists for orgID.
// Loads nav from blueprints/ui-layouts.yaml → inserts into system_settings key "nav_config" if not set.
// Safe to call on every startup — it checks existence before inserting.
func SeedDefaults(ctx context.Context, db *sql.DB, schema string, orgID uuid.UUID) error
```

Implementation rules:

- Load `blueprints/ui-theme.yaml` relative to the working directory using `blueprint.LoadThemeConfig()` — if this loader doesn't exist yet, create `internal/blueprint/theme_loader.go` that reads the YAML and returns a typed struct with fields: `primary_color`, `neutral_palette`, `font_family`, `wordmark`, `logo_path`
- Load `blueprints/ui-layouts.yaml` using `blueprint.LoadLayoutsConfig()` — create the loader if it doesn't exist; returns a struct with a `nav` field (array of nav groups matching `useAppRuntimeConfig` shape)
- INSERT into `branding_configs` only if `SELECT COUNT(*) ... WHERE organization_id = $1` returns 0
- INSERT into `system_settings` only if no row exists with `key = 'nav_config'` for the org
- The nav JSON is stored as `jsonb` — marshal the nav groups slice to JSON before inserting
- Call `SeedDefaults` in `cmd/aethel/main.go` between step 3 (migrations) and step 4 (query registry)

---

## Step 5 — Wire Real Repos in `main.go`

In `cmd/aethel/main.go`, replace the four noop declarations with real implementations:

```go
// Replace these four noops:
var (
    userRepo    domain.UserRepository          = &noopUserRepo{}
    sessionRepo domain.SessionRepository       = &noopSessionRepo{}
    pwResetRepo domain.PasswordResetRepository = &noopPWResetRepo{}
    auditRepo   domain.AuditRepository         = &noopAuditRepo{}
    // ... leave these as noop for Sprint 2:
    dispatchRepo ...
    eventRepo    ...
    routingRepo  ...
    msRepo       ...
    gnRepo       ...
    docTypeRepo  ...
    escRepo      ...
)

// With real implementations:
userRepo    := repos.NewUserRepo(db, queries)
sessionRepo := repos.NewSessionRepo(db, queries)
pwResetRepo := repos.NewPWResetRepo(db, queries)
auditRepo   := repos.NewAuditRepo(db, queries)
```

Delete the four corresponding noop struct definitions (`noopUserRepo`, `noopSessionRepo`, `noopPWResetRepo`, `noopAuditRepo`) from the bottom of `main.go`. Leave the remaining 7 noops intact with their existing comment `// replaced in Sprint 2–4`.

Add the `SeedDefaults` call after the migration step:

```go
// 3b. Seed blueprint defaults (idempotent).
// orgID is loaded from AETHEL_ORG_ID env var or the first organization row in the DB.
```

For the seed call, read `AETHEL_ORG_ID` from the environment to get the org UUID. If the env var is absent, query `SELECT id FROM organizations LIMIT 1` — if the table is empty, skip seeding (this is a fresh DB that needs the org row inserted first via a setup script).

---

## Step 6 — Unit Tests for `auth_service.go`

Create `aethel-core/internal/service/auth_service_test.go`.

Use Go's standard `testing` package only — no test framework dependencies. Write mock implementations of `domain.UserRepository`, `domain.SessionRepository`, `domain.PasswordResetRepository`, and `domain.AuditRepository` inline in the test file (keep them minimal — only implement the methods each test case exercises).

Required test cases:

| Test function                     | What it tests                                                                                                                   |
| --------------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `TestLogin_HappyPath`             | Valid email + correct password → returns `LoginResult` with non-empty `AccessToken` and `RefreshToken`                          |
| `TestLogin_WrongPassword`         | Valid email + wrong password → returns `ErrUnauthorized`; `IncrementFailedLogins` called once                                   |
| `TestLogin_AccountLocked`         | User with `LockedUntil` in the future → returns `ErrAccountLocked`                                                              |
| `TestLogin_UserNotFound`          | Email not in repo → `GetByEmail` returns `ErrNotFound` → service returns `ErrUnauthorized` (do NOT reveal whether email exists) |
| `TestRefreshSession_ValidToken`   | Valid token hash in session repo → returns new access token                                                                     |
| `TestRefreshSession_ExpiredToken` | `GetByTokenHash` returns `ErrNotFound` → returns `ErrUnauthorized`                                                              |
| `TestLogout_RevokesSession`       | `DeleteByID` called with correct session ID                                                                                     |

Run: `go test ./internal/service/... -v` — all 7 must pass.

---

## Constraints — Do NOT Do These

- **Do not touch** `internal/domain/*.go` — interfaces are final
- **Do not touch** any migration SQL files
- **Do not modify** the `QueryRegistry.Get()` panic behavior — it is intentional
- **Do not implement** database repos for dispatch, routing rules, minute sheets, green notes, escalation rules — those are Sprint 2–3 work; leave those noops in place
- **Do not add** external test libraries (testify, gomock, etc.) — use stdlib only
- **Do not run** `pnpm` or any frontend commands — this task is backend only
- **Do not commit** — the user will review and commit manually

---

## Step 7 — Verification (Sprint 1 Definition of Done)

Run each command in order. Every one must succeed before the task is complete.

```bash
# From aethel-core/
cd aethel-core

# 1. Clean build — no compile errors
go build ./...

# 2. Vet — no suspicious constructs
go vet ./...

# 3. Unit tests — all 7 auth_service tests pass
go test ./internal/service/... -v -count=1

# 4. Race detector — no data races in the full test suite
go test ./... -race -count=1

# 5. Confirm real repos compile and satisfy interfaces
# (this will fail at runtime without a DB, but must compile)
go build -o /dev/null ./cmd/aethel

# 6. Queries YAML is loadable
go run ./cmd/aethel migrate validate
```

If any command fails, fix the issue before reporting the task complete.

**Report back** with:

1. Output of `go test ./internal/service/... -v`
2. Confirmation that `go build ./...` and `go vet ./...` are clean
3. A list of every new file created with a one-line description
4. Any deviation from this plan (with justification)

---

## Reference: Sprint 1 Definition of Done (from agile plan)

- `POST /api/v1/auth/login` with valid credentials returns a signed JWT access token and opaque refresh token
- `POST /api/v1/auth/refresh` with a valid refresh token returns a new access token
- A request to a protected endpoint with no token returns `401`; with a token for the wrong role returns `403`
- `PERMISSION_DENIED` events appear in `audit_ledger` when access is denied
- All unit tests pass

The first two items require a live PostgreSQL database; the last three can be verified against the existing test suite. Mark Sprint 1 complete only when all 5 items are verifiable.
