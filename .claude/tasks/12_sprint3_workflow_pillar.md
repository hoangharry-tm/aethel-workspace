# Task 12 — Sprint 3: Green Noting Canvas Repository Layer + Tests

**Working directory:** `aethel-workspace/` (repo root)  
**Primary target:** `aethel-core/`  
**Sprint:** 3 of 6  
**Depends on:** Task 11 complete — `POST /api/v1/dispatches` returns 201.

---

## Dependency Gate — Do Not Skip

Before writing a single line of code, verify Sprint 2 is live:

```bash
cd aethel-core && go build ./...
# Must exit 0 with no output.
```

If the build fails, stop and fix it before continuing. Task 12 builds on top of working Sprint 2 dispatch code.

---

## Invariant Table — Read Before Writing Any Code

| Invariant | Rule | Consequence of violation |
|---|---|---|
| **Immutability** | `green_notes` rows are INSERT-only. No `UPDATE`, no `DELETE`, ever. There is no `updated_at` column. | Data loss; breaks chain verification |
| **Hash formula** | `SHA-256(contentBody \|\| "|" \|\| strconv.Itoa(seq) \|\| "|" \|\| authorID.String() \|\| "|" \|\| prevHash)` — separator `"|"` between every field | Hash mismatch across restarts |
| **First-note anchor** | First note in a sheet uses `previousHash = "genesis"` (the constant `firstNoteAnchor` defined in `workflow_service.go`). In SQL this is stored as the literal string `"genesis"`, NOT NULL. | Chain breaks on re-read |
| **Chain validation order** | Before inserting note N, re-derive note N-1's expected hash from its stored fields and compare with its stored `cryptographic_hash`. Mismatch → `domain.ErrHashChainBroken`. | Silent corruption goes undetected |
| **SELECT FOR UPDATE** | The `GetLastByMinuteSheet` call inside `Append` MUST use `SELECT … FOR UPDATE` within a transaction to prevent two concurrent appends both computing `seq=N`. Skipping this causes a race condition and duplicate-sequence constraint violations under load. | Race condition; duplicate sequence under concurrency |
| **No inline SQL** | Every database query goes through the `QueryRegistry` (`qr.Get("group.name").Stmt.ExecContext(…)`). No `db.QueryRowContext` with literal SQL strings anywhere in `repos/`. | Breaks prepared-statement caching; violates code conventions |
| **Single-tenant orgID** | `app.OrgID` is the package-level `uuid.UUID` set once at boot. Never thread `orgID` through function parameters in `repos/` calls or store it in JWT claims. The domain interfaces accept `orgID` as a parameter to satisfy interface contracts — it may be passed as `app.OrgID` at the call site but repos can use it in SQL only if the column exists in the schema. See the note below on schema reality. | Architecture violation |
| **Schema reality** | The actual `minute_sheets` migration (migration 12) has columns: `id`, `dispatch_id`, `created_at`, `updated_at`. It does NOT have `organization_id`, `status`, `approved_by_id`, or `approved_at`. The `green_notes` migration (migration 13) has: `id`, `minute_sheet_id`, `sequence_order`, `author_officer_id`, `content_body`, `cryptographic_hash`, `previous_hash`, `is_signed`, `digital_signature`, `created_at`. It does NOT have `organization_id`. Read the migrations before writing any repo SQL. | SQL syntax errors at runtime |

---

## Step 0 — Load Context (Mandatory)

Read every file in this list before writing any code. Then output a numbered list of exactly what you will create or modify, with one sentence per item. Do not begin implementation until you have produced that list.

```
# Migrations — actual DB schema (source of truth for column names)
aethel-core/internal/database/migrations/20260526000012_create_minute_sheets.up.sql
aethel-core/internal/database/migrations/20260526000013_create_green_notes.up.sql

# Domain types and interfaces (already defined — do not duplicate)
aethel-core/internal/domain/workflow.go
aethel-core/internal/domain/errors.go
aethel-core/internal/domain/governance.go     ← audit event constants

# Existing stub service (already written — understand before implementing repos)
aethel-core/internal/service/workflow_service.go

# Existing dispatch service (already wires minute sheet creation — do not duplicate)
aethel-core/internal/service/dispatch_service.go

# Existing handler (already written — understand the URL parameter names)
aethel-core/internal/api/handlers/workflow.go

# Existing routes (already registered — do not add workflow routes again)
aethel-core/internal/api/server.go

# Query registry pattern (follow this exact pattern for new repos)
aethel-core/internal/database/repos/dispatch_repo.go

# Existing named queries (add workflow groups here — do not duplicate existing keys)
aethel-core/internal/database/queries/queries.yaml

# Test pattern to follow (mock structs, table-driven tests, stdlib only)
aethel-core/internal/service/auth_service_test.go

# Architecture docs
CLAUDE.md
docs/guides/go-developer-guide.md
```

---

## What Already Exists — Do Not Recreate

The following are complete and working. Read them; do not modify unless a bug is found:

