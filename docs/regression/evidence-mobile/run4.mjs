// Pha 1 còn lại (web): NAV-11 (ngăn xếp Back), PRF-04 (Lighthouse mobile /checkin)
import { createRequire } from "node:module";
import { readFileSync, writeFileSync } from "node:fs";
const require = createRequire("/srv/work/TA_Agent_dev3/frontend/package.json");
const { chromium } = require("@playwright/test");
const BASE = "https://v2202609419695521098.happysrv.de";
const PWQ = readFileSync("/srv/work/.qc3-secret", "utf8").trim();
const IDS = JSON.parse(readFileSync(new URL("./vps-ids.json", import.meta.url), "utf8"));
const OUT = "/srv/work/.qc3-out"; const R = [];
const rec = (tc, kq, note) => { R.push({ tc, kq, note }); console.log(kq, tc, note.slice(0, 220)); };
const b = await chromium.launch({ args: ["--no-sandbox"] });
const ctx = await b.newContext({ viewport: { width: 390, height: 844 }, ignoreHTTPSErrors: true, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh", hasTouch: true, isMobile: true });
const p = await ctx.newPage();
await p.goto(BASE + "/login", { waitUntil: "domcontentloaded" });
await p.locator("input[type=email]").fill("sv.qc3@edupilot.local");
await p.locator("input[type=password]").fill(PWQ);
await p.keyboard.press("Enter");
await p.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 30000 }).catch(() => {});
const seen = [];
for (const [path, name] of [["/", "Hôm nay"], ["/calendar", "Lịch"], ["/checkin?course=" + IDS.QC3M1, "Điểm danh"]]) {
  await p.goto(BASE + path, { waitUntil: "domcontentloaded" }); await p.waitForTimeout(3500);
  seen.push({ path, h1: await p.locator("h1").first().innerText().catch(() => ""), url: p.url().replace(BASE, "") });
}
const backs = [];
for (let i = 0; i < 2; i++) { await p.goBack(); await p.waitForTimeout(1500); backs.push(p.url().replace(BASE, "")); }
await p.screenshot({ path: `${OUT}/shots/NAV-11-stack.png` });
const okStack = backs[0].startsWith("/calendar") && (backs[1] === "/" || backs[1] === "");
rec("TC-MOB-NAV-11", okStack ? "P" : `F:ngăn xếp Back=${JSON.stringify(backs)}`, `mở lần lượt ${seen.map((s) => s.path).join(" → ")}; Back 1=${backs[0]}, Back 2=${backs[1]}`);
await b.close();
writeFileSync(`${OUT}/results-run4.json`, JSON.stringify(R, null, 1));
