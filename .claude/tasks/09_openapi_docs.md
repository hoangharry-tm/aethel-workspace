# Task 09 — OpenAPI 3.1 Specification + Scalar API Explorer

**Working directory:** `aethel-workspace/` (repo root)  
**Primary target:** `aethel-core/`  
**Depends on:** Task 06 complete (`go build ./...` clean), Task 08 security corrections applied  
**New route served:** `GET /api/docs` (Scalar UI), `GET /api/docs/openapi.yaml` (raw spec)

---

## Your Role

You are a backend engineer responsible for API documentation. Your deliverable is a **spec-first OpenAPI 3.1 document** that is the authoritative contract for every route in Aethel Workspace, served through a Scalar interactive explorer embedded in the Go binary with zero external file dependencies at runtime.

**Philosophy this task follows:**

| Principle | Application |
|---|---|
| **Spec-first** | The YAML spec is written by hand and is the source of truth — not generated from code comments. The implementation must conform to the spec, never the reverse. |
| **Single source of schemas** | Every schema is defined once in `components/schemas`. No inline `type: object` definitions inside `paths`. DRY is non-negotiable. |
| **Fail fast** | The spec is validated at startup. An invalid spec panics the process — it is a programmer error, not a recoverable condition. |
| **Zero runtime I/O** | The spec YAML and Scalar HTML are embedded into the binary with `go:embed`. The server reads no files after startup. |
| **Security by default** | All endpoints inherit the global `BearerAuth` security scheme. Public endpoints explicitly opt out with `security: []`. This makes unauthenticated endpoints visible and intentional. |
| **Clean code** | The docs package has one responsibility: serve the spec and the UI. It has no knowledge of domain logic, no imported service or repository types. |
| **Feature-gated** | The route is registered conditionally — disabled in production via `AETHEL_DISABLE_API_DOCS=true` env var. Enabled by default (self-hosted software; the operator controls network access). |

---

## Step 0 — Load Context (Mandatory)

Read every file listed. Then output a numbered list of every file you will create or modify. Do not write any code before producing that list.

```
# Route definitions — every route in this file must appear in the spec
docs/architecture/architecture-api-routes.md

# Existing handler patterns to understand response shapes
aethel-core/internal/api/handlers/auth.go           ← also contains multi-tenant orgId bugs to fix (see Step 1)
aethel-core/internal/api/server.go                  ← how routes are registered; where to mount /api/docs
aethel-core/internal/api/handlers/                  ← list all handler files; understand writeJSON/writeError
aethel-core/cmd/aethel/main.go                      ← startup sequence; where to call ValidateSpec()
aethel-core/go.mod                                  ← existing dependencies; you will add one new one

# Domain types — source of truth for schema property names
aethel-core/internal/domain/dispatch.go
aethel-core/internal/domain/user.go
aethel-core/internal/domain/workflow.go
aethel-core/internal/domain/governance.go

# Architecture reference
CLAUDE.md                                            ← single-tenant model; JWT claims (sub, role, iat, exp, jti only)
docs/architecture/architecture-security.md           ← CSRF cookie pattern, rate limiting, auth flow
```

---

## Step 1 — Prerequisite: Fix Lingering Multi-Tenant References in `auth.go`

Before adding any new code, fix the existing `auth.go` handler. It still contains `orgId` fields that pre-date the single-tenant decision. These are bugs — remove them now.

Open `aethel-core/internal/api/handlers/auth.go`.

**Changes required:**

1. Remove `OrgID string json:"orgId"` from `loginRequest` struct. The handler does not need org routing — the installation has one org, whose ID is the `app.OrgID` constant.

2. Remove the `uuid.Parse(req.OrgID)` block. Remove the `orgID` variable. Update the `h.svc.Login(...)` call to not pass `orgID`.

3. In `Logout`: remove `orgIDStr, _ := rbac.OrgIDFromCtx(r.Context())`, remove `orgID` variable, update `h.svc.Logout(...)` call to not pass `orgID`.

4. In `RequestPasswordReset`: remove `OrgID string json:"orgId"` from the request struct, remove `uuid.Parse(req.OrgID)` block, update `h.svc.RequestPasswordReset(...)` call to not pass `orgID`.

