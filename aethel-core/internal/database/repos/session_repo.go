package repos

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"aethel-core/internal/domain"
)

// SessionRepo implements domain.SessionRepository backed by PostgreSQL.
type SessionRepo struct {
	db *sql.DB
}

func NewSessionRepo(db *sql.DB) *SessionRepo {
	return &SessionRepo{db: db}
}

func (r *SessionRepo) Create(ctx context.Context, s *domain.Session) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO user_sessions (id, user_id, session_token_hash, expires_at, client_ip_address, user_agent, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,now())
	`, s.ID, s.UserID, s.SessionTokenHash, s.ExpiresAt, s.ClientIPAddress, s.UserAgent)
	return err
}

func (r *SessionRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.Session, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, session_token_hash, expires_at, client_ip_address, user_agent, created_at
		FROM user_sessions WHERE session_token_hash = $1
	`, tokenHash)

	s := &domain.Session{}
	var ip, ua sql.NullString
	err := row.Scan(&s.ID, &s.UserID, &s.SessionTokenHash, &s.ExpiresAt, &ip, &ua, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if ip.Valid {
		s.ClientIPAddress = &ip.String
	}
	if ua.Valid {
		s.UserAgent = &ua.String
	}
	return s, nil
}

func (r *SessionRepo) DeleteByID(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM user_sessions WHERE id=$1`, id)
	return err
}

func (r *SessionRepo) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM user_sessions WHERE user_id=$1`, userID)
	return err
}

// RotateSession atomically deletes the old session and inserts the new one.
// If the old token is presented a second time, the row is gone and GetByTokenHash returns ErrNotFound.
func (r *SessionRepo) RotateSession(ctx context.Context, oldID uuid.UUID, newSession *domain.Session) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx, `DELETE FROM user_sessions WHERE id=$1`, oldID); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO user_sessions (id, user_id, session_token_hash, expires_at, client_ip_address, user_agent, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,now())
	`, newSession.ID, newSession.UserID, newSession.SessionTokenHash, newSession.ExpiresAt,
		newSession.ClientIPAddress, newSession.UserAgent); err != nil {
		return err
	}

	return tx.Commit()
}
