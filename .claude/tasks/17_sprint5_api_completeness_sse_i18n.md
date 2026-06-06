# Task 17 — Sprint 5: API Completeness, SSE, and i18n

**Sprint goal:** Complete the remaining API surface (admin CRUD, notifications, SSE real-time stream), wire CSRF middleware, finalize the route registry, and deliver a fully internationalized frontend with Vietnamese + English locale support. End with an end-to-end wiring test that proves login → dispatch → green note works against the real backend.

**Prerequisites:** Tasks 10–16 complete. `go build ./...`, `go vet ./...`, and `pnpm build` all pass before this task begins.

**Primary outputs:**
- `internal/api/handlers/admin.go` — all admin CRUD endpoints
- `internal/transport/sse.go` — goroutine-safe SSEBroker
- `internal/api/handlers/notifications.go` — notification + stream endpoints
- `aethel-view/locales/en.json` + `aethel-view/locales/vi.json` — full i18n locale files
- All `$t('key')` replacements across 19 Vue pages and shared components
- Route registry finalized in `api/server.go`
- `internal/database/queries/queries.yaml` — 3 new query groups added
- CSRF middleware registered in middleware stack
- E2E wiring test confirming full reception workflow against real backend

---

## Execution Flow

```
[Step 0]  Pre-flight → /tmp/t17-preflight.md
              ↓
[Parallel] Agent 1 — Admin CRUD Handlers (Go)
           Agent 2 — SSEBroker + Notification Endpoints (Go)
           Agent 3 — Frontend i18n (Nuxt 4)
              ↓ all complete
[Serial]   Agent 4 — Route Registry + CSRF + queries.yaml + Outbound wiring
              ↓
[Serial]   Agent 5 — E2E Wiring Test
              ↓
[Serial]   Agent 6 — Build Verification + Commit
```

Agents 1, 2, and 3 write to **non-overlapping files** and can run fully in parallel.
Agent 4 runs after Agents 1 and 2 complete.
Agent 5 runs last with a running backend.

---

## Step 0 — Pre-flight (run in orchestrating session before spawning agents)

```bash
REPO=/Users/hoangharry/mh_code/internships/Bravo/aethel-workspace
cd "$REPO"

echo "## Current Go files" > /tmp/t17-preflight.md
find aethel-core -name "*.go" | sort >> /tmp/t17-preflight.md

echo -e "\n## Current Vue/TS files" >> /tmp/t17-preflight.md
find aethel-view/app -name "*.vue" -o -name "*.ts" | sort >> /tmp/t17-preflight.md

echo -e "\n## queries.yaml current groups" >> /tmp/t17-preflight.md
grep "^[a-z]" aethel-core/internal/database/queries/queries.yaml | head -40 >> /tmp/t17-preflight.md

echo -e "\n## Routes currently in server.go" >> /tmp/t17-preflight.md
grep -n "r\.Get\|r\.Post\|r\.Patch\|r\.Delete\|r\.Put" aethel-core/internal/api/server.go >> /tmp/t17-preflight.md

echo -e "\n## Git HEAD" >> /tmp/t17-preflight.md
git rev-parse --short HEAD >> /tmp/t17-preflight.md

echo -e "\n## Git status" >> /tmp/t17-preflight.md
git status --short >> /tmp/t17-preflight.md

echo "Pre-flight complete."
cat /tmp/t17-preflight.md
```

---

## Agent 1 — Admin CRUD Handlers

**You are a senior Go engineer.** Implement all admin API endpoints in `aethel-core/internal/api/handlers/admin.go`. This file may already exist as a partial stub — read its current state first, then extend it to cover every admin route.

**Working directory:** `aethel-core/`

### Step 1: Read context

```bash
cat /tmp/t17-preflight.md
cat internal/api/handlers/admin.go 2>/dev/null || echo "File does not exist yet"
cat internal/api/handlers/deps.go
cat internal/domain/governance.go
cat internal/domain/dispatch.go 2>/dev/null || find internal/domain -name "*.go" | xargs grep -l "Dispatch\|Document\|Escalation\|RoutingRule" | head -5
cat docs/architecture/architecture-api-routes.md | grep -A3 "admin\|users\|document-types\|routing-rules\|escalation\|reports\|settings\|branding"
```

### Step 2: Read repo interfaces you will call

