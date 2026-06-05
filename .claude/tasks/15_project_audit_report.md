# Task 15 — Full-Project Audit & Progress Report

**Purpose:** Scan every file in the repository, check against defined criteria, and produce a comprehensive HTML audit report. No application code is written or modified in this task — read-only audit only. If critical issues are found, a Task 16 fix file is generated as a follow-up.

**Primary output:** `docs/reports/audit-$(date +%Y-%m-%d).html` — a beautiful, self-contained HTML report
**Secondary output (conditional):** `.claude/tasks/16_critical_fixes.md` — generated only if ❌ critical findings exist

---

## ⚠️ Important: This Task Is State-Aware

Tasks 11, 12, 13, and 14 have been executed before this task runs. Baseline file counts and package lists below reflect the expected post-Tasks-11–14 state. Every agent **must discover the actual current state** from the filesystem and git history — never trust the hardcoded lists below as exhaustive. Use them as a minimum baseline; any files found beyond the list must also be audited.

---

## Step 0 — Pre-flight Sync (Run This First, Before Spawning Any Agent)

Before spawning any parallel agent, the orchestrating session must run the following and write the output to `/tmp/audit-preflight.md`. Every agent will read this file at the start of their work.

```bash
REPO=/Users/hoangharry/mh_code/internships/Bravo/aethel-workspace
cd "$REPO"

# 1. Pull the latest state from the remote
git pull origin dev

# 2. Capture recent commit history (last 30 commits)
echo "## Recent Commits" > /tmp/audit-preflight.md
git log --oneline -30 >> /tmp/audit-preflight.md

# 3. Capture all files changed in the last 30 commits
echo -e "\n## Files Changed in Last 30 Commits" >> /tmp/audit-preflight.md
git diff HEAD~30..HEAD --stat 2>/dev/null >> /tmp/audit-preflight.md || git diff $(git rev-list --max-parents=0 HEAD)..HEAD --stat >> /tmp/audit-preflight.md

# 4. Capture all new files added (untracked + recently committed)
echo -e "\n## All Go Files in aethel-core (current)" >> /tmp/audit-preflight.md
find aethel-core -name "*.go" | sort >> /tmp/audit-preflight.md

echo -e "\n## All Vue/TS Files in aethel-view (current)" >> /tmp/audit-preflight.md
find aethel-view/app -name "*.vue" -o -name "*.ts" | sort >> /tmp/audit-preflight.md

echo -e "\n## All Task Files" >> /tmp/audit-preflight.md
ls .claude/tasks/ >> /tmp/audit-preflight.md

echo -e "\n## Git Status" >> /tmp/audit-preflight.md
git status --short >> /tmp/audit-preflight.md

echo -e "\n## Git HEAD" >> /tmp/audit-preflight.md
git rev-parse --short HEAD >> /tmp/audit-preflight.md

echo "Pre-flight complete. Context written to /tmp/audit-preflight.md"
cat /tmp/audit-preflight.md
```

Only after this completes successfully, spawn Agents 1–5 in parallel.

---

## How This Task Runs

```
[Step 0]  Pre-flight sync → /tmp/audit-preflight.md
              ↓
[Parallel] Agent 1 — UI/UX + Frontend Auditor     → aethel-view/ (code + design + a11y)
           Agent 2 — Go Backend Auditor            → aethel-core/ (progress + code quality)
           Agent 3 — Database Auditor              → migrations + schema + queries
           Agent 4 — Cybersecurity Auditor         → auth, middleware, secrets, headers, FE→BE
           Agent 5 — DevOps Auditor                → Docker, K8s, CI/CD, scripts
              ↓ all complete
[Serial]   Agent 6 — Synthesis Agent              → merge + score + beautiful HTML report
              ↓
[Conditional] Agent 7 — Task 16 Writer            → only if ❌ critical findings exist
```

Each agent writes its findings to `/tmp/audit-agent-N.md` (N = 1..5).
Agent 6 reads all five and writes `docs/reports/audit-YYYY-MM-DD.html`.
Agent 7 (if needed) reads Agent 6's HTML and writes `.claude/tasks/16_critical_fixes.md`.

---

## Audit Criteria Reference (All Agents Must Apply)

### Completion Scoring

| Score | Meaning |
|-------|---------|
| ✅ Done | Fully implemented, tested, no known gaps |
| 🔄 Partial | Core logic exists but missing tests, edge cases, or wiring |
| ⚠️ Stub | File exists with placeholder / "under construction" content only |
| ❌ Missing | File or feature expected but not present at all |
| 🚫 Blocked | Cannot complete without a prerequisite that is not done |

### Quality Scoring (for design + code quality checks)

| Score | Meaning |
|-------|---------|
| ⭐⭐⭐ Excellent | Professional, idiomatic, no obvious improvements |
| ⭐⭐ Good | Solid, minor style or pattern gaps |
| ⭐ Needs Work | Functional but significant quality issues |
| 💀 Poor | Breaks conventions, hard to maintain, needs rework |

### Percentage formula

```
completion % = (Done×1.0 + Partial×0.5 + Stub×0.1) / total_items × 100
```

Apply per section and for the overall score. `total_items` = everything you actually find on disk, not the baseline list in this file.

---

## Agent 1 — UI/UX Designer + Frontend Developer Auditor

**Scope:** `aethel-view/` — all pages, components, composables, plugins, middleware, assets

**You are a senior NuxtJS 4 + Vue 3 + TypeScript engineer AND a UI/UX designer with expertise in accessibility, design systems, and user experience quality. You will audit both technical correctness AND visual/UX quality.**

### Step 1: Read pre-flight context

```bash
cat /tmp/audit-preflight.md
```

Note every `.vue` and `.ts` file listed under "All Vue/TS Files in aethel-view (current)" — these are the actual files that exist right now. Audit all of them, not just the baseline list below.

### Step 2: Discover all pages

```bash
find aethel-view/app/pages -name "*.vue" | sort
```

For **every file found** (not just the baseline), report:

| Column | What to record |
|--------|----------------|
| File path | Relative to `aethel-view/` |
| Line count | `wc -l` |
| Status | ✅ / 🔄 / ⚠️ / ❌ using completion scoring above |
| definePageMeta | Correct layout + middleware set? |
| Data source | Mock (`useMockData`) / Real (`$fetch`) / Both / None |
| Semantic CSS | Zero palette violations? |
| Notes | Any obvious issues |

**Baseline pages** (written at task-author time — treat as minimum; audit any additional pages found):
```
pages/index.vue                       pages/search.vue
pages/auth/login.vue                  pages/admin/users.vue
pages/dashboard.vue                   pages/admin/routing-rules.vue
pages/dispatch/inbound/index.vue      pages/admin/document-types.vue
pages/dispatch/inbound/new.vue        pages/admin/escalation.vue
pages/dispatch/outbound/index.vue     pages/admin/audit-log.vue
pages/documents/[id].vue              pages/admin/reports.vue
pages/my-documents.vue                pages/admin/settings.vue
pages/outgoing/new.vue                pages/admin/branding.vue
                                      pages/admin/navigation.vue
```

