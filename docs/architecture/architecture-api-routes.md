# Architecture — API Route Design

**Audience:** Go engineers, IT administrators
**Status:** Active

Diagram: [docs/diagrams/api-route-resolution.mmd](../diagrams/api-route-resolution.mmd)

---

## Design Philosophy

API routes in Aethel Core are convention-driven by the domain model. Each route has hardcoded defaults baked into the Go binary: a pattern, an HTTP method, a handler reference, and a required permission. These defaults are production-ready and require no blueprint to function.

Routes are not IT-configurable. The `blueprints/server-routes.yaml` file no longer exists — route renaming was removed in the architectural pivot to runtime-configurable with compile-time defaults (2026-05-27). If a deployment requires custom path prefixes, configure a reverse proxy (Nginx, Caddy) to rewrite paths before they reach the Go server.

This design has a specific consequence: every route must have a `permission` field. The router registration rejects any route definition that is missing a permission. A route with no permission check cannot exist in this system.

---

## Runtime Configuration API

The config API serves organization branding, navigation structure, and feature flags to the Nuxt SSR frontend. All endpoints under `/api/v1/config` are served from a single in-memory config cache (5-min TTL). Admin PATCH endpoints invalidate the cache so the next request re-fetches from the database.

### Read endpoints (all authenticated, `dispatch.view` minimum)

| Method | Pattern | Permission | Description |
|---|---|---|---|
| GET | `/api/v1/config` | `dispatch.view` | Full org config (branding + nav + features + org profile); cached |
| GET | `/api/v1/config/branding` | `dispatch.view` | Branding fields only |
| GET | `/api/v1/config/nav` | `dispatch.view` | Nav seed + any runtime overrides stored in `system_settings` |
| GET | `/api/v1/config/features` | `dispatch.view` | Feature flags |

### Write endpoints (admin only)

| Method | Pattern | Permission | Description |
|---|---|---|---|
| PATCH | `/api/v1/admin/config/branding` | `admin.access` | Update branding; invalidates config cache |
| PATCH | `/api/v1/admin/config/nav` | `admin.access` | Update nav overrides; invalidates config cache |
| PATCH | `/api/v1/admin/config/features` | `admin.access` | Update feature flags; invalidates config cache |
| PATCH | `/api/v1/admin/config/org` | `admin.access` | Update org profile |

### Cache invalidation pattern

On a successful PATCH, the Go handler calls `cache.Invalidate(orgID)` which removes the org's entry from the in-memory map. The next `GET /api/v1/config` request for that org triggers a fresh DB read and re-populates the cache. The frontend calls `refresh()` from `useRuntimeConfig()` immediately after a successful PATCH to update local state without a full page reload.

---

## Path Parameter Type Constraints

Chi supports inline regex constraints on path parameters. The router validates the parameter before the handler runs; a request with an invalid parameter value receives `404 Not Found` without reaching the handler.

| Constraint syntax | What it validates | Example usage |
|---|---|---|
| `{id:uuid}` | Full UUID format (8-4-4-4-12 hex groups) | `/dispatches/{id:uuid}` |
| `{year:\d{4}}` | Exactly 4 digits | `/reports/{year:\d{4}}` |
| `{month:\d{2}}` | Exactly 2 digits | `/reports/{year:\d{4}}/{month:\d{2}}` |
| `{slug:[a-z0-9-]+}` | Lowercase alphanumeric with hyphens | `/doc-types/{slug:[a-z0-9-]+}` |

The `uuid` constraint is a named pattern registered in `api/server.go` at startup using `chi.RegisterPatternKey("uuid", "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}")`. This keeps route definitions readable.

---

## Route Registry at Startup

The route registry is built in `api/server.go` during server initialization. The process:

