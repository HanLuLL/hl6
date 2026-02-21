export interface User {
  id: number;
  logto_id: string;
  email: string;
  name: string;
  avatar_url: string;
  role: "user" | "admin";
  created_at: string;
  updated_at: string;
}

export interface Domain {
  id: number;
  name: string;
  cloudflare_zone_id: string;
  credit_cost: number;
  is_active: boolean;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface Subdomain {
  id: number;
  domain_id: number;
  user_id: number;
  name: string;
  fqdn: string;
  domain: Domain;
  dns_records?: DNSRecord[];
  created_at: string;
  updated_at: string;
}

export interface DNSRecord {
  id: number;
  subdomain_id: number;
  type: "A" | "CNAME" | "AAAA";
  name: string;
  content: string;
  ttl: number;
  proxied: boolean;
  cloudflare_record_id: string;
  created_at: string;
  updated_at: string;
}

export interface CreditBalance {
  balance: number;
  transactions: CreditTransaction[];
}

export interface CreditTransaction {
  id: number;
  user_id: number;
  amount: number;
  type: "grant" | "deduct" | "refund";
  description: string;
  balance_after: number;
  created_at: string;
}

export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

export interface PaginatedResponse<T> extends ApiResponse<T> {
  total: number;
  page: number;
  per_page: number;
}

export interface Stats {
  users: number;
  domains: number;
  subdomains: number;
  dns_records: number;
}

export interface AuditLog {
  id: number;
  user_id: number;
  action: string;
  resource: string;
  resource_id: number;
  details: Record<string, unknown>;
  user?: User;
  created_at: string;
}
