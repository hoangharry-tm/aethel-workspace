# Task 16 — Pre-Sprint-5 Critical Fixes

**Purpose:** Fix all critical (🔴) and important (🟡) findings from the Task 15 audit before continuing with the agile plan.
**Generated:** 2026-06-05
**Source:** Task 15 audit — overall score 84%

---

## Why This Task Exists Before Sprint 5

The Task 15 audit uncovered a production-blocking Dockerfile bug that makes every CD image build fail, four critical security holes (no CSP header, stored XSS via unguarded `v-html`, hardcoded fallback JWT secret, no request body size limit), and 14 important backend/test/ops gaps that will compound further in Sprint 5 if left unaddressed — including two hot-path panics on every login, an expired-session regression where `session_repo` inline SQL skips the `expires_at` filter, and a CI pipeline that cannot run integration tests because it has no PostgreSQL service container.

---

## Fix Blocks (execute in this order)

---

### Block 1 — Security Fixes (Critical 🔴)

#### 1.1 — Dockerfile outside-build-context (CD breakage)
- **Finding:** `aethel-core/Dockerfile:29` — `COPY ../blueprints ./blueprints` references a path outside the `./aethel-core` build context. Every `docker build -f aethel-core/Dockerfile aethel-core/` invocation in the CD pipeline fails. Dev compose works only because blueprints are volume-mounted.
- **Required fix:** Move the `docker build` invocation to the repo root (pass `-f aethel-core/Dockerfile .`) so the context includes both `aethel-core/` and `blueprints/`. Update `Makefile` target `build-be` and `.github/workflows/cd.yml` accordingly. The `COPY ../blueprints ./blueprints` line then becomes `COPY blueprints ./blueprints` (from root context).
- **Acceptance criteria:** `docker build -f aethel-core/Dockerfile .` (run from repo root) exits 0 with no "path not found" errors.
- **Estimated effort:** S

#### 1.2 — Missing Content-Security-Policy header
- **Finding:** `internal/api/middleware/security_headers.go` — no `Content-Security-Policy` header is set anywhere in the middleware. Combined with finding 1.3 (v-html XSS), this leaves the admin pages with zero browser-level mitigation against script injection.
- **Required fix:** Add a CSP header to the `SecurityHeaders` middleware function. Minimum viable policy: `default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'`. The `'unsafe-inline'` style allowance is needed for Nuxt SSR inline styles; tighten once SSR nonces are wired.
- **Acceptance criteria:** `grep -n "Content-Security-Policy" aethel-core/internal/api/middleware/security_headers.go` returns at least one match.
- **Estimated effort:** S

#### 1.3 — Stored XSS via v-html without DOMPurify
- **Finding:** `aethel-view/app/components/blocks/BlockRichText.vue:54` — `v-html="content"` renders database content as raw HTML with no sanitization. Any user who can write to a rich-text field can inject persistent JavaScript that executes for every admin who views the page.
- **Required fix:** Install DOMPurify (`pnpm add dompurify` + `pnpm add -D @types/dompurify` in `aethel-view/`). Replace `v-html="content"` with `v-html="sanitizedContent"` where `sanitizedContent` is a computed property: `computed(() => DOMPurify.sanitize(props.content ?? ''))`. Import DOMPurify at the top of the script block.
- **Acceptance criteria:** `grep -n "v-html" aethel-view/app/components/blocks/BlockRichText.vue` returns zero raw `v-html="content"` references (only `v-html="sanitizedContent"` or equivalent computed name); `grep -n "DOMPurify" aethel-view/app/components/blocks/BlockRichText.vue` returns at least one match.
- **Estimated effort:** S

#### 1.4 — Hardcoded fallback JWT secret
- **Finding:** `internal/service/auth_service.go:233-235` and `internal/api/server.go:241-243` — fallback string `"dev-secret-change-in-production"` is used when the `AETHEL_JWT_SECRET` env var is missing. A misconfigured production deployment uses a publicly known secret, enabling universal token forgery.
- **Required fix:** In both locations, replace the fallback with a hard `log.Fatal` (or `os.Exit(1)` with a clear message): `if secret == "" { log.Fatal("AETHEL_JWT_SECRET environment variable is required — set a 64-byte base64-encoded random string") }`. Remove the fallback string entirely from both files.
- **Acceptance criteria:** `grep -rn "dev-secret" aethel-core/` returns 0 matches.
- **Estimated effort:** S

