# Task 13 — Complete Stub Frontend Pages + Green Notes Tab

**Date authored**: 2026-06-04  
**Phase**: Frontend enrichment (Phase 1 follow-up)  
**Executing model**: Must have Figma MCP tools available

---

## What already exists — Do NOT touch

The following files are fully working and must not be modified under any circumstances:

| File | Why it must not change |
|---|---|
| `aethel-view/app/assets/css/main.css` | Single source of truth for semantic CSS variables — Layer 1 of design system |
| `aethel-view/app/app.vue` | Runtime override point for CSS variables — Layer 2 of design system |
| `aethel-view/app/pages/admin/users.vue` | Working page — reference only |
| `aethel-view/app/pages/admin/routing-rules.vue` | Working page — reference only |
| `aethel-view/app/pages/admin/settings.vue` | Working page — reference only |
| `aethel-view/app/pages/admin/branding.vue` | Working page — reference only |
| `aethel-view/app/pages/admin/navigation.vue` | Working page — reference only |
| `aethel-view/app/pages/dashboard.vue` | Working page — do not touch |
| `aethel-view/app/composables/useMockData.ts` | Shared state — add mock data only if needed; never remove existing exports |
| `aethel-view/app/components/**/*` | All existing shared/block/layout components — use them, do not modify |
| `aethel-view/nuxt.config.ts` | Requires autoConfirm check before any modification |
| `aethel-view/app.config.ts` | Requires autoConfirm check before any modification |

---

## Invariants — Violations will cause task failure

| Rule | Requirement |
|---|---|
| **Semantic token rule** | Components use ONLY `text-body`, `text-muted`, `text-accent`, `bg-surface`, `bg-subtle` — never raw palette names (`text-slate-800`, `text-indigo-600`, `bg-white`, `bg-slate-50`, `text-gray-*`, `text-zinc-*`) |
| **Nuxt UI component rule** | Use `UButton`, `UTable`, `UModal`, `UBadge`, `UAlert`, `UCard`, `UTabs`, `UInput`, `USelect`, `UTextarea`, `UFormField`, `UPagination`, `UIcon`, `USeparator` — never raw HTML form elements styled with hardcoded colors |
| **Color attribute rule** | Always `color="primary"` on Nuxt UI components; never `color="indigo"` or any palette name; urgency badges follow the pattern: IMMEDIATE → `color="error"`, PRIORITY → `color="warning"`, ROUTINE → `color="success"` |
| **Icon rule** | Icons use `i-lucide-*` notation exclusively — no emoji, no heroicons, no other icon sets |
| **Figma-first rule** | Before writing any Vue code, the executing session MUST open the Figma file, study the design system page, and create wireframe frames for each new page. Only proceed to code after Figma frames exist (or after 2 minutes with no response) |
| **Mock data rule** | All reactive data comes from `useMockData()` composable or inline `ref()`/`reactive()` defined in the component. No hardcoded data arrays placed outside `<script setup>`. No API calls. |
| **Script setup rule** | Every Vue file uses `<script setup lang="ts">` only — no Options API, no `export default {}` |
| **Page meta rule** | All admin pages must have `definePageMeta({ layout: 'workspace', middleware: ['role'], requiredRole: 'ADMIN' })` as the first statement in `<script setup>` |
| **No console.log rule** | No `console.log`, `console.warn`, or `console.error` in committed code |

---

## Learn from these files first

Before writing any component, read these files in full to internalize the code patterns:

1. **`aethel-view/app/pages/admin/routing-rules.vue`** (360 lines) — gold standard for admin CRUD pages: modal pattern, reactive form, table display, toast notifications
2. **`aethel-view/app/pages/admin/users.vue`** (222 lines) — compact admin page pattern with UTable and UModal
3. **`aethel-view/app/pages/admin/settings.vue`** (249 lines) — card-based layout, USwitch toggles, section grouping
4. **`aethel-view/app/pages/documents/[id].vue`** (417 lines) — the enrichment target; understand its current tab-less structure and modal patterns
5. **`aethel-view/app/components/shared/EventTimeline.vue`** — timeline component interface (props shape)
6. **`aethel-view/app/components/blocks/BlockTimeline.vue`** — block timeline interface
7. **`aethel-view/app/components/blocks/BlockStatCard.vue`** — KPI card props
8. **`aethel-view/app/composables/useMockData.ts`** — understand all exported types and data shapes before using them

---

## Agent assignment table

| Agent | File(s) to write | Page purpose | Key components |
|---|---|---|---|
| **Agent A** | `app/pages/admin/audit-log.vue` | Immutable security event log viewer | `UTable`, `UInput`, `USelect`, `UPagination`, `UBadge`, `<pre>` expandable row |
| **Agent B** | `app/pages/admin/document-types.vue` + `app/pages/admin/escalation.vue` | Document type CRUD + escalation rules manager | `UCard`, `UModal`, `UToggle`, `UTable`, `UrgencyBadge` |
| **Agent C** | `app/pages/admin/reports.vue` | Analytics dashboard with KPI cards and chart placeholders | `BlockStatCard`, `UTable`, `UButton`, `USelect` |
| **Agent D** | `app/pages/documents/[id].vue` | Enrich existing page with Green Notes tab | `UTabs`, `UModal`, `UTextarea`, `UIcon` (link/chain icons), `EventTimeline` pattern |

