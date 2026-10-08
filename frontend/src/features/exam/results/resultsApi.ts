import type { StatusTone } from "@/shared/ui";

// Hợp đồng thật: backend-go/api/openapi.yaml (`/courses/{id}/exams/{eid}/results…`, `/stats`, `/appeals…`, `/publish-hold`, `/regrade`, `/items/{id}/override`; thao tác 26, 42, 44–50, 55, 56).
export type RowStatus = "NOT_STARTED" | "ABSENT" | "IN_PROGRESS" | "GRADING" | "GRADED";
export type Student = { id: string; full_name: string; student_code: string };
export type Row = {
  attempt_id: string | null;
  student: Student;
  status: RowStatus;
  auto_score: string | null;
  score: string | null;
  adjusted: boolean;
  submitted_at: string | null;
  submit_reason: string | null;
  /** chỉ có với Giảng viên */
  flags?: { similarity: number; tab_hidden: number; paste: number };
};
export type Progress = { not_started: number; in_progress: number; grading: number; graded: number; absent: number };
export type ResultsPage = { progress: Progress; items: Row[]; next_cursor: string | null };

export type ItemOption = { id: string; body: string };
export type AnswerShape = { option_ids?: string[]; value?: boolean } | null;
export type ResultItem = {
  item_id: string;
  position: number;
  type: "MCQ_SINGLE" | "MCQ_MULTI" | "TRUE_FALSE" | "CODE" | string;
  stem: string;
  options: ItemOption[];
  earned: string;
  max: string;
  correct: boolean | null;
  mine: AnswerShape;
  answer: AnswerShape;
  explanation: string | null;
  overridden: boolean;
  samples: Array<{ name: string; verdict: string; time_ms: number; memory_kb: number; input: string; expected: string }>;
  hidden: { passed: number; total: number } | null;
  final_submission: { id: string; language: string; source: string; created_at: string; compile_ok: boolean } | null;
  compile_log: string | null;
};
export type StudentResult = {
  exam: { id: string; title: string; max_score: string; published_at: string | null; reveal_answers: boolean; appeal_days: number; appeal_open_until: string | null };
  score: string | null;
  score_adjusted: boolean;
  appeal: { status: "OPEN" | "UPHELD" | "ADJUSTED" | null; response: string | null };
  items: ResultItem[];
};

export type StaffSubmission = {
  id: string;
  item_id: string;
  status: string;
  verdict: string | null;
  language: string;
  source: string;
  created_at: string;
  compile_ok: boolean | null;
  compile_log: string | null;
  tests: Array<{ position: number; is_sample: boolean; verdict: string; time_ms: number; memory_kb: number }>;
};
export type Appeal = {
  id: string;
  attempt_id: string;
  status: "OPEN" | "UPHELD" | "ADJUSTED";
  reason: string;
  response: string | null;
  created_at: string;
  responded_at: string | null;
  score_before: string | null;
  score_after: string | null;
  version: number;
  student?: { full_name: string; student_code: string };
};
export type Detail = {
  attempt_id: string;
  student: Student;
  status: RowStatus;
  version: number;
  auto_score: string | null;
  score: string | null;
  adjust: { score: string; reason: string; at: string } | null;
  submitted_at: string | null;
  submit_reason: string | null;
  items: ResultItem[];
  submissions: StaffSubmission[];
  integrity?: { tab_hidden_count: number; tab_hidden_ms: number; paste_count: number; paste_chars: number; offline_count: number; takeover_count: number; chat_blocked_count: number };
  appeal: Appeal | null;
};
export type Stats = {
  distribution: Array<{ from: string; to: string; count: number }>;
  mean: string;
  median: string;
  hardest: Array<{ item_id: string; title: string; correct_rate: string }>;
  code: Array<{ item_id: string; title: string; mean_ratio: string; ce_rate: string }>;
};

export const resPath = (course: string, exam: string) => `/courses/${course}/exams/${exam}`;

/** Điểm hiển thị dấu phẩy ("7.75" → "7,75"); rỗng → "—". Chỉ để đọc: điểm cuối do máy chủ làm tròn một lần. */
export const vnum = (s: string | null | undefined): string => (s == null || s === "" ? "—" : s.replace(".", ","));
/** "0.67" → "67 %". */
export const pct = (s: string): string => `${Math.round(Number(s) * 100)} %`;

export const STATUS_VI: Record<RowStatus, string> = { NOT_STARTED: "Chưa bắt đầu", ABSENT: "Vắng", IN_PROGRESS: "Đang làm", GRADING: "Đang chấm", GRADED: "Đã chấm" };
export const STATUS_TONE: Record<RowStatus, StatusTone> = { NOT_STARTED: "neutral", ABSENT: "neutral", IN_PROGRESS: "blue", GRADING: "amber", GRADED: "green" };
export const REASON_VI: Record<string, string> = { MANUAL: "Nộp tay", TIMEOUT: "Hết giờ", CLOSED: "Bài đóng" };
export const VERDICT_VI: Record<string, string> = { AC: "Đúng", WA: "Sai kết quả", TLE: "Quá thời gian", MLE: "Quá bộ nhớ", RE: "Lỗi khi chạy", CE: "Lỗi biên dịch", IE: "Lỗi hệ thống", OLE: "Quá nhiều đầu ra" };

/** Tải một tệp `Blob` về máy (CSV). */
export function saveBlob(b: Blob, name: string): void {
  const url = URL.createObjectURL(b);
  const a = document.createElement("a");
  a.href = url;
  a.download = name;
  document.body.appendChild(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}
