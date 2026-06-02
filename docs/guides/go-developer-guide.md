# Go Developer Guide — Aethel Core

This document is the authoritative reference for engineers writing Go code
in `aethel-core/`. Read it before touching any file in that directory.

The guide covers the four non-obvious aspects of this codebase:

1. How blueprints are loaded into Go structs
2. How SQL migration files are rendered and executed
3. How named queries from `aethel-core/internal/database/queries/queries.yaml` are bound and called
4. How the runtime config API and in-memory cache work

Standard Go practices (error wrapping, context propagation, testing) are
assumed knowledge and are not repeated here.

---

## 1. Project layout

```
aethel-core/
├── cmd/
│   └── aethel/
│       └── main.go            # binary entry point + CLI wiring
├── internal/
│   ├── blueprint/
│   │   ├── loader.go          # load + validate YAML into typed structs
│   │   └── database_config.go # DatabaseConfig struct (server-database.yaml)
│   ├── config/
│   │   ├── cache.go           # ConfigCache: single config struct, 5-min TTL
│   │   ├── loader.go          # LoadConfig: queries branding_configs + system_settings
│   │   └── handler.go         # HTTP handlers for GET /api/v1/config and PATCH endpoints
│   ├── database/
│   │   ├── connect.go         # open *sql.DB from DatabaseConfig
│   │   ├── blueprint_context.go  # T(), E(), Schema template helpers
│   │   ├── migrator.go        # migration runner
│   │   ├── query_registry.go  # load + prepare named queries at startup
│   │   ├── queries/
│   │   │   └── queries.yaml   # named SQL queries (internal developer file)
│   │   └── migrations/        # 42 SQL template files (up + down)
│   ├── rbac/
│   │   └── middleware.go      # permission enforcement
│   └── api/
│       ├── server.go          # HTTP server wiring
│       └── handlers/          # one file per route group
└── blueprints -> ../blueprints  # symlink so the binary finds blueprints/
```

Two rules that apply everywhere in `internal/`:

- **No package may import `internal/api`** — the handler layer is a consumer
  of everything else, never a dependency.
- **No file outside `internal/database/` may construct raw SQL strings** —
  all SQL lives in the migration files or in `internal/database/queries/queries.yaml`.

---

## 2. Blueprint loading

### 2.1 Go structs

The IT-facing blueprint file (`server-database.yaml`) maps to a typed struct in `internal/blueprint/`. The queries file (`internal/database/queries/queries.yaml`) maps to `QueriesConfig`. Use `gopkg.in/yaml.v3` for decoding. Define structs to match the YAML exactly; do not invent fields that aren't in the conventions doc.

```go
// internal/blueprint/database_config.go

package blueprint

type DatabaseConfig struct {
    Metadata             Metadata                       `yaml:"metadata"`
    GlobalDatabaseDefaults GlobalDatabaseDefaults       `yaml:"global_database_defaults"`
    Environments         map[string]EnvironmentConfig   `yaml:"environments"`
    Schema               SchemaConfig                   `yaml:"schema"`
    Partitioning         map[string]PartitionConfig     `yaml:"partitioning"`
    Extensions           ExtensionConfig                `yaml:"extensions"`
    Performance          PerformanceConfig              `yaml:"performance"`
}

type Metadata struct {
    Version          string `yaml:"version"`
    EngineTarget     string `yaml:"engine_target"`
    StrictValidation bool   `yaml:"strict_validation"`
}

type EnvironmentConfig struct {
    Connection ConnectionConfig `yaml:"connection"`
    Pooling    PoolingConfig    `yaml:"pooling"`
    Migrations MigrationConfig  `yaml:"migrations"`
}

type ConnectionConfig struct {
    Host                string `yaml:"host"`
    Port                int    `yaml:"port"`
    Database            string `yaml:"database"`
    User                string `yaml:"user"`
    SSLMode             string `yaml:"ssl_mode"`
    SSLRootCertPath     string `yaml:"ssl_root_cert_path"`
    ConnectionStringEnv string `yaml:"connection_string_env"`
}

type PoolingConfig struct {
    MaxOpenConnections          int `yaml:"max_open_connections"`
    MaxIdleConnections          int `yaml:"max_idle_connections"`
    ConnectionMaxLifetimeMinutes int `yaml:"connection_max_lifetime_minutes"`
    ConnectionMaxIdleTimeMinutes int `yaml:"connection_max_idle_time_minutes"`
}

type MigrationConfig struct {
    Directory          string `yaml:"directory"`
    AutoRunOnStartup   bool   `yaml:"auto_run_on_startup"`
    TableName          string `yaml:"table_name"`
    LockTimeoutSeconds int    `yaml:"lock_timeout_seconds"`
}

type SchemaConfig struct {
    DefaultSchema string            `yaml:"default_schema"`
    NameAliases   map[string]string `yaml:"name_aliases"`
    EnumAliases   map[string]string `yaml:"enum_aliases"`
}

type PerformanceConfig struct {
    StatementTimeoutMs            int `yaml:"statement_timeout_ms"`
    IdleInTransactionTimeoutMs    int `yaml:"idle_in_transaction_timeout_ms"`
    LockTimeoutMs                 int `yaml:"lock_timeout_ms"`
}
```

