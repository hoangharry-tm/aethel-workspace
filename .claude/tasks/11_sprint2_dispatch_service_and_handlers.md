# Task 11 — Sprint 2: Dispatch Service & Handler Hardening

> **Executing agent**: read every file listed in "Read First" blocks before writing a single line of code. The codebase is more complete than the task title suggests — your job is to harden, extend, and test what exists, NOT to rewrite it.

---

## Invariant Table (enforce everywhere, no exceptions)

| Rule | Detail |
|---|---|
| **Single-tenant** | `app.OrgID` (package-level `uuid.UUID`, set at boot) is the only org reference. Never add `orgID` as a function parameter or JWT claim. The context key `rbac.OrgIDFromCtx` already injects `app.OrgID` — use that in handlers. |
| **Named queries** | All SQL that performs JOINs, aggregations, or touches more than one table goes in `aethel-core/internal/database/queries/queries.yaml`. Simple single-table CRUD may be inline in repo files. Never add inline SQL to service or handler files. |
| **No mock DB** | Integration tests in `aethel-core/internal/integration/` use a real PostgreSQL instance via `AETHEL_DB_DSN`. No sqlmock. No in-memory SQLite. |
| **Transaction rule** | Any operation that writes to two or more tables atomically must use a `*sql.Tx`. Never let a service method leave the DB in a half-written state. |
| **Do not rewrite** | If a file already implements correct behaviour, add/extend it — do not replace it. |
| **No orgID in params** | Repository method signatures that accept `orgID uuid.UUID` do so to satisfy the interface; they ignore it in SQL (single-tenant: one org per install). Do not add new `orgID` parameters to service methods. |

---

## Step 0 — Baseline audit (MANDATORY before any edits)

Run the following and read the output before proceeding:

```bash
cd aethel-core
go build ./...
go vet ./...
```

Both commands must exit 0. If they do not, fix compilation errors first and do not proceed until they are clean.

---

## What already exists (do NOT rewrite)

Read each file listed here before touching it:

| File | What is in it |
|---|---|
| `aethel-core/internal/app/org.go` | `OrgID` var + `LoadOrgID(ctx, db)` — complete |
| `aethel-core/internal/config/cache.go` | `ConfigCache` with Get/Set/Invalidate — complete |
| `aethel-core/internal/config/loader.go` | `LoadOrgConfig(ctx, db)` querying branding, nav, features, org — complete |
| `aethel-core/internal/config/handler.go` | All four GET handlers + all four PATCH handlers (branding, nav, features, org) — **complete and correct** |
| `aethel-core/internal/domain/dispatch.go` | All dispatch domain types + repository interfaces — complete |
| `aethel-core/internal/service/dispatch_service.go` | `DispatchService` with Create, GetByID, ListInbox, ListOutbound, ListByUser, UpdateStatus, Assign, Acknowledge, GetTimeline, evaluateRules, ruleMatches — **complete** |
| `aethel-core/internal/api/handlers/dispatch.go` | All HTTP handlers wired to the service — **complete** |
| `aethel-core/internal/api/server.go` | All dispatch + config routes registered — complete |
| `aethel-core/cmd/aethel/main.go` | `DispatchService` instantiated and wired — complete |
| `aethel-core/internal/database/queries/queries.yaml` | dispatch, dispatch_event, routing_rule, auth, governance queries — complete |
| `aethel-core/internal/database/repos/dispatch_repo.go` | All repo methods — complete |
| `aethel-core/internal/database/repos/dispatch_event_repo.go` | Create + ListByDispatch — complete |
| `aethel-core/internal/database/repos/routing_rule_repo.go` | List, GetByID, Create, Update, Delete — complete |
| `aethel-core/internal/integration/dispatch_test.go` | Four integration tests (TestCreateDispatch, TestAcknowledgeDelivery, TestConfigCacheInvalidation, TestRoutingRuleEngine) — complete |

---

## What this task must deliver

### Block A — Verify and fix the dispatch service condition-matching field names

**Read first**: `aethel-core/internal/service/dispatch_service.go` (function `conditionMatch`), then `aethel-core/internal/integration/dispatch_test.go` (TestRoutingRuleEngine seed SQL for condition_type).

