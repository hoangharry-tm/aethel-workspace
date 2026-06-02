//go:build integration

// Run with: go test ./internal/integration/... -tags integration -v
// Requires: AETHEL_DB_DSN environment variable pointing to a test PostgreSQL 16 instance.
// The test creates dispatches against the live database — ensure migrations are applied.
package integration

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/google/uuid"

	"aethel-core/internal/app"
	"aethel-core/internal/blueprint"
	"aethel-core/internal/config"
	"aethel-core/internal/database"
	"aethel-core/internal/database/repos"
	"aethel-core/internal/domain"
	"aethel-core/internal/service"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("AETHEL_DB_DSN")
	if dsn == "" {
		t.Skip("AETHEL_DB_DSN not set — skipping integration tests")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open test DB: %v", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("ping test DB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func buildRegistry(t *testing.T, db *sql.DB) *database.QueryRegistry {
	t.Helper()
	cfg, err := blueprint.LoadQueriesConfig("../../internal/database/queries/queries.yaml")
	if err != nil {
		t.Fatalf("load queries config: %v", err)
	}
	reg, err := database.BuildQueryRegistry(context.Background(), db, cfg)
	if err != nil {
		t.Fatalf("build query registry: %v", err)
	}
	return reg
}

// seedOrg inserts a test organization and sets app.OrgID.
func seedOrg(t *testing.T, db *sql.DB) uuid.UUID {
	t.Helper()
	orgID := uuid.New()
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO organizations (id, name, slug) VALUES ($1, $2, $3) ON CONFLICT (slug) DO NOTHING`,
		orgID, "Test Org", "test-org-"+orgID.String()[:8],
	)
	if err != nil {
		t.Fatalf("seed org: %v", err)
	}
	// Re-load in case ON CONFLICT skipped our insert.
	if err := app.LoadOrgID(context.Background(), db); err != nil {
		t.Fatalf("load org ID: %v", err)
	}
	return app.OrgID
}

// seedDocType inserts a document type for use in dispatch tests.
func seedDocType(t *testing.T, db *sql.DB, orgID uuid.UUID) uuid.UUID {
	t.Helper()
	dtID := uuid.New()
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO document_types (id, organization_id, name, code, is_active)
		 VALUES ($1, $2, 'Test Doc', 'TEST', true) ON CONFLICT DO NOTHING`,
		dtID, orgID,
	)
	if err != nil {
		t.Fatalf("seed doc type: %v", err)
	}
	return dtID
}

