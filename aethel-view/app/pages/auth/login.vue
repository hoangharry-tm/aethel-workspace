<script setup lang="ts">
definePageMeta({ layout: 'auth' })

const { t } = useI18n()
const { login, requestPasswordReset } = useAuth()
const router = useRouter()

const form = reactive({ email: '', password: '' })
const loading = ref(false)
const error = ref<string | null>(null)

async function handleLogin() {
  error.value = null
  loading.value = true
  try {
    await login({ email: form.email, password: form.password })
    await router.push('/dashboard')
  } catch (e: unknown) {
    const status = (e as { statusCode?: number }).statusCode
    if (status === 401) {
      error.value = t('errors.loginFailed')
    } else if (status === 423) {
      error.value = 'Account is temporarily locked due to too many failed attempts. Try again in 15 minutes.'
    } else if (status === 429) {
      error.value = 'Too many login attempts. Please wait before trying again.'
    } else {
      error.value = t('errors.serverError')
    }
  } finally {
    loading.value = false
  }
}

const resetEmail = ref('')
const resetSent = ref(false)
const showReset = ref(false)
const resetLoading = ref(false)

async function handlePasswordReset() {
  resetLoading.value = true
  try {
    // Always show the same success message regardless of whether the email exists.
    await requestPasswordReset(resetEmail.value)
    resetSent.value = true
  } finally {
    resetLoading.value = false
  }
}
</script>

<template>
  <div class="p-8">
    <div class="mb-6">
      <h2 class="text-xl font-bold text-body">{{ $t('auth.loginTitle') }}</h2>
      <p class="text-sm text-muted mt-1">
        {{ $t('auth.loginSubtitle') }}
      </p>
    </div>

    <form v-if="!showReset" class="space-y-4" @submit.prevent="handleLogin">
      <UFormField :label="$t('auth.email')" name="email">
        <UInput
          v-model="form.email"
          type="email"
          placeholder="you@aethel.org"
          icon="i-lucide-mail"
          autocomplete="email"
          class="w-full"
          required
        />
      </UFormField>

      <UFormField :label="$t('auth.password')" name="password">
        <UInput
          v-model="form.password"
          type="password"
          placeholder="Your password"
          icon="i-lucide-lock"
          autocomplete="current-password"
          class="w-full"
          required
        />
      </UFormField>

      <div class="flex justify-end">
        <button
          type="button"
          class="text-xs text-accent hover:underline"
          @click="showReset = true"
        >
          {{ $t('auth.forgotPassword') }}
        </button>
      </div>

      <UAlert
        v-if="error"
        color="error"
        variant="soft"
        :title="error"
        icon="i-lucide-alert-circle"
        class="mb-2"
      />

      <UButton
        type="button"
        color="primary"
        variant="solid"
        block
        :loading="loading"
        @click="handleLogin"
      >
        {{ $t('auth.login') }}
      </UButton>
    </form>

    <!-- Password reset form -->
    <div v-else class="space-y-4">
      <button
        class="flex items-center gap-1 text-sm text-muted hover:text-body mb-2"
        @click="showReset = false; resetSent = false; resetEmail = ''"
      >
        <UIcon name="i-lucide-arrow-left" class="h-4 w-4" />
        {{ $t('auth.backToLogin') }}
      </button>

      <div v-if="!resetSent" class="space-y-4">
        <p class="text-sm text-muted">
          Enter your email address and we'll send you a reset link if an account exists.
        </p>
        <UFormField :label="$t('auth.email')" name="reset-email">
          <UInput
            v-model="resetEmail"
            type="email"
            placeholder="you@aethel.org"
            icon="i-lucide-mail"
            class="w-full"
          />
        </UFormField>
        <UButton
          color="primary"
          variant="solid"
          block
          :loading="resetLoading"
          @click="handlePasswordReset"
        >
          Send reset link
        </UButton>
      </div>

      <UAlert
        v-else
        color="success"
        variant="soft"
        title="Check your email"
        description="If that address is registered, a password reset link has been sent."
        icon="i-lucide-mail-check"
      />
    </div>
  </div>
</template>
