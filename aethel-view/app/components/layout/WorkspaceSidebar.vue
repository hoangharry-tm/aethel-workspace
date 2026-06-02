<script setup lang="ts">
import { useMockData } from '~/composables/useMockData'
import { useSidebarDrawer } from '~/composables/useSidebarDrawer'
import { useAppRuntimeConfig } from '~/composables/useRuntimeConfig'
import type { NavGroup } from '~/composables/useRuntimeConfig'

const { currentUser } = useMockData()
const { isOpen: isDrawerOpen, close: closeDrawer } = useSidebarDrawer()
const { config } = useAppRuntimeConfig()
// Route guards and nav visibility use the real role from JWT, not the prototype mock.
const { user: authUser } = useAuth()

const isCollapsed = ref(false)
const route = useRoute()

const { documents } = useMockData()

const pendingCount = computed(() =>
  documents.filter(d => d.status === 'PENDING_ASSIGNMENT' && d.isInbound).length,
)

// Hardcoded fallback nav — used if runtime config nav is empty
const hardcodedNavGroups = computed<NavGroup[]>(() => [
  {
    label: 'Reception',
    roles: ['RECEPTION', 'ADMIN'],
    items: [
      { label: 'Dashboard', icon: 'i-lucide-layout-dashboard', to: '/dashboard', badge: null },
      { label: 'Inbound', icon: 'i-lucide-inbox', to: '/dispatch/inbound', badge: pendingCount.value },
      { label: 'Outbound', icon: 'i-lucide-send', to: '/dispatch/outbound', badge: null },
    ],
  },
  {
    label: 'My Work',
    roles: ['ADMIN', 'RECEPTION', 'USER'],
    items: [
      { label: 'My Documents', icon: 'i-lucide-files', to: '/my-documents', badge: null },
      { label: 'Submit Outgoing', icon: 'i-lucide-file-up', to: '/outgoing/new', badge: null },
      { label: 'Search', icon: 'i-lucide-search', to: '/search', badge: null },
    ],
  },
  {
    label: 'Administration',
    roles: ['ADMIN'],
    items: [
      { label: 'Users', icon: 'i-lucide-users', to: '/admin/users', badge: null },
      { label: 'Document Types', icon: 'i-lucide-tag', to: '/admin/document-types', badge: null },
      { label: 'Routing Rules', icon: 'i-lucide-git-merge', to: '/admin/routing-rules', badge: null },
      { label: 'Escalation', icon: 'i-lucide-bell-ring', to: '/admin/escalation', badge: null },
      { label: 'Audit Log', icon: 'i-lucide-shield', to: '/admin/audit-log', badge: null },
      { label: 'Reports', icon: 'i-lucide-bar-chart-2', to: '/admin/reports', badge: null },
      { label: 'Settings', icon: 'i-lucide-settings', to: '/admin/settings', badge: null },
      { label: 'Branding', icon: 'i-lucide-palette', to: '/admin/branding', badge: null },
      { label: 'Navigation', icon: 'i-lucide-layout-list', to: '/admin/navigation', badge: null },
    ],
  },
])

// Use runtime config nav when available, fall back to hardcoded
const navGroups = computed<NavGroup[]>(() =>
  config.value.nav.length > 0 ? config.value.nav : hardcodedNavGroups.value,
)

// Gate nav groups using the real JWT role. Falls back to mock role in prototype mode
// (when the user hasn't authenticated through the real backend yet).
const visibleGroups = computed(() => {
  const role = authUser.value?.role ?? currentUser.value.role
  return navGroups.value.filter(g => g.roles.includes(role))
})

function isActive(to: string) {
  return route.path === to || route.path.startsWith(to + '/')
}

function handleNav() {
  if (isDrawerOpen.value) closeDrawer()
}
</script>

