<script setup lang="ts">
import UrgencyBadge from '~/components/shared/UrgencyBadge.vue'
import type { UrgencyLevel } from '~/composables/useMockData'

definePageMeta({ layout: 'workspace', middleware: ['role'], requiredRole: 'ADMIN' })

interface EscalationRule {
  id: string
  name: string
  threshold: number
  action: string
  priorityFilter: string
  active: boolean
}

const toast = useToast()

const escalationRules = ref<EscalationRule[]>([
  { id: '1', name: 'IMMEDIATE — 2-Hour Alert', threshold: 2, action: 'NOTIFY', priorityFilter: 'IMMEDIATE', active: true },
  { id: '2', name: 'IMMEDIATE — 4-Hour Reassign', threshold: 4, action: 'REASSIGN', priorityFilter: 'IMMEDIATE', active: true },
  { id: '3', name: 'PRIORITY — 24-Hour Alert', threshold: 24, action: 'NOTIFY', priorityFilter: 'PRIORITY', active: true },
  { id: '4', name: 'ALL — 72-Hour Escalation', threshold: 72, action: 'ESCALATE_STATUS', priorityFilter: 'ALL', active: false },
  { id: '5', name: 'ROUTINE — 5-Day Reminder', threshold: 120, action: 'NOTIFY', priorityFilter: 'ROUTINE', active: true },
])

const actionOptions = [
  { label: 'Reassign to Supervisor', value: 'REASSIGN' },
  { label: 'Send Notification', value: 'NOTIFY' },
  { label: 'Escalate Status', value: 'ESCALATE_STATUS' },
]

const priorityOptions = [
  { label: 'All Priorities', value: 'ALL' },
  { label: 'Immediate Only', value: 'IMMEDIATE' },
  { label: 'Priority Only', value: 'PRIORITY' },
  { label: 'Routine Only', value: 'ROUTINE' },
]

const actionLabel: Record<string, string> = {
  REASSIGN: 'Reassign to Supervisor',
  NOTIFY: 'Send Notification',
  ESCALATE_STATUS: 'Escalate Status',
}

const actionIcon: Record<string, string> = {
  REASSIGN: 'i-lucide-user-check',
  NOTIFY: 'i-lucide-bell',
  ESCALATE_STATUS: 'i-lucide-arrow-up-circle',
}

const showRuleModal = ref(false)
const isEditingRule = ref(false)
const selectedRule = ref<EscalationRule | null>(null)
const ruleForm = reactive({ name: '', threshold: 4, action: 'NOTIFY', priorityFilter: 'ALL', active: true })

function isUrgencyLevel(value: string): value is UrgencyLevel {
  return value === 'IMMEDIATE' || value === 'PRIORITY' || value === 'ROUTINE'
}

function handleToggleRule(rule: EscalationRule) {
  rule.active = !rule.active
  toast.add({ title: rule.active ? 'Rule enabled' : 'Rule disabled', color: rule.active ? 'success' : 'neutral', icon: 'i-lucide-toggle-right' })
}

function handleOpenAddRule() {
  isEditingRule.value = false
  selectedRule.value = null
  Object.assign(ruleForm, { name: '', threshold: 4, action: 'NOTIFY', priorityFilter: 'ALL', active: true })
  showRuleModal.value = true
}

function handleOpenEditRule(rule: EscalationRule) {
  isEditingRule.value = true
  selectedRule.value = rule
  Object.assign(ruleForm, { name: rule.name, threshold: rule.threshold, action: rule.action, priorityFilter: rule.priorityFilter, active: rule.active })
  showRuleModal.value = true
}

function handleSaveRule() {
  if (isEditingRule.value && selectedRule.value) {
    Object.assign(selectedRule.value, ruleForm)
    toast.add({ title: 'Rule updated', color: 'success', icon: 'i-lucide-check-circle' })
  }
  else {
    escalationRules.value.push({ id: String(Date.now()), ...ruleForm })
    toast.add({ title: 'Rule created', color: 'success', icon: 'i-lucide-check-circle' })
  }
  showRuleModal.value = false
}

function handleDeleteRule(rule: EscalationRule) {
  escalationRules.value = escalationRules.value.filter(r => r.id !== rule.id)
  toast.add({ title: 'Rule deleted', color: 'success', icon: 'i-lucide-check-circle' })
}
</script>