5. Change `ErrAccountLocked` response status from `http.StatusForbidden` (403) to `http.StatusLocked` (423). HTTP 423 "Locked" is the semantically correct status for a temporarily locked account. 403 means "you don't have permission" — 423 means "this resource is locked."

After changes, run:
```bash
cd aethel-core && go build ./... && go vet ./...
```
Both must pass before Step 2.

---

## Step 2 — Add Dependency

Add the OpenAPI 3.1 validation library:

```bash
cd aethel-core
go get github.com/getkin/kin-openapi/openapi3@latest
go mod tidy
```

This library is used only at startup for spec validation. It is not in the request path.

---

## Step 3 — Create the Package Structure

Create the directory `aethel-core/internal/api/docs/`. It will contain exactly three files:

```
aethel-core/internal/api/docs/
├── handler.go       ← Go: ValidateSpec(), Handler(), feature flag check
├── openapi.yaml     ← OpenAPI 3.1 specification (the main deliverable)
└── scalar.html      ← Scalar UI HTML (single file, loads from CDN)
```

No other files belong in this package. The package name is `docs`.

---

## Step 4 — Create `scalar.html`

Create `aethel-core/internal/api/docs/scalar.html`:

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Aethel Workspace — API Reference</title>
  <style>
    body { margin: 0; padding: 0; }
  </style>
</head>
<body>
  <script
    id="api-reference"
    data-url="/api/docs/openapi.yaml"
    data-proxy-url=""
  ></script>
  <script>
    // Pre-configure Scalar with Aethel-specific defaults.
    document.getElementById('api-reference').dataset.configuration = JSON.stringify({
      theme: 'default',
      layout: 'modern',
      defaultHttpClient: { targetKey: 'js', clientKey: 'fetch' },
      authentication: {
        preferredSecurityScheme: 'BearerAuth',
        http: { bearer: { token: '' } }
      },
      hideModels: false,
      hideDownloadButton: false,
      showSidebar: true,
    })
  </script>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>
```

---

## Step 5 — Create `handler.go`

Create `aethel-core/internal/api/docs/handler.go`:

```go
// Package docs serves the OpenAPI 3.1 specification and the Scalar interactive
// API explorer. It has no knowledge of domain types, services, or repositories.
// Its only dependency is the embedded spec and HTML files.
package docs

import (
	"context"
	_ "embed"
	"net/http"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
)

//go:embed openapi.yaml
var specBytes []byte

//go:embed scalar.html
var scalarHTML []byte

// ValidateSpec parses and validates the embedded OpenAPI spec at startup.
// Panics on any validation error — an invalid spec is a programmer error,
// not a recoverable runtime condition. Call this once from main.go before
// registering routes.
func ValidateSpec() {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(specBytes)
	if err != nil {
		panic("openapi: failed to parse spec: " + err.Error())
	}
	if err := doc.Validate(context.Background()); err != nil {
		panic("openapi: spec validation failed: " + err.Error())
	}
}

// Enabled reports whether the API docs route should be registered.
// Returns false when AETHEL_DISABLE_API_DOCS=true is set — useful for
// deployments that want to restrict access to the API explorer.
// Enabled by default; self-hosted operators control network-level access.
func Enabled() bool {
	return os.Getenv("AETHEL_DISABLE_API_DOCS") != "true"
}

// Handler returns an http.Handler that serves the Scalar UI and the raw
// OpenAPI YAML spec. Mount this at /api/docs in the chi router.
//
// Routes served (relative to mount point):
//   GET /              → Scalar UI (HTML)
//   GET /openapi.yaml  → Raw OpenAPI 3.1 spec (YAML)
func Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache") // always serve fresh during development
		_, _ = w.Write(scalarHTML)
	})

	mux.HandleFunc("GET /openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=300")
		_, _ = w.Write(specBytes)
	})

	return mux
}
```

---

## Step 6 — Register the Route in `server.go`

Open `aethel-core/internal/api/server.go`. Find where routes are registered.

Add the docs route **outside** of the authenticated route group — API docs are publicly accessible (the operator controls network access):

```go
import "aethel-core/internal/api/docs"

