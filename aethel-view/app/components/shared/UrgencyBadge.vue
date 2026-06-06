<script setup lang="ts">
import type { UrgencyLevel } from '~/composables/useMockData'

interface Props {
  level: UrgencyLevel
}

const props = defineProps<Props>()
const { t } = useI18n()

const config = computed(() => {
  switch (props.level) {
    case 'IMMEDIATE':
      return { color: 'error' as const, icon: 'i-lucide-zap' }
    case 'PRIORITY':
      return { color: 'warning' as const, icon: 'i-lucide-alert-triangle' }
    case 'ROUTINE':
      return { color: 'success' as const, icon: 'i-lucide-minus' }
  }
})
</script>

<template>
  <UBadge
    :color="config.color"
    variant="soft"
    :leading-icon="config.icon"
    size="sm"
  >
    {{ t(`urgency.${level}`) }}
  </UBadge>
</template>
