<script setup lang="ts">
import { useMockData } from "~/composables/useMockData";

definePageMeta({ layout: "auth" });

const { setRole } = useMockData();
const router = useRouter();

const form = reactive({
  email: "",
  password: "",
});

const loading = ref(false);

async function handleLogin() {
  // TODO: add connection to BE service for authenticate user
  loading.value = true;
  // Simulate network delay
  await new Promise((resolve) => setTimeout(resolve, 800));
  setRole("RECEPTION");
  loading.value = false;
  await router.push("/dashboard");
}
</script>

<template>
  <div class="p-8">
    <div class="mb-6">
      <h2 class="text-xl font-bold text-body">Sign in to Aethel Workspace</h2>
      <p class="text-sm text-muted mt-1">
        Enter your credentials to access your workspace.
      </p>
    </div>

    <form class="space-y-4" @submit.prevent="handleLogin">
      <UFormField label="Email address" name="email">
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

      <UFormField label="Password" name="password">
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
          class="text-xs text-accent hover:text-accent hover:underline"
        >
          Forgot password?
        </button>
      </div>

      <UButton
        type="submit"
        color="primary"
        variant="solid"
        block
        :loading="loading"
      >
        Sign in
      </UButton>
    </form>
  </div>
</template>
