// DEF-R6-C1 nhánh «sửa»: sửa sự kiện lịch tại chỗ (tên + giờ) trên ep-r; desktop 1440 và mobile 375
import { chromium, newCtx, text, out, BASE } from "./ulib.mjs";
import { execFileSync } from "node:child_process";
const U = "01a1f03c-40fc-77f6-a505-c519802a3e52"; // QC3P8U, gv.s8f = TEACHER
const psql = (sql) => execFileSync("docker", ["exec", "-i", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-F|", "-c", sql], { encoding: "utf8" }).trim();
const API = BASE + "/api/v1";
const tok = async () => (await (await fetch(API + "/auth/login", { method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify({ email: "gv.s8f@edupilot.local", password: "Edupilot#Seed-2026" }) })).json()).access_token;
process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";
const stamp = String(Date.now()).slice(-6);
const mkEvent = async (title, iso) => { const r = await fetch(`${API}/courses/${U}/calendar/events`, { method: "POST", headers: { "content-type": "application/json", authorization: "Bearer " + (await tok()), "idempotency-key": "qc3c-" + Date.now() + Math.random() }, body: JSON.stringify({ type: "OTHER", title, starts_at: iso, ends_at: null, location: "P.QC3" }) }); const j = await r.json(); return { status: r.status, id: j.id ?? j.event?.id }; };
const b = await chromium.launch({ headless: true });
const run = async (tag, w, h, startIso) => {
  const t0 = `QC3 sửa ${tag} ${stamp}`, t1 = `${t0} (đã đổi)`;
  const ev = await mkEvent(t0, startIso); const id = ev.id;
  const row0 = psql(`select title,starts_at,version from calendar_events where id='${id}'`);
  const { ctx, page } = await newCtx(b, "gv.s8f", w, h); await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), U);
  const reqs = []; page.on("response", (r) => { const q = r.request(); if (/calendar\/events\//.test(r.url()) && ["PUT", "DELETE"].includes(q.method())) reqs.push(`${q.method()} ${new URL(r.url()).pathname.replace("/api/v1", "")} ${r.status()}`); });
  for (let i = 0; i < 4; i++) { await page.goto(`${BASE}/calendar?course=${U}`); await page.waitForTimeout(3500); if ((await page.getByText(t0).count()) > 0) break; await page.evaluate((c) => localStorage.setItem("ep:ui:course", c), U); }
  const seen = await page.getByText(t0).count();
  await page.getByRole("button", { name: `Thêm với ${t0}` }).first().click(); await page.waitForTimeout(400);
  const dlgOpen = await page.locator("dialog[open]").count();
  await page.getByRole("menuitem", { name: "Sửa" }).click(); await page.waitForTimeout(600);
  const f = page.locator("form").filter({ has: page.getByLabel("Tên sự kiện") });
  const old = await f.getByLabel("Bắt đầu").inputValue(); const oldTitle = await f.getByLabel("Tên sự kiện").inputValue();
  const d = new Date(old + ":00Z"); d.setUTCHours(d.getUTCHours() + 2); const nv = d.toISOString().slice(0, 16);
  await f.getByLabel("Tên sự kiện").fill(t1); await f.getByLabel("Bắt đầu").fill(nv);
  await page.screenshot({ path: `shots/edit-${tag}-form.png` });
  await f.getByRole("button", { name: "Lưu" }).click(); await page.waitForTimeout(2500);
  const err = (await text(page)).match(/(Không tìm thấy|không thành công|Lỗi[^\n]{0,60})/);
  const row1 = psql(`select title,starts_at,version from calendar_events where id='${id}'`);
  await page.reload(); await page.waitForTimeout(3500); await page.evaluate((c) => localStorage.setItem("ep:ui:course", c), U);
  const after = (await page.getByText(t1).count()) > 0, gone = (await page.getByText(t0, { exact: true }).count()) === 0;
  await page.screenshot({ path: `shots/edit-${tag}-after.png` });
  const put = reqs.filter((x) => x.startsWith("PUT")), idOk = put.length === 1 && put[0].includes(`/events/${id} `) && !put[0].includes("calendar_event");
  const dbOk = row1.startsWith(t1 + "|") && row1 !== row0 && row1.split("|")[2] === String(Number(row0.split("|")[2]) + 1);
  const ok = ev.status < 300 && seen > 0 && dlgOpen === 0 && put.length === 1 && put[0].endsWith(" 200") && idOk && dbOk && !err && after && gone;
  out(`TC-P8-03-11b-${tag}`, ok, `${w}px: tạo qua API ${ev.status}; thấy trên lịch=${seen > 0}; menu «Thêm với…»→Sửa (modal=${dlgOpen}); sửa tên «${oldTitle}»→«${t1}» và giờ ${old}→${nv} (UTC+7 trong ô); yêu cầu ${JSON.stringify(reqs)} (uuid thô=${idOk}); DB trước [${row0}] → sau [${row1}] (đổi=${dbOk}); thông báo lỗi=${err ? err[0] : "không"}; sau tải lại thấy tên mới=${after}, tên cũ biến mất=${gone}; sự kiện ${id}`);
  await ctx.close();
};
try { await run("d1440", 1440, 900, "2026-11-26T03:00:00Z"); await run("m375", 375, 812, "2026-11-27T03:00:00Z"); } finally { await b.close(); }
