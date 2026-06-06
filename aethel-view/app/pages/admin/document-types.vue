<script setup lang="ts">
definePageMeta({ layout: 'workspace', middleware: ['role'], requiredRole: 'ADMIN' })

interface DocType {
  id: string
  name: string
  description: string
  icon: string
  docCount: number
  active: boolean
}

const toast = useToast()

const docTypes = ref<DocType[]>([
  { id: '1', name: 'Inbound Correspondence', description: 'Letters and parcels received from external senders', icon: 'i-lucide-mail', docCount: 41, active: true },
  { id: '2', name: 'Internal Memo', description: 'Internal communications between departments', icon: 'i-lucide-clipboard', docCount: 28, active: true },
  { id: '3', name: 'Contract', description: 'Legal agreements and binding documents', icon: 'i-lucide-file-text', docCount: 15, active: true },
  { id: '4', name: 'Report', description: 'Analytical and audit reports submitted to management', icon: 'i-lucide-bar-chart-2', docCount: 9, active: true },
  { id: '5', name: 'Invoice', description: 'Financial billing documents from vendors and suppliers', icon: 'i-lucide-receipt', docCount: 33, active: false },
  { id: '6', name: 'Regulation Notice', description: 'Official directives from regulatory authorities', icon: 'i-lucide-scroll', docCount: 6, active: true },
])

const iconOptions = [
  { label: 'Mail', value: 'i-lucide-mail' },
  { label: 'Clipboard', value: 'i-lucide-clipboard' },
  { label: 'File Text', value: 'i-lucide-file-text' },
  { label: 'Bar Chart', value: 'i-lucide-bar-chart-2' },
  { label: 'Receipt', value: 'i-lucide-receipt' },
  { label: 'Scroll', value: 'i-lucide-scroll' },
  { label: 'Book Open', value: 'i-lucide-book-open' },
  { label: 'Folder', value: 'i-lucide-folder' },
]

const showDocTypeModal = ref(false)
const showDeleteModal = ref(false)
const isEditingDocType = ref(false)
const selectedDocType = ref<DocType | null>(null)
const docTypeForm = reactive({ name: '', description: '', icon: 'i-lucide-file-text', active: true })

function handleOpenAddModal() {
  isEditingDocType.value = false
  selectedDocType.value = null
  Object.assign(docTypeForm, { name: '', description: '', icon: 'i-lucide-file-text', active: true })
  showDocTypeModal.value = true
}

function handleOpenEditModal(type: DocType) {
  isEditingDocType.value = true
  selectedDocType.value = type
  Object.assign(docTypeForm, { name: type.name, description: type.description, icon: type.icon, active: type.active })
  showDocTypeModal.value = true
}

function handleOpenDeleteModal(type: DocType) {
  selectedDocType.value = type
  showDeleteModal.value = true
}

function handleSaveDocType() {
  if (isEditingDocType.value && selectedDocType.value) {
    const idx = docTypes.value.findIndex(d => d.id === selectedDocType.value!.id)
    const target = idx !== -1 ? docTypes.value[idx] : undefined
    if (target) Object.assign(target, docTypeForm)
    toast.add({ title: 'Updated', description: `"${docTypeForm.name}" updated.`, color: 'success', icon: 'i-lucide-check-circle' })
  }
  else {
    docTypes.value.push({ id: String(Date.now()), ...docTypeForm, docCount: 0 })
    toast.add({ title: 'Created', description: `"${docTypeForm.name}" created.`, color: 'success', icon: 'i-lucide-check-circle' })
  }
  showDocTypeModal.value = false
}

function handleDeleteConfirm() {
  if (!selectedDocType.value) return
  docTypes.value = docTypes.value.filter(d => d.id !== selectedDocType.value!.id)
  toast.add({ title: 'Deleted', description: `"${selectedDocType.value.name}" deleted.`, color: 'success', icon: 'i-lucide-check-circle' })
  showDeleteModal.value = false
  selectedDocType.value = null
}
</script>