Agents A, B, C, D work on non-overlapping files and must execute **in parallel** after the Figma design phase completes.

---

## Phase 1 — Figma Design (do this first, before any code)

### Step 1.1 — Open the Figma file

Invoke the `/figma-use` skill, then use the `use_figma` tool to open the Figma project:

- **File key**: `aqW7snNu6m0RoD0ZXrMH0f`
- **Action**: Read the existing pages — specifically the "Design System" page to understand the component library in use

### Step 1.2 — Study the design system page

Use `get_design_context` on the Design System page of the Figma file. Note:
- Color tokens already defined (primary = indigo-600, neutral = slate)
- Typography scale (Inter font family)
- Component patterns already present (cards, badges, tables, modals)

### Step 1.3 — Create wireframe frames for each new page

For each of the 5 targets below, use `use_figma` to create a new wireframe frame in the Figma file (add them to a new page titled "Task 13 — Stub Pages"):

1. **Audit Log** — filter bar at top, data table with expandable rows, pagination
2. **Document Types** — 3-col card grid, add/edit modal
3. **Escalation Rules** — table with inline toggle, add rule modal
4. **Reports Dashboard** — 4 KPI stat cards in a row, two chart placeholder boxes, recent activity table
5. **Green Notes Tab** — vertical hash-chain timeline within the document detail tab layout

### Step 1.4 — Proceed

After creating the Figma frames (or after 2 minutes with no response from Figma tools), proceed to Phase 2. Do not block on Figma approval.

---

## Phase 2 — Parallel Implementation

Spawn 4 subagents simultaneously. Each agent receives its specification below.

---

### Agent A specification — `admin/audit-log.vue`

**File to create**: `aethel-view/app/pages/admin/audit-log.vue`

**Page purpose**: The Audit Ledger viewer. Only `sys_admin` role can access this in production; for the prototype, use `requiredRole: 'ADMIN'` in `definePageMeta`.

**Layout structure** (single `<div class="space-y-6">` root):

1. **Page header row**  
   - `<h1 class="text-xl font-bold text-body">Audit Ledger</h1>`  
   - Subtitle: `<p class="text-sm text-muted">Immutable security event log with tamper detection</p>`  
   - "Export CSV" `UButton` with `variant="outline"` and `icon="i-lucide-download"` aligned to the right

2. **Filter bar** (`UCard` with `class="p-4"`, flex row, gap-3, flex-wrap):  
   - Date From: `UInput` `type="date"` `v-model="filterDateFrom"` label="From"  
   - Date To: `UInput` `type="date"` `v-model="filterDateTo"` label="To"  
   - Event Type: `USelect` `v-model="filterEventType"` with options: `[{ label: 'All Events', value: '' }, { label: 'Login Success', value: 'LOGIN_SUCCESS' }, { label: 'RBAC Denied', value: 'RBAC_DENIED' }, { label: 'Security Breach Attempt', value: 'SECURITY_BREACH_ATTEMPT' }, { label: 'Dispatch Created', value: 'DISPATCH_CREATED' }, { label: 'Config Changed', value: 'CONFIG_CHANGED' }, { label: 'User Updated', value: 'USER_UPDATED' }]`  
   - User search: `UInput` `v-model="filterUser"` `placeholder="Filter by user..."` `icon="i-lucide-search"`  
   - "Clear Filters" `UButton` `color="neutral"` `variant="ghost"` shown only when any filter is active

3. **Audit table** (`UTable`):  
   Columns: `[{ key: 'timestamp', label: 'Timestamp' }, { key: 'eventType', label: 'Event Type' }, { key: 'actor', label: 'User' }, { key: 'resource', label: 'Resource' }, { key: 'ipAddress', label: 'IP Address' }, { key: 'status', label: 'Status' }]`  
   
   - `eventType` cell: `UBadge` with color logic:
     - `SECURITY_BREACH_ATTEMPT` → `color="error"`
     - `RBAC_DENIED` → `color="warning"`
     - `RBAC_ELEVATION_ATTEMPT` → `color="error"`
     - `LOGIN_SUCCESS` → `color="success"`
     - `DISPATCH_CREATED` → `color="primary"`
     - `CONFIG_CHANGED` → `color="warning"`
     - `USER_UPDATED` → `color="neutral"`
   - `status` cell: `UBadge` — `ALLOWED` → `color="success"`, `DENIED` → `color="error"`, `FLAGGED` → `color="warning"`
   - **Row click expands inline**: use a `expandedRow` ref. Clicking a row toggles `expandedRow === row.id`. Show expanded payload below the row using a custom slot or a second row — render a `<pre class="bg-subtle rounded p-3 text-xs font-mono text-body overflow-x-auto">` block with the JSON payload

