// Shape matches GET /api/v1/config response

export interface NavItem {
  label: string;
  icon: string;
  to: string;
  badge?: number | null;
}

export interface NavGroup {
  label: string;
  roles: string[];
  items: NavItem[];
}

export interface AppRuntimeConfig {
  branding: {
    primaryColor: string;
    neutralPalette: string;
    fontFamily: string;
    logoUrl: string | null;
    wordmark: string;
  };
  nav: NavGroup[];
  features: {
    greenNotingEnabled: boolean;
    externalSmtpEnabled: boolean;
    require2faForAdmin: boolean;
  };
  org: {
    name: string;
    timezone: string;
    locale: string;
    contactEmail: string;
  };
}

const defaultConfig: AppRuntimeConfig = {
  branding: {
    primaryColor: "#4f46e5",
    neutralPalette: "slate",
    fontFamily: "Inter",
    logoUrl: null,
    wordmark: "Aethel Workspace",
  },
  nav: [
    {
      label: "Reception",
      roles: ["RECEPTION", "ADMIN"],
      items: [
        {
          label: "Dashboard",
          icon: "i-lucide-layout-dashboard",
          to: "/dashboard",
          badge: null,
        },
        {
          label: "Inbound",
          icon: "i-lucide-inbox",
          to: "/dispatch/inbound",
          badge: null,
        },
        {
          label: "Outbound",
          icon: "i-lucide-send",
          to: "/dispatch/outbound",
          badge: null,
        },
      ],
    },
    {
      label: "My Work",
      roles: ["ADMIN", "RECEPTION", "USER"],
      items: [
        {
          label: "My Documents",
          icon: "i-lucide-files",
          to: "/my-documents",
          badge: null,
        },
        {
          label: "Submit Outgoing",
          icon: "i-lucide-file-up",
          to: "/outgoing/new",
          badge: null,
        },
        {
          label: "Search",
          icon: "i-lucide-search",
          to: "/search",
          badge: null,
        },
      ],
    },
    {
      label: "Administration",
      roles: ["ADMIN"],
      items: [
        {
          label: "Users",
          icon: "i-lucide-users",
          to: "/admin/users",
          badge: null,
        },
        {
          label: "Document Types",
          icon: "i-lucide-tag",
          to: "/admin/document-types",
          badge: null,
        },
        {
          label: "Routing Rules",
          icon: "i-lucide-git-merge",
          to: "/admin/routing-rules",
          badge: null,
        },
        {
          label: "Escalation",
          icon: "i-lucide-bell-ring",
          to: "/admin/escalation",
          badge: null,
        },
        {
          label: "Audit Log",
          icon: "i-lucide-shield",
          to: "/admin/audit-log",
          badge: null,
        },
        {
          label: "Reports",
          icon: "i-lucide-bar-chart-2",
          to: "/admin/reports",
          badge: null,
        },
        {
          label: "Settings",
          icon: "i-lucide-settings",
          to: "/admin/settings",
          badge: null,
        },
        {
          label: "Branding",
          icon: "i-lucide-palette",
          to: "/admin/branding",
          badge: null,
        },
        {
          label: "Navigation",
          icon: "i-lucide-layout-list",
          to: "/admin/navigation",
          badge: null,
        },
      ],
    },
  ],
  features: {
    greenNotingEnabled: false,
    externalSmtpEnabled: false,
    require2faForAdmin: false,
  },
  org: {
    name: "Aethel Demo Org",
    timezone: "UTC",
    locale: "en-US",
    contactEmail: "admin@aethel.org",
  },
};

export function useAppRuntimeConfig() {
  const config = useState<AppRuntimeConfig>("app-runtime-config", () => ({
    ...defaultConfig,
  }));
  const isLoading = ref(false);

  // In production this would call GET /api/v1/config
  async function refresh() {
    isLoading.value = true;
    await new Promise((resolve) => setTimeout(resolve, 300));
    // Simulated API response — replace with: config.value = await $fetch('/api/v1/config')
    isLoading.value = false;
  }

  function updateBranding(partial: Partial<AppRuntimeConfig["branding"]>) {
    config.value = {
      ...config.value,
      branding: {
        ...config.value.branding,
        ...partial,
      },
    };
  }

  function updateOrg(partial: Partial<AppRuntimeConfig["org"]>) {
    config.value = {
      ...config.value,
      org: {
        ...config.value.org,
        ...partial,
      },
    };
  }

  function updateFeatures(partial: Partial<AppRuntimeConfig["features"]>) {
    config.value = {
      ...config.value,
      features: {
        ...config.value.features,
        ...partial,
      },
    };
  }

  function updateNav(nav: NavGroup[]) {
    config.value = {
      ...config.value,
      nav,
    };
  }

  return {
    config,
    isLoading,
    refresh,
    updateBranding,
    updateOrg,
    updateFeatures,
    updateNav,
  };
}
