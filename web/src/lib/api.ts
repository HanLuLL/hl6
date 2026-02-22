import type { ApiResponse, PaginatedResponse } from "@/types";

const BASE_URL = "/api/v1";

export class ApiError extends Error {
  messageKey?: string;
  constructor(message: string, messageKey?: string) {
    super(message);
    this.messageKey = messageKey;
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

  const res = await fetch(`${BASE_URL}${path}`, {
    ...options,
    headers,
    credentials: "include",
  });

  if (res.status === 401) {
    // Don't auto-redirect for /auth/me — let the app handle auth state
    if (!path.includes("/auth/me")) {
      window.location.href = "/api/v1/auth/login";
    }
    throw new ApiError("Not authenticated", "error.missingToken");
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({ message: res.statusText }));
    throw new ApiError(body.message || res.statusText, body.message_key);
  }

  return res.json();
}

export const api = {
  // Auth
  getMe: () => request<ApiResponse<{ user: import("@/types").User; credits: number }>>("/auth/me"),
  logout: () => request<ApiResponse<{ logout_url: string }>>("/auth/logout", { method: "POST" }),

  // Domains
  listDomains: () => request<ApiResponse<import("@/types").Domain[]>>("/domains"),

  // Subdomains
  listSubdomains: () => request<ApiResponse<import("@/types").Subdomain[]>>("/subdomains"),
  getSubdomain: (id: number) => request<ApiResponse<import("@/types").Subdomain>>(`/subdomains/${id}`),
  claimSubdomain: (data: { domain_id: number; name: string }) =>
    request<ApiResponse<import("@/types").Subdomain>>("/subdomains", { method: "POST", body: JSON.stringify(data) }),
  releaseSubdomain: (id: number) =>
    request<ApiResponse<{ message: string }>>(`/subdomains/${id}`, { method: "DELETE" }),

  // DNS Records
  listRecords: (subdomainId: number) =>
    request<ApiResponse<import("@/types").DNSRecord[]>>(`/subdomains/${subdomainId}/records`),
  createRecord: (subdomainId: number, data: { type: string; content: string; ttl?: number; proxied?: boolean }) =>
    request<ApiResponse<import("@/types").DNSRecord>>(`/subdomains/${subdomainId}/records`, { method: "POST", body: JSON.stringify(data) }),
  updateRecord: (subdomainId: number, recordId: number, data: { content: string; ttl?: number; proxied?: boolean }) =>
    request<ApiResponse<import("@/types").DNSRecord>>(`/subdomains/${subdomainId}/records/${recordId}`, { method: "PUT", body: JSON.stringify(data) }),
  deleteRecord: (subdomainId: number, recordId: number) =>
    request<ApiResponse<{ message: string }>>(`/subdomains/${subdomainId}/records/${recordId}`, { method: "DELETE" }),

  // Credits
  getCredits: () => request<ApiResponse<import("@/types").CreditBalance>>("/credits"),
  listTransactions: (page = 1, perPage = 20) =>
    request<PaginatedResponse<import("@/types").CreditTransaction[]>>(`/credits/transactions?page=${page}&per_page=${perPage}`),

  // Admin
  adminCreateDomain: (data: { name: string; cloudflare_zone_id: string; cloudflare_account_id: number; description: string; group_access: { group_id: number; credit_cost: number; max_dns_records?: number | null }[] }) =>
    request<ApiResponse<{ domain: import("@/types").Domain; group_access: import("@/types").DomainGroupAccess[] }>>("/admin/domains", { method: "POST", body: JSON.stringify(data) }),
  adminUpdateDomain: (id: number, data: { cloudflare_zone_id?: string; cloudflare_account_id?: number; is_active?: boolean; description?: string; group_access?: { group_id: number; credit_cost: number; max_dns_records?: number | null }[] }) =>
    request<ApiResponse<{ domain: import("@/types").Domain; group_access: import("@/types").DomainGroupAccess[] }>>(`/admin/domains/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  adminDeleteDomain: (id: number) =>
    request<ApiResponse<{ message: string }>>(`/admin/domains/${id}`, { method: "DELETE" }),
  adminListDomainsFull: () =>
    request<ApiResponse<import("@/types").DomainWithGroupAccess[]>>("/admin/domains-full"),
  adminListCloudflareZones: (accountId: number) =>
    request<ApiResponse<import("@/types").CloudflareZone[]>>(`/admin/cloudflare/accounts/${accountId}/zones`),
  adminGrantCredits: (data: { user_id: number; amount: number; description: string }) =>
    request<ApiResponse<{ user_id: number; granted: number; balance: number }>>("/admin/credits/grant", { method: "POST", body: JSON.stringify(data) }),
  adminListUsers: (page = 1, perPage = 20) =>
    request<PaginatedResponse<import("@/types").User[]>>(`/admin/users?page=${page}&per_page=${perPage}`),
  adminGetStats: () => request<ApiResponse<import("@/types").Stats>>("/admin/stats"),
  adminListAuditLogs: (page = 1, perPage = 20) =>
    request<PaginatedResponse<import("@/types").AuditLog[]>>(`/admin/audit-logs?page=${page}&per_page=${perPage}`),

  // Admin: User Groups
  adminListGroups: () =>
    request<ApiResponse<import("@/types").UserGroup[]>>("/admin/groups"),
  adminCreateGroup: (data: { name: string; is_default?: boolean }) =>
    request<ApiResponse<import("@/types").UserGroup>>("/admin/groups", { method: "POST", body: JSON.stringify(data) }),
  adminUpdateGroup: (id: number, data: { name?: string; is_default?: boolean }) =>
    request<ApiResponse<import("@/types").UserGroup>>(`/admin/groups/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  adminDeleteGroup: (id: number, migrateTo: number) =>
    request<ApiResponse<{ message: string }>>(`/admin/groups/${id}?migrate_to=${migrateTo}`, { method: "DELETE" }),
  adminUpdateUserGroup: (userId: number, groupId: number) =>
    request<ApiResponse<{ message: string }>>(`/admin/users/${userId}/group`, { method: "PUT", body: JSON.stringify({ group_id: groupId }) }),

  // Admin: System Config
  adminGetConfig: () =>
    request<ApiResponse<Record<string, string>>>("/admin/config"),
  adminUpdateConfig: (data: Record<string, string>) =>
    request<ApiResponse<{ message: string }>>("/admin/config", { method: "PUT", body: JSON.stringify(data) }),

  // Admin: Cloudflare Accounts
  adminListCloudflareAccounts: () =>
    request<ApiResponse<import("@/types").CloudflareAccount[]>>("/admin/cloudflare/accounts"),
  adminCreateCloudflareAccount: (data: { name: string; api_token: string }) =>
    request<ApiResponse<import("@/types").CloudflareAccount>>("/admin/cloudflare/accounts", { method: "POST", body: JSON.stringify(data) }),
  adminUpdateCloudflareAccount: (id: number, data: { name: string; api_token?: string }) =>
    request<ApiResponse<import("@/types").CloudflareAccount>>(`/admin/cloudflare/accounts/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  adminDeleteCloudflareAccount: (id: number) =>
    request<ApiResponse<{ message: string }>>(`/admin/cloudflare/accounts/${id}`, { method: "DELETE" }),
};
