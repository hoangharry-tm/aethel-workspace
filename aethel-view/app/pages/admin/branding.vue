<script setup lang="ts">
import { useAppRuntimeConfig } from '~/composables/useRuntimeConfig'

definePageMeta({ layout: 'workspace', middleware: ['role'], requiredRole: 'ADMIN' })

const { config, updateBranding } = useAppRuntimeConfig()
const toast = useToast()

const form = reactive({
  primaryColor: config.value.branding.primaryColor,
  neutralPalette: config.value.branding.neutralPalette,
  fontFamily: config.value.branding.fontFamily,
  wordmark: config.value.branding.wordmark,
  logoFileName: '',
})

const neutralOptions = [
  { label: 'Slate', value: 'slate' },
  { label: 'Zinc', value: 'zinc' },
  { label: 'Gray', value: 'gray' },
  { label: 'Stone', value: 'stone' },
  { label: 'Neutral', value: 'neutral' },
]

const fontOptions = [
  { label: 'Inter', value: 'Inter' },
  { label: 'Roboto', value: 'Roboto' },
  { label: 'Open Sans', value: 'Open Sans' },
  { label: 'Nunito', value: 'Nunito' },
  { label: 'Plus Jakarta Sans', value: 'Plus Jakarta Sans' },
]

const hexInput = ref(form.primaryColor)

function onColorInput(event: Event) {
  form.primaryColor = (event.target as HTMLInputElement).value
  hexInput.value = form.primaryColor
}

function onHexInput(val: string) {
  if (/^#[0-9A-Fa-f]{6}$/.test(val)) {
    form.primaryColor = val
  }
}

function handleLogoSelect(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (file) {
    form.logoFileName = file.name
  }
}

function saveBranding() {
  updateBranding({
    primaryColor: form.primaryColor,
    neutralPalette: form.neutralPalette,
    fontFamily: form.fontFamily,
    wordmark: form.wordmark,
  })
  toast.add({
    title: 'Branding saved',
    description: 'Changes have been applied organization-wide.',
    color: 'success',
    icon: 'i-lucide-check',
  })
}

// Mini preview fake nav items
const previewNavItems = ['Dashboard', 'Inbound', 'Outbound']
</script>

