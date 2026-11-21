import { test, type Page } from "@playwright/test";
import { appendFileSync } from "node:fs";
import { execFileSync } from "node:child_process";
import { cid, go, login } from "./lib";
// QC2 sprint 8 US-P6-07 (nhập XLSX, giao diện) trên ep-r thật, lớp QD8R71. Kết quả: /tmp/qc2-s8r/ui07.tsv
test.setTimeout(240_000);
const OUT = "/tmp/qc2-s8r/ui07.tsv";
const res = (tc: string, ok: boolean, note: string) => { appendFileSync(OUT, `${tc}\t${ok ? "P" : "F"}\t${note.replace(/\s+/g, " ").slice(0, 420)}\n`); console.log(`${ok ? "P" : "F"} ${tc} ${note}`); };
const sql = (q: string) => execFileSync("docker", ["exec", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", q], { encoding: "utf8" }).trim();
const X = "/tmp/qc-p6-xlsx";
const D = (p: Page) => p.locator("dialog[open]").first();
const dt = async (p: Page) => (await D(p).innerText().catch(() => "")).replace(/\s+/g, " ");
async function mk(browser: any, w = 1440, h = 900) { const c = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: h }, locale: "vi-VN" }); const p: Page = await c.newPage(); await login(p, "teacher"); return { c, p }; }
async function openImport(p: Page) { await go(p, `/gradebook?course=${cid("QD8R71")}`); await p.waitForTimeout(2500); await p.getByRole("button", { name: "Thêm hành động" }).click(); await p.getByRole("menuitem", { name: /Nhập từ XLSX/ }).click(); await p.waitForTimeout(900); }
async function toStep3(p: Page, file: string) { await D(p).locator("input[type=file]").setInputFiles(`${X}/${file}.xlsx`); await p.waitForTimeout(600); await D(p).getByRole("button", { name: "Tiếp tục" }).click(); await p.waitForTimeout(2200); await D(p).getByRole("button", { name: /Xem trước/ }).click(); await p.waitForTimeout(3500); }

test("[07-09,07-44,07-45,07-59] ba bước, gợi ý cột, lỗi tiếng Việt, công bố mặc định tắt, MSSV chưa xác minh", async ({ browser }) => {
  const { p } = await mk(browser);
  // MSSV tự khai (joined_via=CODE) cho 20229003 trong lớp thử → STUDENT_CODE_UNVERIFIED
  sql(`update enrollments set joined_via='CODE' where course_id='${cid("QD8R71")}' and student_code_snapshot='20229003'`);
  await openImport(p); const s1 = await dt(p);
  await D(p).locator("input[type=file]").setInputFiles(`${X}/errors.xlsx`); await p.waitForTimeout(600);
  const b1 = await D(p).getByRole("button", { name: "Tiếp tục" }).count(); await D(p).getByRole("button", { name: "Tiếp tục" }).click(); await p.waitForTimeout(2200);
  const s2 = await dt(p); const sel = await p.evaluate(() => [...document.querySelectorAll("dialog[open] select")].map((s) => (s as HTMLSelectElement).selectedOptions[0]?.textContent ?? ""));
  res("TC-P6-07-09", sel.length >= 2 && /MSSV/.test(sel[0]) && /Điểm/.test(sel[1]) && /Ánh xạ|Cột MSSV/.test(s2), `bước 2 gợi ý theo tên cột: MSSV→«${sel[0]}», Điểm→«${sel[1]}»; chọn lại được (select); bước 3 xem trước có`);
  await D(p).getByRole("button", { name: /Xem trước/ }).click(); await p.waitForTimeout(3500); const s3 = await dt(p);
  const eng = /STUDENT_NOT_IN_COURSE|DUPLICATE_IN_FILE|SCORE_OUT_OF_RANGE|SCORE_PRECISION|SCORE_NOT_NUMBER|STUDENT_CODE_UNVERIFIED/.test(s3);
  const vi = ["MSSV không có trong lớp", "Điểm ngoài thang 0–10", "Điểm có quá 2 chữ số thập phân"].every((x) => s3.includes(x));
  res("TC-P6-07-44", b1 === 1 && /Ghi|Bỏ qua \d+ dòng lỗi/.test(s3) && vi && !eng, `3 bước (Tiếp tục → Xem trước → «${(s3.match(/(Bỏ qua \d+ dòng lỗi và ghi \d+ dòng|Ghi \d+ dòng)/) || [""])[0]}»); lỗi theo hàng bằng chữ Việt (${vi}), không mã tiếng Anh thô (${!eng}); tóm tắt «${(s3.match(/\d+ dòng hợp lệ[^D]*/) || [""])[0]}»`);
  const cb = D(p).getByRole("checkbox"); const off = !(await cb.isChecked()); await p.keyboard.press("Escape"); await p.waitForTimeout(400);
  // 07-59: midterm.xlsx với 20229003 đã tự khai
  await openImport(p); await toStep3(p, "midterm"); const t59 = await dt(p);
  res("TC-P6-07-59", /MSSV chưa có trong danh sách lớp/.test(t59) && !/STUDENT_CODE_UNVERIFIED/.test(t59) && !/Ngô Ngọc Cẩm.{0,40}(gợi ý|ghép)/.test(t59), `hàng có MSSV tự khai: «${(t59.match(/Dòng \d+ · 20229003 · [^D]*/) || ["(không thấy)"])[0].slice(0, 90)}»; không mã thô, không gợi ý ghép tên`);
  sql(`update enrollments set joined_via=(select joined_via from enrollments where course_id='${cid("QD8R71")}' and student_code_snapshot='20229002') where course_id='${cid("QD8R71")}' and student_code_snapshot='20229003'`);
  await p.keyboard.press("Escape"); await p.waitForTimeout(400);
  // 07-45: công bố mặc định tắt; bật + ghi → published_at
  await openImport(p); await toStep3(p, "midterm"); const cb2 = D(p).getByRole("checkbox"); const def = !(await cb2.isChecked());
  const pub0 = sql(`select published_at is not null from grade_items i join grade_schemes s on s.id=i.scheme_id where i.course_id='${cid("QD8R71")}' and s.status='CONFIRMED' and i.key='midterm'`);
  await cb2.check(); await D(p).getByRole("button", { name: /^Ghi \d+ dòng$/ }).click(); await p.waitForTimeout(3500);
  const pub1 = sql(`select published_at is not null from grade_items i join grade_schemes s on s.id=i.scheme_id where i.course_id='${cid("QD8R71")}' and s.status='CONFIRMED' and i.key='midterm'`);
  res("TC-P6-07-45", def && off && pub1 === "t", `«Công bố cột cho sinh viên» mặc định tắt (bước 3 lần 1: ${off}, lần 2: ${def}); published_at trước ${pub0 || "?"} → bật + ghi → ${pub1}`);
});

