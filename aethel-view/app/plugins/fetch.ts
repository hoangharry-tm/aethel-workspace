// Provides $apiFetch — a pre-configured $fetch instance that auto-attaches the Bearer
// token and CSRF header, and retries once on 401 via silent token refresh.
// Use useNuxtApp().$apiFetch(...) for all protected API calls outside of useAuth.ts.
// useAuth.ts itself calls auth endpoints directly with $fetch (no circular dependency).
export default defineNuxtPlugin(() => {
  const auth = useAuth()

  const apiFetch = $fetch.create({
    // credentials: 'include' sends the csrf_token cookie cross-origin so the backend's
    // double-submit CSRF check can actually compare cookie vs header.
    credentials: 'include',
    onRequest({ options }) {
      if (auth.accessToken.value) {
        options.headers = new Headers(options.headers as HeadersInit)
        options.headers.set('Authorization', `Bearer ${auth.accessToken.value}`)
        options.headers.set('X-CSRF-Token', auth.getCSRFToken())
      }
    },

    async onResponseError({ response, options, request }) {
      if (response.status !== 401) return

      // Guard against infinite loop: if the refresh request itself fails, go to login.
      if (typeof request === 'string' && request.includes('/auth/refresh')) {
        await navigateTo('/auth/login', { replace: true })
        return
      }

      const refreshed = await auth.refresh()
      if (!refreshed) {
        await navigateTo('/auth/login', { replace: true })
        return
      }

      // Retry the original request with the new token.
      const headers = new Headers(options.headers as HeadersInit)
      headers.set('Authorization', `Bearer ${auth.accessToken.value}`)
      headers.set('X-CSRF-Token', auth.getCSRFToken())
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      return $fetch(request as string, { ...options, headers } as any)
    },
  })

  return {
    provide: { apiFetch },
  }
})
