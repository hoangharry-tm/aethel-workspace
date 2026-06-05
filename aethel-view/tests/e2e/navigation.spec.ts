// Navigation stability test — verifies the auth session survives 100 consecutive hard page
// loads without going stale or bouncing to /auth/login.
//
// Requires both servers running:
//   aethel-core:  cd aethel-core && go run ./cmd/server
//   aethel-view:  cd aethel-view && pnpm dev
//
// Run:
//   pnpm exec playwright test --config playwright.e2e.config.ts
import { test, expect, type Page } from "@playwright/test";

const BASE = "http://localhost:3000";
const CREDENTIALS = { email: "admin@aethel.dev", password: "test1234" };

// All routes accessible under the ADMIN role, in a natural traversal order.
const ADMIN_ROUTES = [
  "/dashboard",
  "/dispatch/inbound",
  "/dispatch/outbound",
  "/dispatch/inbound/new",
  "/search",
  "/admin/users",
  "/admin/document-types",
  "/admin/routing-rules",
  "/admin/escalation",
  "/admin/reports",
  "/admin/audit-log",
  "/admin/settings",
  "/admin/branding",
  "/admin/navigation",
];

// Logs in via the login form and waits until the workspace is visible.
async function login(page: Page): Promise<void> {
  await page.goto(`${BASE}/auth/login`, { waitUntil: "networkidle" });
  await page.locator('input[type="email"]').fill(CREDENTIALS.email);
  await page.locator('input[type="password"]').fill(CREDENTIALS.password);
  await page.locator('button[type="submit"]').click();
  await page.waitForURL(`${BASE}/dashboard`, { timeout: 20_000 });
  // Confirm workspace layout rendered after login.
  await expect(page.locator("header").first()).toBeVisible({ timeout: 10_000 });
}

test.describe("Navigation stability — 100-page hard-refresh traversal", () => {
  // 100 full page loads with network-idle wait; allow 5 minutes total.
  test.setTimeout(300_000);

  test("session and page render survive 100 consecutive navigations", async ({
    page,
  }) => {
    // Collect auth 401s from the network layer to surface them in the report.
    const refreshFailures: string[] = [];
    page.on("response", (res) => {
      if (res.url().includes("/api/v1/auth/refresh") && res.status() !== 200) {
        refreshFailures.push(
          `[nav to ${page.url()}] refresh → ${res.status()}`,
        );
      }
    });

    // Collect JS console errors for diagnostics.
    const consoleErrors: string[] = [];
    page.on("console", (msg) => {
      if (msg.type() === "error") consoleErrors.push(msg.text());
    });

    // ── Login ──────────────────────────────────────────────────────────────────
    await login(page);

    // ── 100 hard navigations ───────────────────────────────────────────────────
    // Each goto() is a full browser navigation (new SSR request + client hydration +
    // auth cookie recovery). This is the exact code path that produced 401s before
    // the credentials:include + CORS origin fix.
    for (let i = 0; i < 100; i++) {
      const route = ADMIN_ROUTES[i % ADMIN_ROUTES.length]!;
      const label = `Navigation #${i + 1} → ${route}`;

      await page.goto(`${BASE}${route}`, { waitUntil: "networkidle" });

      // Must not have been bounced back to the login page.
      expect(page.url(), `${label}: redirected to /auth/login`).not.toContain(
        "/auth/login",
      );

      // The workspace layout (<aside> sidebar + <header> navbar) must be present.
      // Their absence means SSR rendered an unauthenticated shell or a blank error state.
      await expect(
        page.locator("aside").first(),
        `${label}: workspace sidebar not rendered`,
      ).toBeVisible({ timeout: 8_000 });

      await expect(
        page.locator("header").first(),
        `${label}: workspace navbar not rendered`,
      ).toBeVisible({ timeout: 8_000 });
    }

    // ── Auth health summary ────────────────────────────────────────────────────
    expect(
      refreshFailures,
      `refresh token endpoint returned non-200:\n${refreshFailures.join("\n")}`,
    ).toHaveLength(0);

    // Log console errors as a soft warning (many pages stub APIs, so errors are expected
    // in some columns; include them in the report but don't fail the test on them).
    if (consoleErrors.length > 0) {
      console.warn(
        `[navigation.spec] ${consoleErrors.length} JS console error(s) observed:`,
      );
      consoleErrors.slice(0, 10).forEach((e) => console.warn("  ", e));
    }
  });
});

test.describe("Client-side soft navigation stability", () => {
  // Tests Vue Router navigation (no full page reload) — verifies the in-memory
  // access token survives multiple client-side route transitions.
  test.setTimeout(120_000);

  test("workspace renders correctly across 50 soft navigations", async ({
    page,
  }) => {
    await login(page);

    for (let i = 0; i < 50; i++) {
      const route = ADMIN_ROUTES[i % ADMIN_ROUTES.length]!;
      const label = `Soft nav #${i + 1} → ${route}`;

      // Click an anchor pointing to the route if it exists in the sidebar,
      // otherwise fall back to a goto() (hard nav) for routes not in the nav.
      const link = page.locator(`a[href="${route}"]`).first();
      const linkVisible = await link.isVisible().catch(() => false);

      if (linkVisible) {
        await link.click();
        await page.waitForLoadState("networkidle");
      } else {
        await page.goto(`${BASE}${route}`, { waitUntil: "networkidle" });
      }

      expect(page.url(), `${label}: redirected to /auth/login`).not.toContain(
        "/auth/login",
      );
      await expect(
        page.locator("header").first(),
        `${label}: navbar gone`,
      ).toBeVisible({ timeout: 6_000 });
    }
  });
});
