import { createRequire } from "node:module";
import { readFileSync } from "node:fs";
const require = createRequire("/srv/work/TA_Agent_dev3/frontend/package.json");
const { chromium } = require("@playwright/test");
const BASE = "https://v2202609419695521098.happysrv.de";
const PW = readFileSync("/srv/work/.qc3-secret", "utf8").trim();
const IDS = JSON.parse(readFileSync(new URL("./vps-ids.json", import.meta.url), "utf8"));
const b = await chromium.launch({ args: ["--no-sandbox"] });
async function mk(w, h, user, course, mobile) {
  const ctx = await b.newContext({ viewport: { width: w, height: h }, ignoreHTTPSErrors: true, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh", hasTouch: true, isMobile: mobile });
  await ctx.addInitScript((c) => { try { localStorage.setItem("ep:ui:course", c); } catch {} }, course);
  const p = await ctx.newPage();
  await p.goto(BASE + "/login", { waitUntil: "domcontentloaded" });
  await p.locator("input[type=email]").fill(`${user}@edupilot.local`);
  await p.locator("input[type=password]").fill(PW);
  await p.keyboard.press("Enter");
  await p.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 30000 }).catch(() => {});
  return { ctx, p };
}
// A) 1024: nav
{
  const { ctx, p } = await mk(1024, 768, "sv.gioi", IDS.QC3M1, false);
  await p.goto(BASE + "/?course=" + IDS.QC3M1, { waitUntil: "domcontentloaded" }); await p.waitForTimeout(7000);
  console.log("A1024", JSON.stringify(await p.evaluate(() => ({ url: location.pathname, navs: [...document.querySelectorAll("nav")].map((n) => ({ aria: n.getAttribute("aria-label"), w: Math.round(n.getBoundingClientRect().width), links: [...n.querySelectorAll("a")].map((a) => ({ t: (a.innerText || "").trim(), h: a.getAttribute("href"), box: (() => { const b = a.getBoundingClientRect(); return [Math.round(b.width), Math.round(b.height)]; })() })).slice(0, 14) })) })), null, 1));
  await p.screenshot({ path: "/srv/work/.qc3-out/shots/dbg2-1024.png" });
  await ctx.close();
}
// B) 390: timeline links + nav boxes
{
  const { ctx, p } = await mk(390, 844, "sv.nguyco", IDS.QC3M1, true);
  await p.goto(BASE + "/?course=" + IDS.QC3M1, { waitUntil: "domcontentloaded" }); await p.waitForTimeout(7000);
  console.log("B390", JSON.stringify(await p.evaluate(() => ({
    links: [...document.querySelectorAll("a")].filter((a) => (a.getAttribute("href") || "").includes("/checkin")).map((a) => ({ t: (a.innerText || "").trim(), h: a.getAttribute("href") })),
    navBoxes: [...document.querySelectorAll("nav a, nav button")].map((e) => { const b = e.getBoundingClientRect(); return { t: (e.innerText || "").trim(), w: Math.round(b.width), h: Math.round(b.height) }; }),
    body: document.body.innerText.replace(/\s+/g, " ").slice(0, 500),
  })), null, 1));
  await p.screenshot({ path: "/srv/work/.qc3-out/shots/dbg2-390.png" });
  // C) /checkin QC3M2: nhập mã sai
  await p.goto(BASE + "/checkin?course=" + IDS.QC3M2, { waitUntil: "domcontentloaded" }); await p.waitForTimeout(6000);
  const form = await p.evaluate(() => ({ inputs: [...document.querySelectorAll("input")].map((i) => ({ t: i.type, ph: i.placeholder, id: i.id })), btns: [...document.querySelectorAll("button")].map((b) => (b.innerText || "").trim()).filter(Boolean), body: document.body.innerText.replace(/\s+/g, " ").slice(0, 400) }));
  console.log("C-form", JSON.stringify(form, null, 1));
  const inp = p.locator("input").first();
  if (await inp.count()) { await inp.fill("ABC123"); await p.waitForTimeout(200); }
  const btn = p.getByRole("button", { name: "Điểm danh" }).first();
  if (await btn.count()) { await btn.click(); await p.waitForTimeout(4000); } else console.log("C no-btn");
  console.log("C-after", JSON.stringify(await p.evaluate(() => ({ body: document.body.innerText.replace(/\s+/g, " ").slice(0, 500), reqs: performance.getEntriesByType("resource").filter((r) => r.name.includes("checkins")).map((r) => r.name) })), null, 1));
  await p.screenshot({ path: "/srv/work/.qc3-out/shots/dbg2-checkin-m2.png" });
  await ctx.close();
}
await b.close();
