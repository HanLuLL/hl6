import type {
  ApiResponse,
  PaginatedResponse,
  User,
  Domain,
  Subdomain,
  DNSRecord,
  CreditBalance,
  CreditTransaction,
  DomainGroupAccess,
  DomainWithGroupAccess,
  CloudflareZone,
  Stats,
  AuditLog,
  UserGroup,
  CloudflareAccount,
  Notification,
  OffsetPaginatedResponse,
  BrandingResponse,
  ReferralInfo,
  UserWithInviter,
  AdminDNSRecord,
  AdminConfigPayload,
  OIDCStatusPayload,
} from "@/types";

function normalizeApiBaseUrl(rawValue: string | undefined): string {
  const value = rawValue?.trim() ?? "";
  const baseUrl = value.length > 0 ? value : "/api/v1";

  if (baseUrl === "/") {
    return "";
  }

  return baseUrl.endsWith("/") ? baseUrl.slice(0, -1) : baseUrl;
}

export const API_BASE_URL = normalizeApiBaseUrl(import.meta.env.VITE_API_BASE_URL);

export function buildApiUrl(path: string): string {
  const normalizedPath = path.startsWith("/") ? path : `/${path}`;
  return `${API_BASE_URL}${normalizedPath}`;
}

export class ApiError extends Error {
  messageKey?: string;
  data?: unknown;
  constructor(message: string, messageKey?: string, data?: unknown) {
    super(message);
    this.messageKey = messageKey;
    this.data = data;
  }
}

export function getErrorMessage(err: unknown, t?: (key: string) => string): string {
  if (err instanceof ApiError && err.messageKey && t) {
    const translated = t(err.messageKey);
    if (translated !== err.messageKey) return translated;
  }
  if (err instanceof Error) return err.message;
  return String(err);
}

async function request<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string>),
  };

  const res = await fetch(buildApiUrl(path), {
    ...options,
    headers,
    credentials: "include",
  });

  if (res.status === 401) {
    if (!path.includes("/auth/me")) {
      const key = "hl6_401_count";
      const timeKey = "hl6_401_time";
      const now = Date.now();
      const lastTime = parseInt(sessionStorage.getItem(timeKey) || "0");
      let count = parseInt(sessionStorage.getItem(key) || "0");

      // Reset counter if more than 5 minutes since last 401
      if (now - lastTime > 5 * 60 * 1000) {
        count = 0;
      }

      count++;
      sessionStorage.setItem(key, String(count));
      sessionStorage.setItem(timeKey, String(now));

      if (count <= 3) {
        window.location.href = buildApiUrl("/auth/login");
      }
    }
    throw new ApiError("Not authenticated", "error.missingToken");
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({ message: res.statusText }));
    throw new ApiError(body.message || res.statusText, body.message_key, body.data);
  }

  return res.json();
}

