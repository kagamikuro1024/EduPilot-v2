import { createRequire } from "node:module";
import { readFileSync } from "node:fs";
const require = createRequire("/srv/work/TA_Agent_dev3/frontend/package.json");
const { chromium } = require("@playwright/test");
const BASE = "https://v2202609419695521098.happysrv.de";
const PW = readFileSync("/srv/work/.qc3-secret", "utf8").trim();
const IDS = JSON.parse(readFileSync(new URL("./vps-ids.json", import.meta.url), "utf8"));
const b = await chromium.launch({ args: ["--no-sandbox"] });
const ctx = await b.newContext({ viewport: { width: 390, height: 844 }, ignoreHTTPSErrors: true, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh", hasTouch: true, isMobile: true });
await ctx.addInitScript((c) => { try { localStorage.setItem("ep:ui:course", c); } catch {} }, IDS.QC3M1);
const p = await ctx.newPage();
await p.goto(BASE + "/login", { waitUntil: "domcontentloaded" });
await p.locator("input[type=email]").fill("sv.nguyco@edupilot.local");
await p.locator("input[type=password]").fill(PW);
await p.keyboard.press("Enter");
await p.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 30000 }).catch(() => {});
await p.goto(BASE + "/?course=" + IDS.QC3M1, { waitUntil: "domcontentloaded" }); await p.waitForTimeout(7000);
console.log(JSON.stringify(await p.evaluate(() => ({
  anchors: [...document.querySelectorAll("a")].filter((a) => (a.getAttribute("href") || "").includes("/checkin")).map((a) => ({ t: (a.innerText || "").trim(), h: a.getAttribute("href"), li: (a.closest("li")?.innerText || "").replace(/\s+/g, " ").slice(0, 80) })),
  rows: [...document.querySelectorAll("main li")].map((li) => li.innerText.replace(/\s+/g, " ").slice(0, 90)).slice(0, 12),
})), null, 1));
await p.screenshot({ path: "/srv/work/.qc3-out/shots/dbg3-today.png" });
await b.close();
