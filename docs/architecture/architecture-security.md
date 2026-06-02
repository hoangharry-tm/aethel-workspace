# Architecture — Embedded Security Measures

**Audience:** Go engineers, security reviewers
**Status:** Active

Aethel Core does not require a third-party identity provider. All authentication, authorization, audit, and tamper detection logic is implemented in the Go backend. This document describes each measure and the rationale for the choices made.

---

## Password Storage

Passwords are hashed with **Argon2id** before storage. Argon2id is the winner of the Password Hashing Competition (2015) and is the current OWASP recommendation for password hashing.

The cost parameters are hardcoded as Go constants with OWASP-recommended defaults and configurable via environment variables:

| Parameter | Default | Environment variable |
|---|---|---|
| `memory_kib` | 65536 (64 MiB) | `AETHEL_ARGON2_MEMORY_KIB` |
| `iterations` | 3 | `AETHEL_ARGON2_ITERATIONS` |
| `parallelism` | 4 | `AETHEL_ARGON2_PARALLELISM` |
| `salt_length` | 16 | — (not configurable) |
| `key_length` | 32 | — (not configurable) |

The default parameters (64 MiB, 3 iterations, 4 threads) are within OWASP's minimum recommendations for interactive logins. A production deployment that is not under severe memory pressure should set `AETHEL_ARGON2_MEMORY_KIB=131072` (128 MiB).

The hash format stored in the database is the PHC string format: `$argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>`. This format embeds the algorithm, version, and parameters, so stored hashes are self-describing and can be verified even if the cost parameters change in the blueprint.

A password change does not require all users to re-hash. The new parameters take effect on the next login for each user.

---

## JWT

The server issues JWTs on successful login. Two algorithms are supported, selected via blueprint:

**HS256 (HMAC-SHA256)** — A single shared secret signs and verifies all tokens. Simpler to operate. Appropriate for single-instance deployments or environments where the signing key never leaves the server. The secret is read from `AETHEL_JWT_SECRET` at startup.

**RS256 (RSA-SHA256)** — An asymmetric keypair. The private key signs tokens; the public key verifies them. This allows future scenarios where a second service verifies tokens without having access to the private key. The private key is read from `AETHEL_JWT_RSA_PRIVATE_KEY` (PEM-encoded) at startup.

JWT claims:

| Claim | Type | Notes |
|---|---|---|
| `sub` | UUID string | User ID |
| `org` | UUID string | Organization ID |
| `role` | string | `ADMIN`, `RECEPTION`, `USER`, `SYS_ADMIN` |
| `iat` | Unix timestamp | Issued at |
| `exp` | Unix timestamp | Expiry |
| `jti` | UUID string | JWT ID — used for token revocation if needed |

The access token is short-lived (default 15 minutes). The server validates signature and `exp` on every authenticated request without a database query. The refresh token is an opaque random string stored in `user_sessions`; it is exchanged for a new access token and has a longer TTL (default 7 days).

---

## CSRF

Browser clients that call the API from JavaScript are protected against cross-site request forgery via the **double-submit cookie pattern**:

1. On login, the server sets a `csrf_token` cookie (SameSite=Strict, HttpOnly=false, Secure=true in production).
2. JavaScript reads the cookie value and submits it in the `X-CSRF-Token` request header on every state-changing request (POST, PUT, PATCH, DELETE).
3. The CSRF middleware compares the header value to the cookie value. If they differ, the request is rejected with `403 Forbidden`.

This pattern works because a cross-site attacker can trigger a request that includes the cookie (the browser sends it automatically), but cannot read the cookie value from a different origin and therefore cannot set the correct header.

API clients that do not use browser cookies (mobile apps, CLI tools, server-to-server) omit the CSRF flow and rely on the JWT alone. The CSRF check is skipped when no `csrf_token` cookie is present — this exempts non-browser clients without requiring configuration.

---

## Rate Limiting

