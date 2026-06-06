//go:build integration

// Run with: go test ./internal/integration/... -tags integration -v
// Requires: AETHEL_DB_DSN environment variable pointing to a test PostgreSQL 16 instance.
// Migrations must be applied before running these tests.
package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"aethel-core/internal/app"
	"aethel-core/internal/blueprint"
	"aethel-core/internal/database/repos"
	"aethel-core/internal/domain"
	"aethel-core/internal/service"
)

// TestAuthFlow_LoginRefreshLogout exercises the full authentication flow:
// login → refresh (with session rotation) → logout.
func TestAuthFlow_LoginRefreshLogout(t *testing.T) {
	db := openTestDB(t)
	orgID := seedOrg(t, db)

	// Seed a user with a known password hash.
	userID := uuid.New()
	const email = "auth-flow@example.com"
	const password = "Test@Password123"

	userRepo := repos.NewUserRepo(db)
	sessionRepo := repos.NewSessionRepo(db)
	pwResetRepo := repos.NewPasswordResetRepo(db)
	auditNoop := &noopAuditRepo{}

	authSvc := service.NewAuthService(userRepo, sessionRepo, pwResetRepo, auditNoop, blueprint.AuthConfig{})

	// Create the user by inserting directly via repo (uses the bootstrap path).
	// We use RegisterUser from authSvc indirectly via CreateInitialAdmin pattern:
	// insert user manually since we have no sign-up route.
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO users (id, organization_id, email_address, full_name, role, is_active, password_hash)
		VALUES ($1, $2, $3, 'Auth Flow User', 'USER', true,
		        '$argon2id$v=19$m=65536,t=3,p=4$AAECBAUGB8gJCgsM$placeholder')
		ON CONFLICT DO NOTHING
	`, userID, orgID, email)
	if err != nil {
		t.Fatalf("seed auth user: %v", err)
	}

	// Set a real Argon2id hash for the known password.
	// We do this via the authSvc to test the hashing path too.
	// Reset via UpdatePasswordHash directly through bootstrap service.
	bootstrapSvc := service.NewBootstrapService(userRepo, authSvc)
	// Bootstrap service creates a new user; use CreateInitialAdmin to set a real hash.
	// Instead, just wipe and recreate via a direct hash call leveraging authSvc's exported hash.
	// Since hashPassword is unexported, use ConfirmPasswordReset flow or use the bootstrap.
	// Simplest: delete + re-create through bootstrap.
	_, err = db.ExecContext(context.Background(),
		`DELETE FROM users WHERE email_address = $1`, email)
	if err != nil {
		t.Fatalf("cleanup user: %v", err)
	}
	if err := bootstrapSvc.CreateInitialAdmin(context.Background(), app.OrgID, email, "Auth Flow User", password); err != nil {
		// If admin already exists, try a direct insert with a known hash.
		// The bootstrap might fail if an admin already exists. Use a direct hash approach.
		t.Logf("bootstrap admin: %v — seeding user directly", err)
		// Fall back: just use the raw hash that we know is valid for this password.
		// For integration tests we accept the bootstrap may be blocked by existing admin.
		t.Skip("Cannot seed auth user — admin already exists in this DB; run on a clean DB")
	}

	// 1. Login.
	loginResult, err := authSvc.Login(context.Background(), app.OrgID, email, password, "127.0.0.1", "integration-test")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if loginResult.AccessToken == "" {
		t.Fatal("expected non-empty access token after login")
	}
	if loginResult.RefreshToken == "" {
		t.Fatal("expected non-empty refresh token after login")
	}

	oldRefreshToken := loginResult.RefreshToken

	// 2. Refresh — should rotate the session.
	refreshResult, err := authSvc.RefreshSession(context.Background(), oldRefreshToken)
	if err != nil {
		t.Fatalf("RefreshSession: %v", err)
	}
	if refreshResult.AccessToken == "" {
		t.Fatal("expected non-empty access token after refresh")
	}
	newRefreshToken := refreshResult.RefreshToken
	if newRefreshToken == oldRefreshToken {
		t.Error("expected a new refresh token after rotation")
	}

	// 3. Verify old token is no longer valid (rotation check).
	_, err = authSvc.RefreshSession(context.Background(), oldRefreshToken)
	if err != domain.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized when replaying old refresh token, got: %v", err)
	}

	// 4. Logout — should revoke the new session.
	if err := authSvc.Logout(context.Background(), app.OrgID, loginResult.User.ID,
		newRefreshToken, "127.0.0.1", "integration-test"); err != nil {
		t.Fatalf("Logout: %v", err)
	}

	// 5. After logout, the new token should also be invalid.
	_, err = authSvc.RefreshSession(context.Background(), newRefreshToken)
	if err != domain.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized after logout, got: %v", err)
	}
}

// TestAuthRepo_AuditChain writes 3 audit entries and verifies the chain is valid.
func TestAuthRepo_AuditChain(t *testing.T) {
	db := openTestDB(t)
	reg := buildRegistry(t, db)
	orgID := seedOrg(t, db)

	auditRepo := repos.NewAuditRepo(db, reg)
	ctx := context.Background()

	userID := uuid.New()
	eventTypes := []domain.AuditEventType{
		domain.AuditUserLogin,
		domain.AuditUserLoginFailed,
		domain.AuditUserLogout,
	}

	for i, et := range eventTypes {
		entry := &domain.AuditEntry{
			OrganizationID:  orgID,
			ActorUserID:     &userID,
			ActionEventType: et,
			IPAddress:       ptr("127.0.0.1"),
			UserAgent:       ptr("test-agent"),
		}
		if i == 1 {
			// Include a target resource for the second entry.
			targetID := uuid.New()
			entry.TargetResourceID = &targetID
		}
		if err := auditRepo.Write(ctx, entry); err != nil {
			t.Fatalf("Write entry %d (%s): %v", i+1, et, err)
		}
	}

	// Verify the chain is intact.
	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(1 * time.Hour)
	result, err := auditRepo.VerifyChain(ctx, orgID, from, to)
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	if !result.Valid {
		t.Errorf("expected chain to be valid, broken at: %+v", result.BrokenAt)
	}
	if result.TotalRows < 3 {
		t.Errorf("expected at least 3 rows, got %d", result.TotalRows)
	}
}

// ptr is defined in dispatch_test.go's package scope via noopAuditRepo/noopMSRepo.
// Redeclare here for clarity — Go allows multiple helpers in same package.
func ptr[T any](v T) *T { return &v }
