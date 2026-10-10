"use client";

import { useQuery } from "@tanstack/react-query";
import { apiClient, useCursorList } from "@/shared/data";

// Hợp đồng thật: backend-go/api/openapi.yaml (`/courses/{id}/threads*`, `/posts/{id}/*`).
export type Author = { full_name: string; role: string; is_me: boolean };
export type ThreadRow = {
  id: string; title: string; preview: string; tags: string[]; week_no: number | null; author: Author;
  answer_state: "PENDING" | "VERIFIED" | "CORRECTED" | "REJECTED" | null; reply_count: number; last_activity_at: string; created_at: string; ai_state?: string;
};
export type PostCitation = { n: number; document_id: string; title: string; page_no: number | null; snippet: string };
export type ThreadPost = {
  id: string; kind: "AI" | "HUMAN"; author: Author | null; body: string; verification_state: string | null; citations: PostCitation[]; version: number; created_at: string;
  confidence?: string; ai_body?: string; rejected?: boolean;
};
export type ThreadDetail = {
  id: string; title: string; body: string; tags: string[]; week_no: number | null; author: Author; reply_count: number; created_at: string; last_activity_at: string;
  similar_of: string | null; ai_state?: string; ai_skip_reason?: string;
};
export type ThreadView = { thread: ThreadDetail; posts: ThreadPost[]; next_cursor: string | null };
export type Reason = { type: string; count: number };
export type Precheck = { allowed: boolean; reasons: Reason[]; redacted_text: string; redacted_title: string; personal_question: boolean };

export const REASON_LABEL: Record<string, string> = { MSSV: "MSSV", EMAIL: "Email", PHONE: "Số điện thoại", CCCD: "CCCD", NAME: "Họ tên", PERSONAL_QUESTION: "Câu hỏi riêng tư" };
export const describeReasons = (rs: Reason[]) => rs.map((r) => REASON_LABEL[r.type] ?? r.type).join(", ");
export const countReasons = (rs: Reason[]) => rs.filter((r) => r.type !== "PERSONAL_QUESTION").map((r) => `${r.count} ${(REASON_LABEL[r.type] ?? r.type).toLowerCase()}`).join(", ");

export const ANSWER_LABEL: Record<string, { tone: "amber" | "green" | "neutral"; text: string }> = {
  PENDING: { tone: "amber", text: "Chờ xác nhận" },
  VERIFIED: { tone: "green", text: "Đã được giảng viên xác nhận" },
  CORRECTED: { tone: "green", text: "Đã sửa bởi giảng viên" },
  REJECTED: { tone: "neutral", text: "Đã loại" },
};
export const SKIP_LABEL: Record<string, string> = { NO_CONTEXT: "Không có tài liệu liên quan", LOW_SCORE: "Tài liệu chưa đủ sát", LLM_UNAVAILABLE: "AI đang gián đoạn" };

export type ListFilter = { q: string; tag: string; week: string; state: string };
export const threadsKey = (courseId: string) => ["threads", courseId] as const;

export const useThreadList = (courseId: string, f: ListFilter) =>
  useCursorList<ThreadRow>([...threadsKey(courseId), "list", f], `/courses/${courseId}/threads`, { limit: 30, query: { q: f.q || undefined, tag: f.tag || undefined, week: f.week || undefined, state: f.state || undefined } });

export const threadKey = (courseId: string, id: string) => [...threadsKey(courseId), "one", id] as const;
export const useThread = (courseId: string, id: string) =>
  useQuery({ queryKey: threadKey(courseId, id), queryFn: async ({ signal }) => (await apiClient.get<ThreadView>(`/courses/${courseId}/threads/${id}`, { signal, query: { limit: 100 } })).data });

export const useSimilar = (courseId: string, id: string) =>
  useQuery({ queryKey: [...threadsKey(courseId), "similar", id], queryFn: async ({ signal }) => (await apiClient.get<{ items: { id: string; title: string; preview: string }[] }>(`/courses/${courseId}/threads/${id}/similar`, { signal })).data.items });

/** 0.720 → "0,72" */
export const confidenceLabel = (c: string) => Number(c).toFixed(2).replace(".", ",");