test("[07-46,07-47,07-48] ngắt mạng, ô cập nhật + focus, 375 px", async ({ browser }) => {
  const { c, p } = await mk(browser);
  await openImport(p); await D(p).locator("input[type=file]").setInputFiles(`${X}/errors.xlsx`); await p.waitForTimeout(600);
  await D(p).getByRole("combobox").first().selectOption({ label: "Thi cuối kỳ" }).catch(() => undefined); const item0 = await D(p).locator("select").first().evaluate((s) => (s as HTMLSelectElement).selectedOptions[0].textContent);
  await c.setOffline(true); await D(p).getByRole("button", { name: "Tiếp tục" }).click(); await p.waitForTimeout(3500);
  const off = await dt(p); const hasRetry = await D(p).getByRole("button", { name: "Tải lại" }).count();
  await c.setOffline(false); if (hasRetry) { await D(p).getByRole("button", { name: "Tải lại" }).click(); await p.waitForTimeout(4000); }
  const on = await dt(p); const item1 = await D(p).locator("select").first().evaluate((x) => (x as HTMLSelectElement).selectedOptions[0]?.textContent ?? "").catch(() => "");
  res("TC-P6-07-46", hasRetry > 0 && /errors\.xlsx|Không kết nối/.test(off) && /Xem trước|Ánh xạ|Cột MSSV|dòng hợp lệ/.test(on) && !/Không kết nối/.test(on), `ngắt mạng ở bước xem trước: «${(off.match(/Không kết nối[^.]*\.[^.]*\./) || [off.slice(0, 80)])[0]}» + nút Tải lại ${hasRetry}; có mạng → Tải lại → «${(on.match(/\d+ dòng hợp lệ[^D]*/) || [on.slice(0, 80)])[0]}»; item đã chọn «${item0}», sau Tải lại «${item1}»; tệp errors.xlsx còn trong hộp thoại ${/errors\.xlsx/.test(off)}`);
  // 07-47: ghi 3 dòng (errors.xlsx, Thi cuối kỳ), ô cập nhật không tải lại; focus về nút menu
  await p.keyboard.press("Escape"); await p.waitForTimeout(300);
  await openImport(p); await D(p).locator("input[type=file]").setInputFiles(`${X}/errors.xlsx`); await D(p).getByRole("combobox").first().selectOption({ label: "Giữa kỳ" }); await D(p).getByRole("button", { name: "Tiếp tục" }).click(); await p.waitForTimeout(2200); await D(p).getByRole("button", { name: /Xem trước/ }).click(); await p.waitForTimeout(3500);
  const before = await p.evaluate(() => document.querySelector("main")?.innerText.length);
  const mark = await p.evaluate(() => { (window as unknown as { __noReload: number }).__noReload = 1; return 1; });
  await D(p).getByRole("button", { name: /^Bỏ qua \d+ dòng lỗi và ghi \d+ dòng$|^Ghi \d+ dòng$/ }).click(); await p.waitForTimeout(4000);
  const still = await p.evaluate(() => (window as unknown as { __noReload?: number }).__noReload === 1); const dlgOpen = await p.locator("dialog[open]").count();
  const foc = await p.evaluate(() => { const a = document.activeElement as HTMLElement | null; return a && a.tagName !== "BODY" ? (a.getAttribute("aria-label") ?? a.textContent ?? "").slice(0, 40) : "<BODY>"; });
  const cell = await p.evaluate(() => [...document.querySelectorAll("[data-part=grade-row]")].find((r) => /Bùi Thanh Khải/.test((r as HTMLElement).innerText))?.querySelector("input")?.value ?? "(không thấy hàng)");
  res("TC-P6-07-47", still && dlgOpen === 0 && /Thêm hành động/.test(foc), `ghi xong: không tải lại trang ${still}, hộp thoại đóng ${dlgOpen === 0}; focus về «${foc}»; ô điểm của Bùi Thanh Khải «${cell}» (số lượt ${mark}/${before})`);
  // 07-48: 375 px
  const m = await mk(browser, 375, 812); await m.p.goto(`/gradebook?course=${cid("QD8R71")}`).catch(() => undefined); await go(m.p, `/gradebook?course=${cid("QD8R71")}`); await m.p.waitForTimeout(2500);
  const ox = await m.p.evaluate(() => document.documentElement.scrollWidth - window.innerWidth); const menu = await m.p.getByRole("button", { name: /^Thêm/ }).count();
  let note = `không tràn ngang ${ox <= 0}`; let usable = false;
  if (menu) { await m.p.getByRole("button", { name: /^Thêm/ }).first().click(); const it = m.p.getByRole("menuitem", { name: /Nhập từ XLSX/ }); if (await it.count()) { await it.click(); await m.p.waitForTimeout(900); const t = await dt(m.p); const ox2 = await m.p.evaluate(() => document.documentElement.scrollWidth - window.innerWidth); usable = /Nhập điểm từ XLSX|desktop|máy tính/i.test(t) && ox2 <= 0; note += `; hộp thoại: «${t.slice(0, 120)}»`; } else note += "; menu không có Nhập từ XLSX ở 375 px"; }
  res("TC-P6-07-48", ox <= 0 && usable, note);
});

