<script setup lang="ts">
import { useMockData } from '~/composables/useMockData'
import type { Role } from '~/composables/useMockData'

definePageMeta({ layout: 'workspace', middleware: ['role'], requiredRole: 'ADMIN' })

const { users } = useMockData()
const toast = useToast()

const localUsers = ref(users.map(u => ({ ...u })))
const showNewUserModal = ref(false)

const newUserForm = reactive({
  name: '',
  email: '',
  department: '',
  role: 'USER' as Role,
})

const roleOptions = [
  { label: 'User', value: 'USER' },
  { label: 'Reception', value: 'RECEPTION' },
  { label: 'Admin', value: 'ADMIN' },
]

const departmentOptions = [
  { label: 'Finance', value: 'Finance' },
  { label: 'Legal', value: 'Legal' },
  { label: 'HR', value: 'HR' },
  { label: 'Operations', value: 'Operations' },
  { label: 'Procurement', value: 'Procurement' },
  { label: 'Administration', value: 'Administration' },
  { label: 'Reception', value: 'Reception' },
  { label: 'IT', value: 'IT' },
]

const roleColors: Record<Role, 'primary' | 'error' | 'neutral'> = {
  ADMIN: 'error',
  RECEPTION: 'primary',
  USER: 'neutral',
}

function toggleStatus(userId: string) {
  const user = localUsers.value.find(u => u.id === userId)
  if (user) {
    user.status = user.status === 'active' ? 'inactive' : 'active'
    toast.add({
      title: `User ${user.status === 'active' ? 'activated' : 'deactivated'}`,
      color: 'neutral',
    })
  }
}

function createUser() {
  const newUser = {
    id: `u${localUsers.value.length + 1}`,
    name: newUserForm.name,
    email: newUserForm.email,
    role: newUserForm.role,
    department: newUserForm.department,
    avatar: `https://api.dicebear.com/9.x/initials/svg?seed=${newUserForm.name.split(' ').map(w => w[0]).join('')}&backgroundColor=4f46e5&fontColor=ffffff`,
    status: 'active' as const,
  }
  localUsers.value.push(newUser)
  showNewUserModal.value = false
  Object.assign(newUserForm, { name: '', email: '', department: '', role: 'USER' })
  toast.add({ title: 'User created', color: 'success', icon: 'i-lucide-check' })
}
</script>

<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between gap-4 flex-wrap">
      <div>
        <h1 class="text-xl font-bold text-body">
          {{ $t('admin.users') }}
        </h1>
        <p class="text-sm text-muted mt-0.5">
          Manage workspace members and their access roles
        </p>
      </div>
      <UButton
        color="primary"
        variant="solid"
        leading-icon="i-lucide-user-plus"
        @click="showNewUserModal = true"
      >
        {{ $t('admin.createUser') }}
      </UButton>
    </div>

    <!-- Table -->
    <div class="bg-surface rounded-xl border border-border-base overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead class="bg-subtle border-b border-border-base">
            <tr>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider">
                User
              </th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider hidden sm:table-cell">
                Email
              </th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider">
                {{ $t('admin.role') }}
              </th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider hidden md:table-cell">
                {{ $t('admin.department') }}
              </th>
              <th class="px-4 py-3 text-center text-xs font-semibold text-muted uppercase tracking-wider">
                {{ $t('common.status') }}
              </th>
              <th class="px-4 py-3 text-right text-xs font-semibold text-muted uppercase tracking-wider">
                {{ $t('common.actions') }}
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border-faint">
            <tr
              v-for="user in localUsers"
              :key="user.id"
              class="hover:bg-subtle transition-colors"
            >
              <td class="px-4 py-3">
                <div class="flex items-center gap-3">
                  <UAvatar
                    :src="user.avatar"
                    :alt="user.name"
                    size="sm"
                  />
                  <span class="text-sm font-medium text-body">{{ user.name }}</span>
                </div>
              </td>
              <td class="px-4 py-3 hidden sm:table-cell">
                <span class="text-xs text-muted">{{ user.email }}</span>
              </td>
              <td class="px-4 py-3">
                <UBadge
                  :color="roleColors[user.role]"
                  variant="soft"
                  size="sm"
                >
                  {{ user.role }}
                </UBadge>
              </td>
              <td class="px-4 py-3 hidden md:table-cell">
                <span class="text-xs text-muted">{{ user.department }}</span>
              </td>
              <td class="px-4 py-3 text-center">
                <button
                  class="inline-flex items-center gap-1.5 text-xs font-medium"
                  :class="user.status === 'active' ? 'text-emerald-600' : 'text-icon-disabled'"
                  @click="toggleStatus(user.id)"
                >
                  <span
                    class="h-2 w-2 rounded-full"
                    :class="user.status === 'active' ? 'bg-emerald-500' : 'bg-divider'"
                  />
                  {{ user.status === 'active' ? $t('common.active') : $t('common.inactive') }}
                </button>
              </td>
              <td class="px-4 py-3 text-right">
                <div class="flex justify-end gap-1">
                  <UButton
                    icon="i-lucide-pencil"
                    color="neutral"
                    variant="ghost"
                    size="xs"
                  />
                  <UButton
                    icon="i-lucide-user-x"
                    color="neutral"
                    variant="ghost"
                    size="xs"
                    @click="toggleStatus(user.id)"
                  />
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>

  <!-- New User Modal -->
  <UModal v-model:open="showNewUserModal">
    <template #content>
      <div class="p-6 space-y-4">
        <h3 class="text-base font-semibold text-body">
          {{ $t('admin.createUser') }}
        </h3>

        <UFormField :label="$t('admin.fullName')" name="name" required>
          <UInput v-model="newUserForm.name" placeholder="Full name" class="w-full" />
        </UFormField>

        <UFormField :label="$t('auth.email')" name="email" required>
          <UInput v-model="newUserForm.email" type="email" placeholder="email@aethel.org" class="w-full" />
        </UFormField>

        <UFormField :label="$t('admin.department')" name="department" required>
          <USelect v-model="newUserForm.department" :items="departmentOptions" placeholder="Select department" class="w-full" />
        </UFormField>

        <UFormField :label="$t('admin.role')" name="role">
          <USelect v-model="newUserForm.role" :items="roleOptions" class="w-full" />
        </UFormField>

        <div class="flex gap-2 pt-2">
          <UButton color="primary" variant="solid" @click="createUser">
            {{ $t('admin.createUser') }}
          </UButton>
          <UButton color="neutral" variant="outline" @click="showNewUserModal = false">
            {{ $t('common.cancel') }}
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>