```go
// internal/blueprint/queries_config.go

package blueprint

type QueriesConfig struct {
    Metadata            Metadata                      `yaml:"metadata"`
    GlobalQueryDefaults GlobalQueryDefaults            `yaml:"global_query_defaults"`
    Queries             map[string]map[string]Query    `yaml:"queries"`
}

type GlobalQueryDefaults struct {
    TimeoutMs              int  `yaml:"timeout_ms"`
    EnableQueryPlanCaching bool `yaml:"enable_query_plan_caching"`
    MaxRowsPerPage         int  `yaml:"max_rows_per_page"`
}

// Query is a single named query from server-queries.yaml.
type Query struct {
    Statement          string        `yaml:"statement"`
    Params             []QueryParam  `yaml:"params"`
    TimeoutMs          int           `yaml:"timeout_ms"`
    CacheTTLSeconds    int           `yaml:"cache_ttl_seconds"`
    RequiredPermission string        `yaml:"required_permission"`
    Description        string        `yaml:"description"`
}

type QueryParam struct {
    Name     string `yaml:"name"`
    Type     string `yaml:"type"`
    Nullable bool   `yaml:"nullable"`
}
```

### 2.2 Loader

```go
// internal/blueprint/loader.go

package blueprint

import (
    "fmt"
    "os"

    "gopkg.in/yaml.v3"
)

func LoadDatabaseConfig(path string) (*DatabaseConfig, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("read blueprint %s: %w", path, err)
    }
    var cfg DatabaseConfig
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("parse blueprint %s: %w", path, err)
    }
    if err := validateDatabaseConfig(&cfg); err != nil {
        return nil, fmt.Errorf("invalid blueprint %s: %w", path, err)
    }
    return &cfg, nil
}

func LoadQueriesConfig(path string) (*QueriesConfig, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("read blueprint %s: %w", path, err)
    }
    var cfg QueriesConfig
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("parse blueprint %s: %w", path, err)
    }
    return &cfg, nil
}

func validateDatabaseConfig(cfg *DatabaseConfig) error {
    if cfg.Metadata.Version == "" {
        return fmt.Errorf("metadata.version is required")
    }
    if len(cfg.Environments) == 0 {
        return fmt.Errorf("at least one environment is required")
    }
    if cfg.Schema.DefaultSchema == "" {
        return fmt.Errorf("schema.default_schema is required")
    }
    return nil
}
```

### 2.3 Active environment selection

The active environment is read from `AETHEL_ENV`. Do this once at startup;
never re-read the environment variable at request time.

```go
env := os.Getenv("AETHEL_ENV")
if env == "" {
    env = "development"
}
envCfg, ok := dbCfg.Environments[env]
if !ok {
    log.Fatalf("blueprint: environment %q not defined in server-database.yaml", env)
}
```

