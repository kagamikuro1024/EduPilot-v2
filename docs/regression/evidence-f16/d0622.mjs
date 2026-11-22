// D06-22 (sau DEF-R105S-06 f15746a4): /settings/llm ADMIN trên ep-r thật: chữ đỏ, tràn ngang, 40 điểm dừng Tab có viền focus; 1440 và 375
import { createRequire } from "node:module";
const require = createRequire(process.env.HOME + "/Documents/TA_Agent_qc2/frontend/package.json");
const { chromium } = require("@playwright/test");
const b = await chromium.launch(); const res = {};
for (const w of [1440, 375]) {
  const ctx = await b.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: w < 500 ? 812 : 900 }, locale: "vi-VN", hasTouch: w < 500, isMobile: w < 500 });
  const page = await ctx.newPage(); await page.addLocatorHandler(page.getByRole("button", { name: "Đã hiểu" }), async (x) => { await x.click(); });
  await page.goto("https://localhost:1143/login");
  await page.locator("input[type=email]").fill("admin@edupilot.local"); await page.locator("input[type=password]").fill("Edupilot#Seed-2026"); await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login")); await page.locator("[data-part=topbar]").waitFor();
  await page.goto("https://localhost:1143/settings/llm"); await page.waitForTimeout(6000);
  const ov = await page.evaluate(() => ({ sw: document.documentElement.scrollWidth, cw: document.documentElement.clientWidth }));
  const reds = await page.evaluate(() => { const r = {}; document.querySelectorAll("main *").forEach((e) => { const c = getComputedStyle(e).color; if (/rgb\(\s*(1[89]\d|2\d\d),\s*(\d|[1-5]\d),\s*(\d|[1-7]\d)\)/.test(c) && e.children.length === 0 && (e.textContent || "").trim()) { const k = (e.textContent || "").trim().slice(0, 30); r[k] = (r[k] || 0) + 1; } }); return r; });
  const ghost = await page.evaluate(() => Array.from(document.querySelectorAll("main button")).filter((e) => /Cài đặt nâng cao|Đổi hạn mức/.test(e.textContent || "")).map((e) => (e.textContent || "").trim().slice(0, 24) + ":" + (e.getAttribute("data-variant") || "?")));
  res[w] = { overflow: ov.sw > ov.cw + 1, sw: ov.sw, cw: ov.cw, reds, ghost };
  if (w === 1440) {
    let noRing = 0, n = 0; await page.keyboard.press("Tab");
    for (let i = 0; i < 40; i++) { const info = await page.evaluate(() => { const e = document.activeElement; if (!e) return null; const cs = getComputedStyle(e); return { ring: (cs.outlineStyle !== "none" && parseFloat(cs.outlineWidth) > 0) || /rgb/.test(cs.boxShadow) }; }); if (!info) break; n++; if (!info.ring) noRing++; await page.keyboard.press("Tab"); }
    res[w].tab = { stops: n, noRing };
  }
  await page.screenshot({ path: `d0622-${w}.png`, fullPage: true }); await ctx.close();
}
console.log(JSON.stringify(res, null, 1)); await b.close();
