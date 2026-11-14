import { apiClient } from "@/shared/data";

// Hợp đồng thật: backend-go/api/openapi.yaml (`previewFormsImport`, `importFormsResults`; thao tác 15–16 của SRS FEAT-practice 6).
export type FormsQuestionMap = { col: string; question_id?: string | null; accepted?: string[] | null };
export type FormsMapping = { header_row: number; email_col: string; code_col: string; time_col: string; questions: FormsQuestionMap[] };
export type FormsRequest = { upload_id: string; item_id?: string; answer_source?: "BANK" | "MANUAL"; mapping?: FormsMapping };
export type FormsScored = { student: { id: string; full_name: string; student_code: string }; correct: number; total: number; score: string; mismatch: number };
export type FormsUnmatched = { row: number; identifier: string; reason: "NOT_FOUND" | "MISMATCH"; correct: number; total: number; score: string };
export type FormsResult = { matched: number; unmatched: number; duplicates: number; written: number; replayed: boolean; batch_id?: string; items: FormsScored[]; unmatched_rows: FormsUnmatched[] };
export type FormsPreview = { columns: Array<{ col: string; header: string }>; rows: string[][]; rows_total: number; suggested_mapping: FormsMapping; result?: FormsResult };

export const previewForms = async (course: string, body: FormsRequest) => (await apiClient.post<FormsPreview>(`/courses/${course}/forms-import/preview`, body)).data;
export const importForms = async (course: string, body: FormsRequest, idempotencyKey: string) => (await apiClient.post<FormsResult>(`/courses/${course}/forms-import`, body, { idempotencyKey })).data;
