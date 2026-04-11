export interface UserGroup {
  id: number;
  name: string;
  is_default: boolean;
  user_count?: number;
  created_at: string;
  updated_at: string;
}

export interface User {
  id: number;
  external_id: string;
  referral_code?: string;
  email: string;
  name: string;
  avatar_url: string;
  role: "user" | "admin";
  is_banned: boolean;
  banned_reason?: string;
  banned_at?: string;
  banned_by?: number;
  group_id?: number;
  group?: UserGroup;
  created_at: string;
  updated_at: string;
}

export interface Domain {
  id: number;
  name: string;
  cloudflare_zone_id: string;
  cloudflare_account_id: number;
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
  type: "A" | "CNAME" | "AAAA" | "TXT";
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

export interface DailyCheckinStatus {
  enabled: boolean;
  reward: number;
  claimed_today: boolean;
  checkin_date: string;
}

export interface DailyCheckinClaimResult {
  granted: number;
  balance: number;
  claimed_today: boolean;
  checkin_date: string;
}

export interface CreditTransaction {
  id: number;
  user_id: number;
  amount: number;
  type: "grant" | "deduct" | "refund";
  description_key: string;
  description_params?: Record<string, string>;
  balance_after: number;
  created_at: string;
}

export interface ApiResponse<T> {
  code: number;
  message: string;
  message_key?: string;
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
  user_groups: number;
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

export interface CloudflareZone {
  id: string;
  name: string;
  status: string;
}

export interface CloudflareAccount {
  id: number;
  name: string;
  token_hint: string;
  created_at: string;
  updated_at: string;
}

export interface DomainGroupAccess {
  id: number;
  domain_id: number;
  group_id: number;
  credit_cost: number;
  max_dns_records?: number | null;
  group?: UserGroup;
  created_at: string;
  updated_at: string;
}

export interface DomainWithGroupAccess extends Domain {
  group_access: DomainGroupAccess[];
}

export interface Notification {
  id: number;
  title: string;
  content: string;
  type: "normal" | "urgent" | "pinned";
  target_type: "users" | "groups" | "all";
  target_ids?: number[];
  visible_to_new: boolean;
  created_by: number;
  creator?: User;
  is_read?: boolean;
  read_count?: number;
  created_at: string;
  updated_at?: string;
}

export interface OffsetPaginatedResponse<T> {
  code: number;
  message: string;
  data: T;
  total: number;
  offset: number;
  limit: number;
}

export interface ReferralRecord {
  id: number;
  invitee_name: string;
  invitee_created_at: string;
  inviter_credits: number;
  created_at: string;
}

export interface ReferralInfo {
  referral_code: string;
  referral_enabled: boolean;
  records: ReferralRecord[];
  total: number;
  page: number;
  per_page: number;
}

export interface UserWithInviter extends User {
  credits: number;
  invited_by?: {
    id: number;
    name: string;
    email: string;
  };
}

export interface AdminDNSRecord extends DNSRecord {
  user_id: number;
  user_email: string;
  user_name: string;
  domain_id: number;
  domain_name: string;
}

export interface AdminClaimedSubdomain {
  id: number;
  domain_id: number;
  user_id: number;
  fqdn: string;
  domain_name: string;
  user_email: string;
  dns_record_count: number;
  created_at: string;
}

export interface ReservedSubdomainPrefixSettings {
  prefixes: string[];
  min_length: number;
  max_length: number;
}

export interface SubdomainLengthSettings {
  min_length: number;
  max_length: number;
}

export interface BrandingResponse {
  name: string;
  logo_url: string | null;
  favicon_url: string | null;
  version: string;
}

export interface AdminURLRuntime {
  frontend_urls: string[];
  backend_urls: string[];
  frontend_url: string;
  backend_url: string;
  frontend_source: "env" | "db" | "auto" | "fallback";
  backend_source: "env" | "db" | "auto" | "fallback";
  frontend_env_locked: boolean;
  backend_env_locked: boolean;
  confirmed: boolean;
}

export interface AdminOIDCRuntime {
  configured: boolean;
  missing_fields: string[];
  issuer: string;
  client_id: string;
  issuer_source: "env" | "db" | "none";
  client_id_source: "env" | "db" | "none";
  client_secret_source: "env" | "db" | "none";
  issuer_env_locked: boolean;
  client_id_env_locked: boolean;
  client_secret_env_locked: boolean;
  client_secret_configured: boolean;
}

export interface AdminConfigPayload {
  values: Record<string, string>;
  url_runtime: AdminURLRuntime;
  oidc_runtime: AdminOIDCRuntime;
}

export interface OIDCStatusPayload extends AdminOIDCRuntime {
  setup_allowed: boolean;
}
