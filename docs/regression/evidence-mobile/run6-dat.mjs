// DAT-01 (US-AS-14 AC1): lớp 761988, buổi demo mới mở tay, sv.gioi điểm danh trên UI mobile (390×844).
// So sánh trước/sau: 270 attendance_records của 761987 và sha256 seed/expected_crm.csv.
import { createRequire } from "node:module";
import { readFileSync, writeFileSync } from "node:fs";
import { randomUUID } from "node:crypto";
const require = createRequire("/srv/work/TA_Agent_dev3/frontend/package.json");
const { chromium } = require("@playwright/test");
const BASE = "http://127.0.0.1:8488";
const PW = readFileSync("/opt/ep-dat/secrets/seed-password.txt", "utf8").trim();
const C761988 = "01a213ae-d006-7bc4-ac26-30accd9488ca";
const SESSION = "01a213ae-d2a1-7539-83a5-20c8bdefc17a"; // 2026-11-26 06:30–09:00Z (buổi mới, chưa có vòng)
const R = [];
const rec = (tc, kq, note) => { R.push({ tc, kq, note }); console.log(kq, tc, note.slice(0, 300)); };

async function api(path, opt = {}, tk) {
  const r = await fetch(BASE + "/api/v1" + path, {
    ...opt,
    headers: { "content-type": "application/json", ...(tk ? { authorization: "Bearer " + tk } : {}), ...(opt.headers || {}) },
  });
  const t = await r.text(); let j = null; try { j = JSON.parse(t); } catch {}
  return { status: r.status, j, t };
}

const tl = await api("/auth/login", { method: "POST", body: JSON.stringify({ email: "teacher@edupilot.local", password: PW }) });
if (tl.status !== 200) { console.log("login teacher", tl.status, tl.t.slice(0, 200)); process.exit(1); }
const tt = tl.j.access_token;

let ow = await api(`/courses/${C761988}/sessions/${SESSION}/checkin-windows`, {
  method: "POST", headers: { "Idempotency-Key": randomUUID() }, body: JSON.stringify({ minutes: 10 }),
}, tt);
console.log("open window:", ow.status, JSON.stringify(ow.j).slice(0, 300));
if (ow.status !== 201 && ow.status !== 409) process.exit(1);
let wid = ow.j?.id;
if (ow.status === 409 && ow.j?.details?.window_id) wid = ow.j.details.window_id;
const wl = await api(`/courses/${C761988}/sessions/${SESSION}/checkin-windows`, {}, tt);
const open = (wl.j?.items || []).find((x) => !x.closed_at && !x.closedAt);
wid = wid || open?.id;
console.log("window id", wid || "(none)");

const b = await chromium.launch({ args: ["--no-sandbox"] });
const ctx = await b.newContext({ viewport: { width: 390, height: 844 }, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh", hasTouch: true, isMobile: true });
const p = await ctx.newPage();
await p.goto(BASE + "/login", { waitUntil: "domcontentloaded" });
await p.locator("input[type=email]").fill("sv.gioi@edupilot.local");
await p.locator("input[type=password]").fill(PW);
await p.keyboard.press("Enter");
await p.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 30000 }).catch(() => {});
console.log("after login:", new URL(p.url()).pathname);

let ok = false, note = "";
for (let a = 0; a < 3 && !ok; a++) {
  const d = await api(`/courses/${C761988}/sessions/${SESSION}/checkin-windows/${wid}/display`, {}, tt);
  const code = d.j?.short_code || d.j?.shortCode;
  console.log("display", d.status, code);
  if (!code) break;
  await p.goto(`${BASE}/checkin?course=${C761988}`, { waitUntil: "domcontentloaded" });
  const inp = p.locator("input").first();
  await inp.fill(code);
  await p.getByRole("button", { name: /Điểm danh/i }).first().click();
  await p.waitForTimeout(4000);
  const body = await p.locator("body").innerText();
  note = body.replace(/\s+/g, " ").slice(0, 200);
  ok = /Đã điểm danh/.test(body);
  await p.screenshot({ path: "/srv/work/.qc3-out/dat01-after.png" });
}
rec("TC-MOB-DAT-01-ui", ok ? "P" : "F", `ui 390: ${note}`);
await b.close();
writeFileSync("/srv/work/.qc3-out/results-run6.json", JSON.stringify(R, null, 1));
