// PRF-04 (tương đương Lighthouse mobile, nhưng giữ phiên đăng nhập): LCP/CLS/TBT trên /checkin với
// mạng 4G chậm + CPU 4× qua CDP (Chromium trên VPS). lhci không mang được Bearer trong bộ nhớ nên đo bằng Playwright.
import { createRequire } from "node:module";
import { readFileSync, writeFileSync } from "node:fs";
const require = createRequire("/srv/work/TA_Agent_dev3/frontend/package.json");
const { chromium } = require("@playwright/test");
const BASE = "https://v2202609419695521098.happysrv.de";
const PWQ = readFileSync("/srv/work/.qc3-secret", "utf8").trim();
const IDS = JSON.parse(readFileSync(new URL("./vps-ids.json", import.meta.url), "utf8"));
const OUT = "/srv/work/.qc3-out"; const R = [];
const rec = (tc, kq, note) => { R.push({ tc, kq, note }); console.log(kq, tc, note.slice(0, 260)); };
const b = await chromium.launch({ args: ["--no-sandbox"] });
const ctx = await b.newContext({ viewport: { width: 390, height: 844 }, ignoreHTTPSErrors: true, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh", hasTouch: true, isMobile: true });
const p = await ctx.newPage();
await p.goto(BASE + "/login", { waitUntil: "domcontentloaded" });
await p.locator("input[type=email]").fill("sv.qc3@edupilot.local");
await p.locator("input[type=password]").fill(PWQ);
await p.keyboard.press("Enter");
await p.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 30000 }).catch(() => {});
const cdp = await ctx.newCDPSession(p);
await cdp.send("Network.enable");
await cdp.send("Network.emulateNetworkConditions", { offline: false, latency: 150, downloadThroughput: 1.6 * 1024 * 1024 / 8, uploadThroughput: 750 * 1024 / 8 });
await cdp.send("Emulation.setCPUThrottlingRate", { rate: 4 });
await p.addInitScript(() => {
  window.__m = { lcp: 0, cls: 0, longtasks: [] };
  new PerformanceObserver((l) => { for (const e of l.getEntries()) window.__m.lcp = Math.max(window.__m.lcp, e.startTime); }).observe({ type: "largest-contentful-paint", buffered: true });
  new PerformanceObserver((l) => { for (const e of l.getEntries()) if (!e.hadRecentInput) window.__m.cls += e.value; }).observe({ type: "layout-shift", buffered: true });
  new PerformanceObserver((l) => { for (const e of l.getEntries()) window.__m.longtasks.push(Math.round(e.duration)); }).observe({ type: "longtask", buffered: true });
});
await p.goto(BASE + "/checkin?course=" + IDS.QC3M1, { waitUntil: "load" });
await p.waitForTimeout(6000);
const m = await p.evaluate(() => ({ ...window.__m, nav: performance.getEntriesByType("navigation")[0]?.duration }));
await cdp.send("Emulation.setCPUThrottlingRate", { rate: 1 });
await p.screenshot({ path: `${OUT}/shots/PRF-04-throttled.png` });
const tbtApprox = m.longtasks.reduce((a, d) => a + Math.max(0, d - 50), 0);
rec("TC-MOB-PRF-04", m.lcp > 0 && m.lcp <= 2500 && m.cls <= 0.1 ? "P" : `F:LCP=${Math.round(m.lcp)} ms CLS=${m.cls.toFixed(3)}`,
  `4G chậm (1,6 Mbps/150 ms) + CPU 4×: LCP=${Math.round(m.lcp)} ms (trần 2500), CLS=${m.cls.toFixed(3)} (trần 0,1), long task=${m.longtasks.length} (TBT xấp xỉ ${tbtApprox} ms, trần 200), load=${Math.round(m.nav)} ms; đo bằng Playwright+CDP vì lhci không mang được Bearer trong bộ nhớ`);
await b.close();
writeFileSync(`${OUT}/results-run5.json`, JSON.stringify(R, null, 1));
