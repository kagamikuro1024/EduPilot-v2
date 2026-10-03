// Đồng hồ giả lập (SRS 4.8): bắt đầu 29/10/2026 09:20, chạy theo thời gian thật; `Đặt lại dữ liệu demo` đưa về 09:20.
// Mốc bắt đầu `t0` (giờ thật lúc sim = NOW) nằm trong `ep_demo_state` nên tải lại trang không làm đồng hồ nhảy.
// Chưa lưu thì dùng giờ nạp trang; AppShell gọi `useEnsureClock()` để lưu lần đầu.
"use client";

import { useCallback, useEffect, useSyncExternalStore } from "react";
import { NOW } from "@/mock/core";
import { readSlice, writeSlice } from "./demo";

export const CLOCK_KEY = "clock";
type Clock = { t0: number };

let bootT0 = Date.now();

export function resetClock() {
  bootT0 = Date.now();
}

/** Thời điểm giả lập hiện tại (ms). Server: đúng 09:20. */
export function simNowMs(): number {
  if (typeof window === "undefined") return NOW.getTime();
  const t0 = readSlice<Clock | null>(CLOCK_KEY, null)?.t0 ?? bootT0;
  return NOW.getTime() + Math.max(0, Date.now() - t0);
}

/** Giờ giả lập (ms), cập nhật mỗi `period` ms và làm tròn xuống theo `period`. Lần render ở server = đúng 09:20. */
export function useSimNow(period = 30000): number {
  const subscribe = useCallback(
    (cb: () => void) => {
      const t = window.setInterval(cb, period);
      return () => window.clearInterval(t);
    },
    [period],
  );
  const snapshot = useCallback(() => Math.floor(simNowMs() / period) * period, [period]);
  return useSyncExternalStore(subscribe, snapshot, () => NOW.getTime());
}

/** Lưu mốc `t0` nếu chưa có (gọi một lần ở khung app; sau Đặt lại thì lưu mốc mới). */
export function useEnsureClock() {
  useEffect(() => {
    const ensure = () => {
      if (!readSlice<Clock | null>(CLOCK_KEY, null)) writeSlice<Clock>(CLOCK_KEY, { t0: bootT0 });
    };
    ensure();
    const t = window.setInterval(ensure, 2000);
    return () => window.clearInterval(t);
  }, []);
}
