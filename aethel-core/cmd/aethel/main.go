package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	"aethel-core/internal/api"
	"aethel-core/internal/api/handlers"
	"aethel-core/internal/app"
	"aethel-core/internal/blueprint"
	"aethel-core/internal/config"
	"aethel-core/internal/database"
	"aethel-core/internal/database/repos"
	"aethel-core/internal/domain"
	"aethel-core/internal/service"
)

var rootCmd = &cobra.Command{
	Use:   "aethel",
	Short: "Aethel Workspace backend server",
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP server",
	RunE:  runServe,
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Database migration commands",
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Apply all pending migrations",
	RunE:  runMigrateUp,
}

var migrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Roll back migrations",
	RunE:  runMigrateDown,
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration status",
	RunE:  runMigrateStatus,
}

var migrateValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate migration template files",
	RunE:  runMigrateValidate,
}

var migrateSteps int

func init() {
	migrateDownCmd.Flags().IntVar(&migrateSteps, "steps", 1, "number of migrations to roll back")
	migrateCmd.AddCommand(migrateUpCmd, migrateDownCmd, migrateStatusCmd, migrateValidateCmd)
	rootCmd.AddCommand(serveCmd, migrateCmd)
}

func main() {
	_ = godotenv.Load()

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// ── serve ─────────────────────────────────────────────────────────────────────

func runServe(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// 1. Load blueprints.
	dbCfg, queriesCfg, envCfg, err := loadBlueprints()
	if err != nil {
		return err
	}

	// 2. Open database.
	db, err := database.Open(envCfg)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer db.Close()
	slog.Info("database connected")

	// 3. Auto-run migrations if configured.
	if envCfg.Migrations.AutoRunOnStartup {
		m := database.NewMigrator(db, dbCfg, envCfg)
		if err := m.Up(ctx); err != nil {
			return fmt.Errorf("run migrations: %w", err)
		}
	}

	// 4. Load the single org ID (single-tenant: exactly one row in organizations).
	if err := app.LoadOrgID(ctx, db); err != nil {
		slog.Warn("could not load org ID — proceeding without it", "err", err)
	}

	// 5. Build query registry.
	queries, err := database.BuildQueryRegistry(ctx, db, queriesCfg)
	if err != nil {
		return fmt.Errorf("build query registry: %w", err)
	}
	slog.Info("query registry built")

	// 6. Initialize single-tenant config cache.
	configCache := config.NewConfigCache()

	// 7. Build repositories.
	// Sprint 2 — Dispatch pillar: real DB implementations.
	var (
		dispatchRepo domain.DispatchRepository      = repos.NewDispatchRepo(db, queries)
		eventRepo    domain.DispatchEventRepository = repos.NewDispatchEventRepo(db, queries)
		routingRepo  domain.RoutingRuleRepository   = repos.NewRoutingRuleRepo(db, queries)
	)

	// Auth pillar — implemented in Sprint 1 (noops kept until Sprint 1 repos land).
	var (
		userRepo    domain.UserRepository          = &noopUserRepo{}
		sessionRepo domain.SessionRepository       = &noopSessionRepo{}
		pwResetRepo domain.PasswordResetRepository = &noopPWResetRepo{}
		auditRepo   domain.AuditRepository         = &noopAuditRepo{}
	)

	// Sprint 3–4 placeholders.
	var (
		msRepo      domain.MinuteSheetRepository    = &noopMSRepo{}
		gnRepo      domain.GreenNoteRepository      = &noopGNRepo{}
		docTypeRepo domain.DocumentTypeRepository   = &noopDocTypeRepo{}
		escRepo     domain.EscalationRuleRepository = &noopEscRepo{}
	)

	// 8. Wire services.
	authSvc := service.NewAuthService(userRepo, sessionRepo, pwResetRepo, auditRepo)
	dispatchSvc := service.NewDispatchService(dispatchRepo, eventRepo, routingRepo, msRepo, auditRepo)
	workflowSvc := service.NewWorkflowService(msRepo, gnRepo, auditRepo)

	// 9. Wire handlers.
	authHandler := handlers.NewAuthHandler(authSvc)
	dispatchHandler := handlers.NewDispatchHandler(dispatchSvc)
	workflowHandler := handlers.NewWorkflowHandler(workflowSvc)

	adminDeps := handlers.AdminDeps{
		Users:        userRepo,
		DocTypes:     docTypeRepo,
		RoutingRules: routingRepo,
		EscRules:     escRepo,
	}

	// 10. Start HTTP server.
	addr := envAddr()
	srv := api.NewServer(db, queries, configCache, authHandler, dispatchHandler, workflowHandler, auditRepo, adminDeps)
	slog.Info("starting server", "addr", addr)
	return srv.ListenAndServe(addr)
}

// ── migrate commands ──────────────────────────────────────────────────────────

func runMigrateUp(_ *cobra.Command, _ []string) error {
	dbCfg, envCfg, err := loadDatabaseBlueprint()
	if err != nil {
		return err
	}
	db, err := database.Open(envCfg)
	if err != nil {
		return err
	}
	defer db.Close()
	return database.NewMigrator(db, dbCfg, envCfg).Up(context.Background())
}

func runMigrateDown(_ *cobra.Command, _ []string) error {
	dbCfg, envCfg, err := loadDatabaseBlueprint()
	if err != nil {
		return err
	}
	db, err := database.Open(envCfg)
	if err != nil {
		return err
	}
	defer db.Close()
	return database.NewMigrator(db, dbCfg, envCfg).Down(context.Background(), migrateSteps)
}

func runMigrateStatus(_ *cobra.Command, _ []string) error {
	dbCfg, envCfg, err := loadDatabaseBlueprint()
	if err != nil {
		return err
	}
	db, err := database.Open(envCfg)
	if err != nil {
		return err
	}
	defer db.Close()
	return database.NewMigrator(db, dbCfg, envCfg).Status(context.Background())
}

func runMigrateValidate(_ *cobra.Command, _ []string) error {
	dbCfg, envCfg, err := loadDatabaseBlueprint()
	if err != nil {
		return err
	}
	// Validate does not need a live DB — use a stub.
	db, _ := database.Open(envCfg)
	if db != nil {
		defer db.Close()
	}
	// Create a minimal DB-less migrator just for template validation.
	m := database.NewMigrator(nil, dbCfg, envCfg)
	return m.Validate(context.Background())
}

// ── helpers ───────────────────────────────────────────────────────────────────

func loadBlueprints() (*blueprint.DatabaseConfig, *blueprint.QueriesConfig, blueprint.EnvironmentConfig, error) {
	dbCfg, envCfg, err := loadDatabaseBlueprint()
	if err != nil {
		return nil, nil, blueprint.EnvironmentConfig{}, err
	}

	queriesCfg, err := blueprint.LoadQueriesConfig("aethel-core/internal/database/queries/queries.yaml")
	if err != nil {
		return nil, nil, blueprint.EnvironmentConfig{}, fmt.Errorf("load queries blueprint: %w", err)
	}

	return dbCfg, queriesCfg, envCfg, nil
}

// loadDatabaseBlueprint loads only the database config — used by migrate commands
// which don't need the queries blueprint.
func loadDatabaseBlueprint() (*blueprint.DatabaseConfig, blueprint.EnvironmentConfig, error) {
	dbCfg, err := blueprint.LoadDatabaseConfig("blueprints/server-database.yaml")
	if err != nil {
		return nil, blueprint.EnvironmentConfig{}, fmt.Errorf("load database blueprint: %w", err)
	}

	env := os.Getenv("AETHEL_ENV")
	if env == "" {
		env = "development"
	}

	envCfg, ok := dbCfg.Environments[env]
	if !ok {
		return nil, blueprint.EnvironmentConfig{},
			fmt.Errorf("blueprint: environment %q not defined in server-database.yaml", env)
	}

	return dbCfg, envCfg, nil
}

func envAddr() string {
	port := os.Getenv("AETHEL_PORT")
	if port == "" {
		port = "8080"
	}
	if _, err := strconv.Atoi(port); err != nil {
		slog.Warn("invalid AETHEL_PORT, using 8080")
		port = "8080"
	}
	return ":" + port
}

// ── stub repositories (replaced in Sprint 1 / Sprint 3–4) ────────────────────

type noopUserRepo struct{}

func (r *noopUserRepo) GetByID(_ context.Context, _, _ uuid.UUID) (*domain.User, error) {
	return nil, domain.ErrNotFound
}
func (r *noopUserRepo) GetByEmail(_ context.Context, _ uuid.UUID, _ string) (*domain.User, error) {
	return nil, domain.ErrNotFound
}
func (r *noopUserRepo) List(_ context.Context, _ uuid.UUID, _ domain.Page) ([]domain.User, error) {
	return nil, nil
}
func (r *noopUserRepo) Create(_ context.Context, _ *domain.User) error                    { return nil }
func (r *noopUserRepo) Update(_ context.Context, _ *domain.User) error                    { return nil }
func (r *noopUserRepo) UpdatePasswordHash(_ context.Context, _ uuid.UUID, _ string) error { return nil }
func (r *noopUserRepo) IncrementFailedLogins(_ context.Context, _ uuid.UUID) error        { return nil }
func (r *noopUserRepo) ResetFailedLogins(_ context.Context, _ uuid.UUID) error            { return nil }
func (r *noopUserRepo) LockUntil(_ context.Context, _ uuid.UUID, _ time.Time) error       { return nil }
func (r *noopUserRepo) SetLastLogin(_ context.Context, _ uuid.UUID) error                 { return nil }

type noopSessionRepo struct{}

func (r *noopSessionRepo) Create(_ context.Context, _ *domain.Session) error { return nil }
func (r *noopSessionRepo) GetByTokenHash(_ context.Context, _ string) (*domain.Session, error) {
	return nil, domain.ErrNotFound
}
func (r *noopSessionRepo) DeleteByID(_ context.Context, _ uuid.UUID) error     { return nil }
func (r *noopSessionRepo) DeleteByUserID(_ context.Context, _ uuid.UUID) error { return nil }

type noopPWResetRepo struct{}

func (r *noopPWResetRepo) Create(_ context.Context, _ *domain.PasswordResetToken) error { return nil }
func (r *noopPWResetRepo) GetByTokenHash(_ context.Context, _ string) (*domain.PasswordResetToken, error) {
	return nil, domain.ErrNotFound
}
func (r *noopPWResetRepo) MarkUsed(_ context.Context, _ uuid.UUID) error { return nil }

type noopAuditRepo struct{}

func (r *noopAuditRepo) Write(_ context.Context, _ *domain.AuditEntry) error { return nil }
func (r *noopAuditRepo) Query(_ context.Context, _ uuid.UUID, _, _ time.Time, _ domain.Page) ([]domain.AuditEntry, error) {
	return nil, nil
}
func (r *noopAuditRepo) VerifyChain(_ context.Context, _ uuid.UUID, _, _ time.Time) (*domain.ChainVerificationResult, error) {
	return &domain.ChainVerificationResult{Valid: true}, nil
}

// Sprint 3–4 stubs — implemented when Green Noting & Escalation sprints land.

type noopMSRepo struct{}

func (r *noopMSRepo) GetByDispatchID(_ context.Context, _, _ uuid.UUID) (*domain.MinuteSheet, error) {
	return nil, domain.ErrNotFound
}
func (r *noopMSRepo) GetByID(_ context.Context, _, _ uuid.UUID) (*domain.MinuteSheet, error) {
	return nil, domain.ErrNotFound
}
func (r *noopMSRepo) Create(_ context.Context, _ *domain.MinuteSheet) error        { return nil }
func (r *noopMSRepo) Approve(_ context.Context, _, _ uuid.UUID, _ uuid.UUID) error { return nil }

type noopGNRepo struct{}

func (r *noopGNRepo) Create(_ context.Context, _ *domain.GreenNote) error { return nil }
func (r *noopGNRepo) ListByMinuteSheet(_ context.Context, _, _ uuid.UUID) ([]domain.GreenNote, error) {
	return nil, nil
}
func (r *noopGNRepo) GetLastByMinuteSheet(_ context.Context, _, _ uuid.UUID) (*domain.GreenNote, error) {
	return nil, domain.ErrNotFound
}

type noopDocTypeRepo struct{}

func (r *noopDocTypeRepo) List(_ context.Context, _ uuid.UUID) ([]domain.DocumentType, error) {
	return nil, nil
}
func (r *noopDocTypeRepo) GetByID(_ context.Context, _, _ uuid.UUID) (*domain.DocumentType, error) {
	return nil, domain.ErrNotFound
}
func (r *noopDocTypeRepo) Create(_ context.Context, _ *domain.DocumentType) error { return nil }
func (r *noopDocTypeRepo) Update(_ context.Context, _ *domain.DocumentType) error { return nil }
func (r *noopDocTypeRepo) Delete(_ context.Context, _, _ uuid.UUID) error         { return nil }

type noopEscRepo struct{}

func (r *noopEscRepo) List(_ context.Context, _ uuid.UUID) ([]domain.EscalationRule, error) {
	return nil, nil
}
func (r *noopEscRepo) GetByID(_ context.Context, _, _ uuid.UUID) (*domain.EscalationRule, error) {
	return nil, domain.ErrNotFound
}
func (r *noopEscRepo) Create(_ context.Context, _ *domain.EscalationRule) error { return nil }
func (r *noopEscRepo) Update(_ context.Context, _ *domain.EscalationRule) error { return nil }
