import { readFileSync, writeFileSync } from "node:fs";
import { test } from "@playwright/test";
import { needReal, shot, uiLogin } from "./qc-real-lib";

needReal();
const S = JSON.parse(readFileSync("/tmp/q9s/ui.json", "utf8")) as { course: string; aid: string };
test("@real UIG 03-18 trang nộp bài ở 375 px", async ({ page }) => {
  const out: Record<string, unknown> = {};
  await page.setViewportSize({ width: 375, height: 812 });
  await page.addInitScript((c) => localStorage.setItem("ep:ui:course", c), S.course);
  await page.addInitScript(() => { (window as unknown as { __m: { lcp: number; cls: number } }).__m = { lcp: 0, cls: 0 }; new PerformanceObserver((l) => { for (const e of l.getEntries()) (window as unknown as { __m: { lcp: number } }).__m.lcp = e.startTime; }).observe({ type: "largest-contentful-paint", buffered: true }); new PerformanceObserver((l) => { for (const e of l.getEntries() as unknown as { value: number; hadRecentInput: boolean }[]) if (!e.hadRecentInput) (window as unknown as { __m: { cls: number } }).__m.cls += e.value; }).observe({ type: "layout-shift", buffered: true }); });
  await uiLogin(page, "sv35");
  const runs: { lcp: number; cls: number }[] = [];
  for (let i = 0; i < 3; i++) { await page.goto(`/assignments/${S.aid}?course=${S.course}`); await page.locator("main").waitFor(); await page.waitForTimeout(3500); runs.push(await page.evaluate(() => (window as unknown as { __m: { lcp: number; cls: number } }).__m)); }
  out.runs = runs;
  out.btn = await page.getByRole("button", { name: /Nộp bài/ }).first().evaluate((e) => { const r = e.getBoundingClientRect(); return { w: Math.round(r.width), h: Math.round(r.height) }; });
  out.zone = await page.getByLabel(/Vùng nộp bài/).first().evaluate((e) => { const r = (e.closest("label,div") as HTMLElement).getBoundingClientRect(); return { w: Math.round(r.width), h: Math.round(r.height) }; }).catch(() => null);
  out.scrollOk = await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth);
  const order: string[] = []; for (let i = 0; i < 8; i++) { await page.keyboard.press("Tab"); order.push(await page.evaluate(() => { const e = document.activeElement as HTMLElement; return (e.getAttribute("aria-label") ?? e.textContent ?? e.tagName).trim().slice(0, 30); })); } out.tabOrder = order;
  out.live = await page.evaluate(() => Array.from(document.querySelectorAll("[aria-live]")).map((e) => e.getAttribute("aria-live")));
  await shot(page, "p703-375-final");
  writeFileSync("/tmp/q9s/uig.json", JSON.stringify(out, null, 1));
});