---

## 3. Database connection

Build the DSN from the active environment config. Respect the two password
injection paths: full DSN via env var, or individual fields plus
`AETHEL_DB_PASSWORD`.

```go
// internal/database/connect.go

package database

import (
    "database/sql"
    "fmt"
    "os"
    "time"

    _ "github.com/lib/pq"
    "aethel-core/internal/blueprint"
)

func Open(cfg blueprint.EnvironmentConfig) (*sql.DB, error) {
    dsn, err := buildDSN(cfg.Connection)
    if err != nil {
        return nil, err
    }
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        return nil, fmt.Errorf("open db: %w", err)
    }

    p := cfg.Pooling
    db.SetMaxOpenConns(p.MaxOpenConnections)
    db.SetMaxIdleConns(p.MaxIdleConnections)
    db.SetConnMaxLifetime(time.Duration(p.ConnectionMaxLifetimeMinutes) * time.Minute)
    db.SetConnMaxIdleTime(time.Duration(p.ConnectionMaxIdleTimeMinutes) * time.Minute)

    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("ping db: %w", err)
    }
    return db, nil
}

func buildDSN(c blueprint.ConnectionConfig) (string, error) {
    // Full DSN via env var takes precedence over individual fields.
    if c.ConnectionStringEnv != "" {
        dsn := os.Getenv(c.ConnectionStringEnv)
        if dsn == "" {
            return "", fmt.Errorf(
                "connection_string_env=%q is set but the env var is empty",
                c.ConnectionStringEnv,
            )
        }
        return dsn, nil
    }

    password := os.Getenv("AETHEL_DB_PASSWORD")
    return fmt.Sprintf(
        "host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
        c.Host, c.Port, c.Database, c.User, password, c.SSLMode,
    ), nil
}
```

---

## 4. Migration system

### 4.1 BlueprintContext — template helpers

Every SQL migration file is a Go `text/template`. Three substitutions are
available:

| Template syntax | Resolves to |
|---|---|
| `{{ .Schema }}` | `schema.default_schema` from the blueprint |
| `{{ T "dispatches" }}` | alias for `dispatches`, or `"dispatches"` if no alias |
| `{{ E "priority_level" }}` | alias for `priority_level`, or the canonical name |

`T` and `E` must be registered as template `FuncMap` entries, not as
methods called through `{{ call }}`. This is the only way to use the
`{{ T "name" }}` call syntax in Go templates.

```go
// internal/database/blueprint_context.go

package database

import (
    "text/template"
    "aethel-core/internal/blueprint"
)

// BlueprintContext holds the values injected into SQL templates.
type BlueprintContext struct {
    Schema string
    tables map[string]string
    enums  map[string]string
}

func NewBlueprintContext(cfg *blueprint.DatabaseConfig) *BlueprintContext {
    return &BlueprintContext{
        Schema: cfg.Schema.DefaultSchema,
        tables: cfg.Schema.NameAliases,
        enums:  cfg.Schema.EnumAliases,
    }
}

// FuncMap returns a template.FuncMap that exposes T and E.
// Call this when parsing each migration template.
func (b *BlueprintContext) FuncMap() template.FuncMap {
    return template.FuncMap{
        "T": func(canonical string) string {
            if alias, ok := b.tables[canonical]; ok && alias != "" {
                return alias
            }
            return canonical
        },
        "E": func(canonical string) string {
            if alias, ok := b.enums[canonical]; ok && alias != "" {
                return alias
            }
            return canonical
        },
    }
}
```

### 4.2 Rendering a migration file

```go
func renderMigration(content []byte, ctx *BlueprintContext) (string, error) {
    tmpl, err := template.New("migration").
        Funcs(ctx.FuncMap()).
        Parse(string(content))
    if err != nil {
        return "", fmt.Errorf("parse template: %w", err)
    }
    var buf strings.Builder
    if err := tmpl.Execute(&buf, ctx); err != nil {
        return "", fmt.Errorf("render template: %w", err)
    }
    return buf.String(), nil
}
```