#### 1.5 — No http.MaxBytesReader — JSON body DoS
- **Finding:** `internal/api/server.go` (all handlers) — no `http.MaxBytesReader` wrapping on any request body. A client can stream an arbitrarily large JSON body to any endpoint, exhausting server memory.
- **Required fix:** Add a middleware (or augment the existing `Recoverer` chain) that wraps `r.Body` with `http.MaxBytesReader(w, r.Body, 1<<20)` (1 MiB limit). The cleanest approach is a one-liner middleware registered after `Recoverer` and before `RequestID` in `server.go`. Alternatively add it to the named middleware group in `server.go:91-111` as `maxBodyMiddleware`.
- **Acceptance criteria:** `grep -rn "MaxBytesReader" aethel-core/internal/api/` returns at least one match; `go build ./...` passes.
- **Estimated effort:** S

---

### Block 2 — Backend Completion (Important 🟡)

#### 2.1 — Hot-path panic in generateCSRFToken
- **Finding:** `internal/api/handlers/auth.go:203` — `generateCSRFToken()` calls `crypto/rand.Read` and panics on error. This function is called on every login — a transient OS entropy exhaustion event would crash the entire server process.
- **Required fix:** Change the function signature from `func generateCSRFToken() string` to `func generateCSRFToken() (string, error)`. Return the `crypto/rand.Read` error to the caller. In the caller, handle the error and return HTTP 500 rather than panicking.
- **Acceptance criteria:** `grep -n "panic" aethel-core/internal/api/handlers/auth.go` returns 0 matches; `go vet ./...` passes.
- **Estimated effort:** S

#### 2.2 — Hot-path panic in query_registry.Get()
- **Finding:** `internal/database/query_registry.go:59` — `Get()` panics when a key is missing. This can happen mid-request for any route whose named query was accidentally omitted from `queries.yaml`. A startup-time validation would surface the problem at boot instead.
- **Required fix:** In `BuildQueryRegistry` (query_registry.go), after parsing `queries.yaml`, iterate over a hardcoded list of required query keys and return an error for any missing key — making `BuildQueryRegistry` return `(*QueryRegistry, error)`. In `Get()`, replace the panic with `return "", fmt.Errorf("query key %q not found in registry")` (caller must handle). Update `main.go` / `server.go` to propagate the startup error.
- **Acceptance criteria:** `grep -n "panic" aethel-core/internal/database/query_registry.go` returns 0 matches; `go build ./...` passes.
- **Estimated effort:** M

#### 2.3 — dispatch_service.Create() bypasses repo abstraction
- **Finding:** `internal/service/dispatch_service.go:86-116` — `Create()` issues a raw `tx.ExecContext` with a literal INSERT SQL string instead of calling `s.dispatches.Create()`. This duplicates the `dispatch.create` entry in `queries.yaml`, breaks repo interface testability, and creates two sources of truth for the same query.
- **Required fix:** Remove the inline `tx.ExecContext` block. Wrap the existing `s.dispatches.Create()` call to pass the transaction context. If `DispatchRepo.Create` does not accept a `*sql.Tx`, add a `CreateTx(ctx, tx, dispatch)` variant to the `DispatchRepo` interface and implement it in `dispatch_repo.go` using `qr.Get("dispatch.create")`.
- **Acceptance criteria:** `grep -n "ExecContext\|INSERT INTO dispatches" aethel-core/internal/service/dispatch_service.go` returns 0 matches; `go build ./...` passes.
- **Estimated effort:** M

