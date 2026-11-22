import { createRequire } from "node:module";
const require = createRequire(process.env.HOME + "/Documents/TA_Agent_qc2/frontend/package.json");
const { chromium } = require("@playwright/test");
const b = await chromium.launch(); const out = {};
for (const u of ["admin", "teacher", "ta", "sv.gioi", "sv.kha", "sv.nguyco", "sv.moi"]) {
  const ctx = await b.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" });
  const page = await ctx.newPage(); await page.addLocatorHandler(page.getByRole("button", { name: "Đã hiểu" }), async (x) => { await x.click(); });
  await page.goto("https://localhost:1143/login"); await page.locator("input[type=email]").fill(`${u}@edupilot.local`); await page.locator("input[type=password]").fill("Edupilot#Seed-2026"); await page.keyboard.press("Enter");
  await page.waitForURL((x) => !x.pathname.startsWith("/login")); await page.locator("[data-part=topbar]").waitFor(); await page.locator("[data-part=sidebar] nav a").first().waitFor({ timeout: 30000 }); await page.waitForTimeout(2500);
  out[u] = (await page.locator("[data-part=sidebar] nav a").allInnerTexts()).map((t) => t.replace(/\s+/g, " ").trim()).filter(Boolean);
  const demo = await page.evaluate(() => ({ sw: document.querySelectorAll("[data-part=role-switcher],[data-part=demo-switcher]").length, ck: document.cookie.includes("ep_demo_") }));
  out[u + "_demo"] = demo; await ctx.close();
}
console.log(JSON.stringify(out, null, 1)); await b.close();
