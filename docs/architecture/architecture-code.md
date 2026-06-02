# Architecture — Code Design and Package Structure

**Audience:** Go engineers working in `aethel-core/`
**Status:** Active

Diagram: [docs/diagrams/architecture-code.mmd](../diagrams/architecture-code.mmd)

---

## Why Clean/Layered Architecture

The Go backend is structured as a set of concentric layers with a strict inward dependency rule. This choice is driven by two requirements that are unique to this project:

**Blueprint isolation.** The YAML blueprint system means that configuration types are loaded once at startup and then treated as immutable inputs to the rest of the system. If any layer could arbitrarily re-read blueprints or hold mutable references to them, reasoning about startup order and configuration correctness would become impossible. Layering forces the blueprint package to be a pure input — it has no dependencies, and everything else depends on it (or on types derived from it).

**Testability.** The routing rule engine, the green note hash chain validator, and the escalation evaluator all contain non-trivial logic that must be exercised in isolation, without a running database. Layered architecture makes this straightforward: the domain and service layers do not depend on `database` or `api`, so they can be tested with no infrastructure.

---

## Full Package Layout

```
aethel-core/
├── cmd/
│   └── aethel/
│       └── main.go              # binary entry point + CLI wiring (cobra)
├── internal/
│   ├── blueprint/
│   │   ├── loader.go            # load + validate YAML into typed structs
│   │   └── database_config.go   # DatabaseConfig struct (server-database.yaml only)
│   ├── config/
│   │   ├── cache.go             # ConfigCache: single config struct, 5-min TTL
│   │   ├── loader.go            # LoadOrgConfig: queries branding_configs + system_settings
│   │   └── handler.go           # GET /api/v1/config, PATCH /api/v1/admin/config/*
│   ├── domain/
│   │   ├── dispatch.go          # Dispatch, DispatchEvent, RoutingRule types
│   │   ├── workflow.go          # MinuteSheet, GreenNote types
│   │   ├── governance.go        # AuditEntry, TamperCheck types
│   │   ├── user.go              # User, Session, Permission types
│   │   └── errors.go            # sentinel errors (ErrNotFound, ErrForbidden, etc.)
│   ├── database/
│   │   ├── connect.go           # open *sql.DB from DatabaseConfig
│   │   ├── blueprint_context.go # T(), E(), Schema template helpers
│   │   ├── migrator.go          # migration runner
│   │   ├── query_registry.go    # load + prepare named queries at startup
│   │   ├── queries/
│   │   │   └── queries.yaml     # named SQL queries (internal developer file)
│   │   └── migrations/          # 42 SQL template files (up + down)
│   ├── service/
│   │   ├── dispatch_service.go  # routing rule engine, dispatch lifecycle
│   │   ├── workflow_service.go  # minute sheet management, green note chaining
│   │   ├── auth_service.go      # register, login, JWT issuance, session management
│   │   └── escalation_service.go # escalation rule evaluation (called by worker)
│   ├── worker/
│   │   └── escalation_worker.go # background goroutine: tick-based rule evaluation
│   ├── rbac/
│   │   └── middleware.go        # permission enforcement
│   ├── transport/
│   │   └── sse.go               # Server-Sent Events broker for notifications
│   └── api/
│       ├── server.go            # HTTP server wiring, route registration
│       └── handlers/            # one file per route group
│           ├── auth.go
│           ├── dispatch.go
│           ├── workflow.go
│           ├── governance.go
│           └── admin.go
└── blueprints -> ../blueprints  # symlink so the binary finds blueprints/
```

---

## Layer Responsibilities

### `cmd/` — CLI wiring only

`cmd/aethel/main.go` is the binary entry point. Its only job is to wire the startup sequence: parse CLI flags, load blueprints, open the database, run migrations, build the query registry, and start the HTTP server. It delegates every decision to the packages below.

`cmd/` uses `cobra` for subcommand routing (`aethel serve`, `aethel migrate up`, `aethel migrate status`, `aethel migrate validate`, `aethel migrate down`). No business logic lives here.

### `internal/blueprint/` — configuration types

Blueprint structs are the canonical Go representation of the YAML files. This package has **no dependencies on any other internal package**. It only imports the YAML decoder (`gopkg.in/yaml.v3`) and the standard library.

Blueprint structs are loaded once in `cmd/`, validated immediately, and then passed as value types (not pointers to mutable state) into every downstream package that needs them.

### `internal/domain/` — shared domain types

This package defines the core entities — `Dispatch`, `GreenNote`, `AuditEntry`, `User`, `Permission` — and the repository interfaces that the service layer depends on. It is imported by `database` (which implements the interfaces) and by `service` and `api` (which use the types). It never imports any other `internal/` package.

Domain types are plain Go structs with no framework dependencies. Repository interfaces are defined here, near their consumers, rather than in the database layer. This inverts the dependency so that the database package adapts to the domain's contract, not the other way around.

### `internal/database/` — infrastructure

This package owns everything that touches the database: `*sql.DB` management, migration rendering and execution, and the named query registry. It implements the repository interfaces defined in `internal/domain/`.

No package other than `internal/database/` may construct raw SQL strings. This rule enforces that all SQL is either in a migration file or in `internal/database/queries/queries.yaml`.

### `internal/service/` — application logic

Service functions orchestrate domain operations: the dispatch routing rule engine, the green note hash chain validator, authentication and JWT issuance, and escalation rule evaluation. Service functions call repository interfaces (defined in `domain/`) — they never import `database/` directly.

