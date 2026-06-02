# Task 08 — Sprint 1.5: Authentication Wiring + Full Security Hardening

**Working directory:** `aethel-workspace/` (repo root)  
**Targets:** `aethel-core/` (backend) + `aethel-view/` (frontend)  
**Sprint:** 1.5 — runs after Task 06 (Sprint 1) is complete and verified  
**Depends on:** `go build ./...` clean in `aethel-core/`, `POST /api/v1/auth/login` returning a real JWT

---

## Your Role

You are a security-focused full-stack engineer. Your job is to implement every security measure described in `docs/architecture/architecture-security.md` and wire the Nuxt frontend authentication flow to the real Go backend. Security is the highest-priority concern in this task. If a shortcut compromises security — even slightly — do not take it. Correctness and safety come before brevity.

This task has two halves that build on each other:

1. **Backend hardening** — enforce every security measure in the Go server that the design specifies but that is not yet confirmed implemented (security headers, CSRF, rate limiting, refresh token rotation, JWT claim correction, database role restriction).
2. **Frontend auth wiring** — replace the mock login flow with a real, cryptographically-safe auth composable and route guard system. Never store tokens in `localStorage`. Never trust client-side role data over server-issued JWT claims.

---

## Threat Model — What You Are Defending Against

Keep this in mind for every decision in this task. A reviewer will check each measure against this list.

| Threat | Mitigation | Where |
|---|---|---|
| Password database breach | Argon2id — memory-hard, salted, PHC format | `auth_service.go` (Sprint 1) |
| Token theft via XSS | Access token in memory only (`useState`), never `localStorage`; refresh token in `httpOnly` cookie | `useAuth.ts` |
| Cross-site request forgery | Double-submit cookie pattern (`csrf_token` cookie + `X-CSRF-Token` header) | Backend CSRF middleware + frontend interceptor |
| Credential stuffing / brute force login | Per-IP rate limiting (pre-auth) + account lockout after N failures | Backend rate limiter + `auth_service.go` |
| Session hijacking after logout | Server-side session deletion on logout (not just client-side clear) | `POST /api/v1/auth/logout` handler |
| Refresh token reuse / theft | Refresh token rotation: each `/auth/refresh` call issues a new refresh token and invalidates the old one | `auth_service.go` + `session_repo.go` |
| JWT replay after account deactivation | Access tokens are short-lived (15 min); deactivated users cannot refresh | Auth middleware + `auth_service.Login` |
| Clickjacking | `X-Frame-Options: DENY` header | Security headers middleware |
| MIME sniffing | `X-Content-Type-Options: nosniff` header | Security headers middleware |
| Information leakage on login | Never reveal whether an email address exists — always return the same error for wrong email and wrong password | `auth_service.Login` (verify in Sprint 1 code) |
| Secrets in logs | Env vars never logged, never in error messages | Startup code review |
| Audit ledger tampering | PostgreSQL role has no `UPDATE`/`DELETE` on `audit_ledger` | DB permission hardening script |
| Privilege escalation | RBAC middleware rejects mismatched role claims; `SYS_ADMIN` gated separately | `rbac/middleware.go` |
| Insecure JWT org claim | `org` UUID removed from JWT claims (single-tenant; claim is meaningless and adds surface area) | JWT issuer in `auth_service.go` |

---

## Step 0 — Load Context (Mandatory)

Read every file listed. Then output a numbered list of exactly what you will create or modify. Do not write any code before producing that list.