<template>
  <div class="space-y-6">
    <!-- Header -->
    <div>
      <h1 class="text-xl font-bold text-body">
        Branding
      </h1>
      <p class="text-sm text-muted mt-0.5">
        Customize workspace appearance, logo, and color scheme
      </p>
    </div>

    <!-- Two-column layout -->
    <div class="grid grid-cols-1 xl:grid-cols-2 gap-6 items-start">
      <!-- Left column: controls -->
      <div class="space-y-5">
        <!-- Brand Colors -->
        <div class="bg-surface rounded-xl border border-border-base overflow-hidden">
          <div class="px-4 py-3 border-b border-border-faint">
            <h2 class="text-sm font-semibold text-body">
              Brand Colors
            </h2>
          </div>
          <div class="p-4 space-y-4">
            <!-- Primary Color -->
            <div class="space-y-1.5">
              <label class="text-sm font-medium text-body">Primary Color</label>
              <div class="flex items-center gap-2">
                <!-- Hidden native color input -->
                <input
                  id="color-primary"
                  type="color"
                  :value="form.primaryColor"
                  class="sr-only"
                  @input="onColorInput"
                >
                <label
                  for="color-primary"
                  :style="{ background: form.primaryColor }"
                  class="w-8 h-8 rounded-md cursor-pointer border border-border-base inline-block flex-shrink-0 shadow-sm"
                  title="Click to pick color"
                />
                <UInput
                  :model-value="hexInput"
                  size="sm"
                  placeholder="#4f46e5"
                  class="w-32 font-mono"
                  @update:model-value="onHexInput"
                />
                <span class="text-xs text-icon-disabled">Hex value</span>
              </div>
            </div>

            <!-- Neutral Palette -->
            <UFormField label="Neutral Palette" name="neutralPalette">
              <USelect
                v-model="form.neutralPalette"
                :items="neutralOptions"
                class="w-full"
              />
            </UFormField>
          </div>
        </div>

        <!-- Typography -->
        <div class="bg-surface rounded-xl border border-border-base overflow-hidden">
          <div class="px-4 py-3 border-b border-border-faint">
            <h2 class="text-sm font-semibold text-body">
              Typography
            </h2>
          </div>
          <div class="p-4">
            <UFormField label="Font Family" name="fontFamily">
              <USelect
                v-model="form.fontFamily"
                :items="fontOptions"
                class="w-full"
              />
            </UFormField>
          </div>
        </div>

        <!-- Identity -->
        <div class="bg-surface rounded-xl border border-border-base overflow-hidden">
          <div class="px-4 py-3 border-b border-border-faint">
            <h2 class="text-sm font-semibold text-body">
              Identity
            </h2>
          </div>
          <div class="p-4 space-y-4">
            <UFormField label="Workspace Name" name="wordmark">
              <UInput
                v-model="form.wordmark"
                placeholder="Aethel Workspace"
                class="w-full"
              />
            </UFormField>

            <!-- Logo upload -->
            <div class="space-y-1.5">
              <label class="text-sm font-medium text-body">Logo</label>
              <div class="flex items-center gap-2">
                <input
                  id="logo-upload"
                  type="file"
                  accept="image/*"
                  class="sr-only"
                  @change="handleLogoSelect"
                >
                <label for="logo-upload">
                  <UButton
                    as="span"
                    variant="outline"
                    color="neutral"
                    leading-icon="i-lucide-upload"
                    class="cursor-pointer"
                  >
                    Choose file
                  </UButton>
                </label>
                <span class="text-sm text-muted truncate max-w-[200px]">
                  {{ form.logoFileName || 'No file chosen' }}
                </span>
              </div>
              <p class="text-xs text-icon-disabled">
                Recommended: SVG or PNG, 200x40px, transparent background
              </p>
            </div>
          </div>
        </div>

        <!-- Save button -->
        <UButton
          color="primary"
          variant="solid"
          leading-icon="i-lucide-save"
          class="w-full"
          @click="saveBranding"
        >
          Save Branding
        </UButton>
      </div>

      <!-- Right column: live preview -->
      <div class="space-y-3">
        <div class="bg-surface rounded-xl border border-border-base overflow-hidden">
          <div class="px-4 py-3 border-b border-border-faint">
            <h2 class="text-sm font-semibold text-body">
              Live Preview
            </h2>
          </div>

          <!-- Mini workspace shell -->
          <div class="p-4">
            <div class="rounded-lg border border-border-base overflow-hidden flex h-52 shadow-sm">
              <!-- Mini sidebar -->
              <div class="w-36 bg-surface border-r border-border-base flex flex-col flex-shrink-0">
                <!-- Brand row -->
                <div class="flex items-center gap-1.5 px-2 py-2 border-b border-border-faint">
                  <div
                    class="flex h-5 w-5 items-center justify-center rounded flex-shrink-0"
                    :style="{ background: form.primaryColor }"
                  >
                    <UIcon name="i-lucide-building-2" class="h-3 w-3 text-white" />
                  </div>
                  <span class="text-[10px] font-bold text-body truncate">
                    {{ form.wordmark || 'Aethel Workspace' }}
                  </span>
                </div>
                <!-- Nav items -->
                <div class="flex-1 px-1.5 py-2 space-y-0.5">
                  <div
                    v-for="(item, i) in previewNavItems"
                    :key="item"
                    class="flex items-center gap-1.5 rounded px-1.5 py-1 text-[10px] font-medium"
                    :class="i === 0 ? 'text-white' : 'text-muted'"
                    :style="i === 0 ? { background: form.primaryColor } : {}"
                  >
                    <UIcon name="i-lucide-layout-dashboard" class="h-3 w-3 flex-shrink-0" />
                    <span class="truncate">{{ item }}</span>
                  </div>
                </div>
              </div>

              <!-- Mini main area -->
              <div class="flex-1 bg-subtle flex flex-col">
                <!-- Mini topbar -->
                <div class="flex items-center justify-between px-3 py-2 bg-surface border-b border-border-faint">
                  <span class="text-[10px] font-semibold text-body">
                    {{ form.wordmark || 'Aethel Workspace' }}
                  </span>
                  <div
                    class="h-5 w-5 rounded-full flex items-center justify-center text-white text-[8px] font-bold"
                    :style="{ background: form.primaryColor }"
                  >
                    AW
                  </div>
                </div>

                <!-- Mini content area -->
                <div class="flex-1 p-3 space-y-2">
                  <div class="h-2 w-24 rounded bg-divider" />
                  <div class="h-10 rounded bg-surface border border-border-base" />
                  <div class="grid grid-cols-2 gap-1.5">
                    <div class="h-8 rounded bg-surface border border-border-base" />
                    <div class="h-8 rounded bg-surface border border-border-base" />
                  </div>
                </div>
              </div>
            </div>

            <p class="mt-3 text-xs text-muted text-center">
              Changes are applied organization-wide immediately after saving.
            </p>
          </div>
        </div>

        <!-- Font preview card -->
        <div class="bg-surface rounded-xl border border-border-base p-4 space-y-1.5">
          <p class="text-xs font-semibold text-muted uppercase tracking-wider">
            Font preview
          </p>
          <p
            class="text-base font-bold text-body"
            :style="{ fontFamily: form.fontFamily }"
          >
            {{ form.wordmark || 'Aethel Workspace' }}
          </p>
          <p
            class="text-sm text-muted"
            :style="{ fontFamily: form.fontFamily }"
          >
            The quick brown fox jumps over the lazy dog.
          </p>
        </div>
      </div>
    </div>
  </div>
</template>
