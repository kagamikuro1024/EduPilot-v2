import { appendFileSync } from "node:fs";
import { test } from "@playwright/test";
import { sql, PW, OUT } from "./qc-real-lib";
test.skip(!process.env.QC_REAL, "cần stack thật");
const RES = process.env.QC_RES ?? `${OUT}/results_tab.jsonl`;
const verdict = (tc: string, cond: boolean, why: string) => { appendFileSync(RES, JSON.stringify({ tc, kq: cond ? "P" : "F", note: why }) + "\n"); console.log("VERDICT", cond ? "P" : "F", tc, why.slice(0, 300)); };
// R10S (1c3ec9fa): bấm tab «Thi thử» ngay khi /practice còn tải (/me/courses trễ 3 s) ở 375 px, SV sv.kha lớp QD10A.
test("tab Thi thử bấm lúc còn tải @375", async ({ browser }) => {
  const cid = sql(`select id from courses where class_code='QD10A'`)[0][0];
  const ctx = await browser.newContext({ baseURL: "https://localhost:943", ignoreHTTPSErrors: true, viewport: { width: 375, height: 812 }, hasTouch: true, isMobile: true, locale: "vi-VN" });
  await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), cid);
  const page = await ctx.newPage(); await page.addLocatorHandler(page.getByRole("button", { name: "Đã hiểu" }), async (b) => { await b.click(); });
  await page.goto("/login"); await page.locator("input[type=email]").fill("sv.kha@edupilot.local"); await page.locator("input[type=password]").fill(PW); await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login")); await page.locator("[data-part=topbar]").waitFor();
  await page.route(/\/api\/v1\/me\/courses/, async (r) => { await new Promise((s) => setTimeout(s, 3500)); await r.continue(); });
  await page.goto("/practice"); const tab = page.getByRole("tab", { name: /Thi thử/ }); await tab.waitFor({ timeout: 15000 });
  const dis0 = await tab.getAttribute("disabled"); const aria0 = await tab.getAttribute("aria-disabled"); await page.screenshot({ path: `${OUT}/s10-tab375-loading.png` });
  const t0 = Date.now(); await tab.click({ timeout: 20000 }).catch(() => {}); const waited = Date.now() - t0;
  await page.waitForTimeout(1500); const start = page.getByRole("button", { name: "Bắt đầu thi thử" }); const ok = await start.waitFor({ timeout: 15000 }).then(() => true).catch(() => false);
  const sel = await tab.getAttribute("aria-selected"); await page.screenshot({ path: `${OUT}/s10-tab375-after.png` });
  const ov = await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth + 1);
  verdict("R10S-flake-tab", ok && sel === "true" && !ov, `375 touch, /me/courses trễ 3,5 s: tab «Thi thử» lúc còn tải disabled=${dis0} aria-disabled=${aria0}; bấm chờ ${waited} ms; sau khi tải: aria-selected=${sel}, nút «Bắt đầu thi thử» hiện=${ok}; tràn ngang ${ov}`);
  await ctx.close();
});
