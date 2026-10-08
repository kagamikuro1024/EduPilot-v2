import type { StatusTone } from "@/shared/ui";

// Hợp đồng thật: backend-go/api/openapi.yaml (`/courses/{id}/exams…`, thao tác 17–25, 27, 28 của SRS FEAT-weekly-exam 6.2).
export type ExamStatus = "DRAFT" | "SCHEDULED" | "OPEN" | "CLOSED" | "PUBLISHED";
export type ExamKind = "MCQ" | "CODE" | "MIXED";

export type ExamRow = {
  id: string;
  title: string;
  kind: ExamKind;
  status: ExamStatus;
  effective_status: ExamStatus;
  opens_at: string | null;
  closes_at: string | null;
  duration_minutes: number | null;
  items_count: number;
  attempts: { started: number; graded: number };
  published_at: string | null;
  version: number;
  created_at: string;
};
export type ExamItem = {
  id: string;
  question_id: string;
  position: number;
  points: string;
  type: "MCQ_SINGLE" | "MCQ_MULTI" | "TRUE_FALSE" | "CODE";
  title: string;
  topic: string;
  difficulty: "EASY" | "MEDIUM" | "HARD";
  review_status: "DRAFT" | "PENDING" | "APPROVED" | "REJECTED";
  archived: boolean;
};
export type ExamDetail = ExamRow & {
  instructions: string | null;
  shuffle_questions: boolean;
  shuffle_options: boolean;
  max_score: string;
  rounding_step: string;
  multi_scoring: "PARTIAL" | "ALL_OR_NOTHING";
  reveal_answers: boolean;
  appeal_days: number;
  publish_hold: boolean;
  regrading: boolean;
  items: ExamItem[];
  created_by: string;
  updated_at: string;
};
/** Bản của sinh viên: trường giới hạn (không `items`, không DRAFT). */
export type StudentExam = {
  id: string;
  title: string;
  instructions: string | null;
  kind: ExamKind;
  opens_at: string | null;
  closes_at: string | null;
  duration_minutes: number | null;
  max_score: string;
  status: ExamStatus;
  my_attempt: { id: string; status: "IN_PROGRESS" | "GRADING" | "GRADED"; deadline_at: string; submitted_at: string | null } | null;
  my_score: string | null;
};
export type PreviewItem = {
  item_id: string;
  position: number;
  type: ExamItem["type"];
  points: string;
  stem: string;
  options: Array<{ id: string; body: string }>;
  code: null | { languages: string[]; time_limit_ms: number; memory_limit_mb: number; starter_code: Record<string, string>; samples: Array<{ name: string; input: string; expected: string }> };
};
export type Preview = { preview: true; exam: StudentExam; items: PreviewItem[] };
/** Một lỗi `details[]` của 422 (lên lịch trả TOÀN BỘ lỗi một lần). */
export type Problem = { field?: string; code?: string; message?: string };

export const ePath = (courseId: string) => `/courses/${courseId}/exams`;
export const eKey = (courseId: string, ...rest: Array<string | number>) => ["exam", courseId, ...rest] as const;

export const STATUS_LABEL: Record<ExamStatus, string> = { DRAFT: "Nháp", SCHEDULED: "Sắp tới", OPEN: "Đang mở", CLOSED: "Đã đóng", PUBLISHED: "Đã có điểm" };
export const STATUS_TONE: Record<ExamStatus, StatusTone> = { DRAFT: "neutral", SCHEDULED: "blue", OPEN: "green", CLOSED: "amber", PUBLISHED: "neutral" };
export const KIND_LABEL: Record<ExamKind, string> = { MCQ: "Trắc nghiệm", CODE: "Lập trình", MIXED: "Trắc nghiệm và lập trình" };

// ---- Giờ theo Asia/Ho_Chi_Minh (UTC+7, không có giờ mùa hè) ------------------------------------------------------------------
const ICT_MS = 7 * 3600 * 1000;
const DOW = ["Chủ nhật", "Thứ Hai", "Thứ Ba", "Thứ Tư", "Thứ Năm", "Thứ Sáu", "Thứ Bảy"];
const p2 = (n: number) => String(n).padStart(2, "0");
const ict = (iso: string) => new Date(new Date(iso).getTime() + ICT_MS);

/** "Thứ Ba, 09:00 01/12" */
export function fmtWhen(iso: string): string {
  const d = ict(iso);
  return `${DOW[d.getUTCDay()]}, ${p2(d.getUTCHours())}:${p2(d.getUTCMinutes())} ${p2(d.getUTCDate())}/${p2(d.getUTCMonth() + 1)}`;
}
/** "09:00 01/12" */
export function fmtClock(iso: string): string {
  const d = ict(iso);
  return `${p2(d.getUTCHours())}:${p2(d.getUTCMinutes())} ${p2(d.getUTCDate())}/${p2(d.getUTCMonth() + 1)}`;
}
/** Giá trị cho `<input type="datetime-local">`: "2026-12-01T09:00" theo giờ Việt Nam. */
export function toInput(iso: string | null): string {
  if (!iso) return "";
  const d = ict(iso);
  return `${d.getUTCFullYear()}-${p2(d.getUTCMonth() + 1)}-${p2(d.getUTCDate())}T${p2(d.getUTCHours())}:${p2(d.getUTCMinutes())}`;
}
/** Ngược lại: chuỗi `datetime-local` (giờ Việt Nam) → RFC 3339 UTC; rỗng → null. */
export function fromInput(v: string): string | null {
  if (!v) return null;
  const t = new Date(`${v}:00+07:00`);
  return Number.isNaN(t.getTime()) ? null : t.toISOString();
}

/** Khung giờ một dòng: "Thứ Ba, 09:00 01/12 · 45 phút · 12 câu". */
export function windowLine(e: Pick<ExamRow, "opens_at" | "duration_minutes" | "items_count">): string {
  const parts = [e.opens_at ? fmtWhen(e.opens_at) : "Chưa đặt giờ mở"];
  if (e.duration_minutes) parts.push(`${e.duration_minutes} phút`);
  parts.push(`${e.items_count} câu`);
  return parts.join(" · ");
}
