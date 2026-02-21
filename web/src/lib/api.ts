import type { ApiResponse, PaginatedResponse } from "@/types";

let getAccessToken: (() => Promise<string>) | null = null;

export function setTokenGetter(fn: () => Promise<string>) {
  getAccessToken = fn;
}

const BASE_URL = import.meta.env.VITE_API_BASE_URL || "/api/v1";

async function request<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string>),
  };

  if (getAccessToken) {
    try {
      const token = await getAccessToken();
      if (token) {
        headers["Authorization"] = `Bearer ${token}`;
      }
    } catch {
      // token fetch failed, proceed without auth
    }
  }

  const res = await fetch(`${BASE_URL}${path}`, {
    ...options,
    headers,
  });

  if (!res.ok) {
    const body = await res.json().catch(() => ({ message: res.statusText }));
    throw new Error(body.message || res.statusText);
  }

  return res.json();
}

export const api = {
  // Auth
  getMe: () => request<ApiResponse<{ user: import("@/types").User; credits: number }>>("/auth/me"),
  syncUser: (data: { email: string; name: string; avatar_url: string }) =>
    request<ApiResponse<import("@/types").User>>("/auth/sync", { method: "POST", body: JSON.stringify(data) }),

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
  adminCreateDomain: (data: { name: string; cloudflare_zone_id: string; credit_cost: number; description: string }) =>
    request<ApiResponse<import("@/types").Domain>>("/admin/domains", { method: "POST", body: JSON.stringify(data) }),
  adminUpdateDomain: (id: number, data: Partial<import("@/types").Domain>) =>
    request<ApiResponse<import("@/types").Domain>>(`/admin/domains/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  adminListCloudflareZones: () =>
    request<ApiResponse<import("@/types").CloudflareZone[]>>("/admin/cloudflare/zones"),
  adminGrantCredits: (data: { user_id: number; amount: number; description: string }) =>
    request<ApiResponse<{ user_id: number; granted: number; balance: number }>>("/admin/credits/grant", { method: "POST", body: JSON.stringify(data) }),
  adminListUsers: (page = 1, perPage = 20) =>
    request<PaginatedResponse<import("@/types").User[]>>(`/admin/users?page=${page}&per_page=${perPage}`),
  adminGetStats: () => request<ApiResponse<import("@/types").Stats>>("/admin/stats"),
  adminListAuditLogs: (page = 1, perPage = 20) =>
    request<PaginatedResponse<import("@/types").AuditLog[]>>(`/admin/audit-logs?page=${page}&per_page=${perPage}`),
};
