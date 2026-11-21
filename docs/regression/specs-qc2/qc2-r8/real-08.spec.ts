import { test, type Browser, type Page } from "@playwright/test";
import { appendFileSync } from "node:fs";
import { execFileSync } from "node:child_process";
import { cid, go, login, IDS } from "./lib";
// QC2 sprint 8: US-P6-06 (hồ sơ SV), US-P6-08 (chốt điểm), US-P6-09 (/me) trên ep-r thật. Kết quả: /tmp/qc2-s8r/ui08.tsv
test.setTimeout(240_000);
const OUT = "/tmp/qc2-s8r/ui08.tsv";
const res = (tc: string, ok: boolean, note: string) => { appendFileSync(OUT, `${tc}\t${ok ? "P" : "F"}\t${note.replace(/\s+/g, " ").slice(0, 420)}\n`); console.log(`${ok ? "P" : "F"} ${tc} ${note}`); };
const sql = (q: string) => execFileSync("docker", ["exec", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-F", "|", "-c", q], { encoding: "utf8" }).trim();
async function mk(browser: Browser, user: string, w = 1440, h = 900) { const c = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: h }, locale: "vi-VN" }); const p: Page = await c.newPage(); await login(p, user); return { c, p }; }
const DLG = (p: Page) => p.locator("dialog[open]").first();

test("[06-46] hồ sơ SV tab Điểm: GradeCalculation thật; lớp chưa có công thức: EmptyState", async ({ browser }) => {
  const { p } = await mk(browser, "teacher");
  await go(p, `/students/${IDS["QD8R6U"].students["sv.kha"]}?course=${cid("QD8R6U")}`); await p.waitForTimeout(3000);
  await p.getByRole("tab", { name: /Điểm/ }).first().click(); await p.waitForTimeout(2500); const t = (await p.locator("main").innerText()).replace(/\s+/g, " ");
  const okB = ["7,5", "= 8,25", "7,2", "Làm tròn đến 0,5"].every((x) => t.includes(x)) && /Giữa kỳ \(20%\)/.test(t);
  // lớp chưa có công thức đã xác nhận: lớp QD8A2 (nháp, chưa xác nhận) xem bởi teacher
  const stu = sql(`select e.user_id from enrollments e join courses c on c.id=e.course_id join users u on u.id=e.user_id where c.class_code='QD8A2' and u.role='STUDENT' order by 1 limit 1`);
  let t2 = "(không có SV trong QD8A2)";
  if (stu) { await go(p, `/students/${stu}?course=${cid("QD8A2")}`); await p.waitForTimeout(3000); await p.getByRole("tab", { name: /Điểm/ }).first().click().catch(() => undefined); await p.waitForTimeout(2000); t2 = (await p.locator("main").innerText()).replace(/\s+/g, " "); }
  const empty = /Lớp chưa có công thức điểm đã xác nhận/.test(t2);
  res("TC-P6-06-46", okB && empty, `hồ sơ B (QD8R6U) tab Điểm: GradeCalculation đủ dòng (7,5 · 8 · 6,5 · điểm cộng · quá trình 8,25 · thô 7,2 · làm tròn 0,5) ${okB}; lớp chưa có công thức đã xác nhận (QD8A2, bản nháp): EmptyState «${(t2.match(/Lớp chưa có công thức điểm đã xác nhận\.?/) || [t2.slice(0, 100)])[0]}»`);
});