For any page **not in this baseline** that you find on disk, mark it as a bonus and audit it with the same criteria.

### Step 3: Discover all components

```bash
find aethel-view/app/components -name "*.vue" | sort
```

For each component found, check:
- Hardcodes palette classes? (`grep -l "text-slate\|text-indigo\|bg-white\|bg-slate\|text-gray\|text-zinc"`)
- Uses raw `<button>` or `<table>` instead of Nuxt UI primitives?

### Step 4: Composables, plugins, middleware

```bash
find aethel-view/app/composables aethel-view/app/plugins aethel-view/app/middleware -type f | sort
```

For each file found, determine its purpose and check:

**Composables minimum expected** (audit any additional ones found too):
- `useAuth.ts` — access token in `useState` not `localStorage`?
- `useRuntimeConfig.ts` — calls real `$fetch('/api/v1/config')`?
- `useMockData.ts` — prototype data layer, expected to exist

**Plugins:**
- `auth.client.ts` — silent token recovery on page load?

**Middleware:**
- `auth.ts` — redirects unauthenticated?
- `role.ts` — gates by `requiredRole`?

### Step 5: Design system compliance

Run from `aethel-view/`:

```bash
# Must be 0 — palette violations
echo "=== PALETTE VIOLATIONS ===" && grep -rn "text-slate\|text-indigo\|bg-white\|bg-slate\|text-gray\|text-zinc" app/pages/ app/components/ | wc -l
grep -rn "text-slate\|text-indigo\|bg-white\|bg-slate\|text-gray\|text-zinc" app/pages/ app/components/

# Must be > 0 — semantic tokens are being used
echo "=== SEMANTIC TOKEN USAGE ===" && grep -rn "text-body\|text-muted\|text-accent\|bg-surface\|bg-subtle" app/pages/ app/components/ | wc -l

# Must be 0 — localStorage forbidden for auth tokens
echo "=== LOCALSTORAGE USAGE ===" && grep -rn "localStorage" app/ | wc -l
grep -rn "localStorage" app/
```

### Step 6: Build check

```bash
cd aethel-view && pnpm build 2>&1 | tail -30
```

### Step 7: UI/UX Design Quality & Accessibility Audit

This step evaluates the **visual quality, design logic, and accessibility** of the frontend. Read every page and component file; do not rely on running a browser.

#### 7a — Accessibility (a11y)

```bash
# Check for ARIA labels on interactive elements
grep -rn "aria-label\|aria-labelledby\|aria-describedby\|role=" aethel-view/app/pages/ aethel-view/app/components/ | wc -l

# Check for alt text on images
grep -rn "<img\|<NuxtImg\|<UAvatar" aethel-view/app/pages/ aethel-view/app/components/ | grep -v "alt=" | head -20

# Check form inputs have associated labels
grep -rn "UFormField\|UInput\|UTextarea\|USelect" aethel-view/app/pages/ | wc -l
grep -rn "UFormField" aethel-view/app/pages/ | wc -l

# Check loading states exist (screen-reader friendly)
grep -rn "isLoading\|v-if.*loading\|UButton.*loading" aethel-view/app/pages/ | wc -l

# Check keyboard navigation hints (tab trapping in modals)
grep -rn "UModal\|USlideover\|UDrawer" aethel-view/app/pages/ aethel-view/app/components/ | wc -l
```

For each page, assess:
- Are all interactive elements keyboard-reachable? (buttons, links, inputs)
- Do modals/slidecovers trap focus correctly? (Nuxt UI handles this natively — verify UModal is used, not custom `<div>` popups)
- Are form fields wrapped in `<UFormField>` with a `label` prop?
- Are error states communicated (not just by color alone)?
- Is there sufficient color contrast? (text-body on bg-surface is slate-800 on white — verify)

Rate overall accessibility: ⭐⭐⭐ / ⭐⭐ / ⭐ / 💀

#### 7b — Design Consistency

Read the following key pages and assess visual consistency:

```bash
cat aethel-view/app/pages/dashboard.vue
cat aethel-view/app/pages/dispatch/inbound/index.vue
cat aethel-view/app/pages/admin/users.vue
cat aethel-view/app/pages/admin/audit-log.vue
cat aethel-view/app/pages/admin/reports.vue
cat aethel-view/app/pages/admin/document-types.vue
cat aethel-view/app/pages/admin/escalation.vue
```

Check for:
- **Consistent header pattern**: Does every page follow `<div class="space-y-6">` → `<div class="flex items-center justify-between">` header? Or are there outliers?
- **Consistent card pattern**: Is `<UCard>` used uniformly, or do some pages use raw `<div>` with shadow classes?
- **Action button placement**: Is the primary action button always top-right of the header?
- **Empty state quality**: When tables or lists have no data, is there a friendly empty state or just nothing?
- **Typography hierarchy**: Does heading use `text-xl font-bold text-body`? Do subtitles use `text-sm text-muted`?
- **Icon consistency**: Are all icons from `i-lucide-*` family, or are there mixed icon sets?

Rate design consistency: ⭐⭐⭐ / ⭐⭐ / ⭐ / 💀

#### 7c — UX Flow & Logic

Evaluate the logical coherence of user flows:

- **Login → Dashboard flow**: After login, does the user land on the correct page for their role? (RECEPTION → /dashboard, ADMIN → /dashboard, USER → /my-documents or /dashboard)
- **Navigation structure**: Does the sidebar nav logically group related pages? Are ADMIN-only items hidden from RECEPTION/USER?
- **Data mutation feedback**: After creating/editing/deleting items, does the user get a toast notification?
- **Confirmation dialogs**: Are destructive actions (delete) confirmed with a modal?
- **Back navigation**: On detail pages (documents/[id].vue), is there a clear path back?
- **Form validation**: Do forms prevent empty submission? Is the UX clear about required fields?

Rate UX flow quality: ⭐⭐⭐ / ⭐⭐ / ⭐ / 💀

#### 7d — Mobile Responsiveness

```bash
# Check for responsive grid patterns
grep -rn "md:grid-cols\|lg:grid-cols\|sm:flex-row\|md:w-\|lg:hidden\|md:hidden" aethel-view/app/pages/ | wc -l

# Check for mobile sidebar trigger (useSidebarDrawer)
grep -rn "useSidebarDrawer\|sidebar.*drawer\|mobile.*sidebar" aethel-view/app/ -r | wc -l
```

Rate mobile responsiveness: ⭐⭐⭐ / ⭐⭐ / ⭐ / 💀

#### 7e — Aesthetic Quality (Overall Visual Polish)

After reading through the key pages and components, give an honest professional assessment:

- Does the color palette feel cohesive and professional?
- Is there appropriate whitespace between sections?
- Are status badges, urgency indicators, and document status colors used correctly and consistently?
- Do admin pages feel like production-quality software or like prototype scaffolding?
- Is the login page polished? (First impression matters)

