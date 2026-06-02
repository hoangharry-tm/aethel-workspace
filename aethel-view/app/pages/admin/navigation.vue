<script setup lang="ts">
import { useAppRuntimeConfig } from '~/composables/useRuntimeConfig'
import type { NavGroup, NavItem } from '~/composables/useRuntimeConfig'

definePageMeta({ layout: 'workspace', middleware: ['role'], requiredRole: 'ADMIN' })

const { config, updateNav } = useAppRuntimeConfig()
const toast = useToast()

interface LocalNavItem extends NavItem {
  visible: boolean
}
interface LocalNavGroup extends Omit<NavGroup, 'items'> {
  items: LocalNavItem[]
}

// Deep-copy nav groups for local editing
const localGroups = ref<LocalNavGroup[]>(
  config.value.nav.map(g => ({
    ...g,
    items: g.items.map(item => ({ ...item, visible: true })),
  })),
)

const showAddModal = ref(false)
const addToGroupIndex = ref(0)
const newItemForm = reactive({
  label: '',
  icon: 'i-lucide-circle',
  to: '',
})

// Per-item editing state
const editingKey = ref<string | null>(null)
const editingLabel = ref('')

function startEdit(groupIdx: number, itemIdx: number, label: string) {
  editingKey.value = `${groupIdx}-${itemIdx}`
  editingLabel.value = label
}

function commitEdit(groupIdx: number, itemIdx: number) {
  const item = localGroups.value[groupIdx]?.items[itemIdx]
  if (item && editingLabel.value.trim()) {
    item.label = editingLabel.value.trim()
  }
  editingKey.value = null
}

function moveItem(groupIdx: number, itemIdx: number, direction: 'up' | 'down') {
  const items = localGroups.value[groupIdx]?.items
  if (!items) return
  const targetIdx = direction === 'up' ? itemIdx - 1 : itemIdx + 1
  if (targetIdx < 0 || targetIdx >= items.length) return
  const temp = items[itemIdx]!
  items[itemIdx] = items[targetIdx]!
  items[targetIdx] = temp
}

function openAddModal(groupIdx: number) {
  addToGroupIndex.value = groupIdx
  newItemForm.label = ''
  newItemForm.icon = 'i-lucide-circle'
  newItemForm.to = ''
  showAddModal.value = true
}

function addItem() {
  if (!newItemForm.label || !newItemForm.to) return
  const group = localGroups.value[addToGroupIndex.value]
  if (!group) return
  group.items.push({
    label: newItemForm.label,
    icon: newItemForm.icon,
    to: newItemForm.to,
    badge: null,
    visible: true,
  })
  showAddModal.value = false
  toast.add({ title: 'Item added', color: 'success', icon: 'i-lucide-check' })
}

function saveNavigation() {
  // Strip the local-only `visible` field before persisting (filter hidden items)
  const navToSave: NavGroup[] = localGroups.value.map(g => ({
    label: g.label,
    roles: g.roles,
    items: g.items
      .filter(i => i.visible)
      .map(({ label, icon, to, badge }) => ({ label, icon, to, badge })),
  }))
  updateNav(navToSave)
  toast.add({
    title: 'Navigation saved',
    description: 'Nav structure updated for all users.',
    color: 'success',
    icon: 'i-lucide-check',
  })
}

const roleBadgeColor: Record<string, 'primary' | 'success' | 'warning' | 'neutral'> = {
  ADMIN: 'primary',
  RECEPTION: 'success',
  USER: 'warning',
}
</script>