// In the server setup, after all authenticated routes are registered:
if docs.Enabled() {
    r.Mount("/api/docs", docs.Handler())
}
```

Add to `cmd/aethel/main.go`, in the startup sequence after blueprint loading and before the HTTP server starts:

```go
// Validate OpenAPI spec at startup. Panics if the spec is malformed.
// Comment this line out only if you are intentionally shipping a spec update
// that is not yet passing validation.
docs.ValidateSpec()
```

---

## Step 7 — Write `openapi.yaml`

This is the primary deliverable. Write `aethel-core/internal/api/docs/openapi.yaml` in full.

### Structural rules (enforce without exception)

- **Every schema** lives in `components/schemas`. No inline `type: object` with properties inside `paths`.
- **Every error response** references a `components/responses` entry. No inline error response definitions.
- **Every path parameter** (`{id}`, `{att_id}`) references `components/parameters`.
- **Every endpoint** has: `tags`, `operationId`, `summary`, `description`, `responses`.
- **`operationId`** format: `PascalCase` tag + verb + noun. Examples: `AuthLogin`, `DispatchCreate`, `AdminUserList`, `WorkflowGreenNoteAppend`.
- **Public endpoints** declare `security: []` explicitly. All others inherit the global `BearerAuth`.
- **Every response** includes a realistic `example` block.
- **Nullable fields** use `nullable: true` (OpenAPI 3.1 style), not `type: [string, 'null']`.

### Top-level structure

```yaml
openapi: "3.1.0"

info:
  title: Aethel Workspace API
  version: "1.0.0"
  description: |
    REST API for Aethel Workspace — an open-source, self-hosted e-office platform
    for institutional document workflows.

    ## Authentication

    All endpoints (except those marked **Public**) require a JWT access token:

    ```
    Authorization: Bearer <access_token>
    ```

    Obtain a token via `POST /api/v1/auth/login`. Tokens expire in **15 minutes**.
    Use `POST /api/v1/auth/refresh` to renew silently using the `refresh_token` httpOnly cookie.

    ## CSRF Protection

    State-changing requests (`POST`, `PATCH`, `PUT`, `DELETE`) from browser clients must
    include the CSRF token in a header:

    ```
    X-CSRF-Token: <value of the csrf_token cookie>
    ```

    The `csrf_token` cookie is set by the server on login. Non-browser clients (CLI, mobile)
    that do not send cookies are exempt from this requirement.

    ## Rate Limiting

    - **Global:** 600 requests/min per IP (unauthenticated)
    - **Per user:** 300 requests/min per authenticated user
    - **Login endpoint:** 20 requests/min per IP (credential-stuffing protection)

    Responses exceeding the limit receive `429 Too Many Requests` with a `Retry-After` header.

    ## Error Format

    All error responses use a consistent JSON envelope:
    ```json
    { "error": "human-readable description", "field": "fieldName (validation errors only)" }
    ```

  contact:
    name: Aethel Workspace
    url: https://github.com/hoangharry-tm/aethel-workspace

  license:
    name: Apache 2.0
    url: https://www.apache.org/licenses/LICENSE-2.0

servers:
  - url: http://localhost:8080/api/v1
    description: Local development
  - url: "{scheme}://{host}/api/v1"
    description: Self-hosted deployment
    variables:
      scheme:
        default: https
        enum: [https, http]
      host:
        default: your-domain.example.com
        description: Your server hostname

security:
  - BearerAuth: []

tags:
  - name: Auth
    description: |
      Authentication and session management. Login, token refresh, logout,
      and password reset flows.
  - name: Config
    description: |
      Runtime configuration — branding, navigation structure, feature flags, and org profile.
      Read endpoints are available to all authenticated users. Write endpoints require ADMIN.
  - name: Dispatch
    description: |
      **Pillar 1 — DAK Diarization.** Inbound and outbound correspondence tracking.
      Includes receipt, routing, assignment, delivery acknowledgement, and attachments.
  - name: Workflow
    description: |
      **Pillar 2 — Green Noting Canvas.** Minute sheets and cryptographically-chained
      green notes. Each note is immutable after insert; the chain is verified on every append.
  - name: Governance
    description: |
      **Pillar 3 — RBAC Audit Ledger.** Append-only, monthly-partitioned, checksum-chained
      event log. Requires `SYS_ADMIN` role. Includes tamper-detection verification endpoint.
  - name: Admin
    description: |
      Administrative management — users, document types, routing rules, escalation rules,
      reports, and system settings. Requires ADMIN role.
  - name: Notifications
    description: |
      In-app notifications and real-time SSE stream. The SSE endpoint (`/notifications/stream`)
      delivers events to connected browser clients within 2 seconds of the triggering action.
  - name: System
    description: Health and readiness probes. No authentication required.
