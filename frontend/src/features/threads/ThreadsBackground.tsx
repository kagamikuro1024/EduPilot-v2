"use client";

import { useEffect, useRef } from "react";
import { pushNote } from "@/mock/notes";
import { KEYS, type InsightThread } from "@/mock/state";
import { THREADS_LIVE_KEY, THREADS_LIVE_SEED, TA_REPLY_AFTER_MS, type ThreadsLive } from "@/mock/threads";
import { simNowMs } from "@/shared/state/clock";
import { readSlice, writeSlice } from "@/shared/state/demo";
import { fireDueReplies } from "./threadsActions";

/**
 * Chạy ngầm ở mọi route: tới hạn thì phản hồi trễ của trợ giảng xuất hiện + chuông có thông báo (SRS 4.3.1 G).
 * Mốc hẹn lưu trong trạng thái phiên nên rời trang, đổi route hay tải lại trước 6 s vẫn tới hạn đúng lúc;
 * tải lại sau hạn thì phản hồi hiện ngay khi trang mở.
 */
export function ThreadsBackground() {
  const busy = useRef(false);
  useEffect(() => {
    const tick = () => {
      if (busy.current) return;
      const live = readSlice<ThreadsLive>(THREADS_LIVE_KEY, THREADS_LIVE_SEED);
      const now = simNowMs();
      if (!live.due.some((d) => !d.fired && now >= d.sentMs + TA_REPLY_AFTER_MS)) return;
      busy.current = true;
      const r = fireDueReplies(live, readSlice<InsightThread[]>(KEYS.insightThreads, []), now);
      writeSlice<ThreadsLive>(THREADS_LIVE_KEY, r.live);
      for (const n of r.notes) pushNote(n);
      busy.current = false;
    };
    tick();
    const t = window.setInterval(tick, 500);
    return () => window.clearInterval(t);
  }, []);

  return null;
}
