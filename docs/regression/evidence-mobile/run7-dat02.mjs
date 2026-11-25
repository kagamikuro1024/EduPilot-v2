// DAT-02: sau purge, mở lại /checkin bằng mã cũ của cửa sổ đã purge + kiểm /me/attendance.
import { createRequire } from "node:module";
import { readFileSync, writeFileSync } from "node:fs";
const require = createRequire("/srv/work/TA_Agent_dev3/frontend/package.json");
const { chromium } = require("@playwright/test");
const BASE = "http://127.0.0.1:8488";
const PW = readFileSync("/opt/ep-dat/secrets/seed-password.txt", "utf8").trim();
const C = "01a213ae-d006-7bc4-ac26-30accd9488ca";
const OLDCODE = "XXP ART";
const R = []; const rec = (tc, kq, note) => { R.push({ tc, kq, note }); console.log(kq, tc, note.slice(0, 300)); };
const j = async (p, o = {}) => { const r = await fetch(BASE + "/api/v1" + p, { ...o, headers: { "content-type": "application/json", ...(o.headers || {}) } }); const t = await r.text(); let d = null; try { d = JSON.parse(t); } catch {} return { s: r.status, d, t }; };

const li = await j("/auth/login", { method: "POST", body: JSON.stringify({ email: "sv.gioi@edupilot.local", password: PW }) });
const tk = li.d.access_token;
const me = await j("/me/attendance", { headers: { authorization: "Bearer " + tk } });
rec("TC-MOB-DAT-02-api", me.d?.checked_in_at == null ? "P" : "F", `GET /me/attendance ${me.s}: checked_in_at=${JSON.stringify(me.d?.checked_in_at)} checked_in_at_course=${JSON.stringify(me.d?.course_id ?? me.d?.checked_in_course_id)}`);

const b = await chromium.launch({ args: ["--no-sandbox"] });
const ctx = await b.newContext({ viewport: { width: 390, height: 844 }, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh", hasTouch: true, isMobile: true });
const p = await ctx.newPage();
await p.goto(BASE + "/login", { waitUntil: "domcontentloaded" });
await p.locator("input[type=email]").fill("sv.gioi@edupilot.local");
await p.locator("input[type=password]").fill(PW);
await p.keyboard.press("Enter");
await p.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 30000 }).catch(() => {});
await p.goto(`${BASE}/checkin?course=${C}`, { waitUntil: "domcontentloaded" });
await p.locator("input").first().fill(OLDCODE);
await p.getByRole("button", { name: /Điểm danh/i }).first().click();
await p.waitForTimeout(4000);
const body = (await p.locator("body").innerText()).replace(/\s+/g, " ");
const ok = !/Đã điểm danh/.test(body);
rec("TC-MOB-DAT-02-ui", ok ? "P" : "F", `ma cu «${OLDCODE}» sau purge: ${body.slice(0, 220)}`);
await p.screenshot({ path: "/srv/work/.qc3-out/dat02-after.png" });
await b.close();
writeFileSync("/srv/work/.qc3-out/results-run7.json", JSON.stringify(R, null, 1));