```

### `components/securitySchemes`

```yaml
components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
      description: |
        JWT access token from `POST /api/v1/auth/login`.
        Claims: `sub` (user UUID), `role`, `iat`, `exp`, `jti`.
        Expiry: 15 minutes. Use the refresh endpoint to renew.
```

### `components/schemas` — define ALL of these

Define each schema completely. Below are the required schemas and their minimum required properties. Derive exact field names and types from the domain Go structs you read in Step 0. Use `camelCase` for all JSON property names (matching existing handler response shapes).

**Enumerations:**
- `UserRole` — `enum: [ADMIN, RECEPTION, USER, SYS_ADMIN]`
- `PriorityLevel` — `enum: [ROUTINE, PRIORITY, IMMEDIATE]`
- `DispatchStatus` — `enum: [PENDING_ASSIGNMENT, UNDER_REVIEW, IN_TRANSIT, ATTEMPTED_DELIVERY, DELIVERED, ESCALATED, DISPATCHED, REJECTED]`
- `DispatchDirection` — `enum: [INBOUND, OUTBOUND]`
- `MatchOperator` — `enum: [EQUALS, CONTAINS, STARTS_WITH, REGEX]`
- `ConditionType` — `enum: [DOCUMENT_TYPE, SENDER_NAME, SENDER_ORG, URGENCY_LEVEL]`
- `NotificationType` — `enum: [DOCUMENT_ARRIVAL, REMINDER, ESCALATION, SYSTEM]`

**Domain objects:**
- `User` — all fields from `domain.User`; omit `passwordHash`
- `UserSummary` — `id`, `fullName`, `email`, `role`, `departmentId` (lightweight list item)
- `Dispatch` — all fields from `domain.Dispatch`
- `DispatchSummary` — `id`, `trackingNumber`, `direction`, `senderName`, `priorityLevel`, `statusState`, `createdAt` (for list endpoints)
- `DispatchEvent` — all fields from `domain.DispatchEvent`
- `DocumentType` — `id`, `name`, `description`, `defaultUrgencyLevel`, `isActive`, `sortOrder`, `createdAt`
- `MinuteSheet` — `id`, `dispatchId`, `createdAt`, `updatedAt`, plus an embedded `notes` array of `GreenNote`
- `GreenNote` — all fields from `domain.GreenNote`; include `cryptographicHash` and `previousHash`
- `AuditEntry` — all fields from `domain.AuditEntry`
- `RoutingRule` — all fields including `conditions` array and `destinations` array
- `RoutingRuleCondition` — `id`, `conditionType`, `conditionValue`, `matchOperator`
- `RoutingRuleDestination` — `id`, `stopOrder`, `targetUserId`, `targetDepartmentId`, `confirmationRequired`
- `EscalationRule` — all fields
- `Notification` — all fields from `domain.Notification`
- `AppConfig` — branding, nav, features, org sub-objects (must match `AppRuntimeConfig` TypeScript interface exactly)
- `BrandingConfig` — `primaryColor`, `neutralPalette`, `fontFamily`, `logoUrl`, `wordmark`
- `NavGroup` — `label`, `roles`, `items` (array of `NavItem`)
- `NavItem` — `label`, `icon`, `to`, `badge`
- `FeatureFlags` — `greenNotingEnabled`, `externalSmtpEnabled`, `require2faForAdmin`
- `OrgProfile` — `name`, `timezone`, `locale`, `contactEmail`
- `ChainVerificationResult` — `valid` (bool), `checkedRows` (int), `brokenAtId` (nullable string), `brokenAtTimestamp` (nullable date-time)

**Request bodies:**
- `LoginRequest` — `email`, `password`
- `CreateDispatchRequest` — required fields to create a dispatch
- `AssignDispatchRequest` — `userId` or `departmentId`, `isManualOverride`
- `AppendGreenNoteRequest` — `content`, `previousHash`
- `CreateUserRequest` — `email`, `fullName`, `role`, `departmentId`, `password`
- `PatchBrandingRequest` — all optional: `primaryColor`, `neutralPalette`, `fontFamily`, `wordmark`
- `PatchNavRequest` — `nav` (array of `NavGroup`)
- `PatchFeaturesRequest` — all optional boolean flags
- `PatchOrgRequest` — all optional: `name`, `timezone`, `locale`, `contactEmail`

**Pagination wrapper:**
- `PaginatedResponse` — generic wrapper used for all list endpoints:
  ```yaml
  PaginatedResponse:
    type: object
    required: [data, total, page, pageSize]
    properties:
      data:
        type: array
        items: {}   # overridden per-endpoint with allOf
      total: { type: integer }
      page: { type: integer }
      pageSize: { type: integer }
  ```

### `components/responses` — define ALL of these

```yaml
  responses:
    BadRequest:        # 400
    Unauthorized:      # 401
    Forbidden:         # 403 — wrong role
    Locked:            # 423 — account locked
    NotFound:          # 404
    UnprocessableEntity: # 422 — validation error with field
    TooManyRequests:   # 429 — includes Retry-After header
    InternalError:     # 500
