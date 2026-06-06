<script setup lang="ts">
import { useAppRuntimeConfig } from '~/composables/useRuntimeConfig'

definePageMeta({ layout: 'workspace', middleware: ['role'], requiredRole: 'ADMIN' })

const { config, updateOrg, updateFeatures } = useAppRuntimeConfig()
const toast = useToast()

// Section 1: Org Profile
const orgForm = reactive({
  name: config.value.org.name,
  timezone: config.value.org.timezone,
  locale: config.value.org.locale,
  contactEmail: config.value.org.contactEmail,
})

const timezoneOptions = [
  { label: 'UTC', value: 'UTC' },
  { label: 'America/New_York', value: 'America/New_York' },
  { label: 'Europe/London', value: 'Europe/London' },
  { label: 'Asia/Ho_Chi_Minh', value: 'Asia/Ho_Chi_Minh' },
  { label: 'Asia/Singapore', value: 'Asia/Singapore' },
  { label: 'Asia/Tokyo', value: 'Asia/Tokyo' },
  { label: 'Australia/Sydney', value: 'Australia/Sydney' },
]

const localeOptions = [
  { label: 'English (US)', value: 'en-US' },
  { label: 'English (UK)', value: 'en-GB' },
  { label: 'French (France)', value: 'fr-FR' },
  { label: 'Vietnamese', value: 'vi-VN' },
  { label: 'Japanese', value: 'ja-JP' },
]

function saveOrgProfile() {
  updateOrg({ ...orgForm })
  toast.add({
    title: 'Organization profile saved',
    color: 'success',
    icon: 'i-lucide-check',
  })
}

// Section 2: Feature Toggles
const features = reactive({
  greenNotingEnabled: config.value.features.greenNotingEnabled,
  externalSmtpEnabled: config.value.features.externalSmtpEnabled,
  require2faForAdmin: config.value.features.require2faForAdmin,
})

function onToggle(key: keyof typeof features) {
  updateFeatures({ [key]: features[key] })
  toast.add({
    title: 'Setting saved',
    description: `Feature has been ${features[key] ? 'enabled' : 'disabled'}.`,
    color: 'success',
    icon: 'i-lucide-check',
    duration: 2000,
  })
}

// Section 3: DB alias examples (read-only)
const aliasExample = `schema:
  default_schema: "aethel"
  name_aliases:
    dispatches:         "dispatches"
    dispatch_events:    "dispatch_events"
    minute_sheets:      "minute_sheets"
    green_notes:        "green_notes"
    audit_ledger:       "audit_ledger"
  enum_aliases:
    priority_level:     "priority_level"
    dispatch_status:    "dispatch_status"
    user_role:          "user_role"`
</script>

