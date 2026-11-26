import { test, type Browser } from "@playwright/test";
import { PW, sql } from "./qc-real-lib";
const OUT = process.env.QC_OUT ?? "/tmp/x";
test.skip(!process.env.QC_REAL, "cần stack thật");
test("P12-10 UI sau Rút lưu ý", async ({ browser }) => {
  const cid = sql(`select id from courses where class_code='QD9G'`)[0][0]; const a = sql(`select id from assignments where title like '%P12-10 thật%' order by created_at desc limit 1`)[0][0];
  const ctx = await browser.newContext({ baseURL: "https://localhost:943", ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" });
  await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), cid);
  const page = await ctx.newPage(); await page.addLocatorHandler(page.getByRole("button", { name: "Đã hiểu" }), async (b) => { await b.click(); });
  await page.goto("/login"); await page.locator("input[type=email]").fill("teacher@edupilot.local"); await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter"); await page.waitForURL((u) => !u.pathname.startsWith("/login")); await page.locator("[data-part=topbar]").waitFor();
  await page.goto(`/grading?tab=assignments&open=${a}`); await page.waitForTimeout(5000);
  const p = page.locator("[data-part=guidance-panel]"); console.log("PANEL", (await p.innerText()).replace(/\n+/g, " ¦ ").slice(0, 900));
  await page.reload(); await page.waitForTimeout(5000); console.log("PANEL2", (await p.innerText()).replace(/\n+/g, " ¦ ").slice(0, 1200));
  await p.scrollIntoViewIfNeeded(); await p.screenshot({ path: `${OUT}/p1210-panel.png` });
});
