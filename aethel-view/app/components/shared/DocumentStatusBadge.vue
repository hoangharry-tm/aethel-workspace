<script setup lang="ts">
interface Props {
  status: string
}

const props = defineProps<Props>()
const { t } = useI18n()

type BadgeColor = 'neutral' | 'primary' | 'secondary' | 'info' | 'warning' | 'success' | 'error'

const statusConfig = computed<Record<string, { color: BadgeColor, icon: string }>>(() => ({
  PENDING_ASSIGNMENT: { color: 'neutral', icon: 'i-lucide-clock' },
  UNDER_REVIEW: { color: 'primary', icon: 'i-lucide-eye' },
  IN_TRANSIT: { color: 'info', icon: 'i-lucide-truck' },
  ATTEMPTED_DELIVERY: { color: 'warning', icon: 'i-lucide-alert-circle' },
  DELIVERED: { color: 'success', icon: 'i-lucide-check-circle' },
  ESCALATED: { color: 'error', icon: 'i-lucide-bell-ring' },
  DISPATCHED: { color: 'secondary', icon: 'i-lucide-send' },
}))

const config = computed<{ color: BadgeColor, icon: string, label: string }>(() => {
  const cfg = statusConfig.value[props.status]
  if (cfg) {
    return { ...cfg, label: t(`docStatus.${props.status}`) }
  }
  return { color: 'neutral', icon: 'i-lucide-file', label: props.status }
})
</script>

<template>
  <UBadge
    :color="config.color"
    variant="soft"
    :leading-icon="config.icon"
    size="sm"
  >
    {{ config.label }}
  </UBadge>
</template>
