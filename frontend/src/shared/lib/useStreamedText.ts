"use client";

import { useCallback, useEffect, useRef, useState } from "react";

/**
 * Giả lập chữ AI chảy từng đoạn cho bản prototype (thay bằng SSE thật ở P3).
 * Trả chữ đã hiện, cờ xong, và hàm dừng. Chế độ giảm chuyển động: hiện ngay toàn bộ.
 */
export function useStreamedText(full: string | null, { cps = 90, startDelay = 450 }: { cps?: number; startDelay?: number } = {}) {
  const [shown, setShown] = useState("");
  const [done, setDone] = useState(false);
  const timer = useRef<number | undefined>(undefined);

  const stop = useCallback(() => {
    window.clearInterval(timer.current);
    setDone(true);
  }, []);

  useEffect(() => {
    window.clearInterval(timer.current);
    if (full === null) return;
    const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    let i = 0;
    const tickMs = 40;
    const step = Math.max(1, Math.round((cps * tickMs) / 1000));
    const begin = window.setTimeout(() => {
      if (reduce) {
        setShown(full);
        setDone(true);
        return;
      }
      timer.current = window.setInterval(() => {
        // nhảy theo ranh giới từ để chữ tiếng Việt không vỡ dấu giữa chừng
        let next = Math.min(full.length, i + step);
        while (next < full.length && full[next] !== " " && full[next] !== "\n") next++;
        i = next;
        setShown(full.slice(0, i));
        if (i >= full.length) {
          window.clearInterval(timer.current);
          setDone(true);
        }
      }, tickMs);
    }, startDelay);
    return () => {
      window.clearTimeout(begin);
      window.clearInterval(timer.current);
      setShown("");
      setDone(false);
    };
  }, [full, cps, startDelay]);

  return { text: shown, done, stop };
}
