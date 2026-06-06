<script setup lang="ts">
import { useMockData } from '~/composables/useMockData'
import type { UrgencyLevel } from '~/composables/useMockData'

definePageMeta({ layout: 'workspace', middleware: ['role'], requiredRole: 'ADMIN' })

const { routingRules } = useMockData()
const toast = useToast()

const localRules = ref(routingRules.map(r => ({ ...r })))

const showRuleModal = ref(false)
const editingRule = ref<typeof localRules.value[0] | null>(null)
const isEditing = ref(false)

const modalForm = reactive({
  documentType: '',
  senderOrg: '',
  urgency: '' as UrgencyLevel | '',
  destination: '',
  isMultiStop: false,
  stops: ['Reception', ''],
})

const documentTypeOptions = [
  { label: 'Audit Report', value: 'Audit Report' },
  { label: 'Legal Contract', value: 'Legal Contract' },
  { label: 'Invoice', value: 'Invoice' },
  { label: 'Regulatory Notice', value: 'Regulatory Notice' },
  { label: 'Budget Proposal', value: 'Budget Proposal' },
  { label: 'General Correspondence', value: 'General Correspondence' },
]

const urgencyOptions = [
  { label: 'Immediate', value: 'IMMEDIATE' },
  { label: 'Priority', value: 'PRIORITY' },
  { label: 'Routine', value: 'ROUTINE' },
]

const departmentOptions = [
  { label: 'Finance', value: 'Finance' },
  { label: 'Legal', value: 'Legal' },
  { label: 'HR', value: 'HR' },
  { label: 'Operations', value: 'Operations' },
  { label: 'Procurement', value: 'Procurement' },
  { label: 'Administration', value: 'Administration' },
]

function openNewRule() {
  isEditing.value = false
  editingRule.value = null
  Object.assign(modalForm, {
    documentType: '',
    senderOrg: '',
    urgency: '',
    destination: '',
    isMultiStop: false,
    stops: ['Reception', ''],
  })
  showRuleModal.value = true
}

function openEditRule(rule: typeof localRules.value[0]) {
  isEditing.value = true
  editingRule.value = rule
  Object.assign(modalForm, {
    documentType: rule.documentType,
    senderOrg: rule.senderOrg ?? '',
    urgency: rule.urgency ?? '',
    destination: rule.destination,
    isMultiStop: rule.stops.length > 2,
    stops: [...rule.stops],
  })
  showRuleModal.value = true
}

function saveRule() {
  if (isEditing.value && editingRule.value) {
    const idx = localRules.value.findIndex(r => r.id === editingRule.value?.id)
    if (idx !== -1 && localRules.value[idx]) {
      localRules.value[idx] = {
        ...localRules.value[idx]!,
        documentType: modalForm.documentType,
        senderOrg: modalForm.senderOrg || undefined,
        urgency: (modalForm.urgency as UrgencyLevel) || undefined,
        destination: modalForm.destination,
        stops: modalForm.isMultiStop ? modalForm.stops.filter(Boolean) : ['Reception', modalForm.destination],
      }
    }
    toast.add({ title: 'Rule updated', color: 'success', icon: 'i-lucide-check' })
  } else {
    const newRule = {
      id: `r${localRules.value.length + 1}`,
      priority: localRules.value.length + 1,
      documentType: modalForm.documentType,
      senderOrg: modalForm.senderOrg || undefined,
      urgency: (modalForm.urgency as UrgencyLevel) || undefined,
      destination: modalForm.destination,
      stops: modalForm.isMultiStop ? modalForm.stops.filter(Boolean) : ['Reception', modalForm.destination],
      isActive: true,
    }
    localRules.value.push(newRule)
    toast.add({ title: 'Rule created', color: 'success', icon: 'i-lucide-check' })
  }
  showRuleModal.value = false
}

function deleteRule(id: string) {
  localRules.value = localRules.value.filter(r => r.id !== id)
  toast.add({ title: 'Rule deleted', color: 'neutral', icon: 'i-lucide-trash' })
}

function addStop() {
  modalForm.stops.push('')
}

function removeStop(i: number) {
  modalForm.stops.splice(i, 1)
}
</script>

