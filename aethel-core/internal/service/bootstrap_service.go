package service

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"aethel-core/internal/domain"
)

var (
	ErrAdminAlreadyExists = errors.New("admin account already exists")
	ErrWeakPassword       = errors.New("password must be at least 12 characters")
)

type BootstrapService struct {
	users domain.UserRepository
	auth  *AuthService
}

func NewBootstrapService(
	users domain.UserRepository,
	auth *AuthService,
) *BootstrapService {
	return &BootstrapService{
		users: users,
		auth:  auth,
	}
}

func (s *BootstrapService) CreateInitialAdmin(
	ctx context.Context,
	orgID uuid.UUID,
	email string,
	fullName string,
	password string,
) error {
	email = strings.TrimSpace(strings.ToLower(email))
	fullName = strings.TrimSpace(fullName)

	if email == "" {
		return errors.New("email is required")
	}

	if fullName == "" {
		return errors.New("full name is required")
	}

	if len(password) < 12 {
		return ErrWeakPassword
	}

	exists, err := s.users.AdminExists(ctx)
	if err != nil {
		return err
	}

	if exists {
		return ErrAdminAlreadyExists
	}

	hash, err := s.auth.hashPassword(password)
	if err != nil {
		return err
	}

	jobTitle := "System Administrator"

	user := &domain.User{
		ID:             uuid.New(),
		OrganizationID: orgID,
		EmailAddress:   email,
		FullName:       fullName,
		JobTitle:       &jobTitle,
		Role:           domain.RoleSysAdmin,
		IsActive:       true,
		PasswordHash:   hash,
	}

	return s.users.Create(ctx, user)
}
