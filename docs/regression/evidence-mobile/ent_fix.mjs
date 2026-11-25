// ENT-05 (không lộ trạng thái cửa sổ) + ENT-18 (aria-current) — chạy riêng cho gọn.
import { createRequire } from "node:module";
import { readFileSync, writeFileSync } from "node:fs";
const require = createRequire("/srv/work/TA_Agent_dev3/frontend/package.json");
const { chromium } = require("@playwright/test");
const BASE = "https://v2202609419695521098.happysrv.de"; const API = BASE + "/api/v1";
const PWQ = readFileSync("/srv/work/.qc3-secret", "utf8").trim();
const PWT = readFileSync("/srv/work/.qc3-teacher", "utf8").trim();
const IDS = JSON.parse(readFileSync(new URL("./vps-ids.json", import.meta.url), "utf8"));
const OUT = "/srv/work/.qc3-out"; const R = [];
const rec = (tc, kq, note) => { R.push({ tc, kq, note }); console.log(kq, tc, note.slice(0, 200)); };
const t = (await (await fetch(API + "/auth/login", { method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify({ email: "teacher@edupilot.local", password: PWT }) })).json()).access_token;
const H = { "content-type": "application/json", authorization: "Bearer " + t };
// thêm buổi token (buổi cũ đã đủ 3 vòng)
const vn = new Date(Date.now() + 7 * 3600e3).toISOString().slice(0, 10);
const wd = new Date(Date.now() + 7 * 3600e3).getUTCDay() || 7;
await fetch(`${API}/courses/${IDS.QC3M1}/sessions/generate`, { method: "POST", headers: { ...H, "Idempotency-Key": "qc3ent5-g-" + Date.now() }, body: JSON.stringify({ weekdays: [wd], start_time: "08:00", end_time: "08:30", room: "QC3E-ent5", from: vn, to: vn }) });
await new Promise((r) => setTimeout(r, 700));
const list = await (await fetch(`${API}/courses/${IDS.QC3M1}/sessions?limit=100`, { headers: H })).json();
const sess = (list.items || []).find((x) => (x.room || "") === "QC3E-ent5");
await fetch(`${API}/courses/${IDS.QC3M1}/sessions/${sess.id}`, { method: "PUT", headers: H, body: JSON.stringify({ starts_at: new Date(Date.now() - 5 * 60000).toISOString(), ends_at: new Date(Date.now() + 25 * 60000).toISOString(), room: "QC3E-ent5", topic: null, version: sess.version }) });
const b = await chromium.launch({ args: ["--no-sandbox"] });
// 2) ENT-18 trước (đo thanh bên 1024)
{
  const ctx = await b.newContext({ viewport: { width: 1024, height: 768 }, ignoreHTTPSErrors: true, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh" });
  await ctx.addInitScript(() => { try { localStorage.setItem("ep:ui:course", "01a21372-6d00-77b3-9c58-d3b3ea86311c"); } catch {} });
  const p = await ctx.newPage();
  await p.goto(BASE + "/login", { waitUntil: "domcontentloaded" });
  await p.locator("input[type=email]").fill("sv.qc3@edupilot.local");
  await p.locator("input[type=password]").fill(PWQ);
  await p.keyboard.press("Enter");
  await p.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 30000 }).catch(() => {});
  await p.goto(BASE + "/checkin?course=" + IDS.QC3M1, { waitUntil: "domcontentloaded" }); await p.waitForTimeout(5000);
  const info = await p.evaluate(() => [...document.querySelectorAll("nav a")].filter((a) => (a.getAttribute("href") || "") === "/checkin").map((a) => ({ cur: a.getAttribute("aria-current"), label: (a.innerText || a.getAttribute("aria-label") || "").trim() })));
  await p.screenshot({ path: `${OUT}/shots/ENT-18-checkin-active.png` });
  rec("TC-MOB-ENT-18", info.length && info[0].cur === "page" ? "P" : `F:${JSON.stringify(info)}`, `mục nav /checkin khi ở /checkin: ${JSON.stringify(info)}`);
  await ctx.close();
}
// 1) ENT-05
{
  const ctx = await b.newContext({ viewport: { width: 390, height: 844 }, ignoreHTTPSErrors: true, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh", hasTouch: true, isMobile: true });
  await ctx.addInitScript(() => { try { localStorage.setItem("ep:ui:course", "01a21372-6d00-77b3-9c58-d3b3ea86311c"); } catch {} });
  const p = await ctx.newPage();
  await p.goto(BASE + "/login", { waitUntil: "domcontentloaded" });
  await p.locator("input[type=email]").fill("sv.qc3@edupilot.local");
  await p.locator("input[type=password]").fill(PWQ);
  await p.keyboard.press("Enter");
  await p.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 30000 }).catch(() => {});
  await p.goto(BASE + "/?course=" + IDS.QC3M1, { waitUntil: "domcontentloaded" }); await p.waitForTimeout(5000);
  const snap = async () => {
    const a = await p.evaluate(() => ({ nav: document.querySelector("nav")?.outerHTML.replace(/\s+/g, " ") || "", body: document.body.innerText.replace(/\s+/g, " ") }));
    const more = p.getByRole("button", { name: /Thêm/ }).first();
    if (await more.count()) { await more.click(); await p.waitForTimeout(500); }
    const menu = await p.evaluate(() => [...document.querySelectorAll("[role=menu],dialog[open]")].map((e) => e.outerHTML.replace(/\s+/g, " ")).join(""));
    await p.keyboard.press("Escape");
    return { ...a, menu };
  };
  const before = await snap();
  const w = await (await fetch(`${API}/courses/${IDS.QC3M1}/sessions/${sess.id}/checkin-windows`, { method: "POST", headers: { ...H, "Idempotency-Key": "qc3ent5-w-" + Date.now() }, body: JSON.stringify({ minutes: 10 }) })).json();
  await p.reload({ waitUntil: "domcontentloaded" }); await p.waitForTimeout(5000);
  const after = await snap();
  await p.screenshot({ path: `${OUT}/shots/ENT-05-after-open.png` });
  const changed = before.nav !== after.nav || before.menu !== after.menu || before.body !== after.body;
  const leaked = /điểm danh (đang )?mở|đang mở điểm danh/i.test(after.body);
  rec("TC-MOB-ENT-05", w.id && !changed && !leaked ? "P" : `F:cửa sổ=${Boolean(w.id)} đổi=${changed} lộ=${leaked}`, `mở cửa sổ=${w.id ? "ok" : JSON.stringify(w).slice(0, 80)}; DOM nav/menu/body đổi=${changed}; chữ lộ=${leaked}`);
  if (w.id) await fetch(`${API}/courses/${IDS.QC3M1}/sessions/${sess.id}/checkin-windows/${w.id}/close`, { method: "POST", headers: { ...H, "Idempotency-Key": "qc3ent5c-" + Date.now() }, body: "{}" });
  await ctx.close();
}
await b.close();
writeFileSync(`${OUT}/results-ent-fix.json`, JSON.stringify(R, null, 1));
