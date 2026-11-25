// Pha 1 phần web còn lại: THM-06, NAV-02, NAV-03, PRM-10, PRF-07
import { createRequire } from "node:module";
import { readFileSync, writeFileSync } from "node:fs";
const require = createRequire("/srv/work/TA_Agent_dev3/frontend/package.json");
const { chromium } = require("@playwright/test");
const { default: AxeBuilder } = require("@axe-core/playwright");
const BASE = "https://v2202609419695521098.happysrv.de"; const API = BASE + "/api/v1";
const PWQ = readFileSync("/srv/work/.qc3-secret", "utf8").trim();
const PWT = readFileSync("/srv/work/.qc3-teacher", "utf8").trim();
const IDS = JSON.parse(readFileSync(new URL("./vps-ids.json", import.meta.url), "utf8"));
const OUT = "/srv/work/.qc3-out"; const R = [];
const rec = (tc, kq, note) => { R.push({ tc, kq, note }); console.log(kq, tc, note.slice(0, 220)); };
const tk = async (e, p) => (await (await fetch(API + "/auth/login", { method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify({ email: e, password: p }) })).json()).access_token;
const t = await tk("teacher@edupilot.local", PWT); const H = { "content-type": "application/json", authorization: "Bearer " + t };
const vn = new Date(Date.now() + 7 * 3600e3).toISOString().slice(0, 10); const wd = new Date(Date.now() + 7 * 3600e3).getUTCDay() || 7;
async function freshWindow(room) {
  await fetch(`${API}/courses/${IDS.QC3M1}/sessions/generate`, { method: "POST", headers: { ...H, "Idempotency-Key": "qc3r2-g-" + room + Date.now() }, body: JSON.stringify({ weekdays: [wd], start_time: "09:00", end_time: "09:30", room, from: vn, to: vn }) });
  await new Promise((r) => setTimeout(r, 700));
  const list = await (await fetch(`${API}/courses/${IDS.QC3M1}/sessions?limit=100`, { headers: H })).json();
  const sess = (list.items || []).find((x) => (x.room || "") === room);
  await fetch(`${API}/courses/${IDS.QC3M1}/sessions/${sess.id}`, { method: "PUT", headers: H, body: JSON.stringify({ starts_at: new Date(Date.now() - 5 * 60000).toISOString(), ends_at: new Date(Date.now() + 25 * 60000).toISOString(), room, topic: null, version: sess.version }) });
  const w = await (await fetch(`${API}/courses/${IDS.QC3M1}/sessions/${sess.id}/checkin-windows`, { method: "POST", headers: { ...H, "Idempotency-Key": "qc3r2-w-" + room + Date.now() }, body: JSON.stringify({ minutes: 10 }) })).json();
  const disp = await (await fetch(`${API}/courses/${IDS.QC3M1}/sessions/${sess.id}/checkin-windows/${w.id}/display`, { headers: H })).json();
  return { wid: w.id, sid: sess.id, url: disp.qr_url };
}
const b = await chromium.launch({ args: ["--no-sandbox"] });
const mk = async (mw) => {
  const ctx = await b.newContext({ viewport: { width: 390, height: 844 }, ignoreHTTPSErrors: true, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh", hasTouch: true, isMobile: true, colorScheme: mw });
  const p = await ctx.newPage();
  return { ctx, p };
};
const login = async (p, user = "sv.qc3@edupilot.local") => {
  await p.goto(BASE + "/login", { waitUntil: "domcontentloaded" });
  await p.locator("input[type=email]").fill(user);
  await p.locator("input[type=password]").fill(PWQ);
  await p.keyboard.press("Enter");
  await p.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 30000 }).catch(() => {});
  await p.waitForTimeout(1200);
};
// THM-06: axe chế độ tối + nền vẫn sáng
{
  const { ctx, p } = await mk("dark"); await login(p);
  await p.goto(BASE + "/checkin?course=" + IDS.QC3M1, { waitUntil: "domcontentloaded" }); await p.waitForTimeout(4000);
  const info = await p.evaluate(() => ({ bg: getComputedStyle(document.body).backgroundColor, scheme: getComputedStyle(document.documentElement).colorScheme }));
  const res = await new AxeBuilder({ page: p }).withTags(["wcag2aa"]).analyze();
  const contrast = res.violations.filter((v) => v.id === "color-contrast");
  await p.screenshot({ path: `${OUT}/shots/THM-06-dark.png` });
  rec("TC-MOB-THM-06", contrast.length === 0 ? "P" : `F:contrast ${contrast[0].nodes.length} nút`, `colorScheme=dark: nền body=${info.bg}, color-scheme=${info.scheme}; axe contrast=${contrast.length === 0 ? "0 vi phạm" : contrast[0].nodes.length + " nút"}; tổng vi phạm aa=${res.violations.length}`);
  await ctx.close();
}
// PRM-10: lớp không geofence → không hộp quyền, không dòng vị trí
{
  const { ctx, p } = await mk("light"); await login(p); const reqs = [];
  p.on("request", (q) => reqs.push(q.url()));
  await p.goto(BASE + "/checkin?course=" + IDS.QC3M1, { waitUntil: "domcontentloaded" }); await p.waitForTimeout(4000);
  const body = await p.evaluate(() => document.body.innerText);
  const geoReq = reqs.filter((u) => /geolocation|position/i.test(u));
  rec("TC-MOB-PRM-10", !/kiểm tra vị trí/.test(body) && geoReq.length === 0 ? "P" : `F:dòng vị trí=${/kiểm tra vị trí/.test(body)} request=${geoReq.length}`, `QC3M1 geo_check=false: dòng «Lớp này kiểm tra vị trí…»=${/kiểm tra vị trí/.test(body)}; request định vị=${geoReq.length}`);
  await ctx.close();
}
// NAV-02: token flow → Back: không POST lần hai, không lỗi cũ
{
  const w = await freshWindow("QC3E-nav2");
  const { ctx, p } = await mk("light"); await login(p);
  const posts = []; p.on("request", (q) => { if (q.method() === "POST" && /\/checkins$/.test(q.url())) posts.push(q.url()); });
  await p.goto(w.url, { waitUntil: "domcontentloaded" }); await p.waitForTimeout(6000);
  const done = /Đã điểm danh/.test(await p.evaluate(() => document.body.innerText));
  await p.goBack(); await p.waitForTimeout(2000);
  const back = await p.evaluate(() => ({ url: location.pathname + location.search + location.hash, body: document.body.innerText.replace(/\s+/g, " ").slice(0, 120) }));
  await p.goForward(); await p.waitForTimeout(2500);
  const again = await p.evaluate(() => document.body.innerText.replace(/\s+/g, " ").slice(0, 120));
  await p.screenshot({ path: `${OUT}/shots/NAV-02-back.png` });
  const stale = /Mã không đúng|Mã đã hết hạn/.test(again);
  rec("TC-MOB-NAV-02", done && posts.length === 1 && !stale ? "P" : `F:done=${done} POST=${posts.length} lỗi cũ=${stale}`, `sau «Đã điểm danh»: Back → "${back.body.slice(0, 60)}"; Forward → "${again.slice(0, 60)}"; POST=${posts.length}; lỗi cũ=${stale}`);
  await fetch(`${API}/courses/${IDS.QC3M1}/sessions/${w.sid}/checkin-windows/${w.wid}/close`, { method: "POST", headers: { ...H, "Idempotency-Key": "qc3r2c-" + Date.now() }, body: "{}" });
  await ctx.close();
}
// NAV-03: chưa đăng nhập mở liên kết token → login → tự gửi 1 lần; Back không vòng về /login
{
  const w = await freshWindow("QC3E-nav3");
  const { ctx, p } = await mk("light");
  await p.goto(BASE + "/login", { waitUntil: "domcontentloaded" }); await p.waitForTimeout(500);
  const posts = []; p.on("request", (q) => { if (q.method() === "POST" && /\/checkins$/.test(q.url())) posts.push(q.url()); });
  await p.goto(w.url, { waitUntil: "domcontentloaded" });
  await p.waitForTimeout(2500);
  const loginShown = /\/login/.test(p.url());
  if (loginShown) {
    await p.locator("input[type=email]").fill("sv.qc3@edupilot.local");
    await p.locator("input[type=password]").fill(PWQ);
    await p.keyboard.press("Enter");
    await p.waitForTimeout(6000);
  }
  const body = await p.evaluate(() => document.body.innerText.replace(/\s+/g, " ").slice(0, 120));
  const done = /Đã điểm danh/.test(body);
  await p.goBack(); await p.waitForTimeout(2000);
  const backUrl = p.url(); await p.screenshot({ path: `${OUT}/shots/NAV-03-login-back.png` });
  rec("TC-MOB-NAV-03", loginShown && done && posts.length === 1 && !/\/login/.test(backUrl) ? "P" : `F:login=${loginShown} done=${done} POST=${posts.length} backUrl=${backUrl}`, `mở liên kết khi chưa đăng nhập: → /login=${loginShown}; tự gửi 1 lần=${posts.length === 1}; «Đã điểm danh»=${done}; sau Back URL=${backUrl.replace(BASE, "")}`);
  await fetch(`${API}/courses/${IDS.QC3M1}/sessions/${w.sid}/checkin-windows/${w.wid}/close`, { method: "POST", headers: { ...H, "Idempotency-Key": "qc3r2c3-" + Date.now() }, body: "{}" });
  await ctx.close();
}
// PRF-07: từ lúc phản hồi POST tới khi thấy «Đã điểm danh»
{
  const w = await freshWindow("QC3E-prf7");
  const { ctx, p } = await mk("light"); await login(p);
  let respAt = 0, postAt = 0;
  p.on("request", (q) => { if (q.method() === "POST" && /\/checkins$/.test(q.url())) postAt = Date.now(); });
  p.on("response", (r) => { if (r.request().method() === "POST" && /\/checkins$/.test(r.url())) respAt = Date.now(); });
  await p.goto(w.url, { waitUntil: "domcontentloaded" });
  await p.waitForFunction(() => /Đã điểm danh/.test(document.body.innerText), { timeout: 15000 }).catch(() => {});
  const seenAt = Date.now();
  rec("TC-MOB-PRF-07", respAt && seenAt - respAt < 2000 ? "P" : `F:chênh ${seenAt - respAt} ms`, `POST→phản hồi ${respAt - postAt} ms; phản hồi→«Đã điểm danh» ${seenAt - respAt} ms (ngưỡng QC đề xuất ≤ 2000 ms trên VPS)`);
  await fetch(`${API}/courses/${IDS.QC3M1}/sessions/${w.sid}/checkin-windows/${w.wid}/close`, { method: "POST", headers: { ...H, "Idempotency-Key": "qc3r2c7-" + Date.now() }, body: "{}" });
  await ctx.close();
}
await b.close();
writeFileSync(`${OUT}/results-run2.json`, JSON.stringify(R, null, 1));
