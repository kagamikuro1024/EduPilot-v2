"use client";

import { useLayoutEffect } from "react";
import { stashFromLocation } from "./stash";

/** Chạy TRƯỚC `AuthGate` (đứng trước nó trong layout): cất token rồi xoá fragment. */
export function StashGate() {
  // Lần dựng đầu; rồi mỗi lần đổi hash (mở liên kết QR mới trong cùng tab) và mỗi lần Safari phục hồi trang từ bfcache (DEF-MOB-05).
  useLayoutEffect(() => {
    stashFromLocation();
    const onShow = (e: PageTransitionEvent) => {
      if (e.persisted) stashFromLocation();
    };
    window.addEventListener("hashchange", stashFromLocation);
    window.addEventListener("pageshow", onShow);
    return () => {
      window.removeEventListener("hashchange", stashFromLocation);
      window.removeEventListener("pageshow", onShow);
    };
  }, []);
  return null;
}