4. **Pagination**: `UPagination` at bottom, `v-model:page="currentPage"` `:total="filteredEntries.length"` `:page-count="20"`

**Mock data** — define inline in the component (10 entries):
```ts
const auditEntries = ref([
  { id: '1', timestamp: '2026-06-04T08:12:33Z', eventType: 'LOGIN_SUCCESS', actor: 'Alice Thornton', resource: '/auth/login', ipAddress: '10.0.1.42', status: 'ALLOWED', payload: { sessionId: 'sess_abc123', userAgent: 'Mozilla/5.0' } },
  { id: '2', timestamp: '2026-06-04T08:45:11Z', eventType: 'DISPATCH_CREATED', actor: 'Marcus Webb', resource: '/api/v1/dispatches', ipAddress: '10.0.1.55', status: 'ALLOWED', payload: { dispatchId: 'DSP-2026-0041', priority: 'IMMEDIATE' } },
  { id: '3', timestamp: '2026-06-04T09:02:47Z', eventType: 'RBAC_DENIED', actor: 'Priya Sharma', resource: '/admin/audit-log', ipAddress: '10.0.1.71', status: 'DENIED', payload: { requiredRole: 'sys_admin', actualRole: 'USER' } },
  { id: '4', timestamp: '2026-06-04T09:15:22Z', eventType: 'CONFIG_CHANGED', actor: 'Alice Thornton', resource: '/api/v1/admin/config/branding', ipAddress: '10.0.1.42', status: 'ALLOWED', payload: { field: 'primaryColor', oldValue: '#4f46e5', newValue: '#7c3aed' } },
  { id: '5', timestamp: '2026-06-04T10:30:05Z', eventType: 'SECURITY_BREACH_ATTEMPT', actor: 'unknown', resource: '/api/v1/admin/config', ipAddress: '203.0.113.77', status: 'DENIED', payload: { reason: 'Missing auth token', attempts: 3 } },
  { id: '6', timestamp: '2026-06-04T11:00:18Z', eventType: 'LOGIN_SUCCESS', actor: 'Marcus Webb', resource: '/auth/login', ipAddress: '10.0.1.55', status: 'ALLOWED', payload: { sessionId: 'sess_def456' } },
  { id: '7', timestamp: '2026-06-04T11:45:59Z', eventType: 'USER_UPDATED', actor: 'Alice Thornton', resource: '/api/v1/admin/users/usr_009', ipAddress: '10.0.1.42', status: 'ALLOWED', payload: { field: 'role', oldValue: 'USER', newValue: 'RECEPTION' } },
  { id: '8', timestamp: '2026-06-04T12:10:30Z', eventType: 'RBAC_ELEVATION_ATTEMPT', actor: 'Priya Sharma', resource: '/api/v1/admin/users', ipAddress: '10.0.1.71', status: 'FLAGGED', payload: { reason: 'JWT role claim mismatch detected' } },
  { id: '9', timestamp: '2026-06-04T13:22:14Z', eventType: 'DISPATCH_CREATED', actor: 'Marcus Webb', resource: '/api/v1/dispatches', ipAddress: '10.0.1.55', status: 'ALLOWED', payload: { dispatchId: 'DSP-2026-0042', priority: 'ROUTINE' } },
  { id: '10', timestamp: '2026-06-04T14:05:47Z', eventType: 'LOGIN_SUCCESS', actor: 'Priya Sharma', resource: '/auth/login', ipAddress: '10.0.1.71', status: 'ALLOWED', payload: { sessionId: 'sess_ghi789' } },
])
```

**Computed**: `filteredEntries` — filters `auditEntries` by `filterEventType`, `filterUser` (case-insensitive actor match), date range; `paginatedEntries` — slices filteredEntries by `currentPage` and pageSize 20.

**Handlers**:
- `const handleRowClick = (row: AuditEntry) => { expandedRow.value = expandedRow.value === row.id ? null : row.id }`
- `const handleExportCsv = () => { toast.add({ title: 'Export queued', description: 'CSV will download shortly.', color: 'success', icon: 'i-lucide-download' }) }`
- `const handleClearFilters = () => { filterDateFrom.value = ''; filterDateTo.value = ''; filterEventType.value = ''; filterUser.value = '' }`

---

### Agent B specification — `admin/document-types.vue` + `admin/escalation.vue`

#### File 1: `aethel-view/app/pages/admin/document-types.vue`

**Page purpose**: CRUD management for document type definitions.

**Layout structure**:

1. **Header row**: `<h1>Document Types</h1>` (text-xl font-bold text-body) + "Add Document Type" `UButton color="primary" icon="i-lucide-plus"` `@click="handleOpenAddModal"`

