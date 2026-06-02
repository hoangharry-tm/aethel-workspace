// Global $fetch interceptor: attaches Bearer token + CSRF header on every request
// and retries once on 401 after silently refreshing the access token.
export default defineNuxtPlugin(() => {
  const auth = useAuth()

  $fetch.create({
    onRequest({ options }) {
      if (auth.accessToken.value) {
        options.headers = new Headers(options.headers as HeadersInit)
        options.headers.set('Authorization', `Bearer ${auth.accessToken.value}`)
        options.headers.set('X-CSRF-Token', auth.getCSRFToken())
      }
    },

    async onResponseError({ response, options, request }) {
      if (response.status !== 401) return

      // Guard against infinite loop: if the refresh request itself got 401, go to login.
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
})
