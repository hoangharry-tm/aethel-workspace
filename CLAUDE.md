# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Aethel Workspace is a configuration-driven e-office platform. Its defining architectural trait is **runtime-configurable with compile-time defaults**: two IT-facing YAML blueprints provide seed values loaded once at first boot; all runtime configuration (branding, navigation, features) is stored in PostgreSQL and edited through the `/admin/*` pages. The Go backend serves config via `GET /api/v1/config` with a single in-memory config cache (5-min TTL), embedded in the initial SSR HTML — zero client-side config round-trips.

The system is organized around three domain pillars:
1. **DAK Diarization** — inbound correspondence intake and tracking
2. **Green Noting Canvas** — institutional minute sheets and approval workflows
3. **RBAC Audit Ledger** — immutable security event log with tamper detection

## Repository Layout

```
aethel-view/        # Nuxt 4 frontend — UI/UX prototype COMPLETE (Phase 1 done)
aethel-core/        # Go backend — Sprint 0 foundation complete (Phase 2 in progress)
aethel-scripts/     # Build scripts — scaffolded, currently empty
blueprints/         # Declarative YAML configuration files (see Blueprint Status below)
  examples/         # Reference blueprints (do not edit — source of truth for schema)
build/bin/          # Compiled output
```

## Current Phase Status

**Phase 1 — UI/UX Prototype**: Complete as of 2026-05-25.
- 17 pages, 20 user stories, 3 roles (ADMIN / RECEPTION / USER), 0 TypeScript errors
- Figma design export complete: file key `aqW7snNu6m0RoD0ZXrMH0f` (5 pages: Design System, Login, Dashboard, Intake Form + Document Detail, Mobile + Admin)
- **Task 05 — Semantic token migration**: Complete as of 2026-05-29. All 30+ Vue files migrated from hardcoded palette classes to three-layer CSS variable system. Verification grep returns zero matches.

**Phase 2 — Go Backend** (`aethel-core/`): Architecture designed as of 2026-05-26; architectural pivot to runtime-configurable completed 2026-05-27. Database design complete: 42 migration SQL files written (migrations 1–20 original, migration 21 extends `branding_configs`); full backend architecture documented; config API designed (`GET /api/v1/config`, `PATCH /api/v1/admin/config/*`, in-memory cache); DevOps pipeline ready. **Sprint 0 complete** (2026-05-29): domain packages, cobra CLI, 50+ routes, 9-layer middleware, stub repositories — all scaffolded and building.

## Frontend — `aethel-view/`

**Stack**: Nuxt 4 (`compatibilityDate: "2025-07-15"`), Vue 3, Pinia, Nuxt UI v4 (`@nuxt/ui` 4.8.0 / Tailwind CSS v4), TypeScript

All frontend commands run from `aethel-view/`:

```bash
pnpm dev            # Dev server at http://localhost:3000
pnpm build          # Production build
pnpm preview        # Preview production build

pnpm test           # All Vitest tests (unit + nuxt)
pnpm test:unit      # Unit tests only (test/unit/*.spec.ts, Node env)
pnpm test:nuxt      # Nuxt component tests (test/nuxt/*.spec.ts, happy-dom env)
pnpm test:coverage  # Tests with V8 coverage

pnpm test:e2e       # Playwright E2E (tests/*.spec.ts, Chromium)
pnpm test:e2e:ui    # Playwright with interactive UI
```

No lint script defined; ESLint via `eslint.config.mjs` (extends `.nuxt/eslint.config.mjs`). Test projects are named `unit` and `nuxt` — use `--project unit` or `--project nuxt` to target one.

### Critical config files

`aethel-view/CLAUDE.md` (auto-managed by `@oro.ad/nuxt-claude-devtools`) documents a **mandatory check** before modifying `nuxt.config.ts`, `app.config.ts`, or any other Nuxt restart-triggering file. Always read `.claude-devtools/settings.json` first. **autoConfirm is currently DISABLED** — stop and ask the user before touching those files.

### Design System