**Gap to close**: The `conditionMatch` function switches on `c.FieldName`, which is populated from the DB column `condition_type`. The integration test seeds condition_type as `'DOCUMENT_TYPE'` (uppercase). However `conditionMatch` currently matches on the lowercase Go convention field name `"document_type_id"`. Verify that `RoutingRuleRepo.scanRoutingRules` stores `condType.String` directly into `RuleCondition.FieldName`. If `FieldName` receives `"DOCUMENT_TYPE"` from the DB, then `conditionMatch` must match on `"DOCUMENT_TYPE"`, not `"document_type_id"`.

**Steps**:

1. Read `aethel-core/internal/database/repos/routing_rule_repo.go` lines 83–171 (the `scanRoutingRules` function). Confirm that `FieldName` is set to `condType.String` which is the raw DB value (e.g. `"DOCUMENT_TYPE"`).

2. Read `aethel-core/internal/service/dispatch_service.go` function `conditionMatch`. Note the current `case` strings.

3. If there is a mismatch between the DB-stored values and the switch cases, update `conditionMatch` so the cases match the actual DB enum strings stored in `routing_rule_conditions.condition_type`. The DB seeds (in `TestRoutingRuleEngine`) use: `DOCUMENT_TYPE`, and the migration SQL (check `aethel-core/internal/database/migrations/`) may define the allowed enum values for `condition_type`. Use those as the canonical list.

4. Add `PRIORITY_LEVEL` and `SENDER_ORGANIZATION` as additional condition_type cases if not already present, mapped to `d.PriorityLevel` and `d.SenderOrganization` respectively.

5. For `Operator`, verify the DB stores `"EQUALS"` or `"CONTAINS"` (check the integration test seed: `match_operator = 'EQUALS'`). Update the `conditionMatch` switch to handle `"EQUALS"` and `"CONTAINS"` in addition to the existing `"eq"`, `"="`.

**Verification gate A**:
```bash
cd aethel-core
go build ./...
go vet ./...
```
Both must exit 0.

---

### Block B — Add `dispatch.list_inbox_unassigned` query to queries.yaml

**Read first**: `aethel-core/internal/database/queries/queries.yaml` (the `dispatch` section), and `aethel-core/internal/api/handlers/dispatch.go` (`ListInbox` handler).

**Gap to close**: The current `dispatch.list_inbox` query filters on `assigned_department_id = $1`, which means unassigned dispatches (PENDING_ASSIGNMENT with `assigned_department_id IS NULL`) are invisible to the reception inbox. An admin or supervisor needs a way to see unassigned items. Add a new named query for this.

**Steps**:

1. Open `aethel-core/internal/database/queries/queries.yaml`.

2. Inside the `dispatch:` block, after the existing `list_inbox:` entry, add:

```yaml
    list_inbox_unassigned:
      statement: |
        SELECT id, organization_id, tracking_number, direction, document_type_id,
               sender_name, sender_organization,
               recipient_name, recipient_organization, recipient_address,
               assigned_user_id, assigned_department_id, submitted_by_user_id,
               priority_level, status_state, subject_line, delivery_mode,
               is_manually_routed, original_suggested_user_id,
               overdue_at, acknowledged_at, acknowledged_by_user_id,
               is_escalated, created_at, updated_at
        FROM dispatches
        WHERE assigned_department_id IS NULL
          AND status_state = 'PENDING_ASSIGNMENT'
        ORDER BY CASE priority_level
                   WHEN 'IMMEDIATE' THEN 0
                   WHEN 'PRIORITY'  THEN 1
                   ELSE 2
                 END,
                 created_at ASC
        LIMIT $1 OFFSET $2
      params:
        - "integer"  # $1: limit
        - "integer"  # $2: offset
      timeout_ms: 5000
      required_permission: "dispatch.view"
```

3. Add a corresponding method to `aethel-core/internal/database/repos/dispatch_repo.go`:

```go
func (r *DispatchRepo) ListUnassigned(ctx context.Context, _ uuid.UUID, page domain.Page) ([]domain.Dispatch, error) {
    rows, err := r.q.Get("dispatch.list_inbox_unassigned").Stmt.QueryContext(ctx, page.Limit, page.Offset)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    return scanDispatches(rows)
}
```

4. Add `ListUnassigned` to the `DispatchRepository` interface in `aethel-core/internal/domain/dispatch.go`:

```go
ListUnassigned(ctx context.Context, orgID uuid.UUID, page Page) ([]Dispatch, error)
```

5. Add a service method in `aethel-core/internal/service/dispatch_service.go`:

```go
func (s *DispatchService) ListUnassigned(ctx context.Context, page domain.Page) ([]domain.Dispatch, error) {
    return s.dispatches.ListUnassigned(ctx, app.OrgID, page)
}
```

Note: import `"aethel-core/internal/app"` if not already imported.

6. Add an HTTP handler in `aethel-core/internal/api/handlers/dispatch.go`:

```go
func (h *DispatchHandler) ListUnassigned(w http.ResponseWriter, r *http.Request) {
    page := pageFromQuery(r)
    dispatches, err := h.svc.ListUnassigned(r.Context(), page)
    if err != nil {
        writeError(w, "failed to list unassigned dispatches", http.StatusInternalServerError)
        return
    }
    writeJSON(w, http.StatusOK, dispatches)
}
```

7. Register the route in `aethel-core/internal/api/server.go`, after the existing dispatch routes:

```go
r.With(rbac.Require("admin.access")).Get("/dispatches/unassigned", dispatchSvc.ListUnassigned)
```

**Verification gate B**:
```bash
cd aethel-core && go build ./...
```

---

### Block C — Add `dispatch.GetTimeline` route registration check

**Read first**: `aethel-core/internal/api/server.go` (the `/dispatches/{id}` subrouter block).

**Gap to close**: The `GetTimeline` handler exists in `dispatch.go` but may not be registered. Confirm there is a route for `GET /dispatches/{id}/timeline`.

**Steps**:

1. Search `aethel-core/internal/api/server.go` for `timeline`. If a line like:
   ```go
   r.With(rbac.Require("dispatch.view")).Get("/timeline", dispatchSvc.GetTimeline)
   ```
   exists inside the `/dispatches/{id}` route block, this block is done — skip to Block D.

2. If it is missing, add it inside the `r.Route("/dispatches/{id}", ...)` block alongside the other sub-routes:
   ```go
   r.With(rbac.Require("dispatch.view")).Get("/timeline", dispatchSvc.GetTimeline)
   ```

**Verification gate C**:
```bash
cd aethel-core && go build ./...
```

---

### Block D — Harden `DispatchService.Create` with a DB transaction

**Read first**: `aethel-core/internal/service/dispatch_service.go` (the `Create` method), and `aethel-core/internal/database/repos/dispatch_repo.go`.

**Gap to close**: The current `Create` method calls `s.dispatches.Create`, then `s.minuteSheets.Create`, then `s.events.Create` as separate operations. If `minuteSheets.Create` fails after `dispatches.Create` succeeds, the DB is left with a dispatch that has no minute sheet. These three writes must be atomic.

**Important**: The existing `DispatchRepository` and `MinuteSheetRepository` interfaces do not expose `*sql.Tx` variants — do not modify those interfaces. Instead, use the pattern of accepting a `*sql.DB` in the service and calling `db.BeginTx` directly, then doing the three inserts via the `tx` object inside the service.

**Steps**:

1. Add a `db *sql.DB` field to `DispatchService`:

```go
type DispatchService struct {
    dispatches   domain.DispatchRepository
    events       domain.DispatchEventRepository
    routingRules domain.RoutingRuleRepository
    minuteSheets domain.MinuteSheetRepository
    audit        domain.AuditRepository
    db           *sql.DB  // for transactional Create
}
```

2. Update `NewDispatchService` to accept and store `db *sql.DB`:

```go
func NewDispatchService(
    dispatches domain.DispatchRepository,
    events domain.DispatchEventRepository,
    routingRules domain.RoutingRuleRepository,
    minuteSheets domain.MinuteSheetRepository,
    audit domain.AuditRepository,
    db *sql.DB,
) *DispatchService {
    return &DispatchService{
        dispatches:   dispatches,
        events:       events,
        routingRules: routingRules,
        minuteSheets: minuteSheets,
        audit:        audit,
        db:           db,
    }
}
```

3. Rewrite the `Create` method body to wrap the dispatch + minute-sheet inserts in a transaction. Use the following pattern — adapt variable names to match the existing implementation:

```go
func (s *DispatchService) Create(ctx context.Context, d *Dispatch, submitterID, orgID uuid.UUID, ip string) (*domain.Dispatch, error) {
    dispatch := &domain.Dispatch{ /* ... same as before ... */ }

    // Evaluate routing rules (read-only — safe outside the transaction).
    rules, err := s.routingRules.List(ctx, orgID)
    if err != nil {
        return nil, fmt.Errorf("load routing rules: %w", err)
    }
    if dest := s.evaluateRules(dispatch, rules); dest != nil {
        dispatch.AssignedDepartmentID = dest.DepartmentID
        dispatch.AssignedUserID = dest.UserID
    }

    // Transactional writes.
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("begin transaction: %w", err)
    }
    defer func() {
        if err != nil {
            _ = tx.Rollback()
        }
    }()

    // Insert dispatch directly via tx (bypass the repo's prepared stmt for the tx path).
    _, err = tx.ExecContext(ctx, `
        INSERT INTO dispatches (
            id, organization_id, tracking_number, direction, document_type_id,
            sender_name, sender_organization,
            recipient_name, recipient_organization, recipient_address,
            submitted_by_user_id, priority_level, status_state,
            subject_line, delivery_mode,
            is_manually_routed, is_escalated, created_at, updated_at
        ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,false,false,now(),now())
    `,
        dispatch.ID, dispatch.OrganizationID, dispatch.TrackingNumber,
        string(dispatch.Direction), dispatch.DocumentTypeID,
        dispatch.SenderName, dispatch.SenderOrganization,
        dispatch.RecipientName, dispatch.RecipientOrganization, dispatch.RecipientAddress,
        dispatch.SubmittedByUserID, string(dispatch.PriorityLevel), string(dispatch.StatusState),
        dispatch.SubjectLine, dispatch.DeliveryMode,
    )
    if err != nil {
        return nil, fmt.Errorf("insert dispatch: %w", err)
    }

    // Auto-create minute sheet for inbound dispatches inside the same tx.
    if dispatch.Direction == domain.DirectionInbound {
        ms := &domain.MinuteSheet{
            ID:             uuid.New(),
            OrganizationID: orgID,
            DispatchID:     dispatch.ID,
            Status:         domain.MinuteSheetOpen,
        }
        _, err = tx.ExecContext(ctx, `
            INSERT INTO minute_sheets (id, organization_id, dispatch_id, status, created_at, updated_at)
            VALUES ($1, $2, $3, $4, now(), now())
        `, ms.ID, ms.OrganizationID, ms.DispatchID, string(ms.Status))
        if err != nil {
            return nil, fmt.Errorf("insert minute sheet: %w", err)
        }
    }

    if err = tx.Commit(); err != nil {
        return nil, fmt.Errorf("commit transaction: %w", err)
    }

    // Post-commit: append event and audit entry (non-critical — failures are logged, not returned).
    eventType := "DISPATCH_CREATED"
    if dispatch.AssignedDepartmentID != nil || dispatch.AssignedUserID != nil {
        eventType = "ROUTING_APPLIED"
    }
    _ = s.events.Create(ctx, &domain.DispatchEvent{
        ID:             uuid.New(),
        OrganizationID: orgID,
        DispatchID:     dispatch.ID,
        EventType:      eventType,
        ActorUserID:    &submitterID,
        ToDeptID:       dispatch.AssignedDepartmentID,
        ToUserID:       dispatch.AssignedUserID,
    })
    _ = s.audit.Write(ctx, &domain.AuditEntry{
        OrganizationID:   orgID,
        ActorUserID:      &submitterID,
        ActionEventType:  domain.AuditDispatchCreated,
        TargetResourceID: &dispatch.ID,
        IPAddress:        &ip,
    })

    return dispatch, nil
}
```

**Important notes on the above pattern**:
- The `err` variable used in `defer` must be the named return variable or the closure must capture the local `err`. Use a named return (`(dispatch *domain.Dispatch, err error)`) so the deferred rollback sees any error set after `BeginTx`.
- The inline SQL in `Create` duplicates the `dispatch.create` prepared statement intentionally — transactions cannot use prepared statements that were prepared on a different connection. This is the correct pattern.
- Do not remove `s.minuteSheets.Create` from the `MinuteSheetRepository` interface — it is still used by other callers (workflow service, tests).