1. The hardcoded default routes are defined as a slice of `RouteDefinition` structs in the Go binary.
2. Each route is validated: `permission` must be non-empty; pattern syntax must parse; handler reference must resolve to a registered function.
3. The chi router is populated: groups become sub-routers with scoped middleware; routes are registered with their patterns.
4. The RBAC middleware for each route is wired using the route's `permission` field, resolved at startup. A misconfigured permission string causes the process to exit at step 2, not at request time.

Adding a new route requires Go code — write the handler, add the route definition to `api/server.go`, and deploy a new binary. Routes are not runtime-configurable.

---

## All Default API Routes

### Base path: `/api/v1`

#### Reception — Pillar 1 (DAK Diarization)

| Method | Pattern | Handler | Permission | Description |
|---|---|---|---|---|
| GET | `/dispatches` | `dispatch.ListInbox` | `dispatch.view` | Active inbound queue for the current user's department |
| POST | `/dispatches` | `dispatch.Create` | `dispatch.create` | Log a new inbound dispatch |
| GET | `/dispatches/{id:uuid}` | `dispatch.GetByID` | `dispatch.view` | Full dispatch detail with timeline |
| PATCH | `/dispatches/{id:uuid}/status` | `dispatch.UpdateStatus` | `dispatch.create` | Update status state |
| POST | `/dispatches/{id:uuid}/assign` | `dispatch.Assign` | `dispatch.assign` | Manually assign to a user or department |
| POST | `/dispatches/{id:uuid}/acknowledge` | `dispatch.Acknowledge` | `dispatch.deliver` | Record receipt acknowledgement |
| GET | `/dispatches/outbound` | `dispatch.ListOutbound` | `dispatch.view` | Outbound queue |
| POST | `/dispatches/outbound` | `dispatch.CreateOutbound` | `dispatch.create` | Log a new outbound dispatch |
| GET | `/dispatches/{id:uuid}/attachments` | `dispatch.ListAttachments` | `dispatch.view` | List file attachments for a dispatch |
| POST | `/dispatches/{id:uuid}/attachments` | `dispatch.UploadAttachment` | `dispatch.create` | Upload a file attachment |
| DELETE | `/dispatches/{id:uuid}/attachments/{att_id:uuid}` | `dispatch.DeleteAttachment` | `dispatch.assign` | Remove an attachment |

#### User — My Documents

| Method | Pattern | Handler | Permission | Description |
|---|---|---|---|---|
| GET | `/my-dispatches` | `dispatch.ListMyDispatches` | `workflow.view` | Dispatches assigned to the current user |
| GET | `/search` | `search.Search` | `dispatch.view` | Full-text search across dispatches |

#### Workflow — Pillar 2 (Green Noting Canvas)

| Method | Pattern | Handler | Permission | Description |
|---|---|---|---|---|
| GET | `/dispatches/{id:uuid}/minute-sheet` | `workflow.GetMinuteSheet` | `workflow.view` | Fetch the minute sheet and green note timeline |
| POST | `/dispatches/{id:uuid}/green-notes` | `workflow.AppendGreenNote` | `workflow.approve` | Append a new green note (hash chain enforced) |
| GET | `/dispatches/{id:uuid}/green-notes` | `workflow.ListGreenNotes` | `workflow.view` | List all green notes in sequence order |
| POST | `/dispatches/{id:uuid}/minute-sheet/approve` | `workflow.ApproveMinuteSheet` | `workflow.approve` | Mark the minute sheet as approved |

#### Governance — Pillar 3 (RBAC Audit Ledger)

| Method | Pattern | Handler | Permission | Description |
|---|---|---|---|---|
| GET | `/audit-log` | `governance.QueryAuditLog` | `admin.audit` | Query the audit ledger with date range and filters |
| GET | `/audit-log/verify` | `governance.VerifyChain` | `admin.audit` | Verify the checksum chain for a given date range |

#### Admin — Cross-Pillar

