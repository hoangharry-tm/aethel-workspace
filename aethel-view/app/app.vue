<script setup lang="ts">
const { config } = useAppRuntimeConfig()

useHead({
  style: [
    {
      innerHTML: computed(
        () => `:root {
  --color-text-accent: ${config.value.branding.primaryColor};
  --ui-primary: ${config.value.branding.primaryColor};
}`,
      ),
    },
  ],
})
</script>

<template>
  <NuxtRouteAnnouncer />
  <NuxtLayout>
    <NuxtPage :key="$route.path" />
  </NuxtLayout>
</template>
