import { test } from "@playwright/test";
import { sql, PW, OUT } from "./qc-real-lib";
test("probe", async ({ browser }) => {
  const ctx = await browser.newContext({ baseURL: "https://localhost:943", ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" });
  await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), sql(`select id from courses where class_code='QD9G'`)[0][0]);
  const page = await ctx.newPage(); await page.addLocatorHandler(page.getByRole("button", { name: "Đã hiểu" }), async (b) => { await b.click(); });
  await page.goto("/login"); await page.locator("input[type=email]").fill("teacher@edupilot.local"); await page.locator("input[type=password]").fill(PW); await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login")); await page.locator("[data-part=topbar]").waitFor();
  await page.goto("/grading?tab=assignments"); await page.waitForTimeout(3500);
  console.log("ROWS", JSON.stringify((await page.locator("[data-part=assignment-row]").allInnerTexts()).map((t) => t.replace(/\n/g, " ").slice(0, 50))));
  console.log("MAIN", (await page.locator("main").innerText()).replace(/\n/g, " | ").slice(0, 500));
  await page.screenshot({ path: `${OUT}/s106-probe.png` });
});