- `aethel-core/internal/domain/workflow.go` — `MinuteSheet`, `GreenNote`, `MinuteSheetRepository`, `GreenNoteRepository` interfaces, `MinuteSheetStatus` constants
- `aethel-core/internal/domain/errors.go` — `ErrHashChainBroken`, `ErrNotFound`, `ErrConflict`
- `aethel-core/internal/domain/governance.go` — `AuditGreenNoteAppended` constant (value: `"GREEN_NOTE_APPENDED"`)
- `aethel-core/internal/service/workflow_service.go` — `WorkflowService` struct with `GetMinuteSheet`, `ListGreenNotes`, `AppendGreenNote`, `ApproveMinuteSheet`; hash formula via `computeNoteHash`; `firstNoteAnchor = "genesis"`
- `aethel-core/internal/api/handlers/workflow.go` — `WorkflowHandler` with all four handler methods
- `aethel-core/internal/api/server.go` — All four workflow routes registered under `/dispatches/{id}/`

**Important route note**: The handler for `AppendGreenNote` and `ListGreenNotes` reads `chi.URLParam(r, "id")` as the `minuteSheetID`, not `dispatchID`. The `GetMinuteSheet` handler reads `chi.URLParam(r, "id")` as `dispatchID`. This asymmetry is already in place — do not change the handlers.

**Important service note**: `dispatch_service.go` already calls `s.minuteSheets.Create(ctx, ms)` in `DispatchService.Create`. The minute sheet auto-creation is already wired. Do not add it again.

**Missing**: Repository implementations for `MinuteSheetRepository` and `GreenNoteRepository`. Named SQL queries for those repos in `queries.yaml`. Unit tests for the hash chain. An integration test.

---

## Block A — Named Queries in queries.yaml

File: `aethel-core/internal/database/queries/queries.yaml`

The file already has a `workflow:` group with one entry (`fetch_minute_sheet_timeline`). Add two new top-level groups after the existing `workflow:` group. Do not touch any existing keys.

```yaml
  # ══════════════════════════════════════════════════════════════════════════
  # PILLAR 2 — MINUTE SHEET CRUD
  # ══════════════════════════════════════════════════════════════════════════
  minute_sheet:
    create:
      statement: |
        INSERT INTO minute_sheets (id, dispatch_id, created_at, updated_at)
        VALUES ($1, $2, NOW(), NOW())
        RETURNING id, dispatch_id, created_at, updated_at
      params:
        - "uuid"  # $1: id
        - "uuid"  # $2: dispatch_id
      timeout_ms: 3000
      required_permission: "dispatch.create"

    fetch_by_dispatch:
      statement: |
        SELECT id, dispatch_id, created_at, updated_at
        FROM minute_sheets WHERE dispatch_id = $1
      params:
        - "uuid"  # $1: dispatch_id
      timeout_ms: 3000
      required_permission: "dispatch.view"

    fetch_by_id:
      statement: |
        SELECT id, dispatch_id, created_at, updated_at
        FROM minute_sheets WHERE id = $1
      params:
        - "uuid"  # $1: id
      timeout_ms: 3000
      required_permission: "dispatch.view"

  # ══════════════════════════════════════════════════════════════════════════
  # PILLAR 2 — GREEN NOTE CRUD (INSERT-only; no UPDATE/DELETE ever)
  # ══════════════════════════════════════════════════════════════════════════
  green_note:
    fetch_last_for_update:
      statement: |
        SELECT id, minute_sheet_id, sequence_order, author_officer_id,
               content_body, cryptographic_hash, previous_hash,
               is_signed, digital_signature, created_at
        FROM green_notes
        WHERE minute_sheet_id = $1
        ORDER BY sequence_order DESC
        LIMIT 1
        FOR UPDATE
      params:
        - "uuid"  # $1: minute_sheet_id
      timeout_ms: 3000
      required_permission: "workflow.approve"

    insert:
      statement: |
        INSERT INTO green_notes
          (id, minute_sheet_id, sequence_order, author_officer_id,
           content_body, cryptographic_hash, previous_hash,
           is_signed, digital_signature, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
        RETURNING id, minute_sheet_id, sequence_order, author_officer_id,
                  content_body, cryptographic_hash, previous_hash,
                  is_signed, digital_signature, created_at
      params:
        - "uuid"    # $1: id
        - "uuid"    # $2: minute_sheet_id
        - "integer" # $3: sequence_order
        - "uuid"    # $4: author_officer_id
        - "text"    # $5: content_body
        - "varchar" # $6: cryptographic_hash
        - "varchar" # $7: previous_hash
        - "boolean" # $8: is_signed
        - "text"    # $9: digital_signature (nullable)
      timeout_ms: 3000
      required_permission: "workflow.approve"

    fetch_all:
      statement: |
        SELECT id, minute_sheet_id, sequence_order, author_officer_id,
               content_body, cryptographic_hash, previous_hash,
               is_signed, digital_signature, created_at
        FROM green_notes
        WHERE minute_sheet_id = $1
        ORDER BY sequence_order ASC
      params:
        - "uuid"  # $1: minute_sheet_id
      timeout_ms: 5000
      required_permission: "workflow.view"

    fetch_by_id:
      statement: |
        SELECT id, minute_sheet_id, sequence_order, author_officer_id,
               content_body, cryptographic_hash, previous_hash,
               is_signed, digital_signature, created_at
        FROM green_notes WHERE id = $1
      params:
        - "uuid"  # $1: id
      timeout_ms: 3000
      required_permission: "workflow.view"
```