<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-start justify-between gap-4 flex-wrap">
      <div>
        <h1 class="text-xl font-bold text-body">
          Escalation Rules
        </h1>
        <p class="text-sm text-muted mt-0.5">
          Automated alerts triggered when documents exceed SLA thresholds
        </p>
      </div>
      <UButton
        color="primary"
        variant="solid"
        leading-icon="i-lucide-plus"
        @click="handleOpenAddRule"
      >
        Add Rule
      </UButton>
    </div>

    <!-- Rules table -->
    <UCard>
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead class="bg-subtle border-b border-border-base">
            <tr>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider">
                Rule Name
              </th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider">
                Trigger (hours)
              </th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider hidden md:table-cell">
                Action
              </th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider hidden sm:table-cell">
                Priority Filter
              </th>
              <th class="px-4 py-3 text-center text-xs font-semibold text-muted uppercase tracking-wider">
                Status
              </th>
              <th class="px-4 py-3 text-right text-xs font-semibold text-muted uppercase tracking-wider">
                &nbsp;
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border-faint">
            <tr
              v-for="rule in escalationRules"
              :key="rule.id"
              class="hover:bg-subtle transition-colors"
            >
              <td class="px-4 py-3">
                <span class="text-sm font-medium text-body">{{ rule.name }}</span>
              </td>
              <td class="px-4 py-3">
                <UBadge color="neutral" variant="soft" size="sm">
                  {{ rule.threshold }}h
                </UBadge>
              </td>
              <td class="px-4 py-3 hidden md:table-cell">
                <div class="flex items-center gap-1.5">
                  <UIcon :name="actionIcon[rule.action] ?? 'i-lucide-zap'" class="h-4 w-4 text-muted" />
                  <span class="text-sm text-body">{{ actionLabel[rule.action] }}</span>
                </div>
              </td>
              <td class="px-4 py-3 hidden sm:table-cell">
                <UrgencyBadge
                  v-if="isUrgencyLevel(rule.priorityFilter)"
                  :level="rule.priorityFilter as UrgencyLevel"
                />
                <UBadge
                  v-else
                  color="neutral"
                  variant="soft"
                >
                  All Priorities
                </UBadge>
              </td>
              <td class="px-4 py-3 text-center">
                <UToggle
                  :model-value="rule.active"
                  @update:model-value="handleToggleRule(rule)"
                />
              </td>
              <td class="px-4 py-3 text-right">
                <div class="flex justify-end gap-1">
                  <UButton
                    size="xs"
                    color="neutral"
                    variant="ghost"
                    icon="i-lucide-pencil"
                    @click="handleOpenEditRule(rule)"
                  />
                  <UButton
                    size="xs"
                    color="error"
                    variant="ghost"
                    icon="i-lucide-trash-2"
                    @click="handleDeleteRule(rule)"
                  />
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </UCard>
  </div>

  <!-- Add / Edit Rule Modal -->
  <UModal v-model:open="showRuleModal">
    <template #content>
      <div class="p-6 space-y-4">
        <h3 class="text-base font-semibold text-body">
          {{ isEditingRule ? 'Edit Rule' : 'Add Escalation Rule' }}
        </h3>

        <UFormField label="Rule Name" required>
          <UInput
            v-model="ruleForm.name"
            placeholder="e.g. IMMEDIATE 4-Hour Alert"
            class="w-full"
          />
        </UFormField>

        <UFormField label="Overdue threshold (hours)" required>
          <UInput
            v-model.number="ruleForm.threshold"
            type="number"
            placeholder="4"
            class="w-full"
          />
        </UFormField>

        <UFormField label="Action" required>
          <USelect
            v-model="ruleForm.action"
            :items="actionOptions"
            class="w-full"
          />
        </UFormField>

        <UFormField label="Priority Filter">
          <USelect
            v-model="ruleForm.priorityFilter"
            :items="priorityOptions"
            class="w-full"
          />
        </UFormField>

        <UFormField label="Active">
          <UToggle v-model="ruleForm.active" />
        </UFormField>

        <div class="flex gap-2 pt-2">
          <UButton color="primary" @click="handleSaveRule">
            Save Rule
          </UButton>
          <UButton color="neutral" variant="outline" @click="showRuleModal = false">
            Cancel
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>
