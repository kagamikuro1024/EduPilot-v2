"use client";

import { useSyncExternalStore } from "react";

/** `true` khi bề rộng khung nhìn ≥ `px` (theo `matchMedia`, đổi theo thời gian thực; phía máy chủ coi là hẹp). */
export function useMinWidth(px: number): boolean {
  return useSyncExternalStore(
    (cb) => {
      const m = window.matchMedia(`(min-width: ${px}px)`);
      m.addEventListener("change", cb);
      return () => m.removeEventListener("change", cb);
    },
    () => window.matchMedia(`(min-width: ${px}px)`).matches,
    () => false,
  );
}