**YAML alignment warning**: The existing file uses 2-space indentation. Match it exactly. YAML is indentation-sensitive.

---

## Block B — MinuteSheetRepo

File: `aethel-core/internal/database/repos/minute_sheet_repo.go` (create new)

The `MinuteSheetRepository` interface is defined in `domain/workflow.go`. Implement it here.

Key implementation notes:

1. `minute_sheets` has no `status`, `organization_id`, `approved_by_id`, or `approved_at` columns. The domain struct `domain.MinuteSheet` has `Status` and `OrganizationID` — populate `Status` as `domain.MinuteSheetOpen` on every scan (it is not in the DB in this schema version), and set `OrganizationID` to `app.OrgID`.
2. The `Approve` method on the interface: since there is no `status` column in this migration, implement `Approve` as a no-op that returns `nil` (the status is tracked in memory / future migration). Leave a `// TODO(migration-22): add status column` comment.
3. All queries through `qr.Get("minute_sheet.create").Stmt` etc. — no inline SQL.
4. Pattern: follow `dispatch_repo.go` exactly for struct layout, constructor, and scan helpers.

```go
package repos

import (
    "context"
    "database/sql"
    "time"

    "github.com/google/uuid"

    "aethel-core/internal/app"
    "aethel-core/internal/database"
    "aethel-core/internal/domain"
)

type MinuteSheetRepo struct {
    db *sql.DB
    q  *database.QueryRegistry
}

func NewMinuteSheetRepo(db *sql.DB, q *database.QueryRegistry) *MinuteSheetRepo {
    return &MinuteSheetRepo{db: db, q: q}
}

func (r *MinuteSheetRepo) Create(ctx context.Context, ms *domain.MinuteSheet) error {
    row := r.q.Get("minute_sheet.create").Stmt.QueryRowContext(ctx, ms.ID, ms.DispatchID)
    return scanMinuteSheet(row, ms)
}

func (r *MinuteSheetRepo) GetByDispatchID(ctx context.Context, _ uuid.UUID, dispatchID uuid.UUID) (*domain.MinuteSheet, error) {
    ms := &domain.MinuteSheet{}
    row := r.q.Get("minute_sheet.fetch_by_dispatch").Stmt.QueryRowContext(ctx, dispatchID)
    if err := scanMinuteSheet(row, ms); err != nil {
        return nil, err
    }
    return ms, nil
}

func (r *MinuteSheetRepo) GetByID(ctx context.Context, _ uuid.UUID, id uuid.UUID) (*domain.MinuteSheet, error) {
    ms := &domain.MinuteSheet{}
    row := r.q.Get("minute_sheet.fetch_by_id").Stmt.QueryRowContext(ctx, id)
    if err := scanMinuteSheet(row, ms); err != nil {
        return nil, err
    }
    return ms, nil
}

// Approve is a no-op in this schema version — minute_sheets has no status column yet.
// TODO(migration-22): add status, approved_by_id, approved_at columns, then implement.
func (r *MinuteSheetRepo) Approve(_ context.Context, _, _ uuid.UUID, _ uuid.UUID) error {
    return nil
}

func scanMinuteSheet(row *sql.Row, ms *domain.MinuteSheet) error {
    var createdAt, updatedAt time.Time
    err := row.Scan(&ms.ID, &ms.DispatchID, &createdAt, &updatedAt)
    if err == sql.ErrNoRows {
        return domain.ErrNotFound
    }
    if err != nil {
        return err
    }
    ms.CreatedAt = createdAt
    ms.UpdatedAt = updatedAt
    // Status not stored in DB yet — default to OPEN.
    ms.Status = domain.MinuteSheetOpen
    // Single-tenant: inject boot-time OrgID.
    ms.OrganizationID = app.OrgID
    return nil
}
```

---

## Block C — GreenNoteRepo

File: `aethel-core/internal/database/repos/green_note_repo.go` (create new)

This is the most critical file. Read it carefully.

### Concurrency threat model

Two officers may submit a green note at the same millisecond. Without a lock:
- Both goroutines call `GetLastByMinuteSheet` and both see `seq=5`
- Both compute `seq=6`
- One INSERT succeeds; the other violates `UNIQUE (minute_sheet_id, sequence_order)`

**Fix**: use `SELECT … FOR UPDATE` inside a transaction to lock the last-note row. The query `green_note.fetch_last_for_update` in queries.yaml already includes `FOR UPDATE`. Use a `*sql.Tx` for the entire Append operation.