- **Primary**: Indigo (`indigo-600` = `#4f46e5`) — set via `app.config.ts` `ui.colors.primary: 'indigo'`
- **Neutral**: Slate — set via `ui.colors.neutral: 'slate'`
- **Font**: Inter (Google Fonts, imported in `app/assets/css/main.css`)
- **Mode**: Light only
- **Icons**: Lucide via `i-lucide-*` Iconify notation (no emoji icons)
- **Urgency colors**: IMMEDIATE → rose, PRIORITY → amber, ROUTINE → emerald
- **Document status colors**: PENDING_ASSIGNMENT → neutral, UNDER_REVIEW → primary/indigo, IN_TRANSIT → sky, ATTEMPTED_DELIVERY → amber, DELIVERED → emerald, ESCALATED → rose, DISPATCHED → violet, REJECTED → red

### Dynamic Theming

All color styling uses a three-layer CSS variable system so the IT admin can change branding at runtime via `/admin/branding` without a rebuild.

**Layer 1 — `app/assets/css/main.css`**: semantic CSS variable defaults + `@theme` mapping to Tailwind utilities:
```css
:root {
  --color-text-body:   theme(colors.slate.800);
  --color-text-muted:  theme(colors.slate.500);
  --color-text-accent: theme(colors.indigo.600);
  --color-bg-surface:  theme(colors.white);
  --color-bg-subtle:   theme(colors.slate.50);
}
@theme {
  --color-body:   var(--color-text-body);
  --color-muted:  var(--color-text-muted);
  --color-accent: var(--color-text-accent);
}
```

**Layer 2 — `app/app.vue`**: `useHead()` overrides variables at runtime from `useAppRuntimeConfig()`:
- `--ui-primary` and `--color-text-accent` → `config.branding.primaryColor`
- `--ui-neutral` → `config.branding.neutralPalette`

**Layer 3 — Components**: use only semantic class names — never a Tailwind palette name directly:
- `text-body` / `text-muted` / `text-accent` instead of `text-slate-800` / `text-slate-500` / `text-indigo-600`
- `color="primary"` on Nuxt UI components (already wired to `--ui-primary`)

**Rule:** `main.css` is the single place mapping semantic names → palette colors. `app.vue` `useHead()` is the single runtime override point. No component may reference a palette name (`slate`, `indigo`, etc.) directly. **Migration complete (Task 05, 2026-05-29):** all 30+ Vue files have been migrated — `grep "text-slate\|text-indigo\|bg-slate\|bg-white\|border-slate\|border-indigo" app/` returns zero matches.

### Page Structure

All pages use either `layout: 'workspace'` or `layout: 'auth'`. The workspace layout (`app/layouts/workspace.vue`) is a fixed `h-dvh` shell: `WorkspaceSidebar` (left) + topbar `WorkspaceNavbar` + scrollable `<main>` + `NotificationDrawer` (right portal).

| Route | File | Roles |
|---|---|---|
| `/` | `pages/index.vue` | all (redirects to `/dashboard`) |
| `/auth/login` | `pages/auth/login.vue` | public |
| `/dashboard` | `pages/dashboard.vue` | RECEPTION |
| `/dispatch/inbound` | `pages/dispatch/inbound/index.vue` | RECEPTION |
| `/dispatch/inbound/new` | `pages/dispatch/inbound/new.vue` | RECEPTION |
| `/dispatch/outbound` | `pages/dispatch/outbound/index.vue` | RECEPTION |
| `/documents/[id]` | `pages/documents/[id].vue` | all |
| `/my-documents` | `pages/my-documents.vue` | USER |
| `/outgoing/new` | `pages/outgoing/new.vue` | USER |
| `/search` | `pages/search.vue` | all |
| `/admin/users` | `pages/admin/users.vue` | ADMIN |
| `/admin/routing-rules` | `pages/admin/routing-rules.vue` | ADMIN |
| `/admin/document-types` | `pages/admin/document-types.vue` | ADMIN |
| `/admin/escalation` | `pages/admin/escalation.vue` | ADMIN |
| `/admin/audit-log` | `pages/admin/audit-log.vue` | ADMIN (sys_admin) |
| `/admin/reports` | `pages/admin/reports.vue` | ADMIN |
| `/admin/settings` | `pages/admin/settings.vue` | ADMIN — org profile, feature toggles, DB alias display |
| `/admin/branding` | `pages/admin/branding.vue` | ADMIN — live color picker, font selector, logo upload, preview panel |
| `/admin/navigation` | `pages/admin/navigation.vue` | ADMIN — nav tree editor (reorder, visibility, rename, add item) |