```

Every error response includes a `content.application/json.schema` referencing `Error` and a realistic `example`.

### `components/parameters`

```yaml
  parameters:
    ResourceID:
      name: id
      in: path
      required: true
      schema: { type: string, format: uuid }
    AttachmentID:
      name: att_id
      in: path
      required: true
      schema: { type: string, format: uuid }
    PageParam:
      name: page
      in: query
      schema: { type: integer, default: 1, minimum: 1 }
    PageSizeParam:
      name: pageSize
      in: query
      schema: { type: integer, default: 20, minimum: 1, maximum: 100 }
    DateFrom:
      name: from
      in: query
      schema: { type: string, format: date }
      description: Start of date range (inclusive), ISO 8601 date
    DateTo:
      name: to
      in: query
      schema: { type: string, format: date }
      description: End of date range (exclusive), ISO 8601 date
```

### `paths` — every route from `architecture-api-routes.md`

Write every path. The following shows the **complete pattern** for two endpoints. Use this exact pattern for all remaining endpoints — do not deviate from the structure.

**Complete example: POST /auth/login**
```yaml
paths:
  /auth/login:
    post:
      tags: [Auth]
      operationId: AuthLogin
      summary: Log in and receive tokens
      description: |
        Authenticates the user with email and password.
        On success, sets two cookies and returns the access token in the response body:
        - `refresh_token` — httpOnly; JavaScript cannot read this
        - `csrf_token` — readable; JavaScript must send this in `X-CSRF-Token` on subsequent requests

        **Account lockout:** After 5 consecutive failures, the account is locked for 15 minutes.
        The same `invalid credentials` error is returned regardless of whether the email exists
        (prevents user enumeration).
      security: []
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: '#/components/schemas/LoginRequest' }
            example:
              email: "marcus.webb@ministry.gov"
              password: "S3cur3P@ssw0rd!"
      responses:
        '200':
          description: Login successful
          headers:
            Set-Cookie:
              description: Sets `refresh_token` (httpOnly) and `csrf_token` (readable) cookies
              schema: { type: string }
          content:
            application/json:
              schema:
                type: object
                required: [access_token, token_type, expires_in, role]
                properties:
                  access_token: { type: string }
                  token_type: { type: string, enum: [Bearer] }
                  expires_in: { type: integer, description: Seconds until expiry }
                  role: { $ref: '#/components/schemas/UserRole' }
              example:
                access_token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiI4ZjNhMTJiYy0..."
                token_type: Bearer
                expires_in: 900
                role: RECEPTION
        '400': { $ref: '#/components/responses/BadRequest' }
        '401':
          description: Invalid email or password (same response whether email exists or not)
          content:
            application/json:
              schema: { $ref: '#/components/schemas/Error' }
              example: { error: "invalid credentials" }
        '423': { $ref: '#/components/responses/Locked' }
        '429': { $ref: '#/components/responses/TooManyRequests' }