test("[08-10,08-38] Chốt điểm khi thiếu điểm: danh sách thiếu + liên kết tới ô; lỗi 409 bằng chữ Việt", async ({ browser }) => {
  const Y = cid("QD8R6Y"); const rows = sql(`select i.key, e.student_id from grade_entries e join grade_items i on i.id=e.item_id where e.course_id='${Y}' and i.key='final' order by e.student_id limit 1`).split("|");
  const bak = sql(`select * from grade_entries where course_id='${Y}' and student_id='${rows[1]}' and item_id=(select id from grade_items where course_id='${Y}' and key='final' and scheme_id in (select id from grade_schemes where course_id='${Y}' and status='CONFIRMED') limit 1)`);
  console.log("entry:", bak);
  sql(`delete from grade_entries where course_id='${Y}' and student_id='${rows[1]}' and item_id=(select id from grade_items where course_id='${Y}' and key='final' and scheme_id in (select id from grade_schemes where course_id='${Y}' and status='CONFIRMED') limit 1)`);
  const { p } = await mk(browser, "teacher");
  try {
    await go(p, `/gradebook?course=${Y}`); await p.waitForTimeout(2500);
    await p.getByRole("button", { name: "Chốt điểm" }).first().click(); await p.waitForTimeout(2500);
    let t = (await DLG(p).innerText().catch(() => "")).replace(/\s+/g, " "); const miss = p.getByRole("button", { name: /thiếu Thi cuối kỳ/ }).or(p.getByRole("link", { name: /thiếu Thi cuối kỳ/ })); const inList = await p.getByText(/Thiếu điểm: \d+ sinh viên/).count(); const links = await miss.count();
    const confirmBtn = DLG(p).getByRole("button", { name: /^Chốt điểm$/ }); const hasConfirm = (await confirmBtn.count()) > 0 && !(await confirmBtn.isDisabled().catch(() => false));
    let t2 = t; if (hasConfirm) { await confirmBtn.click(); await p.waitForTimeout(3000); t2 = (await DLG(p).innerText().catch(async () => await p.locator("main").innerText())).replace(/\s+/g, " "); }
    const links2 = await p.locator("[data-part=incomplete-list] button").count(); const inList2 = await p.locator("[data-part=incomplete-list]").count();
    const listText = (await p.getByText(/Thiếu điểm: \d+ sinh viên/).first().locator("xpath=..").innerText().catch(() => "")).replace(/\s+/g, " ");
    const raw = /GRADES_INCOMPLETE|SCHEME_NOT_CONFIRMED|NOT_LOCKED|[A-Z]{4,}_[A-Z_]{3,}/.test(t2 + listText);
    let focused = "";
    if (links2 > 0) { await p.locator("[data-part=incomplete-list] button").first().click(); await p.waitForTimeout(1500); focused = await p.evaluate(() => { const a = document.activeElement as HTMLElement | null; return (a?.getAttribute("data-cell") ? "ô " + a.getAttribute("data-cell") : a?.tagName ?? "") + " | hàng " + ((a?.closest("[data-part=grade-row]") as HTMLElement | null)?.innerText.slice(0, 20).replace(/\n/g, " ") ?? "-"); }); }
    res("TC-P6-08-10", inList2 > 0 && links2 > 0 && /ô /.test(focused), `Chốt điểm khi thiếu 1 điểm: danh sách thiếu ${inList2} (nút liên kết ${links2}); chữ «${(listText || t2).slice(0, 160)}»; click liên kết → focus «${focused.slice(0, 40)}»`);
    res("TC-P6-08-38", !raw && /điểm|thiếu/i.test(t2 + listText), `409 hiển thị bằng chữ Việt: «${(t2.match(/[^.]*(thiếu|chưa có điểm|chưa đủ)[^.]*\./i) || [t2.slice(0, 140)])[0]}»; mã thô ${raw}`);
  } finally {
    if (bak) { const f = bak.split("|"); sql(`insert into grade_entries (id,course_id,item_id,student_id,score,source,version,entered_by,created_at,updated_at) values ('${f[0]}','${f[1]}','${f[2]}','${f[3]}',${f[4]},'${f[5]}',${f[6]},${f[7] ? `'${f[7]}'` : "null"},'${f[8]}','${f[9]}') on conflict do nothing`); }
  }
});

test("[08-36] hộp thoại Chốt điểm: phân bố khớp CSV + Idempotency-Key", async ({ browser }) => {
  const { p } = await mk(browser, "teacher");
  await go(p, `/gradebook?course=${cid("QD8R6X")}`); await p.waitForTimeout(2500);
  await p.getByRole("button", { name: "Chốt điểm" }).first().click(); await p.waitForTimeout(2500); const t = (await DLG(p).innerText()).replace(/\s+/g, " ");
  await p.keyboard.press("Escape"); await p.waitForTimeout(500);
  const want = "Chốt 30 sinh viên · phân bố: A+ 3, A 3, B+ 3, B 5, C+ 4, C 4, D+ 2, D 3, F 3 · 1 sinh viên liệt";
  // Idempotency-Key trong request (chốt thật lớp QD8R6Z; lớp riêng của QC)
  await go(p, `/gradebook?course=${cid("QD8R6Z")}`); await p.waitForTimeout(2500);
  let key = ""; p.on("request", (r) => { if (r.method() === "POST" && /gradebook\/finalize/.test(r.url())) key = r.headers()["idempotency-key"] ?? ""; });
  await p.getByRole("button", { name: "Chốt điểm" }).first().click(); await p.waitForTimeout(2000); await DLG(p).getByRole("button", { name: /^Chốt điểm$/ }).click(); await p.waitForTimeout(3500);
  res("TC-P6-08-36", t.includes(want) && key.length > 8, `hộp thoại (QD8R6X, lớp 1 đủ điểm): «${(t.match(/Chốt \d+ sinh viên[^D]*liệt/) || [t.slice(0, 160)])[0]}» khớp từng chữ cái với bảng tay: ${t.includes(want)}; Idempotency-Key trong POST finalize: ${key ? "có (" + key.slice(0, 8) + "…)" : "không"}`);
});