### Components

**Layout** (`app/components/layout/`):
- `WorkspaceSidebar.vue` — collapsible (`w-64`/`w-16`), role-gated nav groups, active state with `bg-accent/5` bg + 3px left accent bar; mobile via `USlideover` + `useSidebarDrawer`
- `WorkspaceNavbar.vue` — 56px topbar; profile dropdown with role switcher (ADMIN / RECEPTION / USER) for prototype demo; notification bell opens `useNotificationDrawer`
- `NotificationDrawer.vue` — `USlideover` from right, w-96; unread items styled with `border-l-2 border-accent bg-accent/5`

**Shared** (`app/components/shared/`):
- `UrgencyBadge.vue` — `UBadge` wrapping IMMEDIATE/PRIORITY/ROUTINE with color/icon from blueprint
- `DocumentStatusBadge.vue` — `UBadge` for all 7 status states
- `EventTimeline.vue` — vertical step list with colored dot + connector line; read-only audit trail

**Blocks** (`app/components/blocks/`): — self-contained cards for admin custom pages
- `BlockStatCard.vue` — KPI card with icon, large value, emerald/rose trend badge
- `BlockDataTable.vue` — sortable table card with empty state
- `BlockFormBuilder.vue` — 2-col form grid with field type dispatch (text/select/date/textarea)
- `BlockTimeline.vue` — vertical connector timeline (same dot pattern as EventTimeline)
- `BlockRichText.vue` — prose content card; if `editable`, shows textarea + save button
- `BlockQuickActions.vue` — horizontal flex of outline UButtons with icon+label

### Composables

- `useMockData()` — `useState`-backed shared state; exports `currentUser` (switchable via `setRole(role)`), `documents` (10 records), `notifications` (5), `routingRules` (5), `users` (8). Default user on load: Marcus Webb (RECEPTION). Role map: ADMIN → Alice Thornton, RECEPTION → Marcus Webb, USER → Priya Sharma.
- `useNotificationDrawer()` — `useState('notif-drawer')` boolean; exposes `open()`, `close()`, `toggle()`
- `useSidebarDrawer()` — same pattern for mobile sidebar
- `useAppRuntimeConfig()` — `useState<AppRuntimeConfig>('app-runtime-config', ...)` SSR-safe config; exports `config`, `isLoading`, `refresh()`, `updateBranding(partial)`, `updateOrg(partial)`, `updateFeatures(partial)`, `updateNav(groups)`. Shape matches planned `GET /api/v1/config`. WorkspaceSidebar reads `config.nav` with hardcoded fallback.

## Blueprint System

Only two files are IT-admin facing. All other configuration is managed via the `/admin/*` pages and stored in PostgreSQL.

| File | Status |
|---|---|
| `blueprints/ui-theme.yaml` | **SEED** — branding seed (5 fields: primary_color, neutral_palette, font_family, wordmark, logo_path); runtime config in `branding_configs` table |
| `blueprints/ui-components.yaml` | **BLOCK REGISTRY** — 6 block type definitions for admin page builder; IT uses `/admin/navigation`, not this file |
| `blueprints/ui-layouts.yaml` | **NAV SEED** — nav tree for first-boot seeding; runtime nav in `system_settings` key `nav_config` |
| `blueprints/server-database.yaml` | **FILLED** — unchanged; the only IT-admin facing YAML file for DB connection + naming. JSON Schema at `blueprints/schemas/server-database.schema.json`. |
| `server-queries.yaml` | **MOVED** — now at `aethel-core/internal/database/queries/queries.yaml`; internal developer file, not IT-facing |
| `server-routes.yaml` | **DELETED** — routes are convention-driven, not IT-configurable |

