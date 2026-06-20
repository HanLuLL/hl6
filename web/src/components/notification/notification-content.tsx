import type { MouseEvent } from "react";
import { useTranslation } from "react-i18next";
import DOMPurify from "dompurify";
import type { Notification } from "@/types";

const ALLOWED_TAGS = ["p", "br", "strong", "em", "u", "s", "h2", "h3", "ul", "ol", "li", "a", "img", "span"];
const ALLOWED_ATTR = ["href", "src", "alt", "target", "style"];

DOMPurify.addHook("afterSanitizeAttributes", (node) => {
  if (node.tagName === "A") {
    node.setAttribute("rel", "noopener noreferrer");
  }
});

function sanitize(html: string) {
  return DOMPurify.sanitize(html, {
    ALLOWED_TAGS,
    ALLOWED_ATTR,
    ALLOW_DATA_ATTR: false,
  });
}

function parseNotificationArgs(raw: Notification["message_args"]): Record<string, unknown> {
  if (!raw) return {};
  if (typeof raw === "string") {
    try {
      const parsed = JSON.parse(raw) as Record<string, unknown>;
      return parsed && typeof parsed === "object" ? parsed : {};
    } catch {
      return {};
    }
  }
  return raw;
}

function SubdomainSuspendedByAuditBody({ args }: { args: Record<string, unknown> }) {
  const { t } = useTranslation();
  const fqdn = String(args.fqdn ?? "");
  const rule = typeof args.rule === "string" ? args.rule.trim() : "";
  return (
    <div className="space-y-2 whitespace-pre-wrap text-sm">
      <p>{t("notification.subdomainSuspended.intro", { fqdn })}</p>
      {rule !== "" ? <p>{t("notification.subdomainSuspended.rule", { rule })}</p> : null}
    </div>
  );
}

function SubdomainReleasedByAdminBody({ args }: { args: Record<string, unknown> }) {
  const { t } = useTranslation();
  const fqdn = String(args.fqdn ?? "");
  const reason = typeof args.reason === "string" ? args.reason.trim() : "";
  return (
    <div className="space-y-2 whitespace-pre-wrap text-sm">
      <p>{t("notification.subdomainReleasedByAdmin.intro", { fqdn })}</p>
      {reason !== "" ? <p>{t("notification.subdomainReleasedByAdmin.reason", { reason })}</p> : null}
    </div>
  );
}

interface NotificationContentProps {
  notification: Notification;
  onImageClick?: (src: string) => void;
}

export function NotificationContent({ notification, onImageClick }: NotificationContentProps) {
  const { t } = useTranslation();
  const key = notification.message_key?.trim();
  const args = parseNotificationArgs(notification.message_args);

  if (key === "notification.subdomainReleasedByAdmin") {
    return <SubdomainReleasedByAdminBody args={args} />;
  }
  if (key === "notification.subdomainSuspended") {
    return <SubdomainSuspendedByAuditBody args={args} />;
  }

  if (key) {
    const fallback = (notification.content || "").trim() || notification.title;
    const translated = t(key, { ...args, defaultValue: fallback } as Record<string, unknown>);
    return (
      <div className="flex-1 min-h-0 overflow-y-auto space-y-2 whitespace-pre-wrap text-sm">{translated}</div>
    );
  }

  const html = sanitize(notification.content || "");
  const handleClick = (e: MouseEvent<HTMLDivElement>) => {
    if (!onImageClick) return;
    const target = e.target as HTMLElement;
    if (target.tagName === "IMG") {
      const src = (target as HTMLImageElement).src;
      if (src) {
        e.preventDefault();
        onImageClick(src);
      }
    }
  };

  return (
    <div
      className="flex-1 min-h-0 overflow-y-auto prose prose-sm dark:prose-invert max-w-none [&_img]:cursor-zoom-in"
      dangerouslySetInnerHTML={{ __html: html }}
      onClick={handleClick}
    />
  );
}
