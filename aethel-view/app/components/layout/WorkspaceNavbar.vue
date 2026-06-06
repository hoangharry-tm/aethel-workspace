<script setup lang="ts">
import { useMockData } from '~/composables/useMockData'
import { useNotificationDrawer } from '~/composables/useNotificationDrawer'
import { useSidebarDrawer } from '~/composables/useSidebarDrawer'

const { t } = useI18n()
const { currentUser, setRole, notifications } = useMockData()
const { open: openNotifications } = useNotificationDrawer()
const { open: openSidebar } = useSidebarDrawer()
const { logout } = useAuth()
const router = useRouter()

const unreadCount = computed(() => notifications.filter(n => !n.read).length)

const searchQuery = ref('')

const route = useRoute()
const pageTitle = computed(() => {
  const map: Record<string, string> = {
    '/dashboard': t('nav.dashboard'),
    '/dispatch/inbound': t('dispatch.inboxTitle'),
    '/dispatch/inbound/new': 'Log Incoming Document',
    '/dispatch/outbound': t('dispatch.outboxTitle'),
    '/my-documents': t('document.myDocuments'),
    '/outgoing/new': t('dispatch.newOutbound'),
    '/search': t('search.title'),
    '/admin/users': t('admin.users'),
    '/admin/document-types': t('admin.documentTypes'),
    '/admin/routing-rules': t('admin.routingRules'),
    '/admin/escalation': t('admin.escalation'),
    '/admin/audit-log': t('admin.auditLog'),
    '/admin/reports': t('admin.reports'),
    '/admin/settings': t('admin.settings'),
    '/admin/branding': t('admin.branding'),
  }
  if (route.path.startsWith('/documents/')) return t('document.details')
  return map[route.path] ?? 'Aethel Workspace'
})

function handleSearch() {
  if (searchQuery.value.trim()) {
    router.push({ path: '/search', query: { q: searchQuery.value } })
  }
}

async function handleLogout() {
  try {
    await logout()
  } finally {
    window.location.replace('/auth/login')
  }
}

const profileItems = computed(() => [
  [
    {
      label: currentUser.value.name,
      slot: 'profile',
      disabled: true,
    },
  ],
  [
    {
      label: t('nav.profile'),
      icon: 'i-lucide-user',
      to: '#',
    },
  ],
  [
    {
      label: `${t('nav.switchRole')}: ADMIN`,
      icon: 'i-lucide-shield',
      onSelect: () => setRole('ADMIN'),
    },
    {
      label: `${t('nav.switchRole')}: RECEPTION`,
      icon: 'i-lucide-inbox',
      onSelect: () => setRole('RECEPTION'),
    },
    {
      label: `${t('nav.switchRole')}: USER`,
      icon: 'i-lucide-user-circle',
      onSelect: () => setRole('USER'),
    },
  ],
  [
    {
      label: t('auth.logout'),
      icon: 'i-lucide-log-out',
      onSelect: handleLogout,
    },
  ],
])
</script>

<template>
  <header class="h-14 bg-surface border-b border-border-base flex items-center px-4 gap-3 flex-shrink-0 z-10">
    <!-- Mobile hamburger -->
    <UButton
      icon="i-lucide-menu"
      color="neutral"
      variant="ghost"
      size="sm"
      class="lg:hidden"
      @click="openSidebar"
    />

    <!-- Page title -->
    <h1 class="text-sm font-semibold text-body truncate flex-shrink-0">
      {{ pageTitle }}
    </h1>

    <!-- Global search — hidden on mobile -->
    <div class="hidden md:flex flex-1 max-w-sm ml-2">
      <UInput
        v-model="searchQuery"
        icon="i-lucide-search"
        :placeholder="$t('common.search') + ' documents...'"
        size="sm"
        class="w-full"
        @keyup.enter="handleSearch"
      />
    </div>

    <div class="flex-1" />

    <!-- Notification bell -->
    <UButton
      icon="i-lucide-bell"
      color="neutral"
      variant="ghost"
      size="sm"
      class="relative"
      :aria-label="$t('nav.notifications')"
      @click="openNotifications"
    >
      <template v-if="unreadCount > 0">
        <span
          class="absolute -top-0.5 -right-0.5 flex h-4 w-4 items-center justify-center rounded-full bg-rose-500 text-[10px] font-bold text-white"
        >
          {{ unreadCount > 9 ? '9+' : unreadCount }}
        </span>
      </template>
    </UButton>

    <!-- Profile dropdown -->
    <UDropdownMenu :items="profileItems">
      <UButton
        color="neutral"
        variant="ghost"
        size="sm"
        class="flex items-center gap-2 px-2"
      >
        <UAvatar
          :src="currentUser.avatar"
          :alt="currentUser.name"
          size="xs"
        />
        <span class="hidden sm:block text-sm font-medium text-body max-w-24 truncate">
          {{ currentUser.name }}
        </span>
        <UIcon name="i-lucide-chevron-down" class="h-3.5 w-3.5 text-icon-disabled" />
      </UButton>

      <template #profile>
        <div class="px-3 py-2">
          <p class="text-sm font-semibold text-body">
            {{ currentUser.name }}
          </p>
          <p class="text-xs text-muted">
            {{ currentUser.email }}
          </p>
          <div class="mt-1 flex items-center gap-1">
            <UBadge color="primary" variant="soft" size="xs">
              {{ currentUser.role }}
            </UBadge>
            <!-- Role switcher is prototype/demo only — route guards use JWT role from useAuth() -->
            <UBadge color="warning" variant="soft" size="xs" class="ml-2">{{ $t('auth.demoOnly') }}</UBadge>
          </div>
        </div>
      </template>
    </UDropdownMenu>
  </header>
</template>