2. **Card grid**: `<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">`  
   Each card: `UCard` containing:
   - Icon circle: `<div class="flex h-10 w-10 items-center justify-center rounded-full bg-accent/10"><UIcon :name="type.icon" class="h-5 w-5 text-accent" /></div>`
   - Name: `<p class="font-semibold text-body">{{ type.name }}</p>`
   - Description: `<p class="text-sm text-muted">{{ type.description }}</p>`
   - Doc count badge: `<UBadge color="neutral" variant="soft">{{ type.docCount }} documents</UBadge>`
   - Active toggle shown inline: `<UBadge :color="type.active ? 'success' : 'neutral'" variant="soft">{{ type.active ? 'Active' : 'Inactive' }}</UBadge>`
   - Action buttons at card footer: Edit `UButton color="neutral" variant="ghost" icon="i-lucide-pencil"`, Delete `UButton color="error" variant="ghost" icon="i-lucide-trash-2"`

3. **Add/Edit modal** (`UModal v-model:open="showDocTypeModal"`):  
   Title: "Add Document Type" or "Edit Document Type" based on `isEditingDocType`  
   Fields:
   - Name: `UFormField label="Name" required` → `UInput v-model="docTypeForm.name" placeholder="e.g. Internal Memo"`
   - Description: `UFormField label="Description"` → `UTextarea v-model="docTypeForm.description" :rows="3"`
   - Icon: `UFormField label="Icon"` → `USelect v-model="docTypeForm.icon"` with options mapping 8 lucide icons: `i-lucide-file-text`, `i-lucide-file-contract`, `i-lucide-receipt`, `i-lucide-bar-chart-2`, `i-lucide-scroll`, `i-lucide-mail`, `i-lucide-clipboard`, `i-lucide-book-open`
   - Active: `UFormField label="Active"` → `UToggle v-model="docTypeForm.active"`
   - Footer: "Save" `UButton color="primary"` + "Cancel" `UButton color="neutral" variant="outline"`

4. **Delete confirmation modal** (`UModal v-model:open="showDeleteModal"`):  
   Warning icon + text: "This will permanently delete the document type. Dispatches using it will not be affected."  
   Buttons: "Delete" `UButton color="error"` + "Cancel" `UButton color="neutral" variant="outline"`

**Mock data** (inline `ref()`):
```ts
const docTypes = ref([
  { id: '1', name: 'Inbound Correspondence', description: 'Letters and parcels received from external senders', icon: 'i-lucide-mail', docCount: 41, active: true },
  { id: '2', name: 'Internal Memo', description: 'Internal communications between departments', icon: 'i-lucide-clipboard', docCount: 28, active: true },
  { id: '3', name: 'Contract', description: 'Legal agreements and binding documents', icon: 'i-lucide-file-contract', docCount: 15, active: true },
  { id: '4', name: 'Report', description: 'Analytical and audit reports submitted to management', icon: 'i-lucide-bar-chart-2', docCount: 9, active: true },
  { id: '5', name: 'Invoice', description: 'Financial billing documents from vendors and suppliers', icon: 'i-lucide-receipt', docCount: 33, active: false },
  { id: '6', name: 'Regulation Notice', description: 'Official directives from regulatory authorities', icon: 'i-lucide-scroll', docCount: 6, active: true },
])
```

**Handlers**:
- `handleOpenAddModal` — resets `docTypeForm`, sets `isEditingDocType = false`, opens modal
- `handleOpenEditModal(type)` — copies type into `docTypeForm`, sets `isEditingDocType = true`, opens modal
- `handleSaveDocType` — adds or updates entry in `docTypes`, closes modal, shows success toast
- `handleDeleteConfirm` — removes entry from `docTypes`, closes delete modal, shows success toast

---

#### File 2: `aethel-view/app/pages/admin/escalation.vue`

**Page purpose**: Manage escalation rules that trigger when dispatches are overdue.

**Layout structure**:

1. **Header row**: `<h1>Escalation Rules</h1>` (text-xl font-bold text-body) + subtitle `<p class="text-sm text-muted">Automated alerts triggered when documents exceed SLA thresholds</p>` + "Add Rule" `UButton color="primary" icon="i-lucide-plus"` `@click="handleOpenAddRule"`

2. **Rules table** (`UTable`):  
   Columns: `[{ key: 'name', label: 'Rule Name' }, { key: 'threshold', label: 'Trigger (hours)' }, { key: 'action', label: 'Action' }, { key: 'priorityFilter', label: 'Priority Filter' }, { key: 'active', label: 'Status' }]`
   
   - `priorityFilter` cell: `UrgencyBadge` component if value is `IMMEDIATE`, `PRIORITY`, or `ROUTINE`; `UBadge color="neutral" variant="soft"` with text "All Priorities" if value is `ALL`
   - `active` cell: `UToggle :model-value="rule.active" @update:model-value="handleToggleRule(rule)"` — inline toggle
   - `action` cell: text with icon prefix (`i-lucide-user-check` for Reassign, `i-lucide-bell` for Notify, `i-lucide-arrow-up-circle` for Escalate Status)
   - Row actions column: Edit `UButton color="neutral" variant="ghost" icon="i-lucide-pencil"`, Delete `UButton color="error" variant="ghost" icon="i-lucide-trash-2"`

