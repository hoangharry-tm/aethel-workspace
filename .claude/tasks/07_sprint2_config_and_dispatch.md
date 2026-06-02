# Task 07 — Sprint 2: Config API + Dispatch Pillar (Single-Tenant)

**Working directory:** `aethel-workspace/` (repo root)  
**Primary target:** `aethel-core/`  
**Sprint:** 2 of 6 (see `docs/plans/agile-implementation-plan.md`)  
**Depends on:** Task 06 complete (`go test ./internal/service/... -v` passes, `go build ./...` clean)

---

## Your Role

You are a senior Go backend engineer implementing Sprint 2 of Aethel Workspace. Task 06 (Sprint 1) produced real repository implementations for the auth layer. Your job has two phases:

1. **Correction pass** — Task 06 was written before a critical architectural decision was finalized. Fix the deviations it introduced before adding any new code.
2. **Sprint 2 implementation** — Config API (cache, loader, handlers), Dispatch pillar (domain repos, service, handlers), and audit event wiring.

Read everything listed below before writing a single line of code. Architecture decisions are documented; do not reinvent or second-guess them.

---

## Step 0 — Load Context (Mandatory, Do Not Skip)

Read every file in this list. Then output a numbered list of exactly what you will create or modify. Do not begin implementation until you have written that list.

```
# Architecture & plan
CLAUDE.md                                                          ← single-tenant model, middleware stack, config cache shape
docs/plans/agile-implementation-plan.md                           ← Sprint 2 definition of done
docs/architecture/architecture-server.md                          ← ConfigCache struct shape (single struct, NOT a map)
docs/architecture/architecture-api-routes.md                      ← all config + dispatch route definitions
docs/architecture/architecture-security.md                        ← deployment model section

# Existing code to understand before touching anything
aethel-core/cmd/aethel/main.go                                    ← current startup sequence, noop stubs
aethel-core/internal/domain/dispatch.go                          ← DispatchRepository, RoutingRuleRepository, DispatchEventRepository interfaces
aethel-core/internal/domain/user.go                              ← UserRepository (already implemented in Sprint 1)
aethel-core/internal/domain/errors.go                            ← sentinel errors
aethel-core/internal/service/dispatch_service.go                 ← what the repos must satisfy
aethel-core/internal/database/repos/                             ← Sprint 1 repos (your correction target)
aethel-core/internal/database/queries/queries.yaml               ← Sprint 1 queries (your correction target)
aethel-core/internal/config/cache.go                             ← may be a stub — you will replace it
aethel-core/internal/config/loader.go                            ← may be a stub — you will replace it
aethel-core/internal/config/handler.go                           ← may be a stub — you will replace it
aethel-core/internal/api/server.go                               ← route registration pattern
aethel-core/internal/api/handlers/auth.go                        ← handler pattern to follow
blueprints/server-database.yaml                                   ← schema name and table aliases
blueprints/ui-theme.yaml                                          ← branding seed shape
blueprints/ui-layouts.yaml                                        ← nav seed shape
aethel-core/internal/database/migrations/20260526000006_create_dispatches.up.sql
aethel-core/internal/database/migrations/20260526000008_create_dispatch_events.up.sql
aethel-core/internal/database/migrations/20260526000009_create_routing_rules.up.sql
aethel-core/internal/database/migrations/20260526000017_create_branding_configs.up.sql
aethel-core/internal/database/migrations/20260526000027000021_extend_branding_configs.up.sql
aethel-core/internal/database/migrations/20260526000016_create_system_settings.up.sql
```

---

## Critical Architectural Invariant — Single-Tenant

> **Read this before touching any file.**

Aethel is a **single-tenant self-hosted** application. Each deployment serves exactly one organization. This decision was finalized after Task 06 was written.

**What this means for your code:**

| Old (multi-tenant, wrong) | New (single-tenant, correct) |
|---|---|
| `WHERE organization_id = $1` in every query | No org filter — all rows belong to this installation |
| `orgID uuid` parameter on every repo method | No orgID parameter |
| `ConfigCache` keyed by `map[uuid.UUID]` | Single `ConfigCache` struct with one cached value |
| `TenantResolver` middleware in the stack | Does not exist — remove any reference |
| `orgID` claim in JWT | Not present — JWT carries only `userID` and `role` |