```
# Architecture & security design
docs/architecture/architecture-security.md            ← FULL DOCUMENT — primary reference for every decision
docs/architecture/architecture-server.md              ← middleware stack order
docs/plans/agile-implementation-plan.md               ← Sprint 1 DoD (confirm it is met before proceeding)
CLAUDE.md                                             ← single-tenant model, JWT claims, middleware stack

# Backend files
aethel-core/internal/service/auth_service.go          ← Login, RefreshSession, RevokeSession — find any orgID refs to remove
aethel-core/internal/api/server.go                    ← current middleware stack; where to add SecurityHeaders + CSRF
aethel-core/internal/api/handlers/auth.go             ← response shapes for login, refresh, logout
aethel-core/internal/database/repos/session_repo.go   ← Create, DeleteByID (refresh token rotation requires DeleteByID + Create in one tx)
aethel-core/cmd/aethel/main.go                        ← startup sequence; confirm secrets are not logged

# Frontend files
aethel-view/app/pages/auth/login.vue                  ← current mock implementation to replace
aethel-view/app/composables/useRuntimeConfig.ts       ← AppRuntimeConfig shape (do not modify)
aethel-view/app/composables/useMockData.ts            ← role switcher (understand before touching)
aethel-view/app/components/layout/WorkspaceSidebar.vue ← reads currentUser from useMockData
aethel-view/app/components/layout/WorkspaceNavbar.vue  ← role switcher UI + logout button location
aethel-view/app/layouts/workspace.vue                 ← layout used by all protected pages
aethel-view/.claude-devtools/settings.json            ← READ THIS before any nuxt.config.ts change
```

---

## Step 1 — Backend: Fix JWT Claims

Open `aethel-core/internal/service/auth_service.go`. Find where JWT claims are assembled (the `IssueAccessToken` function or equivalent).

**Remove the `org` claim.** This system is single-tenant; the `org` UUID in the JWT is meaningless and adds unnecessary surface area. The JWT must contain exactly these claims and no others:

```go
claims := jwt.MapClaims{
    "sub":  userID.String(),    // User UUID
    "role": user.Role,          // "ADMIN" | "RECEPTION" | "USER" | "SYS_ADMIN"
    "iat":  now.Unix(),
    "exp":  now.Add(accessTokenTTL).Unix(),
    "jti":  uuid.New().String(), // JWT ID for future revocation tracking
}
```

Update the Auth middleware in `aethel-core/internal/rbac/middleware.go` (or wherever JWT claims are read from context) to remove any `orgID` extraction.

---

## Step 2 — Backend: Refresh Token Rotation

Open `aethel-core/internal/service/auth_service.go` and find `RefreshSession`.

Implement **refresh token rotation**: every call to the refresh endpoint must issue a brand-new opaque refresh token AND delete the old session row — in a single database transaction. A stolen refresh token used a second time must return `401`.

```go
// RefreshSession flow (enforce this exactly):
// 1. Hash the incoming opaque token with SHA-256.
// 2. Look up the session by token hash. If not found or expired → ErrUnauthorized.
// 3. BEGIN TRANSACTION
// 4. DELETE the old session row (session_repo.DeleteByID).
// 5. INSERT a new session row with a new opaque token + new expiry (session_repo.Create).
// 6. COMMIT
// 7. Issue a new access token JWT.
// 8. Return both the new access token and the new refresh token to the caller.
//
// If the DELETE succeeds but the INSERT fails: ROLLBACK. Return 500.
// The old token is gone and the user must log in again — this is acceptable.
// It is not acceptable to leave the old token valid after issuing a new one.
```

Update `aethel-core/internal/database/repos/session_repo.go` to add a `RotateSession(ctx, oldID, newSession) error` method that performs steps 4–5 in a transaction. Do not do two separate calls from the service layer — the atomicity must be at the repo level.

---

## Step 3 — Backend: Security Headers Middleware

Create `aethel-core/internal/api/middleware/security_headers.go`.

This middleware must be the **first middleware after Recovery** in the stack. It sets security-relevant HTTP response headers on every response regardless of route or authentication state.

```go
package middleware

func SecurityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        h := w.Header()
        h.Set("X-Content-Type-Options", "nosniff")
        h.Set("X-Frame-Options", "DENY")
        h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
        h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
        h.Set("X-XSS-Protection", "0") // Disabled intentionally: modern browsers use CSP; this header causes bugs in legacy IE
        // HSTS: only set over HTTPS. Check the X-Forwarded-Proto header set by the reverse proxy.
        if r.Header.Get("X-Forwarded-Proto") == "https" || r.TLS != nil {
            h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        }
        // API responses must not be cached by intermediaries.
        h.Set("Cache-Control", "no-store")
        next.ServeHTTP(w, r)
    })
}
```