3. **Add/Edit rule modal** (`UModal v-model:open="showRuleModal"`):  
   Fields:
   - Rule Name: `UInput v-model="ruleForm.name" placeholder="e.g. IMMEDIATE 4-Hour Alert"`
   - Overdue threshold: `UInput v-model.number="ruleForm.threshold" type="number" placeholder="4"` + unit label "(hours)"
   - Action: `USelect v-model="ruleForm.action"` options: `[{ label: 'Reassign to Supervisor', value: 'REASSIGN' }, { label: 'Send Notification', value: 'NOTIFY' }, { label: 'Escalate Status', value: 'ESCALATE_STATUS' }]`
   - Priority filter: `USelect v-model="ruleForm.priorityFilter"` options: `[{ label: 'All Priorities', value: 'ALL' }, { label: 'Immediate Only', value: 'IMMEDIATE' }, { label: 'Priority Only', value: 'PRIORITY' }, { label: 'Routine Only', value: 'ROUTINE' }]`
   - Active: `UToggle v-model="ruleForm.active"` label "Enable this rule"
   - Footer: "Save Rule" `UButton color="primary"` + "Cancel" `UButton color="neutral" variant="outline"`

**Mock data** (inline `ref()`):
```ts
const escalationRules = ref([
  { id: '1', name: 'IMMEDIATE — 2-Hour Alert', threshold: 2, action: 'NOTIFY', priorityFilter: 'IMMEDIATE', active: true },
  { id: '2', name: 'IMMEDIATE — 4-Hour Reassign', threshold: 4, action: 'REASSIGN', priorityFilter: 'IMMEDIATE', active: true },
  { id: '3', name: 'PRIORITY — 24-Hour Alert', threshold: 24, action: 'NOTIFY', priorityFilter: 'PRIORITY', active: true },
  { id: '4', name: 'ALL — 72-Hour Escalation', threshold: 72, action: 'ESCALATE_STATUS', priorityFilter: 'ALL', active: false },
  { id: '5', name: 'ROUTINE — 5-Day Reminder', threshold: 120, action: 'NOTIFY', priorityFilter: 'ROUTINE', active: true },
])
```

**Handlers**: `handleToggleRule(rule)` — toggles `rule.active`, shows toast; `handleOpenAddRule`, `handleOpenEditRule(rule)`, `handleSaveRule`, `handleDeleteRule` — same CRUD pattern as document-types.

---

### Agent C specification — `admin/reports.vue`

**File to create**: `aethel-view/app/pages/admin/reports.vue`

**Page purpose**: Analytics dashboard for management reporting.

**Layout structure**:

1. **Header row**: `<h1>Reports & Analytics</h1>` (text-xl font-bold text-body) + subtitle `<p class="text-sm text-muted">Document flow metrics and SLA performance</p>`  
   Also: export controls in a row — "From" `UInput type="date"` + "To" `UInput type="date"` + Format `USelect` (options: PDF, CSV, Excel) + "Export Report" `UButton color="primary" icon="i-lucide-download"`

2. **KPI stat row**: `<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">`  
   Four `BlockStatCard` components. Read the component's prop interface from `aethel-view/app/components/blocks/BlockStatCard.vue` before using it. Pass:
   - Card 1: `title="Total Documents" value="187" icon="i-lucide-files" trend="up" trendValue="+12 this week"`
   - Card 2: `title="Avg Processing Time" value="6.4h" icon="i-lucide-clock" trend="down" trendValue="-0.8h vs last week"`
   - Card 3: `title="SLA Compliance" value="94.1%" icon="i-lucide-shield-check" trend="up" trendValue="+2.3% vs last month"`
   - Card 4: `title="Pending Escalations" value="3" icon="i-lucide-alert-triangle" trend="neutral" trendValue="Same as yesterday"`

3. **Charts section**: `<div class="grid grid-cols-1 lg:grid-cols-2 gap-6">`  
   Two chart placeholder `UCard` components. Each card:
   - Header: card title (e.g., "Document Volume by Period")
   - Body: `<div class="h-64 flex flex-col items-center justify-center gap-2 bg-subtle rounded-xl border border-border-base"><UIcon name="i-lucide-bar-chart-2" class="h-8 w-8 text-muted" /><p class="text-sm text-muted">Chart data loads from API</p><p class="text-xs text-muted">Connect backend to render live chart</p></div>`
   - Chart 1 title: "Document Volume by Period" (bar chart placeholder)
   - Chart 2: use `i-lucide-pie-chart` icon, title "Status Distribution"

4. **Recent activity table**: `UCard` with title "Recent Dispatches"  
   `UTable` with 8–10 recent dispatch rows drawn from `useMockData().documents`  
   Columns: `[{ key: 'trackingNumber', label: 'Tracking #' }, { key: 'subject', label: 'Subject' }, { key: 'status', label: 'Status' }, { key: 'urgency', label: 'Priority' }, { key: 'dateReceived', label: 'Received' }]`
   - `status` cell: `DocumentStatusBadge` component from `components/shared/DocumentStatusBadge.vue`
   - `urgency` cell: `UrgencyBadge` component from `components/shared/UrgencyBadge.vue`
   - `dateReceived` cell: formatted relative time using a `timeAgo(ts)` helper (copy pattern from `documents/[id].vue`)

