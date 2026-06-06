<script setup lang="ts">
import { useMockData } from "~/composables/useMockData";

definePageMeta({ layout: "workspace" });

const route = useRoute();
const { documents } = useMockData();

const searchQuery = ref((route.query.q as string) ?? "");
const statusFilter = ref("");
const urgencyFilter = ref("");
const docTypeFilter = ref("");

// Reka UI (Nuxt UI v4) forbids items with value:"" — the empty string is reserved for
// clearing the model and showing the placeholder. Omit the "All X" sentinel item and
// use the placeholder prop on USelect instead; the filter logic already treats "" as "no filter".
const statusOptions = [
  { label: "Pending Assignment", value: "PENDING_ASSIGNMENT" },
  { label: "Under Review", value: "UNDER_REVIEW" },
  { label: "In Transit", value: "IN_TRANSIT" },
  { label: "Delivered", value: "DELIVERED" },
  { label: "Escalated", value: "ESCALATED" },
  { label: "Dispatched", value: "DISPATCHED" },
];

const urgencyOptions = [
  { label: "Immediate", value: "IMMEDIATE" },
  { label: "Priority", value: "PRIORITY" },
  { label: "Routine", value: "ROUTINE" },
];

const docTypeOptions = [
  { label: "Audit Report", value: "Audit Report" },
  { label: "Legal Contract", value: "Legal Contract" },
  { label: "Invoice", value: "Invoice" },
  { label: "Regulatory Notice", value: "Regulatory Notice" },
  { label: "Budget Proposal", value: "Budget Proposal" },
  { label: "Report", value: "Report" },
];

const filteredResults = computed(() => {
  let results = [...documents];
  const q = searchQuery.value.toLowerCase().trim();

  if (q) {
    results = results.filter(
      (d) =>
        d.trackingNumber.toLowerCase().includes(q) ||
        d.senderName.toLowerCase().includes(q) ||
        d.senderOrg.toLowerCase().includes(q) ||
        d.subject.toLowerCase().includes(q),
    );
  }

  if (statusFilter.value) {
    results = results.filter((d) => d.status === statusFilter.value);
  }

  if (urgencyFilter.value) {
    results = results.filter((d) => d.urgency === urgencyFilter.value);
  }

  if (docTypeFilter.value) {
    results = results.filter((d) => d.documentType === docTypeFilter.value);
  }

  return results;
});

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

const hasFilters = computed(
  () =>
    searchQuery.value ||
    statusFilter.value ||
    urgencyFilter.value ||
    docTypeFilter.value,
);

function clearFilters() {
  searchQuery.value = "";
  statusFilter.value = "";
  urgencyFilter.value = "";
  docTypeFilter.value = "";
}
</script>

<template>
  <div class="space-y-6">
    <div>
      <h1 class="text-xl font-bold text-body">{{ $t('search.title') }}</h1>
      <p class="text-sm text-muted mt-0.5">
        Search by tracking number, sender, or subject
      </p>
    </div>

    <!-- Search bar -->
    <div class="relative">
      <UInput
        v-model="searchQuery"
        icon="i-lucide-search"
        :placeholder="$t('search.placeholder')"
        size="lg"
        class="w-full"
      />
    </div>

    <!-- Filter chips -->
    <div class="flex flex-wrap gap-3 items-center">
      <USelect
        v-model="statusFilter"
        :items="statusOptions"
        placeholder="All Statuses"
        size="sm"
        class="w-44"
      />
      <USelect
        v-model="urgencyFilter"
        :items="urgencyOptions"
        placeholder="All Urgencies"
        size="sm"
        class="w-36"
      />
      <USelect
        v-model="docTypeFilter"
        :items="docTypeOptions"
        placeholder="All Types"
        size="sm"
        class="w-44"
      />
      <UButton
        v-if="hasFilters"
        color="neutral"
        variant="ghost"
        size="sm"
        leading-icon="i-lucide-x"
        @click="clearFilters"
      >
        Clear filters
      </UButton>
    </div>

    <!-- Results count -->
    <p v-if="hasFilters" class="text-sm text-muted">
      {{ filteredResults.length }} result{{
        filteredResults.length !== 1 ? "s" : ""
      }}
      found
    </p>

    <!-- Results table -->
    <div class="bg-surface rounded-xl border border-border-base overflow-hidden">
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
              v-for="doc in filteredResults"
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
                <span class="text-xs text-muted">{{ doc.senderOrg }}</span>
              </td>
              <td class="px-4 py-3">
                <UrgencyBadge :level="doc.urgency" />
              </td>
              <td class="px-4 py-3">
                <DocumentStatusBadge :status="doc.status" />
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

            <!-- Empty state -->
            <tr v-if="filteredResults.length === 0">
              <td colspan="7" class="px-4 py-14 text-center">
                <UIcon
                  name="i-lucide-search-x"
                  class="h-10 w-10 text-icon-faint mx-auto mb-3"
                />
                <p class="text-sm font-medium text-muted">
                  {{ $t('common.noResults') }}
                </p>
                <p class="text-xs text-icon-disabled mt-1">
                  Try adjusting your search terms or filters
                </p>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