Rate overall aesthetic quality: ⭐⭐⭐ / ⭐⭐ / ⭐ / 💀

Write 3–5 sentences of qualitative commentary on the overall design quality for the HTML report.

### Output format — write to `/tmp/audit-agent-1.md`

```markdown
## Frontend Audit (aethel-view/)
_Audited at: [timestamp]_
_Files discovered: N .vue files, M .ts files_

### Pages Summary
| Page | Lines | Status | Data Source | Semantic CSS | Notes |
|------|-------|--------|-------------|--------------|-------|

**Total pages found: N (baseline expected: 19)**
**Completion: X Done + Y Partial + Z Stubs = XX%**

### Components Found
- Total: N components
- Palette violations: N files → [list]
- Raw HTML interactive elements: N files → [list]

### Composables / Plugins / Middleware
| File | Found | Purpose | Correct impl | Notes |
|------|-------|---------|-------------|-------|

### Design System Compliance
- Palette violations: N matches (must be 0)
- Semantic token usage: N matches (healthy if > 0)
- localStorage usage: N matches (must be 0)

### Build Status
- pnpm build: ✅ Success / ❌ Failed
- Error (if any): ...

### UI/UX Design Quality
| Category | Rating | Notes |
|----------|--------|-------|
| Accessibility (a11y) | ⭐⭐⭐/⭐⭐/⭐/💀 | [summary] |
| Design Consistency | ⭐⭐⭐/⭐⭐/⭐/💀 | [summary] |
| UX Flow & Logic | ⭐⭐⭐/⭐⭐/⭐/💀 | [summary] |
| Mobile Responsiveness | ⭐⭐⭐/⭐⭐/⭐/💀 | [summary] |
| Aesthetic Quality | ⭐⭐⭐/⭐⭐/⭐/💀 | [summary] |

**Qualitative design commentary:** [3–5 sentences of honest professional assessment]

**Things that are Good:** [list specific patterns/pages/components that are well done]
**Things to Fix:** [specific actionable issues with file:line references]

### Frontend Completion Score: XX%
### UI/UX Quality Score: XX% (star ratings converted: ⭐⭐⭐=100%, ⭐⭐=67%, ⭐=33%, 💀=0%)
```

---

## Agent 2 — Go Backend Auditor

**Scope:** `aethel-core/` — all `.go` files, `go.mod`, `go.sum`

**You are a senior Go engineer familiar with chi, zerolog, Argon2id, JWT, and PostgreSQL, AND a code quality assessor who evaluates idiomatic Go, error handling, interface design, and maintainability.**

### Step 1: Read pre-flight context

```bash
cat /tmp/audit-preflight.md
```

Note every `.go` file listed under "All Go Files in aethel-core (current)" — audit all of them.

### Step 2: Build and vet

```bash
cd aethel-core
go build ./... 2>&1
go vet ./... 2>&1
```

### Step 3: Discover all packages

```bash
find aethel-core -type d | sort
find aethel-core -name "*.go" | sort
```

For every package directory found, report its status. **Baseline packages** (minimum expected after Tasks 10–14; audit any additional ones found):

```
cmd/aethel/               internal/audit/           internal/api/docs/
internal/app/             internal/blueprint/       internal/rbac/
internal/config/          internal/database/        internal/service/
internal/domain/          internal/api/             internal/worker/
internal/database/repos/  internal/api/handlers/    internal/transport/
internal/database/queries/
```

### Step 4: Repository implementations

```bash
find aethel-core/internal/database/repos -name "*.go" | sort
```

For **every repo file found** (not just the baseline below), check:
- Real implementation or noop/stub? (look for actual SQL calls vs `return nil`)
- Uses `qr.Get("group.name")`? (zero inline SQL strings?)
- References `app.OrgID` instead of orgID method param?

**Baseline repos** (minimum expected after Tasks 10–14):
```
user_repo.go            dispatch_repo.go        minute_sheet_repo.go
session_repo.go         dispatch_event_repo.go  green_note_repo.go
password_reset_repo.go  routing_rule_repo.go    escalation_rule_repo.go
audit_repo.go
```

### Step 5: Service layer

```bash
find aethel-core/internal/service -name "*.go" | sort
```

For each service file found:
- Injects `audit.Writer` (not `domain.AuditRepository` directly)?
- Methods with empty bodies or `// TODO`?

### Step 6: Route coverage

Read `docs/architecture/architecture-api-routes.md` to get the **authoritative** list of expected routes — do not rely on any number written in this task file, as routes may have been added by Tasks 11–14. Then count routes actually registered in `internal/api/server.go`. Report: registered vs expected, list any missing.

### Step 7: Test results

```bash
cd aethel-core
go test ./... 2>&1 | grep -E "^ok|FAIL|no test"
go test ./internal/service/... -v 2>&1 | grep -E "^=== RUN|--- PASS|--- FAIL|--- SKIP"
go test ./internal/database/repos/... -v 2>&1 | grep -E "^=== RUN|--- PASS|--- FAIL|--- SKIP" 2>/dev/null
```

### Step 8: Violation greps

```bash
cd aethel-core

# Must be 0 — no inline SQL in repos or services
echo "=== INLINE SQL ===" && grep -rn "\"SELECT\|\"INSERT\|\"UPDATE\|\"DELETE" internal/database/repos/ internal/service/ | wc -l
grep -rn "\"SELECT\|\"INSERT\|\"UPDATE\|\"DELETE" internal/database/repos/ internal/service/

# Must be 0 — no orgID threaded through methods
echo "=== ORGID PARAMS ===" && grep -rn "orgID\b" internal/service/ internal/database/repos/ | wc -l

# Must be 0 — no org claim in JWT
echo "=== ORG JWT CLAIM ===" && grep -rn "org.*claim\|claims\[.org.\]\|\"org\"" internal/ | wc -l

# Check for noops (unfilled stubs)
echo "=== NOOP REPOS ===" && grep -rn "return nil, nil\|// noop\|// TODO" internal/database/repos/ | wc -l
grep -rn "return nil, nil\|// noop\|// TODO" internal/database/repos/
```

### Step 9: Go Code Quality Assessment

This step evaluates the **idiomatic quality, maintainability, and engineering discipline** of the Go codebase.

#### 9a — Error Handling Quality

```bash
cd aethel-core

# Check for proper error wrapping (fmt.Errorf with %w)
echo "=== PROPER ERROR WRAPPING ===" && grep -rn 'fmt\.Errorf.*%w' internal/ | wc -l

# Check for bare error returns without wrapping (anti-pattern)
echo "=== BARE ERRORS (potential quality gap) ===" && grep -rn 'return err$\|return nil, err$' internal/ | wc -l

# Check for panic usage (should be minimal/none in production code)
echo "=== PANIC USAGE ===" && grep -rn '\bpanic(' internal/ | grep -v "_test.go" | wc -l
grep -rn '\bpanic(' internal/ | grep -v "_test.go"
```