5. **No hardcoded palette classes anywhere.** The `BlockStatCard` component already uses semantic tokens — trust it.

---

### Agent D specification — `documents/[id].vue` Green Notes tab

**File to MODIFY**: `aethel-view/app/pages/documents/[id].vue` (currently 417 lines)

**WARNING**: Read the entire existing file before making any changes. The file has two top-level sections:
- `<script setup lang="ts">` (lines 1–137)
- `<template>` (lines 139–417) — currently a grid layout without tabs

**What to do**: Wrap the existing main content in a `UTabs` component so the page has two tabs:
1. **"Document Details"** tab — contains the existing `<div class="grid grid-cols-1 lg:grid-cols-5 gap-6">` and everything inside it
2. **"Green Notes"** tab — new content described below

**Implementation steps**:

**Step 1 — Add Green Notes data to `<script setup>`**  
After the existing `timeAgo` helper (line ~130), add:

```ts
// Green Notes tab
interface GreenNote {
  id: string
  sequence: number
  authorName: string
  authorRole: string
  timestamp: string
  content: string
  hash: string
  previousHash: string | null
  chainIntact: boolean
}

const greenNotes = ref<GreenNote[]>([
  {
    id: 'gn1',
    sequence: 1,
    authorName: 'Alice Thornton',
    authorRole: 'Admin',
    timestamp: '2026-06-04T08:30:00Z',
    content: 'Document received and reviewed. Routing confirmed to Finance department per standard protocol for IMMEDIATE priority items. No irregularities noted in the sender\'s credentials.',
    hash: 'a3f8c21d9e4b7f0512',
    previousHash: null,
    chainIntact: true,
  },
  {
    id: 'gn2',
    sequence: 2,
    authorName: 'Marcus Webb',
    authorRole: 'Reception',
    timestamp: '2026-06-04T09:15:00Z',
    content: 'Physical handoff to Finance department liaison completed. Document sealed condition confirmed at time of transfer. Recipient signed the custody log.',
    hash: 'b7e3d54a1c9f02867d',
    previousHash: 'a3f8c21d9e4b7f0512',
    chainIntact: true,
  },
  {
    id: 'gn3',
    sequence: 3,
    authorName: 'Priya Sharma',
    authorRole: 'User',
    timestamp: '2026-06-04T11:00:00Z',
    content: 'Document reviewed and contents verified against reference number in our procurement register. Approved for processing. No further action required from my end.',
    hash: 'c1a94e72f0d38b5690',
    previousHash: 'b7e3d54a1c9f02867d',
    chainIntact: true,
  },
])

const showAddNoteModal = ref(false)
const newNoteContent = ref('')
const addNoteLoading = ref(false)
const showApproveModal = ref(false)
const approveLoading = ref(false)

const canApproveSheet = computed(() =>
  currentUser.value.role === 'ADMIN' || currentUser.value.role === 'USER'
)

function handleOpenAddNote() {
  newNoteContent.value = ''
  showAddNoteModal.value = true
}

async function handleSubmitNote() {
  if (!newNoteContent.value.trim()) return
  addNoteLoading.value = true
  await new Promise(resolve => setTimeout(resolve, 700))
  const lastNote = greenNotes.value[greenNotes.value.length - 1]
  const newHash = Math.random().toString(36).slice(2, 20)
  greenNotes.value.push({
    id: `gn${greenNotes.value.length + 1}`,
    sequence: greenNotes.value.length + 1,
    authorName: currentUser.value.name,
    authorRole: currentUser.value.role,
    timestamp: new Date().toISOString(),
    content: newNoteContent.value.trim(),
    hash: newHash,
    previousHash: lastNote?.hash ?? null,
    chainIntact: true,
  })
  addNoteLoading.value = false
  showAddNoteModal.value = false
  newNoteContent.value = ''
  toast.add({ title: 'Note added', description: 'Your note has been appended to the minute sheet.', color: 'success', icon: 'i-lucide-check-circle' })
}

async function handleApproveSheet() {
  approveLoading.value = true
  await new Promise(resolve => setTimeout(resolve, 800))
  approveLoading.value = false
  showApproveModal.value = false
  toast.add({ title: 'Sheet approved', description: 'Minute sheet has been approved and locked.', color: 'success', icon: 'i-lucide-check-circle' })
}
```

**Step 2 — Restructure `<template>`**  
Replace the existing template content (keeping the two modals at the bottom and the back-navigation header) with a `UTabs` structure:

