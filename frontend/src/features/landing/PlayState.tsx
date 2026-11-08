"use client";

import { useEffect } from "react";

/** Gắn / gỡ `data-playing` trên khung video theo sự kiện phát / dừng (CSS dùng nó để ẩn nút phát vẽ sẵn). Không render gì. */
export function PlayState({ video }: { video: string }) {
  useEffect(() => {
    const v = document.getElementById(video) as HTMLVideoElement | null;
    const box = v?.parentElement;
    if (!v || !box) return;
    const on = () => box.setAttribute("data-playing", "");
    const off = () => box.removeAttribute("data-playing");
    v.addEventListener("play", on);
    v.addEventListener("pause", off);
    v.addEventListener("ended", off);
    return () => {
      v.removeEventListener("play", on);
      v.removeEventListener("pause", off);
      v.removeEventListener("ended", off);
    };
  }, [video]);
  return null;
}
