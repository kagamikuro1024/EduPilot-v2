"use client";

import { apiClient, newIdempotencyKey, type ApiResult } from "@/shared/data";
import { openChatStream } from "@/shared/data/chatStream"; // SSE qua POST có Authorization; không qua barrel (ngân sách JS mỗi route)
import type { ExamClock } from "@/shared/lib/examClock";
import { useSession } from "@/shared/session/session";

// Hợp đồng thật: backend-go/api/openapi.yaml (`/courses/{id}/practice…`, thao tác 5–14 của SRS FEAT-practice 6).
export type QType = "MCQ_SINGLE" | "MCQ_MULTI" | "TRUE_FALSE" | "SHORT" | "ESSAY";
export type Mode = "TOPIC" | "MOCK";
export type Status = "IN_PROGRESS" | "SUBMITTED" | "EXPIRED";
/** Câu trả lời đi lên máy chủ: MCQ `option_ids`, đúng–sai `value`, trả lời ngắn / tự luận `text`. */
export type Ans = { option_ids: string[] } | { value: boolean } | { text: string };

export type PQuestion = { type: QType; topic: string; stem: string; options: Array<{ id: string; body: string }> };
export type PFeedback = { correct: boolean | null; key: Record<string, unknown>; explanation: string | null; citations: Array<{ document_id?: string; page?: number; quote?: string }> };
export type PItem = {
  id: string;
  position: number;
  question: PQuestion;
  answer: { answer: Record<string, unknown>; version: number; locked: boolean } | null;
  feedback: PFeedback | null;
};
export type PResult = { correct: number; total: number; score: string; topics: Array<{ topic: string; correct: number; total: number }>; weakest_topic: string | null; expired: boolean };
export type PAttempt = {
  id: string;
  mode: Mode;
  topic: string | null;
  status: Status;
  started_at: string;
  deadline_at: string | null;
  server_now: string;
  item_count: number;
  current_index: number;
  notes: string[];
  version: number;
  items: PItem[];
  result: PResult | null;
};
export type POverview = {
  recommended: { topic: string; count: number } | null;
  topics: Array<{ topic: string; available: number }>;
  in_progress: Array<{ id: string; mode: Mode; topic: string | null; deadline_at: string | null; item_count: number; current_index: number }>;
};
export type PHistoryRow = { id: string; mode: Mode; topic: string | null; status: Status; started_at: string; submitted_at: string | null; item_count: number; correct: number | null };
export type PWeakness = { topics: Array<{ topic: string; answered: number; wrong: number; documents: Array<{ document_id: string; title: string }> }> };
export type EssayFeedback = { overall: string; criteria: Array<{ name: string; comment: string; evidence: string[] }>; injected: boolean };

export const pPath = (course: string) => `/courses/${course}/practice`;
export const pKey = (course: string, ...rest: Array<string | number>) => ["practice", course, ...rest] as const;
export const MIN_COUNT = 5;

/** Lớp của sinh viên đang xem: lớp thật đang chọn nếu là sinh viên của lớp đó, nếu không thì lớp sinh viên đầu tiên. */
export function useStudentCourse(): { state: "loading" } | { state: "none" } | { state: "ready"; id: string; ids: string[] } {
  const { realCourses, realCourseId } = useSession();
  if (!realCourses) return { state: "loading" };
  const mine = realCourses.filter((c) => c.role_in_course === "STUDENT").map((c) => c.id);
  const id = mine.find((c) => c === realCourseId) ?? mine[0];
  return id ? { state: "ready", id, ids: mine } : { state: "none" };
}

/** `X-Practice-Tab`: sinh mỗi lần tải trang, chỉ ở bộ nhớ của trang. */
export const newTabId = () => crypto.randomUUID();
const tabH = (tab?: string) => (tab ? { "X-Practice-Tab": tab } : undefined);

