import { test } from "@playwright/test";
import { execFileSync } from "node:child_process";
import { S, login, res, settle, text, shot, sql, TECH } from "./lib";
// TC-P209-51 (AC17 đầu cuối): D mở /join/<mã> 375 × 812, xem trước, tham gia lớp cần duyệt; GV thấy chuông + hàng chờ, duyệt;
// D thấy lớp ở bộ chọn lớp và «Hôm nay»; GV tạo lại mã; D5 thử mã cũ; 6 mã sai ⇒ 429.
test.setTimeout(300_000);
const clearJoin = () => execFileSync("docker", ["exec", "ep-r-redis-1", "sh", "-c", "redis-cli --scan --pattern 'ep:join:fail:*' | xargs -r redis-cli del"]);
test("[P209-51] join flow end to end", async ({ browser }) => {
  const E = S.E2E; const out: string[] = []; let ok = true; const note = (s: string, c = true) => { out.push(s); ok = ok && c; };
  clearJoin();
  const mD = await browser.newContext({ viewport: { width: 375, height: 812 }, hasTouch: true, ignoreHTTPSErrors: true, locale: "vi-VN" }); const pD = await mD.newPage();
  await login(pD, E.D, S.pwd); await pD.goto(`/join/${E.code}`); await settle(pD, 2500);
  const pv = await text(pD); await shot(pD, "e2e-d-preview"); const needs = /Lớp này cần giảng viên duyệt/.test(pv);
  await pD.getByRole("button", { name: "Tham gia lớp", exact: true }).click(); await settle(pD, 2000); const wait = await text(pD); await shot(pD, "e2e-d-waiting");
  const st1 = sql(`select e.status from enrollments e join users u on u.id=e.user_id where e.course_id='${E.id}' and u.email='${E.D}'`);
  note(`D: xem trước «${pv.slice(pv.indexOf("Lớp "), pv.indexOf("Lớp ") + 90)}»; cần duyệt=${needs}; sau tham gia «${wait.slice(wait.indexOf("Tham gia lớp"), wait.indexOf("Tham gia lớp") + 110)}»; DB status=${st1}`, needs && /chờ giảng viên duyệt/i.test(wait) && st1 === "PENDING");
  // GV: chuông + hàng chờ + duyệt
  const mG = await browser.newContext({ viewport: { width: 1440, height: 900 }, ignoreHTTPSErrors: true, locale: "vi-VN" }); const pG = await mG.newPage();
  await login(pG, "teacher"); await pG.goto(`/?course=${E.id}`); await settle(pG, 2500);
  const bell = pG.getByRole("button", { name: /^Thông báo/ }); const dot = await pG.locator("[data-part=bell-dot]").count(); await bell.click(); await pG.waitForTimeout(1200);
  const bt = await text(pG, "[data-part=notifications]"); await shot(pG, "e2e-gv-bell"); await pG.keyboard.press("Escape");
  const nDb = sql(`select count(*) from notifications where user_id=(select id from users where email='teacher@edupilot.local') and course_id='${E.id}' and created_at > now() - interval '30 minutes'`);
  note(`GV: chuông chấm=${dot}; khung thông báo «${bt.slice(0, 160)}»; thông báo lớp trong DB 30 phút=${nDb}`, Number(nDb) >= 1 && /(tham gia|duyệt|yêu cầu)/i.test(bt));
  await pG.goto(`/class/members?course=${E.id}`); await settle(pG, 2500); await pG.getByRole("tab", { name: /Chờ duyệt/ }).click(); await settle(pG, 1200);
  const row = pG.locator("tbody tr").filter({ hasText: /ue1/ }); const rows = await row.count(); await shot(pG, "e2e-gv-pending");
  await row.first().getByRole("checkbox").check(); await pG.getByRole("button", { name: /^Duyệt \d+$/ }).click(); await settle(pG, 2500);
  const st2 = sql(`select e.status from enrollments e join users u on u.id=e.user_id where e.course_id='${E.id}' and u.email='${E.D}'`); await shot(pG, "e2e-gv-approved");
  note(`GV: hàng chờ của D=${rows}; sau Duyệt DB status=${st2}`, rows === 1 && st2 === "ACTIVE");
  // D thấy lớp
  await pD.goto(`/?course=${E.id}`); await settle(pD, 3000); let dlgTxt = ""; if (await pD.getByRole("dialog").count()) { dlgTxt = (await pD.getByRole("dialog").first().innerText()).replace(/\s+/g, " ").slice(0, 160); await shot(pD, "e2e-d-dialog"); await pD.getByRole("dialog").getByRole("button", { name: "Đã hiểu" }).click({ timeout: 10_000 }); await pD.waitForTimeout(600); }
  const pk = pD.getByRole("button", { name: /^Chọn lớp, đang xem/ }); const hasPk = await pk.count(); let items: string[] = [];
  if (hasPk) { await pk.first().click({ timeout: 10_000 }); await pD.waitForTimeout(600); items = (await pD.getByRole("menuitem").allInnerTexts()).map((x) => x.replace(/\s+/g, " ")); await pD.keyboard.press("Escape"); }
  const bar = await text(pD, "[data-part=topbar]"); await shot(pD, "e2e-d-picker");
  const today = await text(pD); const inPicker = items.some((x) => x.includes(E.class_code)) || bar.includes(E.class_code); const inToday = today.includes(E.class_code) || /Hôm nay/.test(today);
  const ox = await pD.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  note(`D sau duyệt: lớp ${E.class_code} ở bộ chọn/thanh trên=${inPicker} (hộp thoại lúc vào «${dlgTxt}»; bộ chọn có=${hasPk}, ${items.length} mục; thanh «${bar.slice(0, 80)}»); «Hôm nay»=${inToday}; tràn ngang ${ox}`, inPicker && inToday && ox === 0);
  // GV tạo lại mã
  const old = E.code; await pG.goto(`/class/settings?course=${E.id}`); await settle(pG, 2500); await pG.getByRole("button", { name: "Tạo lại mã" }).first().click({ timeout: 10_000 }); await pG.waitForTimeout(700);
  await pG.getByRole("dialog").getByRole("button", { name: "Tạo lại mã" }).click(); await settle(pG, 2500);
  const nw = sql(`select join_code from courses where id='${E.id}'`); note(`GV tạo lại mã: ${old} → ${nw}`, nw !== old && nw.length === 7);
  await mG.close();
  // D5 thử mã cũ rồi 6 mã sai
  clearJoin(); const m5 = await browser.newContext({ viewport: { width: 375, height: 812 }, hasTouch: true, ignoreHTTPSErrors: true, locale: "vi-VN" }); const p5 = await m5.newPage(); const codes: number[] = [];
  p5.on("response", (r) => { if (/\/api\/v1\/courses\/.*join|\/api\/v1\/courses\/preview/.test(r.url()) || /join/.test(r.url())) codes.push(r.status()); });
  await login(p5, E.D5, S.pwd); await p5.goto("/join"); await settle(p5);
  const inp = p5.getByLabel("Mã tham gia"); const btn = p5.getByRole("button", { name: "Xem lớp" });
  await inp.fill(old); await btn.click(); await p5.waitForTimeout(1200); const t1 = await text(p5); const generic = /Mã không hợp lệ hoặc đã hết hạn/.test(t1) && !t1.includes(E.class_code);
  for (const c of ["ZZZZZ22", "ZZZZZ23", "ZZZZZ24", "ZZZZZ25", "ZZZZZ26"]) { await inp.fill(c); if (await btn.isEnabled()) await btn.click(); await p5.waitForTimeout(900); }
  await settle(p5, 800); const lock = await text(p5); await shot(p5, "e2e-d5-locked"); const has429 = codes.includes(429);
  note(`D5: mã cũ «${t1.slice(t1.indexOf("Mã không"), t1.indexOf("Mã không") + 80)}» (câu chung, không lộ lớp)=${generic}; sau 6 lần sai «${lock.slice(lock.indexOf("Bạn đã thử"), lock.indexOf("Bạn đã thử") + 60)}»; mã trạng thái các lời gọi=${JSON.stringify(codes)}; có 429=${has429}; từ kỹ thuật=${TECH.test(lock)}`, generic && has429 && /Thử lại sau\s*\d\d:\d\d/.test(lock));
  await m5.close(); await mD.close();
  res("TC-P209-51", ok, out.join(" ‖ "));
});