### Implementation skeleton

```go
package repos

import (
    "context"
    "database/sql"

    "github.com/google/uuid"

    "aethel-core/internal/database"
    "aethel-core/internal/domain"
)

type GreenNoteRepo struct {
    db *sql.DB
    q  *database.QueryRegistry
}

func NewGreenNoteRepo(db *sql.DB, q *database.QueryRegistry) *GreenNoteRepo {
    return &GreenNoteRepo{db: db, q: q}
}

// Create implements domain.GreenNoteRepository.
// It opens a transaction, locks the last note row with SELECT FOR UPDATE,
// validates the chain, then inserts the new note — all atomically.
func (r *GreenNoteRepo) Create(ctx context.Context, note *domain.GreenNote) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Lock the tail of the chain while we inspect it.
    last, err := fetchLastInTx(ctx, tx, r.q, note.MinuteSheetID)
    if err != nil && err != domain.ErrNotFound {
        return err
    }

    if last != nil {
        // Validate: re-derive last note's hash from its stored fields.
        expected := computeGreenNoteHash(last.ContentBody, last.SequenceOrder, last.AuthorOfficerID, last.PreviousHash)
        if expected != last.CryptographicHash {
            return domain.ErrHashChainBroken
        }
    }

    // Insert the new note within the same transaction.
    stmt := tx.StmtContext(ctx, r.q.Get("green_note.insert").Stmt)
    row := stmt.QueryRowContext(ctx,
        note.ID,
        note.MinuteSheetID,
        note.SequenceOrder,
        note.AuthorOfficerID,
        note.ContentBody,
        note.CryptographicHash,
        note.PreviousHash,
        note.DigitalSignature != nil,
        note.DigitalSignature,
    )
    if err := scanGreenNote(row, note); err != nil {
        return err
    }

    return tx.Commit()
}

func (r *GreenNoteRepo) ListByMinuteSheet(ctx context.Context, _ uuid.UUID, minuteSheetID uuid.UUID) ([]domain.GreenNote, error) {
    rows, err := r.q.Get("green_note.fetch_all").Stmt.QueryContext(ctx, minuteSheetID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    return scanGreenNotes(rows)
}

func (r *GreenNoteRepo) GetLastByMinuteSheet(ctx context.Context, _ uuid.UUID, minuteSheetID uuid.UUID) (*domain.GreenNote, error) {
    // Non-transactional read (used by service layer for display, not for append).
    // For append safety, fetchLastInTx is used internally.
    row := r.q.Get("green_note.fetch_last_for_update").Stmt.QueryRowContext(ctx, minuteSheetID)
    note := &domain.GreenNote{}
    if err := scanGreenNote(row, note); err != nil {
        return nil, err
    }
    return note, nil
}

// fetchLastInTx fetches the last green note for a minute sheet using the
// provided transaction, locking the row (FOR UPDATE) to prevent concurrent appends.
func fetchLastInTx(ctx context.Context, tx *sql.Tx, q *database.QueryRegistry, minuteSheetID uuid.UUID) (*domain.GreenNote, error) {
    stmt := tx.StmtContext(ctx, q.Get("green_note.fetch_last_for_update").Stmt)
    row := stmt.QueryRowContext(ctx, minuteSheetID)
    note := &domain.GreenNote{}
    if err := scanGreenNote(row, note); err != nil {
        return nil, err
    }
    return note, nil
}

func scanGreenNote(row *sql.Row, n *domain.GreenNote) error {
    var sig *string
    err := row.Scan(
        &n.ID,
        &n.MinuteSheetID,
        &n.SequenceOrder,
        &n.AuthorOfficerID,
        &n.ContentBody,
        &n.CryptographicHash,
        &n.PreviousHash,
        new(bool),  // is_signed — not in domain struct; discard
        &sig,
        &n.CreatedAt,
    )
    if err == sql.ErrNoRows {
        return domain.ErrNotFound
    }
    if err != nil {
        return err
    }
    n.DigitalSignature = sig
    return nil
}

func scanGreenNotes(rows *sql.Rows) ([]domain.GreenNote, error) {
    var result []domain.GreenNote
    for rows.Next() {
        n := domain.GreenNote{}
        var sig *string
        err := rows.Scan(
            &n.ID,
            &n.MinuteSheetID,
            &n.SequenceOrder,
            &n.AuthorOfficerID,
            &n.ContentBody,
            &n.CryptographicHash,
            &n.PreviousHash,
            new(bool),
            &sig,
            &n.CreatedAt,
        )
        if err != nil {
            return nil, err
        }
        n.DigitalSignature = sig
        result = append(result, n)
    }
    return result, rows.Err()
}
```

**Important**: The function `computeGreenNoteHash` is defined in `workflow_service.go` in the `service` package as `computeNoteHash`. You cannot call it from `repos/` — different packages. Duplicate the formula locally in `green_note_repo.go` as an unexported `computeGreenNoteHash`:

```go
import (
    "crypto/sha256"
    "encoding/hex"
    "strconv"
)

func computeGreenNoteHash(content string, seq int, authorID uuid.UUID, prevHash string) string {
    payload := content + "|" + strconv.Itoa(seq) + "|" + authorID.String() + "|" + prevHash
    h := sha256.Sum256([]byte(payload))
    return hex.EncodeToString(h[:])
}
```

**Hash formula alignment**: The formula above uses `"|"` as a separator. The existing `workflow_service.go` `computeNoteHash` does NOT use separators — it simply concatenates: `content + strconv.Itoa(seq) + authorID.String() + prevHash`. Before implementing, read the existing `computeNoteHash` function in `workflow_service.go` line-for-line and match it exactly. The repo's chain validator must use the same formula as the service's hash producer, or every chain validation will fail. Use whichever formula is in `workflow_service.go` as the canonical one — do not invent a new formula.

---

## Block D — Wire Repos into Server

File: `aethel-core/cmd/aethel/main.go` (edit)

Read the existing `main.go` to see how other repos are constructed and injected. Then add:

```go
minuteSheetRepo := repos.NewMinuteSheetRepo(db, queries)
greenNoteRepo   := repos.NewGreenNoteRepo(db, queries)
workflowSvc     := service.NewWorkflowService(minuteSheetRepo, greenNoteRepo, auditRepo)
workflowHandler := handlers.NewWorkflowHandler(workflowSvc)
```

Also wire `minuteSheetRepo` into `DispatchService` if it is not already there. Check the `NewDispatchService` signature — it accepts a `domain.MinuteSheetRepository` parameter. Pass `minuteSheetRepo`.

**Verification gate before Block E**:

```bash
cd aethel-core && go build ./...
# Must succeed with zero errors.
```

If it fails, fix compilation before proceeding to tests.

---

## Block E — Unit Tests for Hash Chain

File: `aethel-core/internal/service/workflow_service_test.go` (create new)

Use stdlib only — no external mock libraries. Follow the exact pattern from `auth_service_test.go`:
- All mock types unexported, defined at top of file
- No `t.Parallel()` (keep tests serial for predictability)
- Table-driven where natural; direct subtests where not

### Domain struct alignment note

Before writing mocks, read `domain/workflow.go` to confirm the exact field names:
- `GreenNote.SequenceOrder` (not `Sequence`)
- `GreenNote.AuthorOfficerID` (not `AuthorUserID`)
- `GreenNote.ContentBody` (not `Content`)
- `GreenNote.DigitalSignature *string` (not `SignatureData []byte`)
- `MinuteSheet.Status` is `domain.MinuteSheetStatus`

### Mock repos

```go
package service

import (
    "context"
    "testing"
    "time"

    "github.com/google/uuid"

    "aethel-core/internal/domain"
)

// ── mock repos ─────────────────────────────────────────────────────────────

type mockMinuteSheetRepo struct {
    sheets map[uuid.UUID]*domain.MinuteSheet // keyed by ID
    approvedIDs []uuid.UUID
}

func newMockMinuteSheetRepo() *mockMinuteSheetRepo {
    return &mockMinuteSheetRepo{sheets: make(map[uuid.UUID]*domain.MinuteSheet)}
}

func (r *mockMinuteSheetRepo) Create(_ context.Context, ms *domain.MinuteSheet) error {
    r.sheets[ms.ID] = ms
    return nil
}

func (r *mockMinuteSheetRepo) GetByDispatchID(_ context.Context, _ uuid.UUID, dispatchID uuid.UUID) (*domain.MinuteSheet, error) {
    for _, ms := range r.sheets {
        if ms.DispatchID == dispatchID {
            return ms, nil
        }
    }
    return nil, domain.ErrNotFound
}

func (r *mockMinuteSheetRepo) GetByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.MinuteSheet, error) {
    ms, ok := r.sheets[id]
    if !ok {
        return nil, domain.ErrNotFound
    }
    return ms, nil
}

func (r *mockMinuteSheetRepo) Approve(_ context.Context, _, id uuid.UUID, _ uuid.UUID) error {
    r.approvedIDs = append(r.approvedIDs, id)
    if ms, ok := r.sheets[id]; ok {
        ms.Status = domain.MinuteSheetApproved
    }
    return nil
}

// mockGreenNoteRepo is an in-memory GreenNoteRepository.
// It does NOT validate the chain — the chain validation happens in the real repo.
// The service calls repo.Create with a fully-formed note; the mock just stores it.
type mockGreenNoteRepo struct {
    notes []domain.GreenNote
}

func (r *mockGreenNoteRepo) Create(_ context.Context, note *domain.GreenNote) error {
    r.notes = append(r.notes, *note)
    return nil
}

func (r *mockGreenNoteRepo) ListByMinuteSheet(_ context.Context, _ uuid.UUID, minuteSheetID uuid.UUID) ([]domain.GreenNote, error) {
    var out []domain.GreenNote
    for _, n := range r.notes {
        if n.MinuteSheetID == minuteSheetID {
            out = append(out, n)
        }
    }
    return out, nil
}

func (r *mockGreenNoteRepo) GetLastByMinuteSheet(_ context.Context, _ uuid.UUID, minuteSheetID uuid.UUID) (*domain.GreenNote, error) {
    var last *domain.GreenNote
    for i, n := range r.notes {
        if n.MinuteSheetID == minuteSheetID {
            if last == nil || n.SequenceOrder > last.SequenceOrder {
                last = &r.notes[i]
            }
        }
    }
    if last == nil {
        return nil, domain.ErrNotFound
    }
    return last, nil
}

// mockAuditRepoWf tracks written events.
type mockAuditRepoWf struct {
    events []domain.AuditEventType
}

func (r *mockAuditRepoWf) Write(_ context.Context, e *domain.AuditEntry) error {
    r.events = append(r.events, e.ActionEventType)
    return nil
}
func (r *mockAuditRepoWf) Query(_ context.Context, _ uuid.UUID, _, _ time.Time, _ domain.Page) ([]domain.AuditEntry, error) {
    return nil, nil
}
func (r *mockAuditRepoWf) VerifyChain(_ context.Context, _ uuid.UUID, _, _ time.Time) (*domain.ChainVerificationResult, error) {
    return &domain.ChainVerificationResult{Valid: true}, nil
}
```