```

**Complete example: POST /dispatches/{id}/green-notes**
```yaml
  /dispatches/{id}/green-notes:
    post:
      tags: [Workflow]
      operationId: WorkflowGreenNoteAppend
      summary: Append a green note to a minute sheet
      description: |
        Appends a new green note to the dispatch's minute sheet.

        **Hash chain enforcement:** The `previousHash` field must equal the
        `cryptographicHash` of the most recent note in the sheet. If it does not match,
        the server rejects the request with `409 Conflict` — this indicates either a
        concurrent write race or an attempt to tamper with the chain.

        Notes are **immutable after insert**. There is no update or delete endpoint.
      parameters:
        - $ref: '#/components/parameters/ResourceID'
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: '#/components/schemas/AppendGreenNoteRequest' }
            example:
              content: "Document reviewed. No issues found. Forwarding to department head for approval."
              previousHash: "a3f8c2d1..."
      responses:
        '201':
          description: Green note appended successfully
          content:
            application/json:
              schema: { $ref: '#/components/schemas/GreenNote' }
        '400': { $ref: '#/components/responses/BadRequest' }
        '401': { $ref: '#/components/responses/Unauthorized' }
        '403': { $ref: '#/components/responses/Forbidden' }
        '404': { $ref: '#/components/responses/NotFound' }
        '409':
          description: Hash chain mismatch — previousHash does not match the last note
          content:
            application/json:
              schema: { $ref: '#/components/schemas/Error' }
              example: { error: "hash chain broken: previousHash does not match last note" }
        '500': { $ref: '#/components/responses/InternalError' }