Three layers of rate limiting are applied:

**Per-IP, pre-auth** — Applied in the middleware stack before JWT parsing. Prevents brute-force login attempts and credential stuffing. The IP is extracted from `X-Forwarded-For` (trusting the configured reverse proxy) or `RemoteAddr` if no proxy is present.

**Per-user, post-auth** — Applied after the JWT is validated. Limits the request rate for a single authenticated user regardless of the IP they connect from.

**Per-endpoint cap** — Route groups can declare a lower `rate_limit_rpm` than the global default. This prevents specific high-cost operations (file upload, report generation) from saturating the server.

All three layers use token buckets. The bucket capacity equals the RPM limit; the refill rate is `rpm / 60` tokens per second. The current implementation is in-process (a `sync.Map` of bucket state keyed by IP or user ID). For horizontally scaled deployments, configure rate limiting at the reverse proxy layer (Nginx `limit_req_zone`, Caddy rate limit plugin) and the in-process limiter will be a secondary layer.

---

## TLS

The Go server listens on plain HTTP internally. TLS termination is handled by the reverse proxy (Nginx or Caddy). This is the standard pattern for containerized Go services: the proxy handles certificate management (ACME via Let's Encrypt), HTTP/2, and TLS termination, while the Go server handles only application logic.

For production deployments where the Go server is exposed directly (no reverse proxy), TLS can be enabled by setting `AETHEL_TLS_CERT_FILE` and `AETHEL_TLS_KEY_FILE` environment variables. When both are set, the server calls `http.ListenAndServeTLS` instead of `http.ListenAndServe`. This mode is not recommended for production but is useful for internal tooling.

The database connection uses `ssl_mode: verify-full` in production (see `server-database.yaml`). The connection between the Go process and PostgreSQL is always encrypted when the correct SSL mode is configured.

---

## Deployment Model

Aethel is a **single-tenant self-hosted application**. Each organization downloads, configures, and hosts their own instance. There is no SaaS or managed hosting. One running instance serves exactly one organization.

The `organizations` table holds exactly one row — the installation's own org profile (name, contact, timezone, etc.). The `organization_id` column present on other tables is a fixed UUID constant loaded from that row once at startup; it is not a per-request routing key and there is no TenantResolver middleware. No Row-Level Security is needed because there is only one tenant and no cross-tenant data isolation problem to solve.

This simplifies the security model significantly: authentication is about identifying which *user* is acting, not which *organization* the request belongs to.

---

## Audit Ledger

Every significant action in the system writes a row to the `audit_ledger` table. The following events are always recorded:

| Event type | Trigger |
|---|---|
| `USER_LOGIN` | Successful login |
| `USER_LOGIN_FAILED` | Failed login attempt |
| `USER_LOGOUT` | Explicit logout |
| `SESSION_REVOKED` | Session revoked by admin |
| `PERMISSION_DENIED` | RBAC middleware rejected a request |
| `RBAC_ELEVATION_ATTEMPT` | User attempted to access a resource above their role |
| `SECURITY_BREACH_ATTEMPT` | Request that triggered the anomaly detector |
| `UNAUTHORIZED_ACCESS_BYPASSED` | Request that bypassed expected access controls (only written if tamper detection fires) |
| `DISPATCH_CREATED` | New dispatch logged |
| `DISPATCH_ASSIGNED` | Dispatch assigned to user or department |
| `DISPATCH_DELIVERED` | Delivery acknowledgement recorded |
| `GREEN_NOTE_APPENDED` | Green note added to minute sheet |
| `ADMIN_USER_CREATED` | New user account created |
| `ADMIN_USER_DEACTIVATED` | User account deactivated |
| `ADMIN_SETTINGS_CHANGED` | System settings modified |
| `ROUTING_RULE_MODIFIED` | Routing rule created, updated, or deleted |

The `audit_ledger` table is append-only. No application code issues `UPDATE` or `DELETE` on this table. The PostgreSQL role used by the application (`aethel_prod_app`) must not have `UPDATE` or `DELETE` privileges on `audit_ledger`. This is enforced at the database permission level, not just in the application.

Reading the audit ledger requires the `admin.audit` permission, which is held only by the `SYS_ADMIN` role.

---

## Tamper Detection

The audit ledger includes a `previous_checksum` column that chains rows together. Each row's checksum is computed as:

```
SHA-256(
  row.id ||
  row.actor_user_id ||
  row.action_event_type ||
  row.target_resource_id ||
  row.ip_address ||
  row.created_at::text ||
  previous_row.checksum
)
```

The first row in each partition uses a known sentinel value as the `previous_checksum` (the SHA-256 of the partition's start timestamp). This makes the chain verifiable from a known anchor.

The `/api/v1/audit-log/verify` endpoint (permission: `admin.audit`) re-computes the checksum chain for a given date range and reports any row whose stored checksum does not match the computed value. A mismatch indicates that a row was modified or deleted after insertion, or that the chain was broken by inserting a row out of order.

The verification endpoint does not modify any data. It is a read-only integrity check.

---

## Green Note Integrity

Green notes use a separate hash chain from the audit ledger. Each note's `cryptographic_hash` is:

```
SHA-256(content_body || sequence_order::text || author_officer_id || previous_hash)
```

Where `previous_hash` is the `cryptographic_hash` of the preceding note in the same minute sheet (or a known sentinel for the first note). This chain cannot be broken without invalidating all subsequent hashes.

The `AppendGreenNote` service function validates the chain before inserting a new note: it fetches the last note in the minute sheet, verifies its hash matches the stored value, and rejects the insert if it does not. This means a corrupted chain is detected the moment someone attempts to append a note, not only during an explicit verification query.

Green notes are immutable after insert. There is no update or delete endpoint for individual notes.

Optional digital signatures: the `digital_signature` column stores a base64-encoded signature of the `content_body` using the author's private key (if the deployment enables signature support). Signature verification is out-of-scope for v1 but the column exists to support it.

---

## Secrets Policy

No secret of any kind is stored in YAML blueprint files. Blueprint files are committed to version control and must be treated as public.

Required environment variables:

| Variable | Purpose | Required |
|---|---|---|
| `AETHEL_DB_PASSWORD` | PostgreSQL password | Yes (if `connection_string_env` is empty) |
| `AETHEL_DB_DSN` | Full PostgreSQL DSN including password | Yes (if `connection_string_env` is set) |
| `AETHEL_JWT_SECRET` | Signing secret for HS256 JWT | Yes (if `jwt_algorithm: HS256`) |
| `AETHEL_JWT_RSA_PRIVATE_KEY` | PEM-encoded RSA private key for RS256 JWT | Yes (if `jwt_algorithm: RS256`) |
| `AETHEL_SMTP_PASSWORD` | SMTP relay password for email notifications | No |
| `AETHEL_ENV` | Active environment name | No (default: `development`) |

All of these are read exactly once at startup and stored in the in-process config. They are never logged, never included in error messages, and never passed to any function that could write them to a file or network socket.

---

## Optional Integrations

The following integrations are available but not required. They are configured in the blueprint, not as required infrastructure.

**SMTP (email notifications):** When `AETHEL_SMTP_PASSWORD` is set and SMTP is configured in `system_settings`, the notification service sends email on relevant events (dispatch assigned, escalation fired). If not configured, email notifications are silently skipped and in-app SSE/REST notifications are the only delivery channel.

**S3-compatible storage (attachments):** When an S3 endpoint and bucket are configured in `system_settings`, the attachment service uploads files to object storage and stores the resulting URL in `dispatch_attachments`. If not configured, the attachment service rejects uploads with a `501 Not Implemented` response. Attachment storage is disabled by default.

Neither integration affects the core functionality of the three domain pillars. The system operates fully without either.