<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between gap-4 flex-wrap">
      <h1 class="text-xl font-bold text-body">
        {{ $t('admin.documentTypes') }}
      </h1>
      <UButton
        color="primary"
        variant="solid"
        leading-icon="i-lucide-plus"
        @click="handleOpenAddModal"
      >
        {{ $t('admin.createDocType') }}
      </UButton>
    </div>

    <!-- Card grid -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <UCard
        v-for="type in docTypes"
        :key="type.id"
      >
        <div class="flex items-start justify-between gap-4">
          <div class="flex items-center gap-3">
            <div class="flex h-10 w-10 items-center justify-center rounded-full bg-accent/10 flex-shrink-0">
              <UIcon :name="type.icon" class="h-5 w-5 text-accent" />
            </div>
            <div>
              <p class="font-semibold text-body">
                {{ type.name }}
              </p>
              <p class="text-sm text-muted mt-0.5">
                {{ type.description }}
              </p>
            </div>
          </div>
          <UBadge
            :color="type.active ? 'success' : 'neutral'"
            variant="soft"
            size="xs"
          >
            {{ type.active ? $t('common.active') : $t('common.inactive') }}
          </UBadge>
        </div>

        <USeparator class="my-3" />

        <div class="flex items-center justify-between">
          <UBadge color="neutral" variant="soft" size="sm">
            {{ type.docCount }} documents
          </UBadge>
          <div class="flex gap-1">
            <UButton
              size="xs"
              color="neutral"
              variant="ghost"
              icon="i-lucide-pencil"
              @click="handleOpenEditModal(type)"
            />
            <UButton
              size="xs"
              color="error"
              variant="ghost"
              icon="i-lucide-trash-2"
              @click="handleOpenDeleteModal(type)"
            />
          </div>
        </div>
      </UCard>
    </div>
  </div>

  <!-- Add / Edit Modal -->
  <UModal v-model:open="showDocTypeModal">
    <template #content>
      <div class="p-6 space-y-4">
        <h3 class="text-base font-semibold text-body">
          {{ isEditingDocType ? $t('admin.documentTypes') : $t('admin.createDocType') }}
        </h3>

        <UFormField :label="$t('common.name')" required>
          <UInput
            v-model="docTypeForm.name"
            placeholder="e.g. Internal Memo"
            class="w-full"
          />
        </UFormField>

        <UFormField :label="$t('common.description')">
          <UTextarea
            v-model="docTypeForm.description"
            :rows="3"
            class="w-full"
          />
        </UFormField>

        <UFormField label="Icon">
          <USelect
            v-model="docTypeForm.icon"
            :items="iconOptions"
            class="w-full"
          />
        </UFormField>

        <UFormField :label="$t('common.active')">
          <UToggle v-model="docTypeForm.active" />
        </UFormField>

        <div class="flex gap-2 pt-2">
          <UButton color="primary" @click="handleSaveDocType">
            {{ $t('common.save') }}
          </UButton>
          <UButton color="neutral" variant="outline" @click="showDocTypeModal = false">
            {{ $t('common.cancel') }}
          </UButton>
        </div>
      </div>
    </template>
  </UModal>

  <!-- Delete Confirmation Modal -->
  <UModal v-model:open="showDeleteModal">
    <template #content>
      <div class="p-6 space-y-4">
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 items-center justify-center rounded-full bg-error/10 flex-shrink-0">
            <UIcon name="i-lucide-trash-2" class="h-5 w-5 text-error" />
          </div>
          <div>
            <h3 class="text-base font-semibold text-body">
              {{ $t('common.delete') }} Document Type
            </h3>
            <p class="text-sm text-muted">
              {{ selectedDocType?.name }}
            </p>
          </div>
        </div>
        <p class="text-sm text-muted">
          This will permanently delete the document type. Dispatches using it will not be affected.
        </p>
        <div class="flex gap-2 pt-2">
          <UButton color="error" @click="handleDeleteConfirm">
            {{ $t('common.delete') }}
          </UButton>
          <UButton color="neutral" variant="outline" @click="showDeleteModal = false">
            {{ $t('common.cancel') }}
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>