The `organization_id` column still exists in the database schema (do not alter migrations). It is populated once at first boot from the single row in the `organizations` table. Queries do not filter by it.

**Exception:** The `audit_ledger` INSERT must still include `organization_id` (it is a required non-null column). Read it from the package-level org constant described in Step 1.

---

## Step 1 — Establish the Org Constant

Before correcting any repos, establish a package-level constant that holds the installation's single org ID.

Create `aethel-core/internal/app/org.go`:

```go
package app

import "github.com/google/uuid"

// OrgID is the UUID of the single organization row for this installation.
// Set once at startup by LoadOrgID and read everywhere else.
// Never call LoadOrgID after startup.
var OrgID uuid.UUID

// LoadOrgID queries SELECT id FROM organizations LIMIT 1 and stores the result.
// Returns an error if the table is empty — the caller (main.go) should handle
// this by prompting the operator to run the setup script.
func LoadOrgID(ctx context.Context, db *sql.DB) error { ... }
```

Call `app.LoadOrgID` in `cmd/aethel/main.go` immediately after migrations, before query registry and seed loading.

---

## Step 2 — Correction Pass on Sprint 1 Output

**Do not skip this step.** Sprint 1 was written with multi-tenant assumptions. Fix the deviations now, before adding Sprint 2 code on top of them.

### 2a. Fix `queries.yaml`

Open `aethel-core/internal/database/queries/queries.yaml`.

For every query that currently has `WHERE organization_id = $1 AND ...`:
- Remove the `organization_id = $1` condition and its `$1` parameter
- Renumber the remaining `$N` placeholders starting from `$1`
- Remove `organization_id` from INSERT column lists and their VALUES placeholders

Specific queries to fix (check all of them, not just these):

```yaml
# auth.get_user_by_email: remove organization_id = $1, email becomes $1
auth.get_user_by_email: WHERE email_address = $1 AND is_active = true

# auth.get_user_by_id: remove organization_id = $1, id becomes $1
auth.get_user_by_id: WHERE id = $1

# auth.create_user: remove organization_id from INSERT (it is written separately via app.OrgID in the repo)
# auth.list_users: remove organization_id = $1, simplify to ORDER BY full_name LIMIT $1 OFFSET $2

# session.create_session: remove organization_id from INSERT
# pw_reset.create_pw_reset_token: remove organization_id from INSERT

# audit.write_audit_event: KEEP organization_id in the INSERT — it is a required column.
# Read app.OrgID in the repo method, not from a query parameter.
```

### 2b. Fix repo method signatures

Open each file in `aethel-core/internal/database/repos/`.

Remove `orgID uuid.UUID` from every method signature except `AuditRepo.Write` (which reads `app.OrgID` internally).

Update all method bodies to remove the `orgID` argument from query parameter lists.

### 2c. Fix domain interfaces

Open `aethel-core/internal/domain/user.go` (and any other domain files that declare repository interfaces with `orgID` parameters). Remove `orgID uuid.UUID` from every interface method signature.

Update `aethel-core/internal/service/auth_service.go` to remove `orgID` arguments from all repo calls.

### 2d. Fix `main.go` noop stubs

Remove `orgID` from any noop method stubs.

### 2e. Verify correction is complete

```bash
cd aethel-core
grep -rn "orgID\|organization_id.*\$1" internal/database/repos/ internal/service/ internal/domain/
```

Expected: zero matches (other than comments). If any remain, fix them before proceeding.

```bash
go build ./...
go test ./internal/service/... -v -count=1
```

Both must pass before Step 3.

---

## Step 3 — Config Cache

Replace (or implement from scratch) `aethel-core/internal/config/cache.go`.

The cache is a **single struct** — not a map, not keyed by org. There is only one config for this installation.