Register in `aethel-core/internal/api/server.go`:

```
[Recovery] → [SecurityHeaders] → [RequestID] → [StructuredLogger] → [RateLimiter] → [CORS] → [Auth] → [CSRF] → [RBAC] → [Handler]
```

Update `docs/architecture/architecture-server.md` to add `[SecurityHeaders]` in the documented middleware stack.

---

## Step 4 — Backend: CSRF Middleware

Create `aethel-core/internal/api/middleware/csrf.go`.

Implement the double-submit cookie pattern as specified in `docs/architecture/architecture-security.md`:

```go
// CSRFProtect protects state-changing routes (POST/PUT/PATCH/DELETE) for browser clients.
// Algorithm:
//   1. Read-only methods (GET, HEAD, OPTIONS): pass through.
//   2. If no "csrf_token" cookie: pass through (non-browser/API client path).
//   3. Read the X-CSRF-Token request header.
//   4. Compare header value to cookie value using crypto/subtle.ConstantTimeCompare.
//      ConstantTimeCompare prevents timing attacks that could leak the token length.
//   5. If they differ: respond 403 with JSON body {"error":"csrf token mismatch"}.
//
// The csrf_token cookie is set by the login handler (see Step 6).
func CSRFProtect(next http.Handler) http.Handler
```

Place CSRF middleware **after Auth and before RBAC** in the stack (Step 3 diagram).

CSRF applies only to authenticated routes — if the Auth middleware already rejected the request with 401, CSRF never runs.

---

## Step 5 — Backend: Verify Rate Limiting and Account Lockout

Open `aethel-core/internal/api/server.go` and `aethel-core/internal/api/middleware/` (wherever the rate limiter lives).

Confirm or implement:

**Rate limiter is wired at three levels:**
- Global pre-auth: 600 RPM per IP
- Per-user post-auth: 300 RPM per user ID (read `userID` from JWT context)
- `/api/v1/auth/login` specifically: 20 RPM per IP (much lower — this is the credential stuffing target)

If the rate limiter is a stub, implement a real in-process token bucket using `sync.Map`. Key structure: `"ip:<addr>"` for per-IP, `"user:<uuid>"` for per-user.

**Account lockout is wired:**
Open `auth_service.go`. Verify `Login` does all of the following:
- After a failed password check: call `userRepo.IncrementFailedLogins(ctx, userID)`
- After increment: if `failedLoginAttempts >= lockoutThreshold` (default: 5), call `userRepo.LockUntil(ctx, userID, time.Now().Add(lockoutDuration))` where `lockoutDuration` is 15 minutes
- At the start of `Login`, if `user.LockedUntil != nil && time.Now().Before(*user.LockedUntil)`, return `ErrAccountLocked` **without checking the password** (timing side-channel: a locked user must never reach the Argon2id verification step)
- On successful login: call `userRepo.ResetFailedLogins(ctx, userID)` before issuing tokens
- Write `USER_LOGIN_FAILED` audit event on every failure; write `USER_LOGIN` audit event on success (include client IP from request context)

If any of these are missing from the Sprint 1 implementation, add them now.

---

## Step 6 — Backend: Login Handler Sets Cookies

Open `aethel-core/internal/api/handlers/auth.go`. Find the login handler.

On successful login, the handler must set **two cookies** in addition to returning the access token in the JSON body:

```go
// 1. Refresh token — httpOnly, inaccessible to JavaScript
http.SetCookie(w, &http.Cookie{
    Name:     "refresh_token",
    Value:    refreshToken, // opaque random string
    Path:     "/api/v1/auth/refresh",
    HttpOnly: true,
    Secure:   isHTTPS(r),  // true when X-Forwarded-Proto == "https" or r.TLS != nil
    SameSite: http.SameSiteStrictMode,
    MaxAge:   int(7 * 24 * time.Hour / time.Second),
})

// 2. CSRF token — NOT httpOnly; JavaScript must be able to read it
csrfToken := generateSecureToken(32) // crypto/rand, base64url encoded, 32 bytes
http.SetCookie(w, &http.Cookie{
    Name:     "csrf_token",
    Value:    csrfToken,
    Path:     "/",
    HttpOnly: false,  // intentionally readable by JS
    Secure:   isHTTPS(r),
    SameSite: http.SameSiteStrictMode,
    MaxAge:   int(7 * 24 * time.Hour / time.Second),
})

// JSON response body (access token only — never include refresh token in body)
json.NewEncoder(w).Encode(LoginResponse{
    AccessToken: accessToken,
    TokenType:   "Bearer",
    ExpiresIn:   int(accessTokenTTL / time.Second),
    Role:        user.Role,    // include role so frontend doesn't need to decode JWT
})
```

The refresh handler (`POST /api/v1/auth/refresh`):
- Reads the refresh token from the `refresh_token` cookie (not a request body parameter)
- After rotation (Step 2), sets a new `refresh_token` cookie and a new `csrf_token` cookie
- Returns a new `{ access_token, expires_in }` JSON response

The logout handler (`POST /api/v1/auth/logout`):
- Reads the refresh token from the cookie; calls `sessionRepo.DeleteByID`
- Clears both cookies by setting `MaxAge: -1`
- Writes a `USER_LOGOUT` audit event

---

## Step 7 — Backend: Database Role Hardening

Create `aethel-core/scripts/db-harden.sql`:

```sql
-- Run once as the database superuser after migrations are applied.
-- This removes the ability for the application user to modify the audit ledger.
-- The application connects as 'aethel_app' (or whatever AETHEL_DB_USER is set to).
-- Replace 'aethel_app' with the actual role name from server-database.yaml.

REVOKE UPDATE, DELETE ON TABLE audit_ledger FROM aethel_app;

-- Verify (should show no UPDATE or DELETE privilege for aethel_app):
\dp audit_ledger
```

Create `aethel-scripts/db-harden.sh`:

```bash
#!/usr/bin/env bash
# Usage: PGPASSWORD=... ./db-harden.sh
# Applies db-harden.sql as the superuser. Run once after initial setup.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
psql "${DATABASE_SUPERUSER_DSN}" -f "${SCRIPT_DIR}/../aethel-core/scripts/db-harden.sql"
echo "Database hardening complete."
```

Add a note to `docs/guides/it-customization-guide.md` under a new "Security Hardening" section: this script must be run after initial database setup. Link to `aethel-scripts/db-harden.sh`.

---

## Step 8 — Backend: Verify Secrets Are Never Logged

Open `aethel-core/cmd/aethel/main.go`. Find where environment variables are read (`AETHEL_JWT_SECRET`, `AETHEL_DB_PASSWORD`, `AETHEL_DB_DSN`).

Confirm (or add) these rules:
- Never pass secret env var values to `log.Printf`, `zerolog`, or `fmt.Println`
- On startup success, log only: `"JWT algorithm: HS256"` — not the key value
- On startup failure (missing required env var): log `"required env var AETHEL_JWT_SECRET is not set"` — not a partial value
- Write a grep check into the verification step: `grep -rn "JWT_SECRET\|DB_PASSWORD" aethel-core/ --include="*.go"` must show only assignments and existence checks, never format strings that would print the value.

---

## Step 9 — Frontend: Create `useAuth()` Composable

Create `aethel-view/app/composables/useAuth.ts`.

This is the only source of truth for authentication state in the frontend. No other file may directly access tokens or make auth API calls — all auth logic goes through this composable.

