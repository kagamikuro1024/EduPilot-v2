"use client";

import { useCallback, useEffect, useRef, useState } from "react";

// Bản nháp tự lưu (US-PU-03 AC18): localStorage `ep:draft:<userId|anon>:<key>` = {v:1,text,savedAt}. Chữ KHÔNG BAO GIỜ mất
// vì lưu lỗi: giá trị luôn nằm trong state; lỗi lưu chỉ đổi `status`.
export type DraftStatus = "idle" | "saving" | "saved" | "error";

const PREFIX = "ep:draft:";
const DEBOUNCE_MS = 2000;
const MAX_BYTES = 100 * 1024;
const MAX_AGE_MS = 30 * 24 * 3600 * 1000;
const FORBIDDEN = /secret|password|apikey|key/i;

let pruned = false;
function pruneOld() {
  if (pruned || typeof localStorage === "undefined") return;
  pruned = true;
  try {
    for (let i = localStorage.length - 1; i >= 0; i--) {
      const k = localStorage.key(i);
      if (!k?.startsWith(PREFIX)) continue;
      try {
        const d = JSON.parse(localStorage.getItem(k) ?? "") as { savedAt?: number };
        if (!d.savedAt || Date.now() - d.savedAt > MAX_AGE_MS) localStorage.removeItem(k);
      } catch {
        localStorage.removeItem(k);
      }
    }
  } catch { /* bộ nhớ không đọc được */ }
}

export function useAutosaveDraft(key: string, opts: { userId?: string } = {}) {
  if (FORBIDDEN.test(key)) throw new Error(`useAutosaveDraft: không lưu nháp cho trường bí mật ("${key}")`);
  const storageKey = `${PREFIX}${opts.userId ?? "anon"}:${key}`;
  const [value, setValueState] = useState("");
  const [status, setStatus] = useState<DraftStatus>("idle");
  const latest = useRef("");
  const dirty = useRef(false);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const flush = useCallback(() => {
    if (timer.current) clearTimeout(timer.current);
    timer.current = null;
    if (!dirty.current) return;
    dirty.current = false;
    const text = latest.current;
    if (text.length * 2 > MAX_BYTES) {
      setStatus("error");
      return;
    }
    try {
      if (text === "") localStorage.removeItem(storageKey);
      else localStorage.setItem(storageKey, JSON.stringify({ v: 1, text, savedAt: Date.now() }));
      setStatus("saved");
    } catch {
      setStatus("error"); // đầy / bị chặn: chữ vẫn nằm trong ô
    }
  }, [storageKey]);

  // khôi phục khi mở lại (kể cả sau khi đóng tab đột ngột). Đọc kho ngoài sau khi gắn kết nên cần setState trong effect.
  /* eslint-disable react-hooks/set-state-in-effect */
  useEffect(() => {
    pruneOld();
    try {
      const raw = localStorage.getItem(storageKey);
      if (raw) {
        const d = JSON.parse(raw) as { v?: number; text?: string };
        if (d.v === 1 && typeof d.text === "string") {
          latest.current = d.text;
          setValueState(d.text);
          setStatus("saved");
        }
      }
    } catch { /* nháp hỏng: bỏ */ }
  }, [storageKey]);
  /* eslint-enable react-hooks/set-state-in-effect */

  // lưu ngay khi ẩn tab / rời trang / gỡ gắn kết
  useEffect(() => {
    const hide = () => document.visibilityState === "hidden" && flush();
    document.addEventListener("visibilitychange", hide);
    window.addEventListener("pagehide", flush);
    return () => {
      document.removeEventListener("visibilitychange", hide);
      window.removeEventListener("pagehide", flush);
      flush();
    };
  }, [flush]);

  const setValue = useCallback(
    (v: string) => {
      latest.current = v;
      dirty.current = true;
      setValueState(v);
      setStatus("saving");
      if (timer.current) clearTimeout(timer.current);
      timer.current = setTimeout(flush, DEBOUNCE_MS);
    },
    [flush],
  );

  /** Gọi sau khi GỬI THÀNH CÔNG. */
  const clear = useCallback(() => {
    if (timer.current) clearTimeout(timer.current);
    timer.current = null;
    dirty.current = false;
    latest.current = "";
    setValueState("");
    setStatus("idle");
    try { localStorage.removeItem(storageKey); } catch { /* bỏ qua */ }
  }, [storageKey]);

  return { value, setValue, status, clear };
}

/** Đặt sẵn bản nháp cho một khoá (vd. chuyển nháp Threads sang ô soạn chat riêng): trang đích mở ra sẽ khôi phục ĐÚNG từng byte. */
export function seedDraft(key: string, text: string, userId?: string) {
  try {
    localStorage.setItem(`${PREFIX}${userId ?? "anon"}:${key}`, JSON.stringify({ v: 1, text, savedAt: Date.now() }));
  } catch { /* bộ nhớ đầy / bị chặn: bỏ qua */ }
}