```go
package config

import (
    "sync"
    "time"
)

// AppConfig is the runtime configuration served to the Nuxt frontend.
// Shape must match the AppRuntimeConfig TypeScript interface in
// aethel-view/app/composables/useRuntimeConfig.ts exactly.
type AppConfig struct {
    Branding BrandingConfig `json:"branding"`
    Nav      []NavGroup     `json:"nav"`
    Features FeatureFlags   `json:"features"`
    Org      OrgProfile     `json:"org"`
}

type BrandingConfig struct {
    PrimaryColor   string  `json:"primaryColor"`
    NeutralPalette string  `json:"neutralPalette"`
    FontFamily     string  `json:"fontFamily"`
    LogoURL        *string `json:"logoUrl"`
    Wordmark       string  `json:"wordmark"`
}

type NavGroup struct {
    Label string    `json:"label"`
    Roles []string  `json:"roles"`
    Items []NavItem `json:"items"`
}

type NavItem struct {
    Label string  `json:"label"`
    Icon  string  `json:"icon"`
    To    string  `json:"to"`
    Badge *int    `json:"badge"`
}

type FeatureFlags struct {
    GreenNotingEnabled   bool `json:"greenNotingEnabled"`
    ExternalSmtpEnabled  bool `json:"externalSmtpEnabled"`
    Require2FAForAdmin   bool `json:"require2faForAdmin"`
}

type OrgProfile struct {
    Name         string `json:"name"`
    Timezone     string `json:"timezone"`
    Locale       string `json:"locale"`
    ContactEmail string `json:"contactEmail"`
}

// Cache holds the single runtime config for this installation.
type Cache struct {
    mu        sync.RWMutex
    value     *AppConfig
    expiresAt time.Time
    ttl       time.Duration
}

func NewCache(ttl time.Duration) *Cache

// Get returns the cached config and true if the cache is valid; nil and false on miss.
func (c *Cache) Get() (*AppConfig, bool)

// Set stores a new config value, resetting the TTL.
func (c *Cache) Set(cfg *AppConfig)

// Invalidate clears the cache. The next Get will miss and trigger a DB reload.
func (c *Cache) Invalidate()
```

---

## Step 4 — Config Loader

Replace (or implement from scratch) `aethel-core/internal/config/loader.go`.

```go
package config

// LoadConfig reads the current installation config from the database.
// Queries: branding_configs (one row), system_settings WHERE key = 'nav_config',
//          system_settings WHERE key IN ('feat_green_noting', 'feat_smtp', 'feat_2fa_admin'),
//          organizations (one row for org profile fields).
// Returns a fully populated AppConfig with safe defaults for any missing rows.
func LoadConfig(ctx context.Context, db *sql.DB, schema string) (*AppConfig, error)
```

Implementation rules:
- Use a single transaction with `READ COMMITTED` isolation — no partial reads
- If `branding_configs` has no row, return the default branding (from `blueprints/ui-theme.yaml` seed values hardcoded as constants, not re-reading the file at runtime)
- If `system_settings` has no `nav_config` row, return the default nav (hardcoded, matching `ui-layouts.yaml` seed structure)
- If `organizations` has no row, return an error — the installation is not initialized
- Parse the `nav_config` JSON value from `system_settings` using `encoding/json`

---

## Step 5 — Config API Handlers

Replace (or implement from scratch) `aethel-core/internal/config/handler.go`.

Implement all config endpoints from `docs/architecture/architecture-api-routes.md`:

### GET endpoints (read from cache)

```go
// HandleGetConfig serves the full AppConfig. Cache hit = no DB query.
// On cache miss: calls LoadConfig, stores result in cache, returns it.
func HandleGetConfig(cache *Cache, db *sql.DB, schema string) http.HandlerFunc

// HandleGetBranding, HandleGetNav, HandleGetFeatures:
// same pattern — call HandleGetConfig internally, return the relevant sub-field.
```

### PATCH endpoints (write to DB, then invalidate cache)