`ctx` is passed as the template's data (`{{ .Schema }}` works because
`Schema` is an exported field on `BlueprintContext`). `T` and `E` work
because they are in the `FuncMap`.

### 4.3 Migrator

```go
// internal/database/migrator.go

package database

import (
    "context"
    "crypto/sha256"
    "database/sql"
    "fmt"
    "io/fs"
    "log/slog"
    "path/filepath"
    "sort"
    "strconv"
    "strings"

    "aethel-core/internal/blueprint"
)

type Migrator struct {
    db          *sql.DB
    ctx         *BlueprintContext
    migrationsDir string
    historyTable  string
    lockTimeout   int // seconds
}

func NewMigrator(
    db *sql.DB,
    cfg *blueprint.DatabaseConfig,
    envCfg blueprint.EnvironmentConfig,
) *Migrator {
    return &Migrator{
        db:            db,
        ctx:           NewBlueprintContext(cfg),
        migrationsDir: envCfg.Migrations.Directory,
        historyTable:  envCfg.Migrations.TableName,
        lockTimeout:   envCfg.Migrations.LockTimeoutSeconds,
    }
}

func (m *Migrator) Up(ctx context.Context) error {
    // 1. Advisory lock — prevents concurrent migration runs.
    if err := m.acquireLock(ctx); err != nil {
        return err
    }
    defer m.releaseLock(ctx)

    // 2. Ensure the history table exists (not aliased — must survive before
    //    aliases are resolved).
    if err := m.ensureHistoryTable(ctx); err != nil {
        return err
    }

    // 3. Collect all *.up.sql files, sorted lexicographically.
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

        // 4. Execute inside a single transaction.
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
```

Key invariants to maintain in any future changes to `Migrator`:

- The advisory lock is always acquired before reading the history table.
- The history table name is **never** run through `T()` — it must be
  addressable before aliases are resolved.
- Every migration executes inside a single transaction. If the DDL fails,
  the history row is never inserted. If the history insert fails, the DDL
  is rolled back.
- The `checksum` is the SHA-256 of the **rendered** SQL (after template
  substitution), not the raw template. This lets the runner detect when a
  template was edited after application, which is a dangerous operation.

### 4.4 Writing a new migration file

Follow the naming convention exactly:

```
{YYYYMMDDHHmmss}_{description}.{up|down}.sql
```

Inside the file, use template syntax for every identifier that could be
aliased by IT:

```sql
-- Always: schema name
{{ .Schema }}.some_function()

-- Table names — every reference, including in FKs and indexes
CREATE TABLE {{ .Schema }}.{{ T "dispatches" }} ( ... );
CREATE INDEX {{ T "dispatches" }}_org_idx ON {{ .Schema }}.{{ T "dispatches" }} ( ... );
REFERENCES {{ .Schema }}.{{ T "organizations" }} (id)

-- Enum type names
{{ .Schema }}.{{ E "priority_level" }}
```

Never hard-code a table or enum name as a bare string. The only identifiers
that may be bare strings are:

- Column names (columns are not aliasable)
- Constraint names that embed a table name via `{{ T "..." }}` (the
  constraint name itself uses the template; the embedded part is resolved)
- The history table name inside the migrator's own SQL

---

## 5. Query registry

`server-queries.yaml` externalises complex named queries. The query
registry loads these at startup, prepares them, and exposes a typed lookup
interface to the repository layer.

### 5.1 Registry struct

