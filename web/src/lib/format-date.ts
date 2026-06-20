import i18n from "@/i18n";
import type { TFunction } from "i18next";

/** 将 i18next 语言代码映射为 Intl 的 BCP 47。 */
export function resolveIntlLocale(lang?: string): string {
  const l = (lang ?? i18n.language ?? "en").trim();
  if (l === "zh") return "zh-CN";
  if (l === "zh-Hant") return "zh-Hant";
  return l;
}

const defaultDateOpts: Intl.DateTimeFormatOptions = {
  year: "numeric",
  month: "short",
  day: "numeric",
};

const defaultDateTimeOpts: Intl.DateTimeFormatOptions = {
  year: "numeric",
  month: "short",
  day: "numeric",
  hour: "numeric",
  minute: "2-digit",
};

/** 日历日期（无时间）。无效/空 → 长破折号。 */
export function formatDate(
  iso: string | number | Date | null | undefined,
  opts?: Intl.DateTimeFormatOptions,
  locale?: string,
): string {
  if (iso == null || iso === "") return "—";
  const d = iso instanceof Date ? iso : new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  const loc = locale ?? resolveIntlLocale();
  return d.toLocaleDateString(loc, opts ?? defaultDateOpts);
}

/** 日期与时间（含毫秒）。无效/空 → 长破折号。 */
export function formatDateTimeMs(
  iso: string | number | Date | null | undefined,
  locale?: string,
): string {
  if (iso == null || iso === "") return "—";
  const d = iso instanceof Date ? iso : new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  const loc = locale ?? resolveIntlLocale();
  return d.toLocaleString(loc, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
    second: "2-digit",
    fractionalSecondDigits: 3,
  });
}

/** 日期与时间。无效/空 → 长破折号。 */
export function formatDateTime(
  iso: string | number | Date | null | undefined,
  opts?: Intl.DateTimeFormatOptions,
  locale?: string,
): string {
  if (iso == null || iso === "") return "—";
  const d = iso instanceof Date ? iso : new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  const loc = locale ?? resolveIntlLocale();
  return d.toLocaleString(loc, opts ?? defaultDateTimeOpts);
}

/** 短数字日期+时间（24 小时），用于监控/密集表格。 */
export function formatDateTimeNumeric(
  iso: string | number | Date | null | undefined,
  locale?: string,
): string {
  if (iso == null || iso === "") return "—";
  const d = iso instanceof Date ? iso : new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  const loc = locale ?? resolveIntlLocale();
  return d.toLocaleString(loc, {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  });
}

/** 过去 ISO 日期字符串的相对时间标签。 */
export function formatRelativeTime(dateStr: string, t: TFunction): string {
  const now = Date.now();
  const date = new Date(dateStr).getTime();
  const diff = now - date;
  const minutes = Math.floor(diff / 60000);
  if (minutes < 1) return t("notifications.justNow");
  if (minutes < 60) return t("notifications.minutesAgo", { count: minutes });
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return t("notifications.hoursAgo", { count: hours });
  const days = Math.floor(hours / 24);
  if (days < 30) return t("notifications.daysAgo", { count: days });
  return formatDate(dateStr);
}