Rate error handling quality: ⭐⭐⭐ / ⭐⭐ / ⭐ / 💀

#### 9b — Context Propagation

```bash
cd aethel-core

# Check that ctx is the first parameter on all repo/service methods
grep -rn 'func.*ctx context\.Context' internal/database/repos/ internal/service/ | wc -l

# Check for context.Background() being created inside business logic (anti-pattern)
grep -rn 'context\.Background()' internal/service/ internal/database/repos/ internal/api/handlers/ | wc -l
grep -rn 'context\.Background()' internal/service/ internal/database/repos/
```

Rate context propagation: ⭐⭐⭐ / ⭐⭐ / ⭐ / 💀

#### 9c — Interface Design

```bash
cd aethel-core

# List all domain interfaces
grep -rn 'type.*interface {' internal/domain/ internal/audit/ | sort
wc -l < <(grep -rn 'type.*interface {' internal/domain/ internal/audit/)

# Check interface sizes — count methods per interface
for iface in $(grep -rn 'type.*interface {' internal/domain/ | grep -oP 'type \K\w+'); do
  echo "$iface: $(grep -A 50 "type $iface interface" internal/domain/*.go 2>/dev/null | grep -c '^\s\+[A-Z]')"
done
```

Assess: Are any interfaces too large (>7 methods)? Are they segregated by responsibility? Does each repo implement exactly one domain interface?

Rate interface design: ⭐⭐⭐ / ⭐⭐ / ⭐ / 💀

#### 9d — Naming Conventions & Idiomatic Go

```bash
cd aethel-core

# Check for un-idiomatic names (Go uses camelCase not snake_case for identifiers)
grep -rn 'func.*_[a-z]' internal/ | grep -v "_test.go" | grep -v "\.go:" | head -20

# Check for exported types with redundant package prefix (anti-pattern: service.ServiceConfig)
grep -rn 'type Service.*struct\|type Config.*struct' internal/service/ internal/config/

# Check package comments (every package should have a doc comment)
for pkg in $(find internal -name "*.go" | grep -v "_test" | head -30); do
  dir=$(dirname "$pkg")
  has_doc=$(head -5 "$pkg" | grep -c "^// Package" || true)
  [ "$has_doc" -eq "0" ] && echo "MISSING package comment: $pkg"
done | head -20
```

Rate naming conventions: ⭐⭐⭐ / ⭐⭐ / ⭐ / 💀

#### 9e — Overall Code Quality Commentary

After reviewing the codebase structure, write 3–5 sentences assessing:
- Is the code organized in a way a new Go developer could navigate easily?
- Is the separation of concerns (domain → repo → service → handler) consistently followed?
- Are there any architectural smells (e.g., handlers doing business logic, repos doing HTTP things)?
- What is the overall engineering discipline level of the codebase?

Rate overall Go code quality: ⭐⭐⭐ / ⭐⭐ / ⭐ / 💀

### Output format — write to `/tmp/audit-agent-2.md`

```markdown
## Backend Audit (aethel-core/)
_Audited at: [timestamp]_
_Go files discovered: N_

### Build Status
- go build ./...: ✅ / ❌ (error: ...)
- go vet ./...: ✅ / ❌

### Package Inventory
| Package | Status | Key files | Notes |
|---------|--------|-----------|-------|

### Repository Layer
| Repo File | Exists | Real impl | Zero inline SQL | Uses app.OrgID | Notes |
|-----------|--------|-----------|-----------------|----------------|-------|

**Repos: X/N real, Y stubs/noops**

### Service Layer
| Service | Exists | audit.Writer | Methods complete | TODOs found |
|---------|--------|-------------|-----------------|-------------|

### Route Coverage
- Routes in architecture doc: N (read from the doc)
- Routes registered in server.go: N
- Coverage: XX%
- Missing routes: [list]

### Test Results
- Packages with tests: N
- Tests passing: N
- Tests failing: N (list)
- Packages with no tests: [list]

### Violation Report
- Inline SQL strings: N (must be 0)
- orgID method params: N (must be 0)
- Noop/TODO stubs in repos: N

### Go Code Quality Assessment
| Category | Rating | Notes |
|----------|--------|-------|
| Error Handling | ⭐⭐⭐/⭐⭐/⭐/💀 | [summary] |
| Context Propagation | ⭐⭐⭐/⭐⭐/⭐/💀 | [summary] |
| Interface Design | ⭐⭐⭐/⭐⭐/⭐/💀 | [summary] |
| Naming Conventions | ⭐⭐⭐/⭐⭐/⭐/💀 | [summary] |
| Overall Architecture | ⭐⭐⭐/⭐⭐/⭐/💀 | [summary] |

**Qualitative commentary:** [3–5 sentences of honest professional assessment]

**Things that are Good:** [specific patterns or files that are exemplary]
**Things to Fix:** [specific issues with file:line references]

### Backend Completion Score: XX%
### Go Code Quality Score: XX%
```

---

## Agent 3 — Database Auditor

**Scope:** `aethel-core/internal/database/migrations/`, `aethel-core/internal/database/queries/queries.yaml`, `docs/db-design.mmd`

**You are a senior PostgreSQL engineer.**

### Step 1: Read pre-flight context

```bash
cat /tmp/audit-preflight.md
```

Check "Files Changed in Last 30 Commits" for any new migration files added by Tasks 11–14.

### Step 2: Discover all migration files

```bash
find aethel-core/internal/database/migrations -name "*.sql" | sort
```

Do not assume a fixed count — count what's actually there. For every `.up.sql` found, check a corresponding `.down.sql` exists.

For each migration, record: number, what it creates/alters, and whether it's referenced in `docs/db-design.mmd`.

**Baseline tables** (minimum expected in the ER diagram — check for any additions beyond these):
```
organizations           routing_rules               system_settings
departments             routing_rule_conditions     branding_configs
users                   routing_rule_destinations   audit_ledger (partitioned)
user_sessions           minute_sheets               + migration 21: ALTER branding_configs
password_reset_tokens   green_notes                   adds neutral_palette, font_family, wordmark
notification_preferences notifications
document_types          escalation_rules
dispatches              
dispatch_attachments    
dispatch_events         
```

### Step 3: queries.yaml completeness

```bash
cat aethel-core/internal/database/queries/queries.yaml
```

List every group and query key actually present. Then:

```bash
# Extract all qr.Get() calls from Go source
grep -rn 'qr\.Get(' aethel-core/internal/ | grep -oP '"[^"]+\.(?:[^"]+)"' | sort -u
```

Cross-reference: every key called in Go must exist in the YAML. Report any dangling references.

**Baseline query groups** (minimum expected after Tasks 10–14 — audit any additional groups found):
```
auth / users / sessions / password_reset / dispatch / dispatch_events /
routing_rules / minute_sheets / green_notes / escalation_rules /
governance / config / notifications
```

### Step 4: Template variable usage

```bash
grep -rn "{{ .Schema }}\|{{ T \|{{ E " aethel-core/internal/database/migrations/
```