### Required test cases

#### Test 1 — Five notes, all valid

```go
func TestHashChain_FiveNotesValid(t *testing.T) {
    msRepo    := newMockMinuteSheetRepo()
    noteRepo  := &mockGreenNoteRepo{}
    auditRepo := &mockAuditRepoWf{}
    svc       := NewWorkflowService(msRepo, noteRepo, auditRepo)

    orgID   := uuid.New()
    msID    := uuid.New()
    authorID := uuid.New()

    // Seed a minute sheet.
    ms := &domain.MinuteSheet{ID: msID, DispatchID: uuid.New(), OrganizationID: orgID, Status: domain.MinuteSheetOpen}
    _ = msRepo.Create(context.Background(), ms)

    contents := []string{"Alpha", "Bravo", "Charlie", "Delta", "Echo"}
    for i, content := range contents {
        note, err := svc.AppendGreenNote(context.Background(), orgID, msID, authorID, content, "127.0.0.1")
        if err != nil {
            t.Fatalf("note %d: unexpected error: %v", i+1, err)
        }
        if note.SequenceOrder != i+1 {
            t.Errorf("note %d: want seq=%d, got %d", i+1, i+1, note.SequenceOrder)
        }
    }

    // Verify PreviousHash of note N == CryptographicHash of note N-1.
    notes := noteRepo.notes
    if len(notes) != 5 {
        t.Fatalf("want 5 notes, got %d", len(notes))
    }
    for i := 1; i < len(notes); i++ {
        if notes[i].PreviousHash != notes[i-1].CryptographicHash {
            t.Errorf("chain broken at note %d: previousHash=%q, note[%d].hash=%q",
                i+1, notes[i].PreviousHash, i, notes[i-1].CryptographicHash)
        }
    }
}
```

#### Test 2 — Tamper at note 3, append fails

```go
func TestHashChain_TamperAtNote3(t *testing.T) {
    msRepo    := newMockMinuteSheetRepo()
    noteRepo  := &mockGreenNoteRepo{}
    auditRepo := &mockAuditRepoWf{}
    svc       := NewWorkflowService(msRepo, noteRepo, auditRepo)

    orgID    := uuid.New()
    msID     := uuid.New()
    authorID := uuid.New()

    ms := &domain.MinuteSheet{ID: msID, DispatchID: uuid.New(), OrganizationID: orgID, Status: domain.MinuteSheetOpen}
    _ = msRepo.Create(context.Background(), ms)

    for i := 0; i < 5; i++ {
        _, err := svc.AppendGreenNote(context.Background(), orgID, msID, authorID,
            "note content", "127.0.0.1")
        if err != nil {
            t.Fatalf("setup note %d: %v", i+1, err)
        }
    }

    // Tamper: corrupt note 3's hash (index 2).
    noteRepo.notes[2].CryptographicHash = "deadbeefdeadbeef"

    // Now the last note is note 5. When we try to append note 6, the service
    // fetches note 5 (the last note) and re-derives its hash. Note 5's
    // PreviousHash points to note 4's hash, which is valid. However, note 5's
    // own hash was computed from note 4's correct hash, so note 5 itself is
    // fine. The tamper at note 3 is only detectable by a full chain scan.
    //
    // If the service only validates the *last* note's hash, tamper at note 3
    // will NOT be caught on append — it is caught by the audit/verify endpoint.
    //
    // Adjust this test to reflect actual service behavior:
    // The service calls GetLastByMinuteSheet → note 5, then verifies note 5's
    // stored hash against re-computed hash. Since note 5 is untampered, no error.
    // Tamper detection for interior nodes requires the governance verify-chain endpoint.
    //
    // Test the correct invariant: corrupting the LAST note IS caught.
    noteRepo.notes[4].CryptographicHash = "tampered"

    _, err := svc.AppendGreenNote(context.Background(), orgID, msID, authorID, "note 6", "127.0.0.1")
    if err != domain.ErrHashChainBroken {
        t.Errorf("want ErrHashChainBroken, got %v", err)
    }
}
```

