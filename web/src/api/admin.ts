import { api, type ApiResponse } from "./client"

export interface Domain {
  id: number
  bind_domain: string
  target_url: string
  real_ip_headers: string
  forward_ip_header: string
  request_transform: string
  response_transform: string
  rewrite_host: boolean
  rewrite_mode: string
  is_default: boolean
  created_at: string
  updated_at: string
}

export interface Rule {
  id: number
  domain_id: number
  name: string
  path_prefix: string
  methods: string
  rule_type: "duplicate_ip" | "rate_limit"
  identity_fields: string
  max_attempts: number
  window_seconds: number
  block_seconds: number
  block_status: number
  block_response: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface ProxyLog {
  id: number
  bind_domain: string
  client_ip: string
  method: string
  path: string
  query_params?: Record<string, string[]>
  request_headers?: Record<string, string[]>
  request_body?: string
  user_agent?: string
  target_url: string
  status_code?: number
  blocked: boolean
  rule_id?: number
  rule_name?: string
  message: string
  created_at: string
}

export const login = (username: string, password: string) =>
  api.post<ApiResponse<{ access_token: string; token_type: string }>>("/admin/login", { username, password })

export const logout = () => api.post<ApiResponse<null>>("/admin/logout")

export const me = () => api.get<ApiResponse<{ username: string }>>("/admin/me")

export const listDomains = () => api.get<ApiResponse<Domain[]>>("/admin/domains")

export const createDomain = (data: Partial<Domain>) =>
  api.post<ApiResponse<Domain>>("/admin/domains", data)

export const updateDomain = (id: number, data: Partial<Domain>) =>
  api.put<ApiResponse<null>>(`/admin/domains/${id}`, data)

export const deleteDomain = (id: number) =>
  api.delete<ApiResponse<null>>(`/admin/domains/${id}`)

export const listRules = (domainId: number) =>
  api.get<ApiResponse<Rule[]>>("/admin/rules", { params: { domain_id: domainId } })

export const createRule = (data: Partial<Rule>) =>
  api.post<ApiResponse<Rule>>("/admin/rules", data)

export const updateRule = (id: number, data: Partial<Rule>) =>
  api.put<ApiResponse<null>>(`/admin/rules/${id}`, data)

export const deleteRule = (id: number) =>
  api.delete<ApiResponse<null>>(`/admin/rules/${id}`)

export const listLogs = (page = 1, pageSize = 20) =>
  api.get<ApiResponse<{ list: ProxyLog[]; total: number; page: number; page_size: number }>>("/admin/logs", {
    params: { page, page_size: pageSize },
  })

export interface TopIP {
  client_ip: string
  count: number
}

export interface TopRule {
  rule_id: number
  rule_name: string
  count: number
}

export interface DailyTrend {
  date: string
  count: number
}

export interface BlockedStats {
  total_blocked: number
  today_blocked: number
  unique_ips: number
  active_rules: number
  top_ips: TopIP[]
  top_rules: TopRule[]
  daily_trend: DailyTrend[]
}

export const getBlockedStats = () => api.get<ApiResponse<BlockedStats>>("/admin/logs/stats")

export const listBlockedLogs = (page = 1, pageSize = 20) =>
  api.get<ApiResponse<{ list: ProxyLog[]; total: number; page: number; page_size: number }>>("/admin/blocks", {
    params: { page, page_size: pageSize },
  })
