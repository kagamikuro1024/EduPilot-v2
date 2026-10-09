import { apiClient } from "@/shared/data";
import type { ExamClock } from "@/shared/lib/examClock";
import { examBase } from "./takeApi";

// Hợp đồng thật: backend-go/api/openapi.yaml (thao tác 32–37 của SRS FEAT-weekly-exam 6.2).
export type Verdict = "AC" | "WA" | "TLE" | "MLE" | "RE" | "OLE" | "CE" | "IE";
export type SampleResult = { name: string; verdict: Verdict; time_ms: number; memory_kb: number; input?: string; expected?: string; got?: string };
export type JudgeStatus = "QUEUED" | "RUNNING" | "DONE" | "ERROR" | "SUPERSEDED";
export type RunView = { id: string; status: JudgeStatus; language: string; created_at: string; compile_ok: boolean | null; compile_log?: string; samples: SampleResult[] };
export type SubmissionView = RunView & { is_final: boolean; source?: string };
export type DraftSaved = { rev: number; saved_at: string; server_time: string; deadline_at: string };
export type Lang = "c11" | "cpp17";

const item = (c: string, e: string, a: string, i: string) => `${examBase(c, e)}/attempts/${a}/code/${i}`;
const tabH = (tab: string) => ({ "X-Exam-Tab": tab });

export async function saveDraft(clock: ExamClock, c: string, e: string, a: string, i: string, tab: string, body: { language: Lang; source: string; base_rev: number }) {
  const sent = Date.now();
  const r = await apiClient.put<DraftSaved>(`${item(c, e, a, i)}/draft`, body, { headers: tabH(tab) });
  clock.observe(Date.parse(r.data.server_time), sent, Date.now());
  return r.data;
}

export const runCode = (c: string, e: string, a: string, i: string, tab: string, key: string, body: { language: Lang; source: string }) =>
  apiClient.post<{ run_id: string }>(`${item(c, e, a, i)}/run`, body, { headers: tabH(tab), idempotencyKey: key }).then((r) => r.data);

export const submitCode = (c: string, e: string, a: string, i: string, tab: string, key: string, body: { language: Lang; source: string }) =>
  apiClient.post<{ submission_id: string }>(`${item(c, e, a, i)}/submit`, body, { headers: tabH(tab), idempotencyKey: key }).then((r) => r.data);

export const getRun = (c: string, e: string, a: string, runId: string) => apiClient.get<RunView>(`${examBase(c, e)}/attempts/${a}/runs/${runId}`).then((r) => r.data);

export const getSubmission = (c: string, e: string, a: string, id: string) => apiClient.get<SubmissionView>(`${examBase(c, e)}/attempts/${a}/submissions/${id}`).then((r) => r.data);

export const listSubmissions = (c: string, e: string, a: string, i: string, cursor?: string) =>
  apiClient.get<{ items: SubmissionView[]; next_cursor: string | null }>(`${item(c, e, a, i)}/submissions`, { query: { limit: 20, cursor } }).then((r) => r.data);