#### Test 3 — First note has genesis anchor

```go
func TestAppendGreenNote_FirstNote(t *testing.T) {
    msRepo    := newMockMinuteSheetRepo()
    noteRepo  := &mockGreenNoteRepo{}
    auditRepo := &mockAuditRepoWf{}
    svc       := NewWorkflowService(msRepo, noteRepo, auditRepo)

    orgID    := uuid.New()
    msID     := uuid.New()
    authorID := uuid.New()

    ms := &domain.MinuteSheet{ID: msID, DispatchID: uuid.New(), OrganizationID: orgID, Status: domain.MinuteSheetOpen}
    _ = msRepo.Create(context.Background(), ms)

    note, err := svc.AppendGreenNote(context.Background(), orgID, msID, authorID, "First note", "127.0.0.1")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if note.SequenceOrder != 1 {
        t.Errorf("want Sequence=1, got %d", note.SequenceOrder)
    }
    // The first note's PreviousHash is the sentinel anchor (see firstNoteAnchor in workflow_service.go).
    // Read workflow_service.go to confirm the exact value ("genesis" or "").
    if note.PreviousHash != firstNoteAnchor {
        t.Errorf("want PreviousHash=%q, got %q", firstNoteAnchor, note.PreviousHash)
    }
}
```

#### Test 4 — Approve writes audit entry

```go
func TestApproveMinuteSheet_WritesAuditEntry(t *testing.T) {
    msRepo    := newMockMinuteSheetRepo()
    noteRepo  := &mockGreenNoteRepo{}
    auditRepo := &mockAuditRepoWf{}
    svc       := NewWorkflowService(msRepo, noteRepo, auditRepo)

    orgID      := uuid.New()
    msID       := uuid.New()
    approverID := uuid.New()

    ms := &domain.MinuteSheet{ID: msID, DispatchID: uuid.New(), OrganizationID: orgID, Status: domain.MinuteSheetOpen}
    _ = msRepo.Create(context.Background(), ms)

    err := svc.ApproveMinuteSheet(context.Background(), orgID, msID, approverID)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    // Note: the current workflow_service.ApproveMinuteSheet delegates to repo.Approve
    // and does NOT currently write an audit entry. If the audit write is not present,
    // this test documents that gap and is expected to fail until the service is updated.
    // Update workflow_service.ApproveMinuteSheet to call s.audit.Write after s.minuteSheets.Approve.
    found := false
    for _, ev := range auditRepo.events {
        if ev == domain.AuditMinuteSheetApproved {
            found = true
            break
        }
    }
    if !found {
        t.Errorf("expected MINUTE_SHEET_APPROVED audit event, got %v", auditRepo.events)
    }
}
```

**Audit event constant gap**: The constant `domain.AuditMinuteSheetApproved` does not currently exist in `domain/governance.go`. You must add it:

```go
AuditMinuteSheetApproved AuditEventType = "MINUTE_SHEET_APPROVED"
```

Then update `WorkflowService.ApproveMinuteSheet` to call `s.audit.Write` with this event type after calling `s.minuteSheets.Approve`.

**Verification gate after Block E**:

```bash
cd aethel-core && go test -v ./internal/service/... -run TestHashChain
cd aethel-core && go test -v ./internal/service/... -run TestAppend
cd aethel-core && go test -v ./internal/service/... -run TestApprove
# All must pass.
```

---

## Block F — Integration Test

File: `aethel-core/integration/workflow_test.go` (create new; create the `integration/` directory if missing)

```go
//go:build integration

package integration

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "testing"
)
```

Test: `TestGreenNoteChain_EndToEnd`

Steps:
1. `POST /api/v1/auth/login` with the seed admin credentials from your local `.env` → capture `accessToken`.
2. `POST /api/v1/dispatches` with `direction: "INBOUND"`, valid `documentTypeId`, etc. → capture dispatch `id`.
3. `GET /api/v1/dispatches/{id}/minute-sheet` → assert `200`, body has a minute sheet (no notes yet because `GET /minute-sheet` returns only the sheet, not notes via this handler as currently implemented — adapt based on actual handler response shape).
4. `POST /api/v1/dispatches/{id}/green-notes` with `{"content": "Note One"}` → assert `201`, body has `sequenceOrder: 1`.
5. `POST /api/v1/dispatches/{id}/green-notes` with `{"content": "Note Two"}` → assert `201`, `sequenceOrder: 2`.
6. `POST /api/v1/dispatches/{id}/green-notes` with `{"content": "Note Three"}` → assert `201`, `sequenceOrder: 3`.
7. `GET /api/v1/dispatches/{id}/green-notes` → parse as `[]GreenNote`; assert 3 notes; verify `notes[1].previousHash == notes[0].cryptographicHash` and `notes[2].previousHash == notes[1].cryptographicHash`.
8. `POST /api/v1/dispatches/{id}/minute-sheet/approve` → assert `204`.

