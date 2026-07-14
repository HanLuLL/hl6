// AI 审查系统类型定义

export interface AIModelConfig {
  id: number;
  name: string;
  provider: string;
  api_base_url: string;
  api_key: string; // 脱敏后的 API Key
  model_name: string;
  is_default: boolean;
  is_enabled: boolean;
  max_tokens: number;
  temperature: number;
  rate_limit_rpm: number;
  created_at: string;
  updated_at: string;
}

export interface AIModelConfigInput {
  name: string;
  provider?: string;
  api_base_url: string;
  api_key: string;
  model_name: string;
  is_default?: boolean;
  is_enabled?: boolean;
  max_tokens?: number;
  temperature?: number;
  rate_limit_rpm?: number;
}

export interface AuditPromptTemplate {
  id: number;
  name: string;
  is_default: boolean;
  is_enabled: boolean;
  sort_order: number;
  system_prompt: string;
  user_prompt: string;
  description: string;
  created_by: number;
  updated_by?: number;
  created_at: string;
  updated_at: string;
}

export interface PromptTemplateInput {
  name: string;
  is_default?: boolean;
  is_enabled?: boolean;
  sort_order?: number;
  system_prompt: string;
  user_prompt: string;
  description?: string;
}

export interface AuditAIReview {
  id: number;
  subdomain_id: number;
  scan_id?: number;
  fqdn: string;
  model_config_id: number;
  prompt_template_id?: number;
  input_content: string;
  ai_response: string;
  ai_judgment: "clean" | "violation" | "error";
  violation_types: string[];
  ai_confidence: number;
  ai_suggested_action: string;
  admin_review_status: "pending" | "confirmed" | "overturned" | "dismissed";
  admin_reviewed_by?: number;
  admin_reviewed_at?: string;
  admin_note: string;
  final_action: string;
  tokens_used: number;
  created_at: string;
  updated_at: string;
}

export interface AIAuditStats {
  total_reviews: number;
  pending_reviews: number;
  violation_count: number;
  clean_count: number;
  pending_appeals: number;
}

export interface UserAppeal {
  id: number;
  user_id: number;
  review_id?: number;
  content: string;
  status: "pending" | "approved" | "rejected";
  reviewed_by?: number;
  reviewed_at?: string;
  reply: string;
  created_at: string;
  updated_at: string;
}

export interface BanInfo {
  banned: boolean;
  reason?: string;
  banned_at?: string;
  banned_until?: string;
}