### Output format — write to `/tmp/audit-agent-3.md`

```markdown
## Database Audit
_Audited at: [timestamp]_
_Migration files found: N up + N down_

### Migration Files
- Total up: N  |  Total down: N  |  Paired: ✅/❌
- Highest migration number: N
- Migration 21 (ALTER branding_configs): ✅/❌
- Unpaired migrations: [list]

### Schema vs ER Diagram
| Table | In Diagram | Migration # | Status | Discrepancies |
|-------|-----------|-------------|--------|---------------|

### queries.yaml Coverage
| Group | Exists | Query count | Missing queries |
|-------|--------|-------------|-----------------|

**Total queries defined: N**
**Groups found: N (baseline expected: 13)**

### Dangling qr.Get() Keys (in Go but not in YAML)
- N dangling keys: [list]

### Template Usage
- Migrations using templates: N / N total

### Database Completion Score: XX%
```

---

## Agent 4 — Cybersecurity Auditor (Frontend → Backend)

**Scope:** Entire repository — focus on `aethel-core/internal/`, `aethel-view/app/`, `.env.example`, `aethel-scripts/`. You assess the **full security posture from the browser to the database**, not just the backend middleware stack.

**You are a senior application security engineer AND a full-stack cybersecurity assessor.** Verify that all controls in `docs/architecture/architecture-security.md` are actually implemented, and also evaluate the depth and correctness of each control end-to-end.

### Step 1: Read pre-flight context + security architecture

```bash
cat /tmp/audit-preflight.md
cat docs/architecture/architecture-security.md
```

The security architecture doc is the **authoritative source** of what controls are required. If new controls were added to the doc by recent tasks, audit those too.

### Step 2: Authentication controls

```bash
# JWT algorithm
grep -rn "SigningMethod\|HS256\|RS256\|alg" aethel-core/internal/service/auth_service.go

# JWT claims — should have sub, role, iat, exp, jti — no org
grep -rn "MapClaims\|RegisteredClaims\|\"sub\"\|\"role\"\|\"org\"\|\"jti\"" aethel-core/internal/service/auth_service.go

# Argon2id params
grep -rn "argon2\|IDKey\|Memory\|Iterations\|Parallelism" aethel-core/internal/service/auth_service.go

# httpOnly cookie
grep -rn "HttpOnly\|SetCookie\|refresh_token" aethel-core/internal/api/handlers/auth.go

# localStorage forbidden
grep -rn "localStorage" aethel-view/app/

# Refresh token rotation — look for tx.Commit in session handling
grep -rn "BeginTx\|RotateSession\|DeleteSession.*InsertSession\|tx\.Commit" aethel-core/internal/service/auth_service.go aethel-core/internal/database/repos/session_repo.go
```

### Step 3: Middleware stack order

Read `aethel-core/internal/api/server.go`. List the exact middleware chain order as it appears in the code. Compare to expected:

```
Expected: Recovery → RequestID → StructuredLogger → RateLimiter → CORS → Auth → RBAC → Handler
```

### Step 4: CSRF protection

```bash
find aethel-core/internal -name "csrf*.go" -o -name "*csrf*.go" | sort
grep -rn "ConstantTimeCompare\|csrf\|X-CSRF" aethel-core/internal/
grep -rn "csrf" aethel-view/app/
```

### Step 5: Security headers

```bash
find aethel-core/internal -name "*security*header*" -o -name "*header*security*" | sort
grep -rn "X-Content-Type-Options\|X-Frame-Options\|Strict-Transport\|Content-Security-Policy\|Referrer-Policy" aethel-core/internal/
```

### Step 6: Secrets and account lockout

```bash
# No hardcoded secrets
grep -rn "AETHEL_JWT_SECRET\s*=\s*\|-----BEGIN RSA PRIVATE\|password\s*:=\s*\"" aethel-core/ aethel-view/ --include="*.go" --include="*.ts" --include="*.vue"

# .env.example exists and has no real values (only placeholders)
cat .env.example

# Account lockout — 423 status
grep -rn "423\|StatusLocked\|ErrAccountLocked" aethel-core/internal/

# db-harden.sql
ls aethel-core/scripts/db-harden.sql 2>/dev/null && grep -c "REVOKE" aethel-core/scripts/db-harden.sql
```

### Step 7: Full-Stack Cybersecurity Deep Assessment

This step goes beyond middleware checklist verification to assess the **depth and correctness** of security controls end-to-end.

#### 7a — Input Validation & Injection Prevention

```bash
cd aethel-core

# Verify all DB queries use parameterized queries (no string concatenation with user input)
grep -rn '\$\(.*req\|fmt\.Sprintf.*SELECT\|+ .*WHERE\|+ .*AND ' internal/database/repos/ | head -20

# Verify JSON binding validates input (check for struct tags with binding/validation)
grep -rn 'json.NewDecoder\|json.Unmarshal\|Decode(' internal/api/handlers/ | wc -l

# Look for request body size limits
grep -rn 'MaxBytesReader\|LimitReader\|http.MaxBytesHandler' internal/ | wc -l

# Check for SQL injection patterns (string building in queries)
grep -rn '".*\+.*req\.\|fmt\.Sprintf.*".*SELECT\|fmt\.Sprintf.*".*INSERT' internal/ | grep -v "_test.go"
```

Assessment:
- Are all SQL queries parameterized (using `$1, $2` placeholders)? Confirm via the queries.yaml file.
- Is there a global request body size limit to prevent DoS?
- Are JSON inputs validated (struct field types enforce basic type safety)?

Rate injection prevention: ⭐⭐⭐ / ⭐⭐ / ⭐ / 💀

#### 7b — Authorization Depth (RBAC correctness)

```bash
cd aethel-core

# Read the RBAC implementation
cat internal/rbac/*.go 2>/dev/null || find internal -name "*rbac*" -o -name "*permission*" | head -10

# Check all handlers — do they all go through RBAC middleware or is there any unprotected route?
grep -rn "RequireRole\|RequirePermission\|rbac\." internal/api/handlers/ | wc -l
grep -rn "RequireRole\|RequirePermission" internal/api/server.go | wc -l

# Check audit-log endpoint is sys_admin only
grep -A5 "audit-log\|audit_log\|AuditLog" internal/api/server.go | grep -i "sys_admin\|SysAdmin\|required_role"

# Check governance verify endpoint is sys_admin only  
grep -rn "VerifyChain\|verify.*chain\|audit.*verify" internal/api/server.go internal/api/handlers/ 2>/dev/null
```

Assessment:
- Is every route in `server.go` explicitly protected? (No route should be registered without either `auth` or an explicit public exemption)
- Are the sys_admin-only routes properly gated?
- Can a RECEPTION role user access ADMIN endpoints? (Trace the RBAC logic)
- Is there defense-in-depth: both middleware RBAC check AND service-level role validation?

Rate RBAC correctness: ⭐⭐⭐ / ⭐⭐ / ⭐ / 💀

