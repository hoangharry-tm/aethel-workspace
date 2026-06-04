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

// MinuteSheetRepo implements domain.MinuteSheetRepository.
// Single-tenant: the minute_sheets table has no organization_id column in this schema version.
// OrganizationID is injected from app.OrgID on every scan.
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
