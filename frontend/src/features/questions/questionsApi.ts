// Hợp đồng thật: backend-go/api/openapi.yaml (`/courses/{id}/questions…`, thao tác 1–16 của SRS FEAT-weekly-exam 6.2).
export type QType = "MCQ_SINGLE" | "MCQ_MULTI" | "TRUE_FALSE" | "CODE";
export type Review = "DRAFT" | "PENDING" | "APPROVED" | "REJECTED";
export type Difficulty = "EASY" | "MEDIUM" | "HARD";
export type Origin = "MANUAL" | "AI_DRAFT" | "EXTRACTED" | "GENERATED";

export type QuestionRow = {
  id: string;
  type: QType;
  title: string;
  topic: string;
  difficulty: Difficulty;
  review_status: Review;
  origin: Origin;
  used_in_exams: number;
  version: number;
  archived_at: string | null;
  updated_at: string;
};
export type QOption = { id: string; position: number; body: string; pinned_last: boolean };
export type CodeDetail = {
  languages: Array<"c11" | "cpp17">;
  time_limit_ms: number;
  memory_limit_mb: number;
  output_limit_kb: number;
  checker: "EXACT" | "TOKENS" | "FLOAT_EPS";
  float_eps: string | null;
  starter_code: Record<string, string>;
  reference: { language: string; source: string } | null;
  reference_verified_version: number | null;
  reference_verified_at: string | null;
  tests_version: number;
  tests: { total: number; samples: number; hidden: number; total_weight: number };
};
export type QuestionDetail = Omit<QuestionRow, "used_in_exams"> & {
  stem: string;
  explanation: string | null;
  options: QOption[];
  answer_key: { option_ids?: string[]; value?: boolean } | null;
  code?: CodeDetail;
  used_in_exams: Array<{ id: string; title: string; status: string }>;
  details?: { warnings: Array<{ field: string; code: string; message: string }> };
};
export type Testcase = {
  id: string;
  position: number;
  name: string;
  is_sample: boolean;
  weight: number;
  input: string;
  input_truncated: boolean;
  expected: string;
  expected_truncated: boolean;
  input_bytes: number;
  expected_bytes: number;
  source: "MANUAL" | "IMPORT" | "AI_DRAFT";
  approved: boolean;
};
export type VerifyResult = { ok: boolean; compile_ok: boolean; compile_log?: string; tests_version: number; per_test: Array<{ test_id: string; name: string; verdict: string; time_ms: number }>; message?: string };

export const qPath = (courseId: string) => `/courses/${courseId}/questions`;
export const qKey = (courseId: string, ...rest: Array<string | number | undefined>) => ["questions", courseId, ...rest] as const;

export const TYPE_LABEL: Record<QType, string> = { MCQ_SINGLE: "Một đáp án", MCQ_MULTI: "Nhiều đáp án", TRUE_FALSE: "Đúng–sai", CODE: "Lập trình" };
export const DIFF_LABEL: Record<Difficulty, string> = { EASY: "Dễ", MEDIUM: "Vừa", HARD: "Khó" };
export const REVIEW_LABEL: Record<Review, string> = { DRAFT: "Nháp", PENDING: "Chờ duyệt", APPROVED: "Đã duyệt", REJECTED: "Đã loại" };
export const REVIEW_TONE: Record<Review, "neutral" | "amber" | "green" | "red"> = { DRAFT: "neutral", PENDING: "amber", APPROVED: "green", REJECTED: "red" };
export const ORIGIN_LABEL: Record<Origin, string> = { MANUAL: "Soạn tay", AI_DRAFT: "AI", EXTRACTED: "Trích từ đề", GENERATED: "AI" };
export const VERDICT_LABEL: Record<string, string> = { AC: "Đúng", WA: "Sai kết quả", TLE: "Quá thời gian", MLE: "Quá bộ nhớ", RE: "Lỗi khi chạy", OLE: "In ra quá nhiều", CE: "Lỗi biên dịch", IE: "Hệ thống chưa chạy được" };