```typescript
// Token storage strategy — rationale in comments:
// Access token: in-memory via useState. Not localStorage (XSS risk). Survives
//   navigation. Lost on hard page reload — initAuth() recovers it via the
//   httpOnly refresh cookie without any user interaction.
// Refresh token: httpOnly cookie set/cleared by the server. JavaScript cannot
//   read it. This is the safest possible storage for a long-lived credential.
// CSRF token: readable cookie set by the server on login. Read by this
//   composable and sent in X-CSRF-Token on every state-changing request.

export interface AuthUser {
  id: string
  role: 'ADMIN' | 'RECEPTION' | 'USER' | 'SYS_ADMIN'
}

export interface LoginCredentials {
  email: string
  password: string
}

export function useAuth() {
  // In-memory access token. null = not authenticated.
  const accessToken = useState<string | null>('auth:access-token', () => null)

  // Decoded user from JWT payload. Computed — never stored separately.
  const user = computed<AuthUser | null>(() => {
    if (!accessToken.value) return null
    try {
      // Decode the JWT payload (middle segment). Do NOT verify signature client-side —
      // the server has already verified it. We only need the claims for UI rendering.
      const payload = JSON.parse(atob(accessToken.value.split('.')[1]))
      if (!payload.sub || !payload.role || !payload.exp) return null
      if (Date.now() / 1000 > payload.exp) return null // expired in memory
      return { id: payload.sub, role: payload.role }
    } catch {
      return null
    }
  })

  const isAuthenticated = computed(() => !!user.value)

  // Read the CSRF token from the readable cookie.
  // Returns empty string if not present (non-browser / fresh page load before login).
  function getCSRFToken(): string {
    if (import.meta.server) return ''
    const match = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]+)/)
    return match ? decodeURIComponent(match[1]) : ''
  }

  async function login(credentials: LoginCredentials): Promise<void> {
    // POST /api/v1/auth/login
    // Server sets httpOnly refresh_token cookie and readable csrf_token cookie.
    // Response body contains the access token.
    const data = await $fetch<{ access_token: string; expires_in: number; role: string }>(
      '/api/v1/auth/login',
      {
        method: 'POST',
        body: credentials,
        // Do NOT send credentials: 'include' here — cookies are set by the server
        // response, not sent in this request.
      }
    )
    accessToken.value = data.access_token
  }

  async function logout(): Promise<void> {
    try {
      await $fetch('/api/v1/auth/logout', {
        method: 'POST',
        headers: { 'X-CSRF-Token': getCSRFToken() },
      })
    } finally {
      // Always clear client state, even if server call fails.
      accessToken.value = null
    }
  }

  // Attempts to get a new access token using the httpOnly refresh cookie.
  // Returns true on success, false if the session has expired.
  async function refresh(): Promise<boolean> {
    try {
      const data = await $fetch<{ access_token: string }>('/api/v1/auth/refresh', {
        method: 'POST',
        headers: { 'X-CSRF-Token': getCSRFToken() },
        // The browser sends the refresh_token httpOnly cookie automatically.
      })
      accessToken.value = data.access_token
      return true
    } catch {
      accessToken.value = null
      return false
    }
  }

  // Called once on app start. Attempts silent token recovery.
  // If the httpOnly refresh cookie is present, this succeeds silently.
  // If not, the user must log in manually.
  async function initAuth(): Promise<void> {
    if (accessToken.value) return // already have one (SSR hydration)
    await refresh()
  }

  async function requestPasswordReset(email: string): Promise<void> {
    await $fetch('/api/v1/auth/password-reset/request', {
      method: 'POST',
      body: { email },
      // No CSRF token: this is a public endpoint (no session cookie present)
    })
    // Always resolve without error — do not reveal whether the email exists.
    // The server also follows this rule. The UI shows a generic confirmation.
  }

  return {
    accessToken: readonly(accessToken), // expose as readonly to prevent external mutation
    user,
    isAuthenticated,
    login,
    logout,
    refresh,
    initAuth,
    requestPasswordReset,
    getCSRFToken,
  }
}
```

---

## Step 10 — Frontend: Auth Plugin (Silent Token Recovery)

Create `aethel-view/app/plugins/auth.client.ts`.

The `.client.ts` suffix ensures this runs only in the browser (not during SSR), which is correct because the httpOnly cookie recovery is a browser-only operation.

```typescript
export default defineNuxtPlugin(async () => {
  const auth = useAuth()
  // Attempt silent recovery on every page load.
  // If the user has a valid refresh cookie, they get their access token back
  // without seeing the login page. If not, the route guard will redirect them.
  await auth.initAuth()
})
```

---

