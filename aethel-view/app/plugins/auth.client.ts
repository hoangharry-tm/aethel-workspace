// Runs only in the browser (.client.ts suffix) — httpOnly cookie recovery is browser-only.
// Attempts silent session recovery on every page load. If the user has a valid refresh cookie,
// they get their access token back without seeing the login page.
export default defineNuxtPlugin(async () => {
  const auth = useAuth()
  await auth.initAuth()
})
