"use client";

import { useEffect, useRef } from "react";
import { apiClient } from "@/shared/data";
import { examBase } from "./takeApi";

type Ev = { type: "TAB_HIDDEN" | "TAB_VISIBLE" | "PASTE" | "OFFLINE" | "ONLINE"; client_at: string; meta?: { duration_ms?: number; chars?: number; item_id?: string } };

const FLUSH_MS = 15_000;
const BATCH = 50; // máy chủ nhận 50 sự kiện đầu của một lô
const BUFFER_MAX = 200; // mất mạng lâu: giữ tối đa chừng này, bỏ bớt cũ nhất

/**
 * Ghi tín hiệu để giảng viên tham khảo (US-PE-07 AC3): rời / quay lại tab (kèm thời gian rời), dán (CHỈ độ dài và câu đang mở — không bao giờ nội dung), mất / có mạng.
 * Gửi gộp mỗi 15 s và khi rời trang (`pagehide`). Lỗi gửi KHÔNG ảnh hưởng gì tới việc làm bài: lô được giữ lại gửi sau.
 */
export function useIntegrityEvents({ courseId, examId, attemptId, tab, currentItemId, active }: { courseId: string; examId: string; attemptId: string; tab: string; currentItemId: string | undefined; active: boolean }) {
  const buf = useRef<Ev[]>([]);
  const hiddenAt = useRef<number | null>(null);
  const item = useRef(currentItemId);
  useEffect(() => {
    item.current = currentItemId;
  });

  useEffect(() => {
    if (!active) return;
    const now = () => new Date().toISOString();
    const push = (e: Ev) => {
      buf.current.push(e);
      if (buf.current.length > BUFFER_MAX) buf.current.splice(0, buf.current.length - BUFFER_MAX);
    };
    const flush = () => {
      if (buf.current.length === 0) return;
      const batch = buf.current.splice(0, BATCH);
      apiClient.post(`${examBase(courseId, examId)}/attempts/${attemptId}/events`, { events: batch }, { headers: { "X-Exam-Tab": tab } }).catch(() => {
        buf.current.unshift(...batch); // gửi lại ở nhịp sau
        if (buf.current.length > BUFFER_MAX) buf.current.splice(BUFFER_MAX);
      });
    };
    const closeHidden = () => {
      if (hiddenAt.current === null) return;
      const ms = Math.max(0, Date.now() - hiddenAt.current);
      hiddenAt.current = null;
      push({ type: "TAB_HIDDEN", client_at: now(), meta: { duration_ms: ms } });
    };
    const onVis = () => {
      if (document.visibilityState === "hidden") hiddenAt.current ??= Date.now();
      else {
        closeHidden();
        push({ type: "TAB_VISIBLE", client_at: now() });
      }
    };
    const onPaste = (e: ClipboardEvent) => {
      const chars = e.clipboardData?.getData("text")?.length ?? 0; // chỉ ĐỘ DÀI; nội dung không được đọc ra khỏi biến này
      push({ type: "PASTE", client_at: now(), meta: { chars, ...(item.current ? { item_id: item.current } : {}) } });
    };
    const onOffline = () => push({ type: "OFFLINE", client_at: now() });
    const onOnline = () => push({ type: "ONLINE", client_at: now() });
    const onHide = () => {
      closeHidden();
      flush();
    };
    document.addEventListener("visibilitychange", onVis);
    document.addEventListener("paste", onPaste, true);
    window.addEventListener("offline", onOffline);
    window.addEventListener("online", onOnline);
    window.addEventListener("pagehide", onHide);
    const t = window.setInterval(flush, FLUSH_MS);
    return () => {
      window.clearInterval(t);
      document.removeEventListener("visibilitychange", onVis);
      document.removeEventListener("paste", onPaste, true);
      window.removeEventListener("offline", onOffline);
      window.removeEventListener("online", onOnline);
      window.removeEventListener("pagehide", onHide);
      flush();
    };
  }, [active, courseId, examId, attemptId, tab]);
}