```go
// HandlePatchBranding updates branding_configs and invalidates the cache.
// Body: { "primaryColor"?: string, "neutralPalette"?: string, "fontFamily"?: string, "wordmark"?: string }
// Validates: primaryColor must match ^#[0-9a-fA-F]{6}$ if present.
// neutralPalette must be one of: slate, zinc, gray, stone, neutral.
// Requires: admin.access permission (enforced by RBAC middleware, not this handler).
func HandlePatchBranding(cache *Cache, db *sql.DB, schema string) http.HandlerFunc

// HandlePatchNav updates system_settings key 'nav_config' and invalidates the cache.
// Body: { "nav": NavGroup[] }
// Validates: nav must be a non-empty array; each group must have label, roles, and items.
func HandlePatchNav(cache *Cache, db *sql.DB, schema string) http.HandlerFunc

// HandlePatchFeatures updates system_settings feature flag keys and invalidates the cache.
// Body: { "greenNotingEnabled"?: bool, "externalSmtpEnabled"?: bool, "require2faForAdmin"?: bool }
func HandlePatchFeatures(cache *Cache, db *sql.DB, schema string) http.HandlerFunc

// HandlePatchOrg updates the organizations row and invalidates the cache.
// Body: { "name"?: string, "timezone"?: string, "locale"?: string, "contactEmail"?: string }
func HandlePatchOrg(cache *Cache, db *sql.DB, schema string) http.HandlerFunc
```

Handler pattern (follow the pattern in `internal/api/handlers/auth.go`):
- Decode body with `json.NewDecoder(r.Body).Decode`
- On validation failure: write `400` with `{"error": "...", "field": "..."}` JSON
- On DB error: write `500` with `{"error": "internal error"}` — do not leak DB messages
- On success: write `200` with the updated sub-config as JSON

---

## Step 6 — Dispatch Queries

Append to `aethel-core/internal/database/queries/queries.yaml`.

Add these query groups (derive exact column lists from the migration SQL files you read in Step 0):

