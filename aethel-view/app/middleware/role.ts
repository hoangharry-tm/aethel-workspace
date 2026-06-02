// Role-based access guard. Applied to admin pages via definePageMeta({ requiredRole: 'ADMIN' }).
// Reads role requirement from route meta and redirects to /dashboard if insufficient.
export default defineNuxtRouteMiddleware((to) => {
  const { user } = useAuth()
  const requiredRole = to.meta.requiredRole as string | undefined
  if (!requiredRole) return

  const roleHierarchy: Record<string, number> = {
    USER: 1,
    RECEPTION: 2,
    ADMIN: 3,
    SYS_ADMIN: 4,
  }

  if (!user.value || (roleHierarchy[user.value.role] ?? 0) < (roleHierarchy[requiredRole] ?? 0)) {
    return navigateTo('/dashboard', { replace: true })
  }
})