```bash
cat internal/domain/governance.go
find internal/database/repos -name "*.go" | sort | xargs ls -la
cat internal/database/repos/escalation_rule_repo.go 2>/dev/null
cat internal/database/repos/audit_repo.go 2>/dev/null | head -60
cat internal/service/auth_service.go | head -80
```

### Step 3: Implement admin.go

Implement the following endpoint groups in `internal/api/handlers/admin.go`. Every handler must:
- Decode the request body with `json.NewDecoder(r.Body).Decode(&req)`
- Validate required fields (return `400` with JSON error on missing/invalid input)
- Call the appropriate service or repo method from `Deps`
- Return consistent JSON responses using the project's `writeJSON` helper
- Write an audit event via `deps.AuditWriter` for any state-mutating operation

**User management** (`/api/v1/admin/users`):
- `GET /api/v1/admin/users` — paginated list; query params: `?page=1&per_page=50&role=`
- `POST /api/v1/admin/users` — create user; body: `{email, full_name, role, department_id}`; must NOT allow creating `SYS_ADMIN` role (return 403 if `role == "SYS_ADMIN"`)
- `GET /api/v1/admin/users/{id}` — get single user
- `PATCH /api/v1/admin/users/{id}` — update role or department
- `DELETE /api/v1/admin/users/{id}` — soft delete (set `is_active = false`)

**Document types** (`/api/v1/admin/document-types`):
- `GET /api/v1/admin/document-types` — list all
- `POST /api/v1/admin/document-types` — create; body: `{name, code, description, requires_green_noting}`
- `PATCH /api/v1/admin/document-types/{id}` — update
- `DELETE /api/v1/admin/document-types/{id}` — soft delete

**Routing rules** (`/api/v1/admin/routing-rules`):
- `GET /api/v1/admin/routing-rules` — list with conditions and destinations embedded
- `POST /api/v1/admin/routing-rules` — create rule + conditions + destinations in one transaction
- `PATCH /api/v1/admin/routing-rules/{id}` — update priority/name/active state
- `DELETE /api/v1/admin/routing-rules/{id}` — delete rule and cascade conditions/destinations

**Escalation rules** (`/api/v1/admin/escalation-rules`):
- `GET /api/v1/admin/escalation-rules` — list all
- `POST /api/v1/admin/escalation-rules` — create; body: `{name, threshold_hours, action, is_active, conditions}`
- `PATCH /api/v1/admin/escalation-rules/{id}` — update
- `DELETE /api/v1/admin/escalation-rules/{id}` — delete

**Reports** (`/api/v1/admin/reports`):
- `GET /api/v1/admin/reports/dispatch-volume` — query params: `?from=&to=&group_by=day|week|month`; calls `admin.dispatch_volume_by_period` query
- `GET /api/v1/admin/reports/user-activity` — calls `admin.user_activity_summary` query

**Settings and branding** (these may already exist in config handler — check first):
- `PATCH /api/v1/admin/config/org` — update org name, locale, timezone; invalidate config cache
- `PATCH /api/v1/admin/config/branding` — update primary color, neutral palette, font, wordmark; invalidate cache
- `PATCH /api/v1/admin/config/features` — update feature flags
- `PATCH /api/v1/admin/config/nav` — update nav tree

If any of the config endpoints already exist in `internal/config/handler.go`, do NOT duplicate them. Just confirm they exist and move on.

### Step 4: Add missing repo methods (if needed)

If any repository interface or implementation is missing a method required by the handlers above, add it. Follow the project pattern: interface in `internal/domain/`, implementation in `internal/database/repos/`, named query in `queries.yaml` under the appropriate group.

### Step 5: Verify

```bash
go build ./... 2>&1
go vet ./... 2>&1
```

Both must exit 0. Fix any compile errors before finishing.

### Output

Write findings to `/tmp/t17-agent-1.md`:

```markdown
## Agent 1 — Admin Handlers
_Completed at: [timestamp]_

### Endpoints implemented
| Endpoint | Method | Status | Notes |
|----------|--------|--------|-------|

### New repo methods added (if any)
- [list]

### Build status
- go build: ✅/❌
- go vet: ✅/❌
```

---

## Agent 2 — SSEBroker + Notification Endpoints

**You are a senior Go engineer** with expertise in concurrent systems and HTTP streaming. Implement the real-time notification system.