```html
<template>
  <div class="space-y-6 max-w-6xl">
    <!-- Back nav (keep exactly as-is) -->
    ...

    <UTabs :items="[{ label: 'Document Details', slot: 'details' }, { label: 'Green Notes', slot: 'notes' }]">
      <template #details>
        <!-- paste existing grid layout here verbatim -->
        <div class="grid grid-cols-1 lg:grid-cols-5 gap-6 mt-4">
          ...
        </div>
      </template>

      <template #notes>
        <div class="mt-4 space-y-6">
          <!-- Notes action bar -->
          <div class="flex items-center justify-between">
            <div>
              <p class="text-sm font-semibold text-body">Minute Sheet — Green Notes</p>
              <p class="text-xs text-muted">Hash-chained and tamper-evident</p>
            </div>
            <div class="flex gap-2">
              <UButton
                v-if="canApproveSheet"
                color="success"
                variant="outline"
                leading-icon="i-lucide-check-circle"
                @click="showApproveModal = true"
              >
                Approve Sheet
              </UButton>
              <UButton
                color="primary"
                variant="solid"
                leading-icon="i-lucide-plus"
                @click="handleOpenAddNote"
              >
                Add Note
              </UButton>
            </div>
          </div>

          <!-- Green notes timeline -->
          <div class="space-y-0">
            <template v-for="(note, idx) in greenNotes" :key="note.id">
              <!-- Note card -->
              <div class="flex gap-4">
                <!-- Left: sequence + connector line -->
                <div class="flex flex-col items-center">
                  <div class="flex h-8 w-8 items-center justify-center rounded-full bg-accent/10 text-accent text-xs font-bold flex-shrink-0">
                    {{ note.sequence }}
                  </div>
                  <div v-if="idx < greenNotes.length - 1" class="w-px flex-1 bg-border-base my-1" />
                </div>
                <!-- Right: note content -->
                <div class="pb-6 flex-1">
                  <div class="bg-surface rounded-xl border border-border-base p-4 space-y-3">
                    <div class="flex items-start justify-between gap-2">
                      <div>
                        <p class="text-sm font-semibold text-body">{{ note.authorName }}</p>
                        <p class="text-xs text-muted">{{ note.authorRole }} · {{ timeAgo(note.timestamp) }}</p>
                      </div>
                      <!-- Chain integrity indicator -->
                      <UBadge
                        v-if="note.chainIntact"
                        color="success"
                        variant="soft"
                        size="xs"
                        leading-icon="i-lucide-link"
                      >
                        Chain intact
                      </UBadge>
                      <UBadge
                        v-else
                        color="error"
                        variant="soft"
                        size="xs"
                        leading-icon="i-lucide-link-2-off"
                      >
                        Chain broken
                      </UBadge>
                    </div>
                    <p class="text-sm text-body leading-relaxed">{{ note.content }}</p>
                    <!-- Hash display -->
                    <div class="flex items-center gap-2 pt-1">
                      <UIcon name="i-lucide-fingerprint" class="h-3.5 w-3.5 text-muted flex-shrink-0" />
                      <span class="text-xs font-mono text-muted">{{ note.hash.slice(0, 16) }}...</span>
                      <template v-if="note.previousHash">
                        <UIcon name="i-lucide-arrow-left" class="h-3 w-3 text-muted" />
                        <span class="text-xs font-mono text-muted">{{ note.previousHash.slice(0, 16) }}...</span>
                      </template>
                      <span v-else class="text-xs text-muted italic">genesis note</span>
                    </div>
                  </div>
                </div>
              </div>

              <!-- Chain link connector (between notes, not after last) -->
              <div v-if="idx < greenNotes.length - 1" class="flex gap-4 -mt-5 mb-1">
                <div class="w-8 flex justify-center">
                  <UIcon
                    :name="greenNotes[idx + 1]?.chainIntact ? 'i-lucide-link' : 'i-lucide-link-2-off'"
                    :class="greenNotes[idx + 1]?.chainIntact ? 'text-accent' : 'text-error'"
                    class="h-4 w-4"
                  />
                </div>
              </div>
            </template>

            <!-- Empty state -->
            <div v-if="greenNotes.length === 0" class="flex flex-col items-center gap-3 py-12 text-center">
              <UIcon name="i-lucide-notebook" class="h-8 w-8 text-muted" />
              <p class="text-sm text-muted">No notes yet. Add the first note to start the minute sheet.</p>
            </div>
          </div>
        </div>
      </template>
    </UTabs>
  </div>

  <!-- Keep ALL existing modals (handoff + acknowledge) exactly as-is -->

  <!-- Add Note modal -->
  <UModal v-model:open="showAddNoteModal">
    <template #content>
      <div class="p-6 space-y-4">
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 items-center justify-center rounded-full bg-accent/10">
            <UIcon name="i-lucide-notebook-pen" class="h-5 w-5 text-accent" />
          </div>
          <div>
            <h3 class="text-base font-semibold text-body">Add Green Note</h3>
            <p class="text-xs text-muted">Appended immutably to the minute sheet</p>
          </div>
        </div>
        <UFormField label="Note Content" name="noteContent">
          <UTextarea
            v-model="newNoteContent"
            :rows="5"
            placeholder="Enter your note. This will be hash-chained to the previous entry and cannot be edited after submission."
            class="w-full"
          />
        </UFormField>
        <p class="text-xs text-muted">
          {{ newNoteContent.length }} characters
        </p>
        <div class="flex gap-2 pt-2">
          <UButton
            color="primary"
            variant="solid"
            :loading="addNoteLoading"
            :disabled="!newNoteContent.trim()"
            leading-icon="i-lucide-check"
            @click="handleSubmitNote"
          >
            Submit Note
          </UButton>
          <UButton color="neutral" variant="outline" @click="showAddNoteModal = false">
            Cancel
          </UButton>
        </div>
      </div>
    </template>
  </UModal>

  <!-- Approve Sheet modal -->
  <UModal v-model:open="showApproveModal">
    <template #content>
      <div class="p-6 space-y-4">
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 items-center justify-center rounded-full bg-green-100">
            <UIcon name="i-lucide-check-circle" class="h-5 w-5 text-green-600" />
          </div>
          <div>
            <h3 class="text-base font-semibold text-body">Approve Minute Sheet</h3>
            <p class="text-xs text-muted">{{ doc?.trackingNumber }}</p>
          </div>
        </div>
        <p class="text-sm text-muted">
          Approving the minute sheet locks it for further note additions. This action cannot be undone.
        </p>
        <div class="flex gap-2 pt-2">
          <UButton
            color="success"
            variant="solid"
            :loading="approveLoading"
            leading-icon="i-lucide-check"
            @click="handleApproveSheet"
          >
            Confirm Approval
          </UButton>
          <UButton color="neutral" variant="outline" @click="showApproveModal = false">
            Cancel
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>
```

