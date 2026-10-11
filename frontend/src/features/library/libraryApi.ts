"use client";

import { useQuery } from "@tanstack/react-query";
import { apiClient, useCursorList } from "@/shared/data";

// Hợp đồng thật: backend-go/api/openapi.yaml (`/courses/{id}/library*`).
export type LibItem = { id: string; title: string; type: string; file_kind: "" | "PDF" | "DOCX" | "PPTX"; category: string | null; week_no: number | null; updated_at: string; snippet: string; can_ask_ai: boolean };
export type LibDetail = LibItem & { size_bytes: number | null; page_count: number | null; preview_url: string | null };

export const TYPE_LABEL: Record<string, string> = { LECTURE: "Bài giảng", COURSE_POLICY: "Quy chế môn học", EXAM_PAPER: "Đề tham khảo", OTHER: "Tài liệu khác" };
export const libKey = (course: string) => ["library", course] as const;

export const useLibrary = (course: string, f: { q: string; type: string; week: string }) =>
  useCursorList<LibItem>([...libKey(course), "list", f], `/courses/${course}/library`, { limit: 30, query: { q: f.q.trim().length >= 2 ? f.q.trim() : undefined, type: f.type || undefined, week: f.week || undefined } });

export const useLibDoc = (course: string, id: string) =>
  useQuery({ queryKey: [...libKey(course), "one", id], queryFn: async ({ signal }) => (await apiClient.get<LibDetail>(`/courses/${course}/library/${id}`, { signal })).data });

export async function downloadUrl(course: string, id: string): Promise<string> {
  return (await apiClient.get<{ url: string }>(`/courses/${course}/library/${id}/download`)).data.url;
}

/** "Hỏi AI về tài liệu": tạo phiên chat có `document_id` rồi mở `/chat?session=…`. */
export async function askAboutDoc(course: string, docId: string): Promise<string> {
  return (await apiClient.post<{ id: string }>("/chat/sessions", { course_id: course, document_id: docId })).data.id;
}
