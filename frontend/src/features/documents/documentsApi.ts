"use client";

import { useQuery } from "@tanstack/react-query";
import { apiClient, useCursorList } from "@/shared/data";

// Hợp đồng thật: backend-go/api/openapi.yaml (`/courses/{id}/documents*`, `/uploads/*`).
export type DocType = "LECTURE" | "COURSE_POLICY" | "EXAM_PAPER" | "ANSWER_KEY" | "OTHER";
export type DocStatus = "QUEUED" | "PROCESSING" | "READY" | "FAILED";
export type DocRow = {
  id: string; course_id: string; title: string; type: DocType; filename: string | null; status: DocStatus; error: string | null; page_count: number | null;
  visible_to_students: boolean; use_for_rag: boolean; category: string | null; week_no: number | null; version: number; updated_at: string; shared_from?: string;
};
export type Stats = { total: number; by_status: Record<string, number>; has_course_policy: boolean; chunks: number; embedded_chunks: number };
export type Chunk = { id: string; ord: number; page_no: number | null; heading: string | null; text: string; audience: string };

export const TYPE_LABEL: Record<DocType, string> = { LECTURE: "Bài giảng", COURSE_POLICY: "Quy chế môn học", EXAM_PAPER: "Đề tham khảo", ANSWER_KEY: "Đáp án", OTHER: "Tài liệu khác" };
export const STATUS_LABEL: Record<DocStatus, string> = { QUEUED: "Đang xử lý", PROCESSING: "Đang xử lý", READY: "Sẵn sàng", FAILED: "Lỗi" };
export const docsKey = (course: string) => ["documents", course] as const;

export const useDocList = (course: string, f: { type: string; status: string; q: string }) =>
  useCursorList<DocRow>([...docsKey(course), "list", f], `/courses/${course}/documents`, { limit: 50, query: { type: f.type || undefined, status: f.status || undefined, q: f.q || undefined } });

export const useDocStats = (course: string) =>
  useQuery({ queryKey: [...docsKey(course), "stats"], queryFn: async ({ signal }) => (await apiClient.get<Stats>(`/courses/${course}/documents/stats`, { signal })).data });

export const useChunks = (course: string, doc: string | null) =>
  useCursorList<Chunk>([...docsKey(course), "chunks", doc], `/courses/${course}/documents/${doc}/chunks`, { limit: 30 });
