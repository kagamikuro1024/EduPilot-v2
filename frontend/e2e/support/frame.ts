import type { Page } from "@playwright/test";

export type FrameDeviation = { code: string; detail: string };
export type FrameRow = { kind: string | null; cols: number; headerH: number | null; devs: FrameDeviation[] };

/**
 * Đo hợp đồng khung trang (DESIGN.md §23) trên trang hiện tại, chỉ đọc DOM qua các móc `data-ep-*` (không đọc lớp CSS Module).
 * Mã phép khớp TL-1 mục 1.2: W (độ rộng / mép), H (đầu trang), P (panel), E (rỗng), T (chữ thừa), D (mật độ).
 */
export async function measureFrame(page: Page, opts: { wide: boolean }): Promise<FrameRow> {
  return page.evaluate((wide) => {
    const vis = (e: Element) => (e as HTMLElement).getClientRects().length > 0;
    const devs: { code: string; detail: string }[] = [];
    const add = (code: string, detail: string) => devs.push({ code, detail });
    const main = document.querySelector("main") ?? document.body;
    const pages = [...main.querySelectorAll<HTMLElement>("[data-ep-page]")].filter(vis);
    if (pages.length !== 1) {
      add("W1", `${pages.length} [data-ep-page]`);
      return { kind: null, cols: 0, headerH: null, devs };
    }
    const pg = pages[0];
    const kind = pg.getAttribute("data-ep-page");
    const cs = getComputedStyle(pg);
    const pr = pg.getBoundingClientRect();
    const colL = pr.left + parseFloat(cs.paddingLeft);
    const colR = pr.right - parseFloat(cs.paddingRight);
    const colW = colR - colL;
    const near = (a: number, b: number, tol = 1) => Math.abs(a - b) <= tol;
    if (wide) {
      const wantW = kind === "read" ? 800 : null;
      if (wantW !== null && !near(colW, wantW)) add("W2", `read ${Math.round(colW)} ≠ 800`);
      if (kind === "read") {
        const area = main.getBoundingClientRect();
        if (!near(colL - area.left, area.right - colR, 2)) add("W2", `read lệch tâm ${Math.round(colL - area.left)} / ${Math.round(area.right - colR)}`);
      }
      if (kind === "work" && !near(colL, 240, 2) && colW < 1600) add("W2", `work mép trái ${Math.round(colL)} ≠ 240`);
    } else if (!near(colW, window.innerWidth - 24, 2)) add("W5", `cột ${Math.round(colW)} ≠ ${window.innerWidth - 24}`);
    if (document.documentElement.scrollWidth > window.innerWidth + 1) add("W5", `tràn ngang ${document.documentElement.scrollWidth - window.innerWidth}`);
    // W3: panel cấp cao nhất phủ đúng cột
    const top = [...pg.querySelectorAll<HTMLElement>("[data-ep-panel]")].filter((p) => vis(p) && !p.parentElement?.closest("[data-ep-panel]") && !p.closest("[data-ep-split]:not([data-ep-split=list-detail])"));
    for (const p of top) {
      const r = p.getBoundingClientRect();
      if (!near(r.left, colL, 1) || !near(r.right, colR, 1)) add("W3", `panel ${Math.round(r.left)}→${Math.round(r.right)} ≠ cột ${Math.round(colL)}→${Math.round(colR)}`);
    }
    // H: đầu trang
    const headers = [...pg.querySelectorAll<HTMLElement>("[data-ep-page-header]")].filter(vis);
    const h1s = [...main.querySelectorAll("h1")].filter(vis);
    if (h1s.length !== 1) add("H1", `${h1s.length} h1`);
    let headerH: number | null = null;
    if (headers.length === 1) {
      const h = headers[0];
      headerH = Math.round(h.getBoundingClientRect().height);
      const h1 = h.querySelector("h1");
      if (h1 && !near(h1.getBoundingClientRect().left, colL, 1)) add("W4", `h1 lệch mép ${Math.round(h1.getBoundingClientRect().left - colL)}`);
      const extraP = [...h.querySelectorAll(":scope > p, :scope p")].filter((e) => !e.closest("[data-ep-header-meta]"));
      if (extraP.length) add("H2", `${extraP.length} <p> trong đầu trang`);
      const meta = h.querySelector<HTMLElement>("[data-ep-header-meta]");
      if (meta && wide) {
        const lh = parseFloat(getComputedStyle(meta).lineHeight) || 20;
        if (meta.getBoundingClientRect().height > lh * 1.6) add("H3", "meta quá một dòng");
      }
      const acts = h.querySelector<HTMLElement>("[data-ep-header-actions]");
      if (acts) {
        const btns = [...acts.querySelectorAll<HTMLElement>("button, a")].filter(vis);
        const prim = btns.filter((b) => b.getAttribute("data-variant") === "primary").length;
        if (prim > 1) add("H5", `${prim} nút primary`);
        if (wide && btns.length > 4) add("H5", `${btns.length} nút`);
        const h1r = h1?.getBoundingClientRect();
        const ar = acts.getBoundingClientRect();
        if (wide && h1r && Math.abs(h1r.top + h1r.height / 2 - (ar.top + ar.height / 2)) > 4) add("H4", "nút lệch tâm dọc so với h1");
        if (wide && !near(ar.right, colR, 1)) add("H4", `nút lệch mép phải ${Math.round(ar.right - colR)}`);
      }
    }
    // H6: nút primary ngoài đầu trang
    const stray = [...pg.querySelectorAll<HTMLElement>('[data-variant="primary"]')].filter((b) => vis(b) && !b.closest("[data-ep-page-header]") && !b.closest("[role=dialog]") && !b.closest("form"));
    const sections = pg.querySelectorAll("[data-ep-section]").length;
    if (headers[0]?.querySelector('[data-variant="primary"]') && stray.length && sections === 0) add("H6", `${stray.length} nút primary ngoài đầu trang`);
    // P1: đệm panel
    for (const p of top) {
      if (p.querySelector(":scope > [data-ep-split=list-detail]")) continue;
      const pad = parseFloat(getComputedStyle(p).paddingLeft);
      const want = wide ? 24 : 16;
      if (!near(pad, want, 1)) add("P1", `đệm ${pad} ≠ ${want}`);
    }
    // P3 / P4
    if (document.querySelectorAll("[data-ep-panel] [data-ep-panel]").length) add("P3", "panel lồng panel");
    for (const h2 of [...main.querySelectorAll("h2")].filter(vis)) if (!h2.hasAttribute("data-ep-section-title")) add("P4", `h2 tự dựng «${(h2.textContent ?? "").slice(0, 30)}»`);
    // E: rỗng
    for (const e of [...main.querySelectorAll<HTMLElement>("[data-ep-empty]")].filter(vis)) {
      const kids = [...e.children];
      const svgs = [...e.querySelectorAll("svg")].filter((s) => !s.closest("button, a"));
      if (svgs.length) add("E1", "icon trong rỗng");
      if (kids.some((k) => k.tagName === "P" && k !== kids[0])) add("E1", "chữ thừa trong rỗng");
      const title = (e.querySelector("p")?.textContent ?? "").trim();
      if (title.length > 40 || /\.\s/.test(title) || title.endsWith(".")) add("E3", `tiêu đề rỗng là câu «${title.slice(0, 40)}»`);
      const er = e.getBoundingClientRect();
      const cr = (e.parentElement ?? e).getBoundingClientRect();
      const kid = e.firstElementChild?.getBoundingClientRect();
      if (kid && wide && Math.abs(kid.left + kid.width / 2 - (er.left + er.width / 2)) > 2) add("E2", "khối rỗng không căn giữa");
      void cr;
      const btn = e.querySelector<HTMLElement>("button, a");
      if (btn && btn.getAttribute("data-variant") === "primary") add("E5", "nút rỗng là primary");
      const label = (btn?.textContent ?? "").trim();
      if (label) {
        const dup = [...main.querySelectorAll<HTMLElement>("[data-ep-header-actions] button, [data-ep-header-actions] a")].some((b) => (b.textContent ?? "").trim() === label);
        if (dup) add("E4", `nút rỗng «${label}» trùng đầu trang`);
      }
    }
    // T: chữ thừa
    const info = [...main.querySelectorAll("svg.lucide-info, svg.lucide-circle-help")].filter((s) => vis(s) && !s.closest("[role=status], [role=alert], [data-ep-notice]"));
    if (info.length) add("T1", `${info.length} icon giải thích`);
    return { kind, cols: Math.round(colW), headerH, devs };
  }, opts.wide);
}
