/* eslint-disable @typescript-eslint/no-explicit-any */
// TC-UX01-8 (SmkS4, 2026-11-21): LCP bằng PerformanceObserver (cùng cách docs/sprints/9.1/audit/scripts/shoot.mjs) trên 7 route, 3 lượt, trung vị.
// «Trước» = docs/sprints/9.1/audit/data/*.json (audit trước 9.1, 1440 px, trạng thái full, cùng vai); «sau» = HEAD trên ep-r. Đạt khi sau ≤ trước + 100 ms.
import { test } from "@playwright/test";
import { courseId, expect, login } from "./r91lib";
test.skip(!process.env.QC_REAL, "QC_REAL");
test.describe.configure({ mode: "serial" });
const BEFORE: Record<string, number> = { // vai|route → LCP trước (ms)
  "teacher|/": 268, "teacher|/threads": 504, "teacher|/inbox": 1752, "teacher|/gradebook": 504, "teacher|/settings/llm": 488,
  "student|/chat": 524, "student|/me": 2008,
};
const USER: Record<string, string> = { teacher: "teacher", student: "sv.kha" };
test("TC-UX01-8 LCP sau ≤ trước + 100 ms trên 7 route (3 lượt, trung vị)", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop"); test.setTimeout(450_000);
  const rows: { k: string; before: number; runs: number[]; med: number; ok: boolean }[] = [];
  for (const role of ["teacher", "student"]) {
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh" });
    await ctx.addInitScript(() => {
      (window as any).__m = { lcp: 0 };
      new PerformanceObserver((l) => { for (const e of l.getEntries()) (window as any).__m.lcp = Math.round(e.startTime); }).observe({ type: "largest-contentful-paint", buffered: true });
    });
    const page = await ctx.newPage(); await login(page, USER[role]);
    const c = await courseId(page, "761987");
    await page.evaluate((id) => localStorage.setItem("ep:ui:course", id), c);
    for (const k of Object.keys(BEFORE).filter((x) => x.startsWith(role + "|"))) {
      const route = k.split("|")[1]; const runs: number[] = [];
      for (let i = 0; i < 3; i++) {
        await page.goto(route, { waitUntil: "domcontentloaded" });
        await page.waitForFunction(() => (document.querySelector("main")?.innerText ?? "").trim().length > 20, undefined, { timeout: 25_000 }).catch(() => undefined);
        await page.waitForTimeout(2500);
        runs.push(await page.evaluate(() => (window as any).__m.lcp));
      }
      const med = [...runs].sort((a, b) => a - b)[1];
      rows.push({ k, before: BEFORE[k], runs, med, ok: med <= BEFORE[k] + 100 });
      console.log(`INFO UX01-8 ${k}: trước ${BEFORE[k]} ms | sau ${JSON.stringify(runs)} trung vị ${med} ms | ${med <= BEFORE[k] + 100 ? "ĐẠT" : "VƯỢT"} (hiệu ${med - BEFORE[k]} ms)`);
    }
    await ctx.close();
  }
  expect(rows.length).toBe(Object.keys(BEFORE).length);
  expect(rows.filter((r) => !r.ok).map((r) => `${r.k}: ${r.med} > ${r.before}+100`), "route có LCP sau vượt trước + 100 ms").toEqual([]);
});
