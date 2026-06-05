// Global route guard — runs on every navigation, automatically applied to all routes.
// SSR always passes through: the refresh cookie lives on localhost:8080, not localhost:3000,
// so it is never visible to the Nuxt SSR process. Session recovery happens client-side via
// auth.client.ts (initAuth → refresh). Client-side guard runs after initAuth completes.
export default defineNuxtRouteMiddleware((to) => {
  if (import.meta.server) return

  // /auth/* routes are always accessible. If the user already has a valid session and
  // visits /auth/login, they see the login form — submitting it replaces the token.
  // Forcing a redirect to /dashboard here breaks the ability to re-authenticate or
  // test the login flow while a leftover refresh cookie is still in the browser.
  if (to.path.startsWith('/auth/')) return

  const { isAuthenticated } = useAuth()
  if (!isAuthenticated.value) {
    return navigateTo('/auth/login', { replace: true })
  }
})