`blueprints/examples/` holds reference schemas — do not modify; they are the canonical source for future blueprint authors.

`blueprints/schemas/` holds the JSON Schema file for `server-database.yaml` editor validation.

## Go Backend — `aethel-core/`

**Stack**: Go (to be scaffolded), PostgreSQL 16+

### Database layout

```
aethel-core/
└── internal/
    ├── config/                   # (Sprint 2) in-memory config cache + API handlers
    │   ├── cache.go              # ConfigCache: single config struct, 5-min TTL
    │   ├── loader.go             # LoadOrgConfig: queries branding_configs + system_settings
    │   └── handler.go            # GET /api/v1/config, PATCH /api/v1/admin/config/*
    └── database/
        ├── migrator.go           # (to be written) blueprint-rendered migration runner
        ├── blueprint_context.go  # (to be written) T(), E(), Schema template helpers
        ├── queries/
        │   └── queries.yaml      # named SQL queries (internal developer file, not IT-facing)
        └── migrations/           # 42 SQL files (21 up + 21 down) — WRITTEN ✓
```

### Migration system

Migrations use Go `text/template` syntax for customisable identifiers:
- `{{ .Schema }}` → resolves to `schema.default_schema` from the blueprint
- `{{ T "tablename" }}` → resolves canonical name through `schema.name_aliases`
- `{{ E "enumname" }}` → resolves canonical name through `schema.enum_aliases`

Run from `aethel-core/` (commands pending Go CLI scaffold):
```bash
aethel migrate up          # apply all pending migrations
aethel migrate status      # list applied / pending
aethel migrate validate    # dry-run: render templates, check SQL syntax
aethel migrate down --steps 1   # roll back last migration
```

### Database schema (20 tables + migration 21, 3 pillars)

ER diagram: `docs/db-design.mmd` — open with any Mermaid renderer.

| Migration | Tables / Objects |
|---|---|
| 01 | Extensions: uuid-ossp, pgcrypto, pg_trgm |
| 02 | `organizations` (tenant root) |
| 03 | `departments` (self-referencing hierarchy) |
| 04 | `users`, `user_sessions`, `password_reset_tokens`, `notification_preferences`; enum: `user_role` |
| 05 | `document_types` |
| 06 | `dispatches`; enums: `priority_level`, `dispatch_status` |
| 07 | `dispatch_attachments` |
| 08 | `dispatch_events` (unified timeline log for US-14) |
| 09 | `routing_rules` |
| 10 | `routing_rule_conditions` |
| 11 | `routing_rule_destinations` |
| 12 | `minute_sheets` (Pillar 2) |
| 13 | `green_notes` (Pillar 2 — cryptographic chain) |
| 14 | `notifications` |
| 15 | `escalation_rules` |
| 16 | `system_settings` (key-value store; holds `nav_config` JSON after first boot) |
| 17 | `branding_configs` |
| 18 | `audit_ledger` (Pillar 3 — PARTITION BY RANGE monthly) |
| 19 | Pre-provisioned audit_ledger monthly partitions (12 back, 3 ahead) |
| 20 | `set_updated_at()` function + triggers on all `updated_at` tables |
| 21 | ALTER `branding_configs`: ADD `neutral_palette` varchar(20) default 'slate', `font_family` varchar(100) default 'Inter', `wordmark` varchar(200) default 'Aethel Workspace' (supports runtime branding editor) |

### Key schema decisions

- **Single-tenant self-hosted**: Aethel is not a SaaS product. Each organization downloads, configures, and hosts their own instance. The `organizations` table holds exactly one row — the installation's own org profile. `organization_id` on other tables is a fixed boot-time constant (loaded from that one row at startup), not a per-request routing key. No TenantResolver middleware. No RLS. No multi-tenant query scoping.
- **Audit ledger**: `bigserial` PK (not UUID) for partition performance; `organization_id` is a plain `uuid` (not FK) so records survive org deletion; `previous_checksum` chains rows for tamper detection.
- **Green notes**: Immutable after insert — no `updated_at`. `cryptographic_hash = SHA-256(content || sequence || author)`, `previous_hash` links to prior note.
- **Dispatch events**: Single `dispatch_events` table aggregates all timeline events (routing, handoff, escalation) — avoids multi-table UNIONs in the timeline view.