**Important**: The `UTabs` items prop syntax depends on the installed Nuxt UI v4 version. Read `aethel-view/node_modules/@nuxt/ui/dist/` or look at how `UTabs` is used in any existing page to confirm the correct prop name. Nuxt UI v4 uses `items` prop with `{ label, slot }` objects.

---

## Phase 3 — Verification

After all 4 agents complete their files, run these commands from `aethel-view/`:

### Build check
```bash
cd aethel-view
pnpm build
```
Expected: exits with code 0, zero errors.

### TypeScript check
```bash
cd aethel-view
pnpm typecheck
```
Expected: zero TypeScript errors.

### Palette class grep (must return zero matches)
```bash
grep -rn "text-slate\|text-indigo\|bg-white\|bg-slate\|text-gray\|text-zinc\|text-emerald\|text-rose\|text-amber\|bg-emerald\|bg-rose\|bg-amber\|bg-indigo" \
  aethel-view/app/pages/admin/audit-log.vue \
  aethel-view/app/pages/admin/document-types.vue \
  aethel-view/app/pages/admin/escalation.vue \
  aethel-view/app/pages/admin/reports.vue
```
Expected: no output (zero matches). If any matches appear, fix before declaring done.

Note: `documents/[id].vue` has a pre-existing `bg-emerald-100`/`text-emerald-600` in the acknowledge modal icon circle (lines ~379–380 in the original file). This is a known pre-existing violation that is out of scope for this task — do not introduce new ones, but do not fix old ones either (that is a separate task).

### Dev server visual check
```bash
cd aethel-view
pnpm dev
```
Manually verify in browser:
- `/admin/audit-log` — filter bar renders, table shows 10 entries, row click expands JSON payload
- `/admin/document-types` — 6 cards render, Add modal opens
- `/admin/escalation` — 5 rules in table, inline toggle works, Add modal opens
- `/admin/reports` — 4 stat cards render, 2 chart placeholders render, recent dispatches table shows data
- `/documents/DSP-2026-0001` (or any valid doc id) — two tabs visible, Green Notes tab shows 3 chained notes, Add Note button opens modal

---

## Definition of Done checklist

- [ ] `aethel-view/app/pages/admin/audit-log.vue` created with filter bar, 10-entry table, row-expand, pagination
- [ ] `aethel-view/app/pages/admin/document-types.vue` created with 6-card grid, add/edit modal, delete confirmation
- [ ] `aethel-view/app/pages/admin/escalation.vue` created with 5-rule table, inline active toggle, add/edit modal
- [ ] `aethel-view/app/pages/admin/reports.vue` created with 4 KPI cards, 2 chart placeholders, recent dispatches table, export section
- [ ] `aethel-view/app/pages/documents/[id].vue` enriched with Green Notes tab showing 3 hash-chained notes, Add Note modal, Approve Sheet modal
- [ ] All 5 files use only semantic token classes (`text-body`, `text-muted`, `text-accent`, `bg-surface`, `bg-subtle`)
- [ ] Palette grep returns zero matches on the 4 new admin pages
- [ ] `pnpm build` exits with code 0
- [ ] `pnpm typecheck` exits with code 0
- [ ] All 4 admin pages have `definePageMeta({ layout: 'workspace', middleware: ['role'], requiredRole: 'ADMIN' })`
- [ ] Figma frames for all 5 pages created in file `aqW7snNu6m0RoD0ZXrMH0f` before any code was written
- [ ] No `console.log` statements in any new or modified file
- [ ] No hardcoded data arrays outside `<script setup>`
