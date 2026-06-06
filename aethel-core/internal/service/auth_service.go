package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"

	"aethel-core/internal/audit"
	"aethel-core/internal/blueprint"
	"aethel-core/internal/domain"
)

const (
	// defaultMemoryKiB, defaultIterations, defaultParallelism mirror the blueprint
	// defaults and are used by tests via makeHash(). The live service reads these
	// values from cfg (populated from the blueprint) so they are never read at
	// runtime by AuthService methods.
	defaultMemoryKiB   = 65536
	defaultIterations  = 3
	defaultParallelism = 4

	saltLength       = 16
	keyLength        = 32
	lockoutThreshold = 5
	lockoutDuration  = 15 * time.Minute
)

type AuthService struct {
	users    domain.UserRepository
	sessions domain.SessionRepository
	pwReset  domain.PasswordResetRepository
	audit    audit.Writer
	cfg      blueprint.AuthConfig
}

func NewAuthService(
	users domain.UserRepository,
	sessions domain.SessionRepository,
	pwReset domain.PasswordResetRepository,
	auditWriter audit.Writer,
	cfg blueprint.AuthConfig,
) *AuthService {
	cfg.SetDefaults()
	return &AuthService{
		users:    users,
		sessions: sessions,
		pwReset:  pwReset,
		audit:    auditWriter,
		cfg:      cfg,
	}
}

type LoginResult struct {
	AccessToken  string       `json:"accessToken"`
	RefreshToken string       `json:"refreshToken"`
	User         *domain.User `json:"user"`
}

func (s *AuthService) Login(ctx context.Context, orgID uuid.UUID, email, password, ip, ua string) (*LoginResult, error) {
	user, err := s.users.GetByEmail(ctx, orgID, email)
	if err != nil {
		// Do not reveal whether the email exists — return the same error as wrong password.
		_ = s.writeAudit(ctx, orgID, nil, domain.AuditUserLoginFailed, nil, ip, ua)
		return nil, domain.ErrUnauthorized
	}

	if !user.IsActive {
		_ = s.writeAudit(ctx, orgID, &user.ID, domain.AuditUserLoginFailed, nil, ip, ua)
		return nil, domain.ErrUnauthorized
	}

	// Lockout check runs before Argon2id to prevent timing side-channels.
	if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
		return nil, domain.ErrAccountLocked
	}

	if !s.verifyPassword(password, user.PasswordHash) {
		_ = s.users.IncrementFailedLogins(ctx, user.ID)
		// Lock the account when the threshold is crossed.
		if user.FailedLoginAttempts+1 >= lockoutThreshold {
			_ = s.users.LockUntil(ctx, user.ID, time.Now().Add(lockoutDuration))
		}
		_ = s.writeAudit(ctx, orgID, &user.ID, domain.AuditUserLoginFailed, nil, ip, ua)
		return nil, domain.ErrUnauthorized
	}

	_ = s.users.ResetFailedLogins(ctx, user.ID)
	_ = s.users.SetLastLogin(ctx, user.ID)

	accessToken, err := s.issueAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}

	refreshToken, tokenHash, err := s.generateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	expiresAt := time.Now().Add(s.refreshTokenTTL())
	session := &domain.Session{
		ID:               uuid.New(),
		UserID:           user.ID,
		SessionTokenHash: tokenHash,
		ExpiresAt:        expiresAt,
		ClientIPAddress:  &ip,
		UserAgent:        &ua,
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	_ = s.writeAudit(ctx, orgID, &user.ID, domain.AuditUserLogin, nil, ip, ua)

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

func (s *AuthService) RefreshSession(ctx context.Context, rawRefreshToken string) (*LoginResult, error) {
	tokenHash := hashToken(rawRefreshToken)

	session, err := s.sessions.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	if time.Now().After(session.ExpiresAt) {
		_ = s.sessions.DeleteByID(ctx, session.ID)
		return nil, domain.ErrUnauthorized
	}

	user, err := s.users.GetByID(ctx, uuid.UUID{}, session.UserID)
	if err != nil || !user.IsActive {
		return nil, domain.ErrUnauthorized
	}

	newRefresh, newHash, err := s.generateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	newSession := &domain.Session{
		ID:               uuid.New(),
		UserID:           user.ID,
		SessionTokenHash: newHash,
		ExpiresAt:        time.Now().Add(s.refreshTokenTTL()),
		ClientIPAddress:  session.ClientIPAddress,
		UserAgent:        session.UserAgent,
	}

	// Atomic: deletes old session and inserts new one in a single transaction.
	// If a stolen token is replayed after rotation, the old row is gone → ErrUnauthorized.
	if err := s.sessions.RotateSession(ctx, session.ID, newSession); err != nil {
		return nil, fmt.Errorf("rotate session: %w", err)
	}

	accessToken, err := s.issueAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: newRefresh,
		User:         user,
	}, nil
}

