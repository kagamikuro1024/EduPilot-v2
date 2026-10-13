"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";
import { apiClient, useSSE, useSSEStatus, type CursorPage, type SSEEvent } from "@/shared/data";
import { timeAgo } from "@/shared/lib/timeAgo";
import type { NotificationItem } from "@/shared/shell/NotificationPopover";

type Row = { id: string; type: string; title: string; body: string | null; link: string | null; course_id: string | null; read_at: string | null; created_at: string };
type Page = CursorPage<Row> & { unread_count: number };

const KEY = ["notifications"] as const;

const TYPE_TEXT: Record<string, string> = {
  COURSE_ASSIGNED: "Phân công lớp",
  TICKET_NEW: "Hộp thư hỗ trợ",
  TICKET_STALE: "Hộp thư hỗ trợ",
  TICKET_ANSWERED: "Câu hỏi của bạn",
  AI_PENDING: "Threads",
  POST_REPORTED: "Threads",
  COURSE_TICKETS_STALLED: "Quản trị",
};

/** Chỉ nhận đường dẫn nội bộ (một dấu `/` đầu): thông báo là dữ liệu do máy chủ gửi, không tin để mở trang ngoài. */
function internal(link: string | null): string {
  return link && link.startsWith("/") && !link.startsWith("//") && !link.includes("\\") ? link : "/";
}

/** Khung SSE `notification` (SRS FEAT-escalation-notify 4.4.1) → hàng của danh sách; mục gộp (cùng id) cập nhật tại chỗ rồi nổi lên đầu. */
function applyFrame(p: Page | undefined, e: SSEEvent): Page | undefined {
  if (!p) return p;
  let f: { id: string; type: string; title: string; body: string | null; link: string | null; course_id: string | null; created_at: string; unread_count: number };
  try {
    f = JSON.parse(e.data);
  } catch {
    return p;
  }
  const row: Row = { id: f.id, type: f.type, title: f.title, body: f.body, link: f.link, course_id: f.course_id, read_at: null, created_at: f.created_at };
  return { ...p, unread_count: f.unread_count, items: [row, ...p.items.filter((n) => n.id !== f.id)].slice(0, 30) };
}

/**
 * Chuông thật (`GET /notifications` + sự kiện SSE `notification`): thông báo mới đẩy tức thì qua kết nối `/events` dùng chung của tab; SSE không nối được
 * (429, mất kết nối) thì rơi về làm mới 30 s như cũ, và luôn làm mới khi tab lấy lại focus. Chấm chưa đọc theo `unread_count`;
 * đánh dấu đã đọc lạc quan (hỏng ⇒ tải lại để về đúng trạng thái).
 */
export function useRealNotifications(enabled: boolean, live = enabled) {
  const qc = useQueryClient();
  const sse = useSSEStatus();
  useSSE("notification", (e) => qc.setQueryData<Page>(KEY, (p) => applyFrame(p, e)), { invalidateKeys: [KEY], enabled: live }); // `resync` → đọc bù từ bảng
  const q = useQuery({
    queryKey: KEY,
    enabled,
    staleTime: 0,
    refetchInterval: sse === "open" ? 120_000 : 30_000, // SSE mở: chỉ lưới an toàn thưa; còn lại 30 s
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

  return { items, unread: q.data?.unread_count ?? 0, failed: q.isError && !q.data, stale: q.isError && Boolean(q.data), markRead };
}
