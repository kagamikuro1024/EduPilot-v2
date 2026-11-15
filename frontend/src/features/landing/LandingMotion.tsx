"use client";

import { useEffect } from "react";

/**
 * Phần cần JS của trang giới thiệu (không render gì), theo mẫu `mau/landing.html`: (1) đích con trỏ của đoạn sản phẩm chạy đo từ DOM + nút Tạm dừng (WCAG 2.2.2);
 * (2) tab Giảng viên / Sinh viên (mũi tên trái / phải); (3) `Xem video` mở `<dialog>` gốc, chỉ tải khi mở, đóng thì dừng; (4) hiện theo cuộn một lần + số 146 đếm lên;
 * (5) khung ghim ba màn: cuộn trong mục đổi màn, vạch đỏ chạy theo cuộn. Chuyển động (4)(5) chỉ khi không `prefers-reduced-motion` (lớp `m-on` do script đầu trang đặt).
 */
const PAUSED_KEY = "ep_demo_paused";

export function LandingMotion() {
  useEffect(() => {
    const root = document.querySelector<HTMLElement>("[data-part=landing]");
    if (!root) return;
    const html = document.documentElement;
    const $ = <T extends HTMLElement>(sel: string, ctx: ParentNode = root) => ctx.querySelector<T>(sel);
    const $$ = <T extends HTMLElement>(sel: string, ctx: ParentNode = root) => [...ctx.querySelectorAll<T>(sel)];
    const off: Array<() => void> = [];
    const on = <T extends EventTarget>(t: T, ev: string, fn: EventListenerOrEventListenerObject, opt?: AddEventListenerOptions) => {
      t.addEventListener(ev, fn, opt);
      off.push(() => t.removeEventListener(ev, fn, opt));
    };
    root.setAttribute("data-js", "");
    const motion = !matchMedia("(prefers-reduced-motion: reduce)").matches;
    if (motion) html.classList.add("m-on");

    // đoạn sản phẩm chạy
    const demo = $("[data-demo]");
    const pause = $<HTMLButtonElement>("[data-pause]");
    if (demo && pause) {
      const place = () => {
        const b = demo.getBoundingClientRect();
        const k = b.width / demo.offsetWidth || 1;
        const at = (el: HTMLElement | null, n: number) => {
          if (!el) return;
          const r = el.getBoundingClientRect();
          demo.style.setProperty(`--x${n}`, `${(r.left + r.width / 2 - b.left) / k}px`);
          demo.style.setProperty(`--y${n}`, `${(r.top + r.height / 2 - b.top) / k}px`);
        };
        at($("[data-t1]", demo), 1);
        at($("[data-t2]", demo), 2);
        demo.style.setProperty("--x0", `${demo.offsetWidth * 0.46}px`);
        demo.style.setProperty("--y0", `${demo.offsetHeight * 0.56}px`);
      };
      place();
      on(window, "resize", place);
      const ro = new ResizeObserver(place);
      ro.observe(demo);
      off.push(() => ro.disconnect());
      void document.fonts?.ready.then(place);
      const apply = (p: boolean) => {
        demo.classList.toggle("is-paused", p);
        pause.setAttribute("aria-pressed", String(p));
        pause.setAttribute("aria-label", p ? "Phát tiếp" : "Tạm dừng");
        pause.dataset.paused = p ? "1" : "";
      };
      try {
        apply(sessionStorage.getItem(PAUSED_KEY) === "1"); // nhớ trong phiên (WCAG 2.2.2)
      } catch {
        /* storage bị chặn: coi như chưa dừng */
      }
      on(pause, "click", () => {
        const p = !demo.classList.contains("is-paused");
        apply(p);
        try {
          sessionStorage.setItem(PAUSED_KEY, p ? "1" : "0");
        } catch {
          /* như trên */
        }
      });
    }

    // hai vai: tab bấm, mũi tên trái / phải
    const tabs = $$<HTMLButtonElement>("[role=tab]");
    const roleShots = $$<HTMLElement>("[data-role-shots] img");
    const pick = (t: HTMLButtonElement) => {
      tabs.forEach((x, i) => {
        const on_ = x === t;
        x.setAttribute("aria-selected", String(on_));
        x.tabIndex = on_ ? 0 : -1;
        const panel = document.getElementById(x.getAttribute("aria-controls") ?? "");
        if (panel) panel.hidden = !on_;
        roleShots[i]?.classList.toggle("is-on", on_);
      });
      $("[data-ind]")?.style.setProperty("--i", String(tabs.indexOf(t)));
    };
    tabs.forEach((t) => {
      on(t, "click", () => pick(t));
      on(t, "keydown", (e) => {
        const k = (e as KeyboardEvent).key;
        if (k !== "ArrowRight" && k !== "ArrowLeft") return;
        const n = tabs[(tabs.indexOf(t) + (k === "ArrowRight" ? 1 : tabs.length - 1)) % tabs.length];
        n.focus();
        pick(n);
      });
    });
    if (tabs[0]) pick(tabs[0]);

    // video đầy đủ: <dialog> gốc (khoá focus, Esc, trả focus do trình duyệt lo). Chưa bấm thì không có <source> nào được nạp; đóng thì dừng và bỏ nguồn.
    const dlg = $<HTMLDialogElement>("[data-video-dialog]");
    let full = dlg ? $<HTMLVideoElement>("video", dlg) : null;
    const sources = full ? [...full.querySelectorAll("source")].map((x) => x.cloneNode(true)) : [];
    full?.querySelectorAll("source").forEach((x) => x.remove());
    $$("[data-open-video]").forEach((b) =>
      on(b, "click", () => {
        if (!dlg || !full) return;
        if (!full.getAttribute("poster") && full.dataset.poster) full.setAttribute("poster", full.dataset.poster);
        if (!full.querySelector("source")) full.append(...sources.map((x) => x.cloneNode(true)));
        dlg.showModal();
      }),
    );
    if (dlg) {
      // Chromium không xoá `currentSrc` khi gỡ <source> + load(), nên thay hẳn phần tử bằng bản không nguồn (huỷ luôn việc tải)
      on(dlg, "close", () => {
        if (!full) return;
        full.pause();
        const fresh = full.cloneNode(false) as HTMLVideoElement;
        full.replaceWith(fresh);
        full = fresh;
      });
      on(dlg, "click", (e) => { if (e.target === dlg) dlg.close(); });
    }

    if (motion) {
      // hiện theo cuộn, một lần; 146 đếm lên một lần
      const count = (n: HTMLElement) => {
        const to = Number(n.dataset.count);
        const t0 = performance.now();
        const f = (t: number) => {
          const k = Math.min(1, (t - t0) / 1000);
          n.textContent = String(Math.round(to * (1 - Math.pow(1 - k, 3))));
          if (k < 1) requestAnimationFrame(f);
        };
        requestAnimationFrame(f);
      };
      let seen = false;
      const show = (el: Element) => {
        el.classList.add("is-in");
        const n = $<HTMLElement>("[data-count]", el);
        if (n) count(n);
      };
      const io = new IntersectionObserver(
        (es) => {
          seen = true;
          es.forEach((e) => {
            if (e.isIntersecting) {
              io.unobserve(e.target);
              show(e.target);
            }
          });
        },
        { threshold: 0.35 },
      );
      html.classList.add("m-armed"); // trạng thái ẩn chỉ có hiệu lực từ đây: không script thì thấy đủ
      $$("[data-reveal]").forEach((el) => io.observe(el));
      const t = setTimeout(() => { if (!seen) $$("[data-reveal]").forEach(show); }, 1500);
      off.push(() => { io.disconnect(); clearTimeout(t); });

      // khung ghim: cuộn trong mục để đổi màn
      const story = $("#ba-man");
      if (story) {
        const items = $$("[data-story-item]", story);
        const shots = $$<HTMLElement>("[data-story-shots] img", story);
        const wide = matchMedia("(min-width: 1100px)");
        let queued = false;
        const scrub = () => {
          queued = false;
          if (!wide.matches) return;
          const r = story.getBoundingClientRect();
          const p = Math.min(1, Math.max(0, -r.top / (r.height - innerHeight)));
          const step = Math.min(2, Math.floor(p * 3));
          items.forEach((it, i) => {
            it.style.setProperty("--pp", String(Math.min(1, Math.max(0, p * 3 - i))));
            if (i === step) it.setAttribute("aria-current", "true");
            else it.removeAttribute("aria-current");
          });
          shots.forEach((im, i) => im.classList.toggle("is-on", i === step));
        };
        const sync = () => {
          story.classList.toggle("is-pinned", wide.matches);
          scrub();
        };
        on(wide, "change", sync);
        sync();
        on(window, "scroll", () => { if (!queued) { queued = true; requestAnimationFrame(scrub); } }, { passive: true });
      }
    }
    return () => {
      off.forEach((f) => f());
      html.classList.remove("m-on", "m-armed");
    };
  }, []);
  return null;
}
