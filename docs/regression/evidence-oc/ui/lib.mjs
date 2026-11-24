import { createRequire } from "node:module";
const require = createRequire("/Users/kuro/Documents/TA_Agent_qc2/frontend/package.json");
export const { chromium } = require("@playwright/test");
export const BASE = "https://localhost:943", PW = "Edupilot#Seed-2026";
export async function newCtx(browser, user, w = 1440, h = 900) {
  const ctx = await browser.newContext({ viewport: { width: w, height: h }, ignoreHTTPSErrors: true, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh" });
  const p = await ctx.newPage();
  for (let t = 0; t < 3; t++) {
    await p.goto(`${BASE}/login`); await p.waitForLoadState("networkidle").catch(() => {});
    await p.locator("input[type=email]").fill(`${user}@edupilot.local`); await p.locator("input[type=password]").fill(PW); await p.keyboard.press("Enter");
    if (await p.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 30000 }).then(() => true).catch(() => false)) break;
    await p.waitForTimeout(8000);
  }
  await p.waitForTimeout(1500); return { ctx, page: p };
}
export const text = (p) => p.evaluate(() => document.body.innerText);