// seedUser inserts a minimal user for submitted_by_user_id.
func seedUser(t *testing.T, db *sql.DB, orgID uuid.UUID) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO users (id, organization_id, email_address, full_name, role, is_active, password_hash)
		 VALUES ($1, $2, $3, 'Test User', 'RECEPTION', true, 'hash') ON CONFLICT DO NOTHING`,
		userID, orgID, "test+"+userID.String()[:8]+"@example.com",
	)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return userID
}

// TestCreateDispatch verifies Create stores the dispatch and GetByID retrieves it.
func TestCreateDispatch(t *testing.T) {
	db := openTestDB(t)
	reg := buildRegistry(t, db)
	orgID := seedOrg(t, db)
	dtID := seedDocType(t, db, orgID)
	userID := seedUser(t, db, orgID)

	dispatchRepo := repos.NewDispatchRepo(db, reg)

	d := &domain.Dispatch{
		ID:                uuid.New(),
		OrganizationID:    orgID,
		TrackingNumber:    "AE-" + strings.ToUpper(uuid.New().String()[:8]),
		Direction:         domain.DirectionInbound,
		DocumentTypeID:    dtID,
		SenderName:        "Integration Test Sender",
		SubmittedByUserID: userID,
		PriorityLevel:     domain.PriorityRoutine,
		StatusState:       domain.StatusPendingAssignment,
	}

	if err := dispatchRepo.Create(context.Background(), d); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := dispatchRepo.GetByID(context.Background(), orgID, d.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != d.ID {
		t.Errorf("ID mismatch: got %v want %v", got.ID, d.ID)
	}
	if !strings.HasPrefix(got.TrackingNumber, "AE-") {
		t.Errorf("tracking number %q should start with AE-", got.TrackingNumber)
	}
}

// TestAcknowledgeDelivery verifies status transition through Acknowledge.
func TestAcknowledgeDelivery(t *testing.T) {
	db := openTestDB(t)
	reg := buildRegistry(t, db)
	orgID := seedOrg(t, db)
	dtID := seedDocType(t, db, orgID)
	userID := seedUser(t, db, orgID)

	dispatchRepo := repos.NewDispatchRepo(db, reg)
	eventRepo := repos.NewDispatchEventRepo(db, reg)
	routingRepo := repos.NewRoutingRuleRepo(db, reg)
	_ = routingRepo // satisfies interface; routing rules table may be empty
	msRepo := &noopMSRepo{}
	auditRepo := &noopAuditRepo{}

	_ = service.NewDispatchService(dispatchRepo, eventRepo, routingRepo, msRepo, auditRepo)

	d := &domain.Dispatch{
		ID:                uuid.New(),
		OrganizationID:    orgID,
		TrackingNumber:    "AE-" + strings.ToUpper(uuid.New().String()[:8]),
		Direction:         domain.DirectionInbound,
		DocumentTypeID:    dtID,
		SenderName:        "Ack Test Sender",
		SubmittedByUserID: userID,
		PriorityLevel:     domain.PriorityRoutine,
		StatusState:       domain.StatusPendingAssignment,
	}
	if err := dispatchRepo.Create(context.Background(), d); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := dispatchRepo.UpdateStatus(context.Background(), orgID, d.ID, domain.StatusUnderReview); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	if err := dispatchRepo.Acknowledge(context.Background(), orgID, d.ID, userID); err != nil {
		t.Fatalf("Acknowledge: %v", err)
	}

	got, err := dispatchRepo.GetByID(context.Background(), orgID, d.ID)
	if err != nil {
		t.Fatalf("GetByID after acknowledge: %v", err)
	}
	if got.StatusState != domain.StatusDelivered {
		t.Errorf("status: got %v want DELIVERED", got.StatusState)
	}
	if got.AcknowledgedAt == nil {
		t.Error("acknowledged_at should not be nil after Acknowledge")
	}
}

// TestConfigCacheInvalidation verifies PATCH branding → GET config returns updated value.
func TestConfigCacheInvalidation(t *testing.T) {
	db := openTestDB(t)
	_ = seedOrg(t, db) // ensures app.OrgID is set

	cache := config.NewConfigCache()
	h := config.NewHandler(db, cache)

	// First GET populates the cache.
	rec1 := httptest.NewRecorder()
	h.GetConfig(rec1, httptest.NewRequest(http.MethodGet, "/api/v1/config", nil))
	if rec1.Code != http.StatusOK {
		t.Fatalf("first GET /config: %d %s", rec1.Code, rec1.Body.String())
	}

	// PATCH branding with a new primary color.
	body := `{"primaryColor":"#ff0000"}`
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/config/branding", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	h.PatchBranding(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("PATCH branding: %d %s", rec2.Code, rec2.Body.String())
	}

	// Second GET should reflect the patched color.
	rec3 := httptest.NewRecorder()
	h.GetConfig(rec3, httptest.NewRequest(http.MethodGet, "/api/v1/config", nil))
	if rec3.Code != http.StatusOK {
		t.Fatalf("second GET /config: %d %s", rec3.Code, rec3.Body.String())
	}
	if !strings.Contains(rec3.Body.String(), "#ff0000") && !strings.Contains(rec3.Body.String(), "ff0000") {
		t.Errorf("expected updated primaryColor in response, got: %s", rec3.Body.String())
	}
}

// TestRoutingRuleEngine verifies dispatch service routes based on active rules.
func TestRoutingRuleEngine(t *testing.T) {
	db := openTestDB(t)
	reg := buildRegistry(t, db)
	orgID := seedOrg(t, db)
	dtID := seedDocType(t, db, orgID)
	userID := seedUser(t, db, orgID)

	// Seed a department for the routing rule destination.
	deptID := uuid.New()
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO departments (id, organization_id, name, code) VALUES ($1, $2, 'Routing Dept', 'RDEPT') ON CONFLICT DO NOTHING`,
		deptID, orgID,
	)
	if err != nil {
		t.Logf("seed department: %v (may already exist, continuing)", err)
	}

	// Seed a routing rule with DOCUMENT_TYPE condition.
	ruleID := uuid.New()
	_, err = db.ExecContext(context.Background(),
		`INSERT INTO routing_rules (id, organization_id, name, priority_order, is_active, created_by_user_id)
		 VALUES ($1, $2, 'Test Rule', 1, true, $3) ON CONFLICT DO NOTHING`,
		ruleID, orgID, userID,
	)
	if err != nil {
		t.Fatalf("seed routing rule: %v", err)
	}

	condID := uuid.New()
	_, err = db.ExecContext(context.Background(),
		`INSERT INTO routing_rule_conditions (id, routing_rule_id, condition_type, condition_value, match_operator)
		 VALUES ($1, $2, 'DOCUMENT_TYPE', $3, 'EQUALS') ON CONFLICT DO NOTHING`,
		condID, ruleID, dtID.String(),
	)
	if err != nil {
		t.Fatalf("seed routing rule condition: %v", err)
	}

	_, err = db.ExecContext(context.Background(),
		`INSERT INTO routing_rule_destinations (id, routing_rule_id, stop_order, target_department_id)
		 VALUES ($1, $2, 1, $3) ON CONFLICT DO NOTHING`,
		uuid.New(), ruleID, deptID,
	)
	if err != nil {
		t.Fatalf("seed routing rule destination: %v", err)
	}

	dispatchRepo := repos.NewDispatchRepo(db, reg)
	eventRepo := repos.NewDispatchEventRepo(db, reg)
	routingRepo := repos.NewRoutingRuleRepo(db, reg)
	svc := service.NewDispatchService(dispatchRepo, eventRepo, routingRepo, &noopMSRepo{}, &noopAuditRepo{})

	created, err := svc.Create(context.Background(), &service.Dispatch{
		Direction:      "INBOUND",
		DocumentTypeID: dtID,
		SenderName:     "Rule Test Sender",
		PriorityLevel:  "ROUTINE",
	}, userID, orgID, "127.0.0.1")
	if err != nil {
		t.Fatalf("Create dispatch via service: %v", err)
	}
	if created.AssignedDepartmentID == nil || *created.AssignedDepartmentID != deptID {
		t.Errorf("expected dispatch assigned to dept %v, got %v", deptID, created.AssignedDepartmentID)
	}

	// Verify ROUTING_APPLIED event was logged.
	events, err := eventRepo.ListByDispatch(context.Background(), orgID, created.ID)
	if err != nil {
		t.Fatalf("ListByDispatch: %v", err)
	}
	var found bool
	for _, e := range events {
		if e.EventType == "ROUTING_APPLIED" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected ROUTING_APPLIED event in dispatch_events, got %d events", len(events))
	}
}

// ── lightweight stubs used by integration tests ───────────────────────────────

type noopMSRepo struct{}

func (r *noopMSRepo) GetByDispatchID(_ context.Context, _, _ uuid.UUID) (*domain.MinuteSheet, error) {
	return nil, domain.ErrNotFound
}
func (r *noopMSRepo) GetByID(_ context.Context, _, _ uuid.UUID) (*domain.MinuteSheet, error) {
	return nil, domain.ErrNotFound
}
func (r *noopMSRepo) Create(_ context.Context, _ *domain.MinuteSheet) error        { return nil }
func (r *noopMSRepo) Approve(_ context.Context, _, _ uuid.UUID, _ uuid.UUID) error { return nil }

type noopAuditRepo struct{}

func (r *noopAuditRepo) Write(_ context.Context, _ *domain.AuditEntry) error { return nil }
func (r *noopAuditRepo) Query(_ context.Context, _ uuid.UUID, _, _ time.Time, _ domain.Page) ([]domain.AuditEntry, error) {
	return nil, nil
}
func (r *noopAuditRepo) VerifyChain(_ context.Context, _ uuid.UUID, _, _ time.Time) (*domain.ChainVerificationResult, error) {
	return &domain.ChainVerificationResult{Valid: true}, nil
}
