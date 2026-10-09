// QC tự đo (sprint 5.5): chạy bằng `QC_ROUTES='[["/dev/panels?surface=a&role=teacher","teacher"]]' QC_TAG=ui01 pnpm exec playwright test qc-measure.spec.ts`
// Sao chép vào frontend/e2e khi chạy, xoá sau (không commit ở frontend). Không dùng mã đo của dev; chỉ dùng `asDemo` để vào phiên giả.
import { test } from "@playwright/test";
import { writeFileSync, mkdirSync } from "node:fs";
import { asDemo, type DemoRole } from "./support/session";

const ROUTES: Array<[string, string]> = JSON.parse(process.env.QC_ROUTES ?? "[]");
const TAG = process.env.QC_TAG ?? "x";
const WIDTHS = (process.env.QC_WIDTHS ?? "1440,1024,375").split(",").map(Number);
const OUT = process.env.QC_OUT ?? "../docs/sprints/5.5/qc";
const SHOTS = process.env.QC_SHOTS === "1";
const results: unknown[] = [];

test.describe.configure({ mode: "serial" });
for (const [url, role] of ROUTES) for (const w of WIDTHS) {
  test(`${TAG} ${url} ${role} ${w}`, async ({ browser }) => {
    const ctx = await browser.newContext({ viewport: { width: w, height: w <= 480 ? 812 : 900 }, ignoreHTTPSErrors: true });
    await asDemo(ctx, role as DemoRole);
    const page = await ctx.newPage();
    await page.clock.install({ time: new Date("2026-10-29T09:20:00+07:00") });
    await page.goto(process.env.QC_BASE + url, { waitUntil: "networkidle" });
    await page.waitForTimeout(600);
    const m = await page.evaluate(() => {
      const q = (s: string) => Array.from(document.querySelectorAll<HTMLElement>(s));
      const panels = q("[data-ep-panel]");
      const nest = document.querySelectorAll("[data-ep-panel] [data-ep-panel]").length;
      const title = document.querySelectorAll("[data-ep-panel] h1, [data-ep-panel] h2.ep-section-title").length;
      let strongBad = 0;
      for (const p of panels) { const s = p.querySelectorAll('[data-tone="strong"]'); if (s.length > 3) strongBad++; s.forEach((e) => { if (e.querySelector("[data-ep-panel],[data-ep-panel-section]")) strongBad++; }); }
      // WALL
      let wall = 0; const parents = new Map<Element, HTMLElement[]>();
      for (const p of panels) { const par = p.parentElement!; parents.set(par, [...(parents.get(par) ?? []), p]); }
      for (const [, kids] of parents) for (const a of kids) { const ra = a.getBoundingClientRect(); if (ra.top > 800) continue;
        const same = kids.filter((b) => { const rb = b.getBoundingClientRect(); return Math.abs(rb.top - ra.top) <= 2 && Math.abs(rb.width - ra.width) <= 2 && rb.top <= 800; }); if (same.length >= 3) { wall++; break; } }
      const cs = (e: Element) => getComputedStyle(e);
      const p0 = panels[0]; const c0 = p0 ? cs(p0) : null;
      const rects = panels.map((p) => p.getBoundingClientRect());
      const edge = rects.length ? Math.min(...rects.map((r) => Math.min(r.left, innerWidth - r.right))) : null;
      const touchEls = innerWidth <= 480 ? q('a,button,input,select,textarea,[role="button"],[role="tab"],[role="radio"],[role="checkbox"]').filter((e) => { const r = e.getBoundingClientRect(); return r.width > 0 && r.height > 0 && getComputedStyle(e).visibility !== "hidden" && (r.height < 44 || r.width < 44) && !e.closest("[hidden]"); }).map((e) => `${e.tagName.toLowerCase()}[${(e.getAttribute('aria-label') ?? e.textContent ?? '').trim().slice(0, 30)}] ${Math.round(e.getBoundingClientRect().width)}x${Math.round(e.getBoundingClientRect().height)}`) : null;
      const touchBad = touchEls ? touchEls.length : null;
      const probe = (v: string) => { const d = document.createElement("div"); d.style.cssText = `position:absolute;background:var(${v});color:var(${v})`; document.body.appendChild(d); const bg = getComputedStyle(d).backgroundColor; d.remove(); const cv = document.createElement('canvas'); cv.width = cv.height = 1; const x = cv.getContext('2d', { colorSpace: 'srgb', willReadFrequently: true })!; x.clearRect(0, 0, 1, 1); x.fillStyle = '#000'; x.fillStyle = bg; x.fillRect(0, 0, 1, 1); const p = x.getImageData(0, 0, 1, 1).data; return `rgb(${p[0]}, ${p[1]}, ${p[2]})`; };
      const tokens: Record<string, string> = {}; for (const v of ["--ep-canvas", "--ep-surface", "--ep-surface-strong", "--ep-surface-subtle", "--ep-red-soft", "--ep-ink", "--ep-ink-2", "--ep-ink-3", "--ep-red", "--ep-green", "--ep-blue", "--ep-panel-border", "--ep-radius-panel", "--ep-elevation-1"]) { try { tokens[v] = probe(v); } catch { tokens[v] = "?"; } }
      const raw = (v: string) => getComputedStyle(document.documentElement).getPropertyValue(v).trim();
      const rawTok: Record<string, string> = {}; for (const v of ["--ep-radius-panel", "--ep-elevation-1", "--ep-panel-border"]) rawTok[v] = raw(v);
      return {
        panels: panels.length, nest, title, strongBad, wall, edge: edge === null ? null : Math.round(edge * 100) / 100,
        ox: document.documentElement.scrollWidth > innerWidth + 1 ? 1 : 0, touchBad, touchEls,
        bodyBg: cs(document.body).backgroundColor, panelBg: c0?.backgroundColor ?? null, panelRadius: c0?.borderTopLeftRadius ?? null, panelShadow: c0?.boxShadow ?? null, panelBorder: c0 ? `${c0.borderTopWidth} ${c0.borderTopStyle} ${c0.borderTopColor}` : null,
        dataSurface: document.documentElement.getAttribute("data-surface"), tokens, rawTok,
        h1: q("h1").length, buttonsPrimary: q('button[class*="primary" i],a[class*="primary" i]').length,
      };
    });
    if (SHOTS) { mkdirSync(`${OUT}/shots/${TAG}`, { recursive: true }); await page.screenshot({ path: `${OUT}/shots/${TAG}/${url.replace(/[^a-z0-9]+/gi, "_").replace(/^_|_$/g, "")}-${role}-${w}.png`, fullPage: false }); }
    results.push({ url, role, w, ...m });
    await ctx.close();
  });
}
test.afterAll(() => { mkdirSync(`${OUT}/measure`, { recursive: true }); writeFileSync(`${OUT}/measure/${TAG}.json`, JSON.stringify(results, null, 1)); });
