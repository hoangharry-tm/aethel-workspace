<script setup lang="ts">
definePageMeta({ layout: 'workspace' })

const { t } = useI18n()
const router = useRouter()
const toast = useToast()

const form = reactive({
  senderName: '',
  senderOrg: '',
  documentType: '',
  dateReceived: new Date().toISOString().split('T')[0] ?? '',
  deliveryMode: 'COURIER',
  subject: '',
  urgency: 'ROUTINE',
  department: '',
})

const loading = ref(false)

const documentTypeOptions = [
  { label: 'Audit Report', value: 'Audit Report' },
  { label: 'Legal Contract', value: 'Legal Contract' },
  { label: 'Invoice', value: 'Invoice' },
  { label: 'Regulatory Notice', value: 'Regulatory Notice' },
  { label: 'Budget Proposal', value: 'Budget Proposal' },
  { label: 'Certificate', value: 'Certificate' },
  { label: 'License Agreement', value: 'License Agreement' },
  { label: 'MOU', value: 'MOU' },
  { label: 'Report', value: 'Report' },
  { label: 'General Correspondence', value: 'General Correspondence' },
]

const deliveryModeOptions = [
  { label: 'Courier', value: 'COURIER' },
  { label: 'Post', value: 'POST' },
  { label: 'Hand Delivery', value: 'HAND_DELIVERY' },
  { label: 'Email', value: 'EMAIL' },
]

const urgencyOptions = [
  { label: 'Routine', value: 'ROUTINE' },
  { label: 'Priority', value: 'PRIORITY' },
  { label: 'Immediate', value: 'IMMEDIATE' },
]

const departmentOptions = [
  { label: 'Finance', value: 'Finance' },
  { label: 'Legal', value: 'Legal' },
  { label: 'HR', value: 'HR' },
  { label: 'Operations', value: 'Operations' },
  { label: 'Procurement', value: 'Procurement' },
  { label: 'Administration', value: 'Administration' },
  { label: 'IT', value: 'IT' },
]

async function handleSubmit() {
  loading.value = true
  await new Promise(resolve => setTimeout(resolve, 900))
  const trackingNum = `AWK-2025-${String(Math.floor(1000 + Math.random() * 9000))}`
  toast.add({
    title: 'Document logged successfully',
    description: `Tracking #${trackingNum} generated.`,
    color: 'success',
    icon: 'i-lucide-check-circle',
  })
  loading.value = false
  await router.push('/documents/doc-001')
}
</script>

<template>
  <div class="space-y-6 max-w-5xl">
    <!-- Header -->
    <div class="flex items-center gap-3">
      <UButton
        to="/dispatch/inbound"
        icon="i-lucide-arrow-left"
        color="neutral"
        variant="ghost"
        size="sm"
      />
      <div>
        <h1 class="text-xl font-bold text-body">
          Log Incoming Document
        </h1>
        <p class="text-sm text-muted mt-0.5">
          Record a new document received at reception
        </p>
      </div>
    </div>

    <form @submit.prevent="handleSubmit">
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <!-- Left column -->
        <div class="bg-surface rounded-xl border border-border-base p-6 space-y-5">
          <h2 class="text-sm font-semibold text-body border-b border-border-faint pb-3">
            Document Information
          </h2>

          <UFormField :label="$t('dispatch.sender') + ' Name'" name="senderName" required>
            <UInput
              v-model="form.senderName"
              placeholder="e.g. Ernst & Young LLP"
              class="w-full"
            />
          </UFormField>

          <UFormField :label="$t('dispatch.senderOrg')" name="senderOrg" required>
            <UInput
              v-model="form.senderOrg"
              placeholder="e.g. Ernst & Young"
              class="w-full"
            />
          </UFormField>

          <UFormField :label="$t('dispatch.documentType')" name="documentType" required>
            <USelect
              v-model="form.documentType"
              :items="documentTypeOptions"
              :placeholder="$t('dispatch.documentType')"
              class="w-full"
            />
          </UFormField>

          <UFormField label="Date Received" name="dateReceived" required>
            <UInput
              v-model="form.dateReceived"
              type="date"
              class="w-full"
            />
          </UFormField>

          <UFormField :label="$t('dispatch.deliveryMode')" name="deliveryMode" required>
            <USelect
              v-model="form.deliveryMode"
              :items="deliveryModeOptions"
              class="w-full"
            />
          </UFormField>

          <UFormField label="Subject / Description" name="subject" required>
            <UTextarea
              v-model="form.subject"
              placeholder="Brief description of the document contents..."
              :rows="3"
              class="w-full"
            />
          </UFormField>

          <UFormField label="Urgency Level" name="urgency">
            <USelect
              v-model="form.urgency"
              :items="urgencyOptions"
              class="w-full"
            />
          </UFormField>
        </div>

        <!-- Right column -->
        <div class="space-y-6">
          <!-- Routing -->
          <div class="bg-surface rounded-xl border border-border-base p-6 space-y-5">
            <h2 class="text-sm font-semibold text-body border-b border-border-faint pb-3">
              Routing
            </h2>

            <UFormField
              :label="$t('dispatch.department') + ' / Recipient'"
              name="department"
              hint="Auto-suggested based on document type. You may override."
            >
              <USelect
                v-model="form.department"
                :items="departmentOptions"
                :placeholder="$t('dispatch.assignDepartment')"
                class="w-full"
              />
            </UFormField>

            <div class="rounded-lg bg-accent/5 border border-accent/20 px-3 py-2">
              <p class="text-xs text-accent flex items-start gap-1.5">
                <UIcon name="i-lucide-info" class="h-3.5 w-3.5 mt-0.5 flex-shrink-0" />
                Routing rules will auto-assign the department based on document type. You can override this selection.
              </p>
            </div>
          </div>

          <!-- File attachment -->
          <div class="bg-surface rounded-xl border border-border-base p-6 space-y-3">
            <h2 class="text-sm font-semibold text-body border-b border-border-faint pb-3">
              {{ $t('document.attachments') }}
            </h2>
            <div class="border-2 border-dashed border-border-base rounded-lg p-8 flex flex-col items-center justify-center text-center hover:border-accent/50 hover:bg-accent/5 transition-colors cursor-pointer">
              <UIcon name="i-lucide-upload-cloud" class="h-10 w-10 text-icon-faint mb-3" />
              <p class="text-sm font-medium text-muted">
                Drag PDF here or click to browse
              </p>
              <p class="text-xs text-icon-disabled mt-1">
                Maximum file size: 25MB
              </p>
            </div>
          </div>
        </div>
      </div>

      <!-- Actions -->
      <div class="flex items-center gap-3 mt-6">
        <UButton
          type="submit"
          color="primary"
          variant="solid"
          :loading="loading"
          leading-icon="i-lucide-file-plus"
        >
          Log Document
        </UButton>
        <UButton
          type="button"
          color="neutral"
          variant="outline"
          :to="'/dispatch/inbound'"
        >
          {{ $t('common.cancel') }}
        </UButton>
      </div>
    </form>
  </div>
</template>
