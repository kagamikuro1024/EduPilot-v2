"use client";

import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/shared/data";

// Hợp đồng thật: backend-go/api/openapi.yaml (`/chat/*`, `/me/exam-lock`).
export type ChatSession = { id: string; title: string | null; last_message_at: string; document_id: string | null };
export type ChatCitation = { n: number; document_id: string; title: string; page_no: number | null; snippet: string };
export type ChatBlock = { kind: string; data: unknown };
/** Ticket "Nhờ giảng viên" mà sinh viên được thấy: đúng bốn trường (không độ tin cậy, người nhận, nguồn gốc). */
export type StudentTicket = { id: string; status: "OPEN" | "CLAIMED" | "ANSWERED" | "CLOSED"; created_at: string; answered_at?: string };
export type ChatMessage = {
  id: string;
  role: "USER" | "ASSISTANT";
  content: string;
  streaming: boolean;
  stream_status: "STREAMING" | "DONE" | "FAILED" | "CANCELLED";
  attempt: number;
  citations: ChatCitation[];
  blocks: ChatBlock[];
  low_confidence: boolean;
  degraded: boolean;
  masked_count: number;
  feedback: "HELPFUL" | "NOT_HELPFUL" | null;
  error_code: string | null;
  ticket?: StudentTicket | null;
  can_escalate?: boolean;
  created_at: string;
};

export const sessionsKey = (courseId: string) => ["chat", "sessions", courseId] as const;
export const messagesKey = (sid: string) => ["chat", "messages", sid] as const;
export const LOCK_KEY = ["me", "exam-lock"] as const;

export const useChatSessions = (courseId: string | null) =>
  useQuery({
    queryKey: sessionsKey(courseId ?? ""),
    enabled: Boolean(courseId),
    queryFn: async ({ signal }) => (await apiClient.get<{ items: ChatSession[] }>("/chat/sessions", { query: { course_id: courseId, limit: 50 }, signal })).data.items,
  });

/** Lịch sử cũ → mới (máy chủ trả mới nhất trước) + số ticket đang mở của sinh viên ở lớp (nút nhờ giảng viên khoá khi đạt trần). */
export const useChatMessages = (sid: string | null) =>
  useQuery({
    queryKey: messagesKey(sid ?? ""),
    enabled: Boolean(sid),
    queryFn: async ({ signal }) => {
      const d = (await apiClient.get<{ items: ChatMessage[]; open_tickets?: number }>(`/chat/sessions/${sid}/messages`, { query: { limit: 100 }, signal })).data;
      return { items: d.items.slice().reverse(), openTickets: d.open_tickets ?? 0 };
    },
  });

/** Khoá chat trong giờ thi: đọc khi mở, mỗi 30 s và khi gặp 409 (người gọi `refetch`). */
export const useExamLock = () =>
  useQuery({
    queryKey: LOCK_KEY,
    refetchInterval: 30_000,
    queryFn: async ({ signal }) => (await apiClient.get<{ locked: boolean; until?: string }>("/me/exam-lock", { signal })).data,
  });
