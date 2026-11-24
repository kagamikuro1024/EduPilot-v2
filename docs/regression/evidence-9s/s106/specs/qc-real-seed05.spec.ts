import { test, type Browser } from "@playwright/test";
import { PW, sql } from "./qc-real-lib";
const OUT = "/Users/kuro/Documents/TA_Agent_qc9s/docs/regression/evidence-9s/shots/p7-05-11";
test.skip(!process.env.QC_REAL, "cần stack thật");
test.use({ actionTimeout: 25_000 });
async function pageAs(browser: Browser, user: string, w: number) {
  const cid = sql(`select id from courses where class_code='761987'`)[0][0];
  const ctx = await browser.newContext({ baseURL: "https://localhost:4743", ignoreHTTPSErrors: true, viewport: { width: w, height: 900 }, locale: "vi-VN" });
  await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), cid);
  const page = await ctx.newPage(); await page.addLocatorHandler(page.getByRole("button", { name: "Đã hiểu" }), async (b) => { await b.click(); });
  await page.goto("/login"); await page.locator("input[type=email]").fill(`${user}@edupilot.local`); await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter"); await page.waitForURL((u) => !u.pathname.startsWith("/login")); await page.locator("[data-part=topbar]").waitFor(); return { ctx, page };
}
for (const user of ["admin", "teacher"]) for (const w of [1440, 390]) {
  test(`P7-05-11 ${user} @${w}`, async ({ browser }) => {
    const { ctx, page } = await pageAs(browser, user, w); await page.goto("/settings/integrations"); await page.waitForTimeout(5000);
    const txt = (await page.locator("main").innerText()).replace(/\n+/g, " ¦ ");
    const over = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
    const heads = await page.locator("main h2, main h3, main legend, main [data-part*=section] > :first-child").allInnerTexts();
    const panels = await page.locator("main [data-part=panel], main [class*=Panel], main [class*=card]").count();
    const rules = await page.evaluate(() => [...document.querySelectorAll("main hr, main [class*=divider], main [class*=rule]")].length);
    console.log(`T ${user}@${w} overflowX=${over} panels=${panels} rules=${rules} heads=${JSON.stringify(heads).slice(0, 200)}`);
    console.log(`TXT ${user}@${w}`, txt.slice(0, 1700));
    console.log(`BTN ${user}@${w}`, (await page.locator("main").getByRole("button").allInnerTexts()).join(" | ").slice(0, 400));
    await page.screenshot({ path: `${OUT}/${user}-${w}.png`, fullPage: true }); await ctx.close();
  });
}
