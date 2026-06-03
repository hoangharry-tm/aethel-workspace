package repos

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"aethel-core/internal/domain"
)

// UserRepo implements domain.UserRepository backed by PostgreSQL.
type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) GetByID(ctx context.Context, _ uuid.UUID, id uuid.UUID) (*domain.User, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, organization_id, department_id, email_address, full_name, job_title,
		       role, is_active, password_hash, failed_login_attempts, locked_until, last_login_at,
		       created_at, updated_at
		FROM users WHERE id = $1
	`, id)
	return scanUser(row)
}

func (r *UserRepo) GetByEmail(ctx context.Context, _ uuid.UUID, email string) (*domain.User, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, organization_id, department_id, email_address, full_name, job_title,
		       role, is_active, password_hash, failed_login_attempts, locked_until, last_login_at,
		       created_at, updated_at
		FROM users WHERE email_address = $1
	`, email)
	return scanUser(row)
}

func (r *UserRepo) List(ctx context.Context, _ uuid.UUID, page domain.Page) ([]domain.User, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, organization_id, department_id, email_address, full_name, job_title,
		       role, is_active, password_hash, failed_login_attempts, locked_until, last_login_at,
		       created_at, updated_at
		FROM users ORDER BY full_name ASC LIMIT $1 OFFSET $2
	`, page.Limit, page.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := scanUserRow(rows, &u); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepo) Create(ctx context.Context, u *domain.User) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users (id, organization_id, department_id, email_address, full_name, job_title,
		                   role, is_active, password_hash, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,now(),now())
	`, u.ID, u.OrganizationID, u.DepartmentID, u.EmailAddress, u.FullName, u.JobTitle,
		string(u.Role), u.IsActive, u.PasswordHash)
	return err
}

func (r *UserRepo) Update(ctx context.Context, u *domain.User) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users
		SET full_name=$2, job_title=$3, role=$4, is_active=$5, department_id=$6, updated_at=now()
		WHERE id=$1
	`, u.ID, u.FullName, u.JobTitle, string(u.Role), u.IsActive, u.DepartmentID)
	return err
}

func (r *UserRepo) UpdatePasswordHash(ctx context.Context, userID uuid.UUID, hash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET password_hash=$2, updated_at=now() WHERE id=$1`, userID, hash)
	return err
}

func (r *UserRepo) IncrementFailedLogins(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET failed_login_attempts = failed_login_attempts + 1, updated_at=now() WHERE id=$1`,
		userID)
	return err
}

func (r *UserRepo) ResetFailedLogins(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET failed_login_attempts=0, locked_until=NULL, updated_at=now() WHERE id=$1`,
		userID)
	return err
}

func (r *UserRepo) LockUntil(ctx context.Context, userID uuid.UUID, until time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET locked_until=$2, updated_at=now() WHERE id=$1`, userID, until)
	return err
}

func (r *UserRepo) SetLastLogin(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET last_login_at=now(), updated_at=now() WHERE id=$1`, userID)
	return err
}

func scanUser(row *sql.Row) (*domain.User, error) {
	u := &domain.User{}
	var jobTitle sql.NullString
	var deptID uuid.NullUUID
	var lockedUntil, lastLoginAt sql.NullTime
	err := row.Scan(
		&u.ID, &u.OrganizationID, &deptID, &u.EmailAddress, &u.FullName, &jobTitle,
		&u.Role, &u.IsActive, &u.PasswordHash, &u.FailedLoginAttempts, &lockedUntil, &lastLoginAt,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if deptID.Valid {
		u.DepartmentID = &deptID.UUID
	}
	if jobTitle.Valid {
		u.JobTitle = &jobTitle.String
	}
	if lockedUntil.Valid {
		u.LockedUntil = &lockedUntil.Time
	}
	if lastLoginAt.Valid {
		u.LastLoginAt = &lastLoginAt.Time
	}
	return u, nil
}

func scanUserRow(rows *sql.Rows, u *domain.User) error {
	var jobTitle sql.NullString
	var deptID uuid.NullUUID
	var lockedUntil, lastLoginAt sql.NullTime
	err := rows.Scan(
		&u.ID, &u.OrganizationID, &deptID, &u.EmailAddress, &u.FullName, &jobTitle,
		&u.Role, &u.IsActive, &u.PasswordHash, &u.FailedLoginAttempts, &lockedUntil, &lastLoginAt,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if deptID.Valid {
		u.DepartmentID = &deptID.UUID
	}
	if jobTitle.Valid {
		u.JobTitle = &jobTitle.String
	}
	if lockedUntil.Valid {
		u.LockedUntil = &lockedUntil.Time
	}
	if lastLoginAt.Valid {
		u.LastLoginAt = &lastLoginAt.Time
	}
	return nil
}

func (r *UserRepo) AdminExists(ctx context.Context) (bool, error) {
	const q = `
	SELECT EXISTS (
		SELECT 1
		FROM users
		WHERE role IN ('ADMIN', 'SYS_ADMIN')
		AND is_active = true
	)
	`

	var exists bool
	err := r.db.QueryRowContext(ctx, q).Scan(&exists)
	return exists, err
}