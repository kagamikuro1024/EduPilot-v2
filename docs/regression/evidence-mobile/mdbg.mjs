import { createRequire } from "node:module";
import { readFileSync } from "node:fs";
const require = createRequire("/srv/work/TA_Agent_dev3/frontend/package.json");
const { chromium } = require("@playwright/test");
const BASE = "https://v2202609419695521098.happysrv.de";
const PW = readFileSync("/srv/work/.qc3-secret", "utf8").trim();
const IDS = JSON.parse(readFileSync(new URL("./vps-ids.json", import.meta.url), "utf8"));
const b = await chromium.launch({ args: ["--no-sandbox"] });
async function run(w, h, user, course, mobile) {
  const ctx = await b.newContext({ viewport: { width: w, height: h }, ignoreHTTPSErrors: true, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh", hasTouch: true, isMobile: mobile });
  await ctx.addInitScript((c) => { try { localStorage.setItem("ep:ui:course", c); } catch {} }, course);
  const p = await ctx.newPage();
  await p.goto(BASE + "/login", { waitUntil: "domcontentloaded" });
  await p.locator("input[type=email]").fill(`${user}@edupilot.local`);
  await p.locator("input[type=password]").fill(PW);
  await p.keyboard.press("Enter");
  await p.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 30000 }).catch(() => {});
  await p.goto(BASE + "/?course=" + course); await p.waitForTimeout(6000);
  const d = await p.evaluate(() => ({
    url: location.pathname + location.search,
    navs: [...document.querySelectorAll("nav")].map((n) => ({ aria: n.getAttribute("aria-label"), vis: n.getBoundingClientRect().width > 0, n: n.querySelectorAll("a").length, links: [...n.querySelectorAll("a")].map((a) => a.innerText.trim()) })),
    checkinLinks: [...document.querySelectorAll("a")].filter((a) => (a.getAttribute("href") || "").includes("/checkin")).map((a) => ({ href: a.getAttribute("href"), text: a.innerText.trim() })),
    body: document.body.innerText.replace(/\s+/g, " ").slice(0, 700),
  }));
  console.log(JSON.stringify({ w, h, user, ...d }, null, 1));
  await p.screenshot({ path: `/srv/work/.qc3-out/shots/dbg-${w}-${user}.png` });
  await ctx.close();
}
await run(1024, 768, "sv.gioi", IDS.QC3M1, false);
await run(390, 844, "sv.nguyco", IDS.QC3M1, true);
await b.close();