4. Update `main.go` call to `NewDispatchService` to pass `db` as the last argument:

```go
dispatchSvc := service.NewDispatchService(dispatchRepo, eventRepo, routingRepo, msRepo, auditRepo, db)
```

5. Update integration test `TestRoutingRuleEngine` in `aethel-core/internal/integration/dispatch_test.go` to pass `db` to `NewDispatchService`:

```go
svc := service.NewDispatchService(dispatchRepo, eventRepo, routingRepo, &noopMSRepo{}, &noopAuditRepo{}, db)
```

Do the same for `TestAcknowledgeDelivery` if it uses `service.NewDispatchService`.

**Verification gate D**:
```bash
cd aethel-core
go build ./...
go vet ./...
```

---

### Block E — Add `TestCreateDispatch_NoRuleMatch` integration test

**Read first**: `aethel-core/internal/integration/dispatch_test.go` (the full file — understand the existing `seedOrg`, `seedDocType`, `seedUser` helpers and the `noopMSRepo`/`noopAuditRepo` stubs).

**Gap to close**: There is no test that explicitly verifies the no-rule-match path: dispatch created with no routing rules → `status_state = 'PENDING_ASSIGNMENT'` and `assigned_department_id IS NULL`.

**Steps**:

1. Open `aethel-core/internal/integration/dispatch_test.go`.

2. Add the following test after the existing `TestCreateDispatch` function:

```go
// TestCreateDispatch_NoRuleMatch verifies that when no routing rules exist the dispatch
// remains PENDING_ASSIGNMENT with no department assigned.
func TestCreateDispatch_NoRuleMatch(t *testing.T) {
    db := openTestDB(t)
    reg := buildRegistry(t, db)
    orgID := seedOrg(t, db)
    dtID := seedDocType(t, db, orgID)
    userID := seedUser(t, db, orgID)

    // Delete all routing rules so no rule can match.
    _, err := db.ExecContext(context.Background(),
        `DELETE FROM routing_rule_conditions WHERE routing_rule_id IN (SELECT id FROM routing_rules WHERE organization_id = $1)`,
        orgID,
    )
    if err != nil {
        t.Fatalf("delete routing rule conditions: %v", err)
    }
    _, err = db.ExecContext(context.Background(),
        `DELETE FROM routing_rule_destinations WHERE routing_rule_id IN (SELECT id FROM routing_rules WHERE organization_id = $1)`,
        orgID,
    )
    if err != nil {
        t.Fatalf("delete routing rule destinations: %v", err)
    }
    _, err = db.ExecContext(context.Background(),
        `DELETE FROM routing_rules WHERE organization_id = $1`, orgID,
    )
    if err != nil {
        t.Fatalf("delete routing rules: %v", err)
    }

    dispatchRepo := repos.NewDispatchRepo(db, reg)
    eventRepo := repos.NewDispatchEventRepo(db, reg)
    routingRepo := repos.NewRoutingRuleRepo(db, reg)
    svc := service.NewDispatchService(dispatchRepo, eventRepo, routingRepo, &noopMSRepo{}, &noopAuditRepo{}, db)

    created, err := svc.Create(context.Background(), &service.Dispatch{
        Direction:      "INBOUND",
        DocumentTypeID: dtID,
        SenderName:     "No-Rule Test Sender",
        PriorityLevel:  "ROUTINE",
    }, userID, orgID, "127.0.0.1")
    if err != nil {
        t.Fatalf("Create: %v", err)
    }
    if created.StatusState != domain.StatusPendingAssignment {
        t.Errorf("status: want PENDING_ASSIGNMENT, got %v", created.StatusState)
    }
    if created.AssignedDepartmentID != nil {
        t.Errorf("assigned_department_id: want nil, got %v", created.AssignedDepartmentID)
    }
}
```

**Verification gate E**:
```bash
cd aethel-core && go build ./...
```

---

### Block F — Final verification sequence

Run all gates in order. Every command must exit 0. Fix any failures before proceeding to the next command.

**Gate 1 — Clean build and vet**:
```bash
cd aethel-core
go build ./...
go vet ./...
```

**Gate 2 — Unit tests** (no DB required):
```bash
cd aethel-core
go test ./internal/service/... -v -run TestAuthService
```

