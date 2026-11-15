import { useMutation, useQuery } from "@tanstack/react-query";
import { apiClient } from "@/shared/data";

// Hợp đồng thật: backend-go/api/openapi.yaml (`/admin/observability/*`, `/courses/{id}/observability/summary`). Không có cột nội dung ở bảng yêu cầu.
export type Totals = { requests: number; p50_ms: number; p95_ms: number; errors: number; error_rate: number; fallbacks: number; fallback_rate: number; cost: string; pii_masked: number; window_seconds: number };
export type DayRow = { day: string; requests: number; p95_ms: number; errors: number; escalations: number; pii_masked: number; cost: string };
export type TaskCost = { task: string; requests: number; cost: string; tokens_in: number; tokens_out: number };
export type AdminSummary = { window: string; totals: Totals; days: DayRow[]; tasks: TaskCost[] };
export type RequestRow = {
  id: string; at: string; task: string; lane: string; provider: string | null; model: string | null; latency_ms: number; status: string; error_kind: string | null;
  degraded: boolean; fallback_index: number; pii_masked_count: number; course_id: string | null; course_code: string | null;
};
export type Detail = { id: string; at: string; task: string; provider: string | null; model: string | null; status: string; trace_id: string; trace_url: string | null; masked_prompt: string; tools: string[]; confidence: string | null };
export type CourseSummary = {
  course_id: string; window: string; requests: number; error_rate: number; escalations: number; pii_masked: number; helpful: number; not_helpful: number; thread_correct: number; thread_reject: number;
  graded_changed: number; graded_total: number; change_rate: number; mean_abs_deviation: string; days: DayRow[];
};

export function useAdminSummary() {
  return useQuery({ queryKey: ["observability", "summary"], queryFn: async ({ signal }) => (await apiClient.get<AdminSummary>("/admin/observability/summary", { signal, query: { window: "24h" } })).data, refetchInterval: 60_000 });
}

export function useCourseSummary(courseId: string | null) {
  return useQuery({
    queryKey: ["observability", "course", courseId],
    enabled: courseId !== null,
    queryFn: async ({ signal }) => (await apiClient.get<CourseSummary>(`/courses/${courseId}/observability/summary`, { signal })).data,
  });
}

/** Mở nội dung một yêu cầu: POST có lý do; mỗi lần mở một dòng nhật ký, nên KHÔNG tự thử lại. */
export function useOpenRequest() {
  return useMutation({
    retry: false,
    mutationFn: async (v: { id: string; reason: string }) => (await apiClient.post<Detail>(`/admin/observability/requests/${v.id}/open`, { reason: v.reason })).data,
  });
}

export const STATUS_LABEL: Record<string, string> = {
  ok: "Thành công", degraded: "Trả lời tạm", error: "Lỗi", timeout: "Quá thời gian", rate_limited: "Bị giới hạn", overloaded: "Quá tải",
  cancelled: "Đã huỷ", circuit_open: "Nhà cung cấp tạm ngắt", budget_blocked: "Hết hạn mức", not_configured: "Chưa cấu hình",
};
export const TASK_LABEL: Record<string, string> = {
  CHAT: "Trả lời sinh viên", CLASSIFY: "Phân loại câu hỏi", UTILITY: "Tác vụ nền", GRADING: "Chấm bài", QUESTION_GEN: "Sinh câu hỏi", INSIGHT: "Báo cáo", EMBEDDING: "Đánh chỉ mục",
};

const vnd = new Intl.NumberFormat("vi-VN", { maximumFractionDigits: 0 });
export const money = (cost: string) => `${vnd.format(Math.round(Number(cost)))} đ`;
export const pct = (r: number) => `${(r * 100).toLocaleString("vi-VN", { maximumFractionDigits: 1 })}%`;
export const secs = (ms: number) => (ms >= 1000 ? `${(ms / 1000).toLocaleString("vi-VN", { maximumFractionDigits: 1 })} s` : `${Math.round(ms)} ms`);
