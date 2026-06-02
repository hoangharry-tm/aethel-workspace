package repos

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"aethel-core/internal/database"
	"aethel-core/internal/domain"
)

// DispatchRepo implements domain.DispatchRepository.
// Single-tenant: orgID parameters are accepted to satisfy the interface but not used in SQL.
type DispatchRepo struct {
	db *sql.DB
	q  *database.QueryRegistry
}

func NewDispatchRepo(db *sql.DB, q *database.QueryRegistry) *DispatchRepo {
	return &DispatchRepo{db: db, q: q}
}

func (r *DispatchRepo) Create(ctx context.Context, d *domain.Dispatch) error {
	_, err := r.q.Get("dispatch.create").Stmt.ExecContext(ctx,
		d.ID,
		d.OrganizationID,
		d.TrackingNumber,
		string(d.Direction),
		d.DocumentTypeID,
		d.SenderName,
		d.SenderOrganization,
		d.RecipientName,
		d.RecipientOrganization,
		d.RecipientAddress,
		d.SubmittedByUserID,
		string(d.PriorityLevel),
		string(d.StatusState),
		d.SubjectLine,
		d.DeliveryMode,
	)
	return err
}

func (r *DispatchRepo) GetByID(ctx context.Context, _ uuid.UUID, id uuid.UUID) (*domain.Dispatch, error) {
	row := r.q.Get("dispatch.get_by_id").Stmt.QueryRowContext(ctx, id)
	return scanDispatch(row)
}

func (r *DispatchRepo) GetByTrackingNumber(ctx context.Context, _ uuid.UUID, trackingNumber string) (*domain.Dispatch, error) {
	row := r.q.Get("dispatch.get_by_tracking").Stmt.QueryRowContext(ctx, trackingNumber)
	return scanDispatch(row)
}

func (r *DispatchRepo) ListInbox(ctx context.Context, _ uuid.UUID, deptID uuid.UUID, page domain.Page) ([]domain.Dispatch, error) {
	rows, err := r.q.Get("dispatch.list_inbox").Stmt.QueryContext(ctx, deptID, page.Limit, page.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDispatches(rows)
}

func (r *DispatchRepo) ListOutbound(ctx context.Context, _ uuid.UUID, page domain.Page) ([]domain.Dispatch, error) {
	rows, err := r.q.Get("dispatch.list_outbound").Stmt.QueryContext(ctx, page.Limit, page.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDispatches(rows)
}

func (r *DispatchRepo) ListByUser(ctx context.Context, _ uuid.UUID, userID uuid.UUID, page domain.Page) ([]domain.Dispatch, error) {
	rows, err := r.q.Get("dispatch.list_by_user").Stmt.QueryContext(ctx, userID, page.Limit, page.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDispatches(rows)
}

func (r *DispatchRepo) UpdateStatus(ctx context.Context, _ uuid.UUID, id uuid.UUID, status domain.DispatchStatus) error {
	_, err := r.q.Get("dispatch.update_status").Stmt.ExecContext(ctx, id, string(status))
	return err
}

func (r *DispatchRepo) Assign(ctx context.Context, _ uuid.UUID, id uuid.UUID, userID, deptID *uuid.UUID) error {
	_, err := r.q.Get("dispatch.assign").Stmt.ExecContext(ctx, id, userID, deptID)
	return err
}

func (r *DispatchRepo) Acknowledge(ctx context.Context, _ uuid.UUID, id uuid.UUID, byUserID uuid.UUID) error {
	_, err := r.q.Get("dispatch.acknowledge").Stmt.ExecContext(ctx, id, byUserID)
	return err
}

func (r *DispatchRepo) Escalate(ctx context.Context, _ uuid.UUID, id uuid.UUID) error {
	_, err := r.q.Get("dispatch.escalate").Stmt.ExecContext(ctx, id)
	return err
}

// scanDispatch scans all dispatch columns into a domain.Dispatch struct.
func scanDispatch(row *sql.Row) (*domain.Dispatch, error) {
	d := &domain.Dispatch{}
	err := row.Scan(
		&d.ID,
		&d.OrganizationID,
		&d.TrackingNumber,
		&d.Direction,
		&d.DocumentTypeID,
		&d.SenderName,
		&d.SenderOrganization,
		&d.RecipientName,
		&d.RecipientOrganization,
		&d.RecipientAddress,
		&d.AssignedUserID,
		&d.AssignedDepartmentID,
		&d.SubmittedByUserID,
		&d.PriorityLevel,
		&d.StatusState,
		&d.SubjectLine,
		&d.DeliveryMode,
		&d.IsManuallyRouted,
		&d.OriginalSuggestedUserID,
		&d.OverdueAt,
		&d.AcknowledgedAt,
		&d.AcknowledgedByUserID,
		&d.IsEscalated,
		&d.CreatedAt,
		&d.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return d, err
}

// scanDispatches collects multiple rows into a slice.
func scanDispatches(rows *sql.Rows) ([]domain.Dispatch, error) {
	var result []domain.Dispatch
	for rows.Next() {
		d := domain.Dispatch{}
		err := rows.Scan(
			&d.ID,
			&d.OrganizationID,
			&d.TrackingNumber,
			&d.Direction,
			&d.DocumentTypeID,
			&d.SenderName,
			&d.SenderOrganization,
			&d.RecipientName,
			&d.RecipientOrganization,
			&d.RecipientAddress,
			&d.AssignedUserID,
			&d.AssignedDepartmentID,
			&d.SubmittedByUserID,
			&d.PriorityLevel,
			&d.StatusState,
			&d.SubjectLine,
			&d.DeliveryMode,
			&d.IsManuallyRouted,
			&d.OriginalSuggestedUserID,
			&d.OverdueAt,
			&d.AcknowledgedAt,
			&d.AcknowledgedByUserID,
			&d.IsEscalated,
			&d.CreatedAt,
			&d.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}
