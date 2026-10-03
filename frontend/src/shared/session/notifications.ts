"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";
import { apiClient, type CursorPage } from "@/shared/data";
import { timeAgo } from "@/shared/lib/timeAgo";
import type { NotificationItem } from "@/shared/shell/NotificationPopover";

type Row = { id: string; type: string; title: string; body: string | null; link: string | null; course_id: string | null; read_at: string | null; created_at: string };
type Page = CursorPage<Row> & { unread_count: number };

const KEY = ["notifications"] as const;

const TYPE_TEXT: Record<string, string> = { COURSE_ASSIGNED: "Phân công lớp" };

/** Chỉ nhận đường dẫn nội bộ (một dấu `/` đầu): thông báo là dữ liệu do máy chủ gửi, không tin để mở trang ngoài. */
function internal(link: string | null): string {
  return link && link.startsWith("/") && !link.startsWith("//") && !link.includes("\\") ? link : "/";
}

/**
 * Chuông thật (`GET /notifications`): làm mới mỗi 30 s và khi tab lấy lại focus. Chấm chưa đọc theo `unread_count` của máy chủ;
 * đánh dấu đã đọc lạc quan (hỏng ⇒ tải lại để về đúng trạng thái).
 */
export function useRealNotifications(enabled: boolean) {
  const qc = useQueryClient();
  const q = useQuery({
    queryKey: KEY,
    enabled,
    staleTime: 0,
    refetchInterval: 30_000,
    refetchOnWindowFocus: true,
    queryFn: async ({ signal }): Promise<Page> => (await apiClient.get<Page>("/notifications", { signal, query: { limit: 30 } })).data,
  });

  const items = useMemo<NotificationItem[]>(
    () =>
      (q.data?.items ?? []).map((n) => ({ id: n.id, title: n.title, context: TYPE_TEXT[n.type] ?? "Thông báo", when: timeAgo(n.created_at), href: internal(n.link), read: n.read_at !== null })),
    [q.data],
  );

  const markRead = useCallback(
    (id: string) => {
      qc.setQueryData<Page>(KEY, (p) => {
        const row = p?.items.find((n) => n.id === id);
        if (!p || !row || row.read_at) return p;
        return { ...p, unread_count: Math.max(0, p.unread_count - 1), items: p.items.map((n) => (n.id === id ? { ...n, read_at: new Date().toISOString() } : n)) };
      });
      apiClient.post(`/notifications/${id}/read`).then(
        () => qc.invalidateQueries({ queryKey: KEY }),
        () => qc.invalidateQueries({ queryKey: KEY }),
      );
    },
    [qc],
  );

  return { items, unread: q.data?.unread_count ?? 0, failed: q.isError && !q.data, markRead };
}
