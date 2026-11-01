"use client";

import { useQuery } from "@tanstack/react-query";
import { apiClient, type CursorPage } from "@/shared/data";
import type { AdminCourse } from "@/features/admin/courses/api";

// Hợp đồng thật: backend-go/api/openapi.yaml (`/courses/{id}/integrations/{imap,smtp,teams}`).
// Mật khẩu IMAP và client secret của Teams là ô GHI-CHỈ: máy chủ không bao giờ trả về, frontend không lưu lại.
export type IntegrationStatus = "CONNECTED" | "NOT_CONNECTED" | "ERROR" | "PAUSED";
export type Security = "TLS" | "STARTTLS" | "NONE";
export type ImapConfig = { host: string; port: number; security: Security; username: string; mailbox: string; poll_seconds: number };
export type SmtpConfig = { host: string; port: number; security: Security; from: string };
export type TeamsConfig = { tenant_id: string; client_id: string };

export type IntegrationView<C> = {
  kind: "IMAP" | "SMTP" | "TEAMS";
  configured: boolean;
  enabled: boolean;
  status: IntegrationStatus;
  config: C | null;
  read_only: boolean;
  last_ok_at: string | null;
  last_sync_at: string | null;
  last_error: string | null;
  error_since: string | null;
  /** chỉ Teams */
  sandbox?: boolean;
  /** chỉ Teams: trường đã bật tích hợp Teams chưa */
  available?: boolean;
  version: number;
};

export type ImapInput = { host: string; port: number; security: Security; username: string; password?: string; mailbox?: string; poll_seconds?: number; enabled: boolean; version?: number };
export type TeamsInput = { tenant_id: string; client_id: string; client_secret?: string; enabled: boolean; version?: number };
export type TestOut = { ok: boolean; kind?: string };

export const SECURITY_LABEL: Record<Security, string> = { TLS: "TLS", STARTTLS: "STARTTLS", NONE: "Không" };
export const STATUS_TEXT: Record<IntegrationStatus, string> = { CONNECTED: "Đã kết nối", NOT_CONNECTED: "Chưa kết nối", ERROR: "Lỗi", PAUSED: "Tạm dừng" };
export const STATUS_TONE: Record<IntegrationStatus, "green" | "neutral" | "red" | "amber"> = { CONNECTED: "green", NOT_CONNECTED: "neutral", ERROR: "red", PAUSED: "amber" };

/** Nguyên nhân hỏng đang kéo dài (dòng dưới trạng thái). */
export const ERROR_SHORT: Record<string, string> = {
  UNREACHABLE: "Không kết nối được",
  AUTH: "Sai tài khoản hoặc mật khẩu",
  TLS: "Chứng chỉ không hợp lệ",
  MAILBOX: "Không có hộp thư",
  TIMEOUT: "Máy chủ không phản hồi",
  TEAMS_AUTH_FAILED: "Không đăng nhập được",
  TEAMS_CONSENT_REQUIRED: "Chưa được cấp quyền",
  TEAMS_RESPONSE_INVALID: "Teams trả dữ liệu lạ",
};

/** Kết quả bấm `Test kết nối` của IMAP. */
export const IMAP_TEST_TEXT: Record<string, string> = {
  UNREACHABLE: "Không kết nối được máy chủ",
  AUTH: "Tài khoản hoặc mật khẩu không đúng",
  TLS: "Chứng chỉ máy chủ không hợp lệ",
  MAILBOX: "Không có hộp thư này",
  TIMEOUT: "Máy chủ không phản hồi kịp",
};

export const TEAMS_TEST_TEXT: Record<string, string> = {
  TEAMS_AUTH_FAILED: "Tenant hoặc client secret không đúng",
  TEAMS_CONSENT_REQUIRED:
    "Cần quản trị Microsoft 365 của trường cấp quyền đọc bài tập và danh sách lớp (EduAssignments, EduRoster). Gửi họ liên kết cấp quyền trong thư mời của ứng dụng.",
  TEAMS_RESPONSE_INVALID: "Teams trả dữ liệu không đúng",
  UNREACHABLE: "Không kết nối được máy chủ",
  TIMEOUT: "Máy chủ không phản hồi kịp",
};

const base = (course: string, kind: "imap" | "smtp" | "teams") => `/courses/${course}/integrations/${kind}`;
export const integrationKey = (course: string, kind: "imap" | "smtp" | "teams") => ["integrations", kind, course] as const;

const useIntegration = <C,>(course: string, kind: "imap" | "smtp" | "teams") =>
  useQuery({
    queryKey: integrationKey(course, kind),
    enabled: Boolean(course),
    retry: false,
    queryFn: async ({ signal }) => (await apiClient.get<IntegrationView<C>>(base(course, kind), { signal })).data,
  });

export const useImap = (course: string) => useIntegration<ImapConfig>(course, "imap");
export const useSmtp = (course: string) => useIntegration<SmtpConfig>(course, "smtp");
export const useTeams = (course: string) => useIntegration<TeamsConfig>(course, "teams");

export const saveImap = async (course: string, body: ImapInput) => (await apiClient.put<IntegrationView<ImapConfig>>(base(course, "imap"), body)).data;
export const testImap = async (course: string) => (await apiClient.post<TestOut>(`${base(course, "imap")}/test`)).data;
export const syncImapNow = async (course: string, key: string) =>
  (await apiClient.post<{ job_id: string }>(`${base(course, "imap")}/sync-now`, {}, { idempotencyKey: key })).data;

export const testSmtp = async (course: string) => (await apiClient.post<{ queued: boolean }>(`${base(course, "smtp")}/test`)).data;

export const saveTeams = async (course: string, body: TeamsInput) => (await apiClient.put<IntegrationView<TeamsConfig>>(base(course, "teams"), body)).data;
export const testTeams = async (course: string) => (await apiClient.post<TestOut>(`${base(course, "teams")}/test`)).data;

/** Lớp để Admin chọn (Admin không có lớp trong `/me/courses`). */
export const adminCoursesKey = ["integrations", "courses"] as const;
export const useAdminCourses = (enabled: boolean) =>
  useQuery({
    queryKey: adminCoursesKey,
    enabled,
    queryFn: async ({ signal }) => (await apiClient.get<CursorPage<AdminCourse>>("/admin/courses", { signal, query: { limit: 100 } })).data.items,
  });
