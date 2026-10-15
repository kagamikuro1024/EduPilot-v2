"use client";

import { useEffect } from "react";
import { ApiError } from "./ApiError";
import { apiClient } from "./apiClient";

/** Khu vực đo theo đường dẫn (SRS crm-360 4.6.3): chỉ năm khu vực; mọi route khác — kể cả `/exams/**` và `/me` — không gửi. */
export function areaOf(pathname: string): "CHAT" | "THREADS" | "PRACTICE" | "LIBRARY" | "CALENDAR" | null {
  const seg = pathname.split("/")[1] ?? "";
  switch (seg) {
    case "chat": return "CHAT";
    case "threads": return "THREADS";
    case "practice": return "PRACTICE";
    case "library": return "LIBRARY";
    case "calendar": return pathname === "/calendar" ? "CALENDAR" : null;
    default: return null;
  }
}

const PERIOD_MS = 30_000; // HEARTBEAT_SECONDS: chu kỳ CỐ ĐỊNH (chỉ pha ngẫu nhiên lúc bắt đầu) để mỗi nhịp rơi đúng một khung 30 s của máy chủ
const IDLE_MS = 120_000; // HEARTBEAT_IDLE_SECONDS
const RESUME_MS = 500; // tab hiện lại / có thao tác sau khi nghỉ → gửi tiếp trong ≤ 1 s
const MOVE_THROTTLE_MS = 1_000;
const INPUTS = ["pointerdown", "keydown", "scroll", "touchstart"] as const;
const MIN_GAP_MS = 25_000; // hai nhịp thường cách nhau ≥ 25 s dù có bao nhiêu bộ hẹn giờ (nhịp sau khi nghỉ / tab hiện lại được miễn)

// Trạng thái CẢ TAB, không theo từng lần chạy effect: effect chạy lại (đổi khu vực, cờ lớp tải lại) không được đặt lại mốc thao tác (kéo nhịp quá 120 s idle)
// và không được sinh thêm lịch nhịp thứ hai (QC N1: nhịp 15 s, nhịp sau mốc idle).
let lastInput = 0;
let lastBeatAt = 0;

/**
 * Nhịp thời gian học (US-P5-03): mỗi 30 s khi tab đang hiện VÀ có thao tác trong 120 s gần nhất; ẩn tab → ngừng ngay; mạng đứt → bỏ nhịp (không xếp hàng);
 * `403` / `422` → ngừng im lặng; đổi `area` / gỡ → dọn mọi bộ hẹn giờ. Hook KHÔNG biết cấu hình lớp: máy chủ quyết định đo hay không (`204` luôn).
 * Không ghi nội dung, đường dẫn, tiêu đề hay tên người: chỉ `course_id`, `area`, `tab_id` (UUID mỗi tab).
 */
export function useActivityHeartbeat({ courseId, area, enabled }: { courseId: string | null; area: string | null; enabled: boolean }) {
  useEffect(() => {
    if (!enabled || !courseId || !area) return;
    const tabId = crypto.randomUUID();
    const ctl = new AbortController();
    if (lastInput === 0) lastInput = Date.now(); // vào trang là thao tác đầu tiên của tab
    let lastMove = 0;
    let stopped = false;
    let interval: ReturnType<typeof setInterval> | undefined;
    let resume: ReturnType<typeof setTimeout> | undefined;

    const active = () => document.visibilityState === "visible" && Date.now() - lastInput <= IDLE_MS;
    const beat = (resumed = false) => {
      if (stopped || !active()) return;
      if (!resumed && Date.now() - lastBeatAt < MIN_GAP_MS) return;
      lastBeatAt = Date.now();
      apiClient.post("/activity/heartbeat", { course_id: courseId, area, tab_id: tabId }, { signal: ctl.signal }).catch((e: unknown) => {
        if (e instanceof ApiError && (e.status === 403 || e.status === 422)) stop();
      });
    };
    const stop = () => {
      stopped = true;
      clearInterval(interval);
      clearTimeout(resume);
    };
    const onInput = (e: Event) => {
      if (e.type === "mousemove") {
        if (Date.now() - lastMove < MOVE_THROTTLE_MS) return;
        lastMove = Date.now();
      }
      const wasIdle = !active();
      lastInput = Date.now();
      if (wasIdle && document.visibilityState === "visible") {
        clearTimeout(resume);
        resume = setTimeout(() => beat(true), RESUME_MS);
      }
    };
    const phase = setTimeout(() => {
      beat();
      interval = setInterval(beat, PERIOD_MS);
    }, Math.random() * PERIOD_MS);
    for (const t of INPUTS) window.addEventListener(t, onInput, { passive: true });
    window.addEventListener("mousemove", onInput, { passive: true });
    return () => {
      stop();
      clearTimeout(phase);
      ctl.abort();
      for (const t of INPUTS) window.removeEventListener(t, onInput);
      window.removeEventListener("mousemove", onInput);
    };
  }, [enabled, courseId, area]);
}
