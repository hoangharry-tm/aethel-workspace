package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
	"bufio"
	"strings"
	"errors"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	"aethel-core/internal/api"
	"aethel-core/internal/api/docs"
	"aethel-core/internal/api/handlers"
	"aethel-core/internal/app"
	"aethel-core/internal/blueprint"
	"aethel-core/internal/config"
	"aethel-core/internal/database"
	"aethel-core/internal/database/repos"
	"aethel-core/internal/domain"
	"aethel-core/internal/service"

	"golang.org/x/term"
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

var bootstrapAdminCmd = &cobra.Command{
	Use:   "bootstrap-admin",
	Short: "Create the initial SYS_ADMIN account",
	RunE:  runBootstrapAdmin,
}

var migrateSteps int

func init() {
	migrateDownCmd.Flags().IntVar(
		&migrateSteps,
		"steps",
		1,
		"number of migrations to roll back",
	)

	migrateCmd.AddCommand(
		migrateUpCmd,
		migrateDownCmd,
		migrateStatusCmd,
		migrateValidateCmd,
	)

	rootCmd.AddCommand(
		serveCmd,
		migrateCmd,
		bootstrapAdminCmd,
	)
}

func main() {
	_ = godotenv.Load("../.env")

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

	// Log algorithm only — never log the key value or any secret env var content.
	jwtSecret := os.Getenv("AETHEL_JWT_SECRET")
	if jwtSecret == "" {
		slog.Warn("AETHEL_JWT_SECRET not set — using insecure development default") // safe: logs absence, not value
	} else {
		slog.Info("JWT algorithm: HS256")
	}

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

	// Auth pillar — real implementations wired in Sprint 1.5.
	var (
		userRepo    domain.UserRepository          = repos.NewUserRepo(db)
		sessionRepo domain.SessionRepository       = repos.NewSessionRepo(db)
		pwResetRepo domain.PasswordResetRepository = repos.NewPasswordResetRepo(db)
		auditRepo   domain.AuditRepository         = repos.NewAuditRepo(db, queries)
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

	// 10. Validate OpenAPI spec at startup. Panics if the spec is malformed.
	docs.ValidateSpec()

	// 11. Start HTTP server.
	addr := envAddr()
	srv := api.NewServer(db, queries, configCache, authHandler, dispatchHandler, workflowHandler, auditRepo, adminDeps)
	slog.Info("starting server", "addr", addr)
	return srv.ListenAndServe(addr)
}

// ── bootstrap-admin ─────────────────────────────────────────────────────────────────────
func runBootstrapAdmin(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	// Load DB config.
	_, envCfg, err := loadDatabaseBlueprint()
	if err != nil {
		return err
	}

	// Connect DB.
	db, err := database.Open(envCfg)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

	// Load org ID.
	if err := app.LoadOrgID(ctx, db); err != nil {
		return fmt.Errorf("load org id: %w", err)
	}

	// Build repos.
	userRepo := repos.NewUserRepo(db)
	sessionRepo := repos.NewSessionRepo(db)
	pwResetRepo := repos.NewPasswordResetRepo(db)

	// Build services.
	// Bootstrap uses a write-only audit noop — no query registry is loaded in this path.
	auditRepo := &bootstrapAuditRepo{}
	authSvc := service.NewAuthService(
		userRepo,
		sessionRepo,
		pwResetRepo,
		auditRepo,
	)

	bootstrapSvc := service.NewBootstrapService(
		userRepo,
		authSvc,
	)

	reader := bufio.NewReader(os.Stdin)

	// Email
	fmt.Print("Email: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	email = strings.TrimSpace(email)

	// Full name
	fmt.Print("Full name: ")
	fullName, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	fullName = strings.TrimSpace(fullName)

	// Password
	fmt.Print("Password: ")
	passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	fmt.Println()

	password := strings.TrimSpace(string(passwordBytes))

	// Confirm password
	fmt.Print("Confirm password: ")
	confirmBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	fmt.Println()

	confirmPassword := strings.TrimSpace(string(confirmBytes))

	if password != confirmPassword {
		return errors.New("passwords do not match")
	}

	err = bootstrapSvc.CreateInitialAdmin(
		ctx,
		app.OrgID,
		email,
		fullName,
		password,
	)
	if err != nil {
		return err
	}

	fmt.Println("✓ SYS_ADMIN account created successfully")

	return nil
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

	queriesCfg, err := blueprint.LoadQueriesConfig("./internal/database/queries/queries.yaml")
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

// ── stub repositories (replaced in Sprint 3–4) ───────────────────────────────

// bootstrapAuditRepo is a write-only no-op used by the bootstrap-admin command,
// which does not load the query registry. The serve command uses repos.NewAuditRepo.
type bootstrapAuditRepo struct{}

func (r *bootstrapAuditRepo) Write(_ context.Context, _ *domain.AuditEntry) error { return nil }
func (r *bootstrapAuditRepo) Query(_ context.Context, _ uuid.UUID, _, _ time.Time, _ domain.Page) ([]domain.AuditEntry, error) {
	return nil, nil
}
func (r *bootstrapAuditRepo) VerifyChain(_ context.Context, _ uuid.UUID, _, _ time.Time) (*domain.ChainVerificationResult, error) {
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