**Working directory:** `aethel-core/`

### Step 1: Read context

```bash
cat /tmp/t17-preflight.md
cat internal/domain/governance.go 2>/dev/null | grep -A10 "Notification"
find internal -name "notification*" | sort
cat internal/api/server.go | grep -n "notification\|sse\|stream"
cat docs/architecture/architecture-api-routes.md | grep -A3 "notification\|stream\|sse" -i
```

### Step 2: Implement `internal/transport/sse.go`

Create `internal/transport/sse.go` with a goroutine-safe SSEBroker:

```go
// SSEBroker manages Server-Sent Events connections per user.
// Connections are keyed by userID; multiple tabs for the same user all receive events.
type SSEBroker struct {
    mu      sync.RWMutex
    clients map[uuid.UUID][]chan Event
}

type Event struct {
    Type    string      `json:"type"`
    Payload interface{} `json:"payload"`
}

// New returns an initialized SSEBroker.
func New() *SSEBroker

// Subscribe registers a new SSE channel for userID. Returns the channel and a cleanup function.
func (b *SSEBroker) Subscribe(userID uuid.UUID) (<-chan Event, func())

// Publish sends an event to all active connections for userID.
func (b *SSEBroker) Publish(userID uuid.UUID, event Event)

// ServeHTTP serves the SSE stream for the authenticated user.
// Sets headers: Content-Type: text/event-stream, Cache-Control: no-cache, Connection: keep-alive, X-Accel-Buffering: no
// Writes events in SSE format: "data: {json}\n\n"
// Sends a heartbeat comment (": heartbeat\n\n") every 30 seconds to keep the connection alive through proxies.
// Cleans up the channel when the client disconnects (context done).
func (b *SSEBroker) ServeHTTP(w http.ResponseWriter, r *http.Request)
```

Implementation requirements:
- `sync.RWMutex` protecting the clients map — reads use `RLock`, writes use `Lock`
- Each subscribe call creates a buffered `make(chan Event, 64)` channel — drops silently if full (non-blocking send with `select { case ch <- event: default: }`)
- The cleanup function passed to `Subscribe` must call `b.unsubscribe(userID, ch)` and close the channel
- The 30-second heartbeat uses `time.NewTicker(30 * time.Second)` in a `select` loop alongside the event channel and `ctx.Done()`
- No goroutine leaks: every `Subscribe` must have its cleanup called when the HTTP connection closes

### Step 3: Implement `internal/api/handlers/notifications.go`

```go
// GET /api/v1/notifications
// Returns paginated list of notifications for the authenticated user.
// Query params: ?page=1&per_page=20&unread_only=true
// Response: { notifications: [...], unread_count: N, total: N }
func (h *Handler) ListNotifications(w http.ResponseWriter, r *http.Request)

// PATCH /api/v1/notifications/{id}/read
// Marks a single notification as read. Returns 204 No Content.
func (h *Handler) MarkNotificationRead(w http.ResponseWriter, r *http.Request)

// PATCH /api/v1/notifications/read-all
// Marks all notifications for the authenticated user as read. Returns 204 No Content.
func (h *Handler) MarkAllNotificationsRead(w http.ResponseWriter, r *http.Request)

// GET /api/v1/notifications/stream
// Upgrades the connection to SSE. Calls sseBroker.ServeHTTP after auth check.
// This endpoint must be registered WITHOUT CSRF middleware (SSE uses GET, no state mutation).
func (h *Handler) NotificationStream(w http.ResponseWriter, r *http.Request)
```

The `Handler` struct in `deps.go` must have an `SSEBroker *transport.SSEBroker` field. Add it if not present.

### Step 4: Wire SSEBroker into main.go

In `cmd/aethel/main.go`, after creating the handler deps:
```go
broker := transport.New()
deps.SSEBroker = broker
```

