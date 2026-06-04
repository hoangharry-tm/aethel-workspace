// Token storage rationale:
// Access token: in-memory via useState. Not localStorage (XSS risk). Survives navigation.
//   Lost on hard page reload — initAuth() recovers it via the httpOnly refresh cookie silently.
// Refresh token: httpOnly cookie set/cleared by the server. JavaScript cannot read it.
// CSRF token: readable cookie set by the server on login. Read here, sent as X-CSRF-Token.

export interface AuthUser {
  id: string
  role: 'ADMIN' | 'RECEPTION' | 'USER' | 'SYS_ADMIN'
}

export interface LoginCredentials {
  email: string
  password: string
}

export function useAuth() {
  const { public: { apiBaseUrl } } = useRuntimeConfig()

  // In-memory access token. null = not authenticated.
  const accessToken = useState<string | null>('auth:access-token', () => null)

  // Decoded user from JWT payload. Computed — never stored separately.
  const user = computed<AuthUser | null>(() => {
    if (!accessToken.value) return null
    try {
      // Decode the JWT payload (middle segment) for UI rendering only.
      // Do NOT verify the signature client-side — the server already verified it.
      const parts = accessToken.value.split('.')
      if (parts.length < 3) return null
      const payload = JSON.parse(atob(parts[1] ?? ''))
      if (!payload.sub || !payload.role || !payload.exp) return null
      if (Date.now() / 1000 > payload.exp) return null
      return { id: payload.sub, role: payload.role }
    } catch {
      return null
    }
  })

  const isAuthenticated = computed(() => !!user.value)

  // Read the CSRF token from the readable cookie.
  // Returns empty string if not present (non-browser / fresh page load before login).
  function getCSRFToken(): string {
    if (import.meta.server) return ''
    const match = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]+)/)
    return match?.[1] ? decodeURIComponent(match[1]) : ''
  }

  async function login(credentials: LoginCredentials): Promise<void> {
    const data = await $fetch<{ access_token: string; expires_in: number; role: string }>(
      `${apiBaseUrl}/api/v1/auth/login`,
      {
        method: 'POST',
        body: credentials,
      },
    )
    accessToken.value = data.access_token
  }

  async function logout(): Promise<void> {
    try {
      await $fetch(`${apiBaseUrl}/api/v1/auth/logout`, {
        method: 'POST',
        headers: { 'X-CSRF-Token': getCSRFToken() },
      })
    } finally {
      // Always clear client state, even if the server call fails.
      accessToken.value = null
    }
  }

  // Attempts to get a new access token using the httpOnly refresh cookie.
  // Returns true on success, false if the session has expired.
  async function refresh(): Promise<boolean> {
    try {
      const data = await $fetch<{ access_token: string }>(`${apiBaseUrl}/api/v1/auth/refresh`, {
        method: 'POST',
        headers: { 'X-CSRF-Token': getCSRFToken() },
        // The browser sends the refresh_token httpOnly cookie automatically.
      })
      accessToken.value = data.access_token
      return true
    } catch {
      accessToken.value = null
      return false
    }
  }

  // Called once on app start. Attempts silent token recovery via the refresh cookie.
  // If the httpOnly refresh cookie is present and valid, the user gets their session back.
  // If not, they will be redirected to /auth/login by the route guard.
  async function initAuth(): Promise<void> {
    if (accessToken.value) return
    await refresh()
  }

  async function requestPasswordReset(email: string): Promise<void> {
    await $fetch(`${apiBaseUrl}/api/v1/auth/password-reset/request`, {
      method: 'POST',
      body: { email },
    })
    // Always resolve — do not reveal whether the email exists.
  }

  return {
    accessToken: readonly(accessToken),
    user,
    isAuthenticated,
    login,
    logout,
    refresh,
    initAuth,
    requestPasswordReset,
    getCSRFToken,
  }
}
