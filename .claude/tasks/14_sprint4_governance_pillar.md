# Task 14 — Sprint 4: Governance Pillar

**Depends on:** Tasks 11 and 12 complete (dispatch + workflow layers live)
**Delivers:** Centralized audit writer, tamper-evident chain verification, escalation worker
**Sprint:** 4 of 6 (Weeks 9–10 per agile plan)

---

## Invariant Table

| Rule | Enforcement |
|------|-------------|
| Single-tenant | `app.OrgID` is the only org reference — never thread orgID through method signatures |
| Query registry | ALL SQL via `qr.Get("group.name")` from `queries.yaml` — zero inline SQL in repo files |
| Audit ledger immutable | INSERT only on `audit_ledger` — no UPDATE, no DELETE, ever |
| `audit_ledger.organization_id` is plain UUID, not FK | Records survive org deletion by design — do not add a foreign key constraint |
| Partition-aware queries | `audit_ledger` is RANGE partitioned by month — queries must include a date predicate to avoid full-table scans |
| SYS_ADMIN only | `GET /api/v1/audit-log` and `GET /api/v1/audit-log/verify` require `rbac.Require("audit.view")` which maps exclusively to the `sys_admin` role |
| No mock DB in integration tests | Use `//go:build integration` tag; tests hit real PostgreSQL 16 |
| Escalation worker respects context | Worker goroutine exits cleanly on `ctx.Done()` — no `os.Exit`, no leaked goroutines |

---

## Dependency Gate

**Before starting any block**, verify Sprint 2 and Sprint 3 are live:

```bash
cd aethel-core

# Dispatch endpoint reachable
go build ./...

# Workflow service tests pass
go test ./internal/service/... -run "TestHashChain|TestAppend|TestApprove" -v

# audit_repo exists and compiles
ls internal/database/repos/audit_repo.go
```

If `audit_repo.go` does not exist, read Task 10 and implement it first before continuing.

---

## Codebase Audit (Read Before Every Block)

Read these files **before writing anything**. Note what already exists — do not recreate:

```
internal/domain/governance.go         # AuditEntry, AuditRepository interface, audit event types
internal/domain/errors.go             # sentinel errors
internal/database/repos/audit_repo.go # may already implement Write, Query, VerifyChain
internal/service/auth_service.go      # how audit writes are currently called
internal/service/dispatch_service.go  # how audit writes are currently called
internal/service/workflow_service.go  # how audit writes are currently called
internal/database/queries/queries.yaml # existing query groups
internal/api/server.go                # registered routes
cmd/aethel/main.go                    # startup wiring
```

---

## Block A — Centralized `audit.Writer` Interface

**Goal:** All audit writes across all services funnel through one interface. No service calls the repo directly.

### Step A1 — Define the interface

File: `internal/audit/writer.go` (create new package)

```go
package audit

import (
    "context"
    "github.com/google/uuid"
    "aethel-core/internal/domain"
)

// Writer is the single audit write path for the entire application.
// Every service that needs to record an audit event must receive a Writer
// via constructor injection — never call AuditRepository directly from services.
type Writer interface {
    Write(ctx context.Context, eventType domain.AuditEventType, actorUserID uuid.UUID, resourceType, resourceID string, metadata map[string]any) error
}
```

### Step A2 — Implement `audit.DBWriter`

File: `internal/audit/db_writer.go`

```go
type DBWriter struct {
    repo domain.AuditRepository
}

func NewDBWriter(repo domain.AuditRepository) *DBWriter

func (w *DBWriter) Write(ctx context.Context, eventType domain.AuditEventType, actorUserID uuid.UUID, resourceType, resourceID string, metadata map[string]any) error {
    // marshal metadata to JSON
    // build domain.AuditEntry with app.OrgID, generated ID (uuid.New()), Now()
    // call w.repo.Write(ctx, entry)
}
```

### Step A3 — Update existing services

Read `auth_service.go`, `dispatch_service.go`, `workflow_service.go`. Each currently calls the audit repo directly. Change each service's `audit` field from `domain.AuditRepository` to `audit.Writer`. Update constructors. Update all `s.audit.Write(...)` call sites to use the new signature.

Do NOT change the audit repo itself — only change how services reference the writer.

### Step A4 — Wire in `main.go`

```go
auditRepo   := repos.NewAuditRepo(db, qr)
auditWriter := audit.NewDBWriter(auditRepo)

authSvc     = service.NewAuthService(..., auditWriter)
dispatchSvc = service.NewDispatchService(..., auditWriter)
workflowSvc = service.NewWorkflowService(..., auditWriter)
```

