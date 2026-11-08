"use client";

import { useEffect } from "react";

const CLIP = "/landing/gioi-thieu-clip-v1";
const PAUSED_KEY = "ep_clip_paused";

/**
 * Hai việc cần JS của trang giới thiệu (không render gì): (1) clip lặp ở màn đầu — nạp SAU sự kiện `load` (poster đã giữ LCP), không bao giờ chạy khi
 * `prefers-reduced-motion`, `Save-Data`, dưới 720 px hoặc tab ẩn; có nút Tạm dừng / Phát (WCAG 2.2.2), nhớ trong phiên; (2) nút `Xem video` mở video đầy đủ trong `<dialog>` gốc
 * (khoá focus, Esc, trả focus do trình duyệt lo); đóng thì dừng video; poster của video đầy đủ chỉ gán khi mở. Mọi hiệu ứng khác là CSS.
 */
export function LandingMedia() {
  useEffect(() => {
    const root = document.querySelector<HTMLElement>("[data-part=landing]");
    if (!root) return;
    const off: Array<() => void> = [];
    const on = <T extends EventTarget>(t: T, ev: string, fn: EventListener, opt?: boolean) => {
      t.addEventListener(ev, fn, opt);
      off.push(() => t.removeEventListener(ev, fn, opt));
    };

    // ---- clip
    const clip = root.querySelector<HTMLVideoElement>("[data-clip]");
    const toggle = root.querySelector<HTMLButtonElement>("[data-clip-toggle]");
    const reduce = matchMedia("(prefers-reduced-motion: reduce)").matches;
    const saveData = (navigator as Navigator & { connection?: { saveData?: boolean } }).connection?.saveData === true;
    const wide = matchMedia("(min-width: 720px)").matches;
    let userPaused = false;
    try {
      userPaused = sessionStorage.getItem(PAUSED_KEY) === "1";
    } catch {
      /* storage bị chặn: coi như chưa dừng */
    }
    const label = () => {
      if (!toggle) return;
      toggle.textContent = userPaused ? "Phát" : "Tạm dừng";
      toggle.setAttribute("aria-pressed", String(userPaused));
    };
    const sync = () => {
      if (!clip || !clip.querySelector("source")) return;
      if (userPaused || document.hidden) clip.pause();
      else void clip.play().catch(() => undefined);
    };
    const start = () => {
      if (!clip || reduce || saveData || !wide || document.hidden) return;
      for (const [ext, type] of [["mp4", "video/mp4"], ["webm", "video/webm"]] as const) {
        const src = document.createElement("source");
        src.src = `${CLIP}.${ext}`;
        src.type = type;
        clip.append(src);
      }
      // lỗi tải: giữ poster, bỏ nguồn, ẩn nút (không còn gì để dừng)
      on(clip.lastElementChild as HTMLElement, "error", () => {
        clip.querySelectorAll("source").forEach((x) => x.remove());
        if (toggle) toggle.hidden = true;
      });
      on(clip, "playing", () => clip.setAttribute("data-ready", ""));
      clip.load();
      if (toggle) toggle.hidden = false;
      label();
      sync();
    };
    if (document.readyState === "complete") start();
    else on(window, "load", start);
    if (toggle) {
      on(toggle, "click", () => {
        userPaused = !userPaused;
        try {
          sessionStorage.setItem(PAUSED_KEY, userPaused ? "1" : "0");
        } catch {
          /* như trên */
        }
        label();
        sync();
      });
    }
    on(document, "visibilitychange", sync);

    // ---- hộp thoại video đầy đủ
    const dlg = root.querySelector<HTMLDialogElement>("[data-video-dialog]");
    const full = dlg?.querySelector<HTMLVideoElement>("video");
    const open = root.querySelector<HTMLElement>("[data-open-video]");
    if (dlg && full && open) {
      on(open, "click", () => {
        if (!full.getAttribute("poster") && full.dataset.poster) full.setAttribute("poster", full.dataset.poster);
        dlg.showModal();
      });
      on(dlg, "close", () => full.pause());
      on(dlg, "click", (e) => {
        if (e.target === dlg) dlg.close(); // bấm ra ngoài khung
      });
    }
    return () => off.forEach((f) => f());
  }, []);
  return null;
}