#### 2.4 — escalation_service.ruleApplies() is a logic stub
- **Finding:** `internal/service/escalation_service.go:71` — `ruleApplies()` ignores the `dispatch` argument entirely and always returns `rule.IsActive`. All active escalation rules fire on all overdue dispatches regardless of document type, urgency level, or defined conditions.
- **Required fix:** Implement the evaluation logic: iterate `rule.Conditions` (from `EscalationRule.Conditions` or fetched from `escalation_rule_repo`); for each condition, compare against the dispatch's `document_type_id`, `priority_level`, and `assigned_department_id`. Return `true` only when all conditions match (AND semantics) or at least one matches (OR semantics — check the domain spec in `domain/governance.go`).
- **Acceptance criteria:** `grep -n "ruleApplies\|IsActive" aethel-core/internal/service/escalation_service.go` — the `ruleApplies` function body contains conditional logic referencing dispatch fields; `go test ./internal/service/...` passes.
- **Estimated effort:** M

#### 2.5 — Inline SQL in auth-critical repos (behavioral gap)
- **Finding:** `internal/database/repos/user_repo.go`, `session_repo.go`, `password_reset_repo.go` — all use inline SQL strings instead of `QueryRegistry`. The inline `GetByEmail` in `user_repo.go` queries `WHERE email_address = $1` without scoping by `organization_id` or `is_active = true`, while the YAML version (`auth.get_user_by_email`) includes both filters — a behavioral gap in the most critical auth path.
- **Required fix:** Migrate all three repos to use `qr.Get(...)` calls from `QueryRegistry`. The YAML entries already exist for auth, session, and pw_reset query groups. For `user_repo.GetByEmail`, the inline SQL must match the YAML query exactly: `WHERE organization_id = $1 AND email_address = $2 AND is_active = true`.
- **Acceptance criteria:** `grep -rn '"SELECT\|"INSERT\|"UPDATE\|"DELETE' aethel-core/internal/database/repos/user_repo.go aethel-core/internal/database/repos/session_repo.go aethel-core/internal/database/repos/password_reset_repo.go` returns 0 matches; `go build ./...` passes.
- **Estimated effort:** L

#### 2.6 — CreateUser allows SYS_ADMIN privilege escalation
- **Finding:** `internal/api/handlers/admin.go:53-60` — `CreateUser` accepts `role` directly from the JSON body with no validation. Any `ADMIN` user can POST `{"role":"SYS_ADMIN"}` and create a superuser account.
- **Required fix:** Add a role allowlist check before inserting: only allow `ADMIN`, `RECEPTION`, `USER` values through the `CreateUser` endpoint. `SYS_ADMIN` creation must require a separate privileged path (or be bootstrap-only). Return HTTP 400 with a structured error for invalid role values. Also validate `email` format (regex or `mail.ParseAddress`).
- **Acceptance criteria:** `grep -n "allowedRoles\|SYS_ADMIN\|role.*valid\|ParseAddress" aethel-core/internal/api/handlers/admin.go` returns at least one match; `go build ./...` passes.
- **Estimated effort:** S

#### 2.7 — ConfirmPasswordReset has no minimum password length
- **Finding:** `internal/service/auth_service.go:208-226` — `ConfirmPasswordReset` does not validate minimum password length. The bootstrap path enforces a 12-character minimum; the password reset path accepts a single character.
- **Required fix:** Add a length check at the top of `ConfirmPasswordReset`: `if len(req.NewPassword) < 12 { return domain.ErrPasswordTooShort }`. Define `ErrPasswordTooShort` in `internal/domain/errors.go` and map it to HTTP 400 in the handler.
- **Acceptance criteria:** `grep -n "len(req.NewPassword)\|PasswordTooShort\|ErrPassword" aethel-core/internal/service/auth_service.go` returns at least one match; `go build ./...` passes.
- **Estimated effort:** S