**Gate A:** `go build ./...` must pass before Block B.

---

## Block B — Audit Ledger Queries

File: `internal/database/queries/queries.yaml`

Read the file first. If `governance:` group is missing or incomplete, add:

```yaml
governance:
  query_paged: |
    SELECT id, org_id, event_type, actor_user_id, resource_type, resource_id,
           metadata, ip_address, previous_checksum, checksum, created_at
    FROM audit_ledger
    WHERE org_id = $1
      AND created_at >= $2
      AND created_at < $3
    ORDER BY created_at ASC, id ASC
    LIMIT $4 OFFSET $5

  query_range_for_verify: |
    SELECT id, previous_checksum, checksum, created_at
    FROM audit_ledger
    WHERE org_id = $1
      AND created_at >= $2
      AND created_at < $3
    ORDER BY created_at ASC, id ASC

  fetch_overdue_dispatches: |
    SELECT d.id, d.priority_level, d.assigned_department_id,
           d.created_at, er.id AS rule_id, er.escalation_action,
           er.threshold_hours
    FROM dispatches d
    JOIN escalation_rules er ON er.org_id = d.org_id
      AND (er.priority_filter = 'ALL' OR er.priority_filter = d.priority_level)
      AND er.is_active = true
    WHERE d.org_id = $1
      AND d.status_state NOT IN ('DELIVERED','REJECTED','ESCALATED')
      AND d.created_at < NOW() - (er.threshold_hours || ' hours')::interval
    ORDER BY d.created_at ASC
```

---

## Block C — `governance.VerifyChain`

File: `internal/service/governance_service.go`

```go
package service

type GovernanceService struct {
    audit  domain.AuditRepository
    writer audit.Writer
}

type ChainVerificationResult struct {
    Verified    bool
    TotalRows   int
    BrokenLinks []BrokenLink
}

type BrokenLink struct {
    RowID            uuid.UUID
    ExpectedChecksum string
    ActualChecksum   string
    CreatedAt        time.Time
}

// VerifyChain fetches all audit rows in [from, to) and re-computes the
// previous_checksum chain. Returns all broken links, not just the first.
func (s *GovernanceService) VerifyChain(ctx context.Context, from, to time.Time) (*ChainVerificationResult, error)
```

Chain verification algorithm:
```
rows = fetch all rows in range ordered by created_at ASC, id ASC
for i, row := range rows:
    if i == 0:
        continue  // first row has no predecessor to check
    expected = rows[i-1].Checksum
    if row.PreviousChecksum != expected:
        append BrokenLink{RowID: row.ID, Expected: expected, Actual: row.PreviousChecksum}
result.Verified = len(BrokenLinks) == 0
```

Also implement:
- `QueryAuditLog(ctx, from, to time.Time, page, pageSize int) ([]domain.AuditEntry, error)` — for the paginated list endpoint

**Gate C:** `go test ./internal/service/... -run TestGovernance` — write unit tests inline.

---

## Block D — Governance HTTP Handlers

File: `internal/api/handlers/governance.go`

### `GET /api/v1/audit-log`

Query params: `from` (RFC3339), `to` (RFC3339), `page` (int, default 1), `pageSize` (int, default 20, max 100)

```json
{
  "entries": [...],
  "page": 1,
  "pageSize": 20,
  "total": 143
}
```

### `GET /api/v1/audit-log/verify`

Query params: `from` (RFC3339), `to` (RFC3339)

```json
{
  "verified": true,
  "totalRows": 143,
  "brokenLinks": []
}
```

On verification failure:
```json
{
  "verified": false,
  "totalRows": 143,
  "brokenLinks": [
    {
      "rowId": "...",
      "expectedChecksum": "abc123",
      "actualChecksum": "def456",
      "createdAt": "2026-06-01T10:23:00Z"
    }
  ]
}
```

Both endpoints require `rbac.Require("audit.view")`. The RBAC table must map `audit.view` to the `sys_admin` role only — verify this in `internal/rbac/`.

### Register routes in `server.go`

```go
// Under sys_admin sub-router with rbac.Require("audit.view")
r.Get("/api/v1/audit-log", governanceHandler.QueryAuditLog)
r.Get("/api/v1/audit-log/verify", governanceHandler.VerifyChain)
```

Read `server.go` to find the existing admin route group pattern and extend it — do not create a duplicate router.