```yaml
dispatch:
  create:
    statement: |
      INSERT INTO dispatches (
        id, tracking_number, direction, document_type_id,
        sender_name, sender_organization,
        recipient_name, recipient_organization, recipient_address,
        submitted_by_user_id, priority_level, status_state,
        subject_line, delivery_mode, is_manually_routed,
        is_escalated, created_at, updated_at
      ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,false,now(),now())
    timeout_ms: 3000
    required_permission: "dispatch.create"

  get_by_id:
    statement: |
      SELECT id, tracking_number, direction, document_type_id,
             sender_name, sender_organization,
             recipient_name, recipient_organization, recipient_address,
             assigned_user_id, assigned_department_id, submitted_by_user_id,
             priority_level, status_state, subject_line, delivery_mode,
             is_manually_routed, original_suggested_user_id,
             overdue_at, acknowledged_at, acknowledged_by_user_id,
             handoff_signature_data, is_escalated, created_at, updated_at
      FROM dispatches WHERE id = $1
    timeout_ms: 3000
    required_permission: "dispatch.view"

  list_inbox:
    statement: |
      SELECT id, tracking_number, direction, document_type_id,
             sender_name, priority_level, status_state, created_at
      FROM dispatches
      WHERE assigned_department_id = $1
        AND status_state NOT IN ('DELIVERED','DISPATCHED','REJECTED')
      ORDER BY priority_level DESC, created_at ASC
      LIMIT $2 OFFSET $3
    timeout_ms: 5000
    required_permission: "dispatch.view"

  list_outbound:
    statement: |
      SELECT id, tracking_number, direction, document_type_id,
             recipient_name, recipient_organization, priority_level,
             status_state, created_at
      FROM dispatches
      WHERE direction = 'OUTBOUND'
      ORDER BY created_at DESC
      LIMIT $1 OFFSET $2
    timeout_ms: 5000
    required_permission: "dispatch.view"

  list_by_user:
    statement: |
      SELECT id, tracking_number, direction, document_type_id,
             sender_name, priority_level, status_state, created_at
      FROM dispatches
      WHERE submitted_by_user_id = $1
      ORDER BY created_at DESC
      LIMIT $2 OFFSET $3
    timeout_ms: 5000
    required_permission: "dispatch.view"

  update_status:
    statement: |
      UPDATE dispatches SET status_state = $2, updated_at = now() WHERE id = $1
    timeout_ms: 2000
    required_permission: "dispatch.update"

  assign:
    statement: |
      UPDATE dispatches
      SET assigned_user_id = $2, assigned_department_id = $3,
          is_manually_routed = $4, original_suggested_user_id = $5,
          updated_at = now()
      WHERE id = $1
    timeout_ms: 2000
    required_permission: "dispatch.update"

  acknowledge:
    statement: |
      UPDATE dispatches
      SET acknowledged_at = now(), acknowledged_by_user_id = $2,
          status_state = 'DELIVERED', updated_at = now()
      WHERE id = $1
    timeout_ms: 2000
    required_permission: "dispatch.acknowledge"

  mark_escalated:
    statement: |
      UPDATE dispatches SET is_escalated = true, status_state = 'ESCALATED',
             updated_at = now()
      WHERE id = $1
    timeout_ms: 2000
    required_permission: "dispatch.update"

  search_by_tracking:
    statement: |
      SELECT id, tracking_number, direction, document_type_id,
             sender_name, priority_level, status_state, created_at
      FROM dispatches WHERE tracking_number = $1
    timeout_ms: 2000
    required_permission: "dispatch.view"

  fetch_overdue_for_escalation:
    statement: |
      SELECT id FROM dispatches
      WHERE is_escalated = false
        AND overdue_at IS NOT NULL
        AND overdue_at < now()
        AND status_state NOT IN ('DELIVERED','DISPATCHED','REJECTED')
    timeout_ms: 5000
    required_permission: "dispatch.view"

dispatch_event:
  create:
    statement: |
      INSERT INTO dispatch_events (
        id, dispatch_id, routing_rule_id, routing_stop_order,
        event_type, actor_user_id, target_user_id, target_department_id,
        stop_status, note, metadata, created_at
      ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,now())
    timeout_ms: 2000
    required_permission: "dispatch.view"

  list_for_dispatch:
    statement: |
      SELECT id, dispatch_id, routing_rule_id, routing_stop_order,
             event_type, actor_user_id, target_user_id, target_department_id,
             stop_status, note, metadata, created_at
      FROM dispatch_events
      WHERE dispatch_id = $1
      ORDER BY created_at ASC
    timeout_ms: 3000
    required_permission: "dispatch.view"

routing_rule:
  list_active:
    statement: |
      SELECT r.id, r.name, r.priority_order, r.is_active, r.is_multi_stop,
             r.created_by_user_id, r.created_at, r.updated_at
      FROM routing_rules r
      WHERE r.is_active = true
      ORDER BY r.priority_order ASC
    timeout_ms: 3000
    required_permission: "admin.access"

  get_with_conditions_and_destinations:
    statement: |
      SELECT r.id, r.name, r.priority_order, r.is_multi_stop,
             c.id AS cond_id, c.condition_type, c.condition_value, c.match_operator,
             d.id AS dest_id, d.stop_order, d.target_user_id, d.target_department_id,
             d.confirmation_required
      FROM routing_rules r
      LEFT JOIN routing_rule_conditions c ON c.routing_rule_id = r.id
      LEFT JOIN routing_rule_destinations d ON d.routing_rule_id = r.id
      WHERE r.id = $1
    timeout_ms: 3000
    required_permission: "admin.access"
```

After appending, run:
```bash
go run ./cmd/aethel migrate validate
```
Must still exit 0.

---

## Step 7 — Dispatch Repository

Create `aethel-core/internal/database/repos/dispatch_repo.go`.

```go
// DispatchRepo implements domain.DispatchRepository.
type DispatchRepo struct { db *sql.DB; q *database.QueryRegistry }

func NewDispatchRepo(db *sql.DB, q *database.QueryRegistry) *DispatchRepo
```