// Logout invalidates the specific session identified by rawRefreshToken.
// Passing an empty rawRefreshToken skips session deletion (graceful degradation).
func (s *AuthService) Logout(ctx context.Context, orgID, userID uuid.UUID, rawRefreshToken, ip, ua string) error {
	if rawRefreshToken != "" {
		tokenHash := hashToken(rawRefreshToken)
		if session, err := s.sessions.GetByTokenHash(ctx, tokenHash); err == nil {
			_ = s.sessions.DeleteByID(ctx, session.ID)
		}
	}
	_ = s.writeAudit(ctx, orgID, &userID, domain.AuditUserLogout, nil, ip, ua)
	return nil
}

func (s *AuthService) RequestPasswordReset(ctx context.Context, orgID uuid.UUID, email string) error {
	user, err := s.users.GetByEmail(ctx, orgID, email)
	if err != nil {
		return nil // avoid user enumeration
	}

	token, tokenHash, err := s.generateRefreshToken()
	if err != nil {
		return err
	}
	_ = token // production: send via email

	prt := &domain.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	return s.pwReset.Create(ctx, prt)
}

func (s *AuthService) ConfirmPasswordReset(ctx context.Context, token, newPassword string) error {
	tokenHash := hashToken(token)
	prt, err := s.pwReset.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return domain.ErrUnauthorized
	}
	if prt.UsedAt != nil || time.Now().After(prt.ExpiresAt) {
		return domain.ErrUnauthorized
	}

	hash, err := s.hashPassword(newPassword)
	if err != nil {
		return err
	}

	if err := s.users.UpdatePasswordHash(ctx, prt.UserID, hash); err != nil {
		return err
	}
	return s.pwReset.MarkUsed(ctx, prt.ID)
}

// issueAccessToken issues a signed JWT. The org claim is intentionally absent:
// this is a single-tenant system and including it adds surface area with no benefit.
func (s *AuthService) issueAccessToken(user *domain.User) (string, error) {
	secret := os.Getenv("AETHEL_JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-in-production"
	}

	now := time.Now()
	ttl := time.Duration(s.cfg.AccessTokenTTLMin) * time.Minute
	claims := jwt.MapClaims{
		"sub":  user.ID.String(),
		"role": string(user.Role),
		"iat":  now.Unix(),
		"exp":  now.Add(ttl).Unix(),
		"jti":  uuid.New().String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// AccessTokenTTL returns the configured access token lifetime as a Duration.
// Called by the HTTP handler to populate the expires_in response field.
func (s *AuthService) AccessTokenTTL() time.Duration {
	return time.Duration(s.cfg.AccessTokenTTLMin) * time.Minute
}

// accessTokenTTL is the internal alias used by issueAccessToken.
func (s *AuthService) accessTokenTTL() time.Duration {
	return s.AccessTokenTTL()
}

// refreshTokenTTL returns the configured refresh token / session lifetime as a Duration.
func (s *AuthService) refreshTokenTTL() time.Duration {
	return time.Duration(s.cfg.RefreshTokenTTLDays) * 24 * time.Hour
}

func (s *AuthService) hashPassword(password string) (string, error) {
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	memory := s.cfg.Argon2MemoryKiB
	iterations := s.cfg.Argon2Iterations
	parallelism := s.cfg.Argon2Parallelism

	hash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, keyLength)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		memory, iterations, parallelism, b64Salt, b64Hash,
	), nil
}

func (s *AuthService) verifyPassword(password, phc string) bool {
	var version int
	var memory, iterations uint32
	var parallelism uint8
	var b64Salt, b64Hash string

	_, err := fmt.Sscanf(phc,
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s",
		&version, &memory, &iterations, &parallelism, &b64Salt,
	)
	if err != nil {
		return false
	}

	for i := len(b64Salt) - 1; i >= 0; i-- {
		if b64Salt[i] == '$' {
			b64Hash = b64Salt[i+1:]
			b64Salt = b64Salt[:i]
			break
		}
	}

	salt, err := base64.RawStdEncoding.DecodeString(b64Salt)
	if err != nil {
		return false
	}
	expectedHash, err := base64.RawStdEncoding.DecodeString(b64Hash)
	if err != nil {
		return false
	}

	computed := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(expectedHash)))

	if len(computed) != len(expectedHash) {
		return false
	}
	var diff byte
	for i := range computed {
		diff |= computed[i] ^ expectedHash[i]
	}
	return diff == 0
}

func (s *AuthService) generateRefreshToken() (token, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate token: %w", err)
	}
	token = base64.URLEncoding.EncodeToString(b)
	hash = hashToken(token)
	return token, hash, nil
}

func (s *AuthService) writeAudit(
	ctx context.Context,
	orgID uuid.UUID,
	actorID *uuid.UUID,
	eventType domain.AuditEventType,
	targetID *uuid.UUID,
	ip, ua string,
) error {
	entry := &domain.AuditEntry{
		OrganizationID:   orgID,
		ActorUserID:      actorID,
		ActionEventType:  eventType,
		TargetResourceID: targetID,
		IPAddress:        &ip,
		UserAgent:        &ua,
	}
	return s.audit.Write(ctx, entry)
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

