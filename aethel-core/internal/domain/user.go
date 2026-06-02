package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	RoleAdmin     UserRole = "ADMIN"
	RoleReception UserRole = "RECEPTION"
	RoleUser      UserRole = "USER"
	RoleSysAdmin  UserRole = "SYS_ADMIN"
)

type User struct {
	ID                  uuid.UUID  `json:"id"`
	OrganizationID      uuid.UUID  `json:"organizationId"`
	DepartmentID        *uuid.UUID `json:"departmentId,omitempty"`
	EmailAddress        string     `json:"emailAddress"`
	FullName            string     `json:"fullName"`
	JobTitle            *string    `json:"jobTitle,omitempty"`
	Role                UserRole   `json:"role"`
	IsActive            bool       `json:"isActive"`
	PasswordHash        string     `json:"-"`
	FailedLoginAttempts int        `json:"-"`
	LockedUntil         *time.Time `json:"-"`
	LastLoginAt         *time.Time `json:"lastLoginAt,omitempty"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

type Session struct {
	ID               uuid.UUID `json:"id"`
	UserID           uuid.UUID `json:"userId"`
	SessionTokenHash string    `json:"-"`
	ExpiresAt        time.Time `json:"expiresAt"`
	ClientIPAddress  *string   `json:"clientIpAddress,omitempty"`
	UserAgent        *string   `json:"userAgent,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
}

type PasswordResetToken struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"userId"`
	TokenHash string     `json:"-"`
	ExpiresAt time.Time  `json:"expiresAt"`
	UsedAt    *time.Time `json:"usedAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

type Page struct {
	Limit  int
	Offset int
}

type UserRepository interface {
	GetByID(ctx context.Context, orgID, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, orgID uuid.UUID, email string) (*User, error)
	List(ctx context.Context, orgID uuid.UUID, page Page) ([]User, error)
	Create(ctx context.Context, u *User) error
	Update(ctx context.Context, u *User) error
	UpdatePasswordHash(ctx context.Context, userID uuid.UUID, hash string) error
	IncrementFailedLogins(ctx context.Context, userID uuid.UUID) error
	ResetFailedLogins(ctx context.Context, userID uuid.UUID) error
	LockUntil(ctx context.Context, userID uuid.UUID, until time.Time) error
	SetLastLogin(ctx context.Context, userID uuid.UUID) error
}

type SessionRepository interface {
	Create(ctx context.Context, s *Session) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*Session, error)
	DeleteByID(ctx context.Context, id uuid.UUID) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
	// RotateSession deletes oldID and inserts newSession in a single transaction.
	// A stolen token used a second time is rejected because the old row is gone.
	RotateSession(ctx context.Context, oldID uuid.UUID, newSession *Session) error
}

type PasswordResetRepository interface {
	Create(ctx context.Context, t *PasswordResetToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*PasswordResetToken, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error
}
