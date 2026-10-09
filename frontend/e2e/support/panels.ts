import type { Page } from "@playwright/test";

/** Phép đo luật Panel (D59) trên trang hiện tại: NEST, TITLE, STRONG, WALL, số panel. Dùng cho route thật và ca gieo vi phạm. */
export async function panelViolations(page: Page) {
  return page.evaluate(() => {
    const vis = (e: HTMLElement) => e.getClientRects().length > 0;
    const scope = document.querySelector("main") ?? document.body;
    const panels = [...scope.querySelectorAll<HTMLElement>("[data-ep-panel]")].filter(vis);
    const all = [...document.querySelectorAll<HTMLElement>("[data-ep-panel]")].filter(vis);
    const rects = panels.map((e) => e.getBoundingClientRect());
    let wall = 0;
    for (const r of rects) {
      if (r.top > 800) continue;
      wall = Math.max(wall, rects.filter((q) => q.top <= 800 && Math.abs(q.top - r.top) < 8 && Math.abs(q.width - r.width) < 4).length);
    }
    return {
      panels: panels.length,
      nest: document.querySelectorAll("[data-ep-panel] [data-ep-panel]").length,
      title: [...document.querySelectorAll<HTMLElement>("[data-ep-panel] h1, [data-ep-panel] h2")].filter(vis).length,
      strongOver: all.filter((e) => e.querySelectorAll('[data-tone="strong"]').length > 3).length,
      strongMax: Math.max(0, ...all.map((e) => e.querySelectorAll('[data-tone="strong"]').length)),
      strongNested: document.querySelectorAll('[data-tone="strong"] [data-ep-panel], [data-tone="strong"] [data-ep-panel-section]').length,
      wall: wall >= 3 ? wall : 0,
      h2: [...scope.querySelectorAll<HTMLElement>("h2")].filter(vis).map((h) => (h.textContent ?? "").trim()),
      left: rects.map((r) => Math.round(r.left * 100) / 100),
      right: rects.map((r) => Math.round((window.innerWidth - r.right) * 100) / 100),
      sw: document.documentElement.scrollWidth,
      vw: window.innerWidth,
    };
  });
}