<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between gap-4 flex-wrap">
      <div>
        <h1 class="text-xl font-bold text-body">
          {{ $t('admin.routingRules') }}
        </h1>
        <p class="text-sm text-muted mt-0.5">
          Define how documents are automatically routed by type and sender
        </p>
      </div>
      <UButton
        color="primary"
        variant="solid"
        leading-icon="i-lucide-plus"
        @click="openNewRule"
      >
        {{ $t('admin.createRule') }}
      </UButton>
    </div>

    <!-- Info note -->
    <UAlert
      color="neutral"
      variant="soft"
      icon="i-lucide-info"
      title="Rules are matched by priority order — higher priority rules apply first."
    />

    <!-- Table -->
    <div class="bg-surface rounded-xl border border-border-base overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead class="bg-subtle border-b border-border-base">
            <tr>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider w-10">
                #
              </th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider">
                {{ $t('admin.conditions') }}
              </th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider">
                {{ $t('admin.destinations') }}
              </th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider hidden md:table-cell">
                Route
              </th>
              <th class="px-4 py-3 text-center text-xs font-semibold text-muted uppercase tracking-wider">
                {{ $t('common.active') }}
              </th>
              <th class="px-4 py-3 text-right text-xs font-semibold text-muted uppercase tracking-wider">
                {{ $t('common.actions') }}
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border-faint">
            <tr
              v-for="rule in localRules"
              :key="rule.id"
              class="hover:bg-subtle transition-colors"
            >
              <td class="px-4 py-3">
                <span class="flex items-center gap-1.5">
                  <UIcon name="i-lucide-grip-vertical" class="h-4 w-4 text-icon-faint" />
                  <span class="text-xs font-mono text-muted">{{ rule.priority }}</span>
                </span>
              </td>
              <td class="px-4 py-3">
                <div class="space-y-1">
                  <div class="flex items-center gap-1.5 flex-wrap">
                    <UBadge color="neutral" variant="soft" size="xs">
                      {{ rule.documentType }}
                    </UBadge>
                    <UBadge v-if="rule.urgency" color="warning" variant="soft" size="xs">
                      {{ rule.urgency }}
                    </UBadge>
                  </div>
                  <p v-if="rule.senderOrg" class="text-xs text-icon-disabled">
                    Sender: {{ rule.senderOrg }}
                  </p>
                </div>
              </td>
              <td class="px-4 py-3">
                <span class="text-sm font-medium text-body">{{ rule.destination }}</span>
              </td>
              <td class="px-4 py-3 hidden md:table-cell">
                <div class="flex items-center gap-1 flex-wrap">
                  <template
                    v-for="(stop, i) in rule.stops"
                    :key="stop + i"
                  >
                    <span class="text-xs text-muted">{{ stop }}</span>
                    <UIcon
                      v-if="i < rule.stops.length - 1"
                      name="i-lucide-arrow-right"
                      class="h-3 w-3 text-icon-faint"
                    />
                  </template>
                </div>
              </td>
              <td class="px-4 py-3 text-center">
                <span
                  class="inline-block h-2 w-2 rounded-full"
                  :class="rule.isActive ? 'bg-emerald-500' : 'bg-divider'"
                />
              </td>
              <td class="px-4 py-3 text-right">
                <div class="flex justify-end gap-1">
                  <UButton
                    icon="i-lucide-pencil"
                    color="neutral"
                    variant="ghost"
                    size="xs"
                    @click="openEditRule(rule)"
                  />
                  <UButton
                    icon="i-lucide-trash"
                    color="error"
                    variant="ghost"
                    size="xs"
                    @click="deleteRule(rule.id)"
                  />
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>

  <!-- New/Edit Rule Modal -->
  <UModal v-model:open="showRuleModal">
    <template #content>
      <div class="p-6 space-y-4 max-h-[80vh] overflow-y-auto">
        <h3 class="text-base font-semibold text-body">
          {{ isEditing ? 'Edit Rule' : 'New Routing Rule' }}
        </h3>

        <UFormField :label="$t('dispatch.documentType')" name="documentType" required>
          <USelect
            v-model="modalForm.documentType"
            :items="documentTypeOptions"
            :placeholder="$t('dispatch.documentType')"
            class="w-full"
          />
        </UFormField>

        <UFormField :label="$t('dispatch.senderOrg') + ' (optional)'" name="senderOrg">
          <UInput
            v-model="modalForm.senderOrg"
            placeholder="e.g. Ernst & Young"
            class="w-full"
          />
        </UFormField>

        <UFormField :label="$t('dispatch.priority') + ' (optional)'" name="urgency">
          <USelect
            v-model="modalForm.urgency"
            :items="urgencyOptions"
            placeholder="Any urgency"
            class="w-full"
          />
        </UFormField>

        <UFormField label="Destination Department" name="destination" required>
          <USelect
            v-model="modalForm.destination"
            :items="departmentOptions"
            class="w-full"
          />
        </UFormField>

        <div class="flex items-center gap-2">
          <input
            id="multiStop"
            v-model="modalForm.isMultiStop"
            type="checkbox"
            class="h-4 w-4 rounded border-border-muted text-accent"
          >
          <label for="multiStop" class="text-sm text-body">Multi-stop routing</label>
        </div>

        <div v-if="modalForm.isMultiStop" class="space-y-2">
          <p class="text-xs font-semibold text-muted uppercase tracking-wider">
            Routing Stops (in order)
          </p>
          <div
            v-for="(stop, i) in modalForm.stops"
            :key="i"
            class="flex items-center gap-2"
          >
            <UInput
              v-model="modalForm.stops[i]"
              :placeholder="`Stop ${i + 1}`"
              size="sm"
              class="flex-1"
            />
            <UButton
              v-if="i > 0"
              icon="i-lucide-x"
              color="neutral"
              variant="ghost"
              size="xs"
              @click="removeStop(i)"
            />
          </div>
          <UButton
            color="neutral"
            variant="ghost"
            size="xs"
            leading-icon="i-lucide-plus"
            @click="addStop"
          >
            {{ $t('common.add') }} stop
          </UButton>
        </div>

        <div class="flex gap-2 pt-2 border-t border-border-faint">
          <UButton
            color="primary"
            variant="solid"
            @click="saveRule"
          >
            {{ isEditing ? 'Save Changes' : $t('admin.createRule') }}
          </UButton>
          <UButton
            color="neutral"
            variant="outline"
            @click="showRuleModal = false"
          >
            {{ $t('common.cancel') }}
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>
