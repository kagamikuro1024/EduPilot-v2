"use client";

import { useCallback, useSyncExternalStore } from "react";

// Trạng thái giả lập (SRS FR-X3): MỘT khoá localStorage `ep_demo_state` chứa mọi lát (slice) theo tên.
// Đổi vai vẫn thấy hệ quả của vai trước; `resetDemo()` xoá khoá. Không lưu gì giống token.
export const DEMO_STATE_KEY = "ep_demo_state";

type Bag = Record<string, unknown>;

let cache: Bag | null = null;
const listeners = new Set<() => void>();
const EMPTY: Bag = {};

function read(): Bag {
  if (cache) return cache;
  try {
    const raw = localStorage.getItem(DEMO_STATE_KEY);
    cache = raw ? (JSON.parse(raw) as Bag) : {};
  } catch {
    cache = {};
  }
  return cache;
}

function write(next: Bag) {
  cache = next;
  try {
    localStorage.setItem(DEMO_STATE_KEY, JSON.stringify(next));
  } catch {
    /* chế độ riêng tư / hết dung lượng: giữ trong bộ nhớ */
  }
  listeners.forEach((l) => l());
}

function subscribe(cb: () => void) {
  listeners.add(cb);
  const onStorage = (e: StorageEvent) => {
    if (e.key === DEMO_STATE_KEY) {
      cache = null;
      cb();
    }
  };
  window.addEventListener("storage", onStorage);
  return () => {
    listeners.delete(cb);
    window.removeEventListener("storage", onStorage);
  };
}

/**
 * Đọc / ghi một lát trạng thái giả lập. `initial` là giá trị gốc khi chưa ai ghi (cũng là giá trị khi render
 * ở server nên không lệch hydration). Dùng chung qua các route: cùng `key` = cùng dữ liệu.
 * Cập nhật nhận giá trị mới hoặc hàm `(prev) => next`; `prev` luôn là giá trị hiện hành, không phải `initial` cũ.
 */
export function useDemoSlice<T>(key: string, initial: T): [T, (next: T | ((prev: T) => T)) => void] {
  const bag = useSyncExternalStore(subscribe, read, () => EMPTY);
  const value = (key in bag ? bag[key] : initial) as T;
  const set = useCallback(
    (next: T | ((prev: T) => T)) => {
      const cur = read();
      const prev = (key in cur ? cur[key] : initial) as T;
      write({ ...cur, [key]: typeof next === "function" ? (next as (p: T) => T)(prev) : next });
    },
    // `initial` thường là hằng số module; không đưa vào deps để `set` ổn định.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [key],
  );
  return [value, set];
}

/** `Đặt lại dữ liệu demo`: xoá khoá duy nhất, mọi màn về dữ liệu gốc. */
export function resetDemo() {
  try {
    localStorage.removeItem(DEMO_STATE_KEY);
  } catch {
    /* bỏ qua */
  }
  cache = {};
  listeners.forEach((l) => l());
}