**Gate D:** `go build ./...` must pass.

---

## Block E — Escalation Domain + Repository

### Step E1 — Domain types

Read `internal/domain/`. If `EscalationRule` type or `EscalationRuleRepository` interface are missing, add to `internal/domain/escalation.go`:

```go
type EscalationAction string
const (
    EscalationActionReassign        EscalationAction = "REASSIGN"
    EscalationActionNotify          EscalationAction = "NOTIFY"
    EscalationActionEscalateStatus  EscalationAction = "ESCALATE_STATUS"
)

type EscalationRule struct {
    ID               uuid.UUID
    OrgID            uuid.UUID
    Name             string
    ThresholdHours   int
    PriorityFilter   string  // "ALL" | priority_level enum value
    EscalationAction EscalationAction
    IsActive         bool
    CreatedAt        time.Time
    UpdatedAt        time.Time
}

type EscalationRuleRepository interface {
    ListActive(ctx context.Context) ([]EscalationRule, error)
    GetByID(ctx context.Context, id uuid.UUID) (*EscalationRule, error)
    Create(ctx context.Context, rule *EscalationRule) error
    Update(ctx context.Context, rule *EscalationRule) error
    Delete(ctx context.Context, id uuid.UUID) error
}
```

### Step E2 — Escalation queries in queries.yaml

```yaml
escalation_rules:
  list_active: |
    SELECT * FROM escalation_rules
    WHERE org_id = $1 AND is_active = true
    ORDER BY created_at ASC
  fetch_by_id: "SELECT * FROM escalation_rules WHERE id = $1 AND org_id = $2"
  create: |
    INSERT INTO escalation_rules
      (id, org_id, name, threshold_hours, priority_filter, escalation_action, is_active, created_at, updated_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW()) RETURNING *
  update: |
    UPDATE escalation_rules
    SET name = $2, threshold_hours = $3, priority_filter = $4,
        escalation_action = $5, is_active = $6, updated_at = NOW()
    WHERE id = $1 AND org_id = $7 RETURNING *
  delete: |
    DELETE FROM escalation_rules WHERE id = $1 AND org_id = $2
```

### Step E3 — EscalationRuleRepo implementation

File: `internal/database/repos/escalation_rule_repo.go`

Standard pattern matching other repos: inject `db *sql.DB` and `qr *database.QueryRegistry`, use `qr.Get("escalation_rules.list_active")`, scan rows into `domain.EscalationRule`.

---

## Block F — Escalation Service + Worker

### Step F1 — `escalation_service.go`

File: `internal/service/escalation_service.go`

```go
type EscalationService struct {
    escalationRules domain.EscalationRuleRepository
    dispatches      domain.DispatchRepository
    notifications   domain.NotificationRepository  // may be noop for now
    writer          audit.Writer
}

// EvaluateEscalationRules is called on each worker tick.
// It queries for dispatches that have exceeded their threshold per rule,
// applies the configured action, and writes an audit entry.
func (s *EscalationService) EvaluateEscalationRules(ctx context.Context) error
```

`EvaluateEscalationRules` algorithm:
```
overdue = query governance.fetch_overdue_dispatches (all dispatches past threshold)
for each overdue dispatch:
    switch rule.EscalationAction:
    case ESCALATE_STATUS:
        dispatches.UpdateStatus(ctx, dispatch.ID, domain.StatusEscalated, uuid.Nil)
    case REASSIGN:
        // no-op for now — future sprint adds department reassignment logic
    case NOTIFY:
        // no-op for now — Sprint 5 adds SSE notification
    writer.Write(ctx, domain.AuditEscalationFired, uuid.Nil, "dispatch", dispatch.ID.String(), map[string]any{
        "rule_id": rule.ID, "action": rule.EscalationAction, "threshold_hours": rule.ThresholdHours,
    })
```

Add `AuditEscalationFired AuditEventType = "ESCALATION_FIRED"` to `internal/domain/governance.go` if not already present.

### Step F2 — `escalation_worker.go`

File: `internal/worker/escalation_worker.go`

```go
package worker

// RunEscalationWorker ticks on interval and calls EvaluateEscalationRules.
// Exits when ctx is cancelled (SIGTERM). Logs results with zerolog.
func RunEscalationWorker(ctx context.Context, ticker *time.Ticker, svc *service.EscalationService, log zerolog.Logger) {
    for {
        select {
        case <-ctx.Done():
            log.Info().Msg("escalation worker stopped")
            return
        case <-ticker.C:
            if err := svc.EvaluateEscalationRules(ctx); err != nil {
                log.Error().Err(err).Msg("escalation evaluation failed")
            }
        }
    }
}
```

