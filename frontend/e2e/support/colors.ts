import type { Page } from "@playwright/test";

// Đo màu trong trình duyệt: màu của token (đã giải) → sRGB 8 bit → tỉ lệ tương phản WCAG. Dùng chung cho panels.spec.ts và contrast.spec.ts.
export const COLOR_TOOLS = `(() => {
  const probe = document.createElement("span");
  document.body.appendChild(probe);
  const canvas = document.createElement("canvas");
  canvas.width = canvas.height = 1;
  const ctx = canvas.getContext("2d", { willReadFrequently: true });
  const rgb = (css) => { ctx.clearRect(0, 0, 1, 1); ctx.fillStyle = "#000"; ctx.fillStyle = css; ctx.fillRect(0, 0, 1, 1); const d = ctx.getImageData(0, 0, 1, 1).data; return [d[0], d[1], d[2]]; };
  const token = (name) => { probe.style.color = "var(" + name + ")"; return rgb(getComputedStyle(probe).color); };
  const lin = (v) => { const c = v / 255; return c <= 0.04045 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4); };
  const lum = ([r, g, b]) => 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b);
  const ratio = (a, b) => { const x = lum(a), y = lum(b); return (Math.max(x, y) + 0.05) / (Math.min(x, y) + 0.05); };
  return { rgb, token, ratio, done: () => probe.remove() };
})()`;

/** Màu (`var(--x)`, `oklch(...)`, chuỗi computed) → sRGB 8 bit "r,g,b" qua canvas: so được giữa các cách Chrome tuần tự hoá (oklch / lab / rgb). */
export async function rgbOf(page: Page, expr: string) {
  return page.evaluate((e) => {
    const s = document.createElement("span");
    document.body.appendChild(s);
    s.style.color = e;
    const css = getComputedStyle(s).color;
    s.remove();
    const c = document.createElement("canvas");
    c.width = c.height = 1;
    const x = c.getContext("2d", { willReadFrequently: true })!;
    x.fillStyle = "#000";
    x.fillStyle = css;
    x.fillRect(0, 0, 1, 1);
    return Array.from(x.getImageData(0, 0, 1, 1).data.slice(0, 3)).join(",");
  }, expr);
}
