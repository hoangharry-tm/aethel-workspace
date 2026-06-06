import { defineNuxtConfig } from "nuxt/config";

// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: "2025-07-15",
  devtools: { enabled: true },

  css: ["~/assets/css/main.css"],

  components: [{ path: "~/components", pathPrefix: false }],

  runtimeConfig: {
    public: {
      apiBaseUrl: process.env.NUXT_PUBLIC_API_BASE_URL || 'http://localhost:8080',
    },
  },

  modules: [
    "@nuxt/a11y",
    "@nuxt/eslint",
    "@nuxt/image",
    "@nuxt/scripts",
    "@nuxt/test-utils",
    "@nuxt/ui",
    ["@nuxtjs/i18n", {
      locales: [
        { code: "en", name: "English", file: "en.json" },
        { code: "vi", name: "Tiếng Việt", file: "vi.json" },
      ],
      defaultLocale: "en",
      langDir: "locales/",
      strategy: "no_prefix",
      lazy: true,
    }],
    "@nuxtjs/mcp-toolkit",
    "@oro.ad/nuxt-claude-devtools",
    "@pinia/nuxt",
  ],

});