```

**Apply this same pattern to all remaining routes:**

Organize paths in this order (matches the tag order in the spec):

1. `/auth/login`, `/auth/refresh`, `/auth/logout`, `/auth/password-reset/request`, `/auth/password-reset/confirm`
2. `/config`, `/config/branding`, `/config/nav`, `/config/features`
3. `/admin/config/branding`, `/admin/config/nav`, `/admin/config/features`, `/admin/config/org`
4. `/dispatches` (GET + POST), `/dispatches/{id}`, `/dispatches/{id}/status`, `/dispatches/{id}/assign`, `/dispatches/{id}/acknowledge`
5. `/dispatches/outbound` (GET + POST)
6. `/dispatches/{id}/attachments` (GET + POST), `/dispatches/{id}/attachments/{att_id}` (DELETE)
7. `/my-dispatches`, `/search`
8. `/dispatches/{id}/minute-sheet`, `/dispatches/{id}/green-notes` (GET + POST), `/dispatches/{id}/minute-sheet/approve`
9. `/audit-log`, `/audit-log/verify`
10. All `/admin/users/*`, `/admin/document-types/*`, `/admin/routing-rules/*`, `/admin/escalation-rules/*`
11. `/admin/reports`, `/admin/settings`, `/admin/branding`
12. `/notifications`, `/notifications/{id}/read`, `/notifications/stream`
13. `/healthz`, `/readyz` (system endpoints, tag: System, security: [])

**SSE endpoint note:** `GET /notifications/stream` has a special response:
```yaml
    get:
      tags: [Notifications]
      operationId: NotificationsStream
      summary: Real-time notification stream (SSE)
      description: |
        Server-Sent Events stream. Keep-alive connection; the server pushes events as
        they occur. Reconnects automatically in the browser.
        
        **Event format:**
        ```
        event: notification
        data: {"id":"...","type":"DOCUMENT_ARRIVAL","title":"...","dispatchId":"..."}
        ```
        
        The connection is closed by the server if the user's session expires.
      responses:
        '200':
          description: SSE stream opened
          content:
            text/event-stream:
              schema:
                type: string
                description: Newline-delimited SSE events
```

---

## Step 8 — Update Admin Navigation

Open `aethel-view/app/composables/useRuntimeConfig.ts`. In the default nav config, add an API docs link under the Administration group:

```typescript
{
  label: "API Reference",
  icon: "i-lucide-book-open",
  to: "http://localhost:8080/api/docs",  // external link to Go backend docs
  badge: null,
}
```

This entry only appears in the navigation for ADMIN users (it's already in the Administration group). It opens the Scalar UI served by the Go backend.

---

## Constraints — Do NOT Do These

- **Do not** define any schema inline inside `paths` — every schema goes in `components/schemas`
- **Do not** repeat error response definitions — every error references `components/responses`
- **Do not** generate the spec from Go code comments (swaggo/swag) — the spec is written by hand
- **Do not** add `orgId` back to any request body — the single-tenant fix from Step 1 must hold
- **Do not** import domain, service, or repository packages into the `docs` package — it has no business logic dependencies
- **Do not** read the YAML file from disk at runtime — `go:embed` only
- **Do not** skip the `ValidateSpec()` call in `main.go` — startup validation is required
- **Do not** commit — the user will review and commit manually

---

## Step 9 — Verification

Run every check. All must pass.

```bash
cd aethel-core

# 1. Build — includes go:embed compilation; fails if files are missing
go build ./...

# 2. Vet
go vet ./...

# 3. Multi-tenant remnants are gone from auth handler — must return 0 matches
grep -n "orgId\|OrgID\|orgIDFromCtx\|OrgIDFromCtx" internal/api/handlers/auth.go
# Expected: 0 matches

# 4. Account locked returns 423 not 403 — must return 1 match
grep -n "StatusLocked\|423" internal/api/handlers/auth.go
# Expected: 1 match

# 5. All schemas are in components — no inline objects in paths
grep -n "type: object" internal/api/docs/openapi.yaml \
  | grep -v "components/\|#.*type: object"
# Expected: 0 matches (all objects defined under components)

# 6. Start the server and verify the docs route
go run ./cmd/aethel serve &
SERVER_PID=$!
sleep 2

# Scalar UI is reachable
curl -sf http://localhost:8080/api/docs | grep -q "scalar" && echo "UI: OK"

# Raw spec is valid YAML and contains all expected tags
curl -sf http://localhost:8080/api/docs/openapi.yaml | grep -q "openapi: \"3.1.0\"" && echo "SPEC VERSION: OK"
curl -sf http://localhost:8080/api/docs/openapi.yaml | grep -c "operationId:" | xargs echo "operationId count:"
# Expected: ≥ 40 (one per route)

# Feature flag disables the route
AETHEL_DISABLE_API_DOCS=true go run ./cmd/aethel serve &
FLAGGED_PID=$!
sleep 2
STATUS=$(curl -sw "%{http_code}" -o /dev/null http://localhost:8080/api/docs)
echo "With flag disabled, status: $STATUS"  # Expected: 404
kill $SERVER_PID $FLAGGED_PID 2>/dev/null

# 7. ValidateSpec passes (would have panicked at startup if not)
echo "ValidateSpec: passed (server started successfully)"
```

**Report back with:**

1. Output of `grep -n "orgId\|OrgID" internal/api/handlers/auth.go` — must be empty
2. The `operationId count` line — must be ≥ 40
3. Output of `go build ./...` and `go vet ./...` — both clean
4. A screenshot or copy of `curl -sf http://localhost:8080/api/docs/openapi.yaml | head -30` showing the spec header
5. Confirmation that `AETHEL_DISABLE_API_DOCS=true` returns 404
6. A list of every new or modified file with a one-line description
7. Any endpoint from `architecture-api-routes.md` that is missing from the spec (must be none) — explicitly state "all N routes accounted for"

---

## Reference: Definition of Done

- `GET /api/docs` returns the Scalar UI HTML with the API explorer fully functional
- `GET /api/docs/openapi.yaml` returns a valid OpenAPI 3.1 YAML document
- The spec contains one `operationId` entry for every route in `docs/architecture/architecture-api-routes.md` — no route is missing or undocumented
- Every schema is in `components/schemas` — zero inline object definitions in `paths`
- Every error response references `components/responses` — zero inline error definitions in `paths`
- The server panics at startup if `openapi.yaml` is syntactically invalid
- `AETHEL_DISABLE_API_DOCS=true` makes the route return `404`
- `go build ./...` and `go vet ./...` are clean
- The multi-tenant `orgId` field is removed from all auth handler request structs
- Account locked returns HTTP 423, not 403
