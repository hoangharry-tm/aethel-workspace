<script setup lang="ts">
definePageMeta({ layout: 'workspace', middleware: ['role'], requiredRole: 'ADMIN' })
</script>

<template>
  <div class="space-y-6 max-w-3xl">
    <div>
      <h1 class="text-xl font-bold text-body">
        Document Types
      </h1>
      <p class="text-sm text-muted mt-0.5">
        Configure document categories and their properties
      </p>
    </div>
    <UAlert
      color="warning"
      variant="soft"
      icon="i-lucide-construction"
      title="This section is under construction"
      description="Document type management will be available in an upcoming release."
    />
  </div>
</template>