### Step F3 — Wire worker in `main.go`

```go
escalationSvc := service.NewEscalationService(escalationRuleRepo, dispatchRepo, auditWriter)
ticker := time.NewTicker(60 * time.Second)  // TODO(Sprint 6): read from blueprint
go worker.RunEscalationWorker(ctx, ticker, escalationSvc, logger)
```

The worker must start AFTER the HTTP server is ready to serve (`go httpServer.ListenAndServe()`), not before.

**Gate F:** `go build ./...` must pass.

---

## Block G — Unit + Integration Tests

### G1 — Governance service unit tests

File: `internal/service/governance_service_test.go`

Use stdlib-only inline mock for `domain.AuditRepository`:

```go
type mockAuditRepo struct { entries []domain.AuditEntry }
```

Required tests:
1. `TestVerifyChain_CleanChain` — insert 10 entries with correct `previous_checksum` chain; assert `result.Verified == true`, `len(result.BrokenLinks) == 0`
2. `TestVerifyChain_BrokenAtRow5` — corrupt row 5's `previous_checksum` in the mock; assert `result.Verified == false`, `result.BrokenLinks[0].RowID == row6.ID` (the row that references the corrupted one)
3. `TestVerifyChain_EmptyRange` — empty date range returns `Verified: true`, `TotalRows: 0`

### G2 — Escalation worker unit test

File: `internal/worker/escalation_worker_test.go`

Test: `TestEscalationWorker_StopsOnContextCancel`
- Create a cancelled context
- Start `RunEscalationWorker` in a goroutine
- Assert it exits within 100ms

### G3 — Integration test

File: `internal/integration/governance_test.go` (`//go:build integration`)

```go
func TestAuditLedger_VerifyChain_DetectsTamper(t *testing.T) {
    // 1. Write 10 audit entries via the real AuditRepo
    // 2. GET /api/v1/audit-log/verify?from=...&to=... → assert verified: true
    // 3. Directly UPDATE one row's previous_checksum in the DB (raw sql.DB exec)
    // 4. GET /api/v1/audit-log/verify again → assert verified: false, brokenLinks has 1 entry
}
```

**Gate G:** `go test ./internal/service/... -run TestVerify -v` must pass. `go test ./internal/worker/... -v` must pass.

---

## Block H — Wire Everything in `main.go`

Final wiring checklist (read `main.go` and tick each off):

- [ ] `auditRepo` wired to `repos.NewAuditRepo(db, qr)`
- [ ] `auditWriter` wired to `audit.NewDBWriter(auditRepo)`
- [ ] `governanceSvc` wired to `service.NewGovernanceService(auditRepo, auditWriter)`
- [ ] `escalationRuleRepo` wired to `repos.NewEscalationRuleRepo(db, qr)`
- [ ] `escalationSvc` wired to `service.NewEscalationService(escalationRuleRepo, dispatchRepo, auditWriter)`
- [ ] `governanceHandler` registered in `server.go`
- [ ] Escalation worker goroutine started after HTTP server
- [ ] All existing services updated to accept `audit.Writer` instead of `domain.AuditRepository`

---

## Definition of Done

- [ ] `go build ./...` — zero errors
- [ ] `go vet ./...` — zero warnings
- [ ] `go test -race ./internal/...` — no data races detected
- [ ] `go test ./internal/service/... -run TestVerify` — 3/3 pass
- [ ] `go test ./internal/worker/...` — 1/1 pass
- [ ] `curl GET /api/v1/audit-log` with a non-sys_admin token returns `403`
- [ ] `curl GET /api/v1/audit-log` with a sys_admin token returns paginated entries
- [ ] `curl GET /api/v1/audit-log/verify` on an unmodified ledger returns `{"verified":true}`
- [ ] After directly corrupting one row's `previous_checksum` in the DB, `/verify` returns `{"verified":false,"brokenLinks":[...]}`
- [ ] Escalation worker starts, ticks once, logs output, and stops cleanly on SIGTERM
- [ ] Integration test `TestAuditLedger_VerifyChain_DetectsTamper` passes against real PostgreSQL 16
- [ ] Zero inline SQL strings in any `.go` file (all SQL in `queries.yaml`)
- [ ] `grep -rn "orgID\b" internal/service/ internal/database/repos/` returns zero matches
