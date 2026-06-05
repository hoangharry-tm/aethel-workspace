<script setup lang="ts">
import { useMockData } from '~/composables/useMockData'
import type { TimelineEvent } from '~/components/shared/EventTimeline.vue'

definePageMeta({ layout: 'workspace' })

const route = useRoute()
const { documents, currentUser } = useMockData()
const toast = useToast()

const doc = computed(() => {
  return documents.find(d => d.id === route.params.id) ?? documents[0]!
})

// Handoff modal
const showHandoffModal = ref(false)
const handoffPin = ref('')
const handoffLoading = ref(false)

async function confirmHandoff() {
  handoffLoading.value = true
  await new Promise(resolve => setTimeout(resolve, 800))
  handoffLoading.value = false
  showHandoffModal.value = false
  handoffPin.value = ''
  toast.add({
    title: 'Handoff confirmed',
    description: `Document ${doc.value?.trackingNumber} has been handed over.`,
    color: 'success',
    icon: 'i-lucide-check-circle',
  })
}

// Acknowledge modal
const showAckModal = ref(false)
const ackLoading = ref(false)

async function confirmAcknowledge() {
  ackLoading.value = true
  await new Promise(resolve => setTimeout(resolve, 600))
  ackLoading.value = false
  showAckModal.value = false
  toast.add({
    title: 'Receipt acknowledged',
    description: 'You have acknowledged receipt of this document.',
    color: 'success',
    icon: 'i-lucide-check-circle',
  })
}

function copyTracking() {
  if (doc.value) {
    navigator.clipboard.writeText(doc.value.trackingNumber)
    toast.add({
      title: 'Copied',
      description: `Tracking number ${doc.value.trackingNumber} copied to clipboard.`,
      color: 'neutral',
      icon: 'i-lucide-copy',
    })
  }
}

const timelineEvents = computed<TimelineEvent[]>(() => [
  {
    id: 'e1',
    type: 'LOGGED',
    actorName: 'Marcus Webb',
    actorRole: 'Reception',
    note: 'Document received via courier, sealed and intact.',
    timestamp: doc.value?.dateReceived ?? new Date().toISOString(),
  },
  {
    id: 'e2',
    type: 'ROUTED',
    actorName: 'System',
    actorRole: 'Routing Engine',
    note: `Auto-routed to ${doc.value?.department} department.`,
    timestamp: new Date(new Date(doc.value?.dateReceived ?? '').getTime() + 1000 * 60 * 5).toISOString(),
  },
  {
    id: 'e3',
    type: 'NOTIFIED',
    actorName: 'System',
    actorRole: 'Notification Service',
    note: 'Recipient notified via email and in-app alert.',
    timestamp: new Date(new Date(doc.value?.dateReceived ?? '').getTime() + 1000 * 60 * 8).toISOString(),
  },
  ...(doc.value?.status === 'ATTEMPTED_DELIVERY' || doc.value?.status === 'DELIVERED' || doc.value?.status === 'ESCALATED' ? [
    {
      id: 'e4',
      type: 'HANDOFF_ATTEMPTED' as const,
      actorName: 'James Okonkwo',
      actorRole: 'Reception',
      note: 'Recipient not available at desk. Left notification slip.',
      timestamp: new Date(new Date(doc.value?.dateReceived ?? '').getTime() + 1000 * 60 * 60).toISOString(),
    },
  ] : []),
  ...(doc.value?.status === 'DELIVERED' ? [
    {
      id: 'e5',
      type: 'DELIVERED' as const,
      actorName: 'Marcus Webb',
      actorRole: 'Reception',
      note: 'Document physically handed to recipient. PIN confirmed.',
      timestamp: doc.value?.updatedAt ?? new Date().toISOString(),
    },
  ] : []),
  ...(doc.value?.status === 'ESCALATED' ? [
    {
      id: 'e5',
      type: 'ESCALATED' as const,
      actorName: 'Alice Thornton',
      actorRole: 'Admin',
      note: 'Escalated due to delivery failure and urgency classification.',
      timestamp: doc.value?.updatedAt ?? new Date().toISOString(),
    },
  ] : []),
])

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

