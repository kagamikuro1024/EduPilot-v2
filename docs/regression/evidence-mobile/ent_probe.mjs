// ENT-06…10: một buổi "probe" trong QC3M1, đổi giờ theo từng nhánh, kiểm cờ API + hiển thị UI (Chromium).
import { createRequire } from "node:module";
import { readFileSync, writeFileSync } from "node:fs";
const require = createRequire("/srv/work/TA_Agent_dev3/frontend/package.json");
const { chromium } = require("@playwright/test");
const BASE = "https://v2202609419695521098.happysrv.de";
const API = BASE + "/api/v1";
const PWQ = readFileSync("/srv/work/.qc3-secret", "utf8").trim();
const PWT = readFileSync("/srv/work/.qc3-teacher", "utf8").trim();
const IDS = JSON.parse(readFileSync(new URL("./vps-ids.json", import.meta.url), "utf8"));
const PROBE = JSON.parse(readFileSync(new URL("./vps-probe.json", import.meta.url), "utf8"));
const OUT = "/srv/work/.qc3-out";
const R = [];
const rec = (tc, kq, note) => { R.push({ tc, kq, note }); console.log(kq, tc, note.slice(0, 200)); };
const iso = (d) => d.toISOString().replace(/\.\d+Z$/, "Z");
async function loginApi(email, pw) {
  const r = await fetch(API + "/auth/login", { method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify({ email, password: pw }) });
  return (await r.json()).access_token;
}
const tTok = await loginApi("teacher@edupilot.local", PWT);
async function retime(so, eo, room) {
  const now = Date.now();
  const r = await fetch(`${API}/courses/${IDS.QC3M1}/sessions/${PROBE.sid}`, { method: "PUT", headers: { "content-type": "application/json", authorization: "Bearer " + tTok }, body: JSON.stringify({ starts_at: iso(new Date(now + so * 60000)), ends_at: iso(new Date(now + eo * 60000)), room, topic: null, version: PROBE.version }) });
  const body = await r.json().catch(() => null);
  if (body && body.version) PROBE.version = body.version;
  return r.status;
}
const sTok = await loginApi("sv.qc3@edupilot.local", PWQ);
async function flag(courseId) {
  const r = await fetch(`${API}/me/today`, { headers: { authorization: "Bearer " + sTok } });
  const d = await r.json();
  const rows = (d.timeline || []).filter((t) => t.course.class_code === "QC3M1");
  return rows.map((t) => ({ place: t.place, state: t.state, link: t.checkin_link }));
}
const b = await chromium.launch({ args: ["--no-sandbox"] });
const ctx = await b.newContext({ viewport: { width: 390, height: 844 }, ignoreHTTPSErrors: true, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh", hasTouch: true, isMobile: true });
await ctx.addInitScript((c) => { try { localStorage.setItem("ep:ui:course", c); } catch {} }, IDS.QC3M1);
const p = await ctx.newPage();
await p.goto(BASE + "/login", { waitUntil: "domcontentloaded" });
await p.locator("input[type=email]").fill("sv.qc3@edupilot.local");
await p.locator("input[type=password]").fill(PWQ);
await p.keyboard.press("Enter");
await p.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 30000 }).catch(() => {});
const ui = async () => {
  await p.goto(BASE + "/?course=" + IDS.QC3M1, { waitUntil: "domcontentloaded" }); await p.waitForTimeout(5000);
  return p.evaluate(() => [...document.querySelectorAll("main li")].filter((li) => li.innerText.includes("QC3E-")).map((li) => ({ row: li.innerText.replace(/\s+/g, " ").slice(0, 80), link: [...li.querySelectorAll("a")].some((a) => (a.getAttribute("href") || "").includes("/checkin")) })));
};
const cases = [
  ["TC-MOB-ENT-06", 40, 70, "QC3E-A_xa40", false, "bắt đầu sau ~40 phút → không liên kết"],
  ["TC-MOB-ENT-07", 8, 38, "QC3E-B_gan8", true, "trong 15 phút trước giờ → có liên kết"],
  ["TC-MOB-ENT-08", -20, 40, "QC3E-C_dang", true, "đang diễn ra → có liên kết"],
  ["TC-MOB-ENT-09", -40, -5, "QC3E-D_vuaHet", false, "vừa hết giờ → không liên kết"],
];
for (const [tc, so, eo, room, want, why] of cases) {
  const st = await retime(so, eo, room);
  const f = await flag(IDS.QC3M1);
  const apiLink = f.length ? f[0].link : null;
  const rows = await ui();
  const row = rows.find((r) => r.row.includes(room));
  const uiLink = row ? row.link : null;
  await p.screenshot({ path: `${OUT}/shots/${tc}.png` });
  rec(tc, apiLink === want && uiLink === want ? "P" : `F:${why} (API=${apiLink} UI=${uiLink})`, `PUT ${st}; API checkin_link=${apiLink}; UI hàng=${row ? row.row : "không thấy"} liên kết=${uiLink}; mong đợi=${want}`);
}
// ENT-10: lớp QC3M2 self_checkin=false (buổi E_tat vẫn ở timeline)
{
  const r = await fetch(`${API}/me/today`, { headers: { authorization: "Bearer " + sTok } });
  const d = await r.json();
  const e = (d.timeline || []).filter((t) => t.course.class_code === "QC3M2");
  await p.goto(BASE + "/?course=" + IDS.QC3M2, { waitUntil: "domcontentloaded" }); await p.waitForTimeout(5000);
  const row = await p.evaluate(() => [...document.querySelectorAll("main li")].filter((li) => li.innerText.includes("QC3E-E_tat")).map((li) => ({ row: li.innerText.replace(/\s+/g, " ").slice(0, 80), link: [...li.querySelectorAll("a")].some((a) => (a.getAttribute("href") || "").includes("/checkin")) })));
  await p.screenshot({ path: `${OUT}/shots/TC-MOB-ENT-10.png` });
  const apiLink = e.length ? e[0].checkin_link : null;
  const uiLink = row.length ? row[0].link : null;
  rec("TC-MOB-ENT-10", apiLink === false && uiLink === false ? "P" : `F:lớp tắt vẫn có liên kết (API=${apiLink} UI=${uiLink})`, `QC3M2 self_checkin=false: API=${apiLink}, UI hàng=${row.length ? row[0].row : "không thấy"} liên kết=${uiLink}`);
}
await b.close();
writeFileSync(`${OUT}/results-ent-probe.json`, JSON.stringify(R, null, 1));