<template>
  <div class="space-y-6 max-w-3xl">
    <!-- Header -->
    <div>
      <h1 class="text-xl font-bold text-body">
        {{ $t('admin.settings') }}
      </h1>
      <p class="text-sm text-muted mt-0.5">
        System-wide configuration and preferences
      </p>
    </div>

    <!-- Section 1: Organization Profile -->
    <div class="bg-surface rounded-xl border border-border-base overflow-hidden">
      <div class="px-4 py-3 border-b border-border-faint">
        <h2 class="text-sm font-semibold text-body">
          {{ $t('admin.orgProfile') }}
        </h2>
        <p class="text-xs text-muted mt-0.5">
          Basic details about your organization shown throughout the workspace.
        </p>
      </div>
      <div class="p-4 space-y-4">
        <UFormField :label="$t('admin.orgName')" name="orgName" required>
          <UInput
            v-model="orgForm.name"
            placeholder="Aethel Demo Org"
            class="w-full"
          />
        </UFormField>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <UFormField :label="$t('admin.timezone')" name="timezone">
            <USelect
              v-model="orgForm.timezone"
              :items="timezoneOptions"
              class="w-full"
            />
          </UFormField>

          <UFormField :label="$t('admin.locale')" name="locale">
            <USelect
              v-model="orgForm.locale"
              :items="localeOptions"
              class="w-full"
            />
          </UFormField>
        </div>

        <UFormField label="Contact Email" name="contactEmail">
          <UInput
            v-model="orgForm.contactEmail"
            type="email"
            placeholder="admin@yourorg.com"
            class="w-full"
          />
        </UFormField>

        <div class="pt-2 border-t border-border-faint">
          <UButton
            color="primary"
            variant="solid"
            leading-icon="i-lucide-save"
            @click="saveOrgProfile"
          >
            {{ $t('common.save') }}
          </UButton>
        </div>
      </div>
    </div>

    <!-- Section 2: Feature Toggles -->
    <div class="bg-surface rounded-xl border border-border-base overflow-hidden">
      <div class="px-4 py-3 border-b border-border-faint">
        <h2 class="text-sm font-semibold text-body">
          {{ $t('admin.featureToggles') }}
        </h2>
        <p class="text-xs text-muted mt-0.5">
          Enable or disable platform features. Changes take effect immediately.
        </p>
      </div>
      <div class="divide-y divide-border-faint">
        <!-- Green Noting -->
        <div class="flex items-center justify-between gap-4 px-4 py-4">
          <div class="flex items-center gap-3">
            <div class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg bg-subtle-2">
              <UIcon name="i-lucide-file-text" class="h-4 w-4 text-muted" />
            </div>
            <div>
              <p class="text-sm font-medium text-body">
                Enable Green Noting Canvas
              </p>
              <p class="text-xs text-muted">
                Activate institutional minute sheets and approval workflow
              </p>
            </div>
          </div>
          <USwitch
            v-model="features.greenNotingEnabled"
            color="primary"
            @update:model-value="onToggle('greenNotingEnabled')"
          />
        </div>

        <!-- External SMTP -->
        <div class="flex items-center justify-between gap-4 px-4 py-4">
          <div class="flex items-center gap-3">
            <div class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg bg-subtle-2">
              <UIcon name="i-lucide-mail" class="h-4 w-4 text-muted" />
            </div>
            <div>
              <p class="text-sm font-medium text-body">
                Enable External SMTP for Email Notifications
              </p>
              <p class="text-xs text-muted">
                Send notification emails via a custom SMTP server
              </p>
            </div>
          </div>
          <USwitch
            v-model="features.externalSmtpEnabled"
            color="primary"
            @update:model-value="onToggle('externalSmtpEnabled')"
          />
        </div>

        <!-- 2FA -->
        <div class="flex items-center justify-between gap-4 px-4 py-4">
          <div class="flex items-center gap-3">
            <div class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg bg-subtle-2">
              <UIcon name="i-lucide-shield-check" class="h-4 w-4 text-muted" />
            </div>
            <div>
              <p class="text-sm font-medium text-body">
                Require 2FA for Administrator Accounts
              </p>
              <p class="text-xs text-muted">
                Enforce two-factor authentication for all users with ADMIN role
              </p>
            </div>
          </div>
          <USwitch
            v-model="features.require2faForAdmin"
            color="primary"
            @update:model-value="onToggle('require2faForAdmin')"
          />
        </div>
      </div>
    </div>

    <!-- Section 3: Database Naming (read-only) -->
    <div class="bg-surface rounded-xl border border-border-base overflow-hidden">
      <div class="px-4 py-3 border-b border-border-faint">
        <h2 class="text-sm font-semibold text-body">
          Table &amp; Enum Aliases
        </h2>
        <p class="text-xs text-muted mt-0.5">
          Defined in <code class="bg-subtle-2 rounded px-1 text-body">blueprints/server-database.yaml</code>.
          Changes require a server restart.
        </p>
      </div>
      <div class="p-4 space-y-3">
        <UAlert
          color="info"
          variant="soft"
          icon="i-lucide-info"
          title="To change aliases, edit server-database.yaml and run `aethel migrate validate`."
        />
        <pre class="bg-subtle rounded-lg p-4 text-xs font-mono text-muted overflow-x-auto">{{ aliasExample }}</pre>
      </div>
    </div>
  </div>
</template>
