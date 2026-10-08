import { apiClient, type ApiResult } from "@/shared/data";
import type { ExamKind, StudentExam } from "../examApi";
import type { ExamClock } from "@/shared/lib/examClock";
import type { Answer } from "@/shared/lib/saveQueue";

// Hợp đồng thật: backend-go/api/openapi.yaml (`/courses/{id}/exams/{eid}/attempts…`, thao tác 29–31, 39–41 của SRS FEAT-weekly-exam 6.2).
export type TakeItem = {
  item_id: string;
  position: number;
  type: "MCQ_SINGLE" | "MCQ_MULTI" | "TRUE_FALSE" | "CODE";
  points: string;
  stem: string;
  options: Array<{ id: string; body: string }>;
  code: unknown;
  answer: { option_ids?: string[]; value?: boolean } | null;
};
export type Attempt = { id: string; exam_id: string; status: "IN_PROGRESS" | "GRADING" | "GRADED"; started_at: string; deadline_at: string; server_time: string; writer: { is_you: boolean } };
export type Running = {
  attempt: Attempt;
  exam: { id: string; title: string; instructions: string | null; kind: ExamKind; duration_minutes: number; closes_at: string; multi_scoring: "PARTIAL" | "ALL_OR_NOTHING" };
  items: TakeItem[];
};
export type Submitted = {
  attempt: { id: string; status: "GRADING" | "GRADED"; submitted_at: string; submit_reason: "MANUAL" | "TIMEOUT" | "CLOSED" };
  exam: { id: string; title: string; closes_at: string | null; status: StudentExam["status"] };
};
export type NoAttempt = { attempt: null; exam: StudentExam };
export type Mine = Running | Submitted | NoAttempt;
export type SubmitSummary = { status: "GRADING" | "GRADED"; submitted_at: string; answered: number; total: number };

export const isNone = (m: Mine): m is NoAttempt => m.attempt === null;
export const isRunning = (m: Mine): m is Running => m.attempt !== null && "items" in m;
export const isSubmitted = (m: Mine): m is Submitted => m.attempt !== null && !("items" in m);

export const examBase = (courseId: string, examId: string) => `/courses/${courseId}/exams/${examId}`;

/** `X-Exam-Tab`: mã tab sinh MỖI LẦN TẢI TRANG và giữ trong bộ nhớ của trang (không ở sessionStorage — trình duyệt sao chép nó khi nhân bản tab; góp ý #12). */
export function newTabId(): string {
  return crypto.randomUUID();
}
const tabHeader = (tab: string) => ({ "X-Exam-Tab": tab });

/** Gọi API rồi cho đồng hồ đo lại độ lệch từ `server_time` của thân (nếu có): offset = server_time − (t_gửi + t_nhận)/2. */
async function timed<T>(clock: ExamClock, run: () => Promise<ApiResult<T>>, serverTime: (d: T) => string | undefined): Promise<ApiResult<T>> {
  const sent = Date.now();
  const res = await run();
  const recv = Date.now();
  const st = serverTime(res.data);
  if (st) clock.observe(Date.parse(st), sent, recv);
  return res;
}
const attemptTime = (d: unknown) => {
  const x = d as { server_time?: string; attempt?: { server_time?: string } | null };
  return x.server_time ?? x.attempt?.server_time;
};

export const getMine = (clock: ExamClock, course: string, exam: string, tab?: string, signal?: AbortSignal) =>
  timed<Mine>(clock, () => apiClient.get<Mine>(`${examBase(course, exam)}/attempts/mine`, { headers: tab ? tabHeader(tab) : undefined, signal }), attemptTime).then((r) => r.data);

export const startAttempt = (clock: ExamClock, course: string, exam: string, tab: string, key: string) =>
  timed<Running>(clock, () => apiClient.post<Running>(`${examBase(course, exam)}/attempts`, undefined, { headers: tabHeader(tab), idempotencyKey: key }), attemptTime);

export const saveAnswers = (clock: ExamClock, course: string, exam: string, attempt: string, tab: string, items: Array<{ item_id: string; answer: Answer }>) =>
  timed<{ saved_at: string; server_time: string; deadline_at: string }>(clock, () => apiClient.put(`${examBase(course, exam)}/attempts/${attempt}/answers`, { items }, { headers: tabHeader(tab) }), attemptTime).then((r) => r.data);

export const takeover = (clock: ExamClock, course: string, exam: string, attempt: string, tab: string, reload: boolean) =>
  timed<Attempt>(clock, () => apiClient.post<Attempt>(`${examBase(course, exam)}/attempts/${attempt}/takeover`, reload ? { reload: true } : undefined, { headers: tabHeader(tab) }), attemptTime).then((r) => r.data);

export const submitAttempt = (course: string, exam: string, attempt: string, tab: string, key: string) =>
  apiClient.post<SubmitSummary>(`${examBase(course, exam)}/attempts/${attempt}/submit`, undefined, { headers: tabHeader(tab), idempotencyKey: key }).then((r) => r.data);

export const getResult = (course: string, exam: string, attempt: string) =>
  apiClient.get<{ exam: { max_score: string }; score: string | null }>(`${examBase(course, exam)}/attempts/${attempt}/result`).then((r) => r.data);

/** Đợi `ms` ms (dùng khi gửi lại cùng một ý định lúc mạng chập chờn). */
export function sleep(ms: number): Promise<void> {
  const { promise, resolve } = Promise.withResolvers<void>();
  setTimeout(resolve, ms);
  return promise;
}