| Method | Pattern | Handler | Permission | Description |
|---|---|---|---|---|
| GET | `/admin/users` | `admin.ListUsers` | `admin.access` | List all users in the organization |
| POST | `/admin/users` | `admin.CreateUser` | `admin.access` | Create a new user account |
| GET | `/admin/users/{id:uuid}` | `admin.GetUser` | `admin.access` | Get user detail |
| PATCH | `/admin/users/{id:uuid}` | `admin.UpdateUser` | `admin.access` | Update user profile or role |
| DELETE | `/admin/users/{id:uuid}` | `admin.DeactivateUser` | `admin.access` | Deactivate a user account |
| GET | `/admin/document-types` | `admin.ListDocumentTypes` | `admin.access` | List document type catalogue |
| POST | `/admin/document-types` | `admin.CreateDocumentType` | `admin.access` | Create a document type |
| PATCH | `/admin/document-types/{id:uuid}` | `admin.UpdateDocumentType` | `admin.access` | Update a document type |
| DELETE | `/admin/document-types/{id:uuid}` | `admin.DeleteDocumentType` | `admin.access` | Delete a document type |
| GET | `/admin/routing-rules` | `admin.ListRoutingRules` | `admin.access` | List routing rules in priority order |
| POST | `/admin/routing-rules` | `admin.CreateRoutingRule` | `admin.access` | Create a routing rule |
| PUT | `/admin/routing-rules/{id:uuid}` | `admin.UpdateRoutingRule` | `admin.access` | Update a routing rule |
| DELETE | `/admin/routing-rules/{id:uuid}` | `admin.DeleteRoutingRule` | `admin.access` | Delete a routing rule |
| GET | `/admin/escalation-rules` | `admin.ListEscalationRules` | `admin.access` | List escalation rules |
| POST | `/admin/escalation-rules` | `admin.CreateEscalationRule` | `admin.access` | Create an escalation rule |
| PUT | `/admin/escalation-rules/{id:uuid}` | `admin.UpdateEscalationRule` | `admin.access` | Update an escalation rule |
| GET | `/admin/reports` | `admin.GetReports` | `admin.access` | Aggregated dispatch statistics |
| GET | `/admin/settings` | `admin.GetSettings` | `admin.access` | Read system settings key-value store |
| PATCH | `/admin/settings` | `admin.UpdateSettings` | `admin.access` | Update system settings |
| GET | `/admin/branding` | `admin.GetBranding` | `admin.access` | Read branding configuration |
| PUT | `/admin/branding` | `admin.UpdateBranding` | `admin.access` | Update branding configuration |

#### Auth

| Method | Pattern | Handler | Permission | Description |
|---|---|---|---|---|
| POST | `/auth/login` | `auth.Login` | `public` | Authenticate, receive access + refresh tokens |
| POST | `/auth/refresh` | `auth.Refresh` | `public` | Exchange refresh token for new access token |
| POST | `/auth/logout` | `auth.Logout` | `dispatch.view` | Revoke the current session |
| POST | `/auth/password-reset/request` | `auth.RequestPasswordReset` | `public` | Send password reset email |
| POST | `/auth/password-reset/confirm` | `auth.ConfirmPasswordReset` | `public` | Apply password reset token |

#### Notifications

| Method | Pattern | Handler | Permission | Description |
|---|---|---|---|---|
| GET | `/notifications` | `notifications.List` | `dispatch.view` | List recent notifications for the current user |
| PATCH | `/notifications/{id:uuid}/read` | `notifications.MarkRead` | `dispatch.view` | Mark a notification as read |
| GET | `/notifications/stream` | `notifications.Stream` | `dispatch.view` | SSE stream for real-time delivery |

---

## Security Constraint

The blueprint loader enforces at startup that every registered route has a non-empty `permission` field. The only exceptions are routes explicitly marked `permission: "public"` (login, refresh, password reset). A `public` route still passes through the Auth middleware, but the middleware skips the JWT requirement and allows unauthenticated access.

This constraint exists because a route that silently skips permission checking is a security defect. Making the absence of a permission a hard startup error means it cannot be introduced by accident.
