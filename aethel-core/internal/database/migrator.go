package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"aethel-core/internal/blueprint"
)

type Migrator struct {
	db             *sql.DB
	ctx            *BlueprintContext
	migrationsDir  string
	historyTable   string
	lockTimeout    int
}

func NewMigrator(db *sql.DB, cfg *blueprint.DatabaseConfig, envCfg blueprint.EnvironmentConfig) *Migrator {
	return &Migrator{
		db:            db,
		ctx:           NewBlueprintContext(cfg),
		migrationsDir: envCfg.Migrations.Directory,
		historyTable:  envCfg.Migrations.TableName,
		lockTimeout:   envCfg.Migrations.LockTimeoutSeconds,
	}
}

func (m *Migrator) Up(ctx context.Context) error {
	if err := m.acquireLock(ctx); err != nil {
		return err
	}
	defer m.releaseLock(ctx) //nolint:errcheck

	if err := m.ensureHistoryTable(ctx); err != nil {
		return err
	}

	files, err := m.collectFiles("up")
	if err != nil {
		return err
	}

	applied, err := m.appliedVersions(ctx)
	if err != nil {
		return err
	}

	for _, f := range files {
		version, description := parseMigrationFilename(f)
		if applied[version] {
			slog.Info("migration already applied", "version", version)
			continue
		}

		raw, err := os.ReadFile(filepath.Join(m.migrationsDir, f))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", f, err)
		}

		rendered, err := renderMigration(raw, m.ctx)
		if err != nil {
			return fmt.Errorf("render migration %s: %w", f, err)
		}

		checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(rendered)))

		if err := m.runInTx(ctx, func(tx *sql.Tx) error {
			if _, err := tx.ExecContext(ctx, rendered); err != nil {
				return fmt.Errorf("exec migration %s: %w", f, err)
			}
			_, err = tx.ExecContext(ctx,
				fmt.Sprintf(
					`INSERT INTO %s.%s (version, description, checksum) VALUES ($1, $2, $3)`,
					m.ctx.Schema, m.historyTable,
				),
				version, description, checksum,
			)
			return err
		}); err != nil {
			return err
		}

		slog.Info("migration applied", "version", version, "description", description)
	}
	return nil
}

func (m *Migrator) Down(ctx context.Context, steps int) error {
	if err := m.acquireLock(ctx); err != nil {
		return err
	}
	defer m.releaseLock(ctx) //nolint:errcheck

	if steps <= 0 {
		steps = 1
	}

	applied, err := m.appliedVersionsSorted(ctx)
	if err != nil {
		return err
	}
	if len(applied) == 0 {
		slog.Info("no migrations to roll back")
		return nil
	}

	targets := applied
	if steps < len(targets) {
		targets = targets[len(targets)-steps:]
	}

	// Roll back in reverse order.
	for i := len(targets) - 1; i >= 0; i-- {
		version := targets[i]
		filename := m.findDownFile(version)
		if filename == "" {
			return fmt.Errorf("down migration for version %s not found", version)
		}

		raw, err := os.ReadFile(filepath.Join(m.migrationsDir, filename))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", filename, err)
		}

		rendered, err := renderMigration(raw, m.ctx)
		if err != nil {
			return fmt.Errorf("render migration %s: %w", filename, err)
		}

		if err := m.runInTx(ctx, func(tx *sql.Tx) error {
			if _, err := tx.ExecContext(ctx, rendered); err != nil {
				return fmt.Errorf("exec down migration %s: %w", filename, err)
			}
			_, err = tx.ExecContext(ctx,
				fmt.Sprintf(`DELETE FROM %s.%s WHERE version = $1`, m.ctx.Schema, m.historyTable),
				version,
			)
			return err
		}); err != nil {
			return err
		}

		slog.Info("migration rolled back", "version", version)
	}
	return nil
}

func (m *Migrator) Status(ctx context.Context) error {
	if err := m.ensureHistoryTable(ctx); err != nil {
		return err
	}

	upFiles, err := m.collectFiles("up")
	if err != nil {
		return err
	}

	applied, err := m.appliedVersions(ctx)
	if err != nil {
		return err
	}

	for _, f := range upFiles {
		version, description := parseMigrationFilename(f)
		status := "pending"
		if applied[version] {
			status = "applied"
		}
		fmt.Printf("%-30s %-50s %s\n", version, description, status)
	}
	return nil
}