async function timed<T extends { server_now?: string }>(clock: ExamClock | null, run: () => Promise<ApiResult<T>>): Promise<ApiResult<T>> {
  const sent = Date.now();
  const res = await run();
  if (clock && res.data.server_now) clock.observe(Date.parse(res.data.server_now), sent, Date.now());
  return res;
}

export const getOverview = (c: string, signal?: AbortSignal) => apiClient.get<POverview>(`${pPath(c)}/overview`, { signal }).then((r) => r.data);
export const getWeakness = (c: string, signal?: AbortSignal) => apiClient.get<PWeakness>(`${pPath(c)}/weakness`, { signal }).then((r) => r.data);
export const startPractice = (c: string, body: { mode: Mode; topic?: string; count?: number; duration_minutes?: number }, tab: string, key: string) =>
  apiClient.post<PAttempt>(`${pPath(c)}/start`, body, { headers: tabH(tab), idempotencyKey: key });
export const getAttempt = (c: string, id: string, clock: ExamClock | null, tab?: string, signal?: AbortSignal) =>
  timed(clock, () => apiClient.get<PAttempt>(`${pPath(c)}/attempts/${id}`, { headers: tabH(tab), signal })).then((r) => r.data);
export const patchIndex = (c: string, id: string, tab: string, index: number) => apiClient.patch(`${pPath(c)}/attempts/${id}`, { current_index: index }, { headers: tabH(tab) });
export const submitAttempt = (c: string, id: string, key: string) => apiClient.post<PAttempt>(`${pPath(c)}/attempts/${id}/submit`, undefined, { idempotencyKey: key }).then((r) => r.data);

export type SaveBody = { answer: Ans; version?: number; client_ts?: string };
export type Saved = { version: number; locked: boolean; feedback: PFeedback | null };
export const putAnswer = (c: string, id: string, item: string, tab: string, body: SaveBody) =>
  apiClient.put<Saved>(`${pPath(c)}/attempts/${id}/answers/${item}`, body, { headers: tabH(tab) }).then((r) => r.data);

export const essayFeedback = (c: string, id: string, item: string, key: string) =>
  apiClient.post<EssayFeedback>(`${pPath(c)}/attempts/${id}/items/${item}/feedback`, undefined, { idempotencyKey: key }).then((r) => r.data);

/** Giải thích thêm (SSE): gọi `onText` cho từng đoạn chữ; trả khi luồng `done`; `event: error` → ném. */
export async function explain(c: string, id: string, item: string, signal: AbortSignal, onText: (t: string) => void): Promise<void> {
  let failed = false;
  await openChatStream({
    method: "POST",
    path: `${pPath(c)}/attempts/${id}/items/${item}/explain`,
    signal,
    onFrame: (f) => {
      if (f.event === "token" && typeof f.data.text === "string") onText(f.data.text);
      if (f.event === "error") failed = true;
    },
  });
  if (failed) throw new Error("EXPLAIN_FAILED");
}

export const answerKeyText = (it: PItem): string => {
  const k = it.feedback?.key ?? {};
  const ids = (k.option_ids as string[] | undefined) ?? [];
  if (ids.length > 0) return it.question.options.filter((o) => ids.includes(o.id)).map((o) => o.body).join("; ");
  if (typeof k.value === "boolean") return k.value ? "Đúng" : "Sai";
  if (Array.isArray(k.accepted)) return (k.accepted as string[]).join(" / ");
  return "";
};

export const givenText = (it: PItem): string => {
  const a = it.answer?.answer ?? {};
  const ids = (a.option_ids as string[] | undefined) ?? [];
  if (ids.length > 0) return it.question.options.filter((o) => ids.includes(o.id)).map((o) => o.body).join("; ");
  if (typeof a.value === "boolean") return a.value ? "Đúng" : "Sai";
  if (typeof a.text === "string" && a.text.trim()) return a.text.trim();
  return "Không trả lời";
};

export { newIdempotencyKey };