test("[07-45r] công bố cột: bật + ghi → published_at (cột chưa công bố)", async ({ browser }) => {
  const { p } = await mk(browser);
  const q = `select published_at is not null from grade_items i join grade_schemes s on s.id=i.scheme_id where i.course_id='${cid("QD8R71")}' and s.status='CONFIRMED' and i.key='final'`;
  sql(`update grade_items set published_at=null where course_id='${cid("QD8R71")}' and key='final' and scheme_id in (select id from grade_schemes where course_id='${cid("QD8R71")}' and status='CONFIRMED')`);
  const pub0 = sql(q);
  await openImport(p); await D(p).locator("input[type=file]").setInputFiles(`${X}/final.xlsx`); await D(p).locator("select").first().selectOption({ label: "Thi cuối kỳ" }); await D(p).getByRole("button", { name: "Tiếp tục" }).click(); await p.waitForTimeout(2200); await D(p).getByRole("button", { name: /Xem trước/ }).click(); await p.waitForTimeout(3500);
  const cb = D(p).getByRole("checkbox"); const def = !(await cb.isChecked()); await D(p).getByRole("button", { name: /^Ghi \d+ dòng$|^Bỏ qua/ }).click(); await p.waitForTimeout(3500);
  const off = sql(q);
  await openImport(p); await D(p).locator("input[type=file]").setInputFiles(`${X}/final.xlsx`); await D(p).locator("select").first().selectOption({ label: "Thi cuối kỳ" }); await D(p).getByRole("button", { name: "Tiếp tục" }).click(); await p.waitForTimeout(2200); await D(p).getByRole("button", { name: /Xem trước/ }).click(); await p.waitForTimeout(3500);
  await D(p).getByRole("checkbox").check(); await D(p).getByRole("button", { name: /^Ghi \d+ dòng$|^Bỏ qua/ }).click(); await p.waitForTimeout(3500);
  const on = sql(q);
  res("TC-P6-07-45", def && pub0 === "f" && off === "f" && on === "t", `cột final chưa công bố (${pub0}); công bố mặc định tắt ${def}; ghi khi tắt → published_at ${off}; bật + ghi → ${on}`);
});
