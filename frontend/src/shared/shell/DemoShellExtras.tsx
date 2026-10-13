"use client";

import { useEffect } from "react";
import { BT03_SEED } from "@/mock/assess";
import { COURSE_1 } from "@/mock/core";
import { reviewPending, ticketStats } from "@/mock/derive";
import { KEYS, type Bt03State, type Ticket } from "@/mock/state";
import { mergeTickets } from "@/mock/support";
import { ThreadsBackground } from "@/features/threads/ThreadsBackground";
import { useDemoSlice } from "@/shared/state/demo";

/**
 * Phần của khung chỉ có nghĩa ở phiên MÔ PHỎNG (không có lớp thật): nền Threads giả và badge mô phỏng (N9: tính từ dữ liệu, giảm ngay khi hành động xảy ra).
 * Tách ra khỏi gói chung để phiên thật không tải mã mô phỏng (ngân sách script Lighthouse, TL-1).
 */
export function DemoShellExtras({ scope, now, onBadges }: { scope: string[]; now: number; onBadges: (b: { inbox: number; grading: number }) => void }) {
  const [storedTickets] = useDemoSlice<Ticket[]>(KEYS.tickets, []);
  const [bt03] = useDemoSlice<Bt03State>(KEYS.bt03, BT03_SEED);
  const [approvedIds] = useDemoSlice<string[]>("grading.approved", []);
  const inbox = ticketStats(mergeTickets(storedTickets), scope, now).open;
  const grading = scope.includes(COURSE_1) ? reviewPending(bt03.status, approvedIds).length : 0;
  useEffect(() => onBadges({ inbox, grading }), [inbox, grading, onBadges]);
  return <ThreadsBackground />;
}