#### 2.8 — RBAC Require middleware does not emit audit events
- **Finding:** `internal/rbac/middleware.go:55-78` — when `Require()` rejects a request (403), it does not write a `PERMISSION_DENIED` or `RBAC_ELEVATION_ATTEMPT` audit event. Sprint 1 definition of done explicitly requires RBAC rejections to be logged to the audit ledger.
- **Required fix:** Inject `audit.Writer` into the RBAC middleware (pass it at construction time or pull it from request context). On rejection, call `auditWriter.Write(ctx, audit.Event{Type: "PERMISSION_DENIED", ...})` before writing the 403 response. For `SYS_ADMIN`-only routes (`admin.audit`), use event type `RBAC_ELEVATION_ATTEMPT`.
- **Acceptance criteria:** `grep -n "audit\|PERMISSION_DENIED\|RBAC_ELEVATION" aethel-core/internal/rbac/middleware.go` returns at least one match; `go build ./...` passes.
- **Estimated effort:** M

---

### Block 3 — Database & Test Gaps (Important 🟡)

#### 3.1 — Session expiry regression in session_repo inline SQL
- **Finding:** `internal/database/repos/session_repo.go::GetByTokenHash` — the inline SQL query does NOT include `WHERE expires_at > now()`. The YAML definition `session.get_session_by_token_hash` does include this filter. As a result, expired sessions may be accepted as valid by the auth middleware, meaning users whose sessions have expired are not properly logged out.
- **Required fix:** As part of fix 2.5 (migrating session_repo to QueryRegistry), ensure the `GetByTokenHash` implementation uses the YAML query `session.get_session_by_token_hash` which already has the `expires_at > now()` filter. Do not write a new inline SQL; use the existing YAML entry.
- **Acceptance criteria:** `grep -n "expires_at" aethel-core/internal/database/repos/session_repo.go` returns at least one match (from the QueryRegistry path); `grep -rn '"SELECT.*user_sessions' aethel-core/internal/database/repos/session_repo.go` returns 0 matches.
- **Estimated effort:** S (covered by 2.5)

#### 3.2 — Missing notification queries in queries.yaml
- **Finding:** `aethel-core/internal/database/queries/queries.yaml` — zero queries defined for the `notifications` table. The table exists (migration 14), the domain type exists, and the routes `GET /notifications` and `PATCH /notifications/{id}/read` are registered in `server.go` but are listed as stub/missing routes.
- **Required fix:** Add a `notifications` query group to `queries.yaml` with at minimum: `notifications.list_by_user` (`SELECT ... WHERE user_id = $1 ORDER BY created_at DESC`), `notifications.mark_read` (`UPDATE notifications SET is_read = true WHERE id = $1 AND user_id = $2`), and `notifications.mark_all_read` (`UPDATE notifications SET is_read = true WHERE user_id = $1`). Then implement the notification repo and wire the stub handlers.
- **Acceptance criteria:** `grep -n "notifications" aethel-core/internal/database/queries/queries.yaml` returns at least 3 matches; `go build ./...` passes.
- **Estimated effort:** M

#### 3.3 — CI workflow missing PostgreSQL service container
- **Finding:** `.github/workflows/ci.yml` — the `test-backend` job has no `services: postgres:` block. Integration tests in `aethel-core/internal/integration/` (tagged with `//go:build integration`) require a running PostgreSQL database — they skip gracefully but never actually execute in CI.
- **Required fix:** Add a `services` block to the `test-backend` job in `ci.yml`:
  ```yaml
  services:
    postgres:
      image: postgres:16-alpine
      env:
        POSTGRES_PASSWORD: ci_test_password
        POSTGRES_DB: aethel_test
      options: >-
        --health-cmd pg_isready
        --health-interval 10s
        --health-timeout 5s
        --health-retries 5
      ports:
        - 5432:5432
  ```
  Set the `AETHEL_DSN` env var in the job to point to this service. Add `-tags integration` to the `go test` step.
- **Acceptance criteria:** `grep -n "services:" .github/workflows/ci.yml` returns at least one match; `grep -n "postgres:" .github/workflows/ci.yml` returns at least one match.
- **Estimated effort:** S