Implement every method on `domain.DispatchRepository`:
- `Create(ctx, *domain.Dispatch) error` → `dispatch.create`; generate `tracking_number` as `"AE-" + 8 uppercase hex chars from uuid.New()`
- `GetByID(ctx, id) (*domain.Dispatch, error)` → `dispatch.get_by_id`
- `ListInbox(ctx, deptID, page) ([]domain.Dispatch, error)` → `dispatch.list_inbox`
- `ListOutbound(ctx, page) ([]domain.Dispatch, error)` → `dispatch.list_outbound`
- `ListByUser(ctx, userID, page) ([]domain.Dispatch, error)` → `dispatch.list_by_user`
- `UpdateStatus(ctx, id, status) error` → `dispatch.update_status`
- `Assign(ctx, id, userID, deptID, isManual, origSuggestedUserID) error` → `dispatch.assign`
- `Acknowledge(ctx, id, byUserID) error` → `dispatch.acknowledge`
- `MarkEscalated(ctx, id) error` → `dispatch.mark_escalated`
- `SearchByTracking(ctx, trackingNumber) (*domain.Dispatch, error)` → `dispatch.search_by_tracking`
- `FetchOverdueForEscalation(ctx) ([]uuid.UUID, error)` → `dispatch.fetch_overdue_for_escalation`

Write a private `scanDispatch(row)` helper to avoid repeating the 25-field scan.

Create `aethel-core/internal/database/repos/dispatch_event_repo.go`:
- `Create(ctx, *domain.DispatchEvent) error` → `dispatch_event.create`
- `ListForDispatch(ctx, dispatchID) ([]domain.DispatchEvent, error)` → `dispatch_event.list_for_dispatch`

Create `aethel-core/internal/database/repos/routing_rule_repo.go`:
- `ListActive(ctx) ([]domain.RoutingRule, error)` → `routing_rule.list_active`
- `GetWithDetails(ctx, id) (*domain.RoutingRule, error)` → `routing_rule.get_with_conditions_and_destinations`; scan the join result into the rule + its conditions slice + destinations slice

---

## Step 8 — Wire Sprint 2 in `main.go`

In `cmd/aethel/main.go`:

1. Initialize config cache: `configCache := config.NewCache(5 * time.Minute)`
2. Replace the three dispatch noops with real repos:
   ```go
   dispatchRepo := repos.NewDispatchRepo(db, queries)
   eventRepo    := repos.NewDispatchEventRepo(db, queries)
   routingRepo  := repos.NewRoutingRuleRepo(db, queries)
   ```
3. Delete the three corresponding noop struct definitions. Leave the remaining 4 noops (`msRepo`, `gnRepo`, `docTypeRepo`, `escRepo`) in place with the comment `// implemented in Sprint 3–4`.
4. Register config routes in `api/server.go`:
   ```
   GET  /api/v1/config                   → config.HandleGetConfig(configCache, db, schema)
   GET  /api/v1/config/branding          → config.HandleGetBranding(configCache, db, schema)
   GET  /api/v1/config/nav               → config.HandleGetNav(configCache, db, schema)
   GET  /api/v1/config/features          → config.HandleGetFeatures(configCache, db, schema)
   PATCH /api/v1/admin/config/branding   → config.HandlePatchBranding(configCache, db, schema)
   PATCH /api/v1/admin/config/nav        → config.HandlePatchNav(configCache, db, schema)
   PATCH /api/v1/admin/config/features   → config.HandlePatchFeatures(configCache, db, schema)
   PATCH /api/v1/admin/config/org        → config.HandlePatchOrg(configCache, db, schema)
   ```
5. Register dispatch routes (follow the route table in `docs/architecture/architecture-api-routes.md` — do not invent routes or skip any).

---

## Step 9 — Integration Test (Requires Live PostgreSQL)

Create `aethel-core/internal/integration/dispatch_test.go` with build tag `//go:build integration`.

```go
//go:build integration

// Run with: go test ./internal/integration/... -tags integration -v
// Requires: AETHEL_DB_DSN environment variable pointing to a test PostgreSQL 16 instance.
// The test creates and tears down its own schema to avoid polluting the dev database.
```

