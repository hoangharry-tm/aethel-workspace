# Architecture — Server Components and Request Flow

**Audience:** Go engineers working in `aethel-core/`
**Status:** Active

Diagram: [docs/diagrams/server-request-flow.mmd](../diagrams/server-request-flow.mmd)

---

## Components

| Component | Package | Role |
|---|---|---|
| Blueprint loader | `internal/blueprint/` | Reads and validates YAML at startup; produces immutable config structs |
| HTTP router | `internal/api/server.go` | `chi` router; registers route groups and sub-routers |
| Middleware stack | `internal/api/server.go` | Ordered chain applied globally and per route group |
| Handler layer | `internal/api/handlers/` | Parses requests, calls services, encodes JSON responses |
| Service layer | `internal/service/` | Business logic: routing rules, hash chain validation, auth |
| Repository layer | `internal/database/` | Implements `domain` interfaces; executes SQL via query registry |
| Query registry | `internal/database/query_registry.go` | Prepares named statements from `server-queries.yaml` at startup |
| Background worker | `internal/worker/escalation_worker.go` | Periodic escalation rule evaluation |
| SSE broker | `internal/transport/sse.go` | Manages open SSE connections; publishes notification events |

---

## Why chi

`chi` is a lightweight HTTP router that is 100% compatible with the stdlib `net/http` interface. Every handler is a plain `http.Handler`. This matters because:

- Middleware written for `chi` works on any `net/http` mux.
- The router itself has no opinions about serialization, validation, or dependency injection — those decisions remain in our code.
- Sub-routers (`chi.Router`) allow scoped middleware application: the admin sub-router applies `rbac.Require("admin.access")` to every route under `/api/v1/admin/` without repeating the middleware on each individual route.
- The route pattern syntax supports typed parameter constraints (`{id:[0-9a-f-]{36}}`), which validates UUID path parameters before the handler runs.

No framework generators, no ORM, no global request context: the stdlib remains the foundation.

---

## Middleware Stack

The following middleware are applied in this exact order. Order is not negotiable — each middleware depends on values set by the one before it.

```
[Recovery]        → catches panics, returns 500, logs stack trace
[RequestID]       → generates or propagates X-Request-ID header; sets on context
[StructuredLogger]→ logs method, path, status, latency, request ID (zerolog)
[RateLimiter]     → per-IP token bucket; rejects with 429 if exhausted
[CORS]            → sets Access-Control-* headers; handles preflight OPTIONS
[Auth]            → parses JWT from Authorization: Bearer; sets user on context
[RBAC]            → checks user's role against route's required permission
[Handler]         → application logic
```

**Recovery** must be first because a panic in any downstream middleware or handler would otherwise propagate to the Go HTTP server and produce an empty response with no status code.

**RequestID** must be second so that all subsequent log lines (from the logger and from handler code) carry the same correlation ID.

**StructuredLogger** is placed after RequestID so every log line includes the ID, and after Recovery so it can log the latency of recovered requests.

**RateLimiter** runs before Auth intentionally. Unauthenticated rate limiting prevents credential-stuffing attacks from consuming connection slots while JWT parsing occurs.

**Auth** runs before RBAC because the role is read from the JWT claims.

**RBAC** is the last middleware before the handler. If it passes, the request is authorized.

---

## Authentication

**JWT-based.** The server issues signed JWTs on login. No session cookie is used for API clients. Browser clients that need CSRF protection use the double-submit cookie pattern (see `docs/architecture/architecture-security.md`).

**Algorithm choice:** HS256 (shared secret) or RS256 (asymmetric), selected via environment variable `AETHEL_JWT_ALGORITHM` (default: `RS256`).

The key material is always read from environment variables at startup (`AETHEL_JWT_SECRET` for HS256, `AETHEL_JWT_RSA_PRIVATE_KEY` for RS256). It is never in YAML.

**Token pattern:** Two tokens are issued on login:

- **Access token** — short-lived (configurable, default 15 minutes). Sent in `Authorization: Bearer` on every API request. Stateless — the server verifies the signature and expiry without querying the database.
- **Refresh token** — long-lived (configurable, default 7 days). Stored as an opaque token in the `user_sessions` table. Exchanged at `/api/v1/auth/refresh` for a new access token.

This separation means the access token path (every authenticated request) is database-free and fast, while the refresh path hits the database only once every 15 minutes per client.

---

## Sessions

Sessions are stored in the `user_sessions` table (see migration 04). There is no Redis dependency. This is a deliberate choice for small-to-medium deployments: a single PostgreSQL table is operationally simpler than a Redis cluster, survives process restarts, and is visible to audit queries.

The `user_sessions` table stores: `id`, `user_id`, `session_token_hash` (SHA-256 of the opaque token), `client_ip_address`, `user_agent`, `expires_at`, `created_at`.

If a refresh token is revoked (logout, admin action, detected anomaly), the session row is soft-deleted by setting `revoked_at`. The refresh endpoint checks this before issuing a new access token.

Session storage can be swapped to Redis in a future version without changing the service layer — the service layer calls the `SessionRepository` interface defined in `internal/domain/`.

---

## Rate Limiting

Rate limiting uses an in-process token bucket. No Redis is required. The implementation is per-IP (for unauthenticated requests) and per-user (for authenticated requests). Default limits are hardcoded: 600 RPM globally, 300 RPM for mutation-heavy dispatch endpoints.