```go
// internal/database/query_registry.go

package database

import (
    "context"
    "database/sql"
    "fmt"

    "aethel-core/internal/blueprint"
)

// PreparedQuery holds the compiled statement and its metadata.
type PreparedQuery struct {
    Stmt        *sql.Stmt
    TimeoutMs   int
    Permission  string
}

// QueryRegistry maps "group.name" → PreparedQuery.
type QueryRegistry struct {
    stmts   map[string]*PreparedQuery
    defaults blueprint.GlobalQueryDefaults
}

func BuildQueryRegistry(
    ctx context.Context,
    db *sql.DB,
    cfg *blueprint.QueriesConfig,
) (*QueryRegistry, error) {
    reg := &QueryRegistry{
        stmts:   make(map[string]*PreparedQuery),
        defaults: cfg.GlobalQueryDefaults,
    }

    for group, queries := range cfg.Queries {
        for name, q := range queries {
            key := group + "." + name

            stmt, err := db.PrepareContext(ctx, q.Statement)
            if err != nil {
                return nil, fmt.Errorf("prepare query %q: %w", key, err)
            }

            timeoutMs := q.TimeoutMs
            if timeoutMs == 0 {
                timeoutMs = cfg.GlobalQueryDefaults.TimeoutMs
            }

            reg.stmts[key] = &PreparedQuery{
                Stmt:       stmt,
                TimeoutMs:  timeoutMs,
                Permission: q.RequiredPermission,
            }
        }
    }
    return reg, nil
}

// Get returns the prepared query for the given "group.name" key.
// Panics if the key is not found — a missing key is a programmer error,
// not a runtime condition.
func (r *QueryRegistry) Get(key string) *PreparedQuery {
    pq, ok := r.stmts[key]
    if !ok {
        panic(fmt.Sprintf("query registry: key %q not found — check server-queries.yaml", key))
    }
    return pq
}
```

### 5.2 Using a named query in a repository

```go
// internal/api/handlers/dispatch.go

func (h *DispatchHandler) ListInbox(w http.ResponseWriter, r *http.Request) {
    pq := h.queries.Get("dispatch.fetch_active_inbox")

    ctx, cancel := context.WithTimeout(r.Context(),
        time.Duration(pq.TimeoutMs)*time.Millisecond)
    defer cancel()

    rows, err := pq.Stmt.QueryContext(ctx,
        r.Context().Value(ctxKeyOrgID),  // $1: organization_id
        r.Context().Value(ctxKeyUserID), // $2: user_id
    )
    // ... scan rows
}
```

The parameter order must match the `params` list in `server-queries.yaml`
exactly. There is no named-parameter binding; positional `$N` is the
PostgreSQL standard.

### 5.3 When to add a query to `server-queries.yaml`

Add a query when **any** of these are true:

- It contains a JOIN across more than two tables
- IT admins are likely to want to tune it (different sort, extra filter)
- It is used in a reporting or search context where the plan matters
- It is executed on every page load (hot path)

Simple single-table CRUD (`INSERT`, `UPDATE`, `DELETE`, `SELECT … WHERE id = $1`)
belongs in the repository layer as a Go string constant, not in the blueprint.

---

## 6. Startup sequence

The `main.go` wires everything together in this exact order. Deviating from
the order causes nil pointer panics or races.

```go
func main() {
    // 1. Load blueprints (fail fast on invalid YAML)
    dbCfg, err := blueprint.LoadDatabaseConfig("blueprints/server-database.yaml")
    exitOnError(err)
    queriesCfg, err := blueprint.LoadQueriesConfig(
        "internal/database/queries/queries.yaml")
    exitOnError(err)

    // 2. Select active environment
    env := os.Getenv("AETHEL_ENV")
    if env == "" { env = "development" }
    envCfg := dbCfg.Environments[env]

    // 3. Open database connection
    db, err := database.Open(envCfg)
    exitOnError(err)
    defer db.Close()

    // 4. Run migrations (only when auto_run_on_startup: true)
    if envCfg.Migrations.AutoRunOnStartup {
        migrator := database.NewMigrator(db, dbCfg, envCfg)
        exitOnError(migrator.Up(context.Background()))
    }

    // 5. Seed branding + nav from blueprints (idempotent — skips if already seeded)
    seeder := config.NewSeeder(db)
    exitOnError(seeder.SeedBranding(context.Background(), "blueprints/ui-theme.yaml"))
    exitOnError(seeder.SeedNav(context.Background(), "blueprints/ui-layouts.yaml"))

    // 6. Build query registry (prepares all named statements)
    queries, err := database.BuildQueryRegistry(context.Background(), db, queriesCfg)
    exitOnError(err)

    // 7. Initialize config cache
    configCache := config.NewConfigCache()

    // 8. Wire HTTP server
    srv := api.NewServer(db, queries, configCache)
    exitOnError(srv.ListenAndServe(":8080"))
}
```

