"use client";

import { useEffect } from "react";

// Hộp thoại modal (Dialog, Drawer, ConfirmIrreversible, CommandPalette) khoá cuộn nền: cuộn bánh xe / chạm không làm trang phía sau trôi.
// Đếm số hộp đang mở để chồng hộp thoại không mở khoá sớm.
let locks = 0;
let saved = { overflow: "", paddingRight: "" };

export function useScrollLock(active: boolean) {
  useEffect(() => {
    if (!active) return;
    const root = document.documentElement;
    if (locks++ === 0) {
      saved = { overflow: root.style.overflow, paddingRight: root.style.paddingRight };
      const bar = window.innerWidth - root.clientWidth; // bù độ rộng thanh cuộn để trang không giật ngang
      root.style.overflow = "hidden";
      if (bar > 0) root.style.paddingRight = `${bar}px`;
    }
    return () => {
      if (--locks === 0) {
        root.style.overflow = saved.overflow;
        root.style.paddingRight = saved.paddingRight;
      }
    };
  }, [active]);
}
