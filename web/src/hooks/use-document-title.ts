import { useEffect } from "react";
import { useBranding } from "@/hooks/use-branding";

const TITLE_SEPARATOR = "｜";

export function formatDocumentTitle(pageTitle: string | undefined, brandName: string): string {
  const trimmed = pageTitle?.trim();
  if (!trimmed) {
    return brandName;
  }
  return `${trimmed}${TITLE_SEPARATOR}${brandName}`;
}

export function useDocumentTitle(pageTitle?: string): void {
  const branding = useBranding();

  useEffect(() => {
    document.title = formatDocumentTitle(pageTitle, branding.name);
  }, [pageTitle, branding.name]);
}
