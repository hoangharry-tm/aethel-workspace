package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"

	"aethel-core/internal/domain"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func ptr[T any](v T) *T { return &v }

// makeHash produces a PHC-format Argon2id hash matching auth_service.go defaults.
func makeHash(password string) string {
	salt := make([]byte, saltLength)
	// Use a fixed salt for deterministic test hashes.
	for i := range salt {
		salt[i] = byte(i + 1)
	}
	hash := argon2.IDKey([]byte(password), salt,
		defaultIterations, defaultMemoryKiB, defaultParallelism, keyLength)
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		defaultMemoryKiB, defaultIterations, defaultParallelism, b64Salt, b64Hash)
}

// ── mock repos ────────────────────────────────────────────────────────────────

// mockUserRepo is a minimal in-memory UserRepository for unit tests.
type mockUserRepo struct {
	users                map[string]*domain.User // keyed by email
	incrementCalledFor   []uuid.UUID
	lockCalledFor        []uuid.UUID
	resetCalledFor       []uuid.UUID
	setLastLoginCalledFor []uuid.UUID
}

func newMockUserRepo(users ...*domain.User) *mockUserRepo {
	r := &mockUserRepo{users: make(map[string]*domain.User)}
	for _, u := range users {
		r.users[u.EmailAddress] = u
	}
	return r
}

