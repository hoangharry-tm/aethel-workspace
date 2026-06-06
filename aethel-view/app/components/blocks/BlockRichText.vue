<script setup lang="ts">
interface Props {
  title?: string
  content: string
  editable?: boolean
}

const props = defineProps<Props>()
const toast = useToast()

const editableContent = ref(props.content)

function handleSave() {
  toast.add({
    title: 'Content saved',
    color: 'success',
    icon: 'i-lucide-check',
  })
}
</script>

<template>
  <div class="bg-surface rounded-xl border border-border-base overflow-hidden">
    <div
      v-if="title"
      class="px-4 py-3 border-b border-border-faint"
    >
      <h3 class="text-sm font-semibold text-body">
        {{ title }}
      </h3>
    </div>

    <div class="p-4">
      <div v-if="editable" class="space-y-3">
        <textarea
          v-model="editableContent"
          class="w-full min-h-[160px] rounded-lg border border-border-base bg-subtle px-3 py-2 text-sm text-body font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent resize-y"
          placeholder="Enter HTML content..."
        />
        <UButton
          color="primary"
          variant="solid"
          size="sm"
          leading-icon="i-lucide-save"
          @click="handleSave"
        >
          {{ $t('common.save') }}
        </UButton>
      </div>

      <div
        v-else
        class="prose prose-sm prose-slate max-w-none text-body leading-relaxed"
        v-html="content"
      />
    </div>
  </div>
</template>
