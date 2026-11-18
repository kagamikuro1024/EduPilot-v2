import { test, type Page, type BrowserContext } from "@playwright/test";
import { go, login, FX, shot, log, psql, newUser, expect } from "./lib";
import { execFileSync } from "node:child_process";
const R = (tc: string, ok: boolean, note: string) => log("p503", `${ok ? "PASS" : "FAIL"} ${tc} ${note}`);
const I = (tc: string, note: string) => log("p503", `INFO ${tc} ${note}`);
const FX5 = FX("r4fx") as { X: string; Y: string; S: string[] };
const FM = FX("fx_me") as { M: string; M2: string; U: string[] }; const FA = FX("fx_att") as { C: string; CE: string; CF: string };
const rds = (...a: string[]) => execFileSync("docker", ["exec", "ep-r-redis-1", "redis-cli", ...a], { encoding: "utf8" }).trim();
const uid = (u: string) => psql(`select id from users where email='${u}@edupilot.local'`);
const X = FX5.X, Y = FX5.Y; const [s0, s1, a20, s3, s4, s5, s6, a28, s8, s9] = FX5.S;

/** đăng nhập thật rồi bật đồng hồ giả trên trang mới; trả mảng nhịp (yêu cầu thật tới máy chủ) */
async function hbPage(browser: any, user: string, path: string, vp = { width: 1280, height: 800 }) {
  const { ctx, page } = await newUser(browser, user, vp); const beats: Array<{ t: number; area: string; tab: string; status?: number }> = [];
  page.on("request", (r) => { if (r.url().includes("/activity/heartbeat") && r.method() === "POST") { const b = JSON.parse(r.postData() ?? "{}"); beats.push({ t: Date.now(), area: b.area, tab: b.tab_id }); } });
  page.on("response", (r) => { if (r.url().includes("/activity/heartbeat") && r.request().method() === "POST") { const x = beats.find((b) => b.status === undefined); if (x) x.status = r.status(); } });
  await page.addInitScript(() => { Math.random = () => 0; }); await page.clock.install({ time: new Date() });
  await page.goto(path); await page.waitForTimeout(2500); return { ctx, page, beats };
}
const hide = (page: Page, h: boolean) => page.evaluate((hh) => { Object.defineProperty(document, "visibilityState", { configurable: true, get: () => (hh ? "hidden" : "visible") }); document.dispatchEvent(new Event("visibilitychange")); }, h);

