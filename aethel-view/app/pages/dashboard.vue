<script setup lang="ts">
import { useMockData } from "~/composables/useMockData";

definePageMeta({ layout: "workspace" });

const { t } = useI18n()
const { documents } = useMockData();

const activeFilter = ref<"all" | "pending" | "in-transit" | "delivered">("all");

const inboundDocs = computed(() => documents.filter((d) => d.isInbound));

const filteredDocuments = computed(() => {
  switch (activeFilter.value) {
    case "pending":
      return inboundDocs.value.filter((d) => d.status === "PENDING_ASSIGNMENT");
    case "in-transit":
      return inboundDocs.value.filter((d) => d.status === "IN_TRANSIT");
    case "delivered":
      return inboundDocs.value.filter((d) => d.status === "DELIVERED");
    default:
      return inboundDocs.value;
  }
});

const stats = computed(() => ({
  pending: inboundDocs.value.filter((d) => d.status === "PENDING_ASSIGNMENT")
    .length,
  inTransit: inboundDocs.value.filter((d) => d.status === "IN_TRANSIT").length,
  completed: inboundDocs.value.filter((d) => d.status === "DELIVERED").length,
  escalated: inboundDocs.value.filter((d) => d.status === "ESCALATED").length,
}));

function timeAgo(timestamp: string): string {
  const diff = Date.now() - new Date(timestamp).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.floor(hrs / 24);
  return `${days}d ago`;
}

const filterTabs = computed(() => [
  { key: "all" as const, label: t('common.all') },
  { key: "pending" as const, label: "Pending" },
  { key: "in-transit" as const, label: "In Transit" },
  { key: "delivered" as const, label: "Delivered" },
])

const statCards = computed(() => [
  {
    label: "Pending Pickup",
    value: stats.value.pending,
    icon: "i-lucide-clock",
    color: "text-amber-600",
    bg: "bg-amber-50",
    ring: "ring-amber-200",
  },
  {
    label: t('dashboard.inTransit'),
    value: stats.value.inTransit,
    icon: "i-lucide-truck",
    color: "text-sky-600",
    bg: "bg-sky-50",
    ring: "ring-sky-200",
  },
  {
    label: "Completed Today",
    value: stats.value.completed,
    icon: "i-lucide-check-circle",
    color: "text-emerald-600",
    bg: "bg-emerald-50",
    ring: "ring-emerald-200",
  },
  {
    label: t('dashboard.escalated'),
    value: stats.value.escalated,
    icon: "i-lucide-bell-ring",
    color: "text-rose-600",
    bg: "bg-rose-50",
    ring: "ring-rose-200",
  },
]);
</script>

<template>
  <div class="space-y-6">
    <!-- Page header -->
    <div class="flex items-start justify-between gap-4 flex-wrap">
      <div>
        <h1 class="text-xl font-bold text-body">{{ $t('dashboard.title') }}</h1>
        <p class="text-sm text-muted mt-0.5">
          Live document queue · auto-refreshes every 30s
        </p>
      </div>
      <!-- Live indicator -->
      <div class="flex items-center gap-1.5">
        <span class="relative flex h-2 w-2">
          <span
            class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"
          />
          <span
            class="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"
          />
        </span>
        <span class="text-xs font-medium text-emerald-600">{{ $t('dashboard.live') }}</span>
      </div>
    </div>

    <!-- Stats row -->
    <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
      <div
        v-for="stat in statCards"
        :key="stat.label"
        class="bg-surface rounded-xl border border-border-base p-4 flex items-center gap-3"
      >
        <div
          class="h-10 w-10 rounded-lg flex items-center justify-center flex-shrink-0 ring-1"
          :class="[stat.bg, stat.ring]"
        >
          <UIcon :name="stat.icon" class="h-5 w-5" :class="stat.color" />
        </div>
        <div>
          <p class="text-2xl font-bold text-body">
            {{ stat.value }}
          </p>
          <p class="text-xs text-muted">
            {{ stat.label }}
          </p>
        </div>
      </div>
    </div>

    <!-- Document queue -->
    <div class="bg-surface rounded-xl border border-border-base overflow-hidden">
      <!-- Filter tabs -->
      <div class="flex items-center gap-0 border-b border-border-base px-4 pt-4">
        <button
          v-for="tab in filterTabs"
          :key="tab.key"
          class="px-3 py-2 text-sm font-medium border-b-2 -mb-px transition-colors"
          :class="
            activeFilter === tab.key
              ? 'border-accent text-accent'
              : 'border-transparent text-muted hover:text-body'
          "
          @click="activeFilter = tab.key"
        >
          {{ tab.label }}
        </button>
      </div>

      <!-- Table -->
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead class="bg-subtle border-b border-border-base">
            <tr>
              <th
                class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider"
              >
                {{ $t('dispatch.trackingNumber') }}
              </th>
              <th
                class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider"
              >
                {{ $t('dispatch.subject') }}
              </th>
              <th
                class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider hidden sm:table-cell"
              >
                {{ $t('dispatch.sender') }}
              </th>
              <th
                class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider"
              >
                {{ $t('dispatch.priority') }}
              </th>
              <th
                class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider"
              >
                {{ $t('common.status') }}
              </th>
              <th
                class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider hidden lg:table-cell"
              >
                {{ $t('dispatch.department') }}
              </th>
              <th
                class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider hidden md:table-cell"
              >
                Time
              </th>
              <th
                class="px-4 py-3 text-right text-xs font-semibold text-muted uppercase tracking-wider"
              >
                {{ $t('common.actions') }}
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border-faint">
            <tr
              v-for="doc in filteredDocuments"
              :key="doc.id"
              class="hover:bg-subtle transition-colors"
            >
              <td class="px-4 py-3">
                <span class="font-mono text-xs text-muted">{{
                  doc.trackingNumber
                }}</span>
              </td>
              <td class="px-4 py-3 max-w-xs">
                <p class="text-sm font-medium text-body truncate">
                  {{ doc.subject }}
                </p>
              </td>
              <td class="px-4 py-3 hidden sm:table-cell">
                <p class="text-xs text-muted truncate max-w-32">
                  {{ doc.senderOrg }}
                </p>
              </td>
              <td class="px-4 py-3">
                <UrgencyBadge :level="doc.urgency" />
              </td>
              <td class="px-4 py-3">
                <DocumentStatusBadge :status="doc.status" />
              </td>
              <td class="px-4 py-3 hidden lg:table-cell">
                <span class="text-xs text-muted">{{ doc.department }}</span>
              </td>
              <td class="px-4 py-3 hidden md:table-cell">
                <span class="text-xs text-muted">{{
                  timeAgo(doc.dateReceived)
                }}</span>
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
            <tr v-if="filteredDocuments.length === 0">
              <td colspan="8" class="px-4 py-10 text-center">
                <UIcon
                  name="i-lucide-inbox"
                  class="h-8 w-8 text-icon-faint mx-auto mb-2"
                />
                <p class="text-sm text-icon-disabled">
                  {{ $t('common.noResults') }}
                </p>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