#### 7c — Frontend Security Posture

```bash
cd aethel-view

# Check for XSS vectors — v-html usage (dangerous if with user content)
grep -rn "v-html" app/ | wc -l
grep -rn "v-html" app/

# Check for open redirect vulnerabilities (using user-controlled URLs in navigateTo)
grep -rn "navigateTo.*route\|navigateTo.*query\|navigateTo.*param" app/ | head -10

# Verify CSRF token is sent on state-changing requests
grep -rn "X-CSRF\|csrfToken\|csrf" app/ | wc -l

# Verify access token is NOT in localStorage or sessionStorage
grep -rn "localStorage\|sessionStorage" app/ | wc -l
grep -rn "localStorage\|sessionStorage" app/

# Check for sensitive data in console.log
grep -rn "console\.log" app/ | grep -i "token\|password\|secret\|key" | head -10
```

Assessment:
- Is `v-html` used anywhere? If so, is the content from a trusted source or could it be user-controlled?
- Is the CSRF token attached to all mutating API calls?
- Are there any open redirect risks?
- Does the frontend correctly use httpOnly cookies (meaning: it never reads/writes the refresh token)?

Rate frontend security: ⭐⭐⭐ / ⭐⭐ / ⭐ / 💀

#### 7d — Dependency Vulnerability Check

```bash
# Check Go dependencies for known vulnerabilities (if govulncheck is available)
cd aethel-core && govulncheck ./... 2>&1 | head -30 || echo "govulncheck not installed — skip"

# Check for outdated or suspicious Go deps
cat go.mod | grep -v "^//" | grep "require" -A 100 | head -50

# Check Node/pnpm deps
cd ../aethel-view && cat package.json | grep -E '"dependencies"|"devDependencies"' -A 50 | head -60
```

Assessment: Are there any known CVEs in the direct dependencies? Are dependency versions pinned?

Rate dependency hygiene: ⭐⭐⭐ / ⭐⭐ / ⭐ / 💀

#### 7e — Security Commentary

Write 3–5 sentences giving an honest professional assessment of the overall security posture:
- Is this application ready for production from a security standpoint?
- What are the top 2–3 security risks remaining?
- What security controls are already excellent and should be preserved?

### Output format — write to `/tmp/audit-agent-4.md`

```markdown
## Security Audit (Cybersecurity — Frontend to Backend)
_Audited at: [timestamp]_
_Security architecture doc read: ✅/❌_

### Authentication Controls
| Control | Status | Evidence (file:line) | Notes |
|---------|--------|----------------------|-------|
| JWT algorithm RS256/HS256 | | | |
| JWT claims correct (no org) | | | |
| Argon2id params ≥ 64MiB/3iter/4threads | | | |
| Access token NOT in localStorage | | | |
| Refresh token httpOnly cookie | | | |
| Refresh token rotation atomic | | | |

### Middleware Stack
Expected: Recovery → RequestID → StructuredLogger → RateLimiter → CORS → Auth → RBAC
Actual:   [exact order from server.go]
Match: ✅/❌

### CSRF Protection
| Check | Status | File | Notes |
|-------|--------|------|-------|
| Middleware exists | | | |
| ConstantTimeCompare used | | | |
| Token in readable cookie | | | |
| Applied to POST/PATCH/DELETE | | | |

### Security Headers
| Header | Present | Notes |
|--------|---------|-------|
| X-Content-Type-Options | | |
| X-Frame-Options | | |
| Strict-Transport-Security | | |
| Content-Security-Policy | | |
| Referrer-Policy | | |

### Other Controls
| Control | Status | Notes |
|---------|--------|-------|
| No hardcoded secrets | | |
| .env.example placeholder-only | | |
| Account lockout → 423 | | |
| db-harden.sql exists + REVOKE | | |

### Full-Stack Security Assessment
| Category | Rating | Notes |
|----------|--------|-------|
| Injection Prevention | ⭐⭐⭐/⭐⭐/⭐/💀 | [summary] |
| RBAC Correctness | ⭐⭐⭐/⭐⭐/⭐/💀 | [summary] |
| Frontend Security | ⭐⭐⭐/⭐⭐/⭐/💀 | [summary] |
| Dependency Hygiene | ⭐⭐⭐/⭐⭐/⭐/💀 | [summary] |

**Security commentary:** [3–5 sentences of honest assessment]

**Things that are Good:** [list of well-implemented security controls]
**Things to Fix:** [specific vulnerabilities or gaps with file:line]

### Security Score: XX% (N/M controls ✅)
### Critical security findings (❌ items): [list]
```

---

## Agent 5 — DevOps Auditor

**Scope:** `Makefile`, `docker-compose.yml`, `docker-compose.prod.yml`, `aethel-core/Dockerfile`, `aethel-view/Dockerfile`, `k8s/`, `.github/workflows/`, `aethel-scripts/`

**You are a senior DevOps / platform engineer.**

### Step 1: Read pre-flight context

```bash
cat /tmp/audit-preflight.md
```

Note any new workflow or infrastructure files added by recent tasks.

### Step 2: Discover all infra files

```bash
find . -name "Dockerfile*" -o -name "docker-compose*.yml" | sort
find k8s/ -type f 2>/dev/null | sort
find .github/workflows/ -type f 2>/dev/null | sort
find aethel-scripts/ -type f 2>/dev/null | sort
ls Makefile 2>/dev/null
```

Audit **everything found**, not just the baseline below.

### Step 3: Docker

For each Dockerfile found:
- Multi-stage build? (look for multiple `FROM`)
- Production stage uses distroless or minimal base (`scratch`, `gcr.io/distroless/*`, `alpine`)?
- No secrets in ENV or ARG?

For docker-compose files:
- Services defined: postgres, backend, frontend?
- PostgreSQL port: host `5433` → container `5432`?
- Health checks present?
- Secrets from env vars, not hardcoded?

### Step 4: Kubernetes

```bash
find k8s/ -type f | sort
```

**Baseline manifests** (minimum expected — audit any additional manifests found):
```
k8s/postgres/StatefulSet    k8s/backend/Deployment     k8s/frontend/Deployment
k8s/postgres/PVC            k8s/backend/HPA            k8s/frontend/Service
k8s/postgres/Service        k8s/backend/ConfigMap      k8s/ingress.yaml
```

For each found: does it set `namespace: aethel-workspace`?

### Step 5: GitHub Actions

```bash
find .github/workflows/ -name "*.yml" | sort
cat .github/workflows/*.yml 2>/dev/null
```

For each workflow found, read it and check:
- `ci.yml` — runs `go test ./...` AND `pnpm test` AND lint?
- `cd.yml` — builds + pushes to GHCR on merge to main?
- `security.yml` — runs Trivy + govulncheck + gosec?
- Any workflow: PostgreSQL service defined for integration tests?

### Step 6: Scripts

