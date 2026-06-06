<script setup lang="ts">
interface FieldDef {
  id: string
  label: string
  type: 'text' | 'select' | 'date' | 'textarea'
  options?: string[]
  required?: boolean
}

interface Props {
  title: string
  fields: FieldDef[]
}

defineProps<Props>()

const toast = useToast()
const formValues = ref<Record<string, string>>({})

function handleSubmit() {
  toast.add({
    title: 'Form submitted',
    color: 'success',
    icon: 'i-lucide-check',
  })
}

function selectItems(options: string[] = []) {
  return options.map(o => ({ label: o, value: o }))
}
</script>

<template>
  <div class="bg-surface rounded-xl border border-border-base overflow-hidden">
    <div class="px-4 py-3 border-b border-border-faint">
      <h3 class="text-sm font-semibold text-body">
        {{ title }}
      </h3>
    </div>

    <div class="p-4">
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <UFormField
          v-for="field in fields"
          :key="field.id"
          :label="field.label"
          :name="field.id"
          :required="field.required"
          :class="field.type === 'textarea' ? 'md:col-span-2' : ''"
        >
          <UTextarea
            v-if="field.type === 'textarea'"
            v-model="formValues[field.id]"
            :placeholder="`Enter ${field.label.toLowerCase()}...`"
            :rows="3"
            class="w-full"
          />
          <USelect
            v-else-if="field.type === 'select'"
            v-model="formValues[field.id]"
            :items="selectItems(field.options)"
            :placeholder="`Select ${field.label.toLowerCase()}`"
            class="w-full"
          />
          <UInput
            v-else
            v-model="formValues[field.id]"
            :type="field.type"
            :placeholder="`Enter ${field.label.toLowerCase()}...`"
            class="w-full"
          />
        </UFormField>
      </div>

      <div class="mt-4 pt-4 border-t border-border-faint">
        <UButton
          color="primary"
          variant="solid"
          leading-icon="i-lucide-send"
          @click="handleSubmit"
        >
          {{ $t('common.submit') }}
        </UButton>
      </div>
    </div>
  </div>
</template>