**Use `flag.String` for db-url and base-url:**

```go
var baseURL = flag.String("base-url", "http://localhost:8080", "backend base URL")
```

**Do not use a mock DB in integration tests.** The test must hit a real running backend with a real PostgreSQL. Run with:

```bash
cd aethel-core && go test -v -tags integration ./integration/... \
  -base-url="http://localhost:8080"
```

---

## Block G — Verify Queries Are Registered

File: `aethel-core/internal/database/query_registry.go` (or wherever `QueryRegistry` loads `queries.yaml`)

Read the file to confirm it loads all keys from `queries.yaml` at startup. If there is a whitelist of registered query keys, add `minute_sheet.create`, `minute_sheet.fetch_by_dispatch`, `minute_sheet.fetch_by_id`, `green_note.fetch_last_for_update`, `green_note.insert`, `green_note.fetch_all`, `green_note.fetch_by_id` to it.

If the registry loads all keys dynamically, no action required — just confirm.

---

## Sequential Execution Order

Execute blocks in this order. Do not skip verification gates.

1. **Step 0** — Read all context files; output change list
2. **Block A** — Add queries to `queries.yaml`
3. **Block B** — Write `minute_sheet_repo.go`
4. **Block C** — Write `green_note_repo.go`
5. **Block G** — Confirm query registry loads new keys
6. **Block D** — Wire repos in `main.go`; add `AuditMinuteSheetApproved` constant; update `ApproveMinuteSheet` to write audit
7. **Verification gate**: `go build ./...` must pass
8. **Block E** — Write `workflow_service_test.go`
9. **Verification gate**: `go test -v ./internal/service/... -run TestHashChain|TestAppend|TestApprove` must pass
10. **Block F** — Write `integration/workflow_test.go`
11. **Final verification**: `go build ./...` + unit tests pass

---

## Do NOT Rules

- **Never** write `UPDATE green_notes` or `DELETE FROM green_notes` anywhere.
- **Never** write inline SQL in `repos/` files (no `db.QueryRowContext(ctx, "SELECT ...")` with a literal string).
- **Never** use a mock database (e.g., `DATA-DOG/go-sqlmock`) in unit tests — use in-memory struct mocks only.
- **Never** skip the `SELECT FOR UPDATE` in `fetchLastInTx`. Without it, concurrent appends race to the same sequence number.
- **Never** add a second set of workflow routes to `server.go` — they are already registered.
- **Never** add a second minute sheet creation call in `dispatch_service.go` — it is already there.
- **Never** pass `orgID` through JWT claims — it comes from `app.OrgID` via `rbac.OrgIDFromCtx`.
- **Never** copy the hash formula from this task file without first reading `workflow_service.go computeNoteHash` — the canonical formula is in the source code, not in this task description. Match it exactly.

---

## Definition of Done

Check every item before declaring Task 12 complete.

- [ ] `cd aethel-core && go build ./...` exits 0 with no output
- [ ] `go vet ./...` exits 0 with no output
- [ ] `go test -v ./internal/service/... -run TestHashChain_FiveNotesValid` passes
- [ ] `go test -v ./internal/service/... -run TestHashChain_TamperAtNote3` passes
- [ ] `go test -v ./internal/service/... -run TestAppendGreenNote_FirstNote` passes
- [ ] `go test -v ./internal/service/... -run TestApproveMinuteSheet_WritesAuditEntry` passes
- [ ] `aethel-core/internal/database/repos/minute_sheet_repo.go` exists and implements `domain.MinuteSheetRepository`
- [ ] `aethel-core/internal/database/repos/green_note_repo.go` exists and implements `domain.GreenNoteRepository`
- [ ] No inline SQL in any `repos/` file — every query uses `r.q.Get("…").Stmt`
- [ ] `queries.yaml` has `minute_sheet.*` and `green_note.*` groups with no YAML syntax errors (`python3 -c "import yaml,sys; yaml.safe_load(sys.stdin)" < queries.yaml` exits 0)
- [ ] `domain.AuditMinuteSheetApproved` constant exists in `governance.go`
- [ ] `WorkflowService.ApproveMinuteSheet` calls `s.audit.Write` with `AuditMinuteSheetApproved`
- [ ] `GreenNoteRepo.Create` uses a database transaction with `SELECT FOR UPDATE`
- [ ] First green note has `PreviousHash = firstNoteAnchor` (value from `workflow_service.go` constant)
- [ ] `aethel-core/integration/workflow_test.go` exists with `//go:build integration` tag
- [ ] Integration test compiles: `go build -tags integration ./integration/...`
