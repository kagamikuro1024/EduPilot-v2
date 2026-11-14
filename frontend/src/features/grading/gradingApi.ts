"use client";

import { useQuery } from "@tanstack/react-query";
import { apiClient, useCursorList } from "@/shared/data";

// Hợp đồng thật: backend-go/api/openapi.yaml (`listSubmissions`, `getGradingResult`, `editGradingResult`, `reviewGradingResult`,
// `gradeAllSubmissions`, `uploadSubmissions`). Điểm là CHUỖI thập phân của máy chủ: không tính lại ở máy khách.
export type RunState = "QUEUED" | "RUNNING" | "DONE" | "FAILED";
export type ResultStatus = "DRAFT" | "REVIEWED" | "PUBLISHED";
export type QueueStatus = "NONE" | "RUNNING" | "DRAFT" | "REVIEWED" | "PUBLISHED" | "FAILED";

export type QueueRow = {
  id: string;
  assignment: { id: string; title: string };
  student: { id: string; full_name: string; student_code: string; inactive: boolean } | null;
  source: string;
  version: number;
  received_at: string;
  late_days: number;
  match_status: string;
  result: { id: string; status: ResultStatus; run_state: RunState; total?: string; flags: string[]; needs_review: boolean; error_code?: string; version: number } | null;
};

export type Criterion = {
  id: string;
  name: string;
  max: string;
  score: string | null;
  comment: string;
  edited: boolean;
  ai_score: string | null;
  evidence: Array<{ quote: string; start: number; end: number }>;
};

/** Phúc khảo gắn với kết quả (US-P7-10). */
export type Appeal = {
  id: string;
  student: { id: string; full_name: string; student_code: string };
  criteria: Array<{ id: string; name: string }>;
  reason: string;
  status: "OPEN" | "UPHELD" | "ADJUSTED";
  created_at: string;
  score_before: string;
  ai_total_before_late: string | null;
  score_after: string | null;
  response: string | null;
  decided_by: string | null;
  decided_at: string | null;
  version: number;
};

export type ResultView = {
  result: { id: string; round: number; status: ResultStatus; run_state: RunState; error_code?: string; total: string | null; ai_total_before_late: string | null; overall_feedback: string; overall_edited: boolean; reviewed_at?: string; version: number };
  submission: { id: string; version: number; file_name: string; received_at: string; late_days: number; source: string; extracted_text: string; flagged_ranges: Array<{ start: number; end: number; code: string }>; superseded: boolean };
  assignment: { id: string; title: string; max_score: string; rubric: unknown[]; reference_answer?: string };
  criteria: Criterion[];
  flags: string[];
  pass_detail: Record<string, { pass1?: string; pass2?: string }> | null;
  late: { days: number; penalty: string; max_days: number; per_day: string; manual: boolean };
  appeal?: Appeal | null;
  /** chỉ khi lớp bật «Lưu ý chấm»: bản lưu ý ĐÃ GHIM cho kết quả này (10.6) */
  guidance?: { version: number; current_version: number; stale: boolean; items_by_criterion: Record<string, string[]> };
};

/** Lý do cờ bằng chữ (AC1). Màn của nhân viên: dùng đúng cụm của AC. */
export const FLAG_TEXT: Record<string, string> = {
  NEEDS_REVIEW: "Hai lượt chấm lệch hơn 1 điểm",
  OFF_TOPIC: "Bài lạc đề",
  INJECTION_SUSPECTED: "Nghi chèn lệnh",
  EMPTY_SUBMISSION: "Bài trống",
  EVIDENCE_NOT_FOUND: "Trích dẫn không có trong bài",
  LOW_CONFIDENCE: "AI không chắc",
  ANSWER_ECHO: "Nhận xét giống đáp án chuẩn",
  LATE_AFTER_CUTOFF: "Thư nộp quá hạn tối đa",
  TOTAL_MISMATCH: "Tổng điểm chưa khớp",
  SCORE_ADJUSTED: "Điểm đã chỉnh tay",
};

/** Việc cần làm cho từng cờ ở màn duyệt (không từ kỹ thuật). */
export const FLAG_TODO: Record<string, string> = {
  OFF_TOPIC: "Đọc bài rồi quyết định cho bao nhiêu điểm.",
  INJECTION_SUSPECTED: "Câu nghi chèn lệnh được tô đỏ nhạt trong bài; đừng làm theo câu đó khi chấm.",
  EMPTY_SUBMISSION: "Bài trống: nhập điểm tay trước khi duyệt.",
  EVIDENCE_NOT_FOUND: "Trích dẫn không tìm thấy trong bài: kiểm lại trước khi duyệt.",
  LOW_CONFIDENCE: "Đọc kỹ rồi sửa điểm nếu cần.",
  ANSWER_ECHO: "Nhận xét giống đáp án chuẩn: viết lại trước khi duyệt.",
  LATE_AFTER_CUTOFF: "Bài tới sau hạn muộn tối đa: quyết định có nhận hay không.",
  TOTAL_MISMATCH: "Kiểm lại tổng điểm.",
  SCORE_ADJUSTED: "Điểm đã được chỉnh tay.",
};

