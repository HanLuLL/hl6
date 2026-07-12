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
  if (meta.site_author) {
    ensureMetaName("author", meta.site_author);
  }

  // Open Graph tags
  ensureMetaProperty("og:site_name", meta.site_name);
  if (meta.site_description) {
    ensureMetaProperty("og:description", meta.site_description);
  }
  ensureMetaProperty("og:type", "website");
  if (meta.site_og_image) {
    ensureMetaProperty("og:image", meta.site_og_image);
  }

  // Twitter Card tags
  if (meta.twitter_card) {
    ensureMetaName("twitter:card", meta.twitter_card);
  }
  if (meta.twitter_site) {
    ensureMetaName("twitter:site", meta.twitter_site);
  }
  if (meta.site_og_image) {
    ensureMetaName("twitter:image", meta.site_og_image);
  }

  // Analytics (Google Analytics gtag.js)
  if (meta.analytics_id) {
    injectAnalytics(meta.analytics_id);
  }
}

function injectAnalytics(id: string): void {
  if (document.getElementById("ga-gtag-src")) return;

  const script1 = document.createElement("script");
  script1.id = "ga-gtag-src";
  script1.async = true;
  script1.src = `https://www.googletagmanager.com/gtag/js?id=${id}`;
  document.head.appendChild(script1);

  const script2 = document.createElement("script");
  script2.id = "ga-gtag-init";
  script2.text = `
    window.dataLayer = window.dataLayer || [];
    function gtag(){dataLayer.push(arguments);}
    gtag('js', new Date());
    gtag('config', '${id}');
  `;
  document.head.appendChild(script2);
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
          site_author: res.data.site_author || "",
          site_og_image: res.data.site_og_image || "",
          twitter_card: res.data.twitter_card || "summary_large_image",
          twitter_site: res.data.twitter_site || "",
          analytics_id: res.data.analytics_id || "",
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