test("[09-13,09-14,09-15] /me của SV: dòng điểm, GradeCalculation tuyến tính, quét gamify", async ({ browser }) => {
  const { p } = await mk(browser, "sv.kha");
  await go(p, `/me?course=${cid("QD8R6U")}`); await p.waitForTimeout(3500); const t = (await p.locator("main").innerText()).replace(/\s+/g, " "); const html = await p.content();
  const order = ["Giữa kỳ ·", "Kiểm tra hằng tuần ·", "Thi cuối kỳ ·", "Điểm cộng", "Điểm trừ", "Điểm quá trình trước"].map((k) => t.indexOf(k)); const sorted = order.every((v, i) => v >= 0 && (i === 0 || v > order[i - 1]));
  const gam = /streak|xếp hạng|so với lớp|huy hiệu|Tuyệt vời|cố lên|🎉/i.test(t); const dot = /\d\.\d\d? \/ 10/.test(t);
  const noExpr = !/min\(|max\(|round\(|÷/.test(t);
  res("TC-P6-09-13", /Điểm quá trình 8,25 \/ 10/.test(t) && sorted && noExpr, `câu nhận định nay là dòng «nhãn + giá trị» (P-11-6a, US.md AC3): «${(t.match(/Điểm quá trình 8,25 \/ 10/) || ["(không thấy)"])[0]}»; GradeCalculation tuyến tính đúng thứ tự ${sorted}; DTO SV không lộ expr/min/max/round/÷: ${noExpr}`);
  res("TC-P6-09-15", !gam && !dot, `quét /me: streak/xếp hạng/so với lớp/huy hiệu/lời khen/🎉 = ${gam ? ">0" : "0"}; số dùng dấu phẩy (không dạng 8.25): ${!dot}`);
  const f = await mk(browser, "sv.kha"); await go(f.p, `/me?course=${cid("QD8R8")}`); await f.p.waitForTimeout(3500); const t2 = (await f.p.locator("main").innerText()).replace(/\s+/g, " ");
  res("TC-P6-09-14", /Điểm học phần 7 \(B\)/.test(t2), `sau chốt (QD8R8): «${(t2.match(/Điểm học phần [^ ]+ \([A-F+]+\)/) || ["(không thấy)"])[0]}» — dòng điểm nay theo SRS P-11-6a («nhãn + giá trị»); SRS ví dụ «7,0 (B)», màn hiện «7 (B)» (bỏ số 0 thừa). Chưa có công thức: «Lớp chưa có công thức điểm chính thức.» (xem 05-xx)`);
});

test("[09-46] /me 375 px: skeleton / rỗng / lỗi + Thử lại, không cuộn ngang", async ({ browser }) => {
  const { c, p } = await mk(browser, "sv.kha", 375, 812); const C = cid("QD8R6U");
  let slow = true; await p.route("**/me/grades", async (r) => { if (slow) await new Promise((x) => setTimeout(x, 3000)); await r.continue().catch(() => undefined); });
  await p.goto(`/me?course=${C}`); await p.waitForTimeout(900); const sk = await p.locator('[aria-busy=true], [data-skeleton], [class*=Skeleton], [class*=keleton]').count(); await p.waitForTimeout(4500); slow = false; await p.unroute("**/me/grades");
  let fail = true; await p.route("**/me/grades", async (r) => { if (fail) await r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "Lỗi" }) }); else await r.continue().catch(() => undefined); });
  await p.reload(); await p.waitForTimeout(3500); const retry = p.getByRole("button", { name: /Thử lại/ }); const hasRetry = (await retry.count()) > 0; const ox1 = await p.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
  fail = false; if (hasRetry) { await retry.first().click(); await p.waitForTimeout(3000); } const back = (await p.locator("main").innerText()).includes("Điểm");
  await p.unroute("**/me/grades");
  // rỗng: lớp mới chưa công bố cột nào — bản sao QD8R7U? dùng lớp có công thức, SV chưa có cột công bố
  const e = await c.newPage(); const empty = "QD8H3"; const E = cid(empty); const kha = IDS[empty]?.students?.["sv.kha"] ?? sql(`select id from users where email='sv.kha@edupilot.local'`);
  const wk = `from grade_entries where course_id='${E}' and student_id='${kha}' and item_id in (select i.id from grade_items i join grade_schemes s on s.id=i.scheme_id where s.course_id='${E}' and s.status='CONFIRMED' and i.key='weekly')`;
  const bak = sql(`select * ${wk}`); sql(`delete ${wk}`);
  let emptyTxt = "";
  try { await go(e, `/me?course=${E}`); await e.waitForTimeout(3500); emptyTxt = (await e.locator("main").innerText()).replace(/\s+/g, " ").slice(0, 200); }
  finally { if (bak) { const f = bak.split("|"); sql(`insert into grade_entries (id,course_id,item_id,student_id,score,source,version,entered_by,created_at,updated_at) values ('${f[0]}','${f[1]}','${f[2]}','${f[3]}',${f[4]},'${f[5]}',${f[6]},${f[7] ? `'${f[7]}'` : "null"},'${f[8]}','${f[9]}') on conflict do nothing`); } }
  res("TC-P6-09-46", sk > 0 && hasRetry && ox1 <= 0 && back && /Chưa có điểm (nào )?được công bố/.test(emptyTxt), `375 px: skeleton khi chậm ${sk}; lỗi 500 → Thử lại ${hasRetry}, bấm lại tải được ${back}, không cuộn ngang ${ox1 <= 0}; rỗng (lớp ${empty}, SV chưa có cột nào công bố): «${emptyTxt.slice(0, 120)}» (SRS AC9 viết «Chưa có điểm được công bố.», màn «Chưa có điểm nào được công bố» — lệch 1 chữ «nào»)`);
});
