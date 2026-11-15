// QC nhóm E: smoke UI trên stack ep-r (https://localhost:1143). Dùng: node ui-smoke.mjs <outdir> <user:path> ...
import { createRequire } from "node:module";
import { mkdirSync } from "node:fs";
const require = createRequire("/Users/kuro/Documents/TA_Agent_qc2/frontend/package.json");
const { chromium } = require("@playwright/test");
const BASE = process.env.QC_BASE ?? "https://localhost:1143", PW = process.env.QC_PW;
const out = process.argv[2]; mkdirSync(out, { recursive: true });
const jobs = process.argv.slice(3).map((a) => { const i = a.indexOf(":"); return [a.slice(0, i), a.slice(i + 1)]; });
const exe = process.env.PW_EXE;
const browser = await chromium.launch({ headless: true, ...(exe ? { executablePath: exe } : {}) });
const res = [];
for (const width of (process.env.WIDTHS ?? "1440,375").split(",").map(Number)) {
  const ctxs = {};
  for (const [user, path] of jobs) {
    if (!ctxs[user]) {
      const ctx = await browser.newContext({ viewport: { width, height: width === 375 ? 812 : 900 }, ignoreHTTPSErrors: true, locale: "vi-VN" });
      const p = await ctx.newPage();
      await p.goto(`${BASE}/login`); await p.getByLabel(/email/i).fill(`${user}@edupilot.local`); await p.getByLabel(/mật khẩu/i).first().fill(PW);
      await p.getByRole("button", { name: /đăng nhập/i }).click(); await p.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 20000 }).catch(() => {});
      ctxs[user] = ctx; await p.close();
    }
    const page = await ctxs[user].newPage(); const errs = []; const bad = [];
    page.on("console", (m) => m.type() === "error" && errs.push(m.text().slice(0, 100)));
    page.on("response", (r) => { if (r.status() >= 500) { bad.push(r.status()); console.error("5xx", r.status(), r.url().slice(-80)); } });
    let status = "ok", sw = 0, title = "";
    try {
      await page.goto(`${BASE}${path}`, { waitUntil: "domcontentloaded", timeout: 30000 });
      await page.waitForTimeout(3500);
      sw = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
      title = (await page.locator("h1").first().textContent({ timeout: 3000 }).catch(() => "")) ?? "";
      if (page.url().includes("/login")) status = "redirect-login";
      await page.screenshot({ path: `${out}/${user}${path.replace(/\W+/g, "_")}-${width}.png` });
    } catch (e) { status = "err " + String(e).slice(0, 80); }
    res.push({ user, path, width, status, overflow: sw, h1: title.trim().slice(0, 40), errors: errs.length, http5xx: bad.length });
    await page.close();
  }
  for (const c of Object.values(ctxs)) await c.close();
}
await browser.close();
console.log(JSON.stringify(res, null, 0).replace(/\},\{/g, "},\n{"));