test("07 tab ẩn", async ({ browser }) => {
  const { ctx, page, beats } = await hbPage(browser, a28, `/chat?course=${X}`); await page.clock.runFor(1000); await page.waitForTimeout(800); const n1 = beats.length;
  await hide(page, true); await page.clock.runFor(150_000); await page.waitForTimeout(800); const n2 = beats.length;
  await hide(page, false); const t0 = Date.now(); await page.keyboard.press("Shift"); await page.clock.runFor(1000); await expect.poll(() => beats.length).toBeGreaterThan(n2); const dt = Date.now() - t0;
  R("TC-P5-03-07", n1 >= 1 && n2 === n1 && beats.length > n2, `/chat (đồng hồ giả): nhịp đầu ${n1}; ẩn tab (visibilityState=hidden) 150 s giả lập: số nhịp ${n1} → ${n2} (không nhịp nào); hiện lại + thao tác: nhịp gửi tiếp sau ${dt} ms thật (≤ 1 s giả lập), tổng ${beats.length}; trạng thái trả về ${JSON.stringify(beats.map((b) => b.status))}`);
  await ctx.close();
});
test("08 idle 2 phút", async ({ browser }) => {
  const { ctx, page, beats } = await hbPage(browser, s8, `/threads?course=${X}`); await page.clock.runFor(130_000); await page.waitForTimeout(1500); const atIdle = beats.length; await page.clock.runFor(180_000); await page.waitForTimeout(1500); const after = beats.length;
  await page.mouse.click(300, 300); await page.clock.runFor(1000); await page.waitForTimeout(1500); const resumed = beats.length;
  R("TC-P5-03-08", atIdle >= 4 && atIdle <= 5 && after === atIdle && resumed === atIdle + 1, `thao tác lúc t0 rồi dừng; chạy 130 s giả lập: ${atIdle} nhịp (nhịp cuối trước mốc idle 120 s), thêm 180 s: ${after} (ngừng); chạm lại → ${resumed} (tiếp tục)`);
  await ctx.close();
});
test("09 dọn khi đổi trang", async ({ browser }) => {
  const { ctx, page, beats } = await hbPage(browser, s9, `/chat?course=${X}`); await page.clock.runFor(1000); await page.waitForTimeout(800);
  const nav = async (href: string) => { await page.locator("[data-part=sidebar]").locator(`a[href^="${href}"]`).first().click(); await page.keyboard.press("Shift"); await page.waitForTimeout(500); };
  await nav("/library"); await page.clock.runFor(31_000); await page.waitForTimeout(1000); const lib = beats.at(-1)?.area; const nLib = beats.length;
  await nav("/"); await page.clock.runFor(100_000); await page.waitForTimeout(1500); const nHome = beats.length;
  R("TC-P5-03-09", beats[0]?.area === "CHAT" && lib === "LIBRARY" && nHome === nLib, `/chat → nhịp ${beats[0]?.area}; sang /library → nhịp kế area=${lib} (khu vực mới); sang / (không đo) rồi chạy 100 s giả lập: nhịp ${nLib} → ${nHome} (không nhịp thừa, bộ hẹn giờ đã dọn); chuỗi area=${JSON.stringify(beats.map((b) => b.area))}`);
  await ctx.close();
});
test("10 mạng đứt, 403/422", async ({ browser }) => {
  const { ctx, page, beats } = await hbPage(browser, s6, `/chat?course=${X}`); await page.clock.runFor(1000); await page.waitForTimeout(800); const base = beats.length;
  await ctx.setOffline(true); await page.keyboard.press("Shift"); await page.clock.runFor(95_000); await page.waitForTimeout(1500); const off = beats.length; await ctx.setOffline(false); await page.keyboard.press("Shift"); await page.clock.runFor(1_000); await page.waitForTimeout(1500); const on = beats.length;
  // 403 / 422
  let mode = 403; await page.route(/\/activity\/heartbeat/, (r) => r.fulfill({ status: mode, contentType: "application/json", body: JSON.stringify({ code: mode === 403 ? "FORBIDDEN" : "VALIDATION_FAILED", message: "x" }) }));
  await page.keyboard.press("Shift"); await page.clock.runFor(31_000); await page.waitForTimeout(1000); const n403 = beats.length; const t403 = await page.locator("main").innerText(); mode = 422; await page.keyboard.press("Shift"); await page.clock.runFor(31_000); await page.waitForTimeout(1000); const n422 = beats.length;
  await page.clock.runFor(31_000); await page.waitForTimeout(800); const n3 = beats.length; const alert = await page.locator("[role=alert],[data-ep-toast],[class*=toast]").count(); const dlg = await page.getByRole("dialog").count();
  R("TC-P5-03-10", off - base <= 4 && on - off <= 2 && n403 - on === 1 && n422 - n403 === 1 && n3 - n422 === 1 && alert === 0 && dlg === 0, `setOffline 95 s giả lập: số yêu cầu ${base} → ${off} (mỗi chu kỳ 30 s một lần, không dồn hàng đợi); online + thao tác: ${off} → ${on} (không gửi bù); máy chủ trả 403: +${n403 - on} nhịp/chu kỳ, 422: +${n422 - n403}, sau đó +${n3 - n422}; không xếp hàng lại, không báo người dùng (alert/toast=${alert}, hộp thoại=${dlg})`);
  await ctx.close();
});
test("13 chỉ khu vực + trang không đo", async ({ browser }) => {
  const { ctx, page, beats } = await hbPage(browser, s5, `/`); await page.clock.runFor(100_000); const pg: string[] = [];
  for (const u of [`/?course=${X}`, `/me?course=${X}`, `/exams?course=${X}`, `/settings`]) { await go(page, u).catch(() => undefined); await page.keyboard.press("Shift"); await page.clock.runFor(100_000); await page.waitForTimeout(800); pg.push(`${u.split("?")[0]}:${beats.length}`); }
  const stuBeats = beats.length;
  const t = await hbPage(browser, "qc3t", `/gradebook?course=${X}`); for (const u of [`/students?course=${X}`, `/class/settings?course=${X}`, `/calendar?course=${X}`, `/chat?course=${X}`, `/threads?course=${X}`]) { await go(t.page, u).catch(() => undefined); await t.page.keyboard.press("Shift"); await t.page.clock.runFor(100_000); await t.page.waitForTimeout(800); }
  const areas = psql(`select string_agg(distinct area,',' order by area) from study_time_daily`); const cols = psql(`select string_agg(column_name,',' order by ordinal_position) from information_schema.columns where table_name='study_time_daily'`);
  const rk = rds("keys", "ep:study:*").split("\n").filter(Boolean).filter((k) => !k.includes(":total:") && !k.includes(":dirty") && !k.includes(":flushed")); const bad = rk.filter((k) => !/^ep:study:[0-9a-f-]{36}:[0-9a-f-]{36}:\d{8}:(CHAT|THREADS|PRACTICE|LIBRARY|CALENDAR)$/.test(k));
  R("TC-P5-03-13", stuBeats === 0 && t.beats.length === 0 && bad.length === 0 && /^(CALENDAR|CHAT|LIBRARY|PRACTICE|THREADS)(,(CALENDAR|CHAT|LIBRARY|PRACTICE|THREADS))*$/.test(areas), `SV duyệt /, /me, /exams, /settings (mỗi trang 100 s giả lập, ${pg.join(" ")}): ${stuBeats} nhịp; GV (qc3t) duyệt /gradebook, /students, /class/settings, /calendar, /chat, /threads: ${t.beats.length} nhịp; study_time_daily.area ∈ {${areas}}; cột bảng: ${cols} (không nội dung / đường dẫn / tiêu đề / tên); khoá Redis ep:study:* sai khuôn=${bad.length}/${rk.length}`);
  await ctx.close(); await t.ctx.close();
});
test("18/20/21 hộp thoại lần đầu, chưa ack, staff", async ({ browser }) => {
  psql(`update users set tracking_notice_ack_at=null where email='${a20}@edupilot.local'`); rds("DEL", `ep:trk:${uid(a20)}`);
  const { ctx, page, beats } = await hbPage(browser, a20, `/chat?course=${X}`); await page.clock.runFor(100_000); await page.waitForTimeout(1500); const before = beats.length;
  const dlg = page.getByRole("dialog"); await dlg.waitFor({ timeout: 15000 }); const dt = (await dlg.innerText()).replace(/\n+/g, " | ");
  await shot(page, "p503-18");
  await dlg.getByRole("button", { name: "Đã hiểu" }).click(); await page.waitForTimeout(1500); const gone = (await page.getByRole("dialog").count()) === 0; const ack = psql(`select tracking_notice_ack_at is not null from users where email='${a20}@edupilot.local'`);
  await page.reload(); await page.waitForTimeout(3000); const again = await page.getByRole("dialog").count();
  R("TC-P5-03-18", /Chat/.test(dt) && /Luyện đề/.test(dt) && /Thư viện/.test(dt) && /Lịch/.test(dt) && /Không ghi nội dung/.test(dt) && /Giảng viên xem được/.test(dt) && /không dùng để tính điểm/.test(dt) && /Bạn có thể tắt ở Cài đặt\./.test(dt) && gone && ack === "t" && again === 0, `SV chưa ack: hộp thoại «${dt}»; bấm «Đã hiểu» → hộp đóng=${gone}, users.tracking_notice_ack_at đặt=${ack}; tải lại: hộp thoại ${again} (chỉ một lần); (TC gốc ghi «Threads»: chữ hiển thị là «Hỏi đáp công khai» theo luật chữ UI)`);
  R("TC-P5-03-20", before === 0, `UI chưa ack: 100 s giả lập trên /chat trong khi hộp thoại hiện → hook gửi ${before} nhịp (khớp phần API: curl nhịp → 204, không cộng)`);
  await ctx.close();
  for (const u of ["qc3t", "qc3ta", "admin"]) { const x = await hbPage(browser, u, u === "admin" ? `/` : `/chat?course=${X}`); await x.page.clock.runFor(100_000); await x.page.waitForTimeout(1200); const d = await x.page.getByRole("dialog").count(); R(`TC-P5-03-21-${u}`, d === 0 && x.beats.length === 0, `${u}: hộp thoại minh bạch=${d}, nhịp gửi=${x.beats.length}`); await x.ctx.close(); }
});
test("41/37 class settings + form Mở lớp", async ({ browser }) => {
  // chưa có sinh viên (CE)
  const t = await newUser(browser, "qc3t"); await go(t.page, `/class/settings?course=${FA.CE}`); const sw = t.page.getByRole("switch", { name: "Cho sinh viên tắt đo thời gian học" }); await sw.waitFor({ timeout: 20000 });
  const en0 = await sw.isEnabled(); const c0 = await sw.isChecked(); await sw.click(); await t.page.waitForTimeout(1500); const db1 = psql(`select allow_study_time_opt_out from courses where id='${FA.CE}'`); const c1 = await sw.isChecked(); await sw.click(); await t.page.waitForTimeout(1500); const db2 = psql(`select allow_study_time_opt_out from courses where id='${FA.CE}'`);
  // đã có SV (M2: allow=false)
  await go(t.page, `/class/settings?course=${FM.M2}`); const sw2 = t.page.getByRole("switch", { name: "Cho sinh viên tắt đo thời gian học" }); await sw2.waitFor(); const dis2 = await sw2.isDisabled(); const desc2 = await sw2.evaluate((e) => { const id = e.getAttribute("aria-describedby"); return id ? document.getElementById(id)?.innerText ?? "" : (e.getAttribute("title") ?? ""); }); const p2 = await t.page.locator("main p", { hasText: desc2 || "§§§" }).count();
  await go(t.page, `/class/settings?course=${FM.M}`); const sw3 = t.page.getByRole("switch", { name: "Cho sinh viên tắt đo thời gian học" }); await sw3.waitFor(); const dis3 = await sw3.isDisabled(); const ch3 = await sw3.isChecked(); await shot(t.page, "p503-41");
  const ta = await newUser(browser, "qc3ta"); await go(ta.page, `/class/settings?course=${FA.CE}`); const swT = ta.page.getByRole("switch", { name: "Cho sinh viên tắt đo thời gian học" }); await swT.waitFor(); const disT = await swT.isDisabled();
  const g = await ta.page.evaluate(async (c) => { const r = await fetch(`/api/v1/courses/${c}`, { headers: { authorization: "Bearer " + ((window as any).__tok ?? "") } }); return r.status; }, FA.CE).catch(() => 0);
  const q = psql(`select allow_study_time_opt_out from courses where id='${FM.M2}'`);
  R("TC-P5-03-41", en0 && db1 !== db2 && db2 === "t" && dis2 && !!desc2 && p2 === 0 && dis3 && disT, `GV, lớp CE chưa có SV: công tắc đổi được=${en0} (bật=${c0}); bấm → DB ${db1}, bấm lại → ${db2}; lớp M2 đã có SV: công tắc disabled=${dis2}, tooltip lý do ở chính công tắc «${desc2}», dòng mô tả riêng=${p2}; lớp M đã có SV: disabled=${dis3}, giá trị bật=${ch3}; TA lớp CE: công tắc disabled=${disT} (chỉ đọc); GET /courses/{id} bằng Member trả cờ: courses.allow_study_time_opt_out(M2)=${q} (đọc qua API ở TC-P5-03-36/43)`);
  const adm = await newUser(browser, "admin"); await go(adm.page, `/admin/courses`); const open = adm.page.getByRole("button", { name: /Mở lớp/ }).first(); await open.waitFor({ timeout: 20000 }); await open.click(); await adm.page.waitForTimeout(1200);
  const form = (await adm.page.locator("[role=dialog],form").last().innerText()).replace(/\n+/g, " | "); await shot(adm.page, "p503-37"); const sws = await adm.page.getByRole("switch").count();
  R("TC-P5-03-37-ui", !/tắt đo|thời gian học|allow_study/i.test(form) && sws === 0, `form «Mở lớp» (admin): ${form.slice(0, 300)}; công tắc=${sws}; trường «tắt đo / thời gian học»: ${/tắt đo|thời gian học/i.test(form)}`);
  await t.ctx.close(); await ta.ctx.close(); await adm.ctx.close();
});
test("P5-06 22/23/24/25 settings", async ({ browser }) => {
  const u = await newUser(browser, "qc3me01"); await go(u.page, `/settings`); await u.page.waitForTimeout(1500);
  const grp = u.page.locator("section,div").filter({ has: u.page.getByText("Thời gian học", { exact: true }) }).last(); const gt = (await grp.innerText()).replace(/\n+/g, " | ");
  const swX = u.page.getByRole("switch", { name: /QC3M5/ }), swY = u.page.getByRole("switch", { name: /QC3M6/ });
  const nX = await swX.count(), nY = await swY.count(); const dX = await swX.isEnabled(), cX = await swX.isChecked(), dY = await swY.isDisabled(), cY = await swY.isChecked();
  const descY = await swY.evaluate((e) => { const id = e.getAttribute("aria-describedby"); return id ? document.getElementById(id)?.innerText ?? "" : (e.getAttribute("title") ?? ""); }); const toastsBefore = await u.page.locator("[data-ep-toast],[class*=toast]").count();
  const label = await u.page.getByText("Ghi thời gian học").count();
  const ps: any[] = []; u.page.on("request", (r) => { if (r.method() === "PUT" && r.url().endsWith("/me/settings")) ps.push(JSON.parse(r.postData() ?? "{}")); });
  const t0 = Date.now(); await swX.click(); await expect.poll(async () => await swX.isChecked()).toBe(false); const lat = Date.now() - t0; await u.page.waitForTimeout(1500); const toasts = await u.page.locator("[data-ep-toast],[class*=toast]").count();
  await shot(u.page, "p506-22");
  R("TC-P5-06-22", nX === 1 && nY === 1 && dX && cX && dY && cY && !!descY && toasts === 0 && ps.length === 1, `/settings của qc3me01 (ghi danh M lớp cho tắt, M2 không cho tắt): nhóm «${gt.slice(0, 260)}»; công tắc theo lớp M=${nX} (đổi được=${dX}, bật=${cX}), M2=${nY} (disabled=${dY}, bật=${cY}); tooltip lý do ở chính công tắc M2 «${descY}»; nhãn «Ghi thời gian học» chữ riêng=${label} (tên công tắc dạng «mã lớp · tên»); bấm tắt M: đổi sau ${lat} ms, PUT ${JSON.stringify(ps)}; toast=${toasts}`);
  // 23
  const st = psql(`select track_study_time from user_settings where user_id='${uid("qc3me01")}'`);
  const api = await u.page.evaluate(async ([m, m2]) => { const get = async (c: string) => { const r = await fetch(`/api/v1/courses/${c}/me/study-time?weeks=8`, { credentials: "include" }); return r.status; }; return [await get(m), await get(m2)]; }, [FM.M, FM.M2]).catch(() => []);
  await go(u.page, `/me?course=${FM.M}`); await u.page.waitForTimeout(1500); const meM = (await u.page.locator("main").innerText()); await go(u.page, `/me?course=${FM.M2}`); await u.page.waitForTimeout(1500); const meM2 = (await u.page.locator("main").innerText());
  // 409: ngữ cảnh khác đổi trước
  const o = await newUser(browser, "qc3me01"); await go(o.page, `/settings`); await o.page.waitForTimeout(1000);
  await go(u.page, `/settings`); await u.page.waitForTimeout(1000);
  await o.page.getByRole("switch", { name: /QC3M5/ }).click(); await o.page.waitForTimeout(1500);
  await u.page.getByRole("switch", { name: /QC3M5/ }).click(); await u.page.waitForTimeout(2000); const notice = (await u.page.locator("main").innerText()).match(/[^\n]*(đã đổi|cập nhật|thay đổi|Thử lại|lại)[^\n]*/g) ?? []; const stateU = await u.page.getByRole("switch", { name: /QC3M5/ }).isChecked(); const dbv = psql(`select track_study_time from user_settings where user_id='${uid("qc3me01")}'`);
  R("TC-P5-06-23", st === "f" && /Đang tắt/.test(meM) && /Tuần này: \d+ phút/.test(meM2) && !/Đang tắt/.test(meM2), `tắt công tắc M (DB track_study_time=${st}); /me lớp M: «${(meM.match(/Thời gian học[\s\S]{0,30}/) ?? [""])[0].replace(/\n/g, " | ")}» (Đang tắt=${/Đang tắt/.test(meM)}); /me lớp M2: «${(meM2.match(/Tuần này[^\n]*/) ?? ["(không có biểu đồ)"])[0]}»; sai version (ngữ cảnh khác đổi trước): dòng thông báo ${JSON.stringify(notice.slice(0, 3))}, công tắc sau đó bật=${stateU}, DB=${dbv}`);
  await o.ctx.close(); psql(`update user_settings set track_study_time=true where user_id='${uid("qc3me01")}'`); rds("DEL", `ep:trk:${uid("qc3me01")}`);
  // 24
  psql(`update users set tracking_notice_ack_at=null where email='qc3me04@edupilot.local'`); rds("DEL", `ep:trk:${uid("qc3me04")}`);
  const res: Record<string, number> = {}; for (const who of ["qc3me04", "qc3t", "qc3ta", "admin"]) { const x = await newUser(browser, who); await go(x.page, `/settings`); await x.page.waitForTimeout(1200); res[who] = await x.page.getByText("Thời gian học", { exact: true }).count() + (await x.page.getByRole("switch", { name: /·/ }).count()); await x.ctx.close(); }
  psql(`update users set tracking_notice_ack_at=now() where email='qc3me04@edupilot.local'`); rds("DEL", `ep:trk:${uid("qc3me04")}`);
  R("TC-P5-06-24", Object.values(res).every((v) => v === 0), `mở /settings: SV chưa «Đã hiểu», TEACHER, TA, ADMIN — số phần tử nhóm «Thời gian học»/công tắc lớp: ${JSON.stringify(res)}`);
  await u.ctx.close();
  // 25
  const m = await newUser(browser, "qc3t", { width: 375, height: 812 }); await go(m.page, `/class/settings?course=${FM.M2}`); const sw = m.page.getByRole("switch", { name: "Cho sinh viên tắt đo thời gian học" }); await sw.waitFor(); const box = await sw.evaluate((e) => { const b = e.getBoundingClientRect(); const row = e.closest("label,div,li") as HTMLElement; const rb = row?.getBoundingClientRect(); return { h: Math.round(b.height), rowH: Math.round(rb?.height ?? 0), w: Math.round(b.width) }; }); const dis = await sw.isDisabled(); const chk = await sw.isChecked(); const sec = await m.page.locator("main").innerText(); await shot(m.page, "p506-25");
  R("TC-P5-06-25", box.h >= 44 || box.rowH >= 44, `375×812, /class/settings lớp M2 (đã có SV): công tắc «Cho sinh viên tắt đo thời gian học» cao ${box.h} px (hàng ${box.rowH}), disabled=${dis}, trạng thái bật=${chk} (giá trị «Không» = tắt/không cho); ngữ cảnh: ${(sec.match(/Cho sinh viên tắt đo[\s\S]{0,60}/) ?? [""])[0].replace(/\n/g, " | ")}`);
  await m.ctx.close();
});
