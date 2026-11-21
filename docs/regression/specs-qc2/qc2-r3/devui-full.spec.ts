import { test } from "@playwright/test";
import { mkdirSync } from "node:fs";
const OUT = "../docs/regression/evidence-3/gate-self"; mkdirSync(OUT, { recursive: true });
test.beforeEach(async ({}, info) => { test.skip(info.project.name !== "desktop", "1 lần"); });
test("dev-ui full + ui-v3 reference", async ({ page, browser }) => {
  await page.setViewportSize({ width: 1440, height: 900 }); await page.goto("/dev/ui"); await page.locator("main").first().waitFor(); await page.evaluate(() => document.fonts.ready); await page.waitForTimeout(800);
  const H = await page.evaluate(() => document.documentElement.scrollHeight); console.log("H", H);
  const n = Math.ceil(H / 1800);
  for (let i = 0; i < n; i++) { await page.evaluate((y) => window.scrollTo(0, y), i * 1800); await page.waitForTimeout(250); await page.screenshot({ path: `${OUT}/dev-ui-full-${i}.png`, clip: undefined }); }
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } }); const p2 = await ctx.newPage(); await p2.goto("file://" + process.cwd().replace(/frontend$/, "") + "docs/design/edupilot-ui-v3.html"); await p2.waitForTimeout(1500);
  await p2.screenshot({ path: `${OUT}/ui-v3-0.png` }); await p2.evaluate(() => window.scrollTo(0, 900)); await p2.waitForTimeout(300); await p2.screenshot({ path: `${OUT}/ui-v3-1.png`});
});