In-process rate limiting is appropriate for small-scale deployments where a single Go process handles all traffic. For horizontally scaled deployments behind a load balancer, the IT admin should enable rate limiting at the reverse proxy layer (Nginx `limit_req_zone`, Caddy's rate limit plugin).

---

## Real-Time Notifications

Notifications are delivered via Server-Sent Events (SSE) at `GET /api/v1/notifications/stream`. Each authenticated user who opens this endpoint gets a persistent connection. The `transport.SSEBroker` manages the set of open connections, keyed by user ID.

When a service function produces a notification (a new dispatch is assigned, a green note is added, an escalation fires), it calls `broker.Publish(userID, event)`. The broker writes the event to the appropriate connection. If the user has no open connection, the event is dropped (the next page load will fetch missed notifications from the `notifications` table via the REST endpoint).

SSE is chosen over WebSockets for the following reasons:
- SSE is unidirectional (server to client), which matches the notification use case exactly.
- SSE uses plain HTTP/1.1; no protocol upgrade, no websocket frames, no heartbeat negotiation.
- SSE reconnects automatically in the browser without client-side logic.
- SSE is supported natively in all modern browsers without a library.

---

## Background Worker

The escalation worker runs as a goroutine started in `cmd/aethel/main.go` after the HTTP server is wired:

```
ticker := time.NewTicker(interval)  // interval from blueprint
go worker.RunEscalationWorker(ctx, ticker, escalationService)
```

On each tick, the worker calls `service.EvaluateEscalationRules`, which queries for dispatches that have exceeded their escalation threshold and applies the configured escalation action (reassign, notify, escalate status). The default tick interval is 60 seconds; this will be configurable via `system_settings` in a future sprint.

The worker respects context cancellation: when the server receives SIGTERM, the context is cancelled, and the worker exits cleanly before the process terminates.

---

## Request Lifecycle

The following is a step-by-step narrative of what happens from the moment a TCP connection is accepted to the moment a JSON response is written.

1. The Go stdlib `net/http` server accepts the TCP connection and hands it to the `chi` router.
2. The **Recovery** middleware wraps the handler call in a deferred recover. If anything panics below, it logs the stack trace and writes a `500 Internal Server Error`.
3. The **RequestID** middleware checks for an incoming `X-Request-ID` header. If absent, it generates a UUID. It sets the ID on the request context and on the response header.
4. The **StructuredLogger** middleware records the start time and stores a zerolog logger (with `request_id`, `method`, `path`, `remote_addr`) on the context. When the handler returns, it logs `status`, `latency`, and `bytes_written`.
5. The **RateLimiter** middleware checks the per-IP bucket. If the bucket is empty, it writes `429 Too Many Requests` with a `Retry-After` header and stops processing. Otherwise it decrements the bucket and continues.
6. The **CORS** middleware handles `OPTIONS` preflight requests immediately, returning the configured `Access-Control-*` headers. For non-OPTIONS requests, it appends the headers and continues.
7. The **Auth** middleware reads the `Authorization: Bearer <token>` header. It parses and validates the JWT signature and expiry. On failure, it writes `401 Unauthorized`. On success, it stores the `userID` and `role` claims on the request context.
8. The **RBAC** middleware reads the route's required permission (registered at startup) and the user's role from context. It consults the permission table in `internal/rbac/`. On failure, it writes `403 Forbidden` and logs an `RBAC_DENIED` audit event. On success, it continues.
9. The **Handler** function runs. It reads typed values from the request context (user ID), calls the appropriate service function, and encodes the result as JSON with the correct status code.
10. The service function calls repository methods (via the domain interface). The repository uses the query registry to execute prepared SQL statements against PostgreSQL.
12. The response is written. The logger middleware (deferred from step 4) fires and records the outcome.

---

## Runtime Configuration Fetch Flow

The Go backend maintains a single in-memory config cache to serve branding, navigation, and feature flags to the Nuxt SSR frontend with minimal database load. Because Aethel is a single-tenant self-hosted application, there is always exactly one config to cache.

### Cache structure

```go
// internal/config/cache.go
type ConfigCache struct {
    mu        sync.RWMutex
    config    *AppConfig
    expiresAt time.Time
}
```

- Single struct — no map, no org key.
- TTL is 5 minutes.
- `Invalidate()` clears the cached value; the next request re-fetches from DB.

### Request flow

`GET /api/v1/config` handler:

1. `cache.Get()` — cache hit returns immediately without a DB query.
2. On cache miss: call `loader.LoadConfig(ctx, db)`, which queries `branding_configs` + `system_settings WHERE key = 'nav_config'` and constructs the response struct.
3. Store result in cache with `ExpiresAt = now + 5m`.
4. Return the `AppConfig` as JSON.

### Nuxt SSR integration

The Nuxt frontend fetches config server-side during SSR:

```ts
// app/composables/useRuntimeConfig.ts
const { data: appConfig } = await useAsyncData(
  'app-config',
  () => $fetch('/api/v1/config')
)
```

Because this runs on the Nuxt server (not in the browser), the config JSON is embedded directly in the initial HTML payload (`__NUXT_DATA__`). The browser client receives config with the first byte of HTML — zero extra round-trips on first load.

Subsequent client-side navigations use the Pinia/useState cache seeded from the SSR data. No browser-to-API config fetch occurs unless an admin triggers a change.

### Admin save and cache invalidation

When an admin saves config via a PATCH endpoint:

1. The Go handler writes the update to PostgreSQL.
2. On success, it calls `cache.Invalidate(orgID)`.
3. The response body contains the updated config JSON.
4. The frontend calls `appConfig.refresh()` from `useRuntimeConfig()` to update local state immediately — no full page reload required.

### Why this is better than 2-way HTTP

**No FOUC (flash of unconfigured content):** Config arrives embedded in the first HTML response. A purely client-side fetch would cause the page to render with default styling before the config loads.

**DB load proportional to config changes, not page views:** A busy organization with 500 concurrent users generates at most one DB config read per 5 minutes per org — not one read per page view.

**No extra network round-trip:** The browser does not need to make a second HTTP request before rendering the page. Latency for first meaningful paint is unaffected by config fetching.

---

## Health and Readiness Probes

Two endpoints are registered outside the authenticated route tree:

- `GET /healthz` — liveness probe. Returns `200 OK` if the process is running. Does not check the database.
- `GET /readyz` — readiness probe. Returns `200 OK` only if the database is reachable (`db.PingContext`). Returns `503 Service Unavailable` otherwise. Kubernetes should route traffic only when this returns 200.

These endpoints are not rate-limited and not authenticated.