### Reference docs

| Document | Location |
|---|---|
| Docs conventions | `docs/CONVENTIONS.md` |
| ER diagram | `docs/db-design.mmd` |
| Blueprint YAML conventions | `docs/guides/server-blueprint-conventions.md` |
| Migration system design | `docs/plans/migration-strategy.md` |
| IT customisation guide | `docs/guides/it-customization-guide.md` |
| Go developer guide | `docs/guides/go-developer-guide.md` |
| Code architecture | `docs/architecture/architecture-code.md` |
| Server architecture | `docs/architecture/architecture-server.md` |
| API routes + config API | `docs/architecture/architecture-api-routes.md` |
| Security architecture | `docs/architecture/architecture-security.md` |
| Agile implementation plan | `docs/plans/agile-implementation-plan.md` |
| DevOps tooling recommendations | `docs/devops/devops-tooling.md` |
| Runtime config flow diagram | `docs/diagrams/runtime-config-flow.mmd` |

### DevOps layout

```
aethel-workspace/
├── Makefile                   # all dev/build/test/deploy commands (make help)
├── docker-compose.yml         # local dev: postgres + backend + frontend
├── docker-compose.prod.yml    # production overrides
├── .env.example               # copy to .env for local development
├── aethel-view/Dockerfile     # 2-stage Nuxt production image
├── aethel-core/Dockerfile     # 2-stage Go distroless production image
├── k8s/                       # Kubernetes manifests (namespace: aethel-workspace)
│   ├── postgres/              # StatefulSet + PVC + Service + Secret placeholder
│   ├── backend/               # Deployment + HPA + ConfigMap + Secret placeholder
│   ├── frontend/              # Deployment + HPA + Service
│   └── ingress.yaml           # nginx ingress: /api → backend, / → frontend
├── .github/workflows/
│   ├── ci.yml                 # PR/push: test-backend + test-frontend + lint-yaml (parallel)
│   ├── cd.yml                 # merge to main: build + push GHCR → deploy staging
│   └── security.yml           # weekly: Trivy + govulncheck + gosec → GitHub Security
└── aethel-scripts/
    ├── setup-dev.sh           # first-time dev environment setup
    ├── health-check.sh        # verify all services are running
    ├── rotate-jwt-secret.sh   # generate + rotation checklist (never writes secret to disk)
    ├── db-backup.sh           # pg_dump with gzip + optional S3 upload
    └── k8s-rollout.sh         # migrate-then-rollout production deploy coordinator
```

## Key Domain Concepts

- **Dispatch** (`dispatches` table) — a tracked inbound/outbound correspondence item with `priority_level` (`ROUTINE` / `PRIORITY` / `IMMEDIATE`) and `status_state`
- **Minute Sheet + Green Notes** — each dispatch has one minute sheet; green notes are appended sequentially with cryptographic hashes and optional digital signatures
- **Audit Ledger** — append-only, monthly range-partitioned table; `sys_admin` role required to view; tamper events include `SECURITY_BREACH_ATTEMPT`, `UNAUTHORIZED_ACCESS_BYPASSED`, `RBAC_ELEVATION_ATTEMPT`
- **RBAC** — permissions are hierarchical (`dispatch.view`, `workflow.approve`, `archive.view`); some routes additionally check `required_role: "sys_admin"`

## Nuxt Configuration Notes

- `compatibilityDate: "2025-07-15"` — uses Nuxt 4 behavior
- Active modules: `@nuxt/ui`, `@pinia/nuxt`, `@nuxt/eslint`, `@nuxt/image`, `@nuxt/a11y`, `@nuxt/scripts`, `@nuxt/test-utils`, `@nuxtjs/mcp-toolkit`, `@oro.ad/nuxt-claude-devtools`
- `app.config.ts` sets `ui.colors.primary = 'indigo'` and `ui.colors.neutral = 'slate'`
- `nuxt.config.ts` adds `css: ['~/assets/css/main.css']` for Inter font import