export const api = {
  // Branding (public)
  getBranding: (options?: { signal?: AbortSignal }) =>
    request<ApiResponse<BrandingResponse>>("/branding", { signal: options?.signal }),

  // Auth
  getMe: async () => {
    const res = await request<ApiResponse<{ user: User; credits: number }>>("/auth/me");
    sessionStorage.removeItem("hl6_401_count");
    sessionStorage.removeItem("hl6_401_time");
    return res;
  },
  getOIDCStatus: () => request<ApiResponse<OIDCStatusPayload>>("/auth/oidc/status"),
  bootstrapOIDCConfig: (data: { oidc_issuer: string; oidc_client_id: string; oidc_client_secret: string }) =>
    request<ApiResponse<OIDCStatusPayload>>("/auth/oidc/bootstrap", { method: "POST", body: JSON.stringify(data) }),
  logout: () => request<ApiResponse<{ logout_url: string }>>("/auth/logout", { method: "POST" }),

  // Domains
  listDomains: () => request<ApiResponse<Domain[]>>("/domains"),

  // Subdomains
  listSubdomains: () => request<ApiResponse<Subdomain[]>>("/subdomains"),
  getSubdomain: (id: number) => request<ApiResponse<Subdomain>>(`/subdomains/${id}`),
  claimSubdomain: (data: { domain_id: number; name: string }) =>
    request<ApiResponse<Subdomain>>("/subdomains", { method: "POST", body: JSON.stringify(data) }),
  releaseSubdomain: (id: number) =>
    request<ApiResponse<{ message: string }>>(`/subdomains/${id}`, { method: "DELETE" }),

  // DNS Records
  listRecords: (subdomainId: number) =>
    request<ApiResponse<DNSRecord[]>>(`/subdomains/${subdomainId}/records`),
  createRecord: (subdomainId: number, data: { type: string; content: string; ttl?: number; proxied?: boolean }) =>
    request<ApiResponse<DNSRecord>>(`/subdomains/${subdomainId}/records`, { method: "POST", body: JSON.stringify(data) }),
  updateRecord: (subdomainId: number, recordId: number, data: { content: string; ttl?: number; proxied?: boolean }) =>
    request<ApiResponse<DNSRecord>>(`/subdomains/${subdomainId}/records/${recordId}`, { method: "PUT", body: JSON.stringify(data) }),
  deleteRecord: (subdomainId: number, recordId: number) =>
    request<ApiResponse<{ message: string }>>(`/subdomains/${subdomainId}/records/${recordId}`, { method: "DELETE" }),

  // Credits
  getCredits: () => request<ApiResponse<CreditBalance>>("/credits"),
  listTransactions: (page = 1, perPage = 20) =>
    request<PaginatedResponse<CreditTransaction[]>>(`/credits/transactions?page=${page}&per_page=${perPage}`),

  // Referrals
  getReferrals: (page = 1, perPage = 20) =>
    request<ReferralInfo>(`/referrals?page=${page}&per_page=${perPage}`),

  // Admin
  adminCreateDomain: (data: { name: string; cloudflare_zone_id: string; cloudflare_account_id: number; description: string; group_access: { group_id: number; credit_cost: number; max_dns_records?: number | null }[] }) =>
    request<ApiResponse<{ domain: Domain; group_access: DomainGroupAccess[] }>>("/admin/domains", { method: "POST", body: JSON.stringify(data) }),
  adminUpdateDomain: (id: number, data: { cloudflare_zone_id?: string; cloudflare_account_id?: number; is_active?: boolean; description?: string; group_access?: { group_id: number; credit_cost: number; max_dns_records?: number | null }[] }) =>
    request<ApiResponse<{ domain: Domain; group_access: DomainGroupAccess[] }>>(`/admin/domains/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  adminDeleteDomain: (id: number, options?: { force?: boolean; refund?: boolean }) =>
    request<ApiResponse<{ message: string }>>(`/admin/domains/${id}?force=${options?.force ?? false}&refund=${options?.refund ?? false}`, { method: "DELETE" }),
  adminListDomainsFull: () =>
    request<ApiResponse<DomainWithGroupAccess[]>>("/admin/domains-full"),
  adminListCloudflareZones: (accountId: number) =>
    request<ApiResponse<CloudflareZone[]>>(`/admin/cloudflare/accounts/${accountId}/zones`),
  adminGrantCredits: (data: { user_id: number; amount: number; description: string }) =>
    request<ApiResponse<{ user_id: number; granted: number; balance: number }>>("/admin/credits/grant", { method: "POST", body: JSON.stringify(data) }),
  adminListUsers: (page = 1, perPage = 20, search = "") =>
    request<PaginatedResponse<UserWithInviter[]>>(`/admin/users?page=${page}&per_page=${perPage}${search ? `&search=${encodeURIComponent(search)}` : ""}`),
  adminGetStats: () => request<ApiResponse<Stats>>("/admin/stats"),
  adminListAuditLogs: (page = 1, perPage = 20, search = "") =>
    request<PaginatedResponse<AuditLog[]>>(`/admin/audit-logs?page=${page}&per_page=${perPage}${search ? `&search=${encodeURIComponent(search)}` : ""}`),

  // Admin: User Groups
  adminListGroups: () =>
    request<ApiResponse<UserGroup[]>>("/admin/groups"),
  adminCreateGroup: (data: { name: string; is_default?: boolean }) =>
    request<ApiResponse<UserGroup>>("/admin/groups", { method: "POST", body: JSON.stringify(data) }),
  adminUpdateGroup: (id: number, data: { name?: string; is_default?: boolean }) =>
    request<ApiResponse<UserGroup>>(`/admin/groups/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  adminDeleteGroup: (id: number, migrateTo: number) =>
    request<ApiResponse<{ message: string }>>(`/admin/groups/${id}?migrate_to=${migrateTo}`, { method: "DELETE" }),
  adminUpdateUserGroup: (userId: number, groupId: number) =>
    request<ApiResponse<{ message: string }>>(`/admin/users/${userId}/group`, { method: "PUT", body: JSON.stringify({ group_id: groupId }) }),

  // Admin: System Config
  adminGetConfig: () =>
    request<ApiResponse<AdminConfigPayload>>("/admin/config"),
  adminUpdateConfig: (data: Record<string, string>) =>
    request<ApiResponse<{ message: string }>>("/admin/config", { method: "PUT", body: JSON.stringify(data) }),
  adminConfirmUrlConfig: () =>
    request<ApiResponse<{ message: string }>>("/admin/config/url-confirm", { method: "POST" }),

  // Admin: Cloudflare Accounts
  adminListCloudflareAccounts: () =>
    request<ApiResponse<CloudflareAccount[]>>("/admin/cloudflare/accounts"),
  adminCreateCloudflareAccount: (data: { name: string; api_token: string }) =>
    request<ApiResponse<CloudflareAccount>>("/admin/cloudflare/accounts", { method: "POST", body: JSON.stringify(data) }),
  adminUpdateCloudflareAccount: (id: number, data: { name: string; api_token?: string }) =>
    request<ApiResponse<CloudflareAccount>>(`/admin/cloudflare/accounts/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  adminDeleteCloudflareAccount: (id: number) =>
    request<ApiResponse<{ message: string }>>(`/admin/cloudflare/accounts/${id}`, { method: "DELETE" }),

  // Admin: DNS Records
  adminListDNSRecords: (page = 1, perPage = 20, search = "", domainId?: number, groupId?: number) => {
    const params = new URLSearchParams({ page: String(page), per_page: String(perPage) });
    if (search) params.set("search", search);
    if (domainId) params.set("domain_id", String(domainId));
    if (groupId) params.set("group_id", String(groupId));
    return request<PaginatedResponse<AdminDNSRecord[]>>(`/admin/dns-records?${params.toString()}`);
  },
  adminDeleteDNSRecord: (id: number, data: { notify: boolean; reason?: string }) =>
    request<ApiResponse<{ message: string }>>(`/admin/dns-records/${id}`, { method: "DELETE", body: JSON.stringify(data) }),
  adminUpdateBranding: (data: { name: string }) =>
    request<ApiResponse<BrandingResponse>>("/admin/branding", { method: "PUT", body: JSON.stringify(data) }),
  adminUploadBrandingLogo: async (file: File) => {
    const formData = new FormData();
    formData.append("file", file);
    const res = await fetch(buildApiUrl("/admin/branding/logo"), {
      method: "POST",
      body: formData,
      credentials: "include",
    });
    if (!res.ok) {
      const body = await res.json().catch(() => ({ message: res.statusText }));
      throw new ApiError(body.message || res.statusText, body.message_key);
    }
    return res.json() as Promise<ApiResponse<BrandingResponse>>;
  },
  adminDeleteBrandingLogo: () =>
    request<ApiResponse<BrandingResponse>>("/admin/branding/logo", { method: "DELETE" }),

  // Notifications (user)
  listNotifications: (offset = 0, limit = 20) =>
    request<OffsetPaginatedResponse<Notification[]>>(`/notifications?offset=${offset}&limit=${limit}`),
  getNotification: (id: number) =>
    request<ApiResponse<Notification>>(`/notifications/${id}`),
  markNotificationRead: (id: number) =>
    request<ApiResponse<{ message: string }>>(`/notifications/${id}/read`, { method: "POST" }),
  getUnreadStatus: () =>
    request<ApiResponse<{ has_unread: boolean }>>("/notifications/unread"),

  // Admin: Notifications
  adminListNotifications: (page = 1, perPage = 15) =>
    request<PaginatedResponse<Notification[]>>(`/admin/notifications?page=${page}&per_page=${perPage}`),
  adminCreateNotification: (data: {
    title: string;
    content: string;
    type: string;
    target_type: string;
    target_ids?: number[];
    visible_to_new?: boolean;
  }) =>
    request<ApiResponse<Notification>>("/admin/notifications", { method: "POST", body: JSON.stringify(data) }),
  adminUpdateNotification: (id: number, data: { title: string; content: string; type: string }) =>
    request<ApiResponse<Notification>>(`/admin/notifications/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  adminDeleteNotification: (id: number) =>
    request<ApiResponse<{ message: string }>>(`/admin/notifications/${id}`, { method: "DELETE" }),
  adminUploadNotificationImage: async (file: File) => {
    const formData = new FormData();
    formData.append("file", file);
    const res = await fetch(buildApiUrl("/admin/notifications/images"), {
      method: "POST",
      body: formData,
      credentials: "include",
    });
    if (!res.ok) {
      const body = await res.json().catch(() => ({ message: res.statusText }));
      throw new ApiError(body.message || res.statusText, body.message_key);
    }
    return res.json() as Promise<ApiResponse<{ id: number; url: string }>>;
  },
};