## Step 11 — Frontend: Route Middleware

Create `aethel-view/app/middleware/auth.ts`:

```typescript
// Protects all routes that require authentication.
// Redirects to /auth/login if the user is not authenticated.
// Applied globally via definePageMeta or in the workspace layout.
export default defineNuxtRouteMiddleware((to) => {
  // Allow access to auth pages without authentication.
  if (to.path.startsWith('/auth/')) return

  const { isAuthenticated } = useAuth()
  if (!isAuthenticated.value) {
    return navigateTo('/auth/login', { replace: true })
  }
})
```

Create `aethel-view/app/middleware/role.ts`:

```typescript
// Role-based access guard. Usage in page:
//   definePageMeta({ middleware: [{ name: 'role', meta: { requiredRole: 'ADMIN' } }] })
// Or via a custom route-level meta field checked here.
export default defineNuxtRouteMiddleware((to) => {
  const { user } = useAuth()
  const requiredRole = to.meta.requiredRole as string | undefined
  if (!requiredRole) return

  const roleHierarchy: Record<string, number> = {
    USER: 1,
    RECEPTION: 2,
    ADMIN: 3,
    SYS_ADMIN: 4,
  }

  if (!user.value || (roleHierarchy[user.value.role] ?? 0) < (roleHierarchy[requiredRole] ?? 0)) {
    return navigateTo('/dashboard', { replace: true })
  }
})
```

Apply both middleware in `aethel-view/app/layouts/workspace.vue`. Add inside `<script setup>`:

```typescript
definePageMeta({ middleware: ['auth', 'role'] })
```

Apply `requiredRole: 'ADMIN'` meta to all `/admin/*` pages via their individual `definePageMeta`.

---

## Step 12 — Frontend: Wire `login.vue`

Replace the mock `handleLogin` in `aethel-view/app/pages/auth/login.vue` with a real implementation.

Remove the `useMockData` import entirely from this file.

```typescript
const { login } = useAuth()
const router = useRouter()

const form = reactive({ email: '', password: '' })
const loading = ref(false)
const error = ref<string | null>(null) // displayed to user on failure

async function handleLogin() {
  error.value = null
  loading.value = true
  try {
    await login({ email: form.email, password: form.password })
    await router.push('/dashboard')
  } catch (e: unknown) {
    // Map specific HTTP status codes to user-facing messages.
    // Never reveal whether the email exists.
    const status = (e as { statusCode?: number }).statusCode
    if (status === 401) {
      error.value = 'Invalid email or password.'
    } else if (status === 423) {
      error.value = 'Account is temporarily locked due to too many failed attempts. Try again in 15 minutes.'
    } else if (status === 429) {
      error.value = 'Too many login attempts. Please wait before trying again.'
    } else {
      error.value = 'An unexpected error occurred. Please try again.'
    }
  } finally {
    loading.value = false
  }
}
```

Add the error display to the template — place it between the form fields and the submit button:

```html
<UAlert
  v-if="error"
  color="red"
  variant="soft"
  :title="error"
  icon="i-lucide-alert-circle"
  class="mb-2"
/>
```

Wire the "Forgot password" button to open a modal or navigate to `/auth/reset-password`:

```typescript
const { requestPasswordReset } = useAuth()
const resetEmail = ref('')
const resetSent = ref(false)

async function handlePasswordReset() {
  // Always show the same success message regardless of whether the email exists.
  await requestPasswordReset(resetEmail.value)
  resetSent.value = true
}
```

---

## Step 13 — Frontend: Global `$fetch` Interceptor

Create `aethel-view/app/plugins/fetch.ts` (runs on both server and client):

