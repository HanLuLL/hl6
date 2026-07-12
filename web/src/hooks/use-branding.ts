import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import type { BrandingResponse } from "@/types";

const BRANDING_CACHE_KEY = "hl6_branding_cache_v1";
const BRANDING_UPDATED_EVENT = "hl6_branding_updated";
const BRANDING_CACHE_TTL_MS = 24 * 60 * 60 * 1000;
const BRANDING_TIMEOUT_MS = 2_000;
const DEFAULT_FAVICON_URL = "/vite.svg";

export const DEFAULT_BRAND_NAME = "SubDomain";

const DEFAULT_BRANDING: BrandingResponse = {
  name: DEFAULT_BRAND_NAME,
  logo_url: null,
  favicon_url: null,
  version: "0",
  announcement_enabled: false,
  announcement_content: "",
  footer_icp: "",
  footer_icp_link: "",
  footer_content: "",
};

interface CachedBranding {
  data: BrandingResponse;
  expiresAt: number;
}

function normalizeBranding(data: BrandingResponse | null | undefined): BrandingResponse {
  const name = data?.name?.trim() || DEFAULT_BRAND_NAME;
  return {
    name,
    logo_url: data?.logo_url ?? null,
    favicon_url: data?.favicon_url ?? null,
    version: data?.version ?? "0",
    announcement_enabled: data?.announcement_enabled ?? false,
    announcement_content: data?.announcement_content ?? "",
    footer_icp: data?.footer_icp ?? "",
    footer_icp_link: data?.footer_icp_link ?? "",
    footer_content: data?.footer_content ?? "",
  };
}

function readBrandingCache(): BrandingResponse | null {
  if (typeof window === "undefined") {
    return null;
  }

  try {
    const raw = localStorage.getItem(BRANDING_CACHE_KEY);
    if (!raw) {
      return null;
    }

    const parsed = JSON.parse(raw) as CachedBranding;
    if (!parsed || typeof parsed.expiresAt !== "number" || !parsed.data || parsed.expiresAt <= Date.now()) {
      try {
        localStorage.removeItem(BRANDING_CACHE_KEY);
      } catch {
        // Ignore cache cleanup failures.
      }
      return null;
    }

    return normalizeBranding(parsed.data);
  } catch {
    try {
      localStorage.removeItem(BRANDING_CACHE_KEY);
    } catch {
      // Ignore cache cleanup failures.
    }
    return null;
  }
}

function writeBrandingCache(data: BrandingResponse): void {
  if (typeof window === "undefined") {
    return;
  }

  const cache: CachedBranding = {
    data,
    expiresAt: Date.now() + BRANDING_CACHE_TTL_MS,
  };
  localStorage.setItem(BRANDING_CACHE_KEY, JSON.stringify(cache));
}

export function cacheBranding(data: BrandingResponse): BrandingResponse {
  const normalized = normalizeBranding(data);
  writeBrandingCache(normalized);
  if (typeof window !== "undefined") {
    window.dispatchEvent(new CustomEvent(BRANDING_UPDATED_EVENT, { detail: normalized }));
  }
  return normalized;
}

function applyFavicon(faviconURL: string | null): void {
  if (typeof document === "undefined") {
    return;
  }

  const href = faviconURL || DEFAULT_FAVICON_URL;
  let link = document.querySelector<HTMLLinkElement>("link[rel~='icon']");
  if (!link) {
    link = document.createElement("link");
    link.rel = "icon";
    document.head.appendChild(link);
  }
  link.type = href.endsWith(".ico") || href.includes(".ico?") ? "image/x-icon" : "image/svg+xml";
  link.href = href;
}

export function useBranding() {
  const [branding, setBranding] = useState<BrandingResponse>(() => readBrandingCache() ?? DEFAULT_BRANDING);

  useEffect(() => {
    const cached = readBrandingCache();
    if (cached) {
      setBranding(cached);
    }

    const controller = new AbortController();
    const timer = window.setTimeout(() => controller.abort(), BRANDING_TIMEOUT_MS);
    let active = true;

    async function refreshBranding(): Promise<void> {
      try {
        const res = await api.getBranding({ signal: controller.signal });
        if (!active) {
          return;
        }
        const latestBranding = cacheBranding(res.data);
        setBranding(latestBranding);
      } catch {
        if (!active) {
          return;
        }
        if (!cached) {
          setBranding(DEFAULT_BRANDING);
        }
      } finally {
        window.clearTimeout(timer);
      }
    }

    void refreshBranding();

    return () => {
      active = false;
      window.clearTimeout(timer);
      controller.abort();
    };
  }, []);

  useEffect(() => {
    const handleStorage = (event: StorageEvent) => {
      if (event.key !== BRANDING_CACHE_KEY) {
        return;
      }
      const cached = readBrandingCache();
      if (cached) {
        setBranding(cached);
      }
    };

    const handleBrandingUpdated = (event: Event) => {
      const customEvent = event as CustomEvent<BrandingResponse>;
      if (customEvent.detail) {
        setBranding(normalizeBranding(customEvent.detail));
      }
    };

    window.addEventListener("storage", handleStorage);
    window.addEventListener(BRANDING_UPDATED_EVENT, handleBrandingUpdated as EventListener);

    return () => {
      window.removeEventListener("storage", handleStorage);
      window.removeEventListener(BRANDING_UPDATED_EVENT, handleBrandingUpdated as EventListener);
    };
  }, []);

  useEffect(() => {
    applyFavicon(branding.favicon_url);
  }, [branding.favicon_url]);

  return branding;
}
