import { useQuery } from "@tanstack/react-query";
import { apiClient, useCursorList } from "@/shared/data";

// Hợp đồng thật: backend-go/api/openapi.yaml (`/courses/{id}/tickets*`, `/me/tickets`).
export type TicketStatus = "OPEN" | "CLAIMED" | "ANSWERED" | "CLOSED";
export type Person = { id: string; full_name: string; role?: string };
export type CourseRef = { id: string; class_code: string };
export type TicketRow = {
  id: string; status: TicketStatus; kind: string; student: Person; question_preview: string; age_minutes: number;
  claimed_by: Person | null; sla_stage: number; course: CourseRef;
};
export type StaffTicket = {
  id: string; kind: string; status: TicketStatus; student: Person; claimed_by: Person | null; answered_by: Person | null; age_minutes: number;
  sla_stage: number; reopened_count: number; save_as_knowledge: boolean; course: CourseRef; version: number; created_at: string;
};
export type TicketMessage = { id: string; content: string; created_at: string; confidence?: string | null };
export type TranscriptItem = { id: string; role: string; author: { full_name: string; role: string } | "student"; content: string; created_at: string };
export type KnowledgeState = { saved: boolean; loaded: boolean; question?: string; answer?: string };
export type TicketDetail = { ticket: StaffTicket; question: TicketMessage | null; ai_answer: TicketMessage | null; transcript: TranscriptItem[]; knowledge?: KnowledgeState };

export type InboxFilter = "open" | "claimed" | "answered" | "all";
const API_STATUS: Record<InboxFilter, string> = { open: "OPEN", claimed: "CLAIMED", answered: "ANSWERED", all: "ALL" };

export const ticketsKey = ["tickets"] as const;

/** `courseId` là id lớp thật hoặc "all" (mọi lớp của tôi). */
export const useTicketList = (courseId: string, f: InboxFilter) =>
  useCursorList<TicketRow>([...ticketsKey, "list", courseId, f], courseId === "all" ? "/me/tickets" : `/courses/${courseId}/tickets`, { limit: 30, query: { status: API_STATUS[f] } });

export const useTicketDetail = (course: string | undefined, id: string | undefined) =>
  useQuery({
    queryKey: [...ticketsKey, "one", course, id],
    enabled: Boolean(course && id),
    queryFn: async ({ signal }) => (await apiClient.get<TicketDetail>(`/courses/${course}/tickets/${id}`, { signal })).data,
  });

/** 5 phút · 3 giờ · 2 ngày (từ số phút đã chờ). */
export function waitLabel(min: number): string {
  if (min < 60) return `${Math.max(0, min)} phút`;
  if (min < 48 * 60) return `${Math.floor(min / 60)} giờ`;
  return `${Math.floor(min / (24 * 60))} ngày`;
}