This indirection means service logic can be tested with a mock repository without a running database.

### `internal/worker/` — background processing

The escalation worker runs as a goroutine launched by `cmd/`. It ticks on a blueprint-configurable interval and calls `service.EvaluateEscalationRules`. It has no HTTP surface. Its only dependencies are `service/` and `domain/`.

### `internal/rbac/` — cross-cutting access control

The RBAC middleware reads the authenticated user's role (set on the request context by the auth middleware) and checks it against the required permission for the route. Permission strings are defined in `domain/` and referenced in `internal/database/queries/queries.yaml`.

`rbac/` is imported by `api/` (which wires the middleware) and by `service/` (for programmatic permission checks in service functions). It never imports `api/`.

### `internal/transport/` — SSE broker

The SSE (Server-Sent Events) broker manages open connections from browser clients waiting for real-time notifications. It exposes a `Publish(userID, event)` method called by service functions and a `ServeHTTP` handler registered by `api/`. It does not know about routes or authentication — those responsibilities stay in `api/`.

### `internal/api/` — HTTP delivery

Handlers in `api/handlers/` are thin: they parse the HTTP request, call the appropriate service function, and encode the response as JSON. They do not contain business logic. The middleware stack, route registration, and server lifecycle all live in `api/server.go`.

`internal/api/` is the only package that imports everything else. No package may import `internal/api/`.

---

## Dependency Rule

Data flows inward. The outer layer depends on the inner layer; the inner layer does not know about the outer.

```
cmd → api → rbac → service → domain ← database ← blueprint
                 ↑
              transport
```

Specifically:
- `api` imports `service`, `rbac`, `domain`, `transport`
- `service` imports `domain` only
- `database` imports `domain` and `blueprint`
- `rbac` imports `domain`
- `transport` imports `domain`
- `blueprint` imports nothing internal
- `domain` imports nothing internal

Circular imports are a compile error in Go. The architecture makes them structurally impossible given this layering.

---

## Blueprint-Driven Config Without Global Variables

Blueprint structs are injected via struct fields, not via `init()` or package-level variables. The pattern throughout the codebase is:

```go
type DispatchService struct {
    repo    domain.DispatchRepository
    queries *database.QueryRegistry
}

type Server struct {
    db      *sql.DB
    queries *database.QueryRegistry
    sse     *transport.SSEBroker
}
```

`main.go` constructs all dependencies, then passes them as constructor arguments. No package calls `os.Getenv` at request time or reads blueprint files after startup. The environment variable `AETHEL_ENV` is read exactly once in `main.go` during startup.

This approach has two benefits: it makes data flow explicit and auditable, and it makes testing straightforward because any struct can be constructed with mock dependencies.

---

## Error Handling Philosophy

**Fail fast at startup.** Any condition that makes the server non-functional — malformed blueprint, missing environment, database connection failure, named query that fails to prepare — must exit the process immediately with a non-zero status code and a clear error message. A process that starts in a broken state is more dangerous than one that refuses to start.

**Structured errors with context.** All errors are wrapped with `fmt.Errorf("operation: %w", err)` at each layer boundary. The chain of wrapping makes it possible to produce log messages that show the full call path without a stack trace in the log output.

**No panic in the hot path.** `panic` is reserved for programmer errors that are impossible under correct usage — a nil pointer that cannot be nil if the API contract is followed. The only acceptable panic in the hot path is `QueryRegistry.Get()` panicking on a missing key: this is a startup-time programmer error (the key exists in the code but not in the YAML), and it is caught on first boot, not in production.

**Sentinel errors for domain conditions.** `internal/domain/errors.go` defines sentinel errors: `ErrNotFound`, `ErrForbidden`, `ErrConflict`, `ErrHashChainBroken`. Service functions return these; handler functions translate them to the appropriate HTTP status code. This keeps HTTP concepts out of the service layer.

---

## Where Domain Types Live

All shared domain types live in `internal/domain/`. They are the single source of truth for what a `Dispatch`, `GreenNote`, or `AuditEntry` looks like across the codebase. Neither the database package nor the API handlers define their own competing structs for these entities.

Repository interfaces are also defined in `internal/domain/`, co-located with the types they operate on:

```go
// domain/dispatch.go
type DispatchRepository interface {
    GetByID(ctx context.Context, orgID, id uuid.UUID) (*Dispatch, error)
    ListInbox(ctx context.Context, orgID, deptID uuid.UUID, page Page) ([]Dispatch, error)
    Create(ctx context.Context, d *Dispatch) error
    UpdateStatus(ctx context.Context, orgID, id uuid.UUID, status DispatchStatus) error
}
```

The database package provides a concrete implementation. The service package calls the interface. This means the service package never imports `database` — it only depends on the interface contract.

---

## Conventions

**One file per major type.** `dispatch.go` contains only dispatch-related types and its repository interface. Do not aggregate unrelated types into a single file.

**No circular imports.** The compiler enforces this. The layering described above makes it structurally impossible if the dependency rule is followed.

**Interface definitions near their consumers.** Repository interfaces are defined in `domain/`, not in `database/`. HTTP handler interfaces (if any) are defined in `api/`, not in `service/`.

**No package-level `init()` with side effects.** `init()` functions that register things or perform I/O make startup order unpredictable. All initialization is explicit in `main.go`.
