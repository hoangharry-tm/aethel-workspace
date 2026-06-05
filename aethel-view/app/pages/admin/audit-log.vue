<script setup lang="ts">
definePageMeta({ layout: 'workspace', middleware: ['role'], requiredRole: 'ADMIN' })

interface AuditEntry {
  id: string
  timestamp: string
  eventType: string
  actor: string
  resource: string
  ipAddress: string
  status: string
  payload: Record<string, unknown>
}

const toast = useToast()

const auditEntries = ref<AuditEntry[]>([
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

const filterDateFrom = ref('')
const filterDateTo = ref('')
const filterEventType = ref('')
const filterUser = ref('')
const expandedRowId = ref<string | null>(null)
const currentPage = ref(1)
const pageSize = 20

const eventTypeOptions = [
  { label: 'Login Success', value: 'LOGIN_SUCCESS' },
  { label: 'RBAC Denied', value: 'RBAC_DENIED' },
  { label: 'Security Breach Attempt', value: 'SECURITY_BREACH_ATTEMPT' },
  { label: 'Dispatch Created', value: 'DISPATCH_CREATED' },
  { label: 'Config Changed', value: 'CONFIG_CHANGED' },
  { label: 'User Updated', value: 'USER_UPDATED' },
]

const hasActiveFilters = computed(() =>
  !!filterDateFrom.value || !!filterDateTo.value || !!filterEventType.value || !!filterUser.value,
)

const filteredEntries = computed(() => {
  return auditEntries.value.filter((entry) => {
    if (filterEventType.value && entry.eventType !== filterEventType.value) return false
    if (filterUser.value && !entry.actor.toLowerCase().includes(filterUser.value.toLowerCase())) return false
    if (filterDateFrom.value && entry.timestamp < filterDateFrom.value) return false
    if (filterDateTo.value && entry.timestamp > filterDateTo.value + 'T23:59:59Z') return false
    return true
  })
})

const paginatedEntries = computed(() => {
  const start = (currentPage.value - 1) * pageSize
  return filteredEntries.value.slice(start, start + pageSize)
})

function getEventTypeBadgeColor(eventType: string): 'error' | 'warning' | 'success' | 'primary' | 'neutral' {
  switch (eventType) {
    case 'SECURITY_BREACH_ATTEMPT': return 'error'
    case 'RBAC_DENIED': return 'warning'
    case 'RBAC_ELEVATION_ATTEMPT': return 'error'
    case 'LOGIN_SUCCESS': return 'success'
    case 'DISPATCH_CREATED': return 'primary'
    case 'CONFIG_CHANGED': return 'warning'
    case 'USER_UPDATED': return 'neutral'
    default: return 'neutral'
  }
}

function getStatusBadgeColor(status: string): 'success' | 'error' | 'warning' {
  switch (status) {
    case 'ALLOWED': return 'success'
    case 'DENIED': return 'error'
    case 'FLAGGED': return 'warning'
    default: return 'warning'
  }
}

function handleRowClick(row: AuditEntry) {
  expandedRowId.value = expandedRowId.value === row.id ? null : row.id
}

function handleExportCsv() {
  toast.add({ title: 'Export queued', description: 'CSV will download shortly.', color: 'success', icon: 'i-lucide-download' })
}

function handleClearFilters() {
  filterDateFrom.value = ''
  filterDateTo.value = ''
  filterEventType.value = ''
  filterUser.value = ''
}
</script>

<template>
  <div class="space-y-6">
    <!-- Page header -->
    <div class="flex justify-between items-start gap-4 flex-wrap">
      <div>
        <h1 class="text-xl font-bold text-body">
          Audit Ledger
        </h1>
        <p class="text-sm text-muted mt-0.5">
          Immutable security event log with tamper detection
        </p>
      </div>
      <UButton
        variant="outline"
        color="neutral"
        leading-icon="i-lucide-download"
        @click="handleExportCsv"
      >
        Export CSV
      </UButton>
    </div>

    <!-- Filter bar -->
    <UCard class="p-4">
      <div class="flex flex-wrap gap-3 items-end">
        <UFormField label="From">
          <UInput
            v-model="filterDateFrom"
            type="date"
          />
        </UFormField>

        <UFormField label="To">
          <UInput
            v-model="filterDateTo"
            type="date"
          />
        </UFormField>

        <UFormField label="Event Type">
          <USelect
            v-model="filterEventType"
            :items="eventTypeOptions"
            placeholder="All Events"
          />
        </UFormField>

        <UFormField label="User">
          <UInput
            v-model="filterUser"
            placeholder="Filter by user..."
            leading-icon="i-lucide-search"
          />
        </UFormField>

        <UButton
          v-if="hasActiveFilters"
          color="neutral"
          variant="ghost"
          @click="handleClearFilters"
        >
          Clear Filters
        </UButton>
      </div>
    </UCard>

    <!-- Audit table -->
    <UCard>
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead class="bg-subtle border-b border-border-base">
            <tr>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider">
                Timestamp
              </th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider">
                Event Type
              </th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider">
                User
              </th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider hidden md:table-cell">
                Resource
              </th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider hidden lg:table-cell">
                IP Address
              </th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider">
                Status
              </th>
              <th class="px-4 py-3 text-right text-xs font-semibold text-muted uppercase tracking-wider w-10" />
            </tr>
          </thead>
          <tbody>
            <template
              v-for="row in paginatedEntries"
              :key="row.id"
            >
              <tr
                class="border-b border-border-faint hover:bg-subtle transition-colors cursor-pointer"
                @click="handleRowClick(row)"
              >
                <td class="px-4 py-3">
                  <span class="text-xs font-mono text-muted">
                    {{ new Date(row.timestamp).toLocaleString() }}
                  </span>
                </td>
                <td class="px-4 py-3">
                  <UBadge
                    :color="getEventTypeBadgeColor(row.eventType)"
                    variant="soft"
                    size="sm"
                  >
                    {{ row.eventType.replace(/_/g, ' ') }}
                  </UBadge>
                </td>
                <td class="px-4 py-3">
                  <span class="text-sm font-medium text-body">{{ row.actor }}</span>
                </td>
                <td class="px-4 py-3 hidden md:table-cell">
                  <span class="text-xs font-mono text-muted">{{ row.resource }}</span>
                </td>
                <td class="px-4 py-3 hidden lg:table-cell">
                  <span class="text-xs font-mono text-muted">{{ row.ipAddress }}</span>
                </td>
                <td class="px-4 py-3">
                  <UBadge
                    :color="getStatusBadgeColor(row.status)"
                    variant="soft"
                    size="sm"
                  >
                    {{ row.status }}
                  </UBadge>
                </td>
                <td class="px-4 py-3 text-right">
                  <UButton
                    :icon="expandedRowId === row.id ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
                    size="xs"
                    variant="ghost"
                    color="neutral"
                    @click.stop="handleRowClick(row)"
                  />
                </td>
              </tr>
              <tr
                v-if="expandedRowId === row.id"
                :key="`${row.id}-expanded`"
                class="border-b border-border-faint"
              >
                <td
                  colspan="7"
                  class="px-4 py-3 bg-subtle"
                >
                  <div class="flex items-center gap-2 mb-2">
                    <UIcon
                      name="i-lucide-file-json"
                      class="h-4 w-4 text-muted"
                    />
                    <span class="text-xs font-semibold text-muted uppercase tracking-wider">Event Payload</span>
                  </div>
                  <pre class="bg-subtle rounded p-3 text-xs font-mono text-body overflow-x-auto">{{ JSON.stringify(row.payload, null, 2) }}</pre>
                </td>
              </tr>
            </template>
            <tr v-if="paginatedEntries.length === 0">
              <td
                colspan="7"
                class="px-4 py-10 text-center text-sm text-muted"
              >
                <UIcon
                  name="i-lucide-shield-off"
                  class="h-8 w-8 mx-auto mb-2 text-icon-faint"
                />
                No audit entries match the current filters.
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Pagination -->
      <div
        v-if="filteredEntries.length > pageSize"
        class="flex justify-center pt-4 pb-2 border-t border-border-faint mt-2"
      >
        <UPagination
          v-model:page="currentPage"
          :total="filteredEntries.length"
          :page-count="pageSize"
        />
      </div>
    </UCard>
  </div>
</template>