func (r *mockUserRepo) GetByEmail(_ context.Context, _ uuid.UUID, email string) (*domain.User, error) {
	u, ok := r.users[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func (r *mockUserRepo) GetByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.User, error) {
	for _, u := range r.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *mockUserRepo) List(_ context.Context, _ uuid.UUID, _ domain.Page) ([]domain.User, error) {
	return nil, nil
}

func (r *mockUserRepo) Create(_ context.Context, _ *domain.User) error { return nil }

func (r *mockUserRepo) Update(_ context.Context, _ *domain.User) error { return nil }

func (r *mockUserRepo) UpdatePasswordHash(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

func (r *mockUserRepo) IncrementFailedLogins(_ context.Context, userID uuid.UUID) error {
	r.incrementCalledFor = append(r.incrementCalledFor, userID)
	return nil
}

func (r *mockUserRepo) ResetFailedLogins(_ context.Context, userID uuid.UUID) error {
	r.resetCalledFor = append(r.resetCalledFor, userID)
	return nil
}

func (r *mockUserRepo) LockUntil(_ context.Context, userID uuid.UUID, _ time.Time) error {
	r.lockCalledFor = append(r.lockCalledFor, userID)
	return nil
}

func (r *mockUserRepo) SetLastLogin(_ context.Context, userID uuid.UUID) error {
	r.setLastLoginCalledFor = append(r.setLastLoginCalledFor, userID)
	return nil
}

func (r *mockUserRepo) AdminExists(_ context.Context) (bool, error) { return false, nil }

// mockSessionRepo is a minimal in-memory SessionRepository for unit tests.
type mockSessionRepo struct {
	sessions            map[string]*domain.Session // keyed by token hash
	rotateCalledOldID   *uuid.UUID
	rotateCalledNewSess *domain.Session
	deletedByID         *uuid.UUID
}

func newMockSessionRepo() *mockSessionRepo {
	return &mockSessionRepo{sessions: make(map[string]*domain.Session)}
}

func (r *mockSessionRepo) Create(_ context.Context, s *domain.Session) error {
	r.sessions[s.SessionTokenHash] = s
	return nil
}

func (r *mockSessionRepo) GetByTokenHash(_ context.Context, tokenHash string) (*domain.Session, error) {
	s, ok := r.sessions[tokenHash]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return s, nil
}

func (r *mockSessionRepo) DeleteByID(_ context.Context, id uuid.UUID) error {
	r.deletedByID = &id
	for k, s := range r.sessions {
		if s.ID == id {
			delete(r.sessions, k)
			return nil
		}
	}
	return nil
}

func (r *mockSessionRepo) DeleteByUserID(_ context.Context, _ uuid.UUID) error { return nil }

func (r *mockSessionRepo) RotateSession(_ context.Context, oldID uuid.UUID, newSess *domain.Session) error {
	r.rotateCalledOldID = &oldID
	r.rotateCalledNewSess = newSess
	// Simulate the rotation: remove old session (by ID), insert new.
	for k, s := range r.sessions {
		if s.ID == oldID {
			delete(r.sessions, k)
			break
		}
	}
	r.sessions[newSess.SessionTokenHash] = newSess
	return nil
}

// mockPwResetRepo is a no-op PasswordResetRepository for unit tests.
type mockPwResetRepo struct{}

func (r *mockPwResetRepo) Create(_ context.Context, _ *domain.PasswordResetToken) error {
	return nil
}
func (r *mockPwResetRepo) GetByTokenHash(_ context.Context, _ string) (*domain.PasswordResetToken, error) {
	return nil, domain.ErrNotFound
}
func (r *mockPwResetRepo) MarkUsed(_ context.Context, _ uuid.UUID) error { return nil }

// mockAuditRepo is a no-op AuditRepository for unit tests.
type mockAuditRepo struct{}

func (r *mockAuditRepo) Write(_ context.Context, _ *domain.AuditEntry) error { return nil }
func (r *mockAuditRepo) Query(_ context.Context, _ uuid.UUID, _, _ time.Time, _ domain.Page) ([]domain.AuditEntry, error) {
	return nil, nil
}
func (r *mockAuditRepo) VerifyChain(_ context.Context, _ uuid.UUID, _, _ time.Time) (*domain.ChainVerificationResult, error) {
	return &domain.ChainVerificationResult{Valid: true}, nil
}

// ── test cases ────────────────────────────────────────────────────────────────

func TestLogin_HappyPath(t *testing.T) {
	const email = "alice@example.com"
	const password = "correct-password"

	userID := uuid.New()
	orgID := uuid.New()
	user := &domain.User{
		ID:             userID,
		OrganizationID: orgID,
		EmailAddress:   email,
		FullName:       "Alice",
		Role:           domain.RoleAdmin,
		IsActive:       true,
		PasswordHash:   makeHash(password),
	}

	userRepo := newMockUserRepo(user)
	sessionRepo := newMockSessionRepo()
	svc := NewAuthService(userRepo, sessionRepo, &mockPwResetRepo{}, &mockAuditRepo{})

	result, err := svc.Login(context.Background(), orgID, email, password, "127.0.0.1", "TestUA")
	if err != nil {
		t.Fatalf("Login: unexpected error: %v", err)
	}
	if result.AccessToken == "" {
		t.Error("expected non-empty AccessToken")
	}
	if result.RefreshToken == "" {
		t.Error("expected non-empty RefreshToken")
	}
	if result.User == nil || result.User.ID != userID {
		t.Error("expected User to be populated with correct ID")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	const email = "bob@example.com"

	userID := uuid.New()
	orgID := uuid.New()
	user := &domain.User{
		ID:             userID,
		OrganizationID: orgID,
		EmailAddress:   email,
		FullName:       "Bob",
		Role:           domain.RoleReception,
		IsActive:       true,
		PasswordHash:   makeHash("correct-password"),
	}

	userRepo := newMockUserRepo(user)
	sessionRepo := newMockSessionRepo()
	svc := NewAuthService(userRepo, sessionRepo, &mockPwResetRepo{}, &mockAuditRepo{})

	_, err := svc.Login(context.Background(), orgID, email, "wrong-password", "127.0.0.1", "TestUA")
	if err != domain.ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized, got: %v", err)
	}
	if len(userRepo.incrementCalledFor) == 0 {
		t.Error("expected IncrementFailedLogins to be called")
	}
	if userRepo.incrementCalledFor[0] != userID {
		t.Errorf("IncrementFailedLogins called for wrong user: got %v want %v",
			userRepo.incrementCalledFor[0], userID)
	}
}

func TestLogin_AccountLocked(t *testing.T) {
	const email = "charlie@example.com"
	const password = "any-password"

	userID := uuid.New()
	orgID := uuid.New()
	user := &domain.User{
		ID:             userID,
		OrganizationID: orgID,
		EmailAddress:   email,
		FullName:       "Charlie",
		Role:           domain.RoleUser,
		IsActive:       true,
		PasswordHash:   makeHash(password),
		LockedUntil:    ptr(time.Now().Add(1 * time.Hour)),
	}

	userRepo := newMockUserRepo(user)
	sessionRepo := newMockSessionRepo()
	svc := NewAuthService(userRepo, sessionRepo, &mockPwResetRepo{}, &mockAuditRepo{})

	// Should fail with ErrAccountLocked before even checking the password (Argon2id).
	_, err := svc.Login(context.Background(), orgID, email, password, "127.0.0.1", "TestUA")
	if err != domain.ErrAccountLocked {
		t.Fatalf("expected ErrAccountLocked, got: %v", err)
	}
	// IncrementFailedLogins must NOT be called when the account is locked.
	if len(userRepo.incrementCalledFor) != 0 {
		t.Error("IncrementFailedLogins should not be called for a locked account")
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	orgID := uuid.New()
	userRepo := newMockUserRepo() // no users
	sessionRepo := newMockSessionRepo()
	svc := NewAuthService(userRepo, sessionRepo, &mockPwResetRepo{}, &mockAuditRepo{})

	_, err := svc.Login(context.Background(), orgID, "nobody@example.com", "password", "127.0.0.1", "TestUA")
	// Service must not reveal whether the email exists — must return ErrUnauthorized, not ErrNotFound.
	if err != domain.ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized (not ErrNotFound), got: %v", err)
	}
}

func TestRefreshSession_ValidToken(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	user := &domain.User{
		ID:             userID,
		OrganizationID: orgID,
		EmailAddress:   "dave@example.com",
		FullName:       "Dave",
		Role:           domain.RoleUser,
		IsActive:       true,
		PasswordHash:   makeHash("any"),
	}

	userRepo := newMockUserRepo(user)
	sessionRepo := newMockSessionRepo()
	svc := NewAuthService(userRepo, sessionRepo, &mockPwResetRepo{}, &mockAuditRepo{})

	// First, login to get a refresh token.
	result, err := svc.Login(context.Background(), orgID, user.EmailAddress, "any", "127.0.0.1", "UA")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	// Now refresh.
	refreshed, err := svc.RefreshSession(context.Background(), result.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshSession: unexpected error: %v", err)
	}
	if refreshed.AccessToken == "" {
		t.Error("expected non-empty AccessToken after refresh")
	}
	// RotateSession must have been called (token rotation).
	if sessionRepo.rotateCalledOldID == nil {
		t.Error("expected RotateSession to be called")
	}
}

func TestRefreshSession_ExpiredToken(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	// GetByTokenHash returns ErrNotFound for any hash → token unknown/expired.

	svc := NewAuthService(newMockUserRepo(), sessionRepo, &mockPwResetRepo{}, &mockAuditRepo{})

	_, err := svc.RefreshSession(context.Background(), "bogus-refresh-token")
	if err != domain.ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized for expired/unknown token, got: %v", err)
	}
}

func TestLogout_RevokesSession(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	user := &domain.User{
		ID:             userID,
		OrganizationID: orgID,
		EmailAddress:   "eve@example.com",
		FullName:       "Eve",
		Role:           domain.RoleReception,
		IsActive:       true,
		PasswordHash:   makeHash("password"),
	}

	userRepo := newMockUserRepo(user)
	sessionRepo := newMockSessionRepo()
	svc := NewAuthService(userRepo, sessionRepo, &mockPwResetRepo{}, &mockAuditRepo{})

	// Login to establish a session.
	result, err := svc.Login(context.Background(), orgID, user.EmailAddress, "password", "127.0.0.1", "UA")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	// Record the session ID before logout.
	tokenHash := hashToken(result.RefreshToken)
	sess, err := sessionRepo.GetByTokenHash(context.Background(), tokenHash)
	if err != nil {
		t.Fatalf("GetByTokenHash: %v", err)
	}
	sessionID := sess.ID

	// Logout should call GetByTokenHash then DeleteByID.
	if err := svc.Logout(context.Background(), orgID, userID, result.RefreshToken, "127.0.0.1", "UA"); err != nil {
		t.Fatalf("Logout: %v", err)
	}

	// Verify the session was deleted by checking deletedByID.
	if sessionRepo.deletedByID == nil {
		t.Fatal("expected DeleteByID to be called after Logout")
	}
	if *sessionRepo.deletedByID != sessionID {
		t.Errorf("DeleteByID called with wrong session ID: got %v want %v",
			*sessionRepo.deletedByID, sessionID)
	}

	// Verify the session is gone from the store.
	_, err = sessionRepo.GetByTokenHash(context.Background(), tokenHash)
	if err != domain.ErrNotFound {
		t.Errorf("expected session to be gone after logout, got: %v", err)
	}
}
