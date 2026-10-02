"use client";

import { useEffect, type RefObject } from "react";

/**
 * Hàng cuộn ngang thật sự: `data-scroll-x` chỉ có khi nội dung rộng hơn khung (00-6), và `data-fade` ("start" / "end")
 * làm mờ mép còn nội dung để người dùng thấy còn chip / tab phía sau (góp ý 24c). Đặt bằng DOM, không qua state.
 */
export function useScrollRow(ref: RefObject<HTMLDivElement | null>, count: number) {
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const update = () => {
      const max = el.scrollWidth - el.clientWidth;
      if (max > 1) el.setAttribute("data-scroll-x", "");
      else el.removeAttribute("data-scroll-x");
      const fade = [el.scrollLeft > 1 && "start", el.scrollLeft < max - 1 && max > 1 && "end"].filter(Boolean).join(" ");
      if (fade) el.setAttribute("data-fade", fade);
      else el.removeAttribute("data-fade");
    };
    update();
    el.addEventListener("scroll", update, { passive: true });
    const ro = new ResizeObserver(update);
    ro.observe(el);
    for (const child of Array.from(el.children)) ro.observe(child);
    return () => {
      el.removeEventListener("scroll", update);
      ro.disconnect();
    };
  }, [ref, count]);
}