<template>
  <!-- Desktop sidebar -->
  <aside
    class="hidden lg:flex flex-col bg-surface border-r border-border-base transition-all duration-200 flex-shrink-0"
    :class="isCollapsed ? 'w-16' : 'w-64'"
  >
    <!-- Brand -->
    <div class="flex items-center gap-2 px-3 py-4 border-b border-border-faint relative">
      <div class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg bg-accent">
        <UIcon name="i-lucide-building-2" class="h-5 w-5 text-white" />
      </div>
      <span
        v-if="!isCollapsed"
        class="font-bold text-body text-sm truncate"
      >
        Aethel Workspace
      </span>
      <button
        class="ml-auto flex-shrink-0 rounded p-1 text-icon-disabled hover:text-muted hover:bg-subtle-2 transition-colors"
        :title="isCollapsed ? 'Expand sidebar' : 'Collapse sidebar'"
        @click="isCollapsed = !isCollapsed"
      >
        <UIcon
          :name="isCollapsed ? 'i-lucide-chevrons-right' : 'i-lucide-chevrons-left'"
          class="h-4 w-4"
        />
      </button>
    </div>

    <!-- Nav -->
    <nav class="flex-1 overflow-y-auto py-3 px-2 space-y-4">
      <div
        v-for="group in visibleGroups"
        :key="group.label"
        class="space-y-0.5"
      >
        <p
          v-if="!isCollapsed"
          class="px-2 mb-1 text-[10px] font-semibold uppercase tracking-wider text-icon-disabled"
        >
          {{ group.label }}
        </p>
        <USeparator v-else class="my-1" />
        <NuxtLink
          v-for="item in group.items"
          :key="item.to"
          :to="item.to"
          class="flex items-center gap-2.5 rounded-lg px-2 py-2 text-sm font-medium transition-colors group"
          :class="[
            isActive(item.to)
              ? 'bg-accent/5 text-accent'
              : 'text-muted hover:bg-subtle hover:text-body',
            isCollapsed ? 'justify-center' : '',
          ]"
          @click="handleNav"
        >
          <UTooltip
            v-if="isCollapsed"
            :text="item.label"
            :popper="{ placement: 'right' }"
          >
            <UIcon
              :name="item.icon"
              class="h-5 w-5 flex-shrink-0"
            />
          </UTooltip>
          <template v-else>
            <UIcon
              :name="item.icon"
              class="h-5 w-5 flex-shrink-0"
            />
            <span class="flex-1 truncate">{{ item.label }}</span>
            <UBadge
              v-if="item.badge != null && item.badge > 0"
              color="primary"
              variant="soft"
              size="xs"
            >
              {{ item.badge }}
            </UBadge>
          </template>
        </NuxtLink>
      </div>
    </nav>

    <!-- User section -->
    <div class="border-t border-border-faint p-2">
      <div
        class="flex items-center gap-2 rounded-lg p-2"
        :class="isCollapsed ? 'justify-center' : ''"
      >
        <UAvatar
          :src="currentUser.avatar"
          :alt="currentUser.name"
          size="sm"
          class="flex-shrink-0"
        />
        <div
          v-if="!isCollapsed"
          class="flex-1 min-w-0"
        >
          <p class="text-xs font-semibold text-body truncate">
            {{ currentUser.name }}
          </p>
          <UBadge
            color="primary"
            variant="soft"
            size="xs"
          >
            {{ currentUser.role }}
          </UBadge>
        </div>
        <UButton
          v-if="!isCollapsed"
          icon="i-lucide-log-out"
          color="neutral"
          variant="ghost"
          size="xs"
          :to="'/auth/login'"
        />
      </div>
    </div>
  </aside>

  <!-- Mobile drawer -->
  <USlideover
    v-model:open="isDrawerOpen"
    side="left"
    class="lg:hidden"
  >
    <template #content>
      <div class="flex flex-col h-full bg-surface w-64">
        <!-- Brand -->
        <div class="flex items-center gap-2 px-3 py-4 border-b border-border-faint">
          <div class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg bg-accent">
            <UIcon name="i-lucide-building-2" class="h-5 w-5 text-white" />
          </div>
          <span class="font-bold text-body text-sm">Aethel Workspace</span>
          <button
            class="ml-auto rounded p-1 text-icon-disabled hover:text-muted"
            @click="closeDrawer"
          >
            <UIcon name="i-lucide-x" class="h-4 w-4" />
          </button>
        </div>

        <!-- Nav -->
        <nav class="flex-1 overflow-y-auto py-3 px-2 space-y-4">
          <div
            v-for="group in visibleGroups"
            :key="group.label"
            class="space-y-0.5"
          >
            <p class="px-2 mb-1 text-[10px] font-semibold uppercase tracking-wider text-icon-disabled">
              {{ group.label }}
            </p>
            <NuxtLink
              v-for="item in group.items"
              :key="item.to"
              :to="item.to"
              class="flex items-center gap-2.5 rounded-lg px-2 py-2 text-sm font-medium transition-colors"
              :class="isActive(item.to) ? 'bg-accent/5 text-accent' : 'text-muted hover:bg-subtle'"
              @click="handleNav"
            >
              <UIcon :name="item.icon" class="h-5 w-5 flex-shrink-0" />
              <span class="flex-1 truncate">{{ item.label }}</span>
              <UBadge
                v-if="item.badge != null && item.badge > 0"
                color="primary"
                variant="soft"
                size="xs"
              >
                {{ item.badge }}
              </UBadge>
            </NuxtLink>
          </div>
        </nav>

        <!-- User section -->
        <div class="border-t border-border-faint p-2">
          <div class="flex items-center gap-2 rounded-lg p-2">
            <UAvatar
              :src="currentUser.avatar"
              :alt="currentUser.name"
              size="sm"
              class="flex-shrink-0"
            />
            <div class="flex-1 min-w-0">
              <p class="text-xs font-semibold text-body truncate">
                {{ currentUser.name }}
              </p>
              <UBadge color="primary" variant="soft" size="xs">
                {{ currentUser.role }}
              </UBadge>
            </div>
          </div>
        </div>
      </div>
    </template>
  </USlideover>
</template>