```typescript
export default defineNuxtPlugin(() => {
  const auth = useAuth()

  $fetch.create({
    onRequest({ options }) {
      // Attach Bearer token to every request that has one.
      if (auth.accessToken.value) {
        options.headers = {
          ...options.headers,
          Authorization: `Bearer ${auth.accessToken.value}`,
          'X-CSRF-Token': auth.getCSRFToken(),
        }
      }
    },

    async onResponseError({ response, options, request }) {
      if (response.status !== 401) return

      // Attempt silent token refresh on 401.
      // Guard against infinite loop: if the failing request was itself the
      // refresh endpoint, do not retry — redirect to login instead.
      if (typeof request === 'string' && request.includes('/auth/refresh')) {
        await navigateTo('/auth/login', { replace: true })
        return
      }

      const refreshed = await auth.refresh()
      if (!refreshed) {
        await navigateTo('/auth/login', { replace: true })
        return
      }

      // Retry the original request with the new token.
      // useFetch and $fetch callers will see the retried response.
      return $fetch(request as string, {
        ...options,
        headers: {
          ...options.headers,
          Authorization: `Bearer ${auth.accessToken.value}`,
          'X-CSRF-Token': auth.getCSRFToken(),
        },
      })
    },
  })
})
```

---

## Step 14 — Frontend: Logout in Navbar + Real Role in Layout

Open `aethel-view/app/components/layout/WorkspaceNavbar.vue`.

1. Find the logout action in the profile dropdown. Wire it to `useAuth().logout()` followed by `navigateTo('/auth/login')`.
2. The role switcher UI (ADMIN / RECEPTION / USER demo buttons) must **not** affect route guards or RBAC checks. Add a visible dev-only disclaimer badge: `<UBadge color="amber" variant="soft">Demo mode</UBadge>` next to the switcher. Route middleware reads from `useAuth().user.role` only — not from `useMockData`.

Open `aethel-view/app/components/layout/WorkspaceSidebar.vue`.

Find where `currentUser` from `useMockData` is used for role-gating nav groups. Replace the role source with `useAuth().user`:

```typescript
// Before (prototype mock):
const { currentUser } = useMockData()
// Used as: currentUser.value.role

// After (real auth):
const { user } = useAuth()
// Used as: user.value?.role
```

The nav group visibility logic (show/hide based on role) must use the real role from the JWT.

---

## Constraints — Do NOT Do These

- **Never** store the access token in `localStorage`, `sessionStorage`, or a readable cookie — `useState` memory only
- **Never** store the refresh token anywhere in JavaScript — it lives only in the httpOnly cookie
- **Never** decode the JWT to verify its signature on the client — only decode for UI rendering (the server has already verified the signature)
- **Never** return different error messages for "wrong email" vs "wrong password" — always return the same `401` with the same generic message
- **Never** skip the CSRF check on a `POST`/`PATCH`/`DELETE` route that has a CSRF cookie present
- **Never** log a secret value — env var contents, JWT signing keys, passwords, or token values
- **Never** commit `AETHEL_JWT_SECRET` or any secret in any file
- **Never** skip the `crypto/subtle.ConstantTimeCompare` for CSRF token comparison — regular `==` is vulnerable to timing attacks
- **Do not** touch migration SQL files
- **Do not** commit — the user will review and commit manually
- **Do not** run backend commands from `aethel-view/` or frontend commands from `aethel-core/`

---

## Step 15 — Verification (Security Checklist + Functional DoD)

Run every check. Every one must pass before reporting the task complete.

### Backend

```bash
cd aethel-core

# 1. Clean build
go build ./...

# 2. Vet
go vet ./...

# 3. Unit tests (Sprint 1 must still pass)
go test ./internal/service/... -v -count=1

# 4. Race detector
go test ./... -race -count=1

# 5. Secrets are never in format strings — must return 0 matches
grep -rn 'JWT_SECRET\|DB_PASSWORD\|RSA_PRIVATE' internal/ cmd/ --include="*.go" \
  | grep -v "os.Getenv\|os.LookupEnv\|required env var\|// " \
  | wc -l
# Expected: 0

# 6. CSRF token comparison uses ConstantTimeCompare — must return ≥1 match
grep -rn "ConstantTimeCompare" internal/ --include="*.go"
# Expected: at least 1 match in the CSRF middleware

# 7. org claim is not in JWT — must return 0 matches
grep -rn '"org"' internal/service/ internal/api/ --include="*.go" \
  | grep -v "//\|test"
# Expected: 0

# 8. Server starts and health probe responds
go run ./cmd/aethel serve &
SERVER_PID=$!
sleep 2
curl -sf http://localhost:8080/healthz | grep -q "ok" && echo "HEALTH: OK"
# Security headers present on response:
curl -sI http://localhost:8080/healthz | grep -i "x-content-type-options" | grep -q "nosniff" && echo "HEADERS: OK"
# Login endpoint rejects missing body with 400:
curl -sf -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" -d '{}' ; echo ""
# Login endpoint rejects wrong credentials with 401 (not 404):
curl -sw "%{http_code}" -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"nobody@test.com","password":"wrong"}' | tail -c 3
# Expected: 401
kill $SERVER_PID
```