```bash
for f in aethel-scripts/setup-dev.sh aethel-scripts/health-check.sh \
          aethel-scripts/rotate-jwt-secret.sh aethel-scripts/db-backup.sh \
          aethel-scripts/k8s-rollout.sh; do
  [ -f "$f" ] && echo "$f: $(wc -l < $f) lines" || echo "$f: MISSING"
done
```

Also check `aethel-core/scripts/` for any new scripts added:
```bash
find aethel-core/scripts/ -type f 2>/dev/null | sort
```

### Output format — write to `/tmp/audit-agent-5.md`

```markdown
## DevOps Audit
_Audited at: [timestamp]_
_Infra files discovered: N_

### Docker
| Artifact | Exists | Multi-stage | Distroless stage | No hardcoded secrets | Notes |
|----------|--------|-------------|-----------------|----------------------|-------|

### Kubernetes Manifests
| Manifest | Exists | Correct namespace | Notes |
|----------|--------|------------------|-------|

**Manifests: X/N found (baseline expected: 9)**

### GitHub Actions Workflows
| Workflow | Exists | Key steps verified | Notes |
|----------|--------|-------------------|-------|

### Scripts
| Script | Exists | Non-empty | Notes |
|--------|--------|-----------|-------|

### Makefile
- Exists: ✅/❌
- make help: ✅/❌
- Key targets found: [list]

### DevOps Completion Score: XX%
```

---

## Agent 6 — Synthesis Agent (runs after all 5 complete)

### Step 1: Verify all temp files exist

```bash
for i in 1 2 3 4 5; do
  [ -f "/tmp/audit-agent-$i.md" ] && echo "Agent $i: ✅" || echo "Agent $i: ❌ MISSING"
done
```

If any file is missing, wait or re-run the missing agent before continuing.

### Step 2: Read all findings

Read `/tmp/audit-agent-{1..5}.md` in full. Extract from each:
- Completion score (`XX%`)
- UI/UX quality score (Agent 1)
- Go code quality score (Agent 2)
- Security score (Agent 4)
- "Things that are Good" lists
- "Things to Fix" lists
- Any ❌ critical findings

### Step 3: Compute overall score

```
overall = (frontend × 0.25) + (backend × 0.30) + (database × 0.15) + (security × 0.20) + (devops × 0.10)
```

### Step 4: Read the agile plan for sprint status comparison

```bash
cat docs/plans/agile-implementation-plan.md | head -60
```

Use the Implementation Status table in the plan doc as the expected baseline. For each sprint, compare what the plan says vs what the audit findings show.

### Step 5: Create the docs/reports directory

```bash
mkdir -p docs/reports
REPORT_DATE=$(date +%Y-%m-%d)
REPORT="docs/reports/audit-${REPORT_DATE}.html"
GIT_HEAD=$(git rev-parse --short HEAD)
```

### Step 6: Write the beautiful HTML audit report

Write a single self-contained HTML file to `$REPORT`. The HTML must include all CSS inline in a `<style>` block — no external stylesheets. Use the following structure and styling guidelines:

