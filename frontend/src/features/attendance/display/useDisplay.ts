"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { ApiError, apiClient } from "@/shared/data";
import type { CheckinDisplay, CheckinWindow } from "../checkinApi";

export type DisplayState =
  | { kind: "loading" }
  | { kind: "error"; error: unknown }
  | { kind: "none" } // buổi chưa có cửa sổ nào
  | { kind: "closed"; count: number; headcount: number | null; round: number }
  | { kind: "live"; display: CheckinDisplay; window: CheckinWindow; stale: boolean };

const RETRY_SAME_STEP_MS = 500;
const AHEAD_MS = 150;

/**
 * Màn chiếu (DESIGN §14.8.1): hỏi `…/display` MỘT lần mỗi bước, hẹn lần sau đúng `step_expires_at` theo đồng hồ MÁY CHỦ (độ lệch đo từ `server_time`
 * của mỗi phản hồi; không tin đồng hồ máy chiếu). Mất mạng / 5xx: giữ mã cuối tới `step_expires_at + rotate_seconds` rồi báo `stale` (ô trống «Mất kết nối»);
 * thử lại sau 1, 2, 4 s, trần bằng chu kỳ. 409 WINDOW_CLOSED hoặc hết giờ → `closed`.
 */
export function useDisplay(course: string, session: string) {
  const [state, setState] = useState<DisplayState>({ kind: "loading" });
  const [attempt, setAttempt] = useState(0);
  const offset = useRef(0);
  const halt = useRef<() => void>(() => undefined);
  const serverNow = useCallback(() => Date.now() + offset.current, []);

  useEffect(() => {
    let alive = true;
    let timer: ReturnType<typeof setTimeout> | null = null;
    let tick: ReturnType<typeof setInterval> | null = null;
    let live: { display: CheckinDisplay; window: CheckinWindow } | null = null;
    let fails = 0;
    const stop = () => {
      alive = false;
      if (timer) clearTimeout(timer);
      if (tick) clearInterval(tick);
      timer = tick = null;
    };
    halt.current = stop;

    const closed = (win: CheckinWindow) => {
      const d = live?.display;
      stop();
      setState({ kind: "closed", count: d?.checked_in ?? win.checked_in, headcount: d?.headcount ?? win.headcount, round: d?.round ?? win.round });
    };

    // đồng hồ nhịp 1 s: đếm ngược, hết giờ → closed; mất mạng quá hạn mã cuối → stale (render lại, không gọi mạng)
    tick = setInterval(() => {
      if (!live) return;
      const n = Date.now() + offset.current;
      const d = live.display;
      if (n >= Date.parse(d.closes_at)) return closed(live.window);
      const stale = fails > 0 && n > Date.parse(d.step_expires_at) + d.rotate_seconds * 1000;
      setState({ kind: "live", display: d, window: live.window, stale });
    }, 1000);

    async function poll(wid: string, win: CheckinWindow, retriedSameStep: boolean): Promise<void> {
      const again = (ms: number, retried = false) => {
        if (alive) timer = setTimeout(() => void poll(wid, win, retried), ms);
      };
      try {
        const { data } = await apiClient.get<CheckinDisplay>(`/courses/${course}/sessions/${session}/checkin-windows/${wid}/display`);
        if (!alive) return;
        offset.current = Date.parse(data.server_time) - Date.now();
        fails = 0;
        const prev = live?.display;
        live = { display: data, window: win };
        setState({ kind: "live", display: data, window: win, stale: false });
        const same = prev?.step_expires_at === data.step_expires_at;
        const wait = Math.max(0, Date.parse(data.step_expires_at) - (Date.now() + offset.current)) + AHEAD_MS;
        again(same && !retriedSameStep ? RETRY_SAME_STEP_MS : wait, same); // cùng bước: thử lại MỘT lần sau 500 ms
      } catch (e) {
        if (!alive) return;
        if (e instanceof ApiError && e.code === "WINDOW_CLOSED") return closed(win);
        if (!live) return setState({ kind: "error", error: e });
        fails += 1;
        const d = live.display;
        const stale = Date.now() + offset.current > Date.parse(d.step_expires_at) + d.rotate_seconds * 1000;
        setState({ kind: "live", display: d, window: live.window, stale });
        again(Math.min(2 ** (fails - 1) * 1000, d.rotate_seconds * 1000));
      }
    }

    void (async () => {
      try {
        const { data } = await apiClient.get<{ items: CheckinWindow[] }>(`/courses/${course}/sessions/${session}/checkin-windows`);
        if (!alive) return;
        const open = data.items.find((w) => !w.closed_at && Date.parse(w.closes_at) > Date.now() - 60_000);
        if (open) return void (await poll(open.id, open, false));
        const last = data.items[data.items.length - 1];
        stop();
        setState(last ? { kind: "closed", count: last.checked_in, headcount: last.headcount, round: last.round } : { kind: "none" });
      } catch (e) {
        if (alive) setState({ kind: "error", error: e });
      }
    })();

    return stop;
  }, [course, session, attempt]);

  const retry = useCallback(() => {
    setState({ kind: "loading" });
    setAttempt((n) => n + 1);
  }, []);
  /** Đóng ngay ở máy (sau khi bấm «Đóng điểm danh» thành công): dừng hỏi mã, hiện `Đã đóng` với số cuối. */
  const markClosed = useCallback((count: number) => {
    halt.current();
    setState((s) => ({ kind: "closed", count, headcount: s.kind === "live" ? s.display.headcount : null, round: s.kind === "live" ? s.display.round : 1 }));
  }, []);

  return { state, serverNow, retry, markClosed };
}