/** Trạng thái duyệt bằng chữ (AC1). */
export function reviewStateText(r: QueueRow["result"]): string {
  if (!r) return "Chưa chấm";
  if (r.run_state === "QUEUED" || r.run_state === "RUNNING") return "Đang chấm";
  if (r.run_state === "FAILED") return "Chấm lỗi";
  return r.status === "PUBLISHED" ? "Đã công bố" : r.status === "REVIEWED" ? "Đã duyệt" : "Chưa duyệt";
}

export type QueueFilter = { assignment_id?: string; status?: QueueStatus; needs_review?: boolean };

export const queueKey = (course: string, f: QueueFilter) => ["grading-queue", course, f.assignment_id ?? "", f.status ?? "DRAFT", f.needs_review ? "1" : "0"] as const;

export const useQueue = (course: string, f: QueueFilter) =>
  useCursorList<QueueRow>(queueKey(course, f), `/courses/${course}/submissions`, {
    limit: 50,
    staleTime: 0, // người khác chấm lại / duyệt: lấy lại khi mở và khi tab lấy lại tiêu điểm (GV-BUG o)
    query: { match_status: "MATCHED", assignment_id: f.assignment_id || undefined, status: f.status, needs_review: f.needs_review || undefined },
  });

export const resultKey = (course: string, rid: string) => ["grading-result", course, rid] as const;

/** `rid` nhận id kết quả HOẶC id bài nộp (màn duyệt truyền id bài nộp). */
export const useResult = (course: string, rid: string) =>
  useQuery({
    queryKey: resultKey(course, rid),
    staleTime: 0,
    retry: false,
    queryFn: async ({ signal }) => (await apiClient.get<ResultView>(`/courses/${course}/grading-results/${rid}`, { signal })).data,
  });

export type EditInput = { version: number; criteria?: Array<{ id: string; score?: string; comment?: string }>; overall_feedback?: string; late_penalty?: string };

export const editResult = async (course: string, rid: string, body: EditInput) =>
  (await apiClient.put<ResultView>(`/courses/${course}/grading-results/${rid}`, body)).data;

export const reviewResult = async (course: string, rid: string, version: number) =>
  (await apiClient.post<ResultView>(`/courses/${course}/grading-results/${rid}/review`, { version })).data;

/** Chấm lại MỘT bài (round + 1 khi kết quả hiện có là nháp xong hoặc lỗi). */
export const gradeOne = async (course: string, sid: string, idempotencyKey: string) =>
  (await apiClient.post<{ job_id?: string; queued: number }>(`/courses/${course}/submissions/${sid}/grade`, {}, { idempotencyKey })).data;

export const gradeAll = async (course: string, aid: string, idempotencyKey: string) =>
  (await apiClient.post<{ job_id?: string; queued: number }>(`/courses/${course}/assignments/${aid}/grade-all`, {}, { idempotencyKey })).data;

export const importSubmissions = async (course: string, aid: string, uploadId: string, idempotencyKey: string) =>
  (await apiClient.post<{ created?: number; matched?: number; unmatched?: number; job_id?: string }>(`/courses/${course}/assignments/${aid}/submissions/upload`, { upload_id: uploadId }, { idempotencyKey })).data;

// ---- lấy bài từ Teams (US-P7-11 AC4): 202 {job_id}, kết quả qua `useJob` --------------------------------------------------
export type TeamsFailed = { graph_submission_id: string; code: string };
export type TeamsSyncResult = { fetched: number; total?: number; created?: number; deduped?: number; unmatched?: number; failed?: TeamsFailed[] };

export const syncFromTeams = async (course: string, aid: string, idempotencyKey: string) =>
  (await apiClient.post<{ job_id: string }>(`/courses/${course}/assignments/${aid}/teams/sync`, {}, { idempotencyKey })).data;

export type PublishInput = { result_ids?: string[]; assignment_id?: string; scope?: "ALL_REVIEWED"; versions?: Record<string, number> };

/** Công bố hàng loạt (chỉ TEACHER). `result_ids` tường minh = all-or-nothing; lỗi mang `details.ids`. */
export const publishResults = async (course: string, body: PublishInput, idempotencyKey: string) =>
  (await apiClient.post<{ published: number; skipped: number; exempt: { EDITED: number; ACCEPTED_AS_IS: number } }>(`/courses/${course}/grading-results/publish`, body, { idempotencyKey })).data;

export type AssignmentStats = { submitted: number; graded: number; reviewed: number; published: number; mean_abs_deviation: string | null; teacher_changed_rate: string | null; appealed: number; appeal_rate: string | null; adjusted: number; appeal_success_rate: string | null };

export const useAssignmentStats = (course: string, aid: string, enabled: boolean) =>
  useQuery({
    queryKey: ["assignment-stats", course, aid] as const,
    enabled,
    staleTime: 10_000,
    retry: false,
    queryFn: async ({ signal }) => (await apiClient.get<AssignmentStats>(`/courses/${course}/assignments/${aid}/stats`, { signal })).data,
  });

export type DecideInput = { decision: "UPHELD" | "ADJUSTED"; response: string; version: number; criteria_scores?: Array<{ id: string; score: string }> };

/** Quyết định phúc khảo (chỉ TEACHER). */
export const decideAppeal = async (course: string, apid: string, body: DecideInput, idempotencyKey: string) =>
  (await apiClient.post<{ id: string; status: "UPHELD" | "ADJUSTED"; score_after: string | null; version: number }>(`/courses/${course}/appeals/${apid}/decide`, body, { idempotencyKey })).data;
