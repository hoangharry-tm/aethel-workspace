<template>
  <div
    class="min-h-screen bg-subtle flex flex-col items-center justify-center px-4 py-12"
  >
    <!-- Brand header -->
    <div class="flex items-center gap-2.5 mb-8">
      <div
        class="flex h-10 w-10 items-center justify-center rounded-xl bg-accent shadow-sm"
      >
        <UIcon name="i-lucide-building-2" class="h-6 w-6 text-white" />
      </div>
      <span class="text-xl font-bold text-body">Aethel Workspace</span>
    </div>

    <!-- Card slot -->
    <div
      class="w-full max-w-md bg-surface rounded-2xl shadow-sm border border-border-base overflow-hidden"
    >
      <slot />
    </div>

    <!-- Footer -->
    <p class="mt-6 text-xs text-icon-disabled">
      &copy; {{ new Date().getFullYear() }} Aethel Workspace by
      <a
        href="https://www.linkedin.com/in/minh-hoang-ton"
        target="_blank"
        rel="noopener noreferrer"
        class="underline decoration-dotted"
        >Minh Hoang Ton</a
      >. All rights reserved.
    </p>
  </div>
</template>
