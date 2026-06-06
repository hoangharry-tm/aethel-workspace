<script setup lang="ts">
import { useNotificationDrawer } from '~/composables/useNotificationDrawer'
import { useMockData } from '~/composables/useMockData'

const { isOpen, close } = useNotificationDrawer()
const { notifications } = useMockData()

const localNotifications = ref(notifications.map(n => ({ ...n })))

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

function markAllRead() {
  localNotifications.value = localNotifications.value.map(n => ({ ...n, read: true }))
}

const eventLabels: Record<string, string> = {
  ESCALATED: 'Escalated',
  PENDING_ASSIGNMENT: 'New Inbound',
  ATTEMPTED_DELIVERY: 'Delivery Attempt',
  IN_TRANSIT: 'In Transit',
  DELIVERED: 'Delivered',
  DISPATCHED: 'Dispatched',
  UNDER_REVIEW: 'Under Review',
}

const eventColors: Record<string, string> = {
  ESCALATED: 'bg-rose-500',
  PENDING_ASSIGNMENT: 'bg-icon-disabled',
  ATTEMPTED_DELIVERY: 'bg-amber-500',
  IN_TRANSIT: 'bg-sky-500',
  DELIVERED: 'bg-emerald-500',
  DISPATCHED: 'bg-violet-500',
  UNDER_REVIEW: 'bg-accent',
}
</script>

<template>
  <USlideover
    v-model:open="isOpen"
    side="right"
    class="w-80 sm:w-96"
  >
    <template #content>
      <div class="flex flex-col h-full bg-surface">
        <!-- Header -->
        <div class="flex items-center justify-between px-4 py-3 border-b border-border-base">
          <h2 class="text-sm font-semibold text-body">
            {{ $t('nav.notifications') }}
          </h2>
          <div class="flex items-center gap-2">
            <UButton
              color="neutral"
              variant="ghost"
              size="xs"
              @click="markAllRead"
            >
              {{ $t('nav.markAllRead') }}
            </UButton>
            <UButton
              icon="i-lucide-x"
              color="neutral"
              variant="ghost"
              size="xs"
              :aria-label="$t('common.close')"
              @click="close"
            />
          </div>
        </div>

        <!-- List -->
        <div class="flex-1 overflow-y-auto divide-y divide-border-faint">
          <template v-if="localNotifications.length > 0">
            <NuxtLink
              v-for="notif in localNotifications"
              :key="notif.id"
              :to="`/documents/${notif.documentId}`"
              class="flex gap-3 px-4 py-3 hover:bg-subtle transition-colors"
              :class="!notif.read ? 'bg-accent/5 border-l-2 border-accent' : ''"
              @click="close"
            >
              <!-- Urgency dot -->
              <div class="flex-shrink-0 mt-1">
                <span
                  class="inline-block h-2 w-2 rounded-full"
                  :class="eventColors[notif.eventType] ?? 'bg-icon-disabled'"
                />
              </div>

              <!-- Content -->
              <div class="flex-1 min-w-0">
                <p class="text-xs font-medium text-body line-clamp-2 leading-snug">
                  {{ notif.subjectLine }}
                </p>
                <div class="mt-1 flex items-center gap-1.5">
                  <UBadge color="neutral" variant="soft" size="xs">
                    {{ eventLabels[notif.eventType] ?? notif.eventType }}
                  </UBadge>
                  <span class="text-[10px] text-icon-disabled">{{ timeAgo(notif.time) }}</span>
                </div>
                <p class="mt-0.5 text-[10px] font-mono text-icon-disabled">
                  {{ notif.trackingNumber }}
                </p>
              </div>
            </NuxtLink>
          </template>

          <!-- Empty state -->
          <div
            v-else
            class="flex flex-col items-center justify-center py-16 text-center"
          >
            <UIcon name="i-lucide-bell-off" class="h-10 w-10 text-icon-faint mb-3" />
            <p class="text-sm font-medium text-muted">
              {{ $t('nav.noNotifications') }}
            </p>
            <p class="text-xs text-icon-disabled mt-1">
              {{ $t('nav.allCaughtUp') }}
            </p>
          </div>
        </div>
      </div>
    </template>
  </USlideover>
</template>
