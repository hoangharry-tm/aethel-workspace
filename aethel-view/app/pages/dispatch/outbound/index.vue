<script setup lang="ts">
import { useMockData } from '~/composables/useMockData'

definePageMeta({ layout: 'workspace' })

const { documents } = useMockData()
const toast = useToast()

const outboundDocs = computed(() => documents.filter(d => !d.isInbound))

const showDispatchModal = ref(false)
const selectedDocId = ref<string | null>(null)
const dispatchMethod = ref('COURIER')
const dispatchLoading = ref(false)

const deliveryMethodOptions = [
  { label: 'Courier', value: 'COURIER' },
  { label: 'Post', value: 'POST' },
  { label: 'Hand Delivery', value: 'HAND_DELIVERY' },
  { label: 'Email', value: 'EMAIL' },
]

function openDispatch(docId: string) {
  selectedDocId.value = docId
  showDispatchModal.value = true
}

async function confirmDispatch() {
  dispatchLoading.value = true
  await new Promise(resolve => setTimeout(resolve, 700))
  dispatchLoading.value = false
  showDispatchModal.value = false
  toast.add({
    title: 'Document dispatched',
    description: `Document has been marked as dispatched via ${dispatchMethod.value.toLowerCase()}.`,
    color: 'success',
    icon: 'i-lucide-send',
  })
}

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
          {{ $t('dispatch.outboxTitle') }}
        </h1>
        <p class="text-sm text-muted mt-0.5">
          Outgoing dispatch requests from staff
        </p>
      </div>
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
                Requested By
              </th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider">
                {{ $t('dispatch.priority') }}
              </th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider">
                {{ $t('common.status') }}
              </th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider hidden md:table-cell">
                Submitted
              </th>
              <th class="px-4 py-3 text-right text-xs font-semibold text-muted uppercase tracking-wider">
                {{ $t('common.actions') }}
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border-faint">
            <tr
              v-for="doc in outboundDocs"
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
                <span class="text-xs text-muted">{{ doc.senderName }}</span>
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
                <div class="flex justify-end gap-2">
                  <UButton
                    v-if="doc.status === 'PENDING_ASSIGNMENT'"
                    color="primary"
                    variant="soft"
                    size="xs"
                    leading-icon="i-lucide-send"
                    @click="openDispatch(doc.id)"
                  >
                    Dispatch
                  </UButton>
                  <UButton
                    :to="`/documents/${doc.id}`"
                    color="neutral"
                    variant="outline"
                    size="xs"
                  >
                    {{ $t('common.view') }}
                  </UButton>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>

  <!-- Dispatch modal -->
  <UModal v-model:open="showDispatchModal">
    <template #content>
      <div class="p-6 space-y-4">
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 items-center justify-center rounded-full bg-violet-100">
            <UIcon name="i-lucide-send" class="h-5 w-5 text-violet-600" />
          </div>
          <div>
            <h3 class="text-base font-semibold text-body">
              Mark as Dispatched
            </h3>
            <p class="text-xs text-muted">
              Confirm the dispatch method used
            </p>
          </div>
        </div>

        <UFormField :label="$t('dispatch.deliveryMode')" name="method">
          <USelect
            v-model="dispatchMethod"
            :items="deliveryMethodOptions"
            class="w-full"
          />
        </UFormField>

        <div class="flex gap-2 pt-2">
          <UButton
            color="primary"
            variant="solid"
            :loading="dispatchLoading"
            leading-icon="i-lucide-check"
            @click="confirmDispatch"
          >
            Confirm Dispatch
          </UButton>
          <UButton
            color="neutral"
            variant="outline"
            @click="showDispatchModal = false"
          >
            {{ $t('common.cancel') }}
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>