#### 3.4 — CD workflow missing migration step before rollout
- **Finding:** `.github/workflows/cd.yml` — the `deploy-staging` job runs `kubectl set image` immediately after building and pushing the Docker image, with no migration step. Schema changes in the new image will break running pods until migrations are applied manually.
- **Required fix:** In the `deploy-staging` job, before `kubectl set image`, add a step that runs `k8s-rollout.sh` (or inline the migration-job pattern from that script): create a Kubernetes Job that runs `aethel migrate up`, wait for it to complete successfully, then proceed with `kubectl set image`. The script already exists at `aethel-scripts/k8s-rollout.sh` — call it from the workflow step.
- **Acceptance criteria:** `grep -n "k8s-rollout\|migrate up\|kubectl.*Job" .github/workflows/cd.yml` returns at least one match.
- **Estimated effort:** S

---

### Block 4 — Frontend Quality (Important 🟡)

#### 4.1 — Zero ARIA attributes on bare `<button>` elements
- **Finding:** Five bare `<button>` elements have no accessible name (no `aria-label`, `aria-labelledby`, or inner text visible to screen readers): `auth/login.vue:85`, `dashboard.vue:139-151`, `WorkspaceSidebar.vue:103`, `WorkspaceSidebar.vue:226`, `admin/navigation.vue:186`. This is a hard WCAG 2.1 AA failure.
- **Required fix:** Add `aria-label` attributes to each bare `<button>` element with a descriptive string (e.g., `aria-label="Toggle sidebar"`, `aria-label="Clear search"`, `aria-label="Close password reset"`). For icon-only buttons, also add `aria-hidden="true"` to the icon element inside.
- **Acceptance criteria:** `grep -n "<button" aethel-view/app/pages/auth/login.vue aethel-view/app/pages/dashboard.vue aethel-view/app/components/layout/WorkspaceSidebar.vue aethel-view/app/pages/admin/navigation.vue | grep -v "aria-label"` returns 0 matches (all `<button>` elements have `aria-label`); `pnpm build` in `aethel-view/` passes.
- **Estimated effort:** S

#### 4.2 — useRuntimeConfig.refresh() never calls real API
- **Finding:** `aethel-view/app/composables/useRuntimeConfig.ts:176-181` — `refresh()` is a `setTimeout(300ms)` stub that never calls `GET /api/v1/config`. Once the backend is live, the frontend config will never update after initial load unless the user hard-reloads.
- **Required fix:** Replace the `setTimeout` stub in `refresh()` with a real `$apiFetch('/api/v1/config')` call. The response shape already matches the `AppRuntimeConfig` type defined in the composable. Wrap with try/catch and set `isLoading` correctly. This item is partially blocked on backend availability, but the frontend code must be written now so it is ready to connect.
- **Acceptance criteria:** `grep -n "setTimeout\|300" aethel-view/app/composables/useRuntimeConfig.ts` returns 0 matches in the `refresh()` function body; `grep -n "apiFetch.*config\|api/v1/config" aethel-view/app/composables/useRuntimeConfig.ts` returns at least one match; `pnpm build` passes.
- **Estimated effort:** S

---

### Block 5 — DevOps Gaps (Important 🟡)

#### 5.1 — Frontend Dockerfile not using distroless base
- **Finding:** `aethel-view/Dockerfile` — the production stage uses `node:20-alpine` instead of a distroless image. The backend Dockerfile correctly uses `gcr.io/distroless/static-debian12`. Alpine still includes a shell, package manager, and many unused binaries, increasing the attack surface.
- **Required fix:** Switch the final stage of `aethel-view/Dockerfile` to `gcr.io/distroless/nodejs20-debian12` (which includes only the Node.js runtime, no shell). Copy the `.output/` Nuxt directory and set `CMD ["/nodejs/bin/node", ".output/server/index.mjs"]`. Ensure the `aethel` user UID matches (or use the `nonroot` user from distroless at uid 65532).
- **Acceptance criteria:** `grep -n "distroless" aethel-view/Dockerfile` returns at least one match; `docker build -f aethel-view/Dockerfile aethel-view/` exits 0.
- **Estimated effort:** S

