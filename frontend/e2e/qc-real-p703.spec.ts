import { test, type Browser } from "@playwright/test";
import { PW, sql } from "./qc-real-lib";
const OUT = process.env.QC_OUT ?? "/tmp/x";
test.skip(!process.env.QC_REAL, "cần stack thật");
test("P7-03-02 trạng thái phía sinh viên", async ({ browser }) => {
  const cid = sql(`select id from courses where class_code='QD9G'`)[0][0]; const a = sql(`select id from assignments where title like '%P12-10 thật%' order by created_at desc limit 1`)[0][0];
  for (const [user, tag] of [["sv09", "dang-cham"], ["sv.nguyco", "da-co-diem"]]) {
    const ctx = await browser.newContext({ baseURL: "https://localhost:943", ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" });
    await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), cid);
    const page = await ctx.newPage(); await page.addLocatorHandler(page.getByRole("button", { name: "Đã hiểu" }), async (b) => { await b.click(); });
    await page.goto("/login"); await page.locator("input[type=email]").fill(`${user}@edupilot.local`); await page.locator("input[type=password]").fill(PW);
    await page.keyboard.press("Enter"); await page.waitForURL((u) => !u.pathname.startsWith("/login")); await page.locator("[data-part=topbar]").waitFor();
    await page.goto("/assignments"); await page.waitForTimeout(3500);
    const list = (await page.locator("main").innerText()).replace(/\n+/g, " ¦ "); const mine = list.match(/QC106 P12-10[^¦]*¦[^¦]*¦[^¦]*/)?.[0] ?? "(không thấy)";
    await page.goto(`/assignments/${a}`); await page.waitForTimeout(3500);
    const det = (await page.locator("main").innerText()).replace(/\n+/g, " ¦ ");
    console.log(`P703 ${user} LIST «${mine}» ¦ DETAIL «${det.slice(0, 400)}»`); await page.screenshot({ path: `${OUT}/p703-${tag}.png`, fullPage: true }); await ctx.close();
  }
});