**Gate 3 — Integration tests** (requires running PostgreSQL):

Start the database if it is not running:
```bash
# From the workspace root
docker compose up -d postgres
# Wait for it to be ready
docker compose exec postgres pg_isready -U aethel -d aethel_db
```

Apply migrations:
```bash
cd aethel-core
go run ./cmd/aethel migrate up
```

Run integration tests:
```bash
AETHEL_DB_DSN="postgres://aethel:aethel@localhost:5433/aethel_db?sslmode=disable" \
  go test -v -tags integration ./internal/integration/... \
  -run "TestCreateDispatch|TestCreateDispatch_NoRuleMatch|TestAcknowledgeDelivery|TestConfigCacheInvalidation|TestRoutingRuleEngine"
```

Expected: all five tests pass.

**Gate 4 — Live smoke test** (requires server running):

In a separate terminal:
```bash
cd aethel-core
AETHEL_JWT_SECRET=dev-secret-change-in-production go run ./cmd/aethel serve
```

Then:
```bash
# Login
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"emailAddress":"admin@example.com","password":"admin123"}' | jq -r .accessToken)

echo "Token: $TOKEN"

# GET config
curl -s http://localhost:8080/api/v1/config \
  -H "Authorization: Bearer $TOKEN" | jq .branding.primaryColor

# GET unassigned dispatches (new endpoint from Block B)
curl -s http://localhost:8080/api/v1/dispatches/unassigned \
  -H "Authorization: Bearer $TOKEN" | jq length
```

If `/auth/login` returns 401 because no admin exists yet, run:
```bash
cd aethel-core
go run ./cmd/aethel bootstrap-admin
```
Then repeat the smoke test.

---

## Definition of Done

Before marking this task complete, verify every item:

- [ ] `go build ./...` exits 0 with zero warnings
- [ ] `go vet ./...` exits 0
- [ ] `conditionMatch` handles the DB-stored condition_type values (`DOCUMENT_TYPE`, `PRIORITY_LEVEL`, `SENDER_ORGANIZATION`) and operator values (`EQUALS`, `CONTAINS`, `eq`, `=`, `neq`, `!=`)
- [ ] `dispatch.list_inbox_unassigned` query exists in `queries.yaml` and is wired to `GET /api/v1/dispatches/unassigned` with `admin.access` permission
- [ ] `GET /dispatches/{id}/timeline` is registered in `server.go`
- [ ] `DispatchService.Create` uses a `*sql.Tx` for the dispatch + minute-sheet inserts; routing-rule evaluation and event/audit writes remain outside the transaction
- [ ] `TestCreateDispatch_NoRuleMatch` exists in `internal/integration/dispatch_test.go` and passes
- [ ] All four original integration tests still pass: `TestCreateDispatch`, `TestAcknowledgeDelivery`, `TestConfigCacheInvalidation`, `TestRoutingRuleEngine`
- [ ] No inline SQL was added to service or handler files (SQL belongs in `queries.yaml` or repo files only)
- [ ] No `orgID` parameter was added to any service method (use `app.OrgID` directly)

---

## Quick reference — key file paths

```
aethel-core/
├── cmd/aethel/main.go                                    # startup sequence, NewDispatchService call
├── internal/
│   ├── app/org.go                                        # OrgID var + LoadOrgID
│   ├── config/
│   │   ├── cache.go                                      # ConfigCache
│   │   ├── loader.go                                     # LoadOrgConfig
│   │   └── handler.go                                    # GET/PATCH config handlers
│   ├── domain/dispatch.go                                # types + interfaces (add ListUnassigned here)
│   ├── service/dispatch_service.go                       # DispatchService (add db field + transactional Create)
│   ├── api/
│   │   ├── server.go                                     # route registration (add /unassigned + /timeline)
│   │   └── handlers/dispatch.go                          # HTTP handlers (add ListUnassigned)
│   ├── database/
│   │   ├── queries/queries.yaml                          # named SQL (add list_inbox_unassigned)
│   │   └── repos/
│   │       ├── dispatch_repo.go                          # add ListUnassigned method
│   │       ├── dispatch_event_repo.go
│   │       └── routing_rule_repo.go
│   └── integration/dispatch_test.go                      # add TestCreateDispatch_NoRuleMatch
```
