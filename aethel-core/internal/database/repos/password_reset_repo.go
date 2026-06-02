package repos

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"aethel-core/internal/domain"
)

// PasswordResetRepo implements domain.PasswordResetRepository.
type PasswordResetRepo struct {
	db *sql.DB
}

func NewPasswordResetRepo(db *sql.DB) *PasswordResetRepo {
	return &PasswordResetRepo{db: db}
}

func (r *PasswordResetRepo) Create(ctx context.Context, t *domain.PasswordResetToken) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1,$2,$3,$4,now())
	`, t.ID, t.UserID, t.TokenHash, t.ExpiresAt)
	return err
}

func (r *PasswordResetRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.PasswordResetToken, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, token_hash, expires_at, used_at, created_at
		FROM password_reset_tokens WHERE token_hash = $1
	`, tokenHash)

	t := &domain.PasswordResetToken{}
	var usedAt sql.NullTime
	err := row.Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &usedAt, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if usedAt.Valid {
		t.UsedAt = &usedAt.Time
	}
	return t, nil
}

func (r *PasswordResetRepo) MarkUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE password_reset_tokens SET used_at=now() WHERE id=$1`, id)
	return err
}