The broker is passed by pointer — no additional goroutines needed at startup (it's event-driven).

### Step 5: Verify

```bash
go build ./... 2>&1
go vet ./... 2>&1
go test ./internal/transport/... -v 2>&1 || echo "no transport tests yet — OK"
```

Write a unit test `internal/transport/sse_test.go` covering:
- Subscribe + Publish delivers event to channel
- Multiple subscribers for same userID all receive the event
- Cleanup function removes subscriber from map
- Publish to unsubscribed userID is a no-op (no panic)

### Output

Write to `/tmp/t17-agent-2.md`:

```markdown
## Agent 2 — SSEBroker + Notifications
_Completed at: [timestamp]_

### Files created/modified
- internal/transport/sse.go: ✅
- internal/transport/sse_test.go: ✅
- internal/api/handlers/notifications.go: ✅
- internal/api/handlers/deps.go (SSEBroker field): ✅

### Test results
- SSE unit tests: N passing / N failing

### Build status
- go build: ✅/❌
- go vet: ✅/❌
```

---

## Agent 3 — Frontend i18n

**You are a senior Nuxt 4 / Vue 3 engineer** with expertise in @nuxtjs/i18n. Deliver full internationalization support across all 19 pages and shared components.

**Working directory:** `aethel-view/`

**⚠️ IMPORTANT: Read `aethel-view/CLAUDE.md` before touching `nuxt.config.ts`. The `autoConfirm` setting is currently DISABLED — confirm whether any restart-triggering file changes are needed before proceeding.**

### Step 1: Read context

```bash
cat /tmp/t17-preflight.md
cat CLAUDE.md
cat .claude-devtools/settings.json 2>/dev/null || echo "not found"
cat nuxt.config.ts
cat package.json | grep i18n
```

### Step 2: Install @nuxtjs/i18n

```bash
cd /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/aethel-view
pnpm add @nuxtjs/i18n
```

Add to `nuxt.config.ts` modules array:

```typescript
'@nuxtjs/i18n'
```

Add i18n config to `nuxt.config.ts`:

```typescript
i18n: {
  locales: [
    { code: 'en', name: 'English', file: 'en.json' },
    { code: 'vi', name: 'Tiếng Việt', file: 'vi.json' },
  ],
  defaultLocale: 'en',
  langDir: 'locales/',
  strategy: 'no_prefix',
  lazy: true,
},
```

### Step 3: Create locale files

Create `app/locales/en.json` and `app/locales/vi.json`. Read every page in `app/pages/` and every component in `app/components/` and extract ALL visible user-facing strings into the locale files. Structure keys hierarchically by feature:

```json
{
  "common": {
    "save": "Save",
    "cancel": "Cancel",
    "delete": "Delete",
    "edit": "Edit",
    "create": "Create",
    "search": "Search",
    "loading": "Loading...",
    "confirm": "Are you sure?",
    "noResults": "No results found",
    "actions": "Actions",
    "status": "Status",
    "createdAt": "Created at",
    "updatedAt": "Updated at"
  },
  "auth": {
    "login": "Sign in",
    "logout": "Sign out",
    "email": "Email address",
    "password": "Password",
    "forgotPassword": "Forgot password?",
    "loginTitle": "Welcome back",
    "loginSubtitle": "Sign in to your Aethel account"
  },
  "nav": {
    "dashboard": "Dashboard",
    "inbound": "Inbound",
    "outbound": "Outbound",
    "myDocuments": "My Documents",
    "search": "Search",
    "adminUsers": "Users",
    "adminDocTypes": "Document Types",
    "adminRoutingRules": "Routing Rules",
    "adminEscalation": "Escalation",
    "adminAuditLog": "Audit Log",
    "adminReports": "Reports",
    "adminSettings": "Settings",
    "adminBranding": "Branding",
    "adminNavigation": "Navigation"
  },
  "dispatch": { ... },
  "workflow": { ... },
  "admin": { ... },
  "urgency": {
    "IMMEDIATE": "Immediate",
    "PRIORITY": "Priority",
    "ROUTINE": "Routine"
  },
  "docStatus": {
    "PENDING_ASSIGNMENT": "Pending Assignment",
    "UNDER_REVIEW": "Under Review",
    "IN_TRANSIT": "In Transit",
    "ATTEMPTED_DELIVERY": "Attempted Delivery",
    "DELIVERED": "Delivered",
    "ESCALATED": "Escalated",
    "DISPATCHED": "Dispatched",
    "REJECTED": "Rejected"
  }
}
```

The `vi.json` file must have every key present (same structure) with correct Vietnamese translations. Do not leave any key as empty string.

### Step 4: Replace hardcoded strings in pages and components

Read each file listed below and replace every user-facing hardcoded string literal with `{{ $t('key') }}` (template) or `t('key')` (script setup with `const { t } = useI18n()`).

Files to migrate (all pages + all shared components + layout components):

```bash
find app/pages app/components/layout app/components/shared app/components/blocks -name "*.vue" | sort
```

Rules:
- Do NOT translate: CSS class names, route paths, variable names, enum values used in logic
- DO translate: button labels, page titles, table headers, placeholder text, error messages, empty state text, tooltip text, aria-labels
- For script setup, add `const { t } = useI18n()` at the top of the script block
- For computed labels (e.g., urgency badge text), use `t(\`urgency.${item.priority}\`)` or a computed map

### Step 5: Wire locale to org config

In `app/composables/useAppRuntimeConfig.ts`, after `config` is loaded, set the active i18n locale:

```typescript
const { locale } = useI18n()
watch(
  () => config.value?.org?.locale,
  (newLocale) => {
    if (newLocale && ['en', 'vi'].includes(newLocale)) {
      locale.value = newLocale
    }
  },
  { immediate: true }
)
```

### Step 6: Build check

```bash
cd /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/aethel-view
pnpm build 2>&1 | tail -40
```

Must exit 0 with zero TypeScript errors.

### Output

Write to `/tmp/t17-agent-3.md`:

```markdown
## Agent 3 — Frontend i18n
_Completed at: [timestamp]_

### Locale files
- app/locales/en.json: ✅ (N keys)
- app/locales/vi.json: ✅ (N keys, all present)
- Keys missing in vi.json: [list or "none"]

### Pages migrated
| Page | Strings replaced | Notes |
|------|-----------------|-------|

### Components migrated
| Component | Strings replaced | Notes |
|-----------|-----------------|-------|

### Build status
- pnpm build: ✅/❌
- TypeScript errors: N (must be 0)
```

---

## Agent 4 — Route Registry + CSRF + queries.yaml + Outbound Wiring

**Run after Agents 1 and 2 complete.** You are a senior Go engineer. Finalize the route registration, add CSRF middleware to the stack, add the three missing query groups to `queries.yaml`, and confirm outbound dispatch admin endpoints are wired.

**Working directory:** `aethel-core/`

### Step 1: Read current state

```bash
cat /tmp/t17-agent-1.md
cat /tmp/t17-agent-2.md
cat internal/api/server.go
cat docs/architecture/architecture-api-routes.md
cat internal/database/queries/queries.yaml | grep "^[a-z]"
find internal/api/middleware -name "csrf*" -o -name "*csrf*" | sort
```

### Step 2: Add missing queries to queries.yaml

Add the following three groups if not already present. All queries use `$1` for `organization_id`:

```yaml
admin:
  dispatch_volume_by_period: |
    SELECT
      date_trunc($3, created_at) AS period,
      COUNT(*) AS total,
      COUNT(*) FILTER (WHERE priority_level = 'IMMEDIATE') AS immediate,
      COUNT(*) FILTER (WHERE priority_level = 'PRIORITY') AS priority,
      COUNT(*) FILTER (WHERE priority_level = 'ROUTINE') AS routine
    FROM dispatches
    WHERE organization_id = $1
      AND created_at BETWEEN $2::timestamptz AND $4::timestamptz
    GROUP BY 1
    ORDER BY 1

  user_activity_summary: |
    SELECT
      u.id,
      u.full_name,
      u.role,
      COUNT(d.id) AS dispatches_created,
      COUNT(d.id) FILTER (WHERE d.status_state = 'DELIVERED') AS dispatches_delivered,
      MAX(d.created_at) AS last_activity
    FROM users u
    LEFT JOIN dispatches d ON d.created_by = u.id AND d.organization_id = $1
    WHERE u.organization_id = $1 AND u.is_active = true
    GROUP BY u.id, u.full_name, u.role
    ORDER BY last_activity DESC NULLS LAST

search:
  fulltext_dispatches: |
    SELECT
      d.id, d.tracking_number, d.subject, d.priority_level, d.status_state,
      d.created_at, dt.name AS document_type_name,
      dept.name AS assigned_department_name
    FROM dispatches d
    LEFT JOIN document_types dt ON dt.id = d.document_type_id
    LEFT JOIN departments dept ON dept.id = d.assigned_department_id
    WHERE d.organization_id = $1
      AND (
        to_tsvector('english', coalesce(d.subject,'') || ' ' || coalesce(d.tracking_number,'') || ' ' || coalesce(d.sender_name,''))
        @@ plainto_tsquery('english', $2)
        OR d.tracking_number ILIKE '%' || $2 || '%'
      )
    ORDER BY d.created_at DESC
    LIMIT 50

notifications:
  list_by_user: |
    SELECT id, user_id, title, body, type, is_read, related_dispatch_id, created_at
    FROM notifications
    WHERE user_id = $2 AND organization_id = $1
    ORDER BY created_at DESC
    LIMIT $3 OFFSET $4

  unread_count: |
    SELECT COUNT(*) FROM notifications
    WHERE user_id = $2 AND organization_id = $1 AND is_read = false

  mark_read: |
    UPDATE notifications SET is_read = true, updated_at = NOW()
    WHERE id = $2 AND user_id = $3 AND organization_id = $1

  mark_all_read: |
    UPDATE notifications SET is_read = true, updated_at = NOW()
    WHERE user_id = $2 AND organization_id = $1 AND is_read = false
```

### Step 3: Register CSRF middleware

In `internal/api/server.go`, add CSRF middleware to the authenticated route group. The CSRF middleware already exists in `internal/api/middleware/csrf.go` (from Task 08). Register it:

```go
// Inside the authenticated routes group, after Auth middleware:
r.Use(middleware.CSRFMiddleware(cfg))
```

The SSE stream endpoint (`GET /api/v1/notifications/stream`) must be **exempted** from CSRF (GET requests are idempotent; CSRF applies only to POST/PATCH/DELETE/PUT).

### Step 4: Register all new routes in server.go

After reviewing the route table in `docs/architecture/architecture-api-routes.md` and the handlers implemented by Agent 1 and Agent 2, register every missing route in `server.go`. Check for routes that exist in the architecture doc but are not yet registered:

```bash
grep "operationId" docs/architecture/architecture-api-routes.md | wc -l
grep -c "r\.Get\|r\.Post\|r\.Patch\|r\.Delete\|r\.Put" internal/api/server.go
```

Add all missing routes, grouped logically with comments:

```go
// Admin — Users
r.Get("/admin/users", h.ListUsers)
r.Post("/admin/users", h.CreateUser)
// ... etc
```

### Step 5: Verify build

```bash
go build ./... 2>&1
go vet ./... 2>&1
grep -c "r\.Get\|r\.Post\|r\.Patch\|r\.Delete\|r\.Put" internal/api/server.go
```

### Output

Write to `/tmp/t17-agent-4.md`:

```markdown
## Agent 4 — Route Registry + CSRF + queries.yaml
_Completed at: [timestamp]_

### queries.yaml groups added
- admin: ✅/❌
- search: ✅/❌
- notifications: ✅/❌

### CSRF middleware registered: ✅/❌
### SSE endpoint CSRF-exempt: ✅/❌

### Route coverage
- Routes in architecture doc: N
- Routes registered after this agent: N
- Routes still missing: [list or "none"]

### Build: ✅/❌
```

---

## Agent 5 — End-to-End Wiring Test

**Run after Agents 1–4 complete.** You are a senior integration test engineer. Verify that the Nuxt 4 frontend can complete the full reception workflow against the real running Go backend.

**Working directory:** `aethel-core/` and `aethel-view/`

### Step 1: Confirm the backend builds and starts

```bash
cd /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/aethel-core
go build -o /tmp/aethel-server ./cmd/aethel/ 2>&1
echo "Build exit code: $?"
```

### Step 2: Write integration test file `internal/integration/sprint5_test.go`

Create a new integration test file (alongside existing integration tests) that covers the Sprint 5 API surface. The test must use `testutil.NewTestDB()` or the existing test helper pattern.

Test cases to implement:

```go
// TestAdminUsersEndpoints — creates a user, lists users, updates role, deletes user
// TestAdminDocumentTypes — creates a document type, lists, updates, deletes
// TestAdminRoutingRules — creates a routing rule with conditions, lists, deletes
// TestNotificationsList — dispatches a document → verifies notification created → marks read
// TestSearchFulltext — creates dispatch with known subject → fulltext search returns it
// TestConfigAPIRoundtrip — PATCH branding → GET config → verify updated fields reflected
```

Follow the existing integration test patterns in `internal/integration/`:

```bash
ls internal/integration/
cat internal/integration/dispatch_test.go | head -60
```

### Step 3: Run all tests

```bash
cd /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/aethel-core
go test ./... 2>&1 | grep -E "^ok|FAIL|no test files"
go test ./internal/service/... -v 2>&1 | grep -E "PASS|FAIL|SKIP"
go test ./internal/integration/... -v 2>&1 | grep -E "PASS|FAIL|SKIP" || echo "Integration tests require DB — skipped in dry run"
```

### Step 4: Frontend smoke check

```bash
cd /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/aethel-view
pnpm build 2>&1 | tail -20
echo "TypeScript check..."
pnpm exec vue-tsc --noEmit 2>&1 | tail -20
```

### Output

Write to `/tmp/t17-agent-5.md`:

```markdown
## Agent 5 — E2E Wiring Test
_Completed at: [timestamp]_

### Backend build: ✅/❌
### Integration test file written: ✅/❌
### Test results
| Test | Status | Notes |
|------|--------|-------|

### Frontend build: ✅/❌
### TypeScript errors: N (must be 0)
```

---

## Agent 6 — Build Verification + Commit

**Run after all agents complete.** Verify everything builds, then commit.

### Step 1: Final build check

```bash
cd /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/aethel-core
go build ./... && go vet ./... && echo "Go: ✅" || echo "Go: ❌"

cd /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/aethel-view
pnpm build 2>&1 | tail -20 && echo "FE: ✅" || echo "FE: ❌"
```

### Step 2: Violation checks

```bash
cd /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace
echo "=== Inline SQL ===" && grep -rn '"SELECT\|"INSERT\|"UPDATE\|"DELETE' aethel-core/internal/database/repos/ | wc -l
echo "=== Palette violations ===" && grep -rn "text-slate\|text-indigo\|bg-white\|bg-slate" aethel-view/app/pages/ aethel-view/app/components/ | wc -l
echo "=== localStorage ===" && grep -rn "localStorage" aethel-view/app/ | wc -l
```

### Step 3: Commit

```bash
cd /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace
git add aethel-core/ aethel-view/
git commit -m "$(cat <<'EOF'
feat(sprint5): API completeness, SSEBroker, notifications, and i18n

- Admin CRUD endpoints: users, document-types, routing-rules, escalation-rules, reports
- SSEBroker: goroutine-safe per-user event stream with 30s heartbeat
- Notification endpoints: list, mark-read, mark-all-read, SSE stream
- queries.yaml: admin, search.fulltext_dispatches, notifications groups added
- CSRF middleware registered in authenticated route group
- Route registry finalized: all operationIds from architecture-api-routes.md wired
- i18n: @nuxtjs/i18n installed, en.json + vi.json locale files, all pages migrated
- Locale wired to org.locale from useAppRuntimeConfig

Co-Authored-By: Claude Code Task 17 <noreply@anthropic.com>
EOF
)"
```

---

## Definition of Done

- [ ] `go build ./...` passes with zero errors
- [ ] `go vet ./...` passes with zero warnings
- [ ] `pnpm build` passes with zero TypeScript errors
- [ ] All admin endpoints (users, doc-types, routing-rules, escalation-rules, reports, settings) registered and returning non-500 responses for valid requests
- [ ] `GET /api/v1/notifications/stream` returns `Content-Type: text/event-stream` for an authenticated request
- [ ] `SSEBroker` unit tests pass
- [ ] `aethel-view/app/locales/en.json` and `vi.json` exist with matching key sets
- [ ] Zero hardcoded English strings remain in page templates (grep for common literal patterns)
- [ ] `queries.yaml` contains groups: `admin`, `search`, `notifications`
- [ ] CSRF middleware registered in `server.go` for the authenticated group
- [ ] All routes from `architecture-api-routes.md` are registered in `server.go`
- [ ] Sprint 5 integration test file written with ≥5 test cases
- [ ] All tests pass: `go test ./...`
- [ ] Zero inline SQL violations: `grep -rn '"SELECT\|"INSERT' aethel-core/internal/database/repos/` returns 0
- [ ] Zero palette CSS violations: grep returns 0
- [ ] Zero localStorage violations: grep returns 0
- [ ] Changes committed to `dev` branch
