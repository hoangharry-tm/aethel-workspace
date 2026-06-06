<script setup lang="ts">
import { useMockData } from '~/composables/useMockData'

definePageMeta({ layout: 'workspace' })

const { documents } = useMockData()

const inboundDocs = computed(() => documents.filter(d => d.isInbound))

function timeAgo(timestamp: string): string {
  const diff = Date.now() - new Date(timestamp).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return 'just now'
  if (mins < 60) return `${mins}m ago`
  const hrs = Math.floor(mins / 60)
  if (hrs < 24) return `${hrs}h ago`
  const days = Math.floor(hrs / 24)
  return `${days}d ago`
}
</script>

<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between gap-4 flex-wrap">
      <div>
        <h1 class="text-xl font-bold text-body">
          {{ $t('dispatch.inboxTitle') }}
        </h1>
        <p class="text-sm text-muted mt-0.5">
          All incoming documents received at reception
        </p>
      </div>
      <UButton
        to="/dispatch/inbound/new"
        color="primary"
        variant="solid"
        leading-icon="i-lucide-plus"
      >
        {{ $t('dispatch.newInbound') }}
      </UButton>
    </div>

    <!-- Table -->
    <div class="bg-surface rounded-xl border border-border-base overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead class="bg-subtle border-b border-border-base">
            <tr>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider">
                {{ $t('dispatch.trackingNumber') }}
              </th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider">
                {{ $t('dispatch.subject') }}
              </th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider hidden sm:table-cell">
                {{ $t('dispatch.sender') }}
              </th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider">
                {{ $t('dispatch.priority') }}
              </th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider">
                {{ $t('common.status') }}
              </th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider hidden md:table-cell">
                Received
              </th>
              <th class="px-4 py-3 text-right text-xs font-semibold text-muted uppercase tracking-wider">
                {{ $t('common.actions') }}
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border-faint">
            <tr
              v-for="doc in inboundDocs"
              :key="doc.id"
              class="hover:bg-subtle transition-colors"
            >
              <td class="px-4 py-3">
                <span class="font-mono text-xs text-muted">{{ doc.trackingNumber }}</span>
              </td>
              <td class="px-4 py-3 max-w-xs">
                <p class="text-sm font-medium text-body truncate">
                  {{ doc.subject }}
                </p>
              </td>
              <td class="px-4 py-3 hidden sm:table-cell">
                <p class="text-xs text-muted">
                  {{ doc.senderOrg }}
                </p>
              </td>
              <td class="px-4 py-3">
                <UrgencyBadge :level="doc.urgency" />
              </td>
              <td class="px-4 py-3">
                <DocumentStatusBadge :status="doc.status" />
              </td>
              <td class="px-4 py-3 hidden md:table-cell">
                <span class="text-xs text-muted">{{ timeAgo(doc.dateReceived) }}</span>
              </td>
              <td class="px-4 py-3 text-right">
                <UButton
                  :to="`/documents/${doc.id}`"
                  color="neutral"
                  variant="outline"
                  size="xs"
                >
                  {{ $t('common.view') }}
                </UButton>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
