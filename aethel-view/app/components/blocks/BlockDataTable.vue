<script setup lang="ts">
interface Column {
  key: string
  label: string
}

interface Props {
  title: string
  columns: Column[]
  rows: Record<string, unknown>[]
  emptyLabel?: string
}

const props = defineProps<Props>()
const { t } = useI18n()

const sortKey = ref<string | null>(null)
const sortDir = ref<'asc' | 'desc'>('asc')

function toggleSort(key: string) {
  if (sortKey.value === key) {
    sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  }
  else {
    sortKey.value = key
    sortDir.value = 'asc'
  }
}

const sortedRows = computed(() => {
  if (!sortKey.value) return props.rows
  const key = sortKey.value
  return [...props.rows].sort((a, b) => {
    const av = a[key]
    const bv = b[key]
    const aStr = String(av ?? '')
    const bStr = String(bv ?? '')
    return sortDir.value === 'asc'
      ? aStr.localeCompare(bStr)
      : bStr.localeCompare(aStr)
  })
})
</script>

<template>
  <div class="bg-surface rounded-xl border border-border-base overflow-hidden">
    <div class="px-4 py-3 border-b border-border-faint">
      <h3 class="text-sm font-semibold text-body">
        {{ title }}
      </h3>
    </div>

    <div class="overflow-x-auto">
      <table class="w-full text-sm">
        <thead class="bg-subtle border-b border-border-base">
          <tr>
            <th
              v-for="col in columns"
              :key="col.key"
              class="px-4 py-2.5 text-left text-xs font-semibold text-muted uppercase tracking-wider cursor-pointer select-none hover:text-body transition-colors"
              @click="toggleSort(col.key)"
            >
              <div class="flex items-center gap-1">
                {{ col.label }}
                <UIcon
                  v-if="sortKey === col.key"
                  :name="sortDir === 'asc' ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
                  class="h-3 w-3"
                />
                <UIcon
                  v-else
                  name="i-lucide-chevrons-up-down"
                  class="h-3 w-3 text-icon-faint"
                />
              </div>
            </th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border-faint">
          <template v-if="sortedRows.length > 0">
            <tr
              v-for="(row, idx) in sortedRows"
              :key="idx"
              class="transition-colors"
              :class="idx % 2 === 1 ? 'bg-subtle/50' : 'bg-surface'"
            >
              <td
                v-for="col in columns"
                :key="col.key"
                class="px-4 py-2.5 text-body"
              >
                {{ row[col.key] }}
              </td>
            </tr>
          </template>
          <tr v-else>
            <td :colspan="columns.length" class="px-4 py-10 text-center">
              <div class="flex flex-col items-center gap-2 text-icon-disabled">
                <UIcon name="i-lucide-inbox" class="h-8 w-8" />
                <span class="text-sm">{{ emptyLabel ?? t('common.noResults') }}</span>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