Required test cases:

| Test | Steps | Assertion |
|---|---|---|
| `TestCreateDispatch` | Create a dispatch | `GetByID` returns the same dispatch; tracking number starts with `AE-` |
| `TestRoutingRuleEngine` | Seed one routing rule with a DOCUMENT_TYPE condition; create a dispatch with that doc type | `DispatchService.CreateDispatch` assigns the dispatch to the rule's destination; a `ROUTING_APPLIED` event exists in dispatch_events |
| `TestAcknowledgeDelivery` | Create dispatch → `UpdateStatus(DELIVERED)` → `Acknowledge` | Status is DELIVERED; `acknowledged_at` is not null |
| `TestConfigCacheInvalidation` | Call `GET /api/v1/config` → `PATCH /api/v1/admin/config/branding` → `GET /api/v1/config` again | Second GET reads updated primaryColor; cache miss is confirmed by a spy on `LoadConfig` |

Use a real PostgreSQL instance. Do not mock the database. See `docs/guides/go-developer-guide.md` for the test DB setup pattern.

---

## Constraints — Do NOT Do These

- **Do not** add `organization_id` to any new query parameter list (except `audit_ledger` INSERT — see Step 2)
- **Do not** create a `TenantResolver` middleware or reference orgID in the middleware stack
- **Do not** implement minute sheet, green note, or escalation rule repos — those are Sprint 3–4 work; leave their noops in place
- **Do not** touch any migration SQL files
- **Do not** modify `internal/domain/*.go` interface files — the domain is frozen
- **Do not** add external dependencies beyond those already in `go.mod`
- **Do not** commit — the user will review and commit manually
- **Do not** run `pnpm` or any frontend commands — this task is backend only

---

## Step 10 — Verification (Sprint 2 Definition of Done)

Run each command in order. Every one must succeed before the task is complete.

```bash
cd aethel-core

# 1. Clean build
go build ./...

# 2. Vet
go vet ./...

# 3. Single-tenant correction verified — must return zero matches
grep -rn "orgID\b" internal/database/repos/ internal/service/ internal/domain/
# Expected: 0 matches

# 4. Unit tests still pass
go test ./internal/service/... -v -count=1

# 5. Race detector — full suite
go test ./... -race -count=1

# 6. Migration validate still passes
go run ./cmd/aethel migrate validate

# 7. Integration tests (requires live PostgreSQL — skip if not available, but note it)
AETHEL_DB_DSN="postgres://..." go test ./internal/integration/... -tags integration -v

# 8. Server starts and responds to health probe
go run ./cmd/aethel serve &
sleep 2
curl -s http://localhost:8080/healthz | grep -q "ok"
curl -s http://localhost:8080/api/v1/config | python3 -m json.tool
kill %1
```

**Report back with:**

1. Output of `go test ./internal/service/... -v` (unit tests)
2. Output of `grep -rn "orgID\b" internal/database/repos/ internal/service/ internal/domain/` (must be empty)
3. Output of `curl -s http://localhost:8080/api/v1/config` (must be valid JSON matching `AppRuntimeConfig` shape)
4. Integration test results (or a clear note that PostgreSQL was not available and which tests were skipped)
5. A list of every new or modified file with a one-line description of the change
6. Any deviation from this plan with justification

---

## Reference: Sprint 2 Definition of Done (from agile plan)

- `POST /api/v1/dispatches` creates a dispatch, evaluates routing rules, assigns to the matched department, and logs a `DISPATCH_CREATED` audit event
- `GET /api/v1/dispatches` returns the active inbox for the authenticated RECEPTION user's department
- `POST /api/v1/dispatches/{id}/acknowledge` transitions status to DELIVERED and logs `DISPATCH_DELIVERED`
- Integration tests pass against a real PostgreSQL 16 instance
- `GET /api/v1/config` returns valid JSON matching the `AppRuntimeConfig` TypeScript interface
- Cache invalidation: a `PATCH /api/v1/admin/config/branding` followed by `GET /api/v1/config` returns the updated value
