// Global route guard — runs on every navigation, automatically applied to all routes.
// Redirects to /auth/login if the user is not authenticated.
// Auth pages (/auth/*) are always accessible.
export default defineNuxtRouteMiddleware((to) => {
  if (to.path.startsWith('/auth/')) return

  const { isAuthenticated } = useAuth()
  if (!isAuthenticated.value) {
    return navigateTo('/auth/login', { replace: true })
  }
})
