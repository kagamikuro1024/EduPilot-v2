"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { apiClient, useCursorList, useIdempotentMutation } from "@/shared/data";
import { calKey } from "./calendarApi";

// Hợp đồng thật: backend-go/api/openapi.yaml (`/courses/{id}/sessions`, `…/sessions/generate`, `…/sessions/{sid}`).
export type SessionRow = {
  id: string; session_no: number; starts_at: string; ends_at: string; room: string | null; topic: string | null; version: number;
  /** chỉ nhân viên thấy */
  attendance?: { done_at: string | null; has_records: boolean };
};
export type GenerateInput = { weekdays: number[]; start_time: string; end_time: string; room: string | null; from: string; to: string; exclude_dates: string[] };
export type PlannedSession = { session_no: number; starts_at: string; ends_at: string };
export type Preview = { would_create: PlannedSession[]; skipped: { date: string; reason: "EXCLUDED" | "DUPLICATE" }[] };
export type Generated = { created: number; first_session_no: number | null; last_session_no: number | null; skipped: number };
export type SessionEdit = { starts_at: string; ends_at: string; room: string | null; topic: string | null; version: number };

export const sessionsKey = (course: string) => ["sessions", course] as const;

/** Tải hết các trang (lịch một học kỳ ≤ vài chục buổi). */
export function useSessions(course: string) {
  const list = useCursorList<SessionRow>(sessionsKey(course), `/courses/${course}/sessions`, { limit: 100 });
  const { hasNextPage, isFetchingNextPage, fetchNextPage } = list;
  useEffect(() => {
    if (hasNextPage && !isFetchingNextPage) void fetchNextPage();
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);
  return list;
}

/** Buổi đổi ⇒ danh sách buổi, lịch gộp và "Hôm nay" (bước thiết lập, buổi chưa điểm danh) đều làm mới. */
function useRefresh(course: string) {
  const qc = useQueryClient();
  return () => Promise.all([
    qc.invalidateQueries({ queryKey: sessionsKey(course) }),
    qc.invalidateQueries({ queryKey: calKey(course) }),
    qc.invalidateQueries({ queryKey: ["today"] }),
  ]);
}

/** Xem trước: chỉ đọc, server không ghi gì và không đòi khoá idempotency. */
export function usePreviewSessions(course: string) {
  return useMutation({
    mutationFn: async (b: GenerateInput) => (await apiClient.post<Preview>(`/courses/${course}/sessions/generate`, b, { query: { dry_run: true } })).data,
  });
}

/** Tạo thật: `mutate` = ý định mới (khoá mới), `retry` sau lỗi mạng dùng LẠI khoá cũ nên không tạo đôi. */
export function useGenerateSessions(course: string) {
  const refresh = useRefresh(course);
  return useIdempotentMutation(async (b: GenerateInput, key: string) => {
    const r = await apiClient.post<Generated>(`/courses/${course}/sessions/generate`, b, { idempotencyKey: key });
    await refresh();
    return r;
  });
}

export function useSessionMutations(course: string) {
  const refresh = useRefresh(course);
  return {
    update: useMutation({
      mutationFn: async ({ id, ...b }: SessionEdit & { id: string }) => (await apiClient.put<{ session: SessionRow }>(`/courses/${course}/sessions/${id}`, b)).data.session,
      onSuccess: refresh,
      // 409 (sai version / trùng giờ): lấy lại danh sách để người dùng thấy bản hiện hành
      onError: () => { void refresh(); },
    }),
    remove: useMutation({
      mutationFn: (id: string) => apiClient.delete(`/courses/${course}/sessions/${id}`),
      onSettled: refresh, // lỗi (đã có điểm danh) thì dòng trở lại đúng như server có
    }),
  };
}
