package repos

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"aethel-core/internal/database"
	"aethel-core/internal/domain"
)

// NotificationRepo implements domain.NotificationRepository backed by PostgreSQL.
type NotificationRepo struct {
	db *sql.DB
	q  *database.QueryRegistry
}

func NewNotificationRepo(db *sql.DB, q *database.QueryRegistry) *NotificationRepo {
	return &NotificationRepo{db: db, q: q}
}

// ListByUser returns a paginated list of notifications for the given user.
func (r *NotificationRepo) ListByUser(ctx context.Context, orgID, userID uuid.UUID, unreadOnly bool, page domain.Page) ([]domain.Notification, error) {
	// Convert unreadOnly bool to a nullable boolean for the SQL query.
	// When unreadOnly is false we pass NULL so the WHERE clause is skipped.
	var unreadParam interface{}
	if unreadOnly {
		unreadParam = true
	}

	rows, err := r.q.Get("notifications.list_by_user").Stmt.QueryContext(
		ctx,
		orgID,
		userID,
		unreadParam,
		page.Limit,
		page.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []domain.Notification
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, *n)
	}
	return notifications, rows.Err()
}

// UnreadCount returns the number of unread notifications for the user.
func (r *NotificationRepo) UnreadCount(ctx context.Context, orgID, userID uuid.UUID) (int, error) {
	var count int
	err := r.q.Get("notifications.unread_count").Stmt.QueryRowContext(ctx, orgID, userID).Scan(&count)
	return count, err
}

// MarkRead marks a single notification as read for the given user.
func (r *NotificationRepo) MarkRead(ctx context.Context, orgID, notificationID, userID uuid.UUID) error {
	result, err := r.q.Get("notifications.mark_read").Stmt.ExecContext(ctx, orgID, notificationID, userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// MarkAllRead marks all unread notifications for the user as read.
func (r *NotificationRepo) MarkAllRead(ctx context.Context, orgID, userID uuid.UUID) error {
	_, err := r.q.Get("notifications.mark_all_read").Stmt.ExecContext(ctx, orgID, userID)
	return err
}

// ── row scanner ──────────────────────────────────────────────────────────────

type notificationScanner interface {
	Scan(dest ...any) error
}

func scanNotification(s notificationScanner) (*domain.Notification, error) {
	var n domain.Notification
	err := s.Scan(
		&n.ID,
		&n.UserID,
		&n.Title,
		&n.Body,
		&n.Type,
		&n.IsRead,
		&n.RelatedDispatchID,
		&n.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &n, nil
}
