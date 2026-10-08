"use client";

import { useEffect, useRef, useState } from "react";
import { formatRemaining, remainingMs, warnLevel, type ExamClock } from "@/shared/lib/examClock";
import s from "./ExamTimer.module.css";

/**
 * Đồng hồ làm bài: còn lại = deadline − (Date.now() + offset) (US-PE-05 AC3). Chữ số cố định bề rộng nên không nhảy layout; số giây `aria-live="off"`
 * (đọc mỗi giây sẽ làm phiền trình đọc màn hình). Mốc 5 phút / 1 phút báo qua `onWarn` để màn hình đặt MỘT dải chữ `aria-live="polite"`. Hết giờ chỉ báo `onExpire`
 * — máy chủ mới là bên quyết định bài đã hết hạn.
 */
export function ExamTimer({ deadlineMs, clock, onWarn, onExpire }: { deadlineMs: number; clock: ExamClock; onWarn?: (level: 1 | 5 | null) => void; onExpire?: () => void }) {
  const [ms, setMs] = useState(() => remainingMs(deadlineMs, clock, Date.now()));
  const cb = useRef({ onWarn, onExpire });
  useEffect(() => {
    cb.current = { onWarn, onExpire };
  });
  useEffect(() => {
    let level: 1 | 5 | null = null;
    let expired = false;
    const tick = () => {
      const left = remainingMs(deadlineMs, clock, Date.now());
      setMs(left);
      const w = warnLevel(left);
      if (w !== level) {
        level = w;
        cb.current.onWarn?.(w);
      }
      if (left <= 0 && !expired) {
        expired = true;
        cb.current.onExpire?.();
      }
    };
    tick();
    const t = window.setInterval(tick, 250);
    return () => window.clearInterval(t);
  }, [deadlineMs, clock]);
  const low = ms <= 60_000;
  return (
    <span className={[s.time, low ? s.low : ""].join(" ")} role="timer" aria-live="off" aria-label="Thời gian còn lại" data-part="exam-timer">
      {formatRemaining(ms)}
    </span>
  );
}