// Validate renders all migration templates and checks for template errors.
// If db is non-nil, it also performs a SQL syntax check for each .up.sql file by
// wrapping every statement in BEGIN/EXPLAIN/ROLLBACK — this validates SQL syntax
// without modifying the database. Pass a nil db to skip the syntax check.
func (m *Migrator) Validate(ctx context.Context, db *sql.DB) error {
	files, err := m.collectFiles("up")
	if err != nil {
		return err
	}

	for _, f := range files {
		raw, err := os.ReadFile(filepath.Join(m.migrationsDir, f))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", f, err)
		}

		rendered, err := renderMigration(raw, m.ctx)
		if err != nil {
			return fmt.Errorf("template error in %s: %w", f, err)
		}
		slog.Info("migration template valid", "file", f)

		if db != nil {
			if err := checkSQLSyntax(ctx, db, f, rendered); err != nil {
				return err
			}
			slog.Info("migration syntax valid", "file", f)
		}
	}

	// Validate down files as well.
	downFiles, err := m.collectFiles("down")
	if err != nil {
		return err
	}
	for _, f := range downFiles {
		raw, err := os.ReadFile(filepath.Join(m.migrationsDir, f))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", f, err)
		}
		if _, err := renderMigration(raw, m.ctx); err != nil {
			return fmt.Errorf("template error in %s: %w", f, err)
		}
	}

	slog.Info("all migration templates valid", "count", len(files))
	return nil
}

// checkSQLSyntax wraps each SQL statement in the rendered migration inside a
// BEGIN / EXPLAIN / ROLLBACK block so PostgreSQL validates the syntax without
// executing any DDL. Returns a wrapped error that includes the migration filename
// and the offending statement on failure.
func checkSQLSyntax(ctx context.Context, db *sql.DB, filename, rendered string) error {
	// Split on ";\n" to handle multi-statement migrations.  The trailing empty
	// segment produced by a final semicolon is skipped via the blank-line check.
	stmts := strings.Split(rendered, ";\n")
	for _, raw := range stmts {
		stmt := strings.TrimSpace(raw)
		if stmt == "" {
			continue
		}
		// Wrap in a transaction that is always rolled back, so no state is written.
		explainSQL := "BEGIN; EXPLAIN " + stmt + "; ROLLBACK;"
		if _, err := db.ExecContext(ctx, explainSQL); err != nil {
			return fmt.Errorf("sql syntax error in %s: %w\nstatement: %s", filename, err, stmt)
		}
	}
	return nil
}

func (m *Migrator) acquireLock(ctx context.Context) error {
	_, err := m.db.ExecContext(ctx,
		fmt.Sprintf("SET lock_timeout = '%ds'", m.lockTimeout),
	)
	if err != nil {
		return fmt.Errorf("set lock timeout: %w", err)
	}
	_, err = m.db.ExecContext(ctx, "SELECT pg_advisory_lock(8765432101)")
	if err != nil {
		return fmt.Errorf("acquire advisory lock: %w", err)
	}
	return nil
}

func (m *Migrator) releaseLock(ctx context.Context) error {
	_, err := m.db.ExecContext(ctx, "SELECT pg_advisory_unlock(8765432101)")
	return err
}

func (m *Migrator) ensureHistoryTable(ctx context.Context) error {
	_, err := m.db.ExecContext(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.%s (
			version     varchar(30) NOT NULL,
			description varchar(255),
			checksum    varchar(64) NOT NULL,
			applied_at  timestamptz NOT NULL DEFAULT now(),
			CONSTRAINT %s_pkey PRIMARY KEY (version)
		)
	`, m.ctx.Schema, m.historyTable, m.historyTable))
	return err
}

func (m *Migrator) collectFiles(direction string) ([]string, error) {
	entries, err := os.ReadDir(m.migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir %s: %w", m.migrationsDir, err)
	}

	suffix := "." + direction + ".sql"
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), suffix) {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	return files, nil
}

func (m *Migrator) appliedVersions(ctx context.Context) (map[string]bool, error) {
	rows, err := m.db.QueryContext(ctx,
		fmt.Sprintf("SELECT version FROM %s.%s", m.ctx.Schema, m.historyTable),
	)
	if err != nil {
		return nil, fmt.Errorf("query applied versions: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[v] = true
	}
	return applied, rows.Err()
}

func (m *Migrator) appliedVersionsSorted(ctx context.Context) ([]string, error) {
	rows, err := m.db.QueryContext(ctx,
		fmt.Sprintf("SELECT version FROM %s.%s ORDER BY applied_at ASC", m.ctx.Schema, m.historyTable),
	)
	if err != nil {
		return nil, fmt.Errorf("query applied versions: %w", err)
	}
	defer rows.Close()

	var versions []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}
	return versions, rows.Err()
}

func (m *Migrator) findDownFile(version string) string {
	entries, err := os.ReadDir(m.migrationsDir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), version) && strings.HasSuffix(e.Name(), ".down.sql") {
			return e.Name()
		}
	}
	return ""
}

func (m *Migrator) runInTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// parseMigrationFilename extracts version and description from a filename
// like "20260526000001_create_extensions.up.sql".
func parseMigrationFilename(filename string) (version, description string) {
	base := filename
	// Strip direction suffix (.up.sql or .down.sql).
	for _, suffix := range []string{".up.sql", ".down.sql"} {
		base = strings.TrimSuffix(base, suffix)
	}
	idx := strings.Index(base, "_")
	if idx < 0 {
		return base, ""
	}
	return base[:idx], strings.ReplaceAll(base[idx+1:], "_", " ")
}
