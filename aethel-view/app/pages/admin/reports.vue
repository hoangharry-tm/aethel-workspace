<script setup lang="ts">
import { useMockData } from '~/composables/useMockData'

definePageMeta({ layout: 'workspace', middleware: ['role'], requiredRole: 'ADMIN' })

const { documents } = useMockData()
const toast = useToast()

const exportFrom = ref('')
const exportTo = ref('')
const exportFormat = ref('PDF')

const formatItems = [
  { label: 'PDF', value: 'PDF' },
  { label: 'CSV', value: 'CSV' },
  { label: 'Excel', value: 'Excel' },
]

const recentDocuments = computed(() => documents.slice(0, 10))

function timeAgo(ts: string): string {
  const diff = Date.now() - new Date(ts).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 60) return `${mins}m ago`
  const hrs = Math.floor(mins / 60)
  if (hrs < 24) return `${hrs}h ago`
  return `${Math.floor(hrs / 24)}d ago`
}

function handleExport() {
  toast.add({
    title: 'Export queued',
    description: `${exportFormat.value} report will download shortly.`,
    color: 'success',
    icon: 'i-lucide-download',
  })
}
</script>

<template>
  <div class="space-y-6">
    <!-- Page header + export controls -->
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-xl font-bold text-body">
          {{ $t('admin.reports') }}
        </h1>
        <p class="text-sm text-muted mt-0.5">
          Document flow metrics and SLA performance
        </p>
      </div>
      <div class="flex flex-wrap items-end gap-3">
        <UFormField label="From">
          <UInput v-model="exportFrom" type="date" />
        </UFormField>
        <UFormField label="To">
          <UInput v-model="exportTo" type="date" />
        </UFormField>
        <UFormField label="Format">
          <USelect v-model="exportFormat" :items="formatItems" />
        </UFormField>
        <UButton color="primary" icon="i-lucide-download" @click="handleExport">
          {{ $t('common.export') }} Report
        </UButton>
      </div>
    </div>

    <!-- KPI stat row -->
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      <BlockStatCard
        title="Total Documents"
        value="187"
        icon="i-lucide-files"
        trend="up"
        trend-value="+12 this week"
      />
      <BlockStatCard
        title="Avg Processing Time"
        value="6.4h"
        icon="i-lucide-clock"
        trend="down"
        trend-value="-0.8h vs last week"
      />
      <BlockStatCard
        title="SLA Compliance"
        value="94.1%"
        icon="i-lucide-shield-check"
        trend="up"
        trend-value="+2.3% vs last month"
      />
      <BlockStatCard
        title="Pending Escalations"
        value="3"
        icon="i-lucide-alert-triangle"
        trend="neutral"
        trend-value="Same as yesterday"
      />
    </div>

    <!-- Charts section -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <UCard>
        <template #header>
          <p class="font-semibold text-body">
            {{ $t('admin.dispatchVolume') }}
          </p>
        </template>
        <div class="h-64 flex flex-col items-center justify-center gap-2 bg-subtle rounded-xl">
          <UIcon name="i-lucide-bar-chart-2" class="h-8 w-8 text-muted" />
          <p class="text-sm text-muted">
            Chart data loads from API
          </p>
          <p class="text-xs text-muted">
            Connect backend to render live chart
          </p>
        </div>
      </UCard>
      <UCard>
        <template #header>
          <p class="font-semibold text-body">
            Status Distribution
          </p>
        </template>
        <div class="h-64 flex flex-col items-center justify-center gap-2 bg-subtle rounded-xl">
          <UIcon name="i-lucide-pie-chart" class="h-8 w-8 text-muted" />
          <p class="text-sm text-muted">
            Chart data loads from API
          </p>
          <p class="text-xs text-muted">
            Connect backend to render live chart
          </p>
        </div>
      </UCard>
    </div>

    <!-- Recent dispatches table -->
    <UCard>
      <template #header>
        <p class="font-semibold text-body">
          Recent Dispatches
        </p>
      </template>
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead class="bg-subtle border-b border-border-base">
            <tr>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider">
                Tracking #
              </th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider">
                {{ $t('dispatch.subject') }}
              </th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider">
                {{ $t('common.status') }}
              </th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider hidden sm:table-cell">
                {{ $t('dispatch.priority') }}
              </th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider hidden md:table-cell">
                Received
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border-faint">
            <tr
              v-for="doc in recentDocuments"
              :key="doc.id"
              class="hover:bg-subtle transition-colors"
            >
              <td class="px-4 py-3">
                <span class="font-mono text-xs text-muted">{{ doc.trackingNumber }}</span>
              </td>
              <td class="px-4 py-3">
                <span class="text-sm font-medium text-body line-clamp-1">{{ doc.subject }}</span>
              </td>
              <td class="px-4 py-3">
                <DocumentStatusBadge :status="doc.status" />
              </td>
              <td class="px-4 py-3 hidden sm:table-cell">
                <UrgencyBadge :level="doc.urgency" />
              </td>
              <td class="px-4 py-3 hidden md:table-cell">
                <span class="text-sm text-muted">{{ timeAgo(doc.dateReceived) }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </UCard>
  </div>
</template>