#### 5.2 — Frontend k8s deployment missing readOnlyRootFilesystem
- **Finding:** `k8s/frontend/deployment.yaml` — the frontend pod's `securityContext` is missing `readOnlyRootFilesystem: true`. The backend deployment (`k8s/backend/deployment.yaml`) already has this set. Inconsistency leaves the frontend container able to write to its filesystem.
- **Required fix:** Add `readOnlyRootFilesystem: true` to the `securityContext` of the frontend container in `k8s/frontend/deployment.yaml`. If the Nuxt output server writes temp files, add a `emptyDir` volume mounted at the necessary path (typically `/tmp`).
- **Acceptance criteria:** `grep -n "readOnlyRootFilesystem" k8s/frontend/deployment.yaml` returns at least one match with value `true`.
- **Estimated effort:** S

---

### Block 6 — Nice-to-Have Improvements (🟢 — optional, skip if time-constrained)

- `auth/login.vue:103-112` — redundant double-handler: UButton `type="button" @click` plus `form @submit.prevent`; change to `type="submit"` to use single handler path.
- `admin/reports.vue:101-134` — chart stub placeholder divs should be replaced with `UAlert` components showing "requires backend connection" message.
- `internal/config/handler.go:239` — `nullIfEmpty()` Go helper returns its argument unchanged (relies on PostgreSQL `NULLIF`); rename to `maybeNull()` or add a comment explaining the delegation to avoid future confusion.
- `documents/[id].vue:12-13` — `?? documents[0]` fallback shows the wrong document on a bad ID; replace with a 404 / `throw createError({ statusCode: 404 })` call.
- `internal/api/middleware/rate_limiter.go:107-110` — X-Forwarded-For IP taken from untrusted first entry; rate limit bypass via header spoofing. Use rightmost non-private entry or replace with a trusted-proxy library.
- `db-harden.sql:22-25` — REVOKE does not auto-propagate to new monthly audit partitions; add a partition-creation trigger or a monthly cron job to re-apply the REVOKE.
- `workflow.fetch_minute_sheet_timeline` in `queries.yaml` — dead YAML query never called via `qr.Get()`; remove or wire it to reduce startup prepared-statement overhead.
- `CLAUDE.md` migration table — update baseline to document migration 22 (org seed, added 2026-06-03).
- `dispatch/inbound/index.vue`, `dispatch/outbound/index.vue`, `dispatch/inbound/new.vue` — add `definePageMeta({ middleware: ['role'], requiredRole: 'RECEPTION' })` to restrict reception-only pages.
- `auth_service.go:28` vs `auth.go:93` — JWT TTL mismatch: `accessTokenDuration=30min` but `expires_in:900` (15min) advertised in login response; align the advertised value to match the real exp claim.

---

## Definition of Done

- [ ] All Block 1 (🔴 Critical) items resolved
- [ ] All Block 2–5 (🟡 Important) items resolved
- [ ] `go build ./...` and `go vet ./...` pass with zero errors
- [ ] `pnpm build` in `aethel-view/` passes with zero errors
- [ ] `go test ./...` — all tests pass
- [ ] Palette CSS violations: 0 (`grep -rn "text-slate\|text-indigo\|bg-slate\|bg-white\|border-slate\|border-indigo" aethel-view/app/` returns 0)
- [ ] localStorage violations: 0 (`grep -rn "localStorage" aethel-view/app/` returns 0)
- [ ] Inline SQL violations: 0 (`grep -rn '"SELECT\|"INSERT\|"UPDATE\|"DELETE' aethel-core/internal/database/repos/` returns 0)
- [ ] `docker build -f aethel-core/Dockerfile .` succeeds from repo root
- [ ] No hardcoded secrets: `grep -rn "dev-secret" aethel-core/` returns 0
- [ ] BlockRichText.vue: no unguarded v-html (`grep -n 'v-html="content"' aethel-view/app/components/blocks/BlockRichText.vue` returns 0)
- [ ] Content-Security-Policy header present: `grep -n "Content-Security-Policy" aethel-core/internal/api/middleware/security_headers.go` returns at least 1
- [ ] No hot-path panics: `grep -rn "panic" aethel-core/internal/api/handlers/auth.go aethel-core/internal/database/query_registry.go` returns 0
- [ ] Re-run Task 15 after fixes to verify overall score improves above 84%
