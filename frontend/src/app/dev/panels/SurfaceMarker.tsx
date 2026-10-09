"use client";

import { useLayoutEffect } from "react";

/** Đặt `data-surface` trên <html> (panel-variants.css đọc thuộc tính này) và gỡ khi rời trang. */
export function SurfaceMarker({ surface }: { surface: "a" | "b" | "c" }) {
  useLayoutEffect(() => {
    const root = document.documentElement;
    root.setAttribute("data-surface", surface);
    return () => root.removeAttribute("data-surface");
  }, [surface]);
  return null;
}