const deliveryModeLabel: Record<string, string> = {
  POST: 'Post',
  COURIER: 'Courier',
  HAND_DELIVERY: 'Hand Delivery',
  EMAIL: 'Email',
}

// ── Green Notes ──────────────────────────────────────────────────────────────
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
  await new Promise<void>(resolve => setTimeout(resolve, 700))
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
  await new Promise<void>(resolve => setTimeout(resolve, 800))
  approveLoading.value = false
  showApproveModal.value = false
  toast.add({ title: 'Sheet approved', description: 'Minute sheet has been approved and locked.', color: 'success', icon: 'i-lucide-check-circle' })
}

const tabItems = [
  { label: 'Document Details', slot: 'details' as const },
  { label: 'Green Notes', slot: 'notes' as const },
]
</script>

<template>
  <div class="space-y-6 max-w-6xl">
    <!-- Back -->
    <div class="flex items-center gap-3">
      <UButton
        icon="i-lucide-arrow-left"
        color="neutral"
        variant="ghost"
        size="sm"
        @click="$router.back()"
      />
      <h1 class="text-xl font-bold text-body">
        Document Detail
      </h1>
    </div>

    <UTabs :items="tabItems" class="mt-2">
      <template #details="{ item }">
        <div class="grid grid-cols-1 lg:grid-cols-5 gap-6 mt-4">
          <!-- Left panel: 3/5 -->
          <div class="lg:col-span-3 space-y-4">
            <div class="bg-surface rounded-xl border border-border-base p-6 space-y-6">
              <!-- Tracking + badges -->
              <div>
                <div class="flex items-center gap-2 mb-2">
                  <span class="font-mono text-lg font-bold text-body">{{ doc?.trackingNumber }}</span>
                  <UButton
                    icon="i-lucide-copy"
                    color="neutral"
                    variant="ghost"
                    size="xs"
                    @click="copyTracking"
                  />
                </div>
                <div class="flex flex-wrap gap-2">
                  <UrgencyBadge v-if="doc" :level="doc.urgency" />
                  <DocumentStatusBadge v-if="doc" :status="doc.status" />
                </div>
              </div>

              <USeparator />

              <!-- Subject -->
              <div>
                <p class="text-xs font-semibold uppercase tracking-wider text-icon-disabled mb-1">
                  Subject
                </p>
                <p class="text-sm font-medium text-body">
                  {{ doc?.subject }}
                </p>
              </div>

              <!-- Sender info -->
              <div>
                <p class="text-xs font-semibold uppercase tracking-wider text-icon-disabled mb-3">
                  Sender Information
                </p>
                <div class="grid grid-cols-2 gap-4">
                  <div>
                    <p class="text-xs text-muted">
                      Name
                    </p>
                    <p class="text-sm font-medium text-body">
                      {{ doc?.senderName }}
                    </p>
                  </div>
                  <div>
                    <p class="text-xs text-muted">
                      Organization
                    </p>
                    <p class="text-sm font-medium text-body">
                      {{ doc?.senderOrg }}
                    </p>
                  </div>
                  <div>
                    <p class="text-xs text-muted">
                      Delivery Mode
                    </p>
                    <p class="text-sm font-medium text-body">
                      {{ deliveryModeLabel[doc?.deliveryMode ?? ''] ?? doc?.deliveryMode }}
                    </p>
                  </div>
                  <div>
                    <p class="text-xs text-muted">
                      Date Received
                    </p>
                    <p class="text-sm font-medium text-body">
                      {{ doc ? timeAgo(doc.dateReceived) : '' }}
                    </p>
                  </div>
                </div>
              </div>

              <!-- Routing -->
              <div>
                <p class="text-xs font-semibold uppercase tracking-wider text-icon-disabled mb-3">
                  Routing
                </p>
                <div class="mb-2">
                  <p class="text-xs text-muted">
                    Assigned To
                  </p>
                  <p class="text-sm font-medium text-body">
                    {{ doc?.department }} Department
                  </p>
                </div>
                <div v-if="doc && doc.routingChain.length > 1" class="flex items-center gap-1.5 flex-wrap">
                  <template
                    v-for="(stop, i) in doc.routingChain"
                    :key="stop"
                  >
                    <UBadge color="neutral" variant="soft" size="xs">
                      {{ stop }}
                    </UBadge>
                    <UIcon
                      v-if="i < doc.routingChain.length - 1"
                      name="i-lucide-arrow-right"
                      class="h-3 w-3 text-icon-disabled"
                    />
                  </template>
                </div>
              </div>

              <!-- Attachments -->
              <div>
                <p class="text-xs font-semibold uppercase tracking-wider text-icon-disabled mb-3">
                  Attachments
                </p>
                <div
                  v-for="file in doc?.attachments"
                  :key="file"
                  class="flex items-center gap-2 rounded-lg border border-border-base bg-subtle px-3 py-2"
                >
                  <UIcon name="i-lucide-file-text" class="h-5 w-5 text-rose-500 flex-shrink-0" />
                  <span class="text-sm text-body flex-1 truncate">{{ file }}</span>
                  <UButton icon="i-lucide-download" color="neutral" variant="ghost" size="xs" />
                </div>
              </div>

              <!-- Action buttons -->
              <div class="flex flex-wrap gap-2 pt-2">
                <UButton
                  v-if="currentUser.role === 'RECEPTION'"
                  color="primary"
                  variant="solid"
                  leading-icon="i-lucide-hand"
                  @click="showHandoffModal = true"
                >
                  Mark as Handed Over
                </UButton>
                <UButton
                  v-if="currentUser.role === 'USER'"
                  color="primary"
                  variant="solid"
                  leading-icon="i-lucide-check-circle"
                  @click="showAckModal = true"
                >
                  Acknowledge Receipt
                </UButton>
                <UButton
                  color="neutral"
                  variant="outline"
                  leading-icon="i-lucide-printer"
                >
                  Print Tracking Slip
                </UButton>
              </div>
            </div>
          </div>

          <!-- Right panel: 2/5 -->
          <div class="lg:col-span-2">
            <div class="bg-surface rounded-xl border border-border-base p-6 sticky top-6">
              <h2 class="text-sm font-semibold text-body mb-6">
                Event Timeline
              </h2>
              <EventTimeline :events="timelineEvents" />
            </div>
          </div>
        </div>
      </template>

      <template #notes="{ item }">
        <div class="mt-4 space-y-6">
          <!-- Action bar -->
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

          <!-- Timeline -->
          <div class="space-y-0">
            <template v-for="(note, idx) in greenNotes" :key="note.id">
              <div class="flex gap-4">
                <!-- Left: sequence circle + connector -->
                <div class="flex flex-col items-center">
                  <div class="flex h-8 w-8 items-center justify-center rounded-full bg-accent/10 text-accent text-xs font-bold flex-shrink-0">
                    {{ note.sequence }}
                  </div>
                  <div v-if="idx < greenNotes.length - 1" class="w-px flex-1 bg-border-base my-1" />
                </div>
                <!-- Right: note card -->
                <div class="pb-6 flex-1">
                  <div class="bg-surface rounded-xl border border-border-base p-4 space-y-3">
                    <div class="flex items-start justify-between gap-2">
                      <div>
                        <p class="text-sm font-semibold text-body">{{ note.authorName }}</p>
                        <p class="text-xs text-muted">{{ note.authorRole }} · {{ timeAgo(note.timestamp) }}</p>
                      </div>
                      <UBadge v-if="note.chainIntact" color="success" variant="soft" size="xs" leading-icon="i-lucide-link">
                        Chain intact
                      </UBadge>
                      <UBadge v-else color="error" variant="soft" size="xs" leading-icon="i-lucide-link-2-off">
                        Chain broken
                      </UBadge>
                    </div>
                    <p class="text-sm text-body leading-relaxed">{{ note.content }}</p>
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
              <!-- Chain link between notes -->
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

  <!-- Handoff confirmation modal -->
  <UModal v-model:open="showHandoffModal">
    <template #content>
      <div class="p-6 space-y-4">
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 items-center justify-center rounded-full bg-accent/10">
            <UIcon name="i-lucide-hand" class="h-5 w-5 text-accent" />
          </div>
          <div>
            <h3 class="text-base font-semibold text-body">
              Confirm Document Handoff
            </h3>
            <p class="text-xs text-muted">
              {{ doc?.trackingNumber }}
            </p>
          </div>
        </div>

        <p class="text-sm text-muted">
          I confirm this document has been physically handed to the recipient and they have acknowledged receipt.
        </p>

        <UFormField label="Confirmation PIN" name="pin">
          <UInput
            v-model="handoffPin"
            type="password"
            placeholder="Enter your PIN"
            icon="i-lucide-lock"
            class="w-full"
          />
        </UFormField>

        <div class="flex gap-2 pt-2">
          <UButton
            color="primary"
            variant="solid"
            :loading="handoffLoading"
            :disabled="!handoffPin"
            leading-icon="i-lucide-check"
            @click="confirmHandoff"
          >
            Confirm & Sign
          </UButton>
          <UButton
            color="neutral"
            variant="outline"
            @click="showHandoffModal = false"
          >
            Cancel
          </UButton>
        </div>
      </div>
    </template>
  </UModal>

  <!-- Acknowledge modal -->
  <UModal v-model:open="showAckModal">
    <template #content>
      <div class="p-6 space-y-4">
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 items-center justify-center rounded-full bg-emerald-100">
            <UIcon name="i-lucide-check-circle" class="h-5 w-5 text-emerald-600" />
          </div>
          <div>
            <h3 class="text-base font-semibold text-body">
              Acknowledge Receipt
            </h3>
            <p class="text-xs text-muted">
              {{ doc?.trackingNumber }}
            </p>
          </div>
        </div>

        <p class="text-sm text-muted">
          By clicking confirm, you acknowledge that you have received this document and accept responsibility for its handling.
        </p>

        <div class="flex gap-2 pt-2">
          <UButton
            color="primary"
            variant="solid"
            :loading="ackLoading"
            leading-icon="i-lucide-check"
            @click="confirmAcknowledge"
          >
            Confirm Receipt
          </UButton>
          <UButton
            color="neutral"
            variant="outline"
            @click="showAckModal = false"
          >
            Cancel
          </UButton>
        </div>
      </div>
    </template>
  </UModal>

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
        <p class="text-xs text-muted">{{ newNoteContent.length }} characters</p>
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
          <UButton color="neutral" variant="outline" @click="showAddNoteModal = false">Cancel</UButton>
        </div>
      </div>
    </template>
  </UModal>

  <!-- Approve Sheet modal -->
  <UModal v-model:open="showApproveModal">
    <template #content>
      <div class="p-6 space-y-4">
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 items-center justify-center rounded-full bg-success/10">
            <UIcon name="i-lucide-check-circle" class="h-5 w-5 text-success" />
          </div>
          <div>
            <h3 class="text-base font-semibold text-body">Approve Minute Sheet</h3>
            <p class="text-xs text-muted">{{ doc?.trackingNumber }}</p>
          </div>
        </div>
        <p class="text-sm text-muted">Approving the minute sheet locks it for further note additions. This action cannot be undone.</p>
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
          <UButton color="neutral" variant="outline" @click="showApproveModal = false">Cancel</UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>