**Color palette** (matches Aethel's design system):
- Primary / Excellent: `#4f46e5` (indigo-600)
- Success / Good: `#10b981` (emerald-500)
- Warning / Partial: `#f59e0b` (amber-500)
- Danger / Missing: `#ef4444` (red-500)
- Neutral background: `#f8fafc` (slate-50)
- Card background: `#ffffff`
- Text body: `#1e293b` (slate-800)
- Text muted: `#64748b` (slate-500)

**Required HTML sections** (in this order):

1. **`<header>`** — Project name "Aethel Workspace", subtitle "Full-Project Audit Report", date, git HEAD, "Generated by 5-agent parallel audit team"

2. **Overall Score card** — large circular gauge showing `overall%`. Color: green if ≥80%, yellow if ≥60%, red if <60%. Below it: "Frontend: XX% | Backend: XX% | Database: XX% | Security: XX% | DevOps: XX%"

3. **Domain Score Cards** — 5 cards in a grid (or 3 rows of flexible layout), one per domain. Each card: domain name, icon (use emoji: 🎨 Frontend, ⚙️ Backend, 🗄️ Database, 🔒 Security, 🚀 DevOps), score as a horizontal progress bar with color coding, and 1–2 sentence summary from agent findings.

4. **Quality Ratings panel** — a table showing the star ratings (⭐⭐⭐/⭐⭐/⭐/💀) for all quality dimensions:
   - Accessibility | Design Consistency | UX Flow | Mobile Responsiveness | Aesthetic Quality (from Agent 1)
   - Error Handling | Context Propagation | Interface Design | Architecture (from Agent 2)
   - Injection Prevention | RBAC Correctness | Frontend Security | Dependency Hygiene (from Agent 4)

5. **Sprint Progress vs Agile Plan** — a color-coded table with these columns: Sprint, Plan Status, Audit Actual, Gap. Use green rows for "on track", yellow for "partial gap", red for "significant gap".

6. **"What's Good" section** — green-highlighted panel listing all items from the "Things that are Good" lists across all agents. Group by domain. Use ✅ icons.

7. **"Needs Improvement" section** — amber/red-highlighted panel listing all items from "Things to Fix" across all agents. Severity: 🔴 Critical (security/build failures), 🟡 Important (incomplete features), 🟢 Nice-to-have (quality improvements). Include file:line references where available.

8. **Statistics Table** — a clean metrics table:
   | Metric | Value | Status |
   |--------|-------|--------|
   | Total Vue pages | N | ✅/⚠️/❌ |
   | Pages fully complete | N | — |
   | Pages stub/incomplete | N | — |
   | Go packages | N | ✅/⚠️/❌ |
   | Repo implementations (real) | N | — |
   | Repo implementations (stub) | N | — |
   | Unit tests passing | N | ✅/⚠️/❌ |
   | SQL migrations | N | ✅/⚠️/❌ |
   | query.yaml groups | N | ✅/⚠️/❌ |
   | Routes registered | N | ✅/⚠️/❌ |
   | Security controls ✅ | N/M | ✅/⚠️/❌ |
   | Inline SQL violations | N | ✅ if 0 |
   | Palette CSS violations | N | ✅ if 0 |
   | localStorage violations | N | ✅ if 0 |

9. **Qualitative Design Commentary** — a styled blockquote with Agent 1's 3–5 sentence design assessment and Agent 2's Go quality assessment.

10. **Detailed Findings** — collapsible `<details>` sections, one per agent, containing the agent's full markdown findings rendered as HTML.

11. **`<footer>`** — "Aethel Workspace Audit · $(date +%Y-%m-%d) · Generated by Claude Code Task 15"

**HTML style requirements:**
- `font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Inter, sans-serif;`
- `max-width: 1200px; margin: 0 auto; padding: 2rem;` on main container
- Cards: `border-radius: 12px; box-shadow: 0 1px 3px rgba(0,0,0,.1); padding: 1.5rem; background: white;`
- Progress bars: `height: 8px; border-radius: 4px;` with appropriate background color
- Tables: zebra-striped rows, `border-collapse: collapse;`, header with indigo background + white text
- "Needs Improvement" severity badges: pill-shaped colored badges next to each item
- The overall score circle: `width: 160px; height: 160px; border-radius: 50%; border: 12px solid [color]; display: flex; align-items: center; justify-content: center;` with the percentage as large bold text inside

### Step 7: Determine if Task 16 is needed

After writing the HTML file, evaluate:
- Are there any 🔴 Critical findings (security vulnerabilities, build failures, ❌ Missing required features)?
- Are there ≥3 🟡 Important findings?

If yes to either: set `NEEDS_TASK_16=true`. Otherwise: set `NEEDS_TASK_16=false`.

Print to console: `Task 16 needed: [yes/no] — reason: [brief reason]`

### Step 8: Commit the HTML report

```bash
git add docs/reports/
git commit -m "chore(audit): project progress report $(date +%Y-%m-%d)

Overall score: XX%
Frontend: XX% | Backend: XX% | Database: XX% | Security: XX% | DevOps: XX%
Task 16 needed: [yes/no]

Co-Authored-By: Claude Code Task 15 <noreply@anthropic.com>"
git push origin dev
```

### Step 9: Print summary to console

Print the following directly to the terminal:

```
========================================
  AETHEL WORKSPACE — AUDIT SUMMARY
========================================
  Date: YYYY-MM-DD
  Git HEAD: [hash]

  OVERALL SCORE: XX%

  Frontend (25%):  XX%  [progress bar ASCII]
  Backend (30%):   XX%  [progress bar ASCII]
  Database (15%):  XX%  [progress bar ASCII]
  Security (20%):  XX%  [progress bar ASCII]
  DevOps (10%):    XX%  [progress bar ASCII]

  Critical findings: N
  Items to fix: N total (N critical, N important, N nice-to-have)

  HTML report: docs/reports/audit-YYYY-MM-DD.html
  Task 16: [WILL BE WRITTEN / NOT NEEDED]
========================================
```

---

## Agent 7 — Task 16 Writer (CONDITIONAL — only run if Agent 6 sets NEEDS_TASK_16=true)

**You are a senior technical project manager and engineer.** Read the HTML audit report and write `.claude/tasks/16_critical_fixes.md` — a targeted fix task to resolve all critical and important findings before Sprint 5 begins.

### Step 1: Read the audit HTML and all agent temp files

```bash
cat /tmp/audit-agent-1.md
cat /tmp/audit-agent-2.md
cat /tmp/audit-agent-3.md
cat /tmp/audit-agent-4.md
cat /tmp/audit-agent-5.md
```

Extract all "Things to Fix" items, grouped by severity.

### Step 2: Write `.claude/tasks/16_critical_fixes.md`

Structure the task file as follows:

```markdown
# Task 16 — Pre-Sprint-5 Critical Fixes

**Purpose:** Fix all critical (🔴) and important (🟡) findings from Task 15 audit before continuing with the agile plan.
**Generated:** [date]
**Source:** Task 15 audit — overall score XX%

---

## Why This Task Exists Before Sprint 5

[1–2 sentences explaining the top reasons: security gaps, build failures, incomplete features blocking Sprint 5]

---

## Fix Blocks (run in this order)

### Block 1 — Security Fixes (Critical 🔴)
[One subsection per critical security finding. Each must have:]
- Finding: [what the audit found, file:line]
- Required fix: [specific actionable change]
- Acceptance criteria: [how to verify it's fixed]
- Estimated effort: [S/M/L]

### Block 2 — Backend Completion (Important 🟡)
[One subsection per important backend gap]
- Same structure as Block 1

### Block 3 — Frontend Quality (Important 🟡)
[One subsection per important frontend gap]
- Same structure as Block 1

### Block 4 — Nice-to-Have Improvements (🟢 — optional, skip if time-constrained)
[Bullet list only — no detailed subsections]

---

## Definition of Done

- [ ] All Block 1 (🔴 Critical) items resolved
- [ ] All Block 2 and Block 3 (🟡 Important) items resolved
- [ ] `go build ./...` and `go vet ./...` pass with zero errors
- [ ] `pnpm build` in aethel-view passes with zero errors
- [ ] `go test ./...` — all tests pass
- [ ] Palette CSS violations: 0
- [ ] localStorage violations: 0
- [ ] Inline SQL violations: 0
- [ ] Re-run Task 15 after fixes to verify overall score improved
```

Apply the same prompt engineering principles as Tasks 10–14:
- Be **specific**: every fix item must include a file path and describe exactly what to change
- Be **verifiable**: every acceptance criterion must be a grep, test run, or build check
- Do **not** include fixes for nice-to-have items in Block 1/2/3 — keep those clearly separated in Block 4
- If a "Things to Fix" item from an agent is vague (e.g., "improve error handling"), make it specific by referencing the exact file and grep output from the agent's findings

### Step 3: Commit Task 16

```bash
git add .claude/tasks/16_critical_fixes.md
git commit -m "chore(tasks): add Task 16 — pre-Sprint-5 critical fixes from audit

Source: Task 15 audit $(date +%Y-%m-%d)
Critical findings: N | Important: N

Co-Authored-By: Claude Code Task 15 <noreply@anthropic.com>"
git push origin dev
```

---

## Execution Instructions

```
1. Run Step 0 pre-flight (orchestrating session, not a subagent)
2. Verify /tmp/audit-preflight.md was written successfully
3. Spawn Agents 1–5 in a single message (5 parallel Agent tool calls)
   — pass /tmp/audit-preflight.md path in each agent's prompt
4. Wait for all 5 completion notifications
5. Spawn Agent 6 to synthesize → HTML report
6. Agent 6 commits and pushes the HTML report
7. Agent 6 prints console summary and determines if Task 16 is needed
8. IF Task 16 needed: spawn Agent 7 to write .claude/tasks/16_critical_fixes.md
9. HTML report is at: docs/reports/audit-YYYY-MM-DD.html
```

## Definition of Done

- [ ] `/tmp/audit-preflight.md` written with current git state
- [ ] All 5 audit temp files written to `/tmp/audit-agent-{1..5}.md`
- [ ] Final report written to `docs/reports/audit-YYYY-MM-DD.html`
- [ ] HTML report is self-contained (all CSS inline, no external deps)
- [ ] Overall completion percentage computed with correct weighting
- [ ] Executive summary printed to console
- [ ] UI/UX quality ratings included (accessibility, design, aesthetics)
- [ ] Go code quality ratings included (error handling, interfaces, naming)
- [ ] Full-stack security assessment included (injection, RBAC, frontend)
- [ ] "What's Good" section has at least one item per domain
- [ ] "Needs Improvement" section lists all ❌ items with file paths
- [ ] Sprint progress table shows plan vs actual for all sprints
- [ ] Statistics summary table populated with real counts
- [ ] HTML report committed and pushed to `dev` branch
- [ ] Zero application files were modified (read-only audit)
- [ ] IF critical findings exist: `.claude/tasks/16_critical_fixes.md` written and pushed
