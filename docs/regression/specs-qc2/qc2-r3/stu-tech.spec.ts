import { test } from "@playwright/test";
import { asDemo } from "../../../../frontend/e2e/support/session";
test.beforeEach(async ({}, info) => { test.skip(info.project.name !== "desktop", "1 lần"); });
test("SV: /me lỗi — «Chi tiết kỹ thuật» chứa gì", async ({ page, context }) => {
  await page.clock.setFixedTime(new Date("2026-10-29T09:20:00+07:00")); await asDemo(context, "student", { expIn: 366 * 86400 });
  await page.goto("/me"); await page.locator("main").first().waitFor(); await page.waitForTimeout(1500);
  const d = page.locator("main details").first(); const n = await page.locator("main details").count(); let txt = "(không có <details>)";
  if (n) { await d.locator("summary").click(); txt = (await d.innerText()).replace(/\s+/g, " "); }
  console.log("TECH n=" + n + " :: " + txt);
});