### Frontend

```bash
cd aethel-view

# 1. TypeScript — zero errors
pnpm exec nuxi typecheck

# 2. No token stored in localStorage — must return 0 matches
grep -rn "localStorage\|sessionStorage" app/ --include="*.ts" --include="*.vue"
# Expected: 0

# 3. useMockData is not imported in login.vue
grep -rn "useMockData" app/pages/auth/login.vue
# Expected: 0

# 4. useAuth is the only file making auth API calls — confirm no other file calls /auth/ endpoints
grep -rn "auth/login\|auth/refresh\|auth/logout\|auth/password" app/ \
  --include="*.ts" --include="*.vue" \
  | grep -v "useAuth.ts\|middleware/auth.ts"
# Expected: 0 (all auth calls are in useAuth.ts)

# 5. Dev server starts without errors (run, check, stop)
pnpm dev &
FE_PID=$!
sleep 8
curl -sf http://localhost:3000/auth/login | grep -q "Sign in" && echo "LOGIN PAGE: OK"
# Unauthenticated request to protected route redirects to login:
STATUS=$(curl -sw "%{http_code}" -o /dev/null http://localhost:3000/dashboard)
echo "Dashboard unauthenticated status: $STATUS" # Expected: 302 or redirect HTML
kill $FE_PID
```

### Report back with:

1. Output of `go test ./internal/service/... -v` (all tests passing)
2. Output of each security grep (secrets check, ConstantTimeCompare, org claim check, localStorage check)
3. Output of `pnpm exec nuxi typecheck` (zero errors)
4. Confirmation that `curl -sw "%{http_code}"` on the login endpoint with wrong credentials returns `401` — not `404`, not `200`, not `403`
5. Screenshot or terminal output of the dev server with `http://localhost:3000/auth/login` rendering without JS errors in the browser console
6. A list of every new and modified file with a one-line description
7. Any deviation from this plan with a written security justification

---

## Reference: Sprint 1.5 Definition of Done

- `POST /api/v1/auth/login` with valid credentials sets httpOnly `refresh_token` cookie, readable `csrf_token` cookie, and returns `{ access_token, expires_in, role }` in the body
- `POST /api/v1/auth/refresh` rotates the refresh token (old token is deleted, new token is set), returns new access token
- `POST /api/v1/auth/logout` deletes the server-side session, clears both cookies
- Wrong email and wrong password both return `401` with identical response bodies
- An account with 5+ failed logins is locked for 15 minutes; the lock is enforced before Argon2id runs
- Every auth event (`USER_LOGIN`, `USER_LOGIN_FAILED`, `USER_LOGOUT`) appears in `audit_ledger`
- All API responses carry `X-Content-Type-Options`, `X-Frame-Options`, and `Referrer-Policy` headers
- CSRF middleware rejects a `PATCH` request where `X-CSRF-Token` header does not match the `csrf_token` cookie
- The Nuxt login page calls `POST /api/v1/auth/login` for real; successful login navigates to `/dashboard`
- Hard page reload on a protected route silently recovers the session via the httpOnly cookie without redirecting to login
- Navigating to `/dashboard` without a session redirects to `/auth/login`
- Admin pages (`/admin/*`) are inaccessible to `RECEPTION` and `USER` roles — route guard redirects to `/dashboard`
- `grep -rn "localStorage" app/` returns zero matches
- `pnpm exec nuxi typecheck` returns zero errors
