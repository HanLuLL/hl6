import { useEffect } from "react";
import type { ReactNode } from "react";
import { api } from "@/lib/api";
import type { SEOMeta } from "@/types";

const SEO_CACHE_KEY = "hl6_seo_cache_v1";
const SEO_CACHE_TTL_MS = 30 * 60 * 1000; // 30 minutes

interface CachedSEO {
  data: SEOMeta;
  expiresAt: number;
}

function readSEOCache(): SEOMeta | null {
  try {
    const raw = localStorage.getItem(SEO_CACHE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as CachedSEO;
    if (!parsed || parsed.expiresAt <= Date.now()) {
      localStorage.removeItem(SEO_CACHE_KEY);
      return null;
    }
    return parsed.data;
  } catch {
    return null;
  }
}

function writeSEOCache(data: SEOMeta): void {
  const cache: CachedSEO = { data, expiresAt: Date.now() + SEO_CACHE_TTL_MS };
  localStorage.setItem(SEO_CACHE_KEY, JSON.stringify(cache));
}

function ensureMetaTag(attr: string, key: string, content: string): void {
  if (!content) return;
  let tag = document.querySelector<HTMLMetaElement>(`meta[${attr}="${key}"]`);
  if (!tag) {
    tag = document.createElement("meta");
    tag.setAttribute(attr, key);
    document.head.appendChild(tag);
  }
  tag.setAttribute("content", content);
}

function ensureMetaName(name: string, content: string): void {
  ensureMetaTag("name", name, content);
}

function ensureMetaProperty(prop: string, content: string): void {
  ensureMetaTag("property", prop, content);
}

function applySEO(meta: SEOMeta): void {
  // Standard meta tags
  if (meta.site_description) {
    ensureMetaName("description", meta.site_description);
  }
  if (meta.site_keywords) {
    ensureMetaName("keywords", meta.site_keywords);
  }

  // Open Graph tags
  ensureMetaProperty("og:site_name", meta.site_name);
  if (meta.site_description) {
    ensureMetaProperty("og:description", meta.site_description);
  }
  ensureMetaProperty("og:type", "website");
}

export function useSEOInit(): void {
  useEffect(() => {
    // Apply cached SEO immediately
    const cached = readSEOCache();
    if (cached) {
      applySEO(cached);
    }

    // Fetch fresh SEO data
    api.getSEOMeta()
      .then((res) => {
        const meta: SEOMeta = {
          site_name: res.data.site_name || "HL6",
          site_description: res.data.site_description || "",
          site_keywords: res.data.site_keywords || "",
        };
        writeSEOCache(meta);
        applySEO(meta);
      })
      .catch(() => {
        // Silently fail - SEO is non-critical
      });
  }, []);
}

export function SEOProvider({ children }: { children: ReactNode }) {
  useSEOInit();
  return children;
}