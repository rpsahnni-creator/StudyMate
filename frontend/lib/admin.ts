import { API_URL, fetchWithAuth } from "./auth";

export interface AdminUser {
  id: number;
  name: string;
  email: string;
  role: string;
  status: string;
  is_suspended: boolean;
  plan: string;
  subscription_status?: string;
  expires_at?: string;
  scan_count_today: number;
  created_at: string;
}

export interface UsersPage {
  items: AdminUser[];
  total: number;
  page: number;
  limit: number;
}

export interface AdminJob {
  id: number;
  user_email: string;
  status: string;
  progress: number;
  provider?: string;
  error_message?: string;
  created_at: string;
  updated_at: string;
  duration_ms: number;
}

export interface JobsPage {
  items: AdminJob[];
  total: number;
  page: number;
  limit: number;
}

export interface AICostRow {
  day: string;
  provider: string;
  model: string;
  request_count: number;
  total_tokens: number;
  total_cost_estimate: number;
}

export interface AICostSummary {
  total_cost_estimate: number;
  total_requests: number;
  total_tokens: number;
  avg_cost_per_request: number;
  rows: AICostRow[];
  from: string;
  to: string;
}

export interface AuditLogEntry {
  id: number;
  actor_email?: string;
  actor_user_id?: number;
  action: string;
  entity_type: string;
  entity_id?: string;
  created_at: string;
}

export interface AuditLogPage {
  items: AuditLogEntry[];
  total: number;
  page: number;
  limit: number;
}

export interface ContentFlag {
  id: number;
  question_id: number;
  content_hash?: string;
  reason: string;
  reported_by_email?: string;
  status: string;
  resolution_reason?: string;
  resolved_at?: string;
  created_at: string;
}

export interface ContentFlagsPage {
  items: ContentFlag[];
  total: number;
  page: number;
  limit: number;
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const res = await fetchWithAuth(`${API_URL}${path}`, options);
  if (!res.ok) {
    let message = `Request failed (${res.status})`;
    try {
      const data = (await res.json()) as { error?: string };
      message = data.error ?? message;
    } catch {
      // keep default
    }
    throw new Error(message);
  }
  return (await res.json()) as T;
}

export function getUsers(page: number, limit: number, search: string): Promise<UsersPage> {
  const params = new URLSearchParams({ page: String(page), limit: String(limit) });
  if (search) params.set("search", search);
  return request<UsersPage>(`/admin/users?${params.toString()}`);
}

export function suspendUser(userId: number, reason: string): Promise<{ status: string }> {
  return request(`/admin/users/${userId}/suspend`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ reason }),
  });
}

export function unsuspendUser(userId: number): Promise<{ status: string }> {
  return request(`/admin/users/${userId}/unsuspend`, { method: "PUT" });
}

export function getJobs(status: string, page: number, limit: number): Promise<JobsPage> {
  const params = new URLSearchParams({ page: String(page), limit: String(limit) });
  if (status) params.set("status", status);
  return request<JobsPage>(`/admin/jobs?${params.toString()}`);
}

export function retryJob(jobId: number): Promise<{ status: string }> {
  return request(`/admin/jobs/${jobId}/retry`, { method: "POST" });
}

export function getAICosts(from: string, to: string): Promise<AICostSummary> {
  const params = new URLSearchParams();
  if (from) params.set("from", from);
  if (to) params.set("to", to);
  return request<AICostSummary>(`/admin/ai-costs?${params.toString()}`);
}

export function getAuditLogs(page: number, limit: number, userId?: number): Promise<AuditLogPage> {
  const params = new URLSearchParams({ page: String(page), limit: String(limit) });
  if (userId) params.set("userId", String(userId));
  return request<AuditLogPage>(`/admin/audit-logs?${params.toString()}`);
}

export function getContentFlags(status: string, page: number, limit: number): Promise<ContentFlagsPage> {
  const params = new URLSearchParams({ page: String(page), limit: String(limit) });
  if (status) params.set("status", status);
  return request<ContentFlagsPage>(`/admin/content-flags?${params.toString()}`);
}

export function resolveContentFlag(
  flagId: number,
  action: "approved" | "removed",
  reason: string
): Promise<{ status: string }> {
  return request(`/admin/content-flags/${flagId}/resolve`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ action, reason }),
  });
}

// Format a timestamp in Indian Standard Time (IST).
export function formatIST(iso?: string): string {
  if (!iso) return "—";
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "—";
  return date.toLocaleString("en-IN", {
    timeZone: "Asia/Kolkata",
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function formatDuration(ms: number): string {
  if (ms <= 0) return "—";
  if (ms < 1000) return `${ms}ms`;
  const s = Math.floor(ms / 1000);
  if (s < 60) return `${s}s`;
  const m = Math.floor(s / 60);
  return `${m}m ${s % 60}s`;
}