---

## 7. RBAC enforcement

Permissions are defined in `internal/database/queries/queries.yaml` under `required_permission`
and in `internal/rbac/`. The four roles are `ADMIN`, `RECEPTION`, `USER`,
`SYS_ADMIN`. Middleware reads the permission from the query's metadata and
checks it against the authenticated user's role.

The permission string format is `<domain>.<action>` (e.g., `dispatch.view`,
`admin.audit`). Do not invent new permission identifiers without adding them
to the conventions doc.

```go
// Pattern for a route that uses a named query's permission:
func RequireQueryPermission(queries *database.QueryRegistry, key string) func(http.Handler) http.Handler {
    pq := queries.Get(key) // panics at startup if key is missing
    return rbac.Require(pq.Permission)
}
```

Wiring middleware at server startup (not at request time) guarantees that a
misconfigured permission string fails the process at boot, not on a live
request.

---

## 8. Coding conventions

### SQL is always in a file or in the blueprint

| Location | What goes there |
|---|---|
| `migrations/*.sql` | DDL — CREATE TABLE, CREATE INDEX, ALTER, DROP |
| `server-queries.yaml` | Complex read queries, hot-path queries, reportable queries |
| Repository Go file | Simple CRUD: single-table INSERT/UPDATE/DELETE, `SELECT … WHERE id = $1` |
| Anywhere else | Nothing |

A SQL string literal that appears outside these three locations is a code
smell. If you find yourself writing a JOIN in a handler, move it to the
blueprint.

### Blueprint structs are read-only after startup

Load them once in `main`, pass them down as immutable values. Never hold a
pointer to a blueprint struct in a goroutine that could outlive the startup
phase, and never write to a blueprint struct at runtime.

### `T()` and `E()` are for migrations only

`BlueprintContext.FuncMap()` is only called from `Migrator`. The query
registry uses the prepared statements from `server-queries.yaml` which are
already fully rendered SQL. Do not call `T()` or `E()` in handler code.

### Environment variables for secrets, YAML for structure

The blueprint files configure behaviour — timeouts, pool sizes, schema
names. They never store secrets. If you need a new secret at runtime, add
an environment variable and document it in `docs/guides/server-blueprint-conventions.md`
under §3.2.

### Blueprint validation happens at startup, not at request time

`validateDatabaseConfig` runs once when the blueprint is loaded. Do not
add validation logic that re-reads the blueprint on each HTTP request.

### Fail fast on misconfiguration

`main.go` must exit with a non-zero status if any of these fail:

- Blueprint YAML is malformed or fails validation
- Required environment is not defined in the blueprint
- Database connection or ping fails
- Any named query in `internal/database/queries/queries.yaml` fails to prepare

Use `log.Fatalf` or `os.Exit(1)` for these cases. A server that starts
with a broken configuration is worse than one that refuses to start.

---

## 9. Adding a new named query

1. Write the SQL and test it directly in `psql`.
2. Add the query to `aethel-core/internal/database/queries/queries.yaml` under the correct
   group, following the conventions in `docs/guides/server-blueprint-conventions.md`.
3. Define the `params` list to match the positional `$1`, `$2` ... order.
4. Set `required_permission` to an existing permission identifier.
5. Restart the server — `BuildQueryRegistry` will prepare the statement
   at startup and panic if the SQL is rejected by PostgreSQL.
6. Call it from the handler via `h.queries.Get("group.name")`.

Never skip step 5's implicit validation. A query that the registry fails to
prepare at startup is infinitely better than one that panics at 3am on a
live request.

---

