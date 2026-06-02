package repos

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"aethel-core/internal/database"
	"aethel-core/internal/domain"
)

// DispatchEventRepo implements domain.DispatchEventRepository.
type DispatchEventRepo struct {
	db *sql.DB
	q  *database.QueryRegistry
}

func NewDispatchEventRepo(db *sql.DB, q *database.QueryRegistry) *DispatchEventRepo {
	return &DispatchEventRepo{db: db, q: q}
}

func (r *DispatchEventRepo) Create(ctx context.Context, e *domain.DispatchEvent) error {
	_, err := r.q.Get("dispatch_event.create").Stmt.ExecContext(ctx,
		e.ID,
		e.DispatchID,
		e.EventType,
		e.ActorUserID,
		e.ToUserID,       // target_user_id
		e.ToDeptID,       // target_department_id
		e.Metadata,
	)
	return err
}

func (r *DispatchEventRepo) ListByDispatch(ctx context.Context, _ uuid.UUID, dispatchID uuid.UUID) ([]domain.DispatchEvent, error) {
	rows, err := r.q.Get("dispatch_event.list_for_dispatch").Stmt.QueryContext(ctx, dispatchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.DispatchEvent
	for rows.Next() {
		e := domain.DispatchEvent{}
		if err := rows.Scan(
			&e.ID,
			&e.DispatchID,
			&e.EventType,
			&e.ActorUserID,
			&e.ToUserID,   // target_user_id
			&e.ToDeptID,   // target_department_id
			&e.Metadata,
			&e.CreatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}