<template>
  <div class="space-y-6 max-w-2xl">
    <!-- Header -->
    <div class="flex items-center justify-between gap-4 flex-wrap">
      <div>
        <h1 class="text-xl font-bold text-body">
          Navigation
        </h1>
        <p class="text-sm text-muted mt-0.5">
          Manage sidebar nav groups, item order, labels, and visibility
        </p>
      </div>
      <UButton
        color="primary"
        variant="solid"
        leading-icon="i-lucide-save"
        @click="saveNavigation"
      >
        Save Navigation
      </UButton>
    </div>

    <UAlert
      color="info"
      variant="soft"
      icon="i-lucide-info"
      title="Nav changes take effect after saving. Hidden items are still accessible by direct URL."
    />

    <!-- Nav group cards -->
    <div
      v-for="(group, groupIdx) in localGroups"
      :key="group.label"
      class="bg-surface rounded-xl border border-border-base overflow-hidden"
    >
      <!-- Card header -->
      <div class="flex items-center gap-3 px-4 py-3 border-b border-border-faint">
        <span class="text-sm font-semibold text-body">{{ group.label }}</span>
        <div class="flex items-center gap-1">
          <UBadge
            v-for="role in group.roles"
            :key="role"
            :color="roleBadgeColor[role] ?? 'neutral'"
            variant="soft"
            size="xs"
          >
            {{ role }}
          </UBadge>
        </div>
      </div>

      <!-- Item list -->
      <ul class="divide-y divide-border-faint">
        <li
          v-for="(item, itemIdx) in group.items"
          :key="item.to"
          class="flex items-center gap-3 px-4 py-2.5"
          :class="!item.visible ? 'opacity-40' : ''"
        >
          <!-- Drag handle (visual) -->
          <UIcon name="i-lucide-grip-vertical" class="h-4 w-4 text-icon-faint flex-shrink-0" />

          <!-- Item icon -->
          <UIcon :name="item.icon" class="h-4 w-4 text-muted flex-shrink-0" />

          <!-- Editable label -->
          <div class="flex-1 min-w-0">
            <template v-if="editingKey === `${groupIdx}-${itemIdx}`">
              <UInput
                v-model="editingLabel"
                size="sm"
                class="w-full"
                autofocus
                @blur="commitEdit(groupIdx, itemIdx)"
                @keyup.enter="commitEdit(groupIdx, itemIdx)"
                @keyup.escape="editingKey = null"
              />
            </template>
            <button
              v-else
              class="text-sm text-body hover:text-accent transition-colors text-left w-full truncate"
              :title="'Click to rename'"
              @click="startEdit(groupIdx, itemIdx, item.label)"
            >
              {{ item.label }}
            </button>
          </div>

          <!-- Route path -->
          <span class="hidden sm:block text-xs font-mono text-icon-disabled truncate max-w-[120px]">
            {{ item.to }}
          </span>

          <!-- Visibility toggle -->
          <USwitch
            v-model="item.visible"
            size="sm"
            color="primary"
          />

          <!-- Up / Down reorder -->
          <div class="flex gap-0.5">
            <UButton
              icon="i-lucide-chevron-up"
              variant="ghost"
              color="neutral"
              size="xs"
              :disabled="itemIdx === 0"
              @click="moveItem(groupIdx, itemIdx, 'up')"
            />
            <UButton
              icon="i-lucide-chevron-down"
              variant="ghost"
              color="neutral"
              size="xs"
              :disabled="itemIdx === group.items.length - 1"
              @click="moveItem(groupIdx, itemIdx, 'down')"
            />
          </div>
        </li>
      </ul>

      <!-- Card footer: Add Item -->
      <div class="px-4 py-2.5 border-t border-border-faint bg-accent/5">
        <UButton
          variant="ghost"
          color="neutral"
          size="sm"
          leading-icon="i-lucide-plus"
          @click="openAddModal(groupIdx)"
        >
          Add Item
        </UButton>
      </div>
    </div>
  </div>

  <!-- Add Item Modal -->
  <UModal v-model:open="showAddModal">
    <template #content>
      <div class="p-6 space-y-4">
        <h3 class="text-base font-semibold text-body">
          Add Nav Item
        </h3>
        <UFormField label="Label" name="label" required>
          <UInput v-model="newItemForm.label" placeholder="e.g. Reports" class="w-full" />
        </UFormField>
        <UFormField label="Icon (Lucide)" name="icon">
          <UInput v-model="newItemForm.icon" placeholder="i-lucide-circle" class="w-full font-mono" />
        </UFormField>
        <UFormField label="Route" name="to" required>
          <UInput v-model="newItemForm.to" placeholder="/admin/reports" class="w-full font-mono" />
        </UFormField>
        <div class="flex gap-2 pt-2 border-t border-border-faint">
          <UButton color="primary" variant="solid" @click="addItem">
            Add Item
          </UButton>
          <UButton color="neutral" variant="outline" @click="showAddModal = false">
            Cancel
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>