## 10. Adding a new migration

1. Pick a UTC timestamp: `date -u +"%Y%m%d%H%M%S"`.
2. Create `migrations/{timestamp}_{description}.up.sql` and
   `migrations/{timestamp}_{description}.down.sql`.
3. Use `{{ .Schema }}`, `{{ T "..." }}`, `{{ E "..." }}` for every
   aliasable identifier. Never hard-code a table name.
4. Validate the template renders without error:
   ```bash
   go run ./cmd/aethel migrate validate
   ```
5. Apply in development:
   ```bash
   go run ./cmd/aethel migrate up
   ```
6. Write the down migration before committing. A migration without a
   rollback path must not be merged.

### Down migration rules

- `DROP TABLE IF EXISTS` in reverse dependency order (child tables first).
- `DROP TYPE IF EXISTS` for any enum created in the corresponding up.
- `DROP TRIGGER`, then `DROP FUNCTION` for the trigger functions migration.
- `IF EXISTS` on every DROP — the down migration must be idempotent.

---

## 11. Config API and In-Memory Cache

The `internal/config/` package (implemented in Sprint 2) owns the runtime configuration layer.

### Package structure

```
internal/config/
├── cache.go    — ConfigCache struct: sync.RWMutex, map[uuid.UUID]*CachedConfig, TTL
├── loader.go   — LoadOrgConfig(ctx, db, orgID) → OrgConfig
└── handler.go  — HTTP handlers: GET /api/v1/config*, PATCH /api/v1/admin/config/*
```

### Go struct for config response

The `OrgConfig` struct mirrors the TypeScript shape expected by `useRuntimeConfig()` in the frontend:

```go
// internal/config/loader.go

type OrgConfig struct {
    Branding BrandingConfig `json:"branding"`
    Nav      []NavGroup     `json:"nav"`
    Features FeatureFlags   `json:"features"`
    Org      OrgProfile     `json:"org"`
}

type BrandingConfig struct {
    PrimaryColor   string `json:"primaryColor"`
    NeutralPalette string `json:"neutralPalette"`
    FontFamily     string `json:"fontFamily"`
    Wordmark       string `json:"wordmark"`
    LogoPath       string `json:"logoPath"`
}

type NavGroup struct {
    Group      string    `json:"group"`
    Permission string    `json:"permission,omitempty"`
    Roles      []string  `json:"roles"`
    Items      []NavItem `json:"items"`
}

type NavItem struct {
    Label  string `json:"label"`
    Icon   string `json:"icon"`
    Route  string `json:"route"`
}
```

### Cache operations

```go
// cache.Invalidate is called by every successful PATCH handler
cache.Invalidate(orgID uuid.UUID)

// cache.Get returns (config, ok) — ok=false means cache miss, must load from DB
config, ok := cache.Get(orgID)

// cache.Set stores config with TTL
cache.Set(orgID, config, 5*time.Minute)
```

### Cache invalidation pattern

PATCH handlers follow this exact sequence:

1. Validate and parse the request body.
2. Write to PostgreSQL (`branding_configs` or `system_settings`).
3. On success: `cache.Invalidate(orgID)`.
4. Return the updated config as the response body (avoids a second DB read by the frontend).

The frontend's `useRuntimeConfig()` composable calls `appConfig.refresh()` on a successful PATCH response to update the Pinia/useState cache immediately.

---

## 12. Reference

| Document | Location |
|---|---|
| Database ER diagram | `docs/db-design.mmd` |
| Blueprint YAML field reference | `docs/guides/server-blueprint-conventions.md` |
| Migration system design (rationale) | `docs/plans/migration-strategy.md` |
| IT customisation guide | `docs/guides/it-customization-guide.md` |
| DB blueprint file | `blueprints/server-database.yaml` |
| Named queries file | `aethel-core/internal/database/queries/queries.yaml` |
| Migration SQL files | `aethel-core/internal/database/migrations/` |
| Runtime config flow diagram | `docs/diagrams/runtime-config-flow.mmd` |
