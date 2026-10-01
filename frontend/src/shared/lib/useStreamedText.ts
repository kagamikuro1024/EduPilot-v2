"use client";

import { useCallback, useEffect, useRef, useState, useSyncExternalStore } from "react";
import { simNowMs, useSimNow } from "@/shared/state/clock";

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

export type RevealPhase = "typing" | "streaming" | "sourcing" | "done" | "stopped";

const REDUCE_QUERY = "(prefers-reduced-motion: reduce)";
function subscribeReduce(cb: () => void) {
  const m = window.matchMedia(REDUCE_QUERY);
  m.addEventListener("change", cb);
  return () => m.removeEventListener("change", cb);
}

/**
 * Tiến trình một câu trả lời AI MỚI (SRS 4.3.1 E): "đang soạn" → chữ chảy → hiện nguồn → xong.
 * Tính từ mốc `startedAt` (giờ giả lập) nên rời trang / tải lại giữa chừng vẫn tiếp tục đúng chỗ.
 * `startedAt` undefined (bài seed) hoặc `prefers-reduced-motion` → hiện ngay toàn văn ở bước cuối.
 * `stoppedAt` là số ký tự đã hiện khi người dùng bấm Dừng.
 */
export function useTimedReveal({
  body,
  startedAt,
  typingMs,
  streamMs,
  sourcesMs,
  stoppedAt,
}: {
  body: string;
  startedAt: number | undefined;
  typingMs: number;
  streamMs: number;
  sourcesMs: number;
  stoppedAt?: number;
}): { phase: RevealPhase; text: string; chars: number } {
  const reduce = useSyncExternalStore(
    subscribeReduce,
    () => window.matchMedia(REDUCE_QUERY).matches,
    () => false,
  );
  const total = typingMs + streamMs + sourcesMs;
  // Chỉ chạy đồng hồ nhanh khi bài còn đang diễn ra; xong rồi thì nghỉ.
  const running = startedAt !== undefined && !reduce && stoppedAt === undefined && simNowMs() - startedAt < total;
  useSimNow(running ? 80 : 60000); // chỉ để kích hoạt vẽ lại; thời điểm chính xác đọc trực tiếp bên dưới

  if (startedAt === undefined || reduce) return { phase: "done", text: body, chars: body.length };
  if (stoppedAt !== undefined) return { phase: "stopped", text: body.slice(0, stoppedAt), chars: stoppedAt };

  const e = simNowMs() - startedAt;
  if (e < typingMs) return { phase: "typing", text: "", chars: 0 };
  if (e >= typingMs + streamMs) return { phase: e >= total ? "done" : "sourcing", text: body, chars: body.length };

  // nhảy theo ranh giới từ để chữ tiếng Việt không vỡ dấu giữa chừng
  let n = Math.floor((body.length * (e - typingMs)) / streamMs);
  while (n < body.length && body[n] !== " " && body[n] !== "\n") n++;
  return { phase: "streaming", text: body.slice(0, n), chars: n };
}